# Getting started

This short tutorial introduces some of the main features of kdef.

## Tutorial setup

1. Clone the kdef repository and change directory to `docs/tutorial`.

    ```sh
    git clone git@github.com:peter-evans/kdef.git
    cd kdef/docs/tutorial
    ```

2. Execute the following to spin up a two-broker Kafka cluster using Docker compose.

    ```sh
    ZOOKEEPER_PORT=12181 \
    BROKER1_PORT=19092 \
    BROKER2_PORT=19093 \
    docker-compose up -d
    ```

    At the end of the tutorial use the following to bring the cluster down.
    ```sh
    ZOOKEEPER_PORT=12181 \
    BROKER1_PORT=19092 \
    BROKER2_PORT=19093 \
    docker-compose down --volumes
    ```

3. Your current working directory should now be `docs/tutorial`.
    In this directory is a `config.yml` configuration file.
    This is the configuration kdef will use to connect to the Kafka cluster spun up in the previous step.

    ```yml
    --8<-- "tutorial/config.yml"
    ```

## Applying definitions

The cluster we spun up in the previous section has no resources. Let's apply some definitions.

### from files

1. Execute the following to perform a dry-run apply of all definitions under the "definitions" directory.

    ```sh
    kdef apply "definitions/**/*.yml" --dry-run
    ```

2. Remove the `--dry-run` flag and apply the definitions.

    ```sh
    kdef apply "definitions/**/*.yml"
    ```

3. Execute the command in step 2 a second time. You should now see that there are "no changes to apply."

4. Edit `definitions/topic/tutorial_topic1.yml` and make the following changes.

    - `retention.ms: "43200000"`
    - `partitions: 6`

5. Execute steps 1 and 2 again to dry-run and then apply the topic definition update.

### from stdin

1. Execute the following to perform a dry-run apply of a definition passed via stdin.

    ```sh
    cat <<EOF | kdef apply - --dry-run
    apiVersion: v1
    kind: topic
    metadata:
      name: tutorial_topic2
    spec:
      configs:
        retention.ms: "86400000"
      partitions: 3
      replicationFactor: 2
    EOF
    ```

2. Remove the `--dry-run` flag and apply the definition.

    ```sh
    cat <<EOF | kdef apply -
    apiVersion: v1
    kind: topic
    metadata:
      name: tutorial_topic2
    spec:
      configs:
        retention.ms: "86400000"
      partitions: 3
      replicationFactor: 2
    EOF
    ```

3. Execute the command in step 2 a second time. You should now see that there are "no changes to apply."

4. Execute the following to perform a dry-run apply and update the topic created in step 2.

    ```sh
    cat <<EOF | kdef apply - --dry-run
    apiVersion: v1
    kind: topic
    metadata:
      name: tutorial_topic2
    spec:
      configs:
        retention.ms: "43200000"
      partitions: 6
      replicationFactor: 2
    EOF
    ```

5. Remove the `--dry-run` flag and apply the definition update.

    ```sh
    cat <<EOF | kdef apply -
    apiVersion: v1
    kind: topic
    metadata:
      name: tutorial_topic2
    spec:
      configs:
        retention.ms: "43200000"
      partitions: 6
      replicationFactor: 2
    EOF
    ```

## Exporting definitions

Let's export definitions for the resources we created in the previous section.

### to files

1. Execute the following to export `topic` definitions to files in the directory "exported".

    ```sh
    kdef export topic --output-dir exported
    ```

2. Let's additionally export partition assignments for the topics and overwrite the files created in step 1.

    ```sh
    kdef export topic --output-dir exported --overwrite --assignments broker
    ```

### to stdout

1. Execute the following to export `topic` definitions to stdout.
    `--quiet` suppresses log messages so we just see the definitions output.

    ```sh
    kdef export topic --quiet
    ```

2. Alternatively, we can export as JSON.

    ```sh
    kdef export topic --quiet --format json
    ```
