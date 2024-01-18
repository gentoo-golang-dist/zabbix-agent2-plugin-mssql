SELECT bs.database_name as dbname,
  [type],
  DATEDIFF(SECOND, bs.backup_finish_date, getdate()) as time_since_last_backup,
  (DATEDIFF(SECOND, bs.backup_start_date, bs.backup_finish_date)) as duration,
  db.recovery_model as db_recovery_model
FROM msdb.dbo.backupset as bs
LEFT JOIN sys.databases as db ON bs.database_name = db.name
WHERE bs.database_name not in (
  SELECT AGDatabases.database_name AS Databasename
  FROM sys.dm_hadr_availability_group_states States
  INNER JOIN master.sys.availability_groups Groups
    ON States.group_id = Groups.group_id
  INNER JOIN sys.availability_databases_cluster AGDatabases
    ON Groups.group_id = AGDatabases.group_id
  WHERE primary_replica != @@Servername OR primary_replica is NULL
)
GROUP BY bs.database_name,
  backup_finish_date,
  [type],
  backup_start_date,
  db.recovery_model
HAVING backup_finish_date = (
  SELECT MAX(backup_finish_date)
  FROM msdb.dbo.backupset
  WHERE database_name = bs.database_name
    AND bs.type = [type]
)
ORDER BY bs.database_name
