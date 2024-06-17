/*
** Zabbix
** Copyright (C) 2001-2024 Zabbix SIA
**
** Licensed under the Apache License, Version 2.0 (the "License");
** you may not use this file except in compliance with the License.
** You may obtain a copy of the License at
**
**     http://www.apache.org/licenses/LICENSE-2.0
**
** Unless required by applicable law or agreed to in writing, software
** distributed under the License is distributed on an "AS IS" BASIS,
** WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
** See the License for the specific language governing permissions and
** limitations under the License.
**/

package params

import (
	"golang.zabbix.com/sdk/metric"
	"golang.zabbix.com/sdk/uri"
)

//nolint:gochecknoglobals // global constants.
var (
	// BaseParams groups all base parameters common for all connections.
	BaseParams = []*metric.Param{
		URI,
		User,
		Password,
	}

	// CustomQueryParams groups all parameters unique for a custom query metric.
	CustomQueryParams = []*metric.Param{
		QueryName,
	}

	// TLSParams groups all TLS configuration parameters for a connection.
	TLSParams = []*metric.Param{
		CACertPath,
		TrustServerCertificate,
		HostNameInCertificate,
		Encrypt,
		TLSMinVersion,
	}

	// AzureParams groups all item key configuration parameters specific for
	// Azure SQL Database. All item keys except for ping, version and
	// custom query are not meant for Azure SQL Database. (The underlying
	// queries where not designed for Azure) Also Azure does not allow
	// switching databases in a query, hence the only way to gather any data
	// from Azure DB is to specify DB name at connection. This is as an extra
	// bit of functionality to allow users that really want to monitor something
	// on Azure to be able to do that in their very custom own way.
	AzureParams = []*metric.Param{
		Database,
	}

	// URI is a metric param tha specifies database connection URI.
	URI = metric.NewConnParam(
		"URI", "URL connection string to connect to the database.",
	).
		WithDefault("sqlserver://localhost:1433").
		WithSession().
		WithValidator(
			uri.URIValidator{
				Defaults:       URIDefaults,
				AllowedSchemes: []string{"sqlserver"},
			},
		)

		// URIDefaults defines the default values for a DB connection URI.
	URIDefaults = &uri.Defaults{
		Scheme: "sqlserver",
		Port:   "1433",
	}

	// User is a metric param that specifies database user.
	User = metric.NewConnParam(
		"User", "MSSQL database user.",
	)

	// Password is a metric param that specifies database user password.
	Password = metric.NewConnParam(
		"Password", "MSSQL database users password.",
	)

	// QueryName is a metric param that specifies name of a custom query.
	QueryName = metric.NewParam(
		"QueryName",
		"Name of a custom query "+
			"(must be equal to a name of an SQL file without an extension).",
	).SetRequired()

	// CACertPath is a metric param that specifies path to a CA certificate.
	CACertPath = metric.NewSessionOnlyParam(
		"CACertPath",
		"File path of the public key certificate of the CA "+
			"that signed the SQL server certificate.",
	)

	// TrustServerCertificate is a metric param that specifies whether to trust
	// the server certificate without verification.
	TrustServerCertificate = metric.NewSessionOnlyParam(
		"TrustServerCertificate",
		"Trust the server certificate without verification.",
	)

	// HostNameInCertificate is a metric param that specifies common name (CN)
	// in the server certificate.
	HostNameInCertificate = metric.NewSessionOnlyParam(
		"HostNameInCertificate",
		"Common name (CN) in the server certificate.",
	)

	// Encrypt is a metric param that specifies whether to encrypt connection
	// to the server.
	Encrypt = metric.NewSessionOnlyParam(
		"Encrypt",
		"Whether to encrypt connection to the server.",
	).WithValidator(
		metric.SetValidator{
			Set: []string{"", "strict", "disable", "true", "false"},
		},
	)

	// TLSMinVersion is a metric param that specifies minimum TLS version to
	// use.
	TLSMinVersion = metric.NewSessionOnlyParam(
		"TLSMinVersion",
		"Minimum TLS version to use.",
	).WithValidator(
		metric.SetValidator{Set: []string{"", "1.0", "1.1", "1.2", "1.3"}},
	)

	// Database is a metric param tha specifies the database connection should
	// be established to.
	Database = metric.NewSessionOnlyParam(
		"Database", "Database name that the connection will be established to.",
	).
		WithDefault("")
)

// Join combines multiple parameter groups into one.
func Join(params ...[]*metric.Param) []*metric.Param {
	var res []*metric.Param

	for _, p := range params {
		res = append(res, p...)
	}

	return res
}
