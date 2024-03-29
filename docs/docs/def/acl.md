# acl

A definition representing ACLs for a specified Kafka resource.

## Definition

- **apiVersion**: v1
- **kind**: acl
- **metadata** ([Metadata](#metadata))
- **spec** ([Spec](#spec))

## Metadata

- **name** (string), required

    The name of the resource that ACL entries will be applied to.
    For type `cluster` this must be `kafka-cluster`.

- **type** (string), required

    The type of the resource that ACL entries will be applied to.
    Must be one of `topic`, `group`, `cluster`, `transactional_id`, `delegation_token`.

- **resourcePatternType** (string)

    How the resource name will be understood by Kafka.
    Must be one of `literal`, `prefixed`.
    The default value is `literal`.

- **labels** (map[string]string)

    Labels are key-value pairs associated with the definition.

    Labels are not directly used by kdef and have no remote state.
    They are purely for the purposes of storing meaningful attributes with the definition that would be relevant to users.

## Spec

- **acls** ([][ACLEntryGroup](#aclentrygroup))
- **deleteUndefinedAcls** (bool)

    Allows kdef to delete ACLs that are not defined in `acls`. It is highly recommended to set this to `true`. If `false`, changes to ACL entry groups will only create new ACLs and previously defined ACLs will remain attached to the target resource.

    !!! caution
        Enabling allows kdef to permanently delete ACLs. Always confirm operations with `--dry-run`.

## ACLEntryGroup

A group of ACL entries, where specifying more than one value for its properties results in many ACLs being created in a combinatorial fashion.

!!! example
    The following ACL entry group creates six ACLs.
    ```yaml
        - hosts: ["*"]
        operations: ["READ", "WRITE"]
        permissionType: ALLOW
        principals:
            - User:foo
            - User:bar
            - User:baz
    ```
    ```
    "*", "READ", "ALLOW", "User:foo"
    "*", "READ", "ALLOW", "User:bar"
    "*", "READ", "ALLOW", "User:baz"
    "*", "WRITE", "ALLOW", "User:foo"
    "*", "WRITE", "ALLOW", "User:bar"
    "*", "WRITE", "ALLOW", "User:baz"
    ```

- **hosts** ([]string), required

    Host addresses to create ACLs for. The wildcard "*" allows all hosts.

- **operations** ([]string), required

    Operations to create ACLs for. Must be one of `ALL`, `READ`, `WRITE`, `CREATE`, `DELETE`, `ALTER`, `DESCRIBE`, `CLUSTER_ACTION`, `DESCRIBE_CONFIGS`,`ALTER_CONFIGS`,`IDEMPOTENT_WRITE`.

- **permissionType** (string), required

    The permission type for ACLs in this group. Must be either `ALLOW` or `DENY`.

- **principals** ([]string), required

    Principals to create ACLs for. When using Kafka simple authorizer, this must begin with `User:`.

## Examples

```yaml
--8<-- "docs/examples/definitions/acl/cluster/kafka-cluster.yml"
```

```yaml
--8<-- "docs/examples/definitions/acl/topic/store.events.order-created.yml"
```

## Schema

**Definition:**
```js
{
    "apiVersion": string,
    "kind": string,
    "metadata": {
        "name": string,
        "type": string,
        "labels": [
            string
        ]
    },
    "spec": {
        "acls": [
            {
                "hosts": [
                    string
                ],
                "operations": [
                    string
                ],
                "permissionType": string,
                "principals": [
                    string
                ]
            }
        ],
        "deleteUndefinedAcls": bool
    }
}
```
