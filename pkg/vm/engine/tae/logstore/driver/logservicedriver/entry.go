// Copyright 2021 Matrix Origin
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

package logservicedriver

import (
	"bytes"
	"io"
	"math"
	"sync"
	"sync/atomic"

	"github.com/matrixorigin/matrixone/pkg/common/moerr"
	"github.com/matrixorigin/matrixone/pkg/container/types"
	"github.com/matrixorigin/matrixone/pkg/logservice"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/tae/common"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/tae/logstore/driver"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/tae/logstore/driver/entry"
)

type MetaType uint8

const (
	TInvalid MetaType = iota
	TNormal
	TReplay
)

type meta struct {
	metaType    MetaType
	appended    uint64
	addr        map[uint64]uint64
	payloadSize uint64
}

func newMeta() *meta {
	return &meta{addr: make(map[uint64]uint64), metaType: TNormal}
}
func (m *meta) SetType(t MetaType) {
	m.metaType = t
}
func (m *meta) GetType() MetaType {
	return m.metaType
}
func (m *meta) SetAppended(appended uint64) {
	m.appended = appended
}
func (m *meta) GetMinLsn() uint64 {
	min := uint64(0)
	min = math.MaxUint64
	for lsn := range m.addr {
		if lsn < min {
			min = lsn
		}
	}
	return min
}
func (m *meta) GetMaxLsn() uint64 {
	max := uint64(0)
	for lsn := range m.addr {
		if lsn > max {
			max = lsn
		}
	}
	return max
}
func (m *meta) WriteTo(w io.Writer) (n int64, err error) {
	metaType := uint8(m.metaType)
	if _, err = w.Write(types.EncodeUint8(&metaType)); err != nil {
		return
	}
	n += 1
	if _, err = w.Write(types.EncodeUint64(&m.appended)); err != nil {
		return
	}
	n += 8
	length := uint16(len(m.addr))
	if _, err = w.Write(types.EncodeUint16(&length)); err != nil {
		return
	}
	n += 2
	for lsn, offset := range m.addr {
		if _, err = w.Write(types.EncodeUint64(&lsn)); err != nil {
			return
		}
		n += 8
		if _, err = w.Write(types.EncodeUint64(&offset)); err != nil {
			return
		}
		n += 8
	}
	if _, err = w.Write(types.EncodeUint64(&m.payloadSize)); err != nil {
		return
	}
	n += 8
	return
}

func (m *meta) ReadFrom(r io.Reader) (n int64, err error) {
	metaType := uint8(0)
	if _, err = r.Read(types.EncodeUint8(&metaType)); err != nil {
		return
	}
	m.metaType = MetaType(metaType)
	n += 1
	if _, err = r.Read(types.EncodeUint64(&m.appended)); err != nil {
		return
	}
	n += 8
	length := uint16(0)
	if _, err = r.Read(types.EncodeUint16(&length)); err != nil {
		return
	}
	n += 2
	m.addr = make(map[uint64]uint64)
	for i := 0; i < int(length); i++ {
		lsn := uint64(0)
		if _, err = r.Read(types.EncodeUint64(&lsn)); err != nil {
			return
		}
		n += 8
		offset := uint64(0)
		if _, err = r.Read(types.EncodeUint64(&offset)); err != nil {
			return
		}
		n += 8
		m.addr[lsn] = offset
	}
	if _, err = r.Read(types.EncodeUint64(&m.payloadSize)); err != nil {
		return
	}
	n += 8
	return
}

func (m *meta) Unmarshal(buf []byte) error {
	bbuf := bytes.NewBuffer(buf)
	_, err := m.ReadFrom(bbuf)
	return err
}

func (m *meta) Marshal() (buf []byte, err error) {
	var bbuf bytes.Buffer
	if _, err = m.WriteTo(&bbuf); err != nil {
		return
	}
	buf = bbuf.Bytes()
	return
}

