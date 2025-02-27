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

package fileservice

import (
	"bytes"
	"context"
	"encoding/gob"
	"io"
	"testing"

	"github.com/matrixorigin/matrixone/pkg/perfcounter"
	"github.com/stretchr/testify/assert"
)

func testCachingFileService(
	t *testing.T,
	newFS func() CachingFileService,
) {

	fs := newFS()
	fs.SetAsyncUpdate(false)
	ctx := context.Background()
	var counterSet perfcounter.CounterSet
	ctx = perfcounter.WithCounterSet(ctx, &counterSet)

	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(map[int]int{
		42: 42,
	})
	assert.Nil(t, err)
	data := buf.Bytes()

	err = fs.Write(ctx, IOVector{
		FilePath: "foo",
		Entries: []IOEntry{
			{
				Size: int64(len(data)),
				Data: data,
			},
		},
	})
	assert.Nil(t, err)

	makeVec := func() *IOVector {
		return &IOVector{
			FilePath: "foo",
			Entries: []IOEntry{
				{
					Size: int64(len(data)),
					ToObject: func(r io.Reader, data []byte) (any, int64, error) {
						bs, err := io.ReadAll(r)
						assert.Nil(t, err)
						if len(data) > 0 {
							assert.Equal(t, bs, data)
						}
						var m map[int]int
						if err := gob.NewDecoder(bytes.NewReader(bs)).Decode(&m); err != nil {
							return nil, 0, err
						}
						return m, 1, nil
					},
				},
			},
		}
	}

	// nocache
	vec := makeVec()
	vec.NoCache = true
	err = fs.Read(ctx, vec)
	assert.Nil(t, err)
	m, ok := vec.Entries[0].Object.(map[int]int)
	assert.True(t, ok)
	assert.Equal(t, 1, len(m))
	assert.Equal(t, 42, m[42])
	assert.Equal(t, int64(1), vec.Entries[0].ObjectSize)
	assert.Equal(t, int64(0), counterSet.FileService.Cache.Read.Load())
	assert.Equal(t, int64(0), counterSet.FileService.Cache.Hit.Load())

	// read, not hit
	vec = makeVec()
	err = fs.Read(ctx, vec)
	assert.Nil(t, err)
	m, ok = vec.Entries[0].Object.(map[int]int)
	assert.True(t, ok)
	assert.Equal(t, 1, len(m))
	assert.Equal(t, 42, m[42])
	assert.Equal(t, int64(1), vec.Entries[0].ObjectSize)
	assert.Equal(t, int64(1), counterSet.FileService.Cache.Read.Load())
	assert.Equal(t, int64(0), counterSet.FileService.Cache.Hit.Load())

	// read again, hit cache
	vec = makeVec()
	err = fs.Read(ctx, vec)
	assert.Nil(t, err)
	m, ok = vec.Entries[0].Object.(map[int]int)
	assert.True(t, ok)
	assert.Equal(t, 1, len(m))
	assert.Equal(t, 42, m[42])
	assert.Equal(t, int64(1), vec.Entries[0].ObjectSize)
	assert.Equal(t, int64(2), counterSet.FileService.Cache.Read.Load())
	assert.Equal(t, int64(1), counterSet.FileService.Cache.Hit.Load())

	// flush
	fs.FlushCache()

}
