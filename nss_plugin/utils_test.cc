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
#include <pthread.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <stdexcept>

#include "gtest/gtest.h"

using utils::AccountNameToShadowStruct;
using utils::BufferManager;
using utils::EntityList;
using utils::GetDaemonOutput;
using utils::GroupLineToGroupStruct;
using utils::ParseId;
using utils::TokenizeString;
using utils::UserLineToPasswdStruct;

typedef std::pair<const std::string&, const std::string&> RequestResponse;

class LibnssGoogleTest : public ::testing::Test {
 protected:
  LibnssGoogleTest() {
    is_listening_ = false;
    stop_listening_ = false;
    pthread_mutex_init(&mutex_, NULL);
    pthread_cond_init(&listening_cond_, NULL);
    pthread_cond_init(&stop_cond_, NULL);
  }

  ~LibnssGoogleTest() {
    pthread_mutex_destroy(&mutex_);
    pthread_cond_destroy(&listening_cond_);
    pthread_cond_destroy(&stop_cond_);
  }

  static void* ServerThreadMain(void* data) {
    const RequestResponse& rr = *static_cast<RequestResponse*>(data);
    int socket_fd;
    OpenServerSocket(&socket_fd);
    VerifyRequestAndSendResponse(socket_fd, rr);
    CloseServerSocket(socket_fd);
    return NULL;
  }

  static void* NoAcceptServerThreadMain(void*) {
    int socket_fd;
    OpenServerSocket(&socket_fd);
    listen(socket_fd, 5);
    SignalListening();
    WaitForShutdown();
    CloseServerSocket(socket_fd);
    return NULL;
  }

  static void* NoResponseServerThreadMain(void*) {
    int socket_fd;
    OpenServerSocket(&socket_fd);
    listen(socket_fd, 5);
    SignalListening();
    int fd = accept(socket_fd, NULL, NULL);
    WaitForShutdown();
    close(fd);
    CloseServerSocket(socket_fd);
    return NULL;
  }

  static void* PartialResponseServerThreadMain(void*) {
    int socket_fd;
    OpenServerSocket(&socket_fd);
    listen(socket_fd, 5);
    SignalListening();
    int fd = accept(socket_fd, NULL, NULL);
    const char* message = "200";
    write(fd, message, sizeof(message));
    WaitForShutdown();
    close(fd);
    CloseServerSocket(socket_fd);
    return NULL;
  }

  static void WaitForServerToListen() {
    pthread_mutex_lock(&mutex_);
    while (!is_listening_) {
      pthread_cond_wait(&listening_cond_, &mutex_);
    }
    pthread_mutex_unlock(&mutex_);
  }

  static void ShutdownServer() {
    pthread_mutex_lock(&mutex_);
    stop_listening_ = true;
    pthread_cond_broadcast(&stop_cond_);
    pthread_mutex_unlock(&mutex_);
  }

 private:
  static void OpenServerSocket(int* socket_fd) {
    sockaddr_un address;
    address.sun_family = AF_UNIX;
    strncpy(address.sun_path, SOCKET_PATH,
            sizeof(address.sun_path));
    *socket_fd = socket(AF_UNIX, SOCK_STREAM, 0);
    ASSERT_GE(*socket_fd, 0) << strerror(errno);
    int yes = 1;
    int ret = setsockopt(*socket_fd, SOL_SOCKET, SO_REUSEADDR, &yes,
                         sizeof(yes));
    ASSERT_EQ(0, ret) << strerror(errno);
    ret = bind(*socket_fd, reinterpret_cast<sockaddr*>(&address),
                   sizeof(address));
    ASSERT_EQ(0, ret) << strerror(errno);
  }

  static void SignalListening() {
    pthread_mutex_lock(&mutex_);
    is_listening_ = true;
    pthread_cond_broadcast(&listening_cond_);
    pthread_mutex_unlock(&mutex_);
  }

