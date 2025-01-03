/*
** Zabbix
** Copyright (C) 2001-2025 Zabbix SIA
**
** Licensed under the Apache License, Version 2.0 (the "License");
** you may not use this file except in compliance with the License.
** You may obtain a copy of the License at
**
**     http://www.apache.org/licenses/LICENSE-2.0
**
** Unless required by applicable law or agreed to in writing, software
** distributed under the License is distributed on an "AS IS" BASIS,
** WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
** See the License for the specific language governing permissions and
** limitations under the License.
**/

package main

import (
	"errors"
	"fmt"
	"os"

	_ "github.com/microsoft/go-mssqldb"
	"golang.zabbix.com/plugin/mssql/plugin"
	"golang.zabbix.com/sdk/plugin/flag"
	"golang.zabbix.com/sdk/zbxerr"
)

const copyrightMessage = //
`Copyright (C) 2001-%d Zabbix SIA
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.`

//nolint:gochecknoglobals,revive // required ALL_CAPS by build scripts
var (
	PLUGIN_VERSION_RC    = "rc1"
	PLUGIN_VERSION_MAJOR = 6
	PLUGIN_VERSION_MINOR = 0
	PLUGIN_VERSION_PATCH = 38
	PLUGIN_LICENSE_YEAR  = 2025
)

func main() {
	err := flag.HandleFlags(
		plugin.Name,
		os.Args[0],
		fmt.Sprintf(copyrightMessage, PLUGIN_LICENSE_YEAR),
		PLUGIN_VERSION_RC,
		PLUGIN_VERSION_MAJOR,
		PLUGIN_VERSION_MINOR,
		PLUGIN_VERSION_PATCH,
	)
	if err != nil {
		if errors.Is(err, zbxerr.ErrorOSExitZero) {
			return
		}

		panic(err)
	}

	err = plugin.Launch()
	if err != nil {
		panic(err)
	}
}
