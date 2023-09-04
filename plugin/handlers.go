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
	"git.zabbix.com/ap/mssql/plugin/dbconn"
	"git.zabbix.com/ap/plugin-support/zbxerr"
)

func jobStatusHandler(conn dbconn.Queryer) (any, error) {
	//nolint:lll
	const jobStatusQuery = `
SELECT sj.name AS JobName,
  sj.enabled AS Enabled,
  sjs.last_run_outcome AS RunStatus,
  sjs.last_outcome_message AS LastRunStatusMessage,
  sjs.last_run_duration/10000*3600 + sjs.last_run_duration/100%100*60 + sjs.last_run_duration%100 AS RunDuration,
  CASE sjs.last_run_date
    WHEN 0 THEN NULL
    ELSE msdb.dbo.agent_datetime(sjs.last_run_date,sjs.last_run_time)
  END AS LastRunDateTime,
  sja.next_scheduled_run_date AS NextRunDateTime
FROM msdb..sysjobs AS sj
LEFT JOIN msdb..sysjobservers AS sjs
  ON sj.job_id = sjs.job_id
LEFT JOIN (
  SELECT job.job_id,
    max(act.session_id) AS s_id,
    max(act.next_scheduled_run_date) AS next_scheduled_run_date
  FROM msdb..sysjobs AS job
  LEFT JOIN msdb..sysjobactivity AS act
    ON act.job_id = job.job_id
  GROUP BY  job.job_id ) AS sja
    ON sja.job_id = sj.job_id
WHERE Enabled = 1
`

	rows, err := conn.Query(jobStatusQuery)
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to query")
	}

	defer func() { rows.Close() }()

	var result []any

	cols, err := rows.Columns()
	if err != nil {
		return nil, zbxerr.Wrap(err, "failed to get columns")
	}

	result = append(result, cols)

	for rows.Next() {
		row := make([]any, len(cols))

		err = rows.Scan(row...)
		if err != nil {
			return nil, zbxerr.Wrap(err, "failed to scan row")
		}

		result = append(result, row) //nolint:asasalint
	}

	return result, nil
}
