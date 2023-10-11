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
	_ "embed"
	"encoding/json"

	"git.zabbix.com/ap/mssql/plugin/dbconn"
	"git.zabbix.com/ap/mssql/plugin/handlers"
	"git.zabbix.com/ap/mssql/plugin/params"
	"git.zabbix.com/ap/plugin-support/metric"
	"git.zabbix.com/ap/plugin-support/plugin"
	"git.zabbix.com/ap/plugin-support/plugin/container"
	"git.zabbix.com/ap/plugin-support/zbxerr"
)

const (
	// Name of the plugin.
	Name = "MSSQL"

	availabilityGroupGet = mssqlMetricKey("mssql.availability.group.get")
	customQuery          = mssqlMetricKey("mssql.custom.query")
	dbGet                = mssqlMetricKey("mssql.db.get")
	jobStatusGet         = mssqlMetricKey("mssql.job.status.get")
	lastBackupGet        = mssqlMetricKey("mssql.last.backup.get")
	localDBGet           = mssqlMetricKey("mssql.local.db.get")
	mirroringGet         = mssqlMetricKey("mssql.mirroring.get")
	nonLocalDBGet        = mssqlMetricKey("mssql.nonlocal.db.get")
	perfCounterGet       = mssqlMetricKey("mssql.perfcounter.get")
	ping                 = mssqlMetricKey("mssql.ping")
	quorumGet            = mssqlMetricKey("mssql.quorum.get")
	quorumMemberGet      = mssqlMetricKey("mssql.quorum.member.get")
	replicaGet           = mssqlMetricKey("mssql.replica.get")
	version              = mssqlMetricKey("mssql.version")
)

var (
	//go:embed queries/availability.group.get.sql
	availabilityGroupGetQuery string
	//go:embed queries/db.get.sql
	dbGetQuery string
	//go:embed queries/job.status.get.sql
	jobStatusGetQuery string
	//go:embed queries/last.backup.get.sql
	lastBackupGetQuery string
	//go:embed queries/local.db.get.sql
	localDBGetQuery string
	//go:embed queries/mirroring.get.sql
	mirroringGetQuery string
	//go:embed queries/nonlocal.db.get.sql
	nonLocalDBGetQuery string
	//go:embed queries/perfcounter.get.sql
	perfCounterGetQuery string
	//go:embed queries/quorum.get.sql
	quorumGetQuery string
	//go:embed queries/quorum.member.get.sql
	quorumMemberGetQuery string
	//go:embed queries/replica.get.sql
	replicaGetQuery string
)

var (
	_ plugin.Configurator = (*mssqlPlugin)(nil)
	_ plugin.Exporter     = (*mssqlPlugin)(nil)
	_ plugin.Runner       = (*mssqlPlugin)(nil)
)

type mssqlMetricKey string

type mssqlMetric struct {
	metric  *metric.Metric
	handler handlers.HandlerFunc
}

type mssqlPlugin struct {
	plugin.Base
	conns         *dbconn.ConnCollection
	config        *pluginConfig
	metrics       map[mssqlMetricKey]*mssqlMetric
	customQueries handlers.CustomQueries
}

