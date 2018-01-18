[![Build Status](https://travis-ci.org/pmdarrow/mergeit.svg?branch=master)](https://travis-ci.org/pmdarrow/mergeit)

# mergeit!

- Keeps a PR up to date by continually merging in the base branch as it changes
- When all status checks pass and the PR is up-to-date, it will be squashed and merged

## Build

```
$ dep ensure
$ go build
```

## Run

```
$ export SLACK_ACCESS_TOKEN=secret-here
$ export GH_ACCESS_TOKEN=secret-here
$ ./mergeit
```

## Usage

- Invite @mergeit to a Slack channel
- Tell it to merge your PR: `@mergeit merge https://github.com/full/repo/pull/123`
