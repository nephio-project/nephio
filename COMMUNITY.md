# Nephio Project Community membership

This document describes the responsibilities and contributor roles in Nephio Project. 

| Role | Responsibilities | Requirements | Defined by |
| -----| ---------------- | ------------ | -------|
| New contributor |  - | Submits the Pull Request on GitHub | Anyone is welcome to participate |
| Member | Active contributor | Sponsored by one reviewer and has several contributions to the project | Nephio GitHub org member|
| Reviewer | Review contributions from other members | History of review and authorship in the project | [OWNERS] file reviewer entry |
| Approver | Contributions acceptance approval| Highly experienced active reviewer and contributor to the project | [OWNERS] file approver entry|

## New contributor

Anyone is welcome to contribute to Nephio project. Existing members should make a point of being helpful 
for new contributors in any way possible (PR workflow, finding relevant documentation, guiding with coding style etc.)
Remember we were all *new contributors* once.

## Member

Members are active contributors in the community. Members might have PRs and GitHub issues assigned to them and are added to GitHub team(s).
Prow's presubmit tests are automatically run for their PRs as well as GitHub Actions (where applicable). 
Members are welcome to remain continuously active contributors to the community.

### Requirements

- Ensure affiliation is up to date in [openprofile.dev]. 
- Have made **several contributions** to the project or community, enough to
  demonstrate an **ongoing and possible long-term commitment** to the project such as:
    - Authoring or reviewing PRs on GitHub, with at least one **merged** PR.
    - Filing or commenting on issues on GitHub
    - Contributing to SIG, subproject, or community discussions (meetings,
      Slack, mailing list)
- Have read the [contributor guide]
- Sponsored by one reviewer
- **Open an issue [Membership request] against the 'nephio' repo**
   - Ensure your sponsor is @mentioned on the issue
   - Provide information described above that is representative of your work on the project
- Have your sponsoring reviewer reply confirmation of sponsorship
- Once your sponsors have responded, your request will be reviewed by the [GitHub Admin team].

### Responsibilities and privileges

- Responsive to issues and PRs assigned to them
- Responsive to mentions of SIG teams they are members of
- Active owner of code they have contributed
  - Code is well tested
  - Tests consistently pass
  - Addresses bugs or issues discovered after code is accepted
- Members can do `/lgtm` on all PRs.
- They can be assigned to issues and PRs, and people can ask members for reviews with a `/cc @username`.
- Tests will be run against their PRs automatically. No `/ok-to-test` needed or approving GitHub Actions run.
- Members can do `/ok-to-test` for PRs that have a `needs-ok-to-test` label, and close PRs.

## Reviewer

Reviewers check code for quality and correctness. They are knowledgeable about both the project (or parts of it) 
and software engineering principles.

### Requirements

The following apply to the part of codebase for which one would be a reviewer in
an [OWNERS] file.

- Member for at least 2 months
- Primary reviewer for at least 5 PRs to the codebase
- Knowledgeable about the project and codebase
- Sponsored by an approver
  - With no objections from other approvers
  - Done through PR to update the OWNERS file: either self-nominate or any other member of the Nephio Project

### Responsibilities and privileges

- Tests are automatically run for their PRs
- Responsible for project quality control via code reviews
- Expected to be responsive to review requests
- Assigned PRs to review related to subproject of expertise
- Assigned test bugs related to subproject of expertise
- Granted "read access" to Nephio Project repo(s)

## Approver

Code approvers are able to both review and approve code contributions (by setting labels `approved` and `lgtm`). 
While code review is focused on code quality and correctness, approval is focused on broad acceptance of a contribution including: backward and/or forward
compatibility, adhering to API, performance and correctness issues, interactions with other parts of project and so on.

### Requirements

- Reviewer of the codebase for at least 2 months
- Primary reviewer for at least 10 PRs to the codebase
- Nominated by a project owner or other approver
  - With no objections from other project owners or approvers
  - Done through PR to update the OWNERS file

### Responsibilities and privileges

- Demonstrate sound technical judgement
- Responsible for project quality control via code reviews
  - Focus on holistic acceptance of contribution such as dependencies with other features, backwards and/or forwards
    compatibility, API and flag definitions, etc
- Expected to be responsive to review requests
- Mentor contributors and reviewers
- May approve code contributions for acceptance


[new contributors]: /CONTRIBUTING.md
[OWNERS]: https://www.kubernetes.dev/docs/guide/owners/
[openprofile.dev]: https://openprofile.dev/edit/profile
