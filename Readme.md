
## Triage

Interactive command-line GitHub issue & notification triaging tool.

## Installation

Via `go get`:

```
$ GO111MODULE=on go get github.com/tj/triage/cmd/triage@master
```

Via `ops run` by [CTO.ai](https://cto.ai/):

```
$ npm install -g @cto.ai/ops && ops account:signup
$ ops run @tj/triage
```

## Features

Some of the current features include:

- Quickly view and search notifications
- View issue details, labels, and comments
- View notifications without marking them as read
- Mark notifications as read, or unsubscribe entirely
- Unwatch entire repositories
- Add and remove issue labels
- Add comments to issues

Upcoming features may include things like:

- Global priority management across all of your projects
- Automatically prioritize based on your GitHub sponsors
- Templated comment responses

## Screenshots

Notifications listing:

![](https://apex-software.imgix.net/github/tj/triage/notifications.png)

Filtering notifications with the `/` search:

![](https://apex-software.imgix.net/github/tj/triage/search.png)

Viewing issue details:

![](https://apex-software.imgix.net/github/tj/triage/issue.png)

Adding and removing labels:

![](https://apex-software.imgix.net/github/tj/triage/labels.png)

Leaving a comment:

![](https://apex-software.imgix.net/github/tj/triage/comment.png)

---

[![GoDoc](https://godoc.org/github.com/tj/triage?status.svg)](https://godoc.org/github.com/tj/triage)
![](https://img.shields.io/badge/license-MIT-blue.svg)
![](https://img.shields.io/badge/status-stable-green.svg)

## Sponsors

This project is [sponsored](https://github.com/sponsors/tj) by [CTO.ai](https://cto.ai/), making it easy for development teams to create and share workflow automations without leaving the command line.

[![](https://apex-software.imgix.net/github/sponsors/cto.png)](https://cto.ai/)