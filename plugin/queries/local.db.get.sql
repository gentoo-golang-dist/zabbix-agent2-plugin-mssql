SELECT ag.name AS group_name,
  drs.database_state as database_state,
  drs.is_suspended as is_suspended,
  drs.synchronization_health as synchronization_health,
  arcs.replica_server_name AS replica_name,
  db_name(drs.database_id) AS dbname,
  drs.is_local
FROM sys.dm_hadr_database_replica_states drs
JOIN sys.dm_hadr_availability_replica_cluster_states arcs
  ON arcs.replica_id = drs.replica_id
JOIN sys.availability_groups ag
  ON ag.group_id = arcs.group_id
JOIN sys.dm_hadr_availability_replica_states ars
  ON ars.replica_id = arcs.replica_id
WHERE drs.is_local = 1
