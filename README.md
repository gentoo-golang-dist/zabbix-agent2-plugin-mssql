# MSSQL plugin for Zabbix agent 2

This plugin provides a native Zabbix solution to monitor Microsoft SQL servers.

It can monitor several MSSQL instances simultaneously, remote or local.

<!-- TOC start (generated with https://github.com/derlin/bitdowntoc) -->

## Table of contents

- [Requirements](#requirements)
- [Supported Operating Systems and Architectures](#supported-operating-systems-and-architectures)
- [Setup](#setup)
- [Command line options](#command-line-options)
- [Microsoft SQL server requirements](#microsoft-sql-server-requirements)
- [Connection configuration](#connection-configuration)
  - [In metric key parameters](#in-metric-key-parameters)
- [As a named session](#as-a-named-session)
- [Configuration options](#configuration-options)
  - [Plugin settings](#plugin-settings)
    - [`Plugins.MSSQL.System.Path`](#pluginsmssqlsystempath)
    - [`Plugins.MSSQL.Timeout`](#pluginsmssqltimeout)
    - [`Plugins.MSSQL.KeepAlive`](#pluginsmssqlkeepalive)
    - [`Plugins.MSSQL.CustomQueriesDir`](#pluginsmssqlcustomqueriesdir)
  - [Session settings](#session-settings)
    - [`Plugins.MSSQL.Sessions.*.Uri`](#pluginsmssqlsessionsuri)
    - [`Plugins.MSSQL.Sessions.*.User`](#pluginsmssqlsessionsuser)
    - [`Plugins.MSSQL.Sessions.*.Password`](#pluginsmssqlsessionspassword)
    - [`Plugins.MSSQL.Sessions.*.CACertPath`](#pluginsmssqlsessionscacertpath)
    - [`Plugins.MSSQL.Sessions.*.TrustServerCertificate`](#pluginsmssqlsessionstrustservercertificate)
    - [`Plugins.MSSQL.Sessions.*.HostNameInCertificate`](#pluginsmssqlsessionshostnameincertificate)
    - [`Plugins.MSSQL.Sessions.*.Encrypt`](#pluginsmssqlsessionsencrypt)
    - [`Plugins.MSSQL.Sessions.*.TLSMinVersion`](#pluginsmssqlsessionstlsminversion)
    - [`Plugins.MSSQL.Sessions.*.Database`](#pluginsmssqlsessionsdatabase)
  - [Default settings](#default-settings)
    - [`Plugins.MSSQL.Default.Uri`](#pluginsmssqldefaulturi)
    - [`Plugins.MSSQL.Default.User`](#pluginsmssqldefaultuser)
    - [`Plugins.MSSQL.Default.Password`](#pluginsmssqldefaultpassword)
    - [`Plugins.MSSQL.Default.CACertPath`](#pluginsmssqldefaultcacertpath)
    - [`Plugins.MSSQL.Default.TrustServerCertificate`](#pluginsmssqldefaulttrustservercertificate)
    - [`Plugins.MSSQL.Default.HostNameInCertificate`](#pluginsmssqldefaulthostnameincertificate)
    - [`Plugins.MSSQL.Default.Encrypt`](#pluginsmssqldefaultencrypt)
    - [`Plugins.MSSQL.Default.TLSMinVersion`](#pluginsmssqldefaulttlsminversion)
    - [`Plugins.MSSQL.Default.Database`](#pluginsmssqldefaultdatabase)
- [Metric keys](#metric-keys)
  - [`mssql.availability.group.get[<commonParameters>]`](#mssqlavailabilitygroupgetcommonparameters)
  - [`mssql.custom.query[<commonParameters>,<customQueryName>,<customQueryParameters>...]`](#mssqlcustomquerycommonparameterscustomquerynamecustomqueryparameters)
  - [`mssql.db.get`](#mssqldbget)
  - [`mssql.job.status.get`](#mssqljobstatusget)
  - [`mssql.last.backup.get`](#mssqllastbackupget)
  - [`mssql.local.db.get`](#mssqllocaldbget)
  - [`mssql.mirroring.get`](#mssqlmirroringget)
  - [`mssql.nonlocal.db.get`](#mssqlnonlocaldbget)
  - [`mssql.perfcounter.get`](#mssqlperfcounterget)
  - [`mssql.ping`](#mssqlping)
  - [`mssql.quorum.get`](#mssqlquorumget)
  - [`mssql.quorum.member.get`](#mssqlquorummemberget)
  - [`mssql.replica.get`](#mssqlreplicaget)
  - [`mssql.version`](#mssqlversion)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)

<!-- TOC end -->

<!-- TOC --><a name="requirements"></a>

## Requirements

- Zabbix Agent 2 version 6.0.0 or newer
- Go version 1.23 or newer (required only for building the plugin from the source)

<!-- TOC --><a name="supported-operating-systems-and-architectures"></a>

## Supported Operating Systems and Architectures

The plugin will work on all operating systems and architectures that the Go
programming language and Zabbix agent 2 supports.

<!-- TOC --><a name="setup"></a>

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

<!-- TOC --><a name="command-line-options"></a>

## Command line options

The MSSQL plugin is not intended to be used as a command line utility, however
it does provide the following command line options.

- `-h`, `--help` display a help message
- `-V`, `--version` prints program version and license information

<!-- TOC --><a name="microsoft-sql-server-requirements"></a>

## Microsoft SQL server requirements

The plugin has been tested on the following Microsoft SQL server versions:

- 2022
- 2019
- 2017

The plugin requires a user the following permissions to monitor Microsoft SQL
server:

- for MSSQL version 2022
  ```sql
  CREATE LOGIN zabbix WITH PASSWORD = 'password'
  GRANT VIEW SERVER PERFORMANCE STATE TO zabbix
  USE msdb
  CREATE USER zabbix FOR LOGIN zabbix
  GRANT EXECUTE ON msdb.dbo.agent_datetime TO zabbix
  GRANT SELECT ON msdb.dbo.sysjobactivity TO zabbix
  GRANT SELECT ON msdb.dbo.sysjobservers TO zabbix
  GRANT SELECT ON msdb.dbo.sysjobs TO zabbix
  GO
  ```
- for MSSQL versions 2017 and 2019
  ```sql
  CREATE LOGIN zabbix WITH PASSWORD = 'password'
  GRANT VIEW SERVER STATE TO zabbix
  USE msdb
  CREATE USER zabbix FOR LOGIN zabbix
  GRANT EXECUTE ON msdb.dbo.agent_datetime TO zabbix
  GRANT SELECT ON msdb.dbo.sysjobactivity TO zabbix
  GRANT SELECT ON msdb.dbo.sysjobservers TO zabbix
  GRANT SELECT ON msdb.dbo.sysjobs TO zabbix
  GO
  ```

<!-- TOC --><a name="connection-configuration"></a>

## Connection configuration

To gather monitoring data the plugin needs to establish a connection to a MSSQL
server. A connection can be configured in two ways. Read more about each
connection configuration option in the following sections.

<!-- TOC --><a name="in-metric-key-parameters"></a>

### In metric key parameters

Each metric that the plugin provides has parameters for connection
configuration. All keys have parameters for connection configuration.

```
mssql.ping[sqlserver://localhost:1433,stage_user,stage_password]
```

**Note:** User and password are separate parameters. Don't embed credentials in
URI.

- WRONG: `mssql.ping[sqlserver://stage_user:stage_password@localhost:1433]`
- CORRECT: `mssql.ping[sqlserver://localhost:1433,stage_user,stage_password]`

It is also possible to connect to a named instance by providing the instance name in the path part of the URI.

Example:
```
sqlserver://localhost/InstanceName
```

**Note:** If both the instance name and the port are provided, the port will be used for the connection.

Read more about what parameters are available for each metric key in the
section - metric keys.

<!-- TOC --><a name="as-a-named-session"></a>

## As a named session

Named sessions allow grouping database connection settings under a name. Define
named session configuration parameters the following way:

```conf
Plugins.MSSQL.Sessions.StagingEnv.Uri=sqlserver://192.168.1.1
Plugins.MSSQL.Sessions.StagingEnv.User=stage_user
Plugins.MSSQL.Sessions.StagingEnv.Password=stage_password
```

The example above defines a session named `StagingEnv`. The session then can be
used as the first parameter to a metric key `mssql.version[StagingEnv]` as
opposed to defining each parameter separately
`mssql.version[sqlserver://192.168.1.1,stage_user,stage_password]`.

<!-- TOC --><a name="configuration-options"></a>

## Configuration options

<!-- TOC --><a name="plugin-settings"></a>

### Plugin settings

Global setting for the MSSQL plugin. Applied to all connections.

<!-- TOC --><a name="pluginsmssqlsystempath"></a>

#### `Plugins.MSSQL.System.Path`

Path to the MSSQL plugin executable.

Example usage:

```conf
Plugins.MSSQL.System.Path=/usr/sbin/zabbix-agent2-plugin/zabbix-agent2-plugin-mssql
```

<!-- TOC --><a name="pluginsmssqltimeout"></a>

#### `Plugins.MSSQL.Timeout`

Specifies the amount of time to wait for a server to respond when first
connecting and on follow-up operations in the session. Range: 1-30 in seconds.
If not specified, the value defaults to global timeout value defined in agent 2
configuration.

Example usage:

```conf
Plugins.MSSQL.Timeout=10
```

<!-- TOC --><a name="pluginsmssqlkeepalive"></a>

#### `Plugins.MSSQL.KeepAlive`

Specifies the time in seconds for waiting before unused connections will be
closed. Range: 60-900 in seconds. The default value is 300 (seconds).

Example usage:

```conf
Plugins.MSSQL.KeepAlive=600
```

<!-- TOC --><a name="pluginsmssqlcustomqueriesdir"></a>

#### `Plugins.MSSQL.CustomQueriesDir`

Specifies the file path to a directory containing user-defined `.sql` files with
custom queries that the plugin can execute.

The plugin loads all available `.sql` files in the configured directory at
startup. This means that any changes to the custom query files will not be
reflected until the plugin is restarted. The plugin is started and stopped
together with Zabbix agent 2.

Example usage:

```conf
Plugins.MSSQL.CustomQueriesDir=/path/to/custom/queries/dir
```

<!-- TOC --><a name="session-settings"></a>

### Session settings

For following session config options, the `*` symbol in the field name implies a
session name. Replace `*` with the actual (like `production` or `stage`) session
name.

<!-- TOC --><a name="pluginsmssqlsessionsuri"></a>

#### `Plugins.MSSQL.Sessions.*.Uri`

Specifies the URI to connect, for session `*`. The only supported schema is
`sqlserver`. Embedded credentials will be ignored.

Default: `sqlserver://localhost:1433`

Example usage:

```conf
Plugins.MSSQL.Sessions.exampleSession.Uri=sqlserver://localhost:1433
```

<!-- TOC --><a name="pluginsmssqlsessionsuser"></a>

#### `Plugins.MSSQL.Sessions.*.User`

Specifies the username to be sent to a protected MSSQL server for the session
`*`.

Example usage:

```conf
Plugins.MSSQL.Sessions.exampleSession.User=myusername
```

<!-- TOC --><a name="pluginsmssqlsessionspassword"></a>

#### `Plugins.MSSQL.Sessions.*.Password`

Specifies the password to be sent to a protected MSSQL server for session `*`.

Example usage:

```conf
Plugins.MSSQL.Sessions.exampleSession.Password=mypassword
```

<!-- TOC --><a name="pluginsmssqlsessionscacertpath"></a>

#### `Plugins.MSSQL.Sessions.*.CACertPath`

Specifies file path to the public key certificate of the certificate authority
(CA) that issued the certificate of the MSSQL server for the session `*`. The
certificate must be in PEM format.

Example usage:

```conf
Plugins.MSSQL.Sessions.exampleSession.CACertPath=/path/to/certificate.crt
```

<!-- TOC --><a name="pluginsmssqlsessionstrustservercertificate"></a>

#### `Plugins.MSSQL.Sessions.*.TrustServerCertificate`

Specifies whether the plugin should trust the server certificate without
validating it for the session `*`. Possible values: `true`, `false`.

Example usage:

```conf
Plugins.MSSQL.Sessions.exampleSession.TrustServerCertificate=true
```

<!-- TOC --><a name="pluginsmssqlsessionshostnameincertificate"></a>

#### `Plugins.MSSQL.Sessions.*.HostNameInCertificate`

Specifies the common name (CN) of the certificate of the MSSQL server for the
session `*`.

Example usage:

```conf
Plugins.MSSQL.Sessions.exampleSession.HostNameInCertificate=myserver.domain.com
```

<!-- TOC --><a name="pluginsmssqlsessionsencrypt"></a>

#### `Plugins.MSSQL.Sessions.*.Encrypt`

Specifies the connection encryption type for the session `*`. Possible values
are:

- `true` - Data sent between plugin and server is encrypted.
- `false` - Data sent between plugin and server is not encrypted beyond the
  login packet.
- `strict` - Data sent between plugin and server is encrypted E2E using
  [TDS8](https://learn.microsoft.com/en-us/sql/relational-databases/security/networking/tds-8?view=sql-server-ver16).
- `disable` - Data send between plugin and server is not encrypted.

Example usage:

```conf
Plugins.MSSQL.Sessions.exampleSession.Encrypt=true
```

<!-- TOC --><a name="pluginsmssqlsessionstlsminversion"></a>

#### `Plugins.MSSQL.Sessions.*.TLSMinVersion`

Specifies the minimum TLS version to use for session `*`. Possible values are:
`1.0`, `1.1`, `1.2`, `1.3`.

Example usage:

```conf
Plugins.MSSQL.Sessions.exampleSession.TLSMinVersion=1.2
```

<!-- TOC --><a name="pluginsmssqlsessionsdatabase"></a>

#### `Plugins.MSSQL.Sessions.*.Database`

Specifies the database name to connect to.

Example usage:

```conf
Plugins.MSSQL.Sessions.exampleSession.Database=customers
```

<!-- TOC --><a name="default-settings"></a>

### Default settings

`Plugins.MSSQL.Default.*` fields define the default values, that will be used if
no other value is specified. (The `*` symbol implies a specific config field
like `Uri` or `Password`)

<!-- TOC --><a name="pluginsmssqldefaulturi"></a>

#### `Plugins.MSSQL.Default.Uri`

Specifies the default URI to connect. The only supported schema is `sqlserver`.
Embedded credentials will be ignored.

Default: `sqlserver://localhost:1433`

Example usage:

```conf
Plugins.MSSQL.Default.Uri=sqlserver://myserver.domain.com:1433
```

<!-- TOC --><a name="pluginsmssqldefaultuser"></a>

#### `Plugins.MSSQL.Default.User`

Specifies the default username to be sent to a protected MSSQL server.

Example usage:

```conf
Plugins.MSSQL.Default.User=defaultuser
```

<!-- TOC --><a name="pluginsmssqldefaultpassword"></a>

#### `Plugins.MSSQL.Default.Password`

Specifies the default password to be sent to a protected MSSQL server.

Example usage:

```conf
Plugins.MSSQL.Default.Password=defaultpassword
```

<!-- TOC --><a name="pluginsmssqldefaultcacertpath"></a>

#### `Plugins.MSSQL.Default.CACertPath`

This configuration option specifies the default file path to the public key
certificate of the certificate authority (CA) that issued the certificate of the
MSSQL server. The certificate must be in PEM format.

Example usage:

```conf
Plugins.MSSQL.Default.CACertPath=/path/to/default-certificate.pem
```

<!-- TOC --><a name="pluginsmssqldefaulttrustservercertificate"></a>

#### `Plugins.MSSQL.Default.TrustServerCertificate`

Specifies the default behavior of whether the plugin should trust the server
certificate without validating it. Possible values: `true`, `false`.

Example usage:

```conf
Plugins.MSSQL.Default.TrustServerCertificate=false
```

<!-- TOC --><a name="pluginsmssqldefaulthostnameincertificate"></a>

#### `Plugins.MSSQL.Default.HostNameInCertificate`

Specifies the default common name (CN) of the certificate of the MSSQL server.

Example usage:

```conf
Plugins.MSSQL.Default.HostNameInCertificate=defaultserver.domain.com
```

<!-- TOC --><a name="pluginsmssqldefaultencrypt"></a>

#### `Plugins.MSSQL.Default.Encrypt`

Specifies the default connection encryption type. Possible values are:

- `true` - Data sent between plugin and server is encrypted.
- `false` - Data sent between plugin and server is not encrypted beyond the
  login packet.
- `strict` - Data sent between plugin and server is encrypted E2E using
  [TDS8](https://learn.microsoft.com/en-us/sql/relational-databases/security/networking/tds-8?view=sql-server-ver16).
- `disable` - Data send between plugin and server is not encrypted.

Example usage:

```conf
Plugins.MSSQL.Default.Encrypt=true
```

<!-- TOC --><a name="pluginsmssqldefaulttlsminversion"></a>

#### `Plugins.MSSQL.Default.TLSMinVersion`

Specifies the default minimum TLS version to use. Possible values are: `1.0`,
`1.1`, `1.2`, `1.3`.

Example usage:

```conf
Plugins.MSSQL.Default.TLSMinVersion=1.1
```

<!-- TOC --><a name="pluginsmssqldefaultdatabase"></a>

#### `Plugins.MSSQL.Default.Database`

Specifies the default database name to connect to.

Example usage:

```conf
Plugins.MSSQL.Default.Database=prod
```

<!-- TOC --><a name="metric-keys"></a>

## Metric keys

All metric key responses, except for `mssql.ping` and `mssql.version` are JSON
arrays, with objects as elements.

`mssql.version` response format is `string` (**Note** not a JSON string).

`mssql.ping` response format is `0` or `1`.

Every metric key has the following parameters (further referred to as
`<commonParameters>`):

- URI - MSSQL server URI. The only supported schema is `sqlserver`. Embedded
  credentials will be ignored. Example values:
  - `sqlserver://mssql.domain.com:4321`
  - `mssql.domain.com:4321` default scheme `sqlserver` will be used.
  - `sqlserver://mssql.domain.com` default port `1433` will be used.
  - `mssql.domain.com` default scheme `sqlserver` and port `1433` will be used.
- User - Username to send to protected MSSQL server.
- Password - Password to send to protected MSSQL server.

<!-- TOC --><a name="mssqlavailabilitygroupgetcommonparameters"></a>

### `mssql.availability.group.get[<commonParameters>]`

Returns the availability groups.

<!-- TOC --><a name="mssqlcustomquerycommonparameterscustomquerynamecustomqueryparameters"></a>

### `mssql.custom.query[<commonParameters>,<customQueryName>,<customQueryParameters>...]`

Returns the result rows of a custom query.

`<customQueryName>` - The name of the custom query file configured in
`Plugins.MSSQL.CustomQueriesDir` without the `.sql` extension.

`<customQueryParameters>...` is a variadic parameter allowing to pass parameters
from key invocation to query, if the custom query has been configured to accept
parameters. Custom queries can accept parameters in the following way:

```sql
-- coal_chamber.sql
select * from coal_chamber where
  id > @p1 and
  coal_quality = @p3 and
  air_quality = @p2;
```

The metric key to invoke a custom query with parameters would look like this:

```
mssql.custom.query[exampleSession,,,mssql,13,bad_air_quality,good_coal_quality]
```

**Note**: The three commas are required to pass the empty parameters for user
and password parameters. The `QueryName` is the forth parameter after `URI`
(session name is the first parameter, instead of `URI`), `User`, `Password`,
hence the two empty parameters.

<!-- TOC --><a name="mssqldbget"></a>

### `mssql.db.get`

Returns all available databases.

<!-- TOC --><a name="mssqljobstatusget"></a>

### `mssql.job.status.get`

Returns the status of jobs.

<!-- TOC --><a name="mssqllastbackupget"></a>

### `mssql.last.backup.get`

Returns the last backup time for all databases.

<!-- TOC --><a name="mssqllocaldbget"></a>

### `mssql.local.db.get`

Returns databases that are participating in an Always On availability group and
replica (primary or secondary) and are located on the server that the connection
was established to.

<!-- TOC --><a name="mssqlmirroringget"></a>

### `mssql.mirroring.get`

Returns mirroring info.

<!-- TOC --><a name="mssqlnonlocaldbget"></a>

### `mssql.nonlocal.db.get`

Returns databases that are participating in an Always On availability group and
replica (primary or secondary) located on other servers (The database is not
local to the SQL Server instance that the connection was established to).

<!-- TOC --><a name="mssqlperfcounterget"></a>

### `mssql.perfcounter.get`

Returns the performance counters.

<!-- TOC --><a name="mssqlping"></a>

### `mssql.ping`

Ping the database. Test if connection is correctly configured.

<!-- TOC --><a name="mssqlquorumget"></a>

### `mssql.quorum.get`

Returns the quorum info.

<!-- TOC --><a name="mssqlquorummemberget"></a>

### `mssql.quorum.member.get`

Returns the quorum members.

<!-- TOC --><a name="mssqlreplicaget"></a>

### `mssql.replica.get`

Returns the replicas.

<!-- TOC --><a name="mssqlversion"></a>

### `mssql.version`

Returns the MSSQL server version.

<!-- TOC --><a name="troubleshooting"></a>

## Troubleshooting

The plugin sends all of its logs to Zabbix agent 2, that further logs them where
ever agent 2 log location is configured to.

For debugging Zabbix Agent 2 log level setting can be increased either in config
by field `DebugLevel` or by runtime control by running

```sh
zabbix_agent2 -R log_level_increase
```

For more information about Zabbix agent 2 view
[Zabbix documentation](https://www.zabbix.com/documentation/current/en/manual/concepts/agent2).

<!-- TOC --><a name="contributing"></a>

## Contributing

Noticed a bug or have an idea for improvement? Feel free to open an issue or a
feature request in
[Zabbix support system](https://support.zabbix.com/secure/Dashboard.jspa)

Want to contribute? Pull requests are welcome!
