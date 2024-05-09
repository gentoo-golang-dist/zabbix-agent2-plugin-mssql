/*
** Copyright (C) 2001-2024 Zabbix SIA
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
}

type pluginConfig struct {
	plugin.SystemOptions `conf:"optional,name=System"`
	// Timeout is the amount of time to wait for a server to respond when
	// first connecting and on follow up operations in the session.
	Timeout int `conf:"optional,range=1:30"`
	// KeepAlive is a time to wait before unused connections will be closed.
	KeepAlive int `conf:"optional,range=60:900,default=300"`
	// Sessions stores pre-defined named sets of connections settings.
	Sessions map[string]session `conf:"optional"`
	// Default stores default connection parameter values from configuration
	// file.
	Default session `conf:"optional"`
	// CustomQueriesDir is absolute path directory containing user defined
	// *.sql files with custom queries the plugin can execute.
	CustomQueriesDir string `conf:"optional"`
}

// Configure implements the Configurator interface.
// Initializes configuration structures.
func (p *mssqlPlugin) Configure(global *plugin.GlobalOptions, options any) {
	pConfig := &pluginConfig{}

	err := conf.Unmarshal(options, pConfig)
	if err != nil {
		p.Errf("cannot unmarshal configuration options: %s", err.Error())

		return
	}

	p.config = pConfig

	if p.config.Timeout == 0 {
		p.config.Timeout = global.Timeout
	}
}

// Validate implements the Configurator interface.
// Returns an error if validation of a plugin's configuration is failed.
func (*mssqlPlugin) Validate(options any) error {
	var opts pluginConfig

	err := conf.Unmarshal(options, &opts)
	if err != nil {
		return errs.Wrap(err, "failed to unmarshal configuration options")
	}

	return nil
}
