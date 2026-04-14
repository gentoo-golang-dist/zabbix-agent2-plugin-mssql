FROM zabbix/zabbix-agent2:ubuntu-trunk


USER root

WORKDIR /var/lib/zabbix

# dependency needed for zabbix should be removed when ubuntu trunk is fixed
RUN apt-get update && apt-get install -y libc-ares2
RUN apt-get update && apt-get install -y curl

COPY test-keys.sh get-jq.sh ./

RUN bash -c ./get-jq.sh

ENTRYPOINT ["bash", "-c", "/var/lib/zabbix/test-keys.sh"]
