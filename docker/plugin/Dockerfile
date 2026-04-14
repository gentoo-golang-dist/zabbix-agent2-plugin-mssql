FROM golang:1.24.10 AS plugin-builder

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only re-downloading
# them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY main.go ./
COPY plugin ./plugin

RUN go build -v -o mssql-plugin

FROM zabbix/zabbix-agent2:ubuntu-trunk

# dependency needed for zabbix_get should be removed when ubuntu trunk is fixed
USER root
RUN apt-get update && apt-get install -y libc-ares2
USER zabbix

COPY --from=plugin-builder /usr/src/app/mssql-plugin /var/lib/zabbix/mssql-plugin

COPY docker/plugin/setup-files /var/lib/zabbix/setup-files

COPY docker/plugin/mssql.conf /etc/zabbix/zabbix_agentd.d/mssql.conf

ENV GODEBUG="x509negativeserial=1"
