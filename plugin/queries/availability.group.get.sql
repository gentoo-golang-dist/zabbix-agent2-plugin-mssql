SELECT
  ag.name as group_name,
  ISNULL(ags.primary_recovery_health, 2) as primary_recovery_health,
  ISNULL(ags.primary_replica, 'Unknown') as primary_replica,
  ISNULL(
    ags.secondary_recovery_health, 2
  ) as secondary_recovery_health,
  ags.synchronization_health as synchronization_health
FROM
  sys.dm_hadr_availability_group_states ags
JOIN sys.availability_groups ag ON ag.group_id = ags.group_id
