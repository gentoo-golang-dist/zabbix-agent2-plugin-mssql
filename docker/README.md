# MSSQL Plugin Development Environment

## Overview 📖

This Docker Compose setup is designed to facilitate the development and testing
of a Zabbix Agent 2 plugin for Microsoft SQL Server (MSSQL). It automates the
deployment of multiple service containers to create a ready-made environment
that enables developers to test the MSSQL plugin across different versions of
the MSSQL server.

## Architecture 🏗️

The environment consists of several linked Docker containers, each serving a
distinct role:

1. **mssql-plugin**: This container runs the Zabbix Agent 2 with the MSSQL
   plugin, ready for connections and data-gathering operations.
2. **mssql-<version>-plugin-tests**: These containers are dedicated to running
   the plugin tests against specific versions of MSSQL (2022, 2019, and 2017).
   They rely on the `mssql-plugin` service being in a healthy state.
3. **mssql-<version>-setup**: These containers are responsible for executing
   database setup and configuration scripts tailored for each MSSQL version they
   are matched with.
4. **mssql-<version>**: These are the actual database instances of MSSQL for the
   respective versions (2022, 2019, and 2017). They must accept the EULA and
   have a strong SA (System Administrator) password set, which is mandated by
   MSSQL for security reasons. MSSQL database docker images can only be run on
   x86_64 architectures. 😞

## Prerequisites ✅

Before you can use this Docker Compose setup, ensure you have the following
installed:

- Docker Engine
- Docker Compose
- Git (optional, for getting the Dockerfile and related resources)
- Internet access for pulling images from the public Docker registry

## Testing Plugin Changes 🧪

To test changes made to the MSSQL plugin:

1. Implement your changes in the plugin code and ensure the new code resides at
   the expected path set by the `context` and `dockerfile` directives in
   `docker-compose.yml`.
2. Rebuild the Docker images to include your changes.
   ```bash
   docker-compose up --build mssql-plugin
   ```
3. Run the test for the specific MSSQL version you want to validate against.
   ```bash
   docker-compose up mssql-2022-plugin-tests
   ```
   Replace `2022` with `2019` or `2017` to test against other versions. **Note**
   It's recommended to run tests for one version at a time.

## Notes 📝

- This environment encapsulates dependencies and system configurations within
  Docker containers, reducing discrepancies between development and production
  environments.
- Logs and outputs from the tests can be inspected by accessing the respective
  containers' logs.

## Getting Help 🆘

If you encounter issues or have questions about the setup, consider consulting
the documentation for each of the tools used within this development
environment, or reach out to the Zabbix agent 2 MSSQL plugin developers via
[Zabbix support system](https://support.zabbix.com/secure/Dashboard.jspa)

Remember, we refine our software through continuous testing and iteration. This
Docker Compose setup is an essential tool in this continual refinement process.
Happy coding and testing!
