/*
** Zabbix
** Copyright 2001-2023 Zabbix SIA
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
	"git.zabbix.com/ap/plugin-support/metric"
	"git.zabbix.com/ap/plugin-support/uri"
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

	URI = metric.NewConnParam(
		"URI", "URL connection string to connect to the database.",
	).
		WithDefault("sqlserver://localhost:1433").
		WithSession().
		WithValidator(
			uri.URIValidator{
				Defaults: &uri.Defaults{
					Scheme: "sqlserver",
					Port:   "1433",
				},
				AllowedSchemes: []string{"sqlserver"},
			},
		)
	User = metric.NewConnParam(
		"User", "MSSQL database user.",
	)
	Password = metric.NewConnParam(
		"Password", "MSSQL database users password.",
	)

	QueryName = metric.NewParam(
		"QueryName",
		"Name of a custom query "+
			"(must be equal to a name of an SQL file without an extension).",
	).SetRequired()

	CACertPath = metric.NewSessionOnlyParam(
		"CACertPath",
		"File path of the public key certificate of the CA "+
			"that signed the SQL server certificate.",
	)
	TrustServerCertificate = metric.NewSessionOnlyParam(
		"TrustServerCertificate",
		"Trust the server certificate without verification.",
	)
	HostNameInCertificate = metric.NewSessionOnlyParam(
		"HostNameInCertificate",
		"Common name (CN) in the server certificate.",
	)
	Encrypt = metric.NewSessionOnlyParam(
		"Encrypt",
		"Whether to encrypt connection to the server.",
	).WithValidator(
		metric.SetValidator{
			Set: []string{"", "strict", "disable", "true", "false"},
		},
	)
	TLSMinVersion = metric.NewSessionOnlyParam(
		"TLSMinVersion",
		"Minimum TLS version to use.",
	).WithValidator(
		metric.SetValidator{Set: []string{"", "1.0", "1.1", "1.2", "1.3"}},
	)
)

// Join combines multiple parameter groups into one.
func Join(params ...[]*metric.Param) []*metric.Param {
	var res []*metric.Param

	for _, p := range params {
		res = append(res, p...)
	}

	return res
}
