/*
** Copyright (C) 2001-2026 Zabbix SIA
**
** This program is free software: you can redistribute it and/or modify it under the terms of
** the GNU Affero General Public License as published by the Free Software Foundation, version 3.
**
** This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
** without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
** See the GNU Affero General Public License for more details.
**
** You should have received a copy of the GNU Affero General Public License along with this program.
** If not, see <https://www.gnu.org/licenses/>.
**/

package plugin

import (
	"path/filepath"

	"golang.zabbix.com/sdk/conf"
	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/plugin"
)

type session struct {
	URI                    string `conf:"name=Uri,optional"`
	Password               string `conf:"optional"`
	User                   string `conf:"optional"`
	CACertPath             string `conf:"optional"`
	TrustServerCertificate string `conf:"optional"`
	HostNameInCertificate  string `conf:"optional"`
	Encrypt                string `conf:"optional"`
	TLSMinVersion          string `conf:"optional"`
	Database               string `conf:"optional"`

	// json tag is a temporary workaround until metric.SetDefaults() supports integers
	ConnectionTimeout int `conf:"optional,range=1:30" json:"ConnectionTimeout,string"` //nolint:tagalign,tagliatelle
}

type pluginConfig struct {
	System plugin.SystemOptions `conf:"optional"` //nolint:staticcheck
	// LegacyTimeout timeout used for connections.
	//
	// Deprecated: old timeout value kept for compatibility.
	LegacyTimeout int `conf:"name=Timeout,optional,range=1:30"`
	// KeepAlive is a time to wait before unused connections will be closed.
	KeepAlive int `conf:"optional,range=60:900,default=300"`
	// Sessions stores pre-defined named sets of connection's settings.
	Sessions map[string]session `conf:"optional"`
	// Default stores default connection parameter values from configuration
	// file.
	Default session `conf:"optional"`
	// CustomQueriesDir is an absolute path directory containing user defined
	// *.sql files with custom queries the plugin can execute.
	CustomQueriesDir string `conf:"optional"`
	// CustomQueriesEnabled disabled or enabled custom query functionality.
	CustomQueriesEnabled bool `conf:"optional,default=false"`
}

// Configure implements the Configurator interface.
// Initializes configuration structures.
func (p *MssqlPlugin) Configure(global *plugin.GlobalOptions, options any) {
	pConfig := &pluginConfig{}

	err := conf.UnmarshalStrict(options, pConfig)
	if err != nil {
		p.Errf("cannot unmarshal configuration options: %s", err.Error())

		return
	}

	pConfig.setCustomQueriesDirDefault()

	p.config = pConfig

	if p.config.LegacyTimeout != 0 {
		p.Debugf("config value 'Plugins.MSSQL.Timeout' is deprecated")

		if p.config.Default.ConnectionTimeout == 0 {
			p.config.Default.ConnectionTimeout = p.config.LegacyTimeout
		}
	}

	if p.config.Default.ConnectionTimeout == 0 {
		p.config.Default.ConnectionTimeout = global.Timeout
	}

	if p.config.LegacyTimeout == 0 {
		p.config.LegacyTimeout = global.Timeout
	}
}

// Validate implements the Configurator interface.
// Returns an error if validation of a plugin's configuration is failed.
func (*MssqlPlugin) Validate(options any) error {
	var opts pluginConfig

	err := conf.UnmarshalStrict(options, &opts)
	if err != nil {
		return errs.Wrap(err, "failed to unmarshal configuration options")
	}

	if opts.CustomQueriesEnabled && opts.CustomQueriesDir != "" && !filepath.IsAbs(opts.CustomQueriesDir) {
		return errs.Errorf("opts.CustomQueriesDir path: '%s' must be absolute", opts.CustomQueriesDir)
	}

	return nil
}
