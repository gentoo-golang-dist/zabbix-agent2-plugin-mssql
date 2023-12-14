# MSSQL plugin for Zabbix agent 2

This plugin provides a native Zabbix solution to monitor Microsoft SQL servers.

It can monitor several MSSQL instances simultaneously, remote or local.

## Requirements

- Zabbix Agent 2 version 6.0.0 or newer
- Go programming language version 1.20 or newer (required only to build the
  plugin from source)

## Supported MSSQL versions

## Setup

Set `Plugins.MSSQL.System.Path` setting in Zabbix agent 2 configuration file
with the path to the MSSQL plugin executable.

We recommend creating a `mssql.conf` and placing all plugin related
configurations there. Then import the plugin configuration file in Zabbix agent
2 configuration file - `zabbix_agent2.conf`.

Add the following setting to the MSSQL plugin configuration file `mssql.conf`:

```conf
Plugins.MSSQL.System.Path=/path/to/executable/mssql
```

To import the plugin configuration file in Zabbix agent 2 add the following line
to Zabbix agent 2 configuration file - `zabbix_agent2.conf`

```conf
Include=/path/to/config/mssql.conf
```

This is the bare minimum required to get the plugin running. More information
about available configuration settings is available in the section -
Configuration options

## Command line options

The MSSQL plugin is not intended to be used as a command line utility, however
it does provide the following command line options.

- `-h`, `--help` display a help message
- `-V`, `--version` prints program version and license information

## Microsoft SQL server requirements

https://www.sqlserverversions.com/

2017

## Configuration options
