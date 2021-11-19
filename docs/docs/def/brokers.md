# brokers

A definition representing cluster-wide configuration for all Kafka brokers.

## Definition

- **apiVersion**: v1
- **kind**: brokers
- **metadata** ([Metadata](#metadata))
- **spec** ([Spec](#spec))

## Metadata

- **name** (string), required

    An identifier for the definition.
    Can be any value for reference.

## Spec

- **configs** (map[string]string)

    A map of key value config pairs.

- **deleteUndefinedConfigs** (bool)

    Allows kdef to delete configs that are not defined in `configs`.

    !!! caution
        Enabling allows kdef to permanently delete configs. Always confirm operations with `--dry-run`.

## Examples

```yml
--8<-- "examples/definitions/brokers/store-cluster.yml"
```
