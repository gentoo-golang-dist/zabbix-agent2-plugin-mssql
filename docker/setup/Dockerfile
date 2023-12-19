FROM mcr.microsoft.com/mssql/server:2022-CU10-ubuntu-22.04

USER root
WORKDIR /root

COPY setup.sh setup-2022.sql setup-2019-and-below.sql ./

ENTRYPOINT ["bash", "-c", "/root/setup.sh"]
