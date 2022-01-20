# GitHub Actions

kdef can be used in GitHub Actions workflows.
This allows changes to Kafka resource definitions to be reviewed with pull requests and applied on merge (GitOps).

## Pull request workflow

The following example workflow is a good starting point.
It assumes definition files are located under a directory called `defs`.

Pull requests will execute the apply with `--dry-run`, allowing the diff to be reviewed in the Actions run log.
On merge to the default branch the definitions are applied.

This example demonstrates passing secrets for SASL mechanism authentication.
This is preferable to committing sensitive credentials to the repository in `config.yml`.

```yaml
--8<-- "docs/examples/ci/github-actions/kdef.yml"
```

## Manual workflow

It may be desirable to manually run workflows in some situations.

In this `workflow_dispatch` example, a rebalance operation is triggered by setting the property override `-P topic.spec.managedAssignments.balance=all`.

```yaml
--8<-- "docs/examples/ci/github-actions/rebalance.yml"
```
