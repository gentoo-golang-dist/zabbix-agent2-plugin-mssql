#!/bin/bash

if [ "$MSSQL_VERSION" = "2022" ]; then
    /opt/mssql-tools18/bin/sqlcmd \
        -U sa \
        -C \
        -S "$MSSQL_URL" \
        -P "$MSSQL_SA_PASSWORD" \
        -i /root/setup-2022.sql
else
    /opt/mssql-tools18/bin/sqlcmd \
        -U sa \
        -C \
        -S "$MSSQL_URL" \
        -P "$MSSQL_SA_PASSWORD" \
        -i /root/setup-2019-and-below.sql
fi