  static void VerifyRequestAndSendResponse(int socket_fd,
                                           const RequestResponse& rr) {
    listen(socket_fd, 5);
    SignalListening();
    int fd = accept(socket_fd, NULL, NULL);
    ASSERT_GE(fd, 0) << strerror(errno);
    char request_buffer[1024];
    read(fd, request_buffer, sizeof(request_buffer));
    EXPECT_EQ(rr.first, request_buffer);
    // Send 16 bytes at a time to simulate packets.
    for (size_t i = 0; i < rr.second.size(); i += 16) {
      std::string chunk = rr.second.substr(i, 16);
      write(fd, chunk.c_str(), chunk.size());
    }
    close(fd);
    WaitForShutdown();
  }

  static void WaitForShutdown() {
    pthread_mutex_lock(&mutex_);
    while (!stop_listening_) {
      pthread_cond_wait(&stop_cond_, &mutex_);
    }
    pthread_mutex_unlock(&mutex_);
  }

  static void CloseServerSocket(int socket_fd) {
    close(socket_fd);
    unlink(SOCKET_PATH);
  }

  static bool is_listening_;
  static bool stop_listening_;
  static pthread_mutex_t mutex_;
  static pthread_cond_t listening_cond_;
  static pthread_cond_t stop_cond_;
};

bool LibnssGoogleTest::is_listening_;
bool LibnssGoogleTest::stop_listening_;
pthread_mutex_t LibnssGoogleTest::mutex_;
pthread_cond_t LibnssGoogleTest::listening_cond_;
pthread_cond_t LibnssGoogleTest::stop_cond_;

TEST_F(LibnssGoogleTest, CopyStringNormalCase) {
  char buffer[16];
  BufferManager buf(buffer, sizeof(buffer));
  std::string value = "test";

  char* result = buf.AppendString(value);
  EXPECT_STREQ("test", result);
  EXPECT_EQ(buffer, result);
  EXPECT_EQ(11, buf.size());
  EXPECT_EQ(buffer + 5, buf.buffer());
}

TEST_F(LibnssGoogleTest, CopyStringBufferJustBigEnough) {
  char buffer[5];
  BufferManager buf(buffer, sizeof(buffer));
  std::string value = "test";

  char* result = buf.AppendString(value);
  EXPECT_STREQ("test", result);
  EXPECT_EQ(0, buf.size());
}

TEST_F(LibnssGoogleTest, CopyStringBufferTooSmallForNullTerm) {
  char buffer[4];
  BufferManager buf(buffer, sizeof(buffer));
  std::string value = "test";

  ASSERT_THROW(buf.AppendString(value), std::length_error);
}

TEST_F(LibnssGoogleTest, CopyVectorNormalCase) {
  char buffer[64];
  BufferManager buf(buffer, sizeof(buffer));
  std::vector<std::string> value;
  value.push_back("test");
  value.push_back("");
  value.push_back("test2");

  char** result = buf.AppendVector(value);
  EXPECT_STREQ("test", result[0]);
  EXPECT_STREQ("", result[1]);
  EXPECT_STREQ("test2", result[2]);
  EXPECT_EQ(NULL, result[3]);
  size_t data_size = sizeof(char*) * 4 + 12;
  EXPECT_EQ(64 - data_size, buf.size());
  EXPECT_EQ(buffer + data_size, buf.buffer());
}

TEST_F(LibnssGoogleTest, CopyEmptyVector) {
  char buffer[64];
  BufferManager buf(buffer, sizeof(buffer));
  std::vector<std::string> value;

  char** result = buf.AppendVector(value);
  EXPECT_EQ(NULL, result[0]);
}

TEST_F(LibnssGoogleTest, CopyVectorBufferTooSmallForNullTerm) {
  char buffer[sizeof(char*) * 2];
  BufferManager buf(buffer, sizeof(buffer));
  std::vector<std::string> value;
  value.push_back("");

  ASSERT_THROW(buf.AppendVector(value), std::length_error);
}

TEST_F(LibnssGoogleTest, TokenizeStringNormalCase) {
  std::string value = "user:1:2: :dir::";
  std::vector<std::string> result;
  TokenizeString(value, ':', &result);

  ASSERT_EQ(7, result.size());
  EXPECT_STREQ("user", result[0].c_str());
  EXPECT_STREQ("1", result[1].c_str());
  EXPECT_STREQ("2", result[2].c_str());
  EXPECT_STREQ(" ", result[3].c_str());
  EXPECT_STREQ("dir", result[4].c_str());
  EXPECT_STREQ("", result[5].c_str());
  EXPECT_STREQ("", result[6].c_str());
}

