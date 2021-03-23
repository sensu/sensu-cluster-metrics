[![Sensu Bonsai Asset](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/sensu/sensu-cluster-metrics)
![Go Test](https://github.com/sensu/sensu-cluster-metrics/workflows/Go%20Test/badge.svg)
![goreleaser](https://github.com/sensu/sensu-cluster-metrics/workflows/goreleaser/badge.svg)


# sensu-cluster-metrics

## Table of Contents
- [Overview](#overview)
- [Usage examples](#usage-examples)
- [Configuration](#configuration)
  - [Asset registration](#asset-registration)
  - [Check definition](#check-definition)
- [Installation from source](#installation-from-source)
- [Additional notes](#additional-notes)
- [Contributing](#contributing)

## Overview

The sensu-cluster-metrics is a [Sensu Check][6] that queries the Sensu backend graphql endpoint and provides federation cluster metrics.

## Usage examples

### Help Output

Help:

```
Usage:
  sensu-cluster-metrics [flags]
  sensu-cluster-metrics [command]

Available Commands:
  help        Help about any command
  version     Print the version number of this plugin

Flags:
  -a, --apikey string          Sensu apikey for authentication (use envvar CLUSTER_APIKEY in production)
  -h, --help                   help for sensu-cluster-metrics
      --output-format string   metrics output format, supports: opentsdb_line or prometheus_text (default "opentsdb_line")
      --skip-insecure-verify   skip TLS certificate verification (not recommended!)
  -u, --url string             url to access the Sensu federated cluster graphql endpoint (default "http://localhost:8080/graphql")

```
### Environment variables

|Argument               |Environment Variable       |
|-----------------------|---------------------------|
|--apikey               | CLUSTER_APIKEY          |
|--output-format        | CLUSTER_OUTPUT_FORMAT   |
|--url                  | CLUSTER_URL             |


**Security Note:** Care should be taken to not expose the apikey for this check by specifying it
on the command line or by directly setting the environment variable in the check definition.  It is
suggested to make use of [secrets management][7] to surface it as an environment variable.  The
check definition below references it as a secret.  Below is an example secrets definition that make
use of the built-in [env secrets provider][8].

```yml
---
type: Secret
api_version: secrets/v1
metadata:
  name: cluster-apikey
spec:
  provider: env
  id: CLUSTER_APIKEY
```

## Configuration

### Asset registration

[Sensu Assets][10] are the best way to make use of this plugin. If you're not using an asset, please
consider doing so! If you're using sensuctl 5.13 with Sensu Backend 5.13 or later, you can use the
following command to add the asset:

```
sensuctl asset add sensu/sensu-cluster-metrics
```

If you're using an earlier version of sensuctl, you can find the asset on the [Bonsai Asset Index][https://bonsai.sensu.io/assets/sensu/sensu-cluster-metrics].

### Check definition

```yml
---
type: CheckConfig
api_version: core/v2
metadata:
  name: sensu-cluster-metrics
  namespace: default
spec:
  command: sensu-cluster-metrics --output-format "prometheus_text"
  subscriptions:
  - sensu-metrics
  runtime_assets:
  - sensu/sensu-cluster-metrics
  output_metric_format: prometheus_text
  output_metric_handlers:
  - timeseries_database
  secrets:
  - name: CLUSTER_APIKEY
    secret: cluster-apikey
```

## Installation from source

The preferred way of installing and deploying this plugin is to use it as an Asset. If you would
like to compile and install the plugin from source or contribute to it, download the latest version
or create an executable script from this source.

From the local path of the sensu-cluster-metrics repository:

```
go build
```

## Additional notes

## Contributing

For more information about contributing to this plugin, see [Contributing][1].

[1]: https://github.com/sensu/sensu-go/blob/master/CONTRIBUTING.md
[2]: https://github.com/sensu-community/sensu-plugin-sdk
[3]: https://github.com/sensu-plugins/community/blob/master/PLUGIN_STYLEGUIDE.md
[4]: https://github.com/sensu-community/check-plugin-template/blob/master/.github/workflows/release.yml
[5]: https://github.com/sensu-community/check-plugin-template/actions
[6]: https://docs.sensu.io/sensu-go/latest/reference/checks/
[7]: https://github.com/sensu-community/check-plugin-template/blob/master/main.go
[8]: https://bonsai.sensu.io/
[9]: https://github.com/sensu-community/sensu-plugin-tool
[10]: https://docs.sensu.io/sensu-go/latest/reference/assets/
