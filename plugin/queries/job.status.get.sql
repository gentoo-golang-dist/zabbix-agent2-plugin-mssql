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