// Launch launches the MSSQL plugin. Blocks until plugin execution has
// finished.
func Launch() error {
	// because of suboptimal setup flow that plugin-support lib
	// we are forced to allocate custom queries and conns first
	// (without initialising them) to allow registering metrics before receiving
	// config or starting plugin. only then in mssqlPlugin.Start these fields
	// can be properly initialised. may bby joda be with u when trying to
	// folllow this after a month.
	p := &mssqlPlugin{
		customQueries: make(handlers.CustomQueries),
		conns:         &dbconn.ConnCollection{},
	}

	p.metrics = map[mssqlMetricKey]*mssqlMetric{
		availabilityGroupGet: {
			metric: metric.New(
				"Returns the availability groups.",
				[]*metric.Param{params.URI, params.User, params.Password},
				false,
			),
			handler: p.conns.WithConnHandlerFunc(
				handlers.QueryHandlerFunc(availabilityGroupGetQuery),
			),
		},
		customQuery: {
			metric: metric.New(
				"Returns the result rows of a custom query.",
				[]*metric.Param{
					params.URI,
					params.User,
					params.Password,
					params.QueryName,
				},
				true,
			),
			handler: p.conns.WithConnHandlerFunc(
				p.customQueries.HandlerFunc,
			),
		},
		dbGet: {
			metric: metric.New(
				"Returns the availabile databases.",
				[]*metric.Param{params.URI, params.User, params.Password},
				false,
			),
			handler: p.conns.WithConnHandlerFunc(
				handlers.QueryHandlerFunc(dbGetQuery),
			),
		},
		jobStatusGet: {
			metric: metric.New(
				"Return the status of jobs.",
				[]*metric.Param{params.URI, params.User, params.Password},
				false,
			),
			handler: p.conns.WithConnHandlerFunc(
				handlers.QueryHandlerFunc(jobStatusGetQuery),
			),
		},
		lastBackupGet: {
			metric: metric.New(
				"Return the last backup time.",
				[]*metric.Param{params.URI, params.User, params.Password},
				false,
			),
			handler: p.conns.WithConnHandlerFunc(
				handlers.QueryHandlerFunc(lastBackupGetQuery),
			),
		},
		localDBGet: {
			metric: metric.New(
				"Return local DB info.",
				[]*metric.Param{params.URI, params.User, params.Password},
				false,
			),
			handler: p.conns.WithConnHandlerFunc(
				handlers.QueryHandlerFunc(localDBGetQuery),
			),
		},
		mirroringGet: {
			metric: metric.New(
				"Return mirroring info.",
				[]*metric.Param{params.URI, params.User, params.Password},
				false,
			),
			handler: p.conns.WithConnHandlerFunc(
				handlers.QueryHandlerFunc(mirroringGetQuery),
			),
		},
		nonLocalDBGet: {
			metric: metric.New(
				"Return non-local DB info.",
				[]*metric.Param{params.URI, params.User, params.Password},
				false,
			),
			handler: p.conns.WithConnHandlerFunc(
				handlers.QueryHandlerFunc(nonLocalDBGetQuery),
			),
		},
		perfCounterGet: {
			metric: metric.New(
				"Return the performance counters.",
				[]*metric.Param{params.URI, params.User, params.Password},
				false,
			),
			handler: p.conns.WithConnHandlerFunc(
				handlers.QueryHandlerFunc(perfCounterGetQuery),
			),
		},
		ping: {
			metric: metric.New(
				"Ping the database.",
				[]*metric.Param{params.URI, params.User, params.Password},
				false,
			),
			handler: p.conns.PingHandler,
		},
		quorumGet: {
			metric: metric.New(
				"Return the quorum info.",
				[]*metric.Param{params.URI, params.User, params.Password},
				false,
			),
			handler: p.conns.WithConnHandlerFunc(
				handlers.QueryHandlerFunc(quorumGetQuery),
			),
		},
		quorumMemberGet: {
			metric: metric.New(
				"Return the quorum members.",
				[]*metric.Param{params.URI, params.User, params.Password},
				false,
			),
			handler: p.conns.WithConnHandlerFunc(
				handlers.QueryHandlerFunc(quorumMemberGetQuery),
			),
		},
		replicaGet: {
			metric: metric.New(
				"Return the replicas.",
				[]*metric.Param{params.URI, params.User, params.Password},
				false,
			),
			handler: p.conns.WithConnHandlerFunc(
				handlers.QueryHandlerFunc(replicaGetQuery),
			),
		},
		version: {
			metric: metric.New(
				"Return the version.",
				[]*metric.Param{params.URI, params.User, params.Password},
				false,
			),
			handler: p.conns.WithConnHandlerFunc(handlers.VersionHandler),
		},
	}

	err := p.registerMetrics()
	if err != nil {
		return err
	}

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

// Start starts the mssql plugin, setting up the internal connection management.
// Initialised in Start, to ensure that config has been loaded before.
// (Start is called after Configure).
func (p *mssqlPlugin) Start() {
	p.conns.Init(p.config.KeepAlive, p)

	err := p.customQueries.Load(p.config.CustomQueriesDir, p)
	if err != nil {
		// continue without custom queries.
		p.Critf("failed to load custom queries: %s", err.Error())
	}
}

func (p *mssqlPlugin) Stop() {
	p.conns.Close()
}

// Export collects all the metrics.
func (p *mssqlPlugin) Export(
	key string, rawParams []string, _ plugin.ContextProvider,
) (any, error) {
	m, ok := p.metrics[mssqlMetricKey(key)]
	if !ok {
		return nil, zbxerr.Wrapf(
			zbxerr.ErrorUnsupportedMetric, "unknown metric %q", key,
		)
	}

	metricParams, extraParams, hardcodedParams, err := m.metric.EvalParams(
		rawParams, p.config.Sessions,
	)
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to evaluate metric parameters")
	}

	err = metric.SetDefaults(metricParams, hardcodedParams, p.config.Default)
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to set default params")
	}

	res, err := m.handler(metricParams, extraParams...)
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to execute handler")
	}

	jsonRes, err := json.Marshal(res)
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to marshal result to JSON")
	}

	return string(jsonRes), nil
}

func (p *mssqlPlugin) registerMetrics() error {
	metricSet := metric.MetricSet{}

	for k, m := range p.metrics {
		metricSet[string(k)] = m.metric
	}

	err := plugin.RegisterMetrics(p, Name, metricSet.List()...)
	if err != nil {
		return zbxerr.Wrap(err, "failed to register metrics")
	}

	return nil
}
