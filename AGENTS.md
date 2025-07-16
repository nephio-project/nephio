This file contains instructions for AI agents working with this repository.

## OpenSSF Scorecard

The OpenSSF Scorecard workflow is located in `.github/workflows/scorecard.yml`. This workflow runs on every push and pull request to the `main` branch.

To run the scorecard analysis locally, you can use the following command:

```bash
docker run -e GITHUB_AUTH_TOKEN -v $(pwd):/src gcr.io/openssf/scorecard:v4.10.2 --repo=github.com/nephio-project/nephio --format=json --show-details
```

This will run the scorecard analysis on the current state of the repository and output the results in JSON format.
