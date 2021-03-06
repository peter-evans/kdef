# Installation

Install the pre-compiled binary, use Docker, or compile from source.

## Install the pre-compiled binary

Install the pre-compiled binary via one of the following methods.

### homebrew tap

```sh
brew install peter-evans/kdef/kdef
```

### go install

```sh
go install github.com/peter-evans/kdef@latest
```

Install a specific version by using a suffix in the format `@x.x.x`.

### manually

Download the pre-compiled binaries from the [releases page](https://github.com/peter-evans/kdef/releases) and copy them to the desired location.

!!! info
    If you would like to see the kdef binary released via a method not listed please make an [issue](https://github.com/peter-evans/kdef/issues) to discuss its feasibility.

## Running with Docker

kdef can also be executed within a Docker container.

Registries:

- [`peterevans/kdef`](https://hub.docker.com/r/peterevans/kdef)
- [`ghcr.io/peter-evans/kdef`](https://github.com/peter-evans/kdef/pkgs/container/kdef)

Example usage:

```sh
docker run --rm \
    -v $PWD:/var/opt/kdef/my-cluster \
    peterevans/kdef \
    apply "/var/opt/kdef/my-cluster/resources/**/*.yml" \
        --config-path="/var/opt/kdef/my-cluster/config.yml" \
        --dry-run
```

If connecting to a locally running Kafka cluster you may need to use `--net=host` to run the container in the host network.

## Compiling from source

If you would like to build from source follow these steps:

**clone:**

```sh
git clone https://github.com/peter-evans/kdef
cd kdef
```

**get dependencies:**

```sh
go mod tidy
```

**build:**

```sh
go build -o kdef .
```

**verify:**

```sh
./kdef --version
```

## Updating

!!! important
    Until kdef reaches `v1.0.0` and is considered stable, there could be breaking changes in minor releases. Make sure to check the [release](https://github.com/peter-evans/kdef/releases) notes before updating.
