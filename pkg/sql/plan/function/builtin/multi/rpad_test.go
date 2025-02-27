// Copyright 2022 Matrix Origin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package multi

import (
	"testing"

	"github.com/matrixorigin/matrixone/pkg/container/types"
	"github.com/matrixorigin/matrixone/pkg/container/vector"
	"github.com/matrixorigin/matrixone/pkg/testutil"
	"github.com/stretchr/testify/require"
)

func TestRpadVarchar(t *testing.T) {
	cases := []struct {
		name      string
		vecs      []*vector.Vector
		wantBytes []byte
	}{
		{
			name:      "TEST01",
			vecs:      makeRpadVectors("hello", 1, "#", []int{1, 1, 1}),
			wantBytes: []byte("h"),
		},
		{
			name:      "TEST02",
			vecs:      makeRpadVectors("hello", 10, "#", []int{1, 1, 1}),
			wantBytes: []byte("hello#####"),
		},
		{
			name:      "TEST03",
			vecs:      makeRpadVectors("hello", 15, "#@&", []int{1, 1, 1}),
			wantBytes: []byte("hello#@&#@&#@&#"),
		},
		{
			name:      "TEST04",
			vecs:      makeRpadVectors("12345678", 10, "abcdefgh", []int{1, 1, 1}),
			wantBytes: []byte("12345678ab"),
		},
		{
			name:      "TEST05",
			vecs:      makeRpadVectors("hello", 0, "#@&", []int{1, 1, 1}),
			wantBytes: []byte(""),
		},
		{
			name:      "TEST06",
			vecs:      makeRpadVectors("hello", -1, "#@&", []int{1, 1, 1}),
			wantBytes: []byte(nil),
		},
		{
			name:      "Tx",
			vecs:      makeRpadVectors("hello", 1, "", []int{1, 1, 1}),
			wantBytes: []byte("h"),
		},
		{
			name:      "Tx2",
			vecs:      makeRpadVectors("", 5, "x", []int{1, 1, 1}),
			wantBytes: []byte("xxxxx"),
		},
		{
			name:      "Tx3",
			vecs:      makeRpadVectors("你好", 10, "再见", []int{1, 1, 1}),
			wantBytes: []byte("你好再见再见再见再见"),
		},
		{
			name:      "tx4",
			vecs:      makeRpadVectors("hello", -1, "#@&", []int{0, 0, 0}),
			wantBytes: []byte(nil),
		},
		{
			name:      "tx5",
			vecs:      makeRpadVectors("hello", -1, "#@&", []int{0, 0, 1}),
			wantBytes: []byte(nil),
		},
		{
			name:      "tx6",
			vecs:      makeRpadVectors("hello", -1, "#@&", []int{0, 1, 0}),
			wantBytes: []byte(nil),
		},
		{
			name:      "tx6",
			vecs:      makeRpadVectors("hello", -1, "#@&", []int{1, 0, 0}),
			wantBytes: []byte(nil),
		},
		{
			name:      "tx6",
			vecs:      makeRpadVectors("hello", -1, "#@&", []int{1, 1, 0}),
			wantBytes: []byte(nil),
		},
		{
			name:      "tx6",
			vecs:      makeRpadVectors("hello", -1, "#@&", []int{1, 0, 1}),
			wantBytes: []byte(nil),
		},
		{
			name:      "tx6",
			vecs:      makeRpadVectors("hello", -1, "#@&", []int{0, 1, 1}),
			wantBytes: []byte(nil),
		},
		{
			name:      "tx6",
			vecs:      makeRpadVectors("hello", -1, "#@&", []int{1, 1, 1}),
			wantBytes: []byte(nil),
		},
		{
			name:      "tx6",
			vecs:      makeRpadVectors("a你", 15, "见", []int{1, 1, 1}),
			wantBytes: []byte("a你见见见见见见见见见见见见见"),
		},
		{
			name:      "tx6",
			vecs:      makeRpadVectors("a你a", 15, "见a", []int{1, 1, 1}),
			wantBytes: []byte("a你a见a见a见a见a见a见a"),
		},
		{
			name:      "tx6",
			vecs:      makeRpadVectors("a你aa", 15, "见aa", []int{1, 1, 1}),
			wantBytes: []byte("a你aa见aa见aa见aa见a"),
		},
		{
			name:      "tx6",
			vecs:      makeRpadVectors("a你aaa", 15, "见aaa", []int{1, 1, 1}),
			wantBytes: []byte("a你aaa见aaa见aaa见a"),
		},
		{
			name:      "tx6",
			vecs:      makeRpadVectors("a你aaaa", 15, "见aaaa", []int{1, 1, 1}),
			wantBytes: []byte("a你aaaa见aaaa见aaa"),
		},
		{
			name:      "tx6",
			vecs:      makeRpadVectors("aaaaaaaa", 4, "bbb", []int{1, 1, 1}),
			wantBytes: []byte("aaaa"),
		},
	}

	proc := testutil.NewProcess()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rpad, err := Rpad(c.vecs, proc)
			if err != nil {
				t.Fatal(err)
			}
			if c.wantBytes == nil {
				require.Equal(t, rpad.IsConstNull(), true)
			} else {
				require.Equal(t, c.wantBytes, rpad.GetBytesAt(0))
			}

		})
	}

}

func makeRpadVectors(src string, length int64, pad string, nils []int) []*vector.Vector {
	vec := make([]*vector.Vector, 3)
	vec[0] = vector.NewConstBytes(types.T_varchar.ToType(), []byte(src), 1, testutil.TestUtilMp)
	vec[1] = vector.NewConstFixed(types.T_int64.ToType(), length, 1, testutil.TestUtilMp)
	vec[2] = vector.NewConstBytes(types.T_varchar.ToType(), []byte(pad), 1, testutil.TestUtilMp)
	for i, n := range nils {
		if n == 0 {
			vector.SetConstNull(vec[i], 1, testutil.TestUtilMp)
		}
	}
	return vec
}
