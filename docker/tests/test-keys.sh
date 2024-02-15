#!/bin/bash
#
# very basic test for mssql-plugin

DEFAULT_AGENT2_URI="mssql-plugin"
DEFAULT_AGENT2_PORT="10050"

DEFAULT_MSSQL_URL="sqlserver://mssql:1433"
DEFAULT_MSSQL_USER="sa"
DEFAULT_MSSQL_PASSWORD=""
DEFAULT_MSSQL_CUSTOM_QUERY="test"

if [ -z "$AGENT2_URI" ]; then
    AGENT2_URI=$DEFAULT_AGENT2_URI
fi

if [ -z "$AGENT2_PORT" ]; then
    AGENT2_PORT=$DEFAULT_AGENT2_PORT
fi

if [ -z "$MSSQL_URL" ]; then
    MSSQL_URL=$DEFAULT_MSSQL_URL
fi

if [ -z "$MSSQL_USER" ]; then
    MSSQL_USER=$DEFAULT_MSSQL_USER
fi

if [ -z "$MSSQL_PASSWORD" ]; then
    MSSQL_PASSWORD=$DEFAULT_MSSQL_PASSWORD
fi

if [ -z "$MSSQL_CUSTOM_QUERY" ]; then
    MSSQL_CUSTOM_QUERY=$DEFAULT_MSSQL_CUSTOM_QUERY
fi

# mssql.availability.group.get
# mssql.custom.query
# mssql.db.get
# mssql.job.status.get
# mssql.last.backup.get
# mssql.local.db.get
# mssql.mirroring.get
# mssql.nonlocal.db.get
# mssql.perfcounter.get
# mssql.ping
# mssql.quorum.get
# mssql.quorum.member.get
# mssql.replica.get
# mssql.version

params="[$MSSQL_URL,$MSSQL_USER,$MSSQL_PASSWORD]"

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

# TODO: fix
function test_custom_query() {
    out=$(
        zabbix_get \
            -s "$AGENT2_URI" \
            -p "$AGENT2_PORT" \
            -k "mssql.custom.query[$MSSQL_URL,$MSSQL_USER,$MSSQL_PASSWORD,$MSSQL_CUSTOM_QUERY]" 2>&1
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
        -s "$AGENT2_URI" \
        -p "$AGENT2_PORT" \
        -k "mssql.version$params" 2>&1
)
echo "version; $out"

for key in "${keys[@]}"; do
    out=$(
        zabbix_get \
            -s "$AGENT2_URI" \
            -p "$AGENT2_PORT" \
            -k "$key$params" 2>&1
    )

    echo "$out" | ./jq >/dev/null
    if [ $? -ne 0 ]; then
        echo "FAIL $key"
        echo "$out"
        exit 1
    fi

    echo "PASS $key"
done

test_custom_query
