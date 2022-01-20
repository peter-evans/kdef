# topic

Export topics to definitions (Kafka 0.11.0+).

## Synopsis

```sh
kdef export topic [options]
```

Exports to stdout by default. Supply the `--output-dir` option to create definition files.

## Examples

Export all topics to the directory "topics".
```sh
kdef export topic --output-dir "topics"
```

Export all topics to stdout.
```sh
kdef export topic --quiet
```

Export all topics starting with "myapp"
```sh
kdef export topic --match "myapp.*"
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

    Regular expression matching topic names to include.
    The default value is `.*`.

- **--exclude / -e** (string)

    Regular expression matching topic names to exclude.
    The default value is `.^`.

- **--include-internal / -i** (bool)

    Include internal topics.
    The default value is `false`.

- **--assignments / -a** (string)

    Partition assignments to include in topic definitions.
    Must be one of `none`, `broker`, `rack`.
    The default value is `none`.

## Global options

--8<-- "cmd/global-options.md"
