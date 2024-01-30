SELECT sj.name AS job_name,
  sj.enabled AS enabled,
  sjs.last_run_outcome AS run_status,
  sjs.last_outcome_message AS last_run_status_message,
  sjs.last_run_duration/10000*3600 + sjs.last_run_duration/100%100*60 + sjs.last_run_duration%100 AS run_duration,
  CASE sjs.last_run_date
    WHEN 0 THEN NULL
    ELSE CONVERT(
        VARCHAR(25),
        TODATETIMEOFFSET(msdb.dbo.agent_datetime(sjs.last_run_date,sjs.last_run_time), DATEPART(TZoffset, SYSDATETIMEOFFSET())),
        126)
  END AS last_run_date_time,
  sja.next_scheduled_run_date AS next_run_date_time
FROM msdb..sysjobs AS sj
LEFT JOIN msdb..sysjobservers AS sjs
  ON sj.job_id = sjs.job_id
LEFT JOIN (
  SELECT job.job_id,
    max(act.session_id) AS s_id,
    CONVERT(
        VARCHAR(25),
        TODATETIMEOFFSET(max(act.next_scheduled_run_date), DATEPART(TZoffset, SYSDATETIMEOFFSET())),
        126) AS next_scheduled_run_date
  FROM msdb..sysjobs AS job
  LEFT JOIN msdb..sysjobactivity AS act
    ON act.job_id = job.job_id
  GROUP BY  job.job_id ) AS sja
    ON sja.job_id = sj.job_id
WHERE Enabled = 1
