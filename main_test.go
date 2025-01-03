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
	"testing"

	_ "github.com/microsoft/go-mssqldb"
)

func Test_main(t *testing.T) {
	t.Skip("untestable, keep file for coverage reporting")
	t.Parallel()

	tests := []struct {
		name string
	}{}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			main()
		})
	}
}
