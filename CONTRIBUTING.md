# Contributing Guidelines for Nephio

We welcome all contributions, suggestions, and feedback, so please do not hesitate to reach out!

- [Contributing Guidelines for Nephio](#contributing-guidelines-for-nephio)
  - [Engage with us](#engage-with-us)
  - [Ways you can contribute](#ways-you-can-contribute)
    - [1. Report issues](#1-report-issues)
    - [2. Fix or Improve Documentation](#2-fix-or-improve-documentation)
    - [3. Submit Pull Requests](#3-submit-pull-requests)
      - [How to Create a PR](#how-to-create-a-pr)
      - [Developer Certificate of Origin (DCO) Sign off](#developer-certificate-of-origin-dco-sign-off)
      - [Code Review of the Pull Request](#code-review-of-the-pull-request)
      - [Possible problems and quirks](#possible-problems-and-quirks)

## Engage with us

The Nephio website has the most updated information on [how to engage with the Nephio community](https://wiki.nephio.org/display/HOME/How+To+Join+Slack).

Join our community meetings to learn more about Nephio and engage with other contributors.

## Ways you can contribute

### 1. Report issues

Issues to Nephio help improve the project in multiple ways including the following:

- Report potential bugs
- Request a feature

### 2. Fix or Improve Documentation

The [Nephio docs website](https://github.com/nephio-project/docs), like the main Nephio codebase, is stored in its own [git repo](https://github.com/nephio-project/docs). To get started with contributions to the documentation, [follow the guide](https://github.com/nephio-project/docs/blob/main/README.md) on that repository.

### 3. Submit Pull Requests

[Pull requests](https://docs.github.com/en/github/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/about-pull-requests) (PRs) allow you to contribute back the changes you've made on your side enabling others in the community to benefit from your hard work. They are the main source by which all changes are made to this project and are a standard piece of GitHub operational flows.

New contributors may easily view all [open issues labeled as good first issues](https://github.com/nephio-project/nephio/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22) allowing you to get started in an approachable manner.


In the process of submitting your PRs, please read and abide by the template provided to ensure the maintainers are able to understand your changes and quickly come up to speed. There are some important pieces that are required outside the code itself. Some of these are up to you, others are up to the maintainers.

1. Provide Proof Manifests allowing the maintainers and other contributors to verify your changes without requiring they understand the nuances of all your code.
2. For new or changed functionality, this typically requires documentation and so raise a corresponding issue (or, better yet, raise a separate PR) on the [documentation repository](https://github.com/nephio-project/docs).
3. Indicate which release this PR is triaged for (maintainers). This step is important especially for the documentation maintainers in order to understand when and where the necessary changes should be made.

#### How to Create a PR

Head over to the project repository on GitHub and click the **"Fork"** button. With the forked copy, you can try new ideas and implement changes to the project.

1. **Clone the repository to your device:**

Get the link of your forked repository, paste it in your device terminal and clone it using the command.

```sh
git clone https://hostname/YOUR-USERNAME/YOUR-REPOSITORY
```

2. **Create a branch:**

Create a new brach and navigate to the branch using this command.

```sh
git checkout -b <new-branch>
```

Great, it's time to start hacking! You can now go ahead to make all the changes you want.

3. **Stage, Commit, and Push changes:**

Now that we have implemented the required changes, use the command below to stage the changes and commit them.

```sh
git add .
```

```sh
git commit -s -m "Commit message"
```

The `-s` signifies that you have signed off the commit.

Go ahead and push your changes to GitHub using this command.

```sh
git push
```

#### Cherry-pick PRs to release branches

Add repository as remote 

```sh
git remote add <name> https://github.com/nephio-project/nephio
```
Then fetch the branches of remote:

```sh
git fetch <name>
```

 You will notice that there are a number of branches related to Nephio's releases such as release-1.7. You can always view the list of remote branches by using the command below:

```sh
$ git branch -r
...
origin/release-1.5
origin/release-1.6
origin/release-1.7
```

Checkout one of the release branch and cherry-pick the PRs you want to merge into the release branch:

```sh
$ git checkout release-1.7

git cherry-pick <commit-hash> -s

git push --set-upstream origin release-1.7
```

Once the commit has been cherry-picked, the author will need to open a PR merging to the release branch, release-1.7 for example.

#### Developer Certificate of Origin (DCO) Sign off

For contributors to certify that they wrote or otherwise have the right to submit the code they are contributing to the project, we are requiring everyone to acknowledge this by signing their work which indicates you agree to the DCO found [here](https://developercertificate.org/).

To sign your work, just add a line like this at the end of your commit message:

```sh
Signed-off-by: Random J Developer <random@developer.example.org>
```

This can easily be done with the `-s` command line option to append this automatically to your commit message.

```sh
git commit -s -m 'This is my commit message'
```
#### Code Review of the Pull Request

In Nephio Project uses [Prow](https://docs.prow.k8s.io/) to among many other tasks test and manage the Pull Requests. Prow also allows to interact with PR trough ['slash' commands](https://prow.nephio.io/command-help)

##### The Code Review Process
The **author** submits a PR
- Step 0: [Prow bot](https://github.com/apps/nephio-prow) assigns reviewers and approvers for the PR based on [OWNERS files](https://www.kubernetes.dev/docs/guide/owners/) please note that single repository can have multiple OWNERS files, bot will choose reviewers and approvers from file nearest to changed code 
- Step 1: Prow tests the PR
If the author is not (yet) a member of Nephio GitHub organization Prow will add label 'needs-ok-to-test' which will prevent Prow from running tests on it. This is a security measure as many tests allow to execute arbitrary code, create objects in infrastructure and so on. To allow the tests to run any member of organization can use command
/ok-to-test in comment, then tests will run. Status of their execution will be visible on PR page itself or if one wants to see logs, previous runs etc those are available on [Prow's Dashboard](https://prow.nephio.io/)
Definions of those tests are either in local [inrepoconfig](https://docs.prow.k8s.io/docs/inrepoconfig/) file .prow.yaml or in [central Prow configuration](https://github.com/nephio-project/test-infra/tree/main/prow/config)
- Step 2: Review of the PR
Reviewers check the code quality, correctness, software engineering best practices, coding style and so on.
Anyone in the organization can act as a reviewer with the exception of the individual who opened the PR
If PR content looks good to them, a reviewer types /lgtm (**looks** **good** **to** **me**) in the PR comment or review; Prow bot will then apply 'lgtm' label to the PR, this can be canceled by removing label or using '/lgtm cancel' command
- Step 3: Approval of the PR
Only people listed in the relevant OWNERS file in section 'approvers', either directly or through an alias, can act as approvers, including the individual who opened the PR.
Approvers check for acceptance criteria, compatibility with other features, forwards/backwards compatibility, API definitions and so on.
If the code changes gets their approval, an approver types /approve in a PR comment or review; this as well can be canceled with '/approve cancel' command
Prow bot applies an approved label
- Step 4: Automation merges the PR
If all of the following conditions are met:
  - All required labels are present (lgtm, approved)
  - No blocking labels are present (do-not-merge/hold, needs-rebase)
  - There are no presubmit prow jobs failing 
  - Then the PR will automatically be merged

##### Possible problems and quirks

- Approval and lgtm process
  - Technically anyone who is a member of the Nephio GitHub organization can /lgtm a PR which can be both good (reviews from non-members are encouraged as a way of demonstrating experience and intent to become a member or reviewer) and bad (/lgtm’s from members may be a sign that our OWNERS files are too small, or that the existing reviewers are too busy). Note to approvers (who can do both approve and lgtm) - if possible please leave lgtm to other in the spirit of having at least two sets of eyes on every change.
  - Reviewers, and approvers are unresponsive
Please do not rely on GitHub notifications sent to approvers/reviewers or mentions like “pinging @the_rewiever for approval” - those are really hard to filter and reviewers/approvers are very busy people. If your PR doesn't get enough attention consider asking for the review in respective Slack channels.
- Prow things
  - If the presubmit job fails there's a message posted by Prow bot that explains steps how to re-run it, for example with command /test my_awesome_presubmit_test
  - Sometimes it takes a while to merge the PR after it was approved, PR queue can be observed on [Prow's Tide status dashboard](https://prow.nephio.io/tide)
  - If the PR gets final step like approve or lgtm done after 12 hours pass since presubmit tests were run they are getting re-triggered
- GitHub things
  - EasyCLA sometimes improperly checks the status, it can be re-triggered with `/easycla` command or it often sorts out when you close and re-open the PR
