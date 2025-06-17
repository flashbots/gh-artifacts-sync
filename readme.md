# gh-artifacts-sync

Subscribes to Github workflow events, downloads matching artifacts and
synchronises them to the configured destinations.

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
                       # see also: --github-webhook-secret, --github-webhook-secret-path
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

harvest:
  ${ORGANISATION}/${REPO}:
    ${GH_WORKFLOW_FILENAME}:
      artifacts:
        super-cool-app-(\w+)-aarch64-unknown-linux-gnu:  # (\w+) captures version from artifact name
          destinations:
            - type: gcp.artifactregistry.generic
              path: projects/${GCP_PROJECT}$/locations/${GCP_REGION}/repositories/binary
              package: ${ORGANISATION}.super-cool-app.aarch64

        super-cool-app-(\w+)-x86_64-unknown-linux-gnu:
          destinations:
            - type: gcp.artifactregistry.generic
              path: projects/${GCP_PROJECT}$/locations/${GCP_REGION}/repositories/binary
              package: ${ORGANISATION}.super-cool-app.x86_64
```
