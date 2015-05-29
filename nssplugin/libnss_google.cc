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

#include <errno.h>
#include <nss.h>
#include <pthread.h>
#include <stdio.h>
#include <stdexcept>
#include <sstream>

#include "utils.h"  // NOLINT(build/include)

extern "C" {

  nss_status _nss_google_getpwnam_r(const char* name, passwd* pwd, char* buf,
                                    size_t buflen, int* errnop) {
    utils::BufferManager buffer(buf, buflen);
    nss_status status = NSS_STATUS_SUCCESS;
    std::vector<std::string> output_lines;
    std::stringstream command;
    command << "user_by_name " << name;
    try {
      utils::GetDaemonOutput(command.str(), utils::kSingleLineExtendedTimeout,
                             &output_lines);
      utils::UserLineToPasswdStruct(output_lines[0], pwd, &buffer);
    } catch (const std::length_error&) {
      *errnop = ERANGE;
      status = NSS_STATUS_TRYAGAIN;
    } catch (const std::invalid_argument&) {
      *errnop = ENOENT;
      status = NSS_STATUS_NOTFOUND;
    } catch (const std::exception&) {
      *errnop = ENOENT;
      status = NSS_STATUS_TRYAGAIN;
    }
    return status;
  }

  nss_status _nss_google_getpwuid_r(uid_t uid, passwd* pwd, char* buf,
                                    size_t buflen, int* errnop) {
    utils::BufferManager buffer(buf, buflen);
    nss_status status = NSS_STATUS_SUCCESS;
    std::vector<std::string> output_lines;
    std::stringstream command;
    command << "user_by_uid " << uid;
    try {
      utils::GetDaemonOutput(command.str(), utils::kSingleLine, &output_lines);
      utils::UserLineToPasswdStruct(output_lines[0], pwd, &buffer);
    } catch (const std::length_error&) {
      *errnop = ERANGE;
      status = NSS_STATUS_TRYAGAIN;
    } catch (const std::invalid_argument&) {
      *errnop = ENOENT;
      status = NSS_STATUS_NOTFOUND;
    } catch (const std::exception&) {
      *errnop = ENOENT;
      status = NSS_STATUS_TRYAGAIN;
    }
    return status;
  }

  utils::EntityList g_pw_entities;

  nss_status _nss_google_setpwent() {
    nss_status status = NSS_STATUS_SUCCESS;
    const char* command = "users";
    try {
      g_pw_entities.Load(command);
    } catch (const std::exception&) {
      errno = ENOENT;
      status = NSS_STATUS_TRYAGAIN;
    }
    return status;
  }

  nss_status _nss_google_endpwent() {
    g_pw_entities.Clear();
    return NSS_STATUS_SUCCESS;
  }

  nss_status _nss_google_getpwent_r(passwd* pwd, char* buf, size_t buflen,
                                    int* errnop) {
    utils::BufferManager buffer(buf, buflen);
    nss_status status = NSS_STATUS_SUCCESS;
    try {
      utils::UserLineToPasswdStruct(g_pw_entities.Pop(), pwd, &buffer);
    } catch (const std::out_of_range&) {
      status = NSS_STATUS_NOTFOUND;
    } catch (const std::length_error&) {
      *errnop = ERANGE;
      status = NSS_STATUS_TRYAGAIN;
    } catch (const std::exception&) {
      *errnop = ENOENT;
      status = NSS_STATUS_TRYAGAIN;
    }
    return status;
  }

  nss_status _nss_google_getgrnam_r(const char* name, group* grp, char* buf,
                                    size_t buflen, int* errnop) {
    utils::BufferManager buffer(buf, buflen);
    nss_status status = NSS_STATUS_SUCCESS;
    std::vector<std::string> output_lines;
    std::stringstream command;
    command << "group_by_name " << name;
    try {
      utils::GetDaemonOutput(command.str(), utils::kSingleLine, &output_lines);
      utils::GroupLineToGroupStruct(output_lines[0], grp, &buffer);
    } catch (const std::length_error&) {
      *errnop = ERANGE;
      status = NSS_STATUS_TRYAGAIN;
    } catch (const std::invalid_argument&) {
      *errnop = ENOENT;
      status = NSS_STATUS_NOTFOUND;
    } catch (const std::exception&) {
      *errnop = ENOENT;
      status = NSS_STATUS_TRYAGAIN;
    }
    return status;
  }

  nss_status _nss_google_getgrgid_r(uid_t gid, group* grp, char* buf,
                                    size_t buflen, int* errnop) {
    utils::BufferManager buffer(buf, buflen);
    nss_status status = NSS_STATUS_SUCCESS;
    std::vector<std::string> output_lines;
    std::stringstream command;
    command << "group_by_gid " << gid;
    try {
      utils::GetDaemonOutput(command.str(), utils::kSingleLine, &output_lines);
      utils::GroupLineToGroupStruct(output_lines[0], grp, &buffer);
    } catch (const std::length_error&) {
      *errnop = ERANGE;
      status = NSS_STATUS_TRYAGAIN;
    } catch (const std::invalid_argument&) {
      *errnop = ENOENT;
      status = NSS_STATUS_NOTFOUND;
    } catch (const std::exception&) {
      *errnop = ENOENT;
      status = NSS_STATUS_TRYAGAIN;
    }
    return status;
  }

  utils::EntityList g_gr_entities;

  nss_status _nss_google_setgrent() {
    nss_status status = NSS_STATUS_SUCCESS;
    const char* command = "groups";
    try {
      g_gr_entities.Load(command);
    } catch (const std::exception&) {
      errno = ENOENT;
      status = NSS_STATUS_TRYAGAIN;
    }
    return status;
  }

  nss_status _nss_google_endgrent() {
    g_gr_entities.Clear();
    return NSS_STATUS_SUCCESS;
  }

  nss_status _nss_google_getgrent_r(group* grp, char* buf, size_t buflen,
                                    int* errnop) {
    utils::BufferManager buffer(buf, buflen);
    nss_status status = NSS_STATUS_SUCCESS;
    try {
      utils::GroupLineToGroupStruct(g_gr_entities.Pop(), grp, &buffer);
    } catch (const std::out_of_range&) {
      status = NSS_STATUS_NOTFOUND;
    } catch (const std::length_error&) {
      *errnop = ERANGE;
      status = NSS_STATUS_TRYAGAIN;
    } catch (const std::exception&) {
      *errnop = ENOENT;
      status = NSS_STATUS_TRYAGAIN;
    }
    return status;
  }

  nss_status _nss_google_getspnam_r(const char* name, spwd* pwd, char* buf,
                                    size_t buflen, int* errnop) {
    utils::BufferManager buffer(buf, buflen);
    nss_status status = NSS_STATUS_SUCCESS;
    std::vector<std::string> output_lines;
    std::stringstream command;
    command << "is_name " << name;
    try {
      utils::GetDaemonOutput(command.str(), utils::kMultiLine, &output_lines);
      utils::AccountNameToShadowStruct(name, pwd, &buffer);
    } catch (const std::length_error&) {
      *errnop = ERANGE;
      status = NSS_STATUS_TRYAGAIN;
    } catch (const std::invalid_argument&) {
      *errnop = ENOENT;
      status = NSS_STATUS_NOTFOUND;
    } catch (const std::exception&) {
      *errnop = ENOENT;
      status = NSS_STATUS_TRYAGAIN;
    }
    return status;
  }

  utils::EntityList g_sp_entities;

  nss_status _nss_google_setspent() {
    nss_status status = NSS_STATUS_SUCCESS;
    const char* command = "names";
    try {
      g_sp_entities.Load(command);
    } catch (const std::exception&) {
      errno = ENOENT;
      status = NSS_STATUS_TRYAGAIN;
    }
    return status;
  }

  nss_status _nss_google_endspent() {
    g_sp_entities.Clear();
    return NSS_STATUS_SUCCESS;
  }

  nss_status _nss_google_getspent_r(spwd* pwd, char* buf, size_t buflen,
                                    int* errnop) {
    utils::BufferManager buffer(buf, buflen);
    nss_status status = NSS_STATUS_SUCCESS;
    try {
      utils::AccountNameToShadowStruct(g_sp_entities.Pop(), pwd, &buffer);
    } catch (const std::out_of_range&) {
      status = NSS_STATUS_NOTFOUND;
    } catch (const std::length_error&) {
      *errnop = ERANGE;
      status = NSS_STATUS_TRYAGAIN;
    } catch (const std::exception&) {
      *errnop = ENOENT;
      status = NSS_STATUS_TRYAGAIN;
    }
    return status;
  }

}  // extern "C"
