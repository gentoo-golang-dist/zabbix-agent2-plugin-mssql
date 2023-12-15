FROM golang:1.20

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN go build -v -o mssql-plugin

FROM zabbix/zabbix-agent2:ubuntu-trunk
COPY --from=0 /usr/src/app/mssql-plugin /var/lib/zabbix/mssql-plugin

COPY docker/mssql-plugin/custom-queries /var/lib/zabbix/custom-queries

RUN echo "Plugins.MSSQL.System.Path=/var/lib/zabbix/mssql-plugin" >> \
    /etc/zabbix/zabbix_agentd.d/mssql-plugin.conf

RUN echo "Plugins.MSSQL.CustomQueriesDir=/var/lib/zabbix/custom-queries" >> \
    /etc/zabbix/zabbix_agentd.d/mssql-plugin.conf
