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
	"encoding/json"

	"git.zabbix.com/ap/mssql/plugin/dbconn"
	"git.zabbix.com/ap/plugin-support/metric"
	"git.zabbix.com/ap/plugin-support/plugin"
	"git.zabbix.com/ap/plugin-support/plugin/container"
	"git.zabbix.com/ap/plugin-support/uri"
	"git.zabbix.com/ap/plugin-support/zbxerr"
)

const (
	// Name of the plugin.
	Name = "MSSQL"

	keyJobStatus mssqlMetricKey = "mssql.get_job_status"
)

var (
	paramURI = metric.NewConnParam(
		"URI", "URL connection string to connect to the database.",
	).
		WithDefault("sqlserver://localhost:1433").
		WithSession().
		WithValidator(uri.URIValidator{
			Defaults:       &uri.Defaults{Scheme: "sqlserver", Port: "1433"},
			AllowedSchemes: []string{"sqlserver"},
		})
	paramUser     = metric.NewConnParam("User", "MSSQL database user.")
	paramPassword = metric.NewConnParam(
		"Password", "MSSQL database users password.",
	)
)

type mssqlMetricKey string

type mssqlMetric struct {
	metric  *metric.Metric
	handler func(conn dbconn.Queryer) (any, error)
}

type mssqlPlugin struct {
	plugin.Base
	conns   *dbconn.ConnCollection
	config  *pluginConfig
	metrics map[mssqlMetricKey]*mssqlMetric
}

// Launch launches the MSSQL plugin. Blocks until plugin execution has
// finished.
func Launch() error {
	p := &mssqlPlugin{
		conns: dbconn.NewConnCollection(),
		metrics: map[mssqlMetricKey]*mssqlMetric{
			keyJobStatus: {
				metric: metric.New(
					"Return the status of jobs.",
					[]*metric.Param{paramURI, paramUser, paramPassword},
					false,
				),
				handler: jobStatusHandler,
			},
		},
	}

	p.registerMetrics()

	h, err := container.NewHandler(Name)
	if err != nil {
		return zbxerr.Wrap(err, "failed to create new handler")
	}

	p.Logger = h

	err = h.Execute()
	if err != nil {
		return zbxerr.Wrap(err, "failed to execute plugin handler")
	}

	return nil
}

// Export collects all the metrics.
func (p *mssqlPlugin) Export(
	key string, rawParams []string, _ plugin.ContextProvider,
) (any, error) {
	m, ok := p.metrics[mssqlMetricKey(key)]
	if !ok {
		return nil, zbxerr.ErrorUnsupportedMetric
	}

	params, _, hardcodedParams, err := m.metric.EvalParams(
		rawParams, p.config.Sessions,
	)
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to evaluate metric parameters")
	}

	err = metric.SetDefaults(params, hardcodedParams, p.config.Default)
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to set default params")
	}

	c, err := p.conns.Get(
		dbconn.ConnConfig{
			URI:      params[paramURI.Name()],
			User:     params[paramUser.Name()],
			Password: params[paramPassword.Name()],
		},
	)
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to get conn")
	}

	res, err := m.handler(c)
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to execute handler")
	}

	jsonRes, err := json.Marshal(res)
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to marshal result to JSON")
	}

	return string(jsonRes), nil
}

func (p *mssqlPlugin) registerMetrics() {
	metricSet := metric.MetricSet{}

	for k, m := range p.metrics {
		metricSet[string(k)] = m.metric
	}

	plugin.RegisterMetrics(p, Name, metricSet.List()...)
}
