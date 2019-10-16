
> Here you find a [technical view on our secrets](Secrets_technical.md)

# Secrets

There are different types of secrets in the future Steward scenario (independently of what is defined for the first release) being used in different ways and defined by different personas (Steward Ops, CloudCI Ops, Customer):

- [Docker Pull Secrets](#docker-pull-secrets)
    - [Steward System Pull Secret](#steward-system-pull-secret) (Steward)
    - [Jenkinsfile Runner Image Pull Secret](#jenkinsfile-runner-image) (CloudCi & Customer)
    - [Pipeline Step Image Pull Secret](#pipeline-steps--k8s-plugin) (CloudCi & Customer)
- [Git Secrets](#git-secrets)
    - [Jenkinsfile Fetch Git Secret](#jenkinsfile-fetch-git-secret) (CloudCi & Customer)
    - [Source Project Git Secret](#source-project-git-secret) (Customer)
- [General Jenkins Credential Secrets](#general-jenkins-credential-secrets) (Customer)


## Docker Pull Secrets

Docker pull secrets are required by Kubernetes to pull Docker images from private registries.

### Steward System Pull Secret

> **Note:** Does not apply for GCP. If the registry and the K8s cluster are hosted in the same Google account (region?) then no pull secret is required.

For any Steward internal Pods we need a pull secret to fetch the images. Examples are the `run controller` image or the `tenant controller` images.

**Personas:** `Steward Operator`


### Jenkinsfile Runner Image

> **Note:** Does not apply for GCP. If the registry and the K8s cluster are hosted in the same Google account (region?) then no pull secret is required.

A pull secret needs to be specified to pull the (centrally defined) Jenkinsfile Runner Docker image.

> **Note:** If we decide that different Jenkinsfile Runner images can be used we might need additional custom pull secrets for the Jenkinsfile Runner image.

**Personas:** `CloudCi`. (`Customer` if we would allow custom images)


### Pipeline Steps / K8s plugin

Inside pipelines the K8s plugin can be used to execute parts of the pipeline in Docker images, spawned on the K8s cluster.

Also for those images K8s requires pull secrets. Since our customers will be able to define arbitrary custom images they have to be able to provide the secrets.

> **Decision:** Customers provide pull secrets as [General Jenkins Secrets](#general-jenkins-secrets) (see below) and specify them in the pipeline / K8s plugin. The plugin will take care to store the credentials as pull secrets in the run namespace. Therefore no need for us to create pull secrets upfront.
> Disadvantage: **All** pipelines have to specify which credential(s) to use. To enable using images without specifying a credential in the future we could decide to attach the pull secrets to the service account which is executing the pipeline run. In this case the secrets are used for the image pull automatically and don't need to be defined in the pipeline.

**Personas:** `CloudCi` & `Customer`

## Git Secrets

### Jenkinsfile Fetch Secret

We need credentials to fetch the Jenkinsfile, before we execute it.
The fetch is done inside our Jenkinsfile Runner image in the start script. The Git credentials are [injected by Tekton](https://github.com/tektoncd/pipeline/blob/master/docs/auth.md)) and the script can simply do a `git clone` without prior login.

As first step we only plan to support our predefined Jenkinsfile. But in future teams will be able to use custom pipelines, either from the project sources or another custom git repo.

- **Centrally defined Jenkinsfile**: The credentials have to be provided by us.
- **Custom Jenkinsfile**: Credentials have to be provided by customer
    - **In Sources**: Same as [Project Source Fetch Secrets]() (see below)
    - **Separate Repo**: Separate Credentials

**Personas:** `CloudCi` & `Customer`


### Source Project Secret

Once the pipeline is running it needs to sync the project sources. This secret might also require write permissions to the repository, e.g. to create a tag, send status feedback, push an updated version commit.

The required credentials can be provided by customers as [General Jenkins Secrets]() (see below).

**Personas:** `Customer`


## General Jenkins Secrets

Customers can define secrets which will be made available to the Jenkins pipeline as regular `Jenkins Credentials` via the `K8s Credential Provider` plugin. Those credentials can be used inside the pipeline, e.g. to deploy to Cloud Foundry, but also to fetch sources from a specific Git repository, etc.

**Personas:** `CloudCi` & `Customer`

### Jenkins Credential Types

- Username with password
- Secret file
- Secret text
- SSH Username with private key
- Certificate

More examples:

- Docker Host Certificate Authentication
- Kubernetes Service Account
- OpenShift OAuth token
- OpenShift Username and Password
