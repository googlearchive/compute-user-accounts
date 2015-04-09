/* Copyright 2015 Google Inc. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include "utils.h"  // NOLINT(build/include)

#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <sys/select.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <sys/wait.h>
#include <unistd.h>
#include <cassert>
#include <cstdlib>
#include <cstring>
#include <sstream>
#include <stdexcept>

#define CONC(A, B) CONC_(A, B)
#define CONC_(A, B) A ":"#B
#define LOCATION CONC(__FILE__, __LINE__)


namespace utils {

const timeval kNormalTimeout = { 1, 0 };  // 1 second.
const timeval kExtendedTimeout = { 5, 0 };  // 5 seconds.

// AutoFd encapsulates a file descriptor.
class AutoFd {
 public:
  // Creates an AutoFd to manage the lifetime of a file descriptor.
  explicit AutoFd(int fd);
  // Closes the underlying file descriptor.
  ~AutoFd();

  // Gets the underlying file descriptor.
  int get() const;

 private:
  const int fd_;

  // Not copyable or assignable.
  AutoFd& operator=(const AutoFd&);
  AutoFd(const AutoFd&);
};

// AutoLock encapsulates a mutex locking operation.
class AutoLock {
 public:
  // Creates an AutoLock to manage the lifetime of a locking operation and locks
  // the underlying mutex.
  explicit AutoLock(pthread_mutex_t* mutex);
  // Unlocks the underlying mutex.
  ~AutoLock();

 private:
  pthread_mutex_t* const mutex_;

  // Not copyable or assignable.
  AutoLock& operator=(const AutoLock&);
  AutoLock(const AutoLock&);
};

AutoFd::AutoFd(int fd)
    : fd_(fd) {}

AutoFd::~AutoFd() {
  if (fd_ != -1) {
    close(fd_);
  }
}

int AutoFd::get() const {
  return fd_;
}

AutoLock::AutoLock(pthread_mutex_t* mutex)
    : mutex_(mutex) {
  pthread_mutex_lock(mutex_);
}

AutoLock::~AutoLock() {
  pthread_mutex_unlock(mutex_);
}

BufferManager::BufferManager(char* buf, size_t buflen)
    : buf_(buf),
      buflen_(buflen) {}

char* BufferManager::AppendString(const std::string& value) {
  size_t bytes_to_write = value.length() + 1;

  char* result = static_cast<char*>(Reserve(bytes_to_write));
  strncpy(result, value.c_str(), bytes_to_write);

  return result;
}

char** BufferManager::AppendVector(const std::vector<std::string>& value) {
  // Check that we have enough space for the array of pointers and the NULL
  // terminator.
  size_t bytes_to_write = (value.size() + 1) * sizeof(char*);

  char** result = static_cast<char**>(Reserve(bytes_to_write));
  char** curr = result;

  for (size_t i = 0; i < value.size(); i++) {
    *curr = AppendString(value[i]);
    curr++;
  }
  *curr = NULL;

  return result;
}


void BufferManager::CheckSpaceAvailable(size_t bytes_to_write) const {
  if (bytes_to_write > buflen_) {
    throw std::length_error(LOCATION);
  }
}

void* BufferManager::Reserve(size_t bytes) {
  CheckSpaceAvailable(bytes);
  void* result = buf_;
  buf_ += bytes;
  buflen_ -= bytes;
  return result;
}

EntityList::EntityList()
    : index_(0) {
  pthread_mutex_init(&mutex_, NULL);
}

EntityList::~EntityList() {
  pthread_mutex_destroy(&mutex_);
}

void EntityList::Load(const std::string& command) {
  AutoLock lock(&mutex_);
  index_ = 0;
  output_.clear();
  GetDaemonOutput(command, kMultiLine, &output_);
}

void EntityList::Clear() {
  AutoLock lock(&mutex_);
  index_ = 0;
  output_.clear();
}

std::string EntityList::Pop() {
  AutoLock lock(&mutex_);
  if (index_ >= output_.size()) {
    throw std::out_of_range(LOCATION);
  }
  return output_[index_++];
}

void TokenizeString(const std::string& value,
                    char delim,
                    std::vector<std::string>* result) {
  if (value.empty()) {
    return;
  }

  size_t start_pos = 0;
  size_t end_pos;
  do {
    end_pos = value.find(delim, start_pos);
    size_t substr_size = end_pos - start_pos;
    std::string token = value.substr(start_pos, substr_size);
    result->push_back(token);
    start_pos = end_pos + 1;
  } while (end_pos != std::string::npos);
}

uint32_t ParseId(const std::string& value) {
  if (value.empty()) {
    throw std::runtime_error(LOCATION);
  }
  char* end = NULL;
  // NOLINTNEXTLINE(runtime/deprecated_fn)
  int64_t id = std::strtol(value.c_str(), &end, 10);
  if (end != &(*value.end()) || id < 0) {
    throw std::runtime_error(LOCATION);
  }

  return static_cast<uint32_t>(id);
}

enum WaitType { kConnection, kRead, kExtendedRead };

void WaitUntilFdReady(int fd, WaitType wait_type) {
  fd_set fds;
  FD_ZERO(&fds);
  FD_SET(fd, &fds);
  fd_set* read_fds = NULL;
  fd_set* write_fds = NULL;
  timeval timeout;
  switch (wait_type) {
    case kConnection:
      write_fds = &fds;
      timeout = kNormalTimeout;
      break;
    case kRead:
      read_fds = &fds;
      timeout = kNormalTimeout;
      break;
    case kExtendedRead:
      read_fds = &fds;
      timeout = kExtendedTimeout;
      break;
    default:
      assert(false);
  }
  if (select(fd+1, read_fds, write_fds, NULL, &timeout) != 1) {
    throw std::runtime_error(LOCATION);
  }
}

void GetDaemonOutput(const std::string& command,
                     OutputType output_type,
                     std::vector<std::string>* output_lines) {
  sockaddr_un address;
  address.sun_family = AF_UNIX;
  // SOCKET_PATH is defined in the Makefile.
  if (sizeof(address.sun_path) < sizeof(SOCKET_PATH)) {
    throw std::runtime_error(LOCATION);
  }
  strncpy(address.sun_path, SOCKET_PATH, sizeof(address.sun_path));
  AutoFd fd(socket(AF_UNIX, SOCK_STREAM, 0));
  if (fd.get() == -1) {
    throw std::runtime_error(LOCATION);
  }
  // Set the socket to be non-blocking.
  if (fcntl(fd.get(), F_SETFL, O_NONBLOCK)) {
    throw std::runtime_error(LOCATION);
  }
  errno = 0;
  int ret = connect(fd.get(), reinterpret_cast<sockaddr*>(&address),
                    sizeof(address));
  if (errno == EINPROGRESS) {
    WaitUntilFdReady(fd.get(), kConnection);
  } else if (ret) {
    throw std::runtime_error(LOCATION);
  }

  size_t bytes_written = write(fd.get(), command.c_str(), command.size());
  if (bytes_written != command.size()) {
    throw std::runtime_error(LOCATION);
  }

  WaitType type = (output_type != kSingleLineExtendedTimeout) ?
      kRead : kExtendedRead;
  WaitUntilFdReady(fd.get(), type);
  std::ostringstream output;
  char buff[1024];
  ssize_t bytes_read;
  while ((bytes_read = read(fd.get(), buff, sizeof(buff)))) {
    if (bytes_read == -1) {
      throw std::runtime_error(LOCATION);
    }
    output.write(buff, bytes_read);
    WaitUntilFdReady(fd.get(), kRead);
  }

  TokenizeString(output.str(), '\n', output_lines);
  if (!output_lines->size()) {
    throw std::runtime_error(LOCATION);
  }
  std::string result_code = *output_lines->begin();
  output_lines->erase(output_lines->begin());
  if (result_code == "404") {
    // User or group argument was not found.
    throw std::invalid_argument(command);
  } else if (result_code != "200") {
    // Operation did not succeed.
    throw std::runtime_error(LOCATION);
  } else if (output_type != kMultiLine && output_lines->size() != 1) {
    throw std::runtime_error(LOCATION);
  }
}

void UserLineToPasswdStruct(const std::string& line,
                            passwd* pwd,
                            BufferManager* buf) {
  std::vector<std::string> line_fields;
  TokenizeString(line, ':', &line_fields);
  if (line_fields.size() != 6) {
    throw std::runtime_error(LOCATION);
  }

  pwd->pw_name = buf->AppendString(line_fields[0]);
  pwd->pw_passwd = buf->AppendString("x");
  pwd->pw_uid = ParseId(line_fields[1]);
  pwd->pw_gid = ParseId(line_fields[2]);
  pwd->pw_gecos = buf->AppendString(line_fields[3]);
  pwd->pw_dir = buf->AppendString(line_fields[4]);
  pwd->pw_shell = buf->AppendString(line_fields[5]);
}

void GroupLineToGroupStruct(const std::string& line,
                            group* grp,
                            BufferManager* buf) {
  std::vector<std::string> line_fields;
  TokenizeString(line, ':', &line_fields);
  if (line_fields.size() != 3) {
    throw std::runtime_error(LOCATION);
  }
  std::vector<std::string> members;
  TokenizeString(line_fields[2], ',', &members);

  grp->gr_name = buf->AppendString(line_fields[0]);
  grp->gr_passwd = buf->AppendString("x");
  grp->gr_gid = ParseId(line_fields[1]);
  grp->gr_mem = buf->AppendVector(members);
}

void AccountNameToShadowStruct(const std::string& name,
                               spwd* pwd,
                               BufferManager* buf) {
  if (name.find(':') != std::string::npos) {
    throw std::runtime_error(LOCATION);
  }

  pwd->sp_namp = buf->AppendString(name);
  pwd->sp_pwdp = buf->AppendString("*");
  pwd->sp_lstchg = -1;
  pwd->sp_min = -1;
  pwd->sp_max = -1;
  pwd->sp_warn = -1;
  pwd->sp_inact = -1;
  pwd->sp_expire = -1;
  pwd->sp_flag = -1;
}

}  // namespace utils
