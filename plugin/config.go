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

package plugin

import (
	"git.zabbix.com/ap/plugin-support/conf"
	"git.zabbix.com/ap/plugin-support/plugin"
	"git.zabbix.com/ap/plugin-support/zbxerr"
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
}

type pluginConfig struct {
	plugin.SystemOptions `conf:"optional,name=System"`
	// Timeout is the amount of time to wait for a server to respond when
	// first connecting and on follow up operations in the session.
	Timeout int `conf:"optional,range=1:30"`
	// KeepAlive is a time to wait before unused connections will be closed.
	KeepAlive int `conf:"optional,range=0:900,default=60"`
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
		return zbxerr.Wrap(err, "failed to unmarshal configuration options")
	}

	return nil
}
