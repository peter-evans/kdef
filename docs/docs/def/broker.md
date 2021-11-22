# broker

A definition representing a single specified Kafka broker.

## Definition

- **apiVersion**: v1
- **kind**: broker
- **metadata** ([Metadata](#metadata))
- **spec** ([Spec](#spec))

## Metadata

- **name** (string), required

    The ID of the target broker.

## Spec

- **configs** (map[string]string)

    A map of key value config pairs.

    Note that Kafka's API does not allow reading `password` type [broker configs](https://kafka.apache.org/documentation/#brokerconfigs).
    Applying these configs is supported, but `kdef apply` will always show a diff for them.

- **deleteUndefinedConfigs** (bool)

    Allows kdef to delete configs that are not defined in `configs`.

    !!! caution
        Enabling allows kdef to permanently delete configs. Always confirm operations with `--dry-run`.

## Examples

```yml
--8<-- "examples/definitions/broker/1.yml"
```