TEST_F(LibnssGoogleTest, TokenizeEmptyString) {
  std::string value = "";
  std::vector<std::string> result;
  TokenizeString(value, ',', &result);
  ASSERT_TRUE(result.empty());
}

TEST_F(LibnssGoogleTest, TokenizeSingleDelim) {
  std::string value = ":";
  std::vector<std::string> result;
  TokenizeString(value, ':', &result);

  ASSERT_EQ(2, result.size());
}

TEST_F(LibnssGoogleTest, StringToIdNormalCase) {
  std::string value = "123";
  uint32_t result = ParseId(value);
  EXPECT_EQ(123, result);
}

TEST_F(LibnssGoogleTest, StringToIdInvalidString) {
  std::string value = "1.2";
  ASSERT_THROW(ParseId(value), std::runtime_error);
}

TEST_F(LibnssGoogleTest, StringToIdEmptyString) {
  std::string value = "";
  ASSERT_THROW(ParseId(value), std::runtime_error);
}

TEST_F(LibnssGoogleTest, StringToIdOverflow) {
  std::string value = "9999999999999999999";
  uint32_t result = ParseId(value);
  EXPECT_EQ(UINT32_MAX, result);
}

TEST_F(LibnssGoogleTest, StringToIdNegative) {
  std::string value = "-1";
  ASSERT_THROW(ParseId(value), std::runtime_error);
}

TEST_F(LibnssGoogleTest, GetDaemonOutputNormalCase) {
  std::string command = "get_users\n";
  std::string response = "200\n"
      "user2:x:1002:1001::/home/user2:/bin/bash\n"
      "user1:x:1001:1001::/home/user1:/bin/bash";
  RequestResponse rr(command, response);
  pthread_t thread;
  pthread_create(&thread, NULL, ServerThreadMain, &rr);
  WaitForServerToListen();
  std::vector<std::string> output_lines;
  GetDaemonOutput(command, utils::kMultiLine, &output_lines);
  ASSERT_EQ(2, output_lines.size());
  EXPECT_STREQ("user2:x:1002:1001::/home/user2:/bin/bash",
               output_lines[0].c_str());
  EXPECT_STREQ("user1:x:1001:1001::/home/user1:/bin/bash",
               output_lines[1].c_str());
  ShutdownServer();
}

TEST_F(LibnssGoogleTest, GetDaemonOutputNonexistantUser) {
  std::string command = "get_user user\n";
  std::string response = "404";
  RequestResponse rr(command, response);
  pthread_t thread;
  pthread_create(&thread, NULL, ServerThreadMain, &rr);
  WaitForServerToListen();
  std::vector<std::string> output_lines;
  ASSERT_THROW(GetDaemonOutput(command, utils::kSingleLine, &output_lines),
               std::invalid_argument);
  ShutdownServer();
}

TEST_F(LibnssGoogleTest, GetDaemonOutputServerError) {
  std::string command = "get_user user\n";
  std::string response = "500";
  RequestResponse rr(command, response);
  pthread_t thread;
  pthread_create(&thread, NULL, ServerThreadMain, &rr);
  WaitForServerToListen();
  std::vector<std::string> output_lines;
  ASSERT_THROW(GetDaemonOutput(command, utils::kSingleLine, &output_lines),
               std::runtime_error);
  ShutdownServer();
}

TEST_F(LibnssGoogleTest, GetDaemonOutputNoOutput) {
  std::string command = "get_user user\n";
  std::string response = "";
  RequestResponse rr(command, response);
  pthread_t thread;
  pthread_create(&thread, NULL, ServerThreadMain, &rr);
  WaitForServerToListen();
  std::vector<std::string> output_lines;
  ASSERT_THROW(GetDaemonOutput(command, utils::kSingleLine, &output_lines),
               std::runtime_error);
  ShutdownServer();
}

