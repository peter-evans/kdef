# acl

Export resource acls to definitions (Kafka 0.11.0+).

## Synopsis

```sh
kdef export acl [options]
```

Exports to stdout by default. Supply the `--output-dir` option to create definition files.

## Examples

Export all resource acls to the directory "acls".
```sh
kdef export acl --output-dir "acls"
```

Export all topic acls to stdout.
```sh
kdef export acl --type topic --quiet
```

Export all resource acls starting with "myapp".
```sh
kdef export acl --match "myapp.*"
```

## Options

- **--format / -f** (string)

    Resource definition format. Must be either `yaml` or `json`.
    The default value is `yaml`.

- **--output-dir / -o** (string)

    Output directory path for definition files.
    Non-existent directories will be created.

- **--overwrite / -w** (bool)

    Overwrite existing files in output directory.
    The default value is `false`.

- **--match / -m** (string)

    Regular expression matching resource names to include.
    The default value is `.*`.

- **--exclude / -e** (string)

    Regular expression matching resource names to exclude.
    The default value is `.^`.

- **--type / -t** (string)

    ACL resource type.
    Must be one of `any`, `topic`, `group`, `cluster`, `transactional_id`, `delegation_token`.
    The default value is `any`.

- **--auto-group / -g** (bool)

    Combine acls into groups for easier management.
    The default value is `true`.

    See [ACLEntryGroup](../../../def/acl/#aclentrygroup) for further details.

## Global options

--8<-- "docs/cmd/global-options.md"
