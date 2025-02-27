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

package moengine

import (
	"context"

	pkgcatalog "github.com/matrixorigin/matrixone/pkg/catalog"
	"github.com/matrixorigin/matrixone/pkg/container/batch"
	"github.com/matrixorigin/matrixone/pkg/container/vector"
	"github.com/matrixorigin/matrixone/pkg/objectio"
	apipb "github.com/matrixorigin/matrixone/pkg/pb/api"
	"github.com/matrixorigin/matrixone/pkg/vm/engine"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/tae/catalog"
	"github.com/matrixorigin/matrixone/pkg/vm/engine/tae/iface/handle"
)

var (
	_ engine.Relation = (*sysRelation)(nil)
)

func newSysRelation(h handle.Relation) *sysRelation {
	r := &sysRelation{}
	r.handle = h
	r.nodes = append(r.nodes, engine.Node{
		Addr: ADDR,
	})
	return r
}

func isSysRelation(name string) bool {
	if name == pkgcatalog.MO_DATABASE ||
		name == pkgcatalog.MO_TABLES ||
		name == pkgcatalog.MO_COLUMNS {
		return true
	}
	return false
}

func isSysRelationId(id uint64) bool {
	if id == pkgcatalog.MO_DATABASE_ID ||
		id == pkgcatalog.MO_TABLES_ID ||
		id == pkgcatalog.MO_COLUMNS_ID {
		return true
	}
	return false
}

func (s *sysRelation) Write(_ context.Context, _ *batch.Batch) error {
	return ErrReadOnly
}

func (s *sysRelation) AddBlksWithMetaLoc(
	_ context.Context,
	_ []objectio.ZoneMap,
	_ []objectio.Location,
) error {
	return ErrReadOnly
}

func (s *sysRelation) Update(_ context.Context, _ *batch.Batch) error {
	return ErrReadOnly
}

func (s *sysRelation) Delete(_ context.Context, _ *batch.Batch, _ string) error {
	return ErrReadOnly
}

func (s *sysRelation) DeleteByPhyAddrKeys(_ context.Context, _ *vector.Vector) error {
	return ErrReadOnly
}

func (s *sysRelation) AlterTable(context.Context, *apipb.AlterTableReq) error {
	return ErrReadOnly
}

func (s *sysRelation) TableColumns(_ context.Context) ([]*engine.Attribute, error) {
	colDefs := s.handle.GetMeta().(*catalog.TableEntry).GetLastestSchema().ColDefs
	cols, _ := ColDefsToAttrs(colDefs)
	return cols, nil
}
