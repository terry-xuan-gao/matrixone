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

package preinsert

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/matrixorigin/matrixone/pkg/container/batch"
	"github.com/matrixorigin/matrixone/pkg/container/types"
	"github.com/matrixorigin/matrixone/pkg/container/vector"
	mock_frontend "github.com/matrixorigin/matrixone/pkg/frontend/test"
	"github.com/matrixorigin/matrixone/pkg/pb/plan"
	"github.com/matrixorigin/matrixone/pkg/testutil"
	"github.com/matrixorigin/matrixone/pkg/vm/engine"
	"github.com/stretchr/testify/require"
)

var (
	i64typ     = &plan.Type{Id: int32(types.T_int64)}
	varchartyp = &plan.Type{Id: int32(types.T_varchar)}
)

func TestPreInsertNormal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.TODO()
	txnOperator := mock_frontend.NewMockTxnOperator(ctrl)
	txnOperator.EXPECT().Commit(gomock.Any()).Return(nil).AnyTimes()
	txnOperator.EXPECT().Rollback(ctx).Return(nil).AnyTimes()

	txnClient := mock_frontend.NewMockTxnClient(ctrl)
	txnClient.EXPECT().New(gomock.Any(), gomock.Any()).Return(txnOperator, nil).AnyTimes()

	eng := mock_frontend.NewMockEngine(ctrl)
	eng.EXPECT().New(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	eng.EXPECT().Commit(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	eng.EXPECT().Rollback(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	eng.EXPECT().Hints().Return(engine.Hints{
		CommitOrRollbackTimeout: time.Second,
	}).AnyTimes()

	proc := testutil.NewProc()
	proc.TxnClient = txnClient
	proc.Ctx = ctx
	batch1 := &batch.Batch{
		Vecs: []*vector.Vector{
			testutil.MakeInt64Vector([]int64{1, 2, 0}, []uint64{3}),
			testutil.MakeScalarInt64(3, 3),
			testutil.MakeVarcharVector([]string{"a", "b", "c"}, nil),
			testutil.MakeScalarVarchar("d", 3),
			testutil.MakeScalarNull(types.T_int64, 3),
		},
		Zs: []int64{1, 1, 1},
	}
	argument1 := Argument{
		Eg:         eng,
		SchemaName: "testDb",

		TableDef: &plan.TableDef{
			Cols: []*plan.ColDef{
				{Name: "int64_column", Typ: i64typ},
				{Name: "scalar_int64", Typ: i64typ},
				{Name: "varchar_column", Typ: varchartyp},
				{Name: "scalar_varchar", Typ: varchartyp},
				{Name: "int64_column", Typ: i64typ},
			},
		},
	}
	proc.SetInputBatch(batch1)
	_, err := Call(0, proc, &argument1, false, false)
	require.NoError(t, err)
	{
		result := proc.InputBatch()
		// check attr names
		require.Equal(t, []string{"int64_column", "scalar_int64", "varchar_column", "scalar_varchar", "int64_column"}, result.Attrs)
		// check vector
		require.Equal(t, len(batch1.Vecs), len(result.Vecs))
		for i, vec := range result.Vecs {
			require.Equal(t, len(batch1.Zs), vec.Length(), fmt.Sprintf("column number: %d", i))
		}
	}
}

func TestPreInsertNullCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.TODO()
	txnOperator := mock_frontend.NewMockTxnOperator(ctrl)
	txnOperator.EXPECT().Commit(gomock.Any()).Return(nil).AnyTimes()
	txnOperator.EXPECT().Rollback(ctx).Return(nil).AnyTimes()

	txnClient := mock_frontend.NewMockTxnClient(ctrl)
	txnClient.EXPECT().New(gomock.Any(), gomock.Any()).Return(txnOperator, nil).AnyTimes()

	eng := mock_frontend.NewMockEngine(ctrl)
	eng.EXPECT().New(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	eng.EXPECT().Commit(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	eng.EXPECT().Rollback(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	eng.EXPECT().Hints().Return(engine.Hints{
		CommitOrRollbackTimeout: time.Second,
	}).AnyTimes()

	proc := testutil.NewProc()
	proc.TxnClient = txnClient
	proc.Ctx = ctx
	batch2 := &batch.Batch{
		Vecs: []*vector.Vector{
			testutil.MakeInt64Vector([]int64{1, 2, 0}, []uint64{2}),
		},
		Zs: []int64{1, 1, 1},
	}
	argument2 := Argument{
		Eg:         eng,
		SchemaName: "testDb",
		TableDef: &plan.TableDef{
			Cols: []*plan.ColDef{
				{Name: "int64_column_primary", Primary: true, Typ: i64typ,
					Default: &plan.Default{
						NullAbility: false,
					},
				},
			},
		},
	}
	proc.Reg.InputBatch = batch2
	_, err2 := Call(0, proc, &argument2, false, false)
	require.Error(t, err2, "should return error when insert null into primary key column")
}
