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

    If `assignments` or `rackAssignments` are not specified, kdef will create evenly distributed replica assignments for new partitions.
    Partition leaders (the first replica in the assignment) will be assigned based on the broker frequency of partition leaders in the topic, breaking ties with round-robin broker ID.
    Non-leader replicas will be assigned based on broker frequency in the topic, breaking ties with round-robin broker ID.

- **replicationFactor** (int), required

    Replication factor for the topic. Cannot exceed the number of available brokers.

    If `assignments` or `rackAssignments` are not specified, kdef will create evenly distributed replica assignments.
    When increasing the replication factor, replicas will be assigned based on broker frequency in the topic, breaking ties with round-robin broker ID.
    When decreasing the replication factor, replicas are removed based on broker frequency in the topic, breaking ties with the highest replica index. i.e. `[1, 2, 3] -> [1, 2]`.

- **assignments** ([][]int)

    Partition replica assignments by broker ID.
    The number of replica assignments must match `partitions`, and the number of replicas in each assignment must match `replicationFactor`.
    A replica assignment for a partition cannot contain duplicate broker IDs.

    If `assignments` and `rackAssignments` are specified at the same time `assignments` takes precedence.
    `rackAssignments` will be validated but subsequently ignored.

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

    kdef will create evenly distributed replica assignments constrained by the assigned racks.
    Partition leaders (the first replica in the assignment) will be assigned based on the broker frequency of partition leaders in the topic, breaking ties with round-robin broker ID.
    Non-leader replicas will be assigned based on broker frequency in the topic, breaking ties with round-robin broker ID.

    If `assignments` and `rackAssignments` are specified at the same time `assignments` takes precedence.
    `rackAssignments` will be validated but subsequently ignored.

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
