[![Build Status](https://travis-ci.org/pmdarrow/mergeit.svg?branch=master)](https://travis-ci.org/pmdarrow/mergeit)

# mergeit

- Keeps a PR branch up to date with master as master changes
- if PR can be merged (all things approved, for example) it will do so
- add it to a slack channel
- trigger inside slack via `@mergeit merge https://github.com/full/repo/pull/123`

## build

 `go dep`
 `go build`

## run

Set ENVs:
```
SLACK_ACCESS_TOKEN
GH_ACCESS_TOKEN
```

then `./mergeit`

Or, you know, deploy to <insert hosted thing here>
