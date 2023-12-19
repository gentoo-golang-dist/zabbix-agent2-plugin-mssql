---
id: 58cebec7-67f5-43cc-b7f9-5a699cd55b03
template:
  content: |
    # {{ .Name }}

    <rat graph />

    ---

    ## Implemented in

    - []()

    ## Jenkins jobs:

    - branch
      - [backend_agent2_job]()
      - [integration_job]()
---

# Readme

<rat graph />

---

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

## Connection configuration

To gather monitoring data the plugin needs to establish a connection to a MSSQL
server. A connection can be configured in two ways. Read more about each
connection configuration option in the following sections.

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

Read more about what parameters are available for each metric key in the
section - metric keys.

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

## Configuration options

### Plugin settings

Global setting for the MSSQL plugin. Applied to all connections.

#### `Plugins.MSSQL.System.Path`

Path to the MSSQL plugin executable.

Example usage:

```conf
Plugins.MSSQL.System.Path=/usr/sbin/zabbix-agent2-plugin/zabbix-agent2-plugin-mssql
```

#### `Plugins.MSSQL.Timeout`

Specifies the amount of time to wait for a server to respond when first
connecting and on follow-up operations in the session. Range: 1-30 in seconds.

Example usage:

```conf
Plugins.MSSQL.Timeout=10
```

#### `Plugins.MSSQL.KeepAlive`

Specifies the time in seconds for waiting before unused connections will be
closed. Range: 60-900 in seconds.

Example usage:

```conf
Plugins.MSSQL.KeepAlive=600
```

#### `Plugins.MSSQL.CustomQueriesDir`

Specifies the file path to a directory containing user-defined `.sql` files with
custom queries that the plugin can execute.

Example usage:

```conf
Plugins.MSSQL.CustomQueriesDir=/path/to/custom/queries/dir
```

### Session settings

For following session config options, the `*` symbol in the field name implies a
session name. Replace `*` with the actual (like `production` or `stage`) session
name.

#### `Plugins.MSSQL.Sessions.*.Uri`

Specifies the URI to connect, for session `*`. The only supported schema is
`sqlserver`. Embedded credentials will be ignored.

Default: `sqlserver://localhost:1433`

Example usage:

```conf
Plugins.MSSQL.Sessions.exampleSession.Uri=sqlserver://localhost:1433
```

#### `Plugins.MSSQL.Sessions.*.User`

Specifies the username to be sent to a protected MSSQL server for the session
`*`.

Example usage:

```conf
Plugins.MSSQL.Sessions.exampleSession.User=myusername
```

#### `Plugins.MSSQL.Sessions.*.Password`

Specifies the password to be sent to a protected MSSQL server for session `*`.

Example usage:

```conf
Plugins.MSSQL.Sessions.exampleSession.Password=mypassword
```

#### `Plugins.MSSQL.Sessions.*.CACertPath`

Specifies file path to the public key certificate of the certificate authority
(CA) that issued the certificate of the MSSQL server for the session `*`. The
certificate must be in PEM format.

Example usage:

```conf
Plugins.MSSQL.Sessions.exampleSession.CACertPath=/path/to/certificate.crt
```

#### `Plugins.MSSQL.Sessions.*.TrustServerCertificate`

Specifies whether the plugin should trust the server certificate without
validating it for the session `*`. Possible values: `true`, `false`.

Example usage:

```conf
Plugins.MSSQL.Sessions.exampleSession.TrustServerCertificate=true
```

#### `Plugins.MSSQL.Sessions.*.HostNameInCertificate`

Specifies the common name (CN) of the certificate of the MSSQL server for the
session `*`.

Example usage:

```conf
Plugins.MSSQL.Sessions.exampleSession.HostNameInCertificate=myserver.domain.com
```

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

#### `Plugins.MSSQL.Sessions.*.TLSMinVersion`

Specifies the minimum TLS version to use for session `*`. Possible values are:
`1.0`, `1.1`, `1.2`, `1.3`.

Example usage:

```conf
Plugins.MSSQL.Sessions.exampleSession.TLSMinVersion=1.2
```

### Default settings

`Plugins.MSSQL.Default.*` fields define the default values, that will be used if
no other value is specified. (The `*` symbol implies a specific config field
like `Uri` or `Password`)

#### `Plugins.MSSQL.Default.Uri`

Specifies the default URI to connect. The only supported schema is `sqlserver`.
Embedded credentials will be ignored.

Default: `sqlserver://localhost:1433`

Example usage:

```conf
Plugins.MSSQL.Default.Uri=sqlserver://myserver.domain.com:1433
```

#### `Plugins.MSSQL.Default.User`

Specifies the default username to be sent to a protected MSSQL server.

Example usage:

```conf
Plugins.MSSQL.Default.User=defaultuser
```

#### `Plugins.MSSQL.Default.Password`

Specifies the default password to be sent to a protected MSSQL server.

Example usage:

```conf
Plugins.MSSQL.Default.Password=defaultpassword
```

#### `Plugins.MSSQL.Default.CACertPath`

This configuration option specifies the default file path to the public key
certificate of the certificate authority (CA) that issued the certificate of the
MSSQL server. The certificate must be in PEM format.

Example usage:

```conf
Plugins.MSSQL.Default.CACertPath=/path/to/default-certificate.pem
```

#### `Plugins.MSSQL.Default.TrustServerCertificate`

Specifies the default behavior of whether the plugin should trust the server
certificate without validating it. Possible values: `true`, `false`.

Example usage:

```conf
Plugins.MSSQL.Default.TrustServerCertificate=false
```

#### `Plugins.MSSQL.Default.HostNameInCertificate`

Specifies the default common name (CN) of the certificate of the MSSQL server.

Example usage:

```conf
Plugins.MSSQL.Default.HostNameInCertificate=defaultserver.domain.com
```

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

#### `Plugins.MSSQL.Default.TLSMinVersion`

Specifies the default minimum TLS version to use. Possible values are: `1.0`,
`1.1`, `1.2`, `1.3`.

Example usage:

```conf
Plugins.MSSQL.Default.TLSMinVersion=1.1
```

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

### `mssql.availability.group.get[<commonParameters>]`

Returns the availability groups.

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

### `mssql.db.get`

Returns the available databases.

### `mssql.job.status.get`

Returns the status of jobs.

### `mssql.last.backup.get`

Returns the last backup time for all databases.

### `mssql.local.db.get`

Returns local DB info.

### `mssql.mirroring.get`

Returns mirroring info.

### `mssql.nonlocal.db.get`

Returns non-local DB info.

### `mssql.perfcounter.get`

Returns the performance counters.

### `mssql.ping`

Ping the database. Test if connection is correctly configured.

### `mssql.quorum.get`

Returns the quorum info.

### `mssql.quorum.member.get`

Returns the quorum members.

### `mssql.replica.get`

Returns the replicas.

### `mssql.version`

Returns the MSSQL server version.

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

## Contributing

Noticed a bug or have an idea for improvement? Feel free to open an issue or a
feature request in
[Zabbix support system](https://support.zabbix.com/secure/Dashboard.jspa)

Want to contribute? Pull requests are welcome!
