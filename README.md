# Github-team-sync Operator

Synchronize group membership in OpenShift to team membership in GitHub.

## Description

The github-team-sync-operator automatically synchronized OpenShift group membership to GitHub team membership. The operator looks for groups with the `github.oddbit.com/sync` label set to `"true"`. These groups must have the follow annotations:

- `github.oddbit.com/organization` (required): The name of the organization to which the named team belongs.
- `github.oddbit.com/team` (required): Synchronize membership with this team.
- `github.oddbit.com/secret` (optional): The name of a secret that contains a `GITHUB_TOKEN` key with a GitHub personal access token with appropriate access to get team memberships. If this annotation is not available, github-team-sync will attempt to use the global token (see [Configuration](#configuration)).

An example `Group` might look like this:

```
apiVersion: user.openshift.io/v1
kind: Group
metadata:
  name: example-group
  labels:
    github.oddbit.com/sync: "true"
  annotations:
    github.oddbit.com/organization: example-org
    github.oddbit.com/team: example-team
users: []
```

The operator will run immediately whenever a group is created or modified; it will also run periodically (by default every 5 minutes).

## Configuration

Github-team-sync reads environment variables from the `ConfigMap` `github-team-sync-manager-config` and from the `Secret` `github-team-sync-manager-secret`. You may set the following keys:

- `GITHUB_TOKEN`: A personal access token with permissions to read team memberships. This can be overridden per group.
- `GITHUB_SYNC_PERIOD`: How often to run the reconcile loop absent any changes. This default to 5 minutes. The value of this key will be read using [`time.ParseDuration`](https://pkg.go.dev/time#ParseDuration).

## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
