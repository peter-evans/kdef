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

    This is not used directly by kdef and can be any meaningful value for reference.

- **labels** (map[string]string)

    Labels are key-value pairs associated with the definition.

    Labels are not directly used by kdef and have no remote state.
    They are purely for the purposes of storing meaningful attributes with the definition that would be relevant to users.

## Spec

- **configs** (map[string]string)

    A map of key-value config pairs.

- **deleteUndefinedConfigs** (bool)

    Allows kdef to delete configs that are not defined in `configs`.

    !!! caution
        Enabling allows kdef to permanently delete configs. Always confirm operations with `--dry-run`.

## Examples

```yaml
--8<-- "docs/examples/definitions/brokers/store-cluster.yml"
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
