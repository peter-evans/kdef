# Configuration

## Sources

kdef sources configuration in the following ways.

### Config file

kdef will attempt to source configuration from a file named `config.yml` in the current working directory.
This default behaviour can be overridden by specifying the path to a file with the `--config-path` global option.

The easiest way to create a configuration file for your cluster is to run through the short interactive prompt.

```sh
kdef configure
```

### Environment variables

kdef will attempt to source configuration from `KDEF__` prefixed environment variables.
Environment variables override config file configuration.

```sh
KDEF__SEED_BROKERS="b-1.example.amazonaws.com:9098,b-2.example.amazonaws.com:9098" \
KDEF__ALTER_CONFIGS_METHOD="incremental" \
KDEF__TLS__ENABLED="true" \
KDEF__SASL__METHOD="aws_msk_iam" \
kdef export topic
```

### Command-line options

kdef will attempt to source configuration from the `-X` command-line option.
Command-line supplied configuration overrides both config file configuration and environment variables.

```sh
kdef export topic \
  -X seedBrokers=b-1.example.amazonaws.com:9098,b-2.example.amazonaws.com:9098 \
  -X alterConfigsMethod=incremental \
  -X tls.enabled=true \
  -X sasl.method=aws_msk_iam
```

## Config

- **seedBrokers** ([]string)

    One or more seed broker addresses that kdef will connect to.
    The default value is `localhost:9092`.

- **tls** ([TLSConfig](#tlsconfig))

- **sasl** ([SASLConfig](#saslconfig))

- **timeoutMs** (int)

    Timeout in milliseconds to be used by requests with timeouts.
    The default value is `5000`.

- **alterConfigsMethod** (string)

    The method to use when altering configs.
    Must be one of `auto`, `incremental`, `non-incremental`.
    The default value is `auto`.

    Kafka 2.3.0+ supports "incremental alter configs." This is an improved API for altering configs.
    When set to `auto`, kdef will detect what the cluster supports and use `incremental` if available.
    Setting `incremental` or `non-incremental` will save an API call to determine what the cluster supports.

    Note that if the cluster contains brokers with a mix of Kafka versions, some Kafka 2.3.0+ and some Kafka <2.3.0, then `non-incremental` should be used.

## TLSConfig

- **enabled** (bool)

    Set to `true` if connection to cluster brokers requires TLS.
    The default value is `false`.

- **caCertPath** (string)

    Path to a CA cert.

- **clientCertPath** (string)

    Path to a client cert.

- **clientKeyPath** (string)

    Path to a client key.

- **serverName** (string)

    Set if connecting via TLS requires a distinct server name.
    When connecting via TLS, by default the client will use the hostname or IP address of the connected broker as the TLS server name.

## SASLConfig

- **method** (string)

    The required SASL method.
    Must be one of `plain`, `scram-sha-256`, `scram-sha-512`, `aws-msk-iam`.

- **user** (string)

    SASL username.

- **pass** (string)

    SASL password.

- **isToken** (bool)

    Set to `true` if the SASL is from a delegation token.

## Examples

### Amazon MSK

The following configuration can be used to access an Amazon MSK cluster with [IAM Access Control](https://docs.aws.amazon.com/msk/latest/developerguide/iam-access-control.html) enabled.

```yml
--8<-- "examples/config/sasl_aws_msk_iam/config.yml"
```

When executing kdef, the AWS SDK must be instructed as to where credentials can be sourced. See [here](https://docs.aws.amazon.com/sdk-for-go/api/aws/session/) for further documentation.

```sh
AWS_SDK_LOAD_CONFIG=1 AWS_PROFILE=my-profile kdef export topic
```

### SASL/PLAIN

```yml
--8<-- "examples/config/sasl_plain/config.yml"
```
