DECLARE @SQLNAME NVARCHAR(22)
SET @SQLNAME = CASE WHEN @@SERVICENAME = 'MSSQLSERVER' THEN 'SQLServer' ELSE 'MSSQL$' + @@SERVICENAME END
SELECT RTRIM(object_name) AS object_name,
  RTRIM(counter_name) AS counter_name,
  RTRIM(instance_name) AS instance_name,
  RTRIM(cntr_value) AS cntr_value
FROM sys.dm_os_performance_counters
UNION SELECT @SQLNAME AS object_name,
  'Version' AS counter_name,
  @@version AS instance_name,
  CAST(0 as bigint) AS cntr_value
UNION SELECT @SQLNAME AS object_name,
  'Uptime' AS counter_name,
  '' AS instance_name,
  CAST(DATEDIFF(second, sqlserver_start_time, GETDATE()) as bigint) AS cntr_value
FROM sys.dm_os_sys_info
UNION SELECT @SQLNAME + ':Databases' AS object_name,
  'State' AS counter_name,
  name AS instance_name,
  state AS cntr_value
FROM sys.databases
UNION SELECT a.object_name,
  'BufferCacheHitRatio' AS counter_name,
  '' AS instance_name,
  cast(a.cntr_value * ISNULL((100.0 / NULLIF(b.cntr_value,0)),0) AS dec(3, 0)) AS cntr_value
FROM sys.dm_os_performance_counters a
JOIN (
  SELECT cntr_value,
    OBJECT_NAME
  FROM sys.dm_os_performance_counters
  WHERE counter_name = 'Buffer cache hit ratio base'
    AND OBJECT_NAME = @SQLNAME + ':Buffer Manager'
) b
  ON a.OBJECT_NAME = b.OBJECT_NAME
WHERE a.counter_name = 'Buffer cache hit ratio'
  AND a.OBJECT_NAME = @SQLNAME + ':Buffer Manager'
UNION SELECT a.object_name,
  'WorktablesFromCacheRatio' AS counter_name,
  '' AS instance_name,
  cast(a.cntr_value * ISNULL((100.0 / NULLIF(b.cntr_value,0)),0) AS dec(3, 0)) AS cntr_value
FROM sys.dm_os_performance_counters a
JOIN (
  SELECT cntr_value,
    OBJECT_NAME
  FROM sys.dm_os_performance_counters
  WHERE counter_name = 'Worktables From Cache Base'
    AND OBJECT_NAME = @SQLNAME + ':Access Methods'
) b
  ON a.OBJECT_NAME = b.OBJECT_NAME
WHERE a.counter_name = 'Worktables From Cache Ratio'
  AND a.OBJECT_NAME = @SQLNAME + ':Access Methods'
UNION SELECT a.object_name,
  'CacheHitRatio' AS counter_name,
  '_Total' AS instance_name,
  cast(a.cntr_value * ISNULL((100.0 / NULLIF(b.cntr_value,0)),0) AS dec(3, 0)) AS cntr_value
FROM sys.dm_os_performance_counters a
JOIN (
  SELECT cntr_value,
    OBJECT_NAME
  FROM sys.dm_os_performance_counters
  WHERE counter_name = 'Cache Hit Ratio base'
    AND OBJECT_NAME = @SQLNAME + ':Plan Cache'
    AND instance_name = '_Total'
) b
  ON a.OBJECT_NAME = b.OBJECT_NAME
WHERE a.counter_name = 'Cache Hit Ratio'
  AND a.OBJECT_NAME = @SQLNAME + ':Plan Cache'
  AND instance_name = '_Total'
