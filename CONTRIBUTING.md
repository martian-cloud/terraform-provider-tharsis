# Contributing to Tharsis projects.

There are many ways you can contribute to Tharsis projects.  This document describes some of those ways.  It also describes a few things we request you do as part of making any code or documentation contributions.

## Prerequisites

* **Go >= 1.18** ( [https://golang.org/dl/](https://golang.org/dl/) or [https://golang.org/doc/install](https://golang.org/doc/install) )

## Ways to Contribute

- Report bugs.
- Raise security issues.
- Suggest features or enhancements.
- Make and submit changes to fix bugs or add/enhance functionality.
- Write documentation.
- Answer questions other users ask or might have.
- Write tests.

## Reporting Bugs

- Search the existing GitLab issues to see if someone else already reported it.
- Make sure you're using the latest version of the appropriate project(s) in case it might have already been fixed in a later version.
- If no existing issue matches, file a new GitLab issue.
- Please use [this GitLab-supplied template.](https://gitlab.com/gitlab-org/gitlab/-/blob/master/.gitlab/issue_templates/Bug.md)
- Make sure the following items are clearly described in your bug report:
    - the versions of Tharsis projects you're using
    - steps to reproduce the problem
    - how actual results differ from expected results
- If you can include a patch as a proposed fix, please do so.

## Security Issues

If you have discovered something that appears to be a security issue, please report it to the email address listed in the README.md file.

## Suggesting Features or Enhancements

Suggestions for features and enhancements are not required to include a code contribution.  To avoid wasting your time, if you plan to contribute code to implement the feature or enhancement you are suggesting, please file an issue before doing substantial work on the code contribution.  If you are submitting only a suggestion, please make your suggestion as complete and precise as reasonably possible.

## Making Changes (bug fixes or enhancements)

If the project in question has existing unit and/or integration tests, before submitting a code contribution (whether it is a bug fix or an enhancement) make sure to run all available tests:

    make test

    make integration

If the existing tests don't pass with your code contribution, your contribution cannot be accepted until that problem has been resolved.

### Formatting and Style

Please respect the formatting of the project codebase:

- tabs rather than spaces for indentation (we set our IDE to display two spaces for a tab)
- standard Go formatting and error scanning:

    make fmt

    make vet

- we generally try to follow the guidelines in this guide: [Uber's Go styling](https://github.com/uber-go/guide/blob/master/style.md)

## Writing Documentation

If your talents lean more toward writing documentation than code, your contributions of documentation are welcome.  Please make sure your contribution of documentation is accurate.  Also, please try to make it consistent in style with the existing documentation.  There may be other guidelines for documentation style published elsewhere in the project.

## Submitting Changes

- do your development in a feature or bug-fix branch based on "main"
- please submit your contribution of code or documentation as a Git pull request
- please respond as promptly as you can to feedback regarding your contribution (in order to save your time and ours)

## Answering Questions

If your talents include answering questions asked by other users, we encourage you to do so in considerate and helpful ways.  In time, we may establish a discussion forum or other official place to discuss use of Tharsis projects.

## Testing

If you are adding significant new features or functionality, please include unit tests in your contribution.  For larger contributions, you are welcome to include integration tests.

When writing unit tests, please use mocks where appropriate.

## Contributor License Agreement (CLA)

If we have published a Contributor License Agreement prior to the time you submit a contribution, make sure to sign and submit the agreement before or along with your contribution.

## Licensing of Your Contributions

Your contributions will become licensed under the [Mozilla Public License v2.0](https://www.mozilla.org/en-US/MPL/2.0/)
