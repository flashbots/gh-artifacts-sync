# gh-artifacts-sync

Subscribes to Github workflow events, downloads matching artifacts and
synchronise them to the configured destinations.

Supported destinations:

- [GCP Generic Artifact Registry](https://cloud.google.com/artifact-registry/docs/generic)

## Configuring & running

```shell
./gh-artifacts-sync --config /path/to/config.yaml serve \
  --dir-artifacts /temp/dir/with/sufficient/disk/space/for/temporary/downloads \
  --dir-jobs /persistent/dir/to/store/unfinished/synchronisation/jobs
```

```yaml
# config.yaml

github:
  webhook_secret: sss  # configured during gh app installation
                       # see also: --github-webhook-secret
                       #           --github-webhook-secret-path
  app:
    id: nnn               # assigned on gh app creation
    installation_id: mmm  # assigned after gh app installation

    # see also: --github-private-key, --github-private-key-path
    private_key: |
      -----BEGIN RSA PRIVATE KEY-----
      xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
      ...
      -----END RSA PRIVATE KEY-----

log:
  level: info  # or debug, warn, error, etc (see golang zap)
  mode: dev    # or prod for json format

repositories:
  org/repo:
    #
    # releases section configures synchronisation from published releases
    #
    releases:
      (v\d+\.\d+\.\d+):  # match + capture version
        accept_drafts:      false
        accept_prereleases: false

        assets:
          super-cool-app-aarch64-unknown-linux-gnu.zip:  # match only
            destinations:
              - type: gcp.artifactregistry.generic
                path: projects/${GCP_PROJECT}$/locations/${GCP_REGION}/repositories/generic
                package: ${ORGANISATION}.super-cool-app.aarch64

          super-cool-app-x86_64-unknown-linux-gnu.zip:
            destinations:
              - type: gcp.artifactregistry.generic
                path: projects/${GCP_PROJECT}$/locations/${GCP_REGION}/repositories/generic
                package: ${ORGANISATION}.super-cool-app.x86_64

    #
    # containers section configures synchronisation from github packages
    # (with ecosystem type `CONTAINER`).  multiplatform images are supported.
    # attested images are supported as well.
    #
    containers:
      super-cool-app:
        destinations:
          - type: gcp.artifactregistry.docker
            path: ${GCP_REGION}-docker.pkg.dev/${GCP_PROJECT}/${REPO}
            package: super-cool-app
            platforms: [ linux/amd64, linux/arm64 ]  # only sync these platforms

    #
    # workflows section configures synchronisation from the artifacts uploaded
    # by github workflows (those available on workflow run summary page)
    #
    workflows:
      release.yaml:
        artifacts:
          actors: [ user1, user2 ]  # only consider runs triggered by these users
          super-cool-app-(\w+)-aarch64-unknown-linux-gnu:  # match + capture version
            destinations:
              - type: gcp.artifactregistry.generic
                path: projects/${GCP_PROJECT}$/locations/${GCP_REGION}/repositories/generic
                package: ${ORGANISATION}.super-cool-app.aarch64

          super-cool-app-(\w+)-x86_64-unknown-linux-gnu:
            destinations:
              - type: gcp.artifactregistry.generic
                path: projects/${GCP_PROJECT}$/locations/${GCP_REGION}/repositories/generic
                package: ${ORGANISATION}.super-cool-app.x86_64

## CLI parameters

```haskell
NAME:
   gh-artifacts-sync serve - run gh-artifacts-sync server

USAGE:
   gh-artifacts-sync serve [command options]

GLOBAL OPTIONS:
   --config path                         path to the configuration file
   --log-level value, --log.level value  logging level (default: "info") [$GH_ARTIFACTS_SYNC_LOG_LEVEL]
   --log-mode value, --log.mode value    logging mode (default: "prod") [$GH_ARTIFACTS_SYNC_LOG_MODE]
   --version, -v                         print the version

OPTIONS:
   DIR

   --dir-downloads path, --dir.downloads path                          a path to the directory where downloaded artifacts will be temporarily stored (default: "./downloads") [$GH_ARTIFACTS_SYNC_DIR_DOWNLOADS]
   --dir-jobs path, --dir.jobs path                                    a path to the directory where scheduled jobs will be persisted (default: "./jobs") [$GH_ARTIFACTS_SYNC_DIR_JOBS]
   --dir-soft-delete-downloads path, --dir.soft_delete_downloads path  a path to the directory where finalised downloaded will be moved to instead of deleting [$GH_ARTIFACTS_SYNC_DIR_SOFT_DELETE_DOWNLOADS]
   --dir-soft-delete-jobs path, --dir.soft_delete_jobs path            a path to the directory where complete jobs be moved instead of deleting [$GH_ARTIFACTS_SYNC_DIR_SOFT_DELETE_JOBS]

   GITHUB

   --github-app-id id, --github.app.id id                        github app id (default: 0) [$GH_ARTIFACTS_SYNC_GITHUB_APP_ID]
   --github-installation-id id, --github.app.installation_id id  installation id of the github app (default: 0) [$GH_ARTIFACTS_SYNC_GITHUB_INSTALLATION_ID]
   --github-private-key key, --github.app.private_key key        private key of the github app [$GH_ARTIFACTS_SYNC_GITHUB_PRIVATE_KEY]
   --github-private-key-path path                                path to a .pem file with private `key` of the github app [$GH_ARTIFACTS_SYNC_GITHUB_PRIVATE_KEY_PATH]
   --github-webhook-secret token, --github.webhook_secret token  secret token for the github webhook [$GH_ARTIFACTS_SYNC_GITHUB_WEBHOOK_SECRET]
   --github-webhook-secret-path path                             path to a file with secret token for the github webhook [$GH_ARTIFACTS_SYNC_GITHUB_WEBHOOK_SECRET_PATH]

   SERVER

   --server-listen-address host:port, --server.listen_address host:port  host:port for the server to listen on (default: "0.0.0.0:8080") [$GH_ARTIFACTS_SYNC_SERVER_LISTEN_ADDRESS]
```
