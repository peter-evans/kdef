# apply

Apply definitions to a Kafka cluster.

## Synopsis

```sh
kdef apply <definitions>... [options]
kdef apply - [options]
```

`<definitions>...` represents one or more glob patterns matching the paths of definitions to apply.
Directories matching patterns are ignored.

`-` instructs kdef to read definitions from stdin.

## Compatibility

kdef uses Kafka broker APIs.
These are the minimum Kafka versions required to apply each definition kind.

- `acl` (Kafka 0.11.0+)
- `broker` (Kafka 0.11.0+)
- `brokers` (Kafka 0.11.0+)
- `topic` (Kafka 2.4.0+)

## Examples

Apply all definitions in directory "topics" (dry-run).
```sh
kdef apply "topics/*.yml" --dry-run
```

Apply definitions in all directories under "resources" (dry-run).
```sh
kdef apply "resources/**/*.yml" --dry-run
```

Apply a topic definition from stdin (dry-run).
```sh
cat topics/my_topic.yml | kdef apply - --dry-run
```

## Options

- **--format / -f** (string)

    Resource definition format. Must be either `yaml` or `json`.
    The default value is `yaml`.

- **--dry-run / -d** (bool)

    Validate and review the operation only.
    The default value is `false`.

- **--exit-code / -e** (bool)

    Implies `--dry-run` and causes the program to exit with 1 if there are unapplied changes and 0 otherwise.
    The default value is `false`.

- **--json-output / -j** (bool)

    Implies `--quiet` and outputs JSON apply results.
    The default value is `false`.

    Schema:
    ```js
    [
        {
            "local": object, // local definition
            "remote": object, // remote definition
            "data": null|object, // additional data
            "diff": string,
            "error": string,
            "applied": bool
        }
    ]
    ```
    For definition and additional data schemas see the documentation for each definition.

- **--continue-on-error / -c** (bool)

    Applying resource definitions is not interrupted if there are errors.
    The default value is `false`.

- **--reass-await-timeout / -r** (int)

    Time in seconds to wait for topic partition reassignments to complete before timing out.
    The default value is `0`.

    Changes to partition assignments will be reflected immediately by Kafka.
    However, partition reassignment operations will be queued internally and may take time to complete.
    While reassignment operations are in progress for a topic, Kafka rejects further partition changes.

    By default kdef does not wait for reassignment operations to complete and exits immediately.
    Optionally, kdef can be instructed with this option to await the completion of partition reassignments.

- **--prop-override / -P** ([]string)

    Definition property override for overridable properties (e.g. `-P topic.spec.managedAssignments.balance=all`).
    This is a repeatable option.

    Overridable properties:

    - `topic.spec.managedAssignments.balance`
    - `topic.spec.maintainLeaders`

## Global options

--8<-- "cmd/global-options.md"
