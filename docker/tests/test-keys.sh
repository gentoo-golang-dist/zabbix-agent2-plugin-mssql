#!/bin/bash

URI="sqlserver://mssql-server.zabbix.sandbox:1433"
USER="sa"
PASSWORD="zabbix#33"
CUSTOM_QUERY="test"

# availabilityGroupGet = mssqlMetricKey("mssql.availability.group.get")
# customQuery          = mssqlMetricKey("mssql.custom.query")
# dbGet                = mssqlMetricKey("mssql.db.get")
# jobStatusGet         = mssqlMetricKey("mssql.job.status.get")
# lastBackupGet        = mssqlMetricKey("mssql.last.backup.get")
# localDBGet           = mssqlMetricKey("mssql.local.db.get")
# mirroringGet         = mssqlMetricKey("mssql.mirroring.get")
# nonLocalDBGet        = mssqlMetricKey("mssql.nonlocal.db.get")
# perfCounterGet       = mssqlMetricKey("mssql.perfcounter.get")
# ping                 = mssqlMetricKey("mssql.ping")
# quorumGet            = mssqlMetricKey("mssql.quorum.get")
# quorumMemberGet      = mssqlMetricKey("mssql.quorum.member.get")
# replicaGet           = mssqlMetricKey("mssql.replica.get")
# version              = mssqlMetricKey("mssql.version")

params="[$URI,$USER,$PASSWORD]"

keys=(
    "mssql.availability.group.get"
    "mssql.db.get"
    "mssql.job.status.get"
    "mssql.last.backup.get"
    "mssql.local.db.get"
    "mssql.mirroring.get"
    "mssql.nonlocal.db.get"
    "mssql.perfcounter.get"
    "mssql.ping"
    "mssql.quorum.get"
    "mssql.quorum.member.get"
    "mssql.replica.get"
)

function test_custom_query() {
    out=$(
        zabbix_get \
            -s '127.0.0.1' \
            -p '10050' \
            -k "mssql.custom.query[$URI,$USER,$PASSWORD,$CUSTOM_QUERY]"
    )

    echo "$out" | ./jq >/dev/null
    if [ $? -ne 0 ]; then
        echo "FAIL mssql.custom.query"
        echo "$out"
        exit 1
    fi

    echo "PASS mssql.custom.query"
}

out=$(
    zabbix_get \
        -s '127.0.0.1' \
        -p '10050' \
        -k "mssql.version$params"
)
echo "version; $out"

test_custom_query

for key in "${keys[@]}"; do
    out=$(
        zabbix_get \
            -s '127.0.0.1' \
            -p '10050' \
            -k "$key$params"
    )

    echo "$out" | ./jq >/dev/null
    if [ $? -ne 0 ]; then
        echo "FAIL $key"
        echo "$out"
        exit 1
    fi

    echo "PASS $key"
done
