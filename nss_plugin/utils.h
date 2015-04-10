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

#ifndef GCE_ACCOUNTS_UTILS_H_  // NOLINT(build/header_guard)
#define GCE_ACCOUNTS_UTILS_H_  // NOLINT(build/header_guard)

#include <grp.h>
#include <pwd.h>
#include <shadow.h>
#include <stdint.h>
#include <string>
#include <vector>

namespace utils {

enum OutputType { kSingleLine, kMultiLine, kSingleLineExtendedTimeout };

// BufferManager encapsulates and manages a buffer and length.
class BufferManager {
 public:
  // Create a BufferManager that will dole out chunks of buf as requested.
  BufferManager(char* buf, size_t buflen);

  // Copies a string to the buffer.
  //
  // Copied string is guarenteed to be null-terminated.
  //
  // Throws std::length_error exception if the buffer is not large enough to
  // hold the null-terminated string.
  char* AppendString(const std::string& value);

  // Copies a vector of strings to the buffer.
  //
  // Copied vector and all strings are guaranteed to be null-terminated.
  //
  // Throws std::length_error exception if the buffer is not large enough to
  // hold the null-terminated vector.
  char** AppendVector(const std::vector<std::string>& value);

  // Used in tests to verify correct internal structure after use.
  char* buffer() const { return buf_; }
  size_t size() const { return buflen_; }

 private:
  // Return a pointer to a buffer of size bytes.
  void* Reserve(size_t bytes);

  // Throws an std::length_error exception if the buffer cannot hold
  // bytes_to_write.
  void CheckSpaceAvailable(size_t bytes_to_write) const;

  char* buf_;
  size_t buflen_;

  // Not copyable or assignable.
  BufferManager& operator=(const BufferManager&);
  BufferManager(const BufferManager&);
};

// EntityList is a thread-safe list of accounts entities.
class EntityList {
 public:
  EntityList();
  ~EntityList();

  // Loads the entity list using information resulting from the execution of a
  // command.
  void Load(const std::string& command);
  // Empties the entity list.
  void Clear();
  // Returns the next element of the entity list. Throws std::out_of_range
  // exception if the list is empty.
  std::string Pop();

 private:
  pthread_mutex_t mutex_;
  size_t index_;
  std::vector<std::string> output_;
};

// Tokenizes a string according to a delimiter and appends the tokens to the
// result vector.
//
// Empty and trailing tokens are preserved. The empty string has no tokens.
// Examples:
// ("a:b::", ':') => ["a", "b", "", ""]
// (":", ':') => ["", ""]
// ("", ':') => []
void TokenizeString(const std::string& value,
                    char delim,
                    std::vector<std::string>* result);

// Parses an unsigned 32-bit integer represented as a decimal string.
//
// Throws std::runtime_error exception for invalid values.
// Examples: "", "-1", "1.2", "foo"
//
// Returns UINT32_MAX in case of overflow.
uint32_t ParseId(const std::string& value);


// Returns the stdout resulting from the execution of a command. Each line of
// stdout is appended to the output_lines vector with trailing newline
// characters removed.
//
// Command must end with a trailing newline character.
//
// Throws std::invalid_argument exception if execution succeeds, but returns
// result code 404. Throws std::runtime_error exception if execution fails,
// returns an abnormal result code, or does not return the correct number of
// output lines.
void GetDaemonOutput(const std::string& command,
                     OutputType output_type,
                     std::vector<std::string>* output_lines);

// Parses a user information line from the Compute Accounts daemon as a passwd
// entry.
//
// The line should not have a trailing newline character.
//
// Throws std::runtime_error exception if parsing fails.
void UserLineToPasswdStruct(const std::string& line,
                            passwd* pwd,
                            BufferManager* buf);

// Parses a group information line from the Compute Accounts daemon as a group
// entry.
//
// The line should not have a trailing newline character.
//
// Throws std::runtime_error exception if parsing fails.
void GroupLineToGroupStruct(const std::string& line,
                            group* grp,
                            BufferManager* buf);

// Converts and account name from the Compute Accounts daemon to a shadow entry.
//
// The line should not have a trailing newline character.
//
// Throws std::runtime_error exception if conversion fails.
void AccountNameToShadowStruct(const std::string& name,
                               spwd* pwd,
                               BufferManager* buf);

}  // namespace utils

#endif  // GCE_ACCOUNTS_UTILS_H_
