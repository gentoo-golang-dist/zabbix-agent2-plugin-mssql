#!/bin/bash

if [ "$MSSQL_VERSION" = "2022" ]; then
    /opt/mssql-tools/bin/sqlcmd \
        -U sa \
        -S "$MSSQL_URL" \
        -P "$MSSQL_SA_PASSWORD" \
        -i /root/setup-2022.sql
else
    /opt/mssql-tools/bin/sqlcmd \
        -U sa \
        -S "$MSSQL_URL" \
        -P "$MSSQL_SA_PASSWORD" \
        -i /root/setup-2019-and-below.sql
fi
