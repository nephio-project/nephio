# Approval controller

The approval controller automatically takes a package from Draft to Published,
according to a policy that is added as an annotation to a PackageRevision.

Currently the only supported value is a built-in policy called `initial`, which
will publish a Draft if and only if:
- The package readiness gates are all True.
- There is not already a Published revision.

This allows us to use it for initial approvals, but also allows us to then
create a new Draft and have it not be automatically published.

To enable this policy, annotate the package revision with
`approval.nephio.org/policy: initial`.

This controller also can add a readiness gate with a timeout, to prevent it
from moving to Published state too soon. This will be processed before any
policy is applied. To enable this, annotate the package revision with
`approval.nephio.org/delay-gate` and a value being the number of seconds of
delay. For example `approval.nephio.org/delay-gate: 60` to require the gate to
require the condition to have existed for at least 60 seconds before flipping to
`True`.
