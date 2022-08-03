# Nephio

The Nephio project is building a Kubernetes-based automation platform for
deploying and managing highly distributed, interconnected workloads such as 5G
Network Functions, and the underlying infrastructure on which those workloads
depend.

## Community

Please see the following resources for more information:
  * Website: [nephio.org](https://nephio.org)
  * Wiki: [wiki.nephio.org](https://wiki.nephio.org)
  * Slack: [nephio.slack.com](https://nephio.slack.com)
  * Governance:
    [github.com/nephio-project/governance](https://github.com/nephio-project/governance)

## Installation Overview

Nephio is very early in its development; there is no release yet. However if you
wish to experiment with the project or contribute to it, the following
instructions will help you get a pre-release version up.

To install and run Nephio, you will need:
  * A Kubernetes cluster.
  * The Kubernetes CLI client, [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl).
  * The Kpt CLI client, [kpt](https://kpt.dev/installation/kpt-cli).
  * A Git repository provider. As of now, GitHub and Google Cloud Source
    Repositories are supported.
  * An OAuth 2.0 client ID, if you wish to install the GUI. The GUI only works
    with GKE right now, due to how authentication is done.

### Installation Steps
  1. Install the prerequisite tools on your workstation.
     * [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)
     * [kpt](https://kpt.dev/installation/kpt-cli)
  1. Bring Your Own Kubernetes Cluster or [Create a GKE Cluster](#creating-a-gke-cluster)
     * If you bring your own cluster, make sure your `kubectl` context is
       pointing at that cluster as you run the `kpt` and `kubectl` commands that
       follow.
  1. [Install the Nephio Server Components](#installing-the-server-components)
  1. [Install the Nephio Web UI](#installing-the-web-ui) (Optional)
  1. [Create Repositories](#creating-repositories)
  1. [Register Repositories](#registering-repositories)

After that, Nephio will be ready for use.

## Creating a GKE Cluster

These instructions are for GKE Autopilot. You can use any Kubernetes cluster,
though. If you are using a different cluster you can skip to the next section.

To use GKE, you will need a Google Cloud account and project, and you will need
to [install gcloud](https://cloud.google.com/sdk/docs/install) on your
workstation.

Once `gcloud` is installed and your GCP project is created, you need to point
`gcloud` at that project:

```
gcloud config set project YOUR_GCP_PROJECT
```

Next, enable the GKE service on the project:
```
gcloud services enable container.googleapis.com
```

Finally, create the cluster, and then configure `kubectl` to point to the
cluster (you can use a different region, if you prefer):

```
# Create the cluster
gcloud container clusters create-auto --region us-central1 nephio
# This will take a few minutes
# Once it returns, configure kubectl with the credentials for the cluster
gcloud container clusters get-credentials --region us-central1 nephio
```

## Installing the Server Components

The Nephio software runs within the Kubernetes cluster. First, let's create a
working directory for our package files:

```
mkdir nephio-install
cd nephio-install
```

Next fetch the package using `kpt`, and run any `kpt` functions:

```
kpt pkg get --for-deployment https://github.com/nephio-project/nephio-packages.git/nephio-system
kpt fn render nephio-system
```

Now, we apply the package. If we are using GKE Autopilot, we need to give
some extra time for the deployment, as it may need to spin up new nodes which
takes a while. Thus, we add the `--reconcile-timeout=15m` flag.

```
kpt live init nephio-system
kpt live apply nephio-system --reconcile-timeout=15m --output=table
```

## Installing the Web UI

Currently, we can just run the prototype Config-as-Data UI from the [kpt](https://github.com/GoogleContainerTools/kpt)
project. In time we will build our own UI. This prototype GUI only works with
GKE, because the Web UI uses the OAuth user's identity when talking to the
cluster.

### Creating an OAuth 2.0 Client ID

The prototype web interface needs a way to authenticate users. Currently, it
uses Google OAuth 2.0, which requires a client ID and allows you to authenticate
against the GCP identity service. If you wish to try out the prototype UI, you
will need to create a client ID. To create your client ID and secret:

1. Sign in to the [Google Console](https://console.cloud.google.com)
2. Select or create a new project from the dropdown menu on the top bar
3. Navigate to
   [APIs & Services > Credentials](https://console.cloud.google.com/apis/credentials)
4. Click **Create Credentials** and choose `OAuth client ID`
5. Configure an OAuth consent screen, if required
   - For scopes, select `openid`, `auth/userinfo.email`,
     `auth/userinfo.profile`, and `auth/cloud-platform`.
   - Add any users that will want access to the UI if using External user type
6. Set **Application Type** to `Web Application` with these settings:
   - `Name`: Nephio Web UI
   - `Authorized JavaScript origins`: http://localhost:7007
   - `Authorized redirect URIs`:
     http://localhost:7007/api/auth/google/handler/frame
7. Click Create
8. Copy the client ID and client secret displayed

### Installing the Web UI Package

The prototype UI is a separate package, so let's install that now. First, we
need to pre-provision the namespace and the secret with the OAuth 2.0 client ID
and client secret.

```
kubectl create ns nephio-webui
kubectl create secret generic -n nephio-webui cad-google-oauth-client --from-literal=client-id=CLIENT_ID_PLACEHOLDER --from-literal=client-secret=CLIENT_SECRET_PLACEHOLDER
```

Next, we fetch the package, and then execute `kpt fn render` to execute the
`kpt` function pipeline and prepare the package for deployment.

```
kpt pkg get --for-deployment https://github.com/nephio-project/nephio-packages.git/nephio-webui
kpt fn render nephio-webui
```

Then we apply it:

```
kpt live init nephio-webui
kpt live apply nephio-webui --reconcile-timeout=15m --output=table
```

### Accessing the Web UI

For this prototyping, we are not exposing the Web UI via a load balancer
service. This means that the Web UI is only available on an in-cluster IP
address. Thus, we need to port forward via `kubectl` to access the Web UI from
our workstation browser.

```
kubectl port-forward --namespace=nephio-webui svc/nephio-webui 7007
```

You can now access the Web UI on your workstation by visiting
[http://localhost:7007/config-as-data](http://localhost:7007/config-as-data) in
your browser.

You will be given a choice of OAuth 2.0 providers - Google will be the only
option at this time. Clicking on that will allow you to login using your Google
account, which will then be used as the identity that the Web UI uses to
interact with the Kubernetes server.

## Creating Repositories

Nephio can work with repositories in GitHub or in the Google Cloud Source
Repository service. This example will use GitHub.

There are two types of repositories in Nephio: "blueprint" repositories and
"deployment" repositories. The difference is in the validations performed, and
the intended consumption model of the packages (blueprints) in each type of
repository.

  * Blueprint repositories contain packages that could not be (or at least are
    not intended to be) directly instantiated on a Kuberentes cluster. These
    packages require additional customization in order to become actual, running
    workloads on a cluster.
  * Deployment repositories contain packages that are fully prepared for
    consumption by the API server; also known as "fully hydrated". These are the
    repositories that will be watched by the GitOps deployment tool (e.g.,
    ConfigSync) running in the workload cluster.

The prototype UI adds one additional distinction between "Catalog Blueprint"
clusters and "Blueprint" clusters, with the former being intended for public or
vendor upstream packages, and the latter for local, private organizational
versions of those upstream packages and other organization-local packages.

In Nephio, workload clusters are typically associated with deployment
repositories in a one-to-one fashion. It's not strictly necessary but is the
expected, most common usage model.

To create a GitHub repository, see [the GitHub
Help](https://docs.github.com/en/get-started/quickstart/create-a-repo). Nephio
supports public or private repositories in GitHub. Nephio will need a `main`
branch, so go ahead and have GitHub create the `README.md` for you, which will
create that branch.

You will need to create a [Personal Access Token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)
with `repo` scope to use Nephio. You may want to consider creating a separate
"Nephio Test" user account for this purpose. Use of more selectively scoped
authentication such as per-repo [Deploy Keys](https://docs.github.com/en/developers/overview/managing-deploy-keys#deploy-keys)
is something that we can work on in the future.

## Registering Repositories

Registering repositories can be done via the Web UI or using `kpt`. To register
a GitHub repository `nephio-test-catalog-01` in your personal GitHub account:

```
GITHUB_USERNAME=<your github username>
GITHUB_TOKEN=<GitHub Personal Access Token>

kpt alpha repo register \
  --namespace default \
  --repo-basic-username=${GITHUB_USERNAME} \
  --repo-basic-password=${GITHUB_TOKEN} \
  https://github.com/${GITHUB_USERNAME}/nephio-test-catalog-01.git
```

When registering a *deployment* repository, the same command is used, except it
must include the `--deployment` flag:

```
GITHUB_USERNAME=<your github username>
GITHUB_TOKEN=<GitHub Personal Access Token>

kpt alpha repo register \
  --deployment \
  --namespace default \
  --repo-basic-username=${GITHUB_USERNAME} \
  --repo-basic-password=${GITHUB_TOKEN} \
  https://github.com/${GITHUB_USERNAME}/nephio-test-deploy-01.git
```

It is also possible to set a different branch and directory for packages within
the repository; see `kpt alpha repo register --help` for more.
