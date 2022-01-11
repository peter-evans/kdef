# topic

A definition representing a Kafka topic.

## Definition

- **apiVersion**: v1
- **kind**: topic
- **metadata** ([Metadata](#metadata))
- **spec** ([Spec](#spec))
- **state** (State)

    An internal-use only property group that kdef uses to show underlying state changes.

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

- **replicationFactor** (int), required

    Replication factor for the topic. Cannot exceed the number of available brokers.

- **assignments** ([][]int)

    Partition replica assignments by broker ID.
    The number of replica assignments must match `partitions`, and the number of replicas in each assignment must match `replicationFactor`.
    A replica assignment for a partition cannot contain duplicate broker IDs.

    Cannot be specified at the same time as `managedAssignments`.

    !!! example
        Assignments for 3 partitions with a replication factor of 2.
        ```yml
        assignments:
        - [1, 2]
        - [2, 3]
        - [3, 1]
        ```

- **managedAssignments** ([ManagedAssignments](#managedassignments))

    Configuration for kdef-managed partition assignments.
    The definition will default to managed assignments if `assignments` are not specified.

    Cannot be specified at the same time as `assignments`.

- **maintainLeaders** (bool)

    Performs leader election on the preferred leader (the first replica in the assignment) of partitions if leadership has been lost to another broker.

    The default value is `false`.

## ManagedAssignments

When using managed assignments, kdef will make evenly distributed replica assignments based on the configuration in this section.

kdef employs a general strategy of balancing partition replicas across available brokers for topic performance and availability.
In particular, partition leaders (the first replica in the assignment) are evenly distributed across available brokers.

- **rackConstraints** ([][]string)

    Rack ID constraints for partition replica assignment.
    kdef will maintain evenly distributed replica assignments constrained by the specified racks.
    The number of rack constraints must match `partitions`, and the number of replicas in a partition's rack constraints must match `replicationFactor`.

    !!! example
        Rack constraints for 3 partitions with a replication factor of 2.
        ```yml
        rackConstraints:
          - ["zone-a", "zone-b"]
          - ["zone-b", "zone-c"]
          - ["zone-c", "zone-a"]
        ```

        Rack constraints for 3 partitions with a replication factor of 3. This example ensures each partition's leader and follower replicas are all in the same rack.
        ```yml
        rackConstraints:
          - ["zone-a", "zone-a", "zone-a"]
          - ["zone-b", "zone-b", "zone-b"]
          - ["zone-c", "zone-c", "zone-c"]
        ```

- **selection** (string)

    The method used to select a broker for a replica.
    After constraints have been applied, each replica assignment is selected from a pool of qualifying brokers using this method.

    Selection methods:

    - `topic-cluster-use` (default) - Maintain balanced usage of brokers within the topic and cluster. Broker selection for a replica is made based on broker usage within the topic, breaking ties with broker usage across the cluster.
    - `topic-use` - Maintain balanced usage of brokers within the topic. Broker selection for a replica is made based on broker usage within the topic.

    If the above selection methods are unable to narrow the pool to a single broker, ties will broken in two ways.
    When adding replicas, ties will be broken with round-robin broker ID.
    When removing replicas, ties will be broken with the highest replica index.

- **balance** (string)

    The scope of the managed assignments strategy when a topic is applied.

    Balance scopes:

    - `new` (default) - Only new assignments are balanced by the managed strategy. New assignments refers to additional replicas added when partitions are increased, or the replication factor is increased.
    - `all` - All assignments are in scope of the managed strategy, and changes may be made to rebalance replicas across available brokers. Setting this scope is equivalent to performing a partition rebalance on apply.

    !!! tip
        Setting scope `all` permanently in definitions could be disruputive if activity in the cluster causes frequent rebalancing.
        Consider only periodically rebalancing by using the apply command's property override `-P` option to chose when to set the `all` scope.

        i.e. `-P topic.spec.managedAssignments.balance=all`

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
        "deleteUndefinedConfigs": bool,
        "partitions": int,
        "replicationFactor": int,
        "assignments": [
            [
                int
            ]
        ],
        "managedAssignments": {
            "rackConstraints": [
                [
                    string
                ]
            ],
            "selection": string,
            "balance": string
        },
        "maintainLeaders": bool
    },
    "state": {
        "assignments": [
            [
                int
            ]
        ],
        "leaders": [
            int
        ]
    }
}
```

**Additional Data:**

The following additional data is output with the apply result when using the `--json-output` option.
```js
{
    "partitionReassignments": null|[
        {
            "partition": int,
            "replicas": [
                int
            ],
            "addingReplicas": null|[
                int
            ],
            "removingReplicas": null|[
                int
            ]
        }
    ]
}
```
