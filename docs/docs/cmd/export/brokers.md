# brokers

Export cluster-wide broker configuration to definitions (Kafka 0.11.0+).

## Synopsis

```sh
kdef export brokers [options]
```

Exports to stdout by default. Supply the `--output-dir` option to create definition files.

## Examples

Export brokers definition to the directory "brokers".
```sh
kdef export brokers --output-dir "brokers"
```

Export brokers definition to stdout.
```sh
kdef export brokers --quiet
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

## Global options

--8<-- "cmd/global-options.md"