TEST_F(LibnssGoogleTest, GetDaemonOutputNoGroups) {
  std::string command = "get_groups\n";
  std::string response = "200";
  RequestResponse rr(command, response);
  pthread_t thread;
  pthread_create(&thread, NULL, ServerThreadMain, &rr);
  WaitForServerToListen();
  std::vector<std::string> output_lines;
  GetDaemonOutput(command, utils::kMultiLine, &output_lines);
  ASSERT_EQ(0, output_lines.size());
  ShutdownServer();
}

TEST_F(LibnssGoogleTest, GetDaemonOutputNoUser) {
  std::string command = "get_user user\n";
  std::string response = "200";
  RequestResponse rr(command, response);
  pthread_t thread;
  pthread_create(&thread, NULL, ServerThreadMain, &rr);
  WaitForServerToListen();
  std::vector<std::string> output_lines;
  ASSERT_THROW(GetDaemonOutput(command, utils::kSingleLine, &output_lines),
               std::runtime_error);
  ShutdownServer();
}

TEST_F(LibnssGoogleTest, GetDaemonOutputConnectDoesNotHangOnConnect) {
  pthread_t thread;
  pthread_create(&thread, NULL, NoAcceptServerThreadMain, NULL);
  WaitForServerToListen();
  std::vector<std::string> output_lines;
  ASSERT_THROW(GetDaemonOutput("", utils::kMultiLine, &output_lines),
               std::runtime_error);
  ShutdownServer();
}

TEST_F(LibnssGoogleTest, GetDaemonOutputReadDoesNotHang) {
  pthread_t thread;
  pthread_create(&thread, NULL, NoResponseServerThreadMain, NULL);
  WaitForServerToListen();
  std::vector<std::string> output_lines;
  ASSERT_THROW(GetDaemonOutput("", utils::kMultiLine, &output_lines),
               std::runtime_error);
  ShutdownServer();
}

TEST_F(LibnssGoogleTest, GetDaemonOutputPartialReadDoesNotHang) {
  pthread_t thread;
  pthread_create(&thread, NULL, PartialResponseServerThreadMain, NULL);
  WaitForServerToListen();
  std::vector<std::string> output_lines;
  ASSERT_THROW(GetDaemonOutput("", utils::kMultiLine, &output_lines),
               std::runtime_error);
  ShutdownServer();
}

TEST_F(LibnssGoogleTest, GetDaemonOutputHandlesNoSocketGracefully) {
  std::vector<std::string> output_lines;
  ASSERT_THROW(GetDaemonOutput("", utils::kMultiLine, &output_lines),
               std::runtime_error);
}

TEST_F(LibnssGoogleTest, EntityListNormalCase) {
  std::string command = "get_users\n";
  std::string response = "200\n"
      "user2:x:1002:1001::/home/user2:/bin/bash\n"
      "user1:x:1001:1001::/home/user1:/bin/bash";
  RequestResponse rr(command, response);
  pthread_t thread;
  pthread_create(&thread, NULL, ServerThreadMain, &rr);
  WaitForServerToListen();
  EntityList list;
  list.Load(command);
  EXPECT_STREQ("user2:x:1002:1001::/home/user2:/bin/bash",
               list.Pop().c_str());
  EXPECT_STREQ("user1:x:1001:1001::/home/user1:/bin/bash",
               list.Pop().c_str());
  ASSERT_THROW(list.Pop(), std::out_of_range);
  ShutdownServer();
}

TEST_F(LibnssGoogleTest, EntityListClear) {
  std::string command = "get_users\n";
  std::string response = "200\n"
      "user2:x:1002:1001::/home/user2:/bin/bash\n"
      "user1:x:1001:1001::/home/user1:/bin/bash";
  RequestResponse rr(command, response);
  pthread_t thread;
  pthread_create(&thread, NULL, ServerThreadMain, &rr);
  WaitForServerToListen();
  EntityList list;
  list.Load(command);
  EXPECT_STREQ("user2:x:1002:1001::/home/user2:/bin/bash",
               list.Pop().c_str());
  list.Clear();
  ASSERT_THROW(list.Pop(), std::out_of_range);
  ShutdownServer();
}

