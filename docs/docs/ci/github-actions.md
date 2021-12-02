# GitHub Actions

kdef can be used in GitHub Actions workflows.
This allows changes to Kafka resource definitions to be reviewed with pull requests and applied on merge (GitOps).

## Example

The following example workflow is a good starting point.
It assumes definition files are located under a directory called `defs`.

Pull requests will execute the apply with `--dry-run`, allowing the diff to be reviewed in the Actions run log.
On merge to the default branch the definitions are applied.

The example demonstrates passing secrets for SASL mechanism authentication.
This is preferable to committing sensitive credentials to the repository in `config.yml`.

```yml
--8<-- "examples/ci/github-actions/kdef.yml"
```
