SELECT
  ISNULL(m.mirroring_role, 0) as mirroring_role,
  ISNULL(m.mirroring_role_sequence, 0) as mirroring_role_sequence,
  ISNULL(m.mirroring_state, 7) as mirroring_state,
  ISNULL(m.mirroring_witness_state, 3) as mirroring_witness_state,
  ISNULL(m.mirroring_safety_level, 3) as mirroring_safety_level,
  db_name(m.database_id) as dbname
FROM
  sys.database_mirroring as m
WHERE m.mirroring_state_desc IS NOT NULL
