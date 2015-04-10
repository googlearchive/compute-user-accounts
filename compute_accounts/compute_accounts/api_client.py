# Copyright 2015 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Client for sending requests to Compute Accounts service from a GCE VM."""

import logging
import re
import urlparse

from . import entities
from . import exceptions
from . import response_schemas
from . import token_bucket
import jsonschema
import requests
import simplejson

_METADATA_ROOT_URL = 'http://metadata.google.internal/computeMetadata/v1/'
_VIEW_PATH = 'projects/{project}/zones/{zone}/{view_name}'
_INSTANCE_PATH = 'projects/{project}/zones/{zone}/instances/{instance_name}'
_LINUX_VIEWS_RATE_LIMIT = (3, 5 * 60)  # 3 request burst. 1 request / 5 min.
_KEYS_VIEW_RATE_LIMIT = (10, 60)  # 10 request burst. 1 request / 1 min.


class ApiClient(object):
  """Class for sending requests to GCE Guest Accounts API.

  This class is thread-safe.
  """

  def __init__(self, api_root, computeaccounts_api_version,
               compute_api_version):
    """Inits the ApiClient to send request to the API at the specified root."""
    self._logger = logging.getLogger(__name__)
    self._name_regex = re.compile(response_schemas.NAME_REGEX)

    api_root = api_root.rstrip('/')
    view_path = '/'.join(['computeaccounts', computeaccounts_api_version,
                          _VIEW_PATH])
    instance_path = '/'.join(['compute', compute_api_version, _INSTANCE_PATH])
    self._view_url_format = urlparse.urljoin(api_root, view_path)
    self._instance_url_format = urlparse.urljoin(api_root, instance_path)

    self._linux_views_bucket = token_bucket.TokenBucket(
        *_LINUX_VIEWS_RATE_LIMIT)
    self._keys_view_bucket = token_bucket.TokenBucket(*_KEYS_VIEW_RATE_LIMIT)

  def get_users_and_groups(self, for_user_name=None):
    """Retrieve a list of users and groups in the project.

    Args:
      for_user_name: The name of the user who's missing lookup triggered this
      refresh. If this is a scheduled refresh, then None.

    Returns:
      A tuple of users and a tuple of groups.
    """
    self._logger.info('Fetching users and groups.')
    if for_user_name:
      params = {'user': for_user_name}
    else:
      params = None
    json = self._retrieve_view('linuxAccountViews',
                               self._linux_views_bucket,
                               response_schemas.LINUX_VIEWS_SCHEMA,
                               params)
    users = tuple([entities.json_to_user(u) for u in json.get('userViews', [])])
    groups = tuple([entities.json_to_group(g)
                    for g in json.get('groupViews', [])])
    return users, groups

  def get_authorized_keys(self, username):
    """Retrieve a list of authorized key strings for a particular user."""
    self._logger.info('Fetching authorized keys: [%s].', username)
    if not self._name_regex.match(username):
      raise exceptions.NotFoundException('Invalid username.')
    json = self._retrieve_view('authorizedKeysView/' + username,
                               self._keys_view_bucket,
                               response_schemas.AUTHORIZED_KEYS_VIEW_SCHEMA)
    return entities.json_to_authorized_keys(json)

  def _retrieve_view(self, view_name, quota_bucket, response_schema,
                     params=None):
    """Get JSON from the Compute Accounts API for the specified view.

    Sends an HTTP POST request to retrieve the given Compute Accounts Linux
    view. The response is validated based on the given JSON schema.

    Args:
      view_name: The name of the Linux view to retrieve.
      quota_bucket: The token bucket that should be charged for the request.
      response_schema: The JSON schema that the response is validated against.
      params: Additional parameters to send with the request.

    Returns:
      A dict corresponding to the JSON response from the API.

    Raises:
      exceptions.BackendException: An error occured while sending the HTTP
        request or parsing the response.
      exceptions.NotFoundException: The requested view does not exist.
      exceptions.OutOfQuotaException: There is not enough quota to send the
        request.
    """
    try:
      quota_bucket.consume()
      project = self._get_project_name()
      zone = self._get_zone_name()
      instance_name = self._get_instance_name()
      headers = self._get_authorization_header()
      url = self._view_url_format.format(
          project=project, zone=zone, view_name=view_name)
      instance_url = self._instance_url_format.format(
          project=project, zone=zone, instance_name=instance_name)
      query_params = {'instance': instance_url}
      if params:
        query_params.update(params)

      self._logger.info('Sending request: [%s].', url)
      self._logger.debug('Request query params: [%s].', query_params)
      self._logger.debug('Request headers: [%s].', headers)
      response = requests.post(url, params=query_params, headers=headers,
                               verify=True)  # Validate SSL certs.
      if response.status_code == requests.codes.not_found:
        raise exceptions.NotFoundException('URL not found: [{}]', response.url)
      response.raise_for_status()
      self._logger.debug('Recieved response: [%s]', response.text)
      json = response.json()
      jsonschema.validate(json, response_schema)
      self._logger.info('Request succeeded.')
      return json.get('resource', {})
    except requests.HTTPError as e:
      self._logger.exception('Request failed.')
      raise exceptions.BackendException(
          e, 'Http error while sending request: [{}]', e.response.text)
    except requests.RequestException as e:
      self._logger.exception('Request failed.')
      raise exceptions.BackendException(e, 'Error while sending request.')
    except simplejson.JSONDecodeError as e:
      self._logger.exception('JSON decode failed.')
      raise exceptions.BackendException(e, 'Parsing JSON failed: [{}]', e.doc)
    except jsonschema.ValidationError as e:
      self._logger.exception('JSON validation failed.')
      raise exceptions.BackendException(
          e, 'Validating JSON failed: [{}]', e.instance)

  def _get_project_name(self):
    """GET the project name for the current VM from the metadata server."""
    self._logger.debug('Fetching project name.')
    return self._get_metadata_value('project/project-id')

  def _get_instance_name(self):
    """GET the instance name for the current VM from the metadata server."""
    self._logger.debug('Fetching instance name.')
    # string.partition() always returns a triple of strings.
    return self._get_metadata_value('instance/hostname').partition('.')[0]

  def _get_zone_name(self):
    """GET the zone name for the current VM from the metadata server."""
    self._logger.debug('Fetching zone name.')
    # string.rpartition() always returns a triple of strings.
    return self._get_metadata_value('instance/zone').rpartition('/')[2]

  def _get_authorization_header(self):
    """GET API authorization for the current VM from the metadata server."""
    self._logger.debug('Fetching authorization token.')
    json = self._get_metadata_value(
        'instance/service-accounts/default/token', json=True)
    jsonschema.validate(json, response_schemas.AUTHORIZATION_SCHEMA)
    authorization = '{} {}'.format(json['token_type'], json['access_token'])
    return {'Authorization': authorization}

  def _get_metadata_value(self, path, json=False):
    """GET metadata value at the specified path from the metadata server."""
    url = urlparse.urljoin(_METADATA_ROOT_URL, path)
    response = requests.get(url, headers={'Metadata-Flavor': 'Google'})
    response.raise_for_status()
    self._logger.debug('Received response: [%s].', response.text)
    if json:
      return response.json()
    else:
      return response.text