// read: logrecord -> meta+payload -> entry
// write: entries+meta -> payload -> record
type recordEntry struct {
	*meta
	entries []*entry.Entry
	cmd     *ReplayCmd

	payload     []byte
	unmarshaled atomic.Uint32
	mashalMu    sync.RWMutex
}

func newRecordEntry() *recordEntry {
	return &recordEntry{entries: make([]*entry.Entry, 0), meta: newMeta()}
}

func newEmptyRecordEntry(r logservice.LogRecord) *recordEntry {
	payload := make([]byte, len(r.Payload()))
	copy(payload, r.Payload())
	return &recordEntry{payload: payload, meta: newMeta(), mashalMu: sync.RWMutex{}}
}

func (r *recordEntry) replay(h driver.ApplyHandle) (addr *common.ClosedIntervals) {
	bbuf := bytes.NewBuffer(r.payload)
	lsns := make([]uint64, 0)
	for lsn := range r.meta.addr {
		lsns = append(lsns, lsn)
		e := entry.NewEmptyEntry()
		e.ReadFrom(bbuf)
		h(e)
		e.Entry.Free()
	}
	intervals := common.NewClosedIntervalsBySlice(lsns)
	return intervals
}
func (r *recordEntry) append(e *entry.Entry) {
	r.entries = append(r.entries, e)
	r.meta.addr[e.Lsn] = uint64(r.payloadSize)
	r.payloadSize += uint64(e.GetSize())
}

func (r *recordEntry) WriteTo(w io.Writer) (n int64, err error) {
	n1, err := r.meta.WriteTo(w)
	if err != nil {
		return 0, err
	}
	n += n1
	switch r.meta.metaType {
	case TNormal:
		for _, e := range r.entries {
			n1, err = e.WriteTo(w)
			if err != nil {
				return
			}
			n += n1
		}
	case TReplay:
		n1, err = r.cmd.WriteTo(w)
		if err != nil {
			return
		}
		n += n1
	default:
		panic("invalid type")
	}
	return
}

func (r *recordEntry) ReadFrom(reader io.Reader) (n int64, err error) {
	n1, err := r.meta.ReadFrom(reader)
	if err != nil {
		return 0, err
	}
	n += n1
	payload := make([]byte, r.meta.payloadSize)
	n2, err := reader.Read(payload)
	if err != nil {
		return 0, err
	}
	if n2 != int(r.meta.payloadSize) {
		panic(moerr.NewInternalErrorNoCtx("logic err: err is %v, expect %d, get %d", err, r.meta.payloadSize, n2))
	}
	r.payload = payload
	return
}

func (r *recordEntry) Unmarshal(buf []byte) error {
	bbuf := bytes.NewBuffer(buf)
	_, err := r.ReadFrom(bbuf)
	return err
}

func (r *recordEntry) Marshal() (buf []byte, err error) {
	var bbuf bytes.Buffer
	if _, err = r.WriteTo(&bbuf); err != nil {
		return
	}
	buf = bbuf.Bytes()
	return
}
func (r *recordEntry) prepareRecord() (size int) {
	var err error
	r.payload, err = r.Marshal()
	if err != nil {
		panic(err)
	}
	return len(r.payload)
}

func (r *recordEntry) unmarshal() {
	if r.unmarshaled.Load() == 1 {
		return
	}
	r.mashalMu.Lock()
	defer r.mashalMu.Unlock()
	if r.unmarshaled.Load() == 1 {
		return
	}
	buf := r.payload
	r.payload = nil
	err := r.Unmarshal(buf)
	if err != nil {
		panic(err)
	}
	r.unmarshaled.Store(1)
}

func (r *recordEntry) readEntry(lsn uint64) *entry.Entry {
	r.unmarshal()
	offset := r.meta.addr[lsn]
	bbuf := bytes.NewBuffer(r.payload[offset:])
	e := entry.NewEmptyEntry()
	e.ReadFrom(bbuf)
	e.Lsn = lsn
	return e
}
