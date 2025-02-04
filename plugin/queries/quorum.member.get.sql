SELECT a.member_name, a.member_type, a.member_state, a.number_of_quorum_votes, b.cluster_name
FROM sys.dm_hadr_cluster_members a
CROSS JOIN sys.dm_hadr_cluster b
