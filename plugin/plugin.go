/*
** Copyright (C) 2001-2025 Zabbix SIA
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
	_ "embed"
	"os"
	"time"

	"golang.zabbix.com/plugin/mssql/plugin/dbconn"
	"golang.zabbix.com/plugin/mssql/plugin/handlers"
	"golang.zabbix.com/plugin/mssql/plugin/params"
	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/log"
	"golang.zabbix.com/sdk/metric"
	"golang.zabbix.com/sdk/plugin"
	"golang.zabbix.com/sdk/plugin/container"
	"golang.zabbix.com/sdk/zbxerr"
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
	_ plugin.Configurator = (*MssqlPlugin)(nil)
	_ plugin.Exporter     = (*MssqlPlugin)(nil)
	_ plugin.Runner       = (*MssqlPlugin)(nil)
)

// MssqlPlugin hold mssql plugin parameters.
type MssqlPlugin struct {
	plugin.Base
	conns         *dbconn.ConnManager
	config        *pluginConfig
	metrics       map[mssqlMetricKey]*mssqlMetric
	customQueries handlers.CustomQueries
}

type mssqlMetricKey string

type mssqlMetric struct {
	metric  *metric.Metric
	handler handlers.HandlerFunc
}

// New returns a new implementation of mssql plugin.
func New() (*MssqlPlugin, error) {
	// Because of suboptimal setup flow in plugin-support lib,
	// we are forced to allocate custom queries and mgr first
	// (without initializing them) to allow registering metrics before receiving
	// config or starting plugin. Only then in MssqlPlugin.Start these fields
	// can be properly initialized. May baby Yoda be with u when trying to
	// follow this after a month.
	p := &MssqlPlugin{
		customQueries: make(handlers.CustomQueries),
		conns:         &dbconn.ConnManager{},
	}

	err := log.Open(log.Console, log.Info, "", 0)
	if err != nil {
		return nil, errs.Wrap(err, "failed to open log")
	}

	p.Logger = log.New(Name)

	err = p.registerMetrics()
	if err != nil {
		return nil, errs.Wrap(err, "failed to register metrics")
	}

	return p, nil
}

// Run starts the plugin.
func (p *MssqlPlugin) Run() error {
	h, err := container.NewHandler(Name)
	if err != nil {
		return errs.Wrap(err, "failed to create new handler")
	}

	p.Logger = h

	err = h.Execute()
	if err != nil {
		return errs.Wrap(err, "failed to execute plugin handler")
	}

	return nil
}

// Start starts the mssql plugin, setting up the internal connection management.
// Initialized in Start, to ensure that config has been loaded before.
// (Start is called after Configure).
func (p *MssqlPlugin) Start() {
	p.conns.Init(p.config.KeepAlive, p)

	err := p.customQueries.Load(os.DirFS(p.config.CustomQueriesDir), p)
	if err != nil {
		// continue without custom queries.
		p.Critf("failed to load custom queries: %s", err.Error())
	}
}

// Stop stops the mssql plugin, closing all the connections.
func (p *MssqlPlugin) Stop() {
	p.conns.Close()
}

// Export collects all the metrics.
func (p *MssqlPlugin) Export(
	key string, rawParams []string, pluginCtx plugin.ContextProvider,
) (any, error) {
	m, ok := p.metrics[mssqlMetricKey(key)]
	if !ok {
		return nil, errs.Wrapf(
			zbxerr.ErrorUnsupportedMetric, "unknown metric %q", key,
		)
	}

	if mssqlMetricKey(key) == customQuery && !p.config.CustomQueriesEnabled {
		return nil, errs.Errorf("key %q is disabled", key)
	}

	metricParams, extraParams, hardcodedParams, err := m.metric.EvalParams(
		rawParams, p.config.Sessions,
	)
	if err != nil {
		return nil, errs.Wrap(err, "failed to evaluate metric parameters")
	}

	err = metric.SetDefaults(metricParams, hardcodedParams, p.config.Default)
	if err != nil {
		return nil, errs.Wrap(err, "failed to set default params")
	}

	timeout := time.Second * time.Duration(p.config.Timeout)
	if pluginCtx != nil && timeout < time.Second*time.Duration(pluginCtx.Timeout()) {
		timeout = time.Second * time.Duration(pluginCtx.Timeout())
	}

	res, err := m.handler(timeout, metricParams, extraParams...)
	if err != nil {
		return nil, errs.Wrap(err, "failed to execute handler")
	}

	return res, nil
}

func (p *MssqlPlugin) registerMetrics() error {
	p.metrics = map[mssqlMetricKey]*mssqlMetric{
		availabilityGroupGet: {
			metric: metric.New(
				"Returns the availability groups.",
				params.Join(
					params.BaseParams,
					params.TLSParams,
					// AzureParams are added to all item keys (not only the ones
					// that actually need it e.g., ping, version and
					// custom query). This is because EvalParams from plugin-support
					// can't handle session config struct that is superset
					// of params needed by a particular metric. It panics in
					// such a case. 😩🔫
					params.AzureParams,
				),
				false,
			),
			handler: handlers.WithJSONResponse(
				p.conns.WithConnHandlerFunc(
					handlers.QueryHandlerFunc(availabilityGroupGetQuery),
				),
			),
		},
		customQuery: {
			metric: metric.New(
				"Returns the result rows of a custom query.",
				params.Join(
					params.BaseParams,
					params.CustomQueryParams,
					params.TLSParams,
					params.AzureParams,
				),
				true,
			),
			handler: handlers.WithJSONResponse(
				p.conns.WithConnHandlerFunc(
					p.customQueries.HandlerFunc,
				),
			),
		},
		dbGet: {
			metric: metric.New(
				"Returns the available databases.",
				params.Join(
					params.BaseParams,
					params.TLSParams,
					params.AzureParams,
				),
				false,
			),
			handler: handlers.WithJSONResponse(
				p.conns.WithConnHandlerFunc(
					handlers.QueryHandlerFunc(dbGetQuery),
				),
			),
		},
		jobStatusGet: {
			metric: metric.New(
				"Return the status of jobs.",
				params.Join(
					params.BaseParams,
					params.TLSParams,
					params.AzureParams,
				),
				false,
			),
			handler: handlers.WithJSONResponse(
				p.conns.WithConnHandlerFunc(
					handlers.QueryHandlerFunc(jobStatusGetQuery),
				),
			),
		},
		lastBackupGet: {
			metric: metric.New(
				"Return the last backup time for all databases.",
				params.Join(
					params.BaseParams,
					params.TLSParams,
					params.AzureParams,
				),
				false,
			),
			handler: handlers.WithJSONResponse(
				p.conns.WithConnHandlerFunc(
					handlers.QueryHandlerFunc(lastBackupGetQuery),
				),
			),
		},
		localDBGet: {
			metric: metric.New(
				"Return local DB info.",
				params.Join(
					params.BaseParams,
					params.TLSParams,
					params.AzureParams,
				),
				false,
			),
			handler: handlers.WithJSONResponse(
				p.conns.WithConnHandlerFunc(
					handlers.QueryHandlerFunc(localDBGetQuery),
				),
			),
		},
		mirroringGet: {
			metric: metric.New(
				"Return mirroring info.",
				params.Join(
					params.BaseParams,
					params.TLSParams,
					params.AzureParams,
				),
				false,
			),
			handler: handlers.WithJSONResponse(
				p.conns.WithConnHandlerFunc(
					handlers.QueryHandlerFunc(mirroringGetQuery),
				),
			),
		},
		nonLocalDBGet: {
			metric: metric.New(
				"Return non-local DB info.",
				params.Join(
					params.BaseParams,
					params.TLSParams,
					params.AzureParams,
				),
				false,
			),
			handler: handlers.WithJSONResponse(
				p.conns.WithConnHandlerFunc(
					handlers.QueryHandlerFunc(nonLocalDBGetQuery),
				),
			),
		},
		perfCounterGet: {
			metric: metric.New(
				"Return the performance counters.",
				params.Join(
					params.BaseParams,
					params.TLSParams,
					params.AzureParams,
				),
				false,
			),
			handler: handlers.WithJSONResponse(
				p.conns.WithConnHandlerFunc(
					handlers.QueryHandlerFunc(perfCounterGetQuery),
				),
			),
		},
		ping: {
			metric: metric.New(
				"Ping the database.",
				params.Join(
					params.BaseParams,
					params.TLSParams,
					params.AzureParams,
				),
				false,
			),
			handler: p.conns.PingHandler,
		},
		quorumGet: {
			metric: metric.New(
				"Return the quorum info.",
				params.Join(
					params.BaseParams,
					params.TLSParams,
					params.AzureParams,
				),
				false,
			),
			handler: handlers.WithJSONResponse(
				p.conns.WithConnHandlerFunc(
					handlers.QueryHandlerFunc(quorumGetQuery),
				),
			),
		},
		quorumMemberGet: {
			metric: metric.New(
				"Return the quorum members.",
				params.Join(
					params.BaseParams,
					params.TLSParams,
					params.AzureParams,
				),
				false,
			),
			handler: handlers.WithJSONResponse(
				p.conns.WithConnHandlerFunc(
					handlers.QueryHandlerFunc(quorumMemberGetQuery),
				),
			),
		},
		replicaGet: {
			metric: metric.New(
				"Return the replicas.",
				params.Join(
					params.BaseParams,
					params.TLSParams,
					params.AzureParams,
				),
				false,
			),
			handler: handlers.WithJSONResponse(
				p.conns.WithConnHandlerFunc(
					handlers.QueryHandlerFunc(replicaGetQuery),
				),
			),
		},
		version: {
			metric: metric.New(
				"Returns the MSSQL server version.",
				params.Join(
					params.BaseParams,
					params.TLSParams,
					params.AzureParams,
				),
				false,
			),
			handler: p.conns.WithConnHandlerFunc(handlers.VersionHandler),
		},
	}

	metricSet := metric.MetricSet{}

	for k, m := range p.metrics {
		metricSet[string(k)] = m.metric
	}

	err := plugin.RegisterMetrics(p, Name, metricSet.List()...)
	if err != nil {
		return errs.Wrap(err, "failed to register metrics")
	}

	return nil
}
