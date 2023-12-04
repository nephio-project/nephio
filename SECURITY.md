# Security Policy

We're extremely grateful for security researchers and users that report vulnerabilities to the Nephio Open Source Community. All reports are thoroughly investigated by a set of community volunteers.

The Nephio community has adopted the security disclosures and response policy below to respond to security issues.

Please do not report security vulnerabilities through public GitHub issues.

## Supported Versions

The following versions of Nephio project are currently being supported with security updates.

| Version | Supported          |
| ------- | ------------------ |
| >=1.0   | :white_check_mark: |

## Reporting a Vulnerability

### When should you?
- You think you discovered a potential security vulnerability in Nephio.
- You are unsure how a vulnerability affects Nephio.
- You think you discovered a vulnerability in a dependency of Nephio. For those projects, please leverage their reporting policy.

### When you should not?
- You need assistance in configuring Nephio for security - please discuss this is in the [slack channel](https://nephio.slack.com/archives/C05UXLPF4V6).
- You need help applying security-related updates.
- Your issue is not security-related.

### Please use the process below to report a vulnerability to the project:
1. Email the **Nephio security group at sig-security@lists.nephio.org**

    * Please include the information listed below (as much as you can provide) to help us better understand the nature and scope of the possible issue:
        * Type of issue (e.g. buffer overflow, SQL injection, cross-site scripting, etc.)
        * Full paths of the source file(s) related to the manifestation of the issue
        * Location of the affected source code (tag/branch/commit or direct URL) 
        * Any special configuration required to reproduce the issue
        * Step-by-step instructions to reproduce the issue
        * Proof-of-concept or exploit code (if possible)
        * Impact of the issue, including how an attacker might exploit the issue

    * This information will help us triage your report more quickly.

2. The project security team will send an initial response to the disclosure in 3-5 days. Once the vulnerability and fix are confirmed, the team will plan to release the fix based on the severity and complexity.

3. You may be contacted by a project maintainer to further discuss the reported item. Please bear with us as we seek to understand the breadth and scope of the reported problem, recreate it, and confirm if there is a vulnerability present.

## Security bulletins
For information regarding the security of this project please join our [slack channel](https://nephio.slack.com/archives/C05UXLPF4V6).

## Public Disclosure Timing
A public disclosure date is negotiated by the Nephio Security Response Committee and the bug submitter. We prefer to fully disclose the bug as soon as possible once a user mitigation is available. It is reasonable to delay disclosure when the bug or the fix is not yet fully understood, the solution is not well-tested, or for vendor coordination. The timeframe for disclosure is from immediate (especially if it's already publicly known) to a few weeks. For a vulnerability with a straightforward mitigation, we expect report date to disclosure date to be on the order of 7 days. The Nephio Security Response Committee holds the final say when setting a disclosure date.


