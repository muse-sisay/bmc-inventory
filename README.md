# bmc-inventory

Prints the hardware inventory that [bmclib](https://github.com/bmc-toolbox/bmclib) reads from a BMC, as JSON on stdout.

It is built against a bmclib fork ([muse-sisay/bmclib](https://github.com/muse-sisay/bmclib), branch `feat/redfish-gpu-inventory`) that adds GPU discovery to the Redfish inventory: processors a BMC reports with `ProcessorType: GPU` appear under `gpus`. Upstream bmclib does not collect GPUs.

## Usage

```sh
BMC_PASSWORD='…' ./bmc-inventory -host 10.0.0.5 -user root > inventory.json
jq .gpus inventory.json
```

A summary line goes to stderr:

```
provider=gofish cpus=2 gpus=8 nics=4 drives=2 dimms=16
```

| Flag | Default | Meaning |
|---|---|---|
| `-host` | | BMC address (required) |
| `-user` | | BMC username (required) |
| `-password` | `$BMC_PASSWORD` | BMC password; the environment variable keeps it out of shell history and `ps` |
| `-port` | `443` | Redfish port |
| `-provider` | `gofish` | bmclib provider: `gofish`, `dell`, `supermicro`, `lenovo`, `openbmc`, `asrockrack`, or `all` to try every compatible one (IPMI included) |
| `-basic-auth` | `false` | HTTP basic auth instead of a Redfish session |
| `-timeout` | `3m` | overall timeout |
| `-debug` | `false` | bmclib debug logs on stderr |

`asrockrack` does not use the Redfish inventory code, so it never reports GPUs.

## Read-only

The tool logs in, reads the inventory and logs out. It never changes power state, boot device or settings: with the default provider every request is a GET, apart from creating and deleting the Redfish login session (skipped with `-basic-auth`). Provider compatibility checks read the power state; they do not set it.

## When `gpus` is empty

The fork only finds GPUs listed as processors of the single system bmclib inventories. Some GPU servers list them under a second system, or only as PCIe devices. To see where a BMC puts them:

```sh
curl -sku "root:$BMC_PASSWORD" https://$BMC/redfish/v1/Systems | jq .Members
curl -sku "root:$BMC_PASSWORD" https://$BMC/redfish/v1/Systems/<id>/Processors | jq .Members
curl -sku "root:$BMC_PASSWORD" https://$BMC/redfish/v1/Chassis/<id>/PCIeDevices | jq .Members
```

## Build

```sh
CGO_ENABLED=0 go build -o bmc-inventory .
```

To build against a local bmclib checkout instead of the fork:

```sh
go mod edit -replace github.com/bmc-toolbox/bmclib/v2=../bmclib
```
