[![Sensu Bonsai Asset](https://img.shields.io/badge/Bonsai-Download%20Me-brightgreen.svg?colorB=89C967&logo=sensu)](https://bonsai.sensu.io/assets/sensu/sensu-dashboard-metrics)
![Go Test](https://github.com/sensu/sensu-dashboard-metrics/workflows/Go%20Test/badge.svg)
![goreleaser](https://github.com/sensu/sensu-dashboard-metrics/workflows/goreleaser/badge.svg)

# Check Plugin Template

## Overview
check-plugin-template is a template repository which wraps the [Sensu Plugin SDK][2].
To use this project as a template, click the "Use this template" button from the main project page.
Once the repository is created from this template, you can use the [Sensu Plugin Tool][9] to
populate the templated fields with the proper values.

## Functionality

After successfully creating a project from this template, update the `Config` struct with any
configuration options for the plugin, map those values as plugin options in the variable `options`,
and customize the `checkArgs` and `executeCheck` functions in [main.go][7].

When writing or updating a plugin's README from this template, review the Sensu Community
[plugin README style guide][3] for content suggestions and guidance. Remove everything
prior to `# sensu-dashboard-metrics` from the generated README file, and add additional context about the
plugin per the style guide.

## Releases with Github Actions

To release a version of your project, simply tag the target sha with a semver release without a `v`
prefix (ex. `1.0.0`). This will trigger the [GitHub action][5] workflow to [build and release][4]
the plugin with goreleaser. Register the asset with [Bonsai][8] to share it with the community!

***

# sensu-dashboard-metrics

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

The sensu-dashboard-metrics is a [Sensu Check][6] that queries the Sensu backend graphql endpoint and provides federation dashboard metrics.

## Usage examples

### Help Output

Help:

```
Usage:
  sensu-dashboard-metrics [flags]
  sensu-dashboard-metrics [command]

Available Commands:
  help        Help about any command
  version     Print the version number of this plugin

Flags:
  -a, --apikey string          Sensu apikey for authentication (use envvar DASHBOARD_APIKEY in production)
  -h, --help                   help for sensu-dashboard-metrics
      --output-format string   metrics output format, supports: opentsdb_line or prometheus_text (default "opentsdb_line")
      --skip-insecure-verify   skip TLS certificate verification (not recommended!)
  -u, --url string             url to access the Sensu Dashboard Web-UI (default "http://localhost:3000/graphql")

```
### Environment variables

|Argument               |Environment Variable       |
|-----------------------|---------------------------|
|--apikey               | DASHBOARD_APIKEY          |
|--output-format        | DASHBOARD_OUTPUT_FORMAT   |
|--url                  | DASHBOARD_URL             |


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
  name: dashboard-apikey
spec:
  provider: env
  id: DASHBOARD_APIKEY
```

## Configuration

### Asset registration

[Sensu Assets][10] are the best way to make use of this plugin. If you're not using an asset, please
consider doing so! If you're using sensuctl 5.13 with Sensu Backend 5.13 or later, you can use the
following command to add the asset:

```
sensuctl asset add sensu/sensu-dashboard-metrics
```

If you're using an earlier version of sensuctl, you can find the asset on the [Bonsai Asset Index][https://bonsai.sensu.io/assets/sensu/sensu-dashboard-metrics].

### Check definition

```yml
---
type: CheckConfig
api_version: core/v2
metadata:
  name: sensu-dashboard-metrics
  namespace: default
spec:
  command: sensu-dashboard-metrics --output-format "prometheus_text"
  subscriptions:
  - sensu-metrics
  runtime_assets:
  - sensu/sensu-dashboard-metrics
  output_metric_format: prometheus_text
  output_metric_handlers:
  - timeseries_database
  secrets:
  - name: DASHBORD_APIKEY
    secret: dashboard-apikey
```

## Installation from source

The preferred way of installing and deploying this plugin is to use it as an Asset. If you would
like to compile and install the plugin from source or contribute to it, download the latest version
or create an executable script from this source.

From the local path of the sensu-dashboard-metrics repository:

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
