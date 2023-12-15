#!/bin/bash

/opt/mssql-tools/bin/sqlcmd \
    -U sa \
    -P "$MSSQL_SA_PASSWORD" \
    -i /root/setup.sql
