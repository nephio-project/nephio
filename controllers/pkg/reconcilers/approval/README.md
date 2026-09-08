# Approval controller

The approval controller automatically takes a package from Draft to Proposed
and then to Published, according to a policy that is added as an annotation
to a PackageRevision.

Two built-in policies are supported.

`initial` publishes a Draft if and only if:
- The package readiness gates are all True.
- There is not already a Published revision for the package.

This allows us to use it for initial approvals, but also allows us to then
create a new Draft and have it not be automatically published.

`always` publishes every Draft once the readiness gates are met, whether or not
a Published revision already exists.

To choose a policy, annotate the package revision with
`approval.nephio.org/policy: initial` or `approval.nephio.org/policy: always`.
Any other value is rejected with an event and no action is taken.

No delay is applied unless the package revision is annotated with
`approval.nephio.org/delay`. Set it to a Go duration string to require the
revision to have existed for that long before it is proposed and approved. For
example, `approval.nephio.org/delay: 5m` requires five minutes.

The delay is measured from the package revision's creation timestamp, so a
revision that has already waited part of it is not made to wait again. Any
non-negative duration is accepted, including `0s`, which is the same as not
setting the annotation. A negative duration is rejected.
