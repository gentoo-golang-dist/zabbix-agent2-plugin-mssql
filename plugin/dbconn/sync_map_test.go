/*
** Copyright (C) 2001-2025 Zabbix SIA
**
** Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated
** documentation files (the "Software"), to deal in the Software without restriction, including without limitation the
** rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to
** permit persons to whom the Software is furnished to do so, subject to the following conditions:
**
** The above copyright notice and this permission notice shall be included in all copies or substantial portions
** of the Software.
**
** THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE
** WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
** COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
** TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
** SOFTWARE.
**/

package dbconn

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSyncMap_Store(t *testing.T) {
	t.Parallel()

	type arg struct {
		key   string
		value int
	}

	type args []arg

	type want struct {
		cnt int
	}

	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"+single",
			args{
				arg{"key1", 1},
			},
			want{1},
		},
		{
			"+duplicate",
			args{
				arg{"key1", 1},
				arg{"key1", 100},
			},
			want{1},
		},
		{
			"+unique",
			args{
				arg{"key1", 1},
				arg{"key2", 2},
			},
			want{2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := SyncMap[string, int]{}

			for _, v := range tt.args {
				got.Store(v.key, v.value)
			}

			gotCnt := got.Len()

			diff := cmp.Diff(gotCnt, tt.want.cnt)
			if diff != "" {
				t.Errorf("SyncMap.Store() = %s", diff)
			}
		})
	}
}

func TestSyncMap_Load(t *testing.T) {
	t.Parallel()

	type arg struct {
		key   string
		value int
	}

	type args []arg

	type want struct {
		key   string
		value int
	}

	type wants []want

	tests := []struct {
		name  string
		args  args
		wants wants
	}{
		{
			"+valid",
			args{
				arg{"key1", 1},
			},
			wants{
				want{"key1", 1},
			},
		},
		{
			"+duplicate",
			args{
				arg{"key1", 1},
				arg{"key1", 100},
			},
			wants{ //all iterations return last loaded
				want{"key1", 100},
				want{"key1", 100},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := SyncMap[string, int]{}

			for _, v := range tt.args {
				got.Store(v.key, v.value)
			}

			for i, v := range tt.args {
				gotVal, found := got.Load(v.key)

				if !found {
					t.Errorf("SyncMap.Load() = record with key %s not found!", v.key)
				}

				diff := cmp.Diff(gotVal, tt.wants[i].value)

				if diff != "" {
					t.Errorf("SyncMap.Load() = %s", diff)
				}
			}
		})
	}
}

func TestSyncMap_LoadOrStore(t *testing.T) {
	t.Parallel()

	type arg struct {
		key   string
		value int
	}

	type args []arg

	type want struct {
		value  int
		loaded bool
	}

	type wants []want

	tests := []struct {
		name  string
		args  args
		wants wants
	}{
		{
			"+stored",
			args{
				arg{"key1", 1},
				arg{"key2", 2},
			},
			wants{
				want{1, false},
				want{2, false},
			},
		},
		{
			"+loaded",
			args{
				arg{"key1", 1},
				arg{"key1", 2},
			},
			wants{
				want{1, false},
				want{1, true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := SyncMap[string, int]{}

			for i, v := range tt.args {
				gotVal, gotLoaded := got.LoadOrStore(v.key, v.value)
				gotFld := want{gotVal, gotLoaded}

				diff := cmp.Diff(
					gotFld, tt.wants[i],
					cmp.AllowUnexported(want{}))
				if diff != "" {
					t.Errorf("SyncMap.LoadOrStore() = %s", diff)
				}
			}
		})
	}
}

func TestSyncMap_Delete(t *testing.T) {
	t.Parallel()

	type field struct {
		key   string
		value int
	}

	type fields []field

	type arg struct {
		key string
	}

	type want struct {
		found bool
		cnt   int
		value int
	}

	tests := []struct {
		name   string
		fields fields
		arg    arg
		want   want
	}{
		{
			"+deleteExistingFromSingle",
			fields{
				field{"key1", 1},
			},
			arg{"key1"},
			want{false, 0, 0},
		},
		{
			"+deleteExistingFromMany",
			fields{
				field{"key1", 1},
				field{"key2", 2},
				field{"key3", 3},
			},
			arg{"key2"},
			want{false, 2, 0},
		},
		{
			"+deleteUnexistingFromMany",
			fields{
				field{"key1", 1},
				field{"key2", 2},
			},
			arg{"key99"},
			want{false, 2, 0},
		},
		{
			"+deleteUnexistingFromEmpty",
			fields{},
			arg{"key1"},
			want{false, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := SyncMap[string, int]{}

			// Seeding map with the test values provided by 'fields' variable
			for _, v := range tt.fields {
				got.LoadOrStore(v.key, v.value)
			}

			got.Delete(tt.arg.key)

			cnt := got.Len()
			value, found := got.Load(tt.arg.key)

			if cnt != tt.want.cnt ||
				found != tt.want.found ||
				value != tt.want.value {
				t.Error("SyncMap.Delete(): the record has not been deleted!")
			}
		})
	}
}

func TestSyncMap_Range(t *testing.T) {
	t.Parallel()

	type want struct {
		key   string
		value int
	}

	type wants []want

	tests := []struct {
		name  string
		wants wants
	}{
		{
			"+iterateMany",
			wants{
				want{"key1", 1},
				want{"key2", 2},
				want{"key3", 3},
			},
		},
		{
			"+iterateEmpty",
			wants{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := SyncMap[string, int]{}

			// Seeding map with the test values provided by 'fields' variable
			for _, v := range tt.wants {
				got.LoadOrStore(v.key, v.value)
			}

			got.Range(func(key string, value int) bool {
				gotVal, found := got.Load(key)
				if !found || value != gotVal {
					t.Error("SyncMap.Range(): the result does not equal with expected!")
				}

				return true
			})
		})
	}
}

func TestSyncMap_Len(t *testing.T) {
	t.Parallel()

	type field struct {
		key   string
		value int
	}

	type fields []field

	type want struct {
		len int
	}

	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			"+many",
			fields{
				field{"key1", 1},
				field{"key2", 2},
				field{"key3", 3},
			},
			want{3},
		},
		{
			"+empty",
			fields{},
			want{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := SyncMap[string, int]{}

			// Seeding map with the test values provided by 'fields' variable
			for _, v := range tt.fields {
				got.LoadOrStore(v.key, v.value)
			}

			gotLen := got.Len()
			diff := cmp.Diff(gotLen, tt.want.len)

			if diff != "" {
				t.Errorf("SyncMap.Len() = %s", diff)
			}
		})
	}
}
