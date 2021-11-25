# topic

A definition representing a Kafka topic.

## Definition

- **apiVersion**: v1
- **kind**: topic
- **metadata** ([Metadata](#metadata))
- **spec** ([Spec](#spec))

## Metadata

- **name** (string), required

    The topic name.

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

- **partitions** (int), required

    Number of partitions for the topic.

    Note that decreasing the number of partitions is not supported.

    If static `assignments` or `rackAssignments` are not specified, kdef will balance new partition replicas across available brokers.

- **replicationFactor** (int), required

    Replication factor for the topic. Cannot exceed the number of available brokers.

    If static `assignments` or `rackAssignments` are not specified, kdef will balance additional replicas across available brokers.
    When decreasing the replication factor, replicas are removed from the "back" of partitions. i.e. `[1, 2, 3] -> [1, 2]`.

- **assignments** ([][]int)

    Partition replica assignments by broker ID.
    The number of replica assignments must match `partitions`, and the number of replicas in each assignment must match `replicationFactor`.
    A replica assignment cannot contain duplicate broker IDs.

    Cannot be specified at the same time as `rackAssignments`.

    !!! example
        Assignments for 3 partitions with a replication factor of 2.
        ```yml
        assignments:
        - [1, 2]
        - [2, 3]
        - [3, 1]
        ```

- **rackAssignments** ([][]string)

    Partition replica assignments by rack ID.
    The number of rack assignments must match `partitions`, and the number of replicas in each rack assignment must match `replicationFactor`.

    kdef will balance replicas across available brokers within the same zone.

    Cannot be specified at the same time as `assignments`.

    !!! example
        Rack assignments for 3 partitions with a replication factor of 2.
        ```yml
        rackAssignments:
        - ["zone-a", "zone-b"]
        - ["zone-b", "zone-c"]
        - ["zone-c", "zone-a"]
        ```

## Examples

```yml
--8<-- "examples/definitions/topic/store.events.order-created.yml"
```

```yml
--8<-- "examples/definitions/topic/store.events.order-updated.yml"
```

```yml
--8<-- "examples/definitions/topic/store.events.order-picked.yml"
```

```yml
--8<-- "examples/definitions/topic/store.events.order-dispatched.yml"
```