TEST_F(LibnssGoogleTest, UserLineToPasswdStructNormalCase) {
  std::string value = "jsmith:1001:1000:Joe Smith,Room 1007,(234)555-8910,"
      "(234)555-0044,email:/home/jsmith:/bin/sh";
  passwd result;
  char buffer[128];
  BufferManager buf(buffer, sizeof(buffer));
  UserLineToPasswdStruct(value, &result, &buf);
  EXPECT_STREQ("jsmith", result.pw_name);
  EXPECT_STREQ("x", result.pw_passwd);
  EXPECT_EQ(1001, result.pw_uid);
  EXPECT_EQ(1000, result.pw_gid);
  EXPECT_STREQ("Joe Smith,Room 1007,(234)555-8910,(234)555-0044,email",
               result.pw_gecos);
  EXPECT_STREQ("/home/jsmith", result.pw_dir);
  EXPECT_STREQ("/bin/sh", result.pw_shell);
}

TEST_F(LibnssGoogleTest, UserLineToPasswdStructInvalid) {
  std::string value = "jsmith:1001:1000";
  passwd result;
  char buffer[128];
  BufferManager buf(buffer, sizeof(buffer));
  ASSERT_THROW(UserLineToPasswdStruct(value, &result, &buf),
               std::runtime_error);
}

TEST_F(LibnssGoogleTest, GroupLineToGroupStructNormalCase) {
  std::string value = "sudoers:1002:user1,user2,user3";
  group result;
  char buffer[128];
  BufferManager buf(buffer, sizeof(buffer));
  GroupLineToGroupStruct(value, &result, &buf);
  EXPECT_STREQ("sudoers", result.gr_name);
  EXPECT_STREQ("x", result.gr_passwd);
  EXPECT_EQ(1002, result.gr_gid);
  EXPECT_STREQ("user1", result.gr_mem[0]);
  EXPECT_STREQ("user2", result.gr_mem[1]);
  EXPECT_STREQ("user3", result.gr_mem[2]);
  EXPECT_EQ(NULL, result.gr_mem[3]);
}

TEST_F(LibnssGoogleTest, GroupLineToGroupStructEmptyGroup) {
  std::string value = "admins:1003:";
  group result;
  char buffer[128];
  BufferManager buf(buffer, sizeof(buffer));
  GroupLineToGroupStruct(value, &result, &buf);
  EXPECT_STREQ("admins", result.gr_name);
  EXPECT_STREQ("x", result.gr_passwd);
  EXPECT_EQ(1003, result.gr_gid);
  EXPECT_EQ(NULL, result.gr_mem[0]);
}

TEST_F(LibnssGoogleTest, GroupLineToGroupStructInvalid) {
  std::string value = "group:";
  group result;
  char buffer[128];
  BufferManager buf(buffer, sizeof(buffer));
  ASSERT_THROW(GroupLineToGroupStruct(value, &result, &buf),
               std::runtime_error);
}

TEST_F(LibnssGoogleTest, AccountNameToShadowStructNormalCase) {
  std::string value = "jsmith";
  spwd result;
  char buffer[128];
  BufferManager buf(buffer, sizeof(buffer));
  AccountNameToShadowStruct(value, &result, &buf);
  EXPECT_STREQ("jsmith", result.sp_namp);
  EXPECT_STREQ("*", result.sp_pwdp);
  EXPECT_EQ(-1, result.sp_lstchg);
  EXPECT_EQ(-1, result.sp_min);
  EXPECT_EQ(-1, result.sp_max);
  EXPECT_EQ(-1, result.sp_warn);
  EXPECT_EQ(-1, result.sp_inact);
  EXPECT_EQ(-1, result.sp_expire);
  EXPECT_EQ(-1, result.sp_flag);
}

TEST_F(LibnssGoogleTest, AccountNameToShadowStructInvalid) {
  std::string value = "j:smith";
  spwd result;
  char buffer[128];
  BufferManager buf(buffer, sizeof(buffer));
  ASSERT_THROW(AccountNameToShadowStruct(value, &result, &buf),
               std::runtime_error);
}
