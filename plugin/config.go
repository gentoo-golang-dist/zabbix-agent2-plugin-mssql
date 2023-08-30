package plugin

import (
	"git.zabbix.com/ap/plugin-support/conf"
	"git.zabbix.com/ap/plugin-support/plugin"
	"git.zabbix.com/ap/plugin-support/zbxerr"
)

var _ plugin.Configurator = (*mssqlPlugin)(nil)

type session struct {
	URI      string `conf:"name=Uri,optional"`
	Password string `conf:"optional"`
	User     string `conf:"optional"`
}

type pluginConfig struct {
	plugin.SystemOptions `conf:"optional,name=System"`
	// Timeout is the amount of time to wait for a server to respond when
	// first connecting and on follow up operations in the session.
	Timeout int `conf:"optional,range=1:30"`
	// KeepAlive is a time to wait before unused connections will be closed.
	KeepAlive int `conf:"optional,range=60:900,default=60"`
	// Sessions stores pre-defined named sets of connections settings.
	Sessions map[string]session `conf:"optional"`
	// Default stores default connection parameter values from configuration
	// file.
	Default session `conf:"optional"`
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
