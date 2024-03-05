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
