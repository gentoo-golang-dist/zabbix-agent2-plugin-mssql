SELECT
  ars.connected_state as connected_state,
  ars.is_local as is_local,
  arcs.join_state as join_state,
  ISNULL(ars.operational_state, 6) as operational_state,
  ISNULL(ars.recovery_health, 2) as recovery_health,
  ars.role as role,
  ars.synchronization_health as synchronization_health,
  ag.name as group_name,
  arcs.replica_server_name as replica_name
FROM
  sys.dm_hadr_availability_replica_cluster_states as arcs
JOIN sys.availability_groups ag ON ag.group_id = arcs.group_id
JOIN sys.dm_hadr_availability_replica_states ars ON ars.replica_id = arcs.replica_id
