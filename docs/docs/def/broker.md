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

- **labels** (map[string]string)

    Labels are key-value pairs associated with the definition.

    Labels are not directly used by kdef and have no remote state.
    They are purely for the purposes of storing meaningful attributes with the definition that would be relevant to users.

## Spec

- **configs** (map[string]string)

    A map of key-value config pairs.

    Note that Kafka's API does not allow reading `password` type [broker configs](https://kafka.apache.org/documentation/#brokerconfigs).
    Applying these configs is supported, but `kdef apply` will always show a diff for them.

- **deleteUndefinedConfigs** (bool)

    Allows kdef to delete configs that are not defined in `configs`.

    !!! caution
        Enabling allows kdef to permanently delete configs. Always confirm operations with `--dry-run`.

## Examples

```yaml
--8<-- "docs/examples/definitions/broker/1.yml"
```

## Schema

**Definition:**
```js
{
    "apiVersion": string,
    "kind": string,
    "metadata": {
        "name": string,
        "labels": [
            string
        ]
    },
    "spec": {
        "configs": {
            string: string
        },
        "deleteUndefinedConfigs": bool
    }
}
```
