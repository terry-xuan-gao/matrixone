// Copyright 2021 - 2022 Matrix Origin
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
	"strconv"

	"github.com/matrixorigin/matrixone/pkg/common/moerr"
	"github.com/matrixorigin/matrixone/pkg/container/types"
	"github.com/matrixorigin/matrixone/pkg/container/vector"
	"github.com/matrixorigin/matrixone/pkg/vectorize/ceil"
	"github.com/matrixorigin/matrixone/pkg/vm/process"
)

func CeilUint64(vecs []*vector.Vector, proc *process.Process) (*vector.Vector, error) {
	return generalMathMulti("ceil", vecs, proc, ceil.CeilUint64)
}

func CeilInt64(vecs []*vector.Vector, proc *process.Process) (*vector.Vector, error) {
	return generalMathMulti("ceil", vecs, proc, ceil.CeilInt64)
}

// Parse string to float instead of int.
func CeilStr(vecs []*vector.Vector, proc *process.Process) (*vector.Vector, error) {
	values := vector.MustStrCol(vecs[0])
	floatvector, err := proc.AllocVectorOfRows(types.T_float64.ToType(), 0, nil)
	if err != nil {
		return nil, err
	}
	for _, v := range values {
		floatVal, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, err
		}
		if err := vector.AppendFixed(floatvector, floatVal, false, proc.Mp()); err != nil {
			floatvector.Free(proc.Mp())
			return nil, err
		}
	}
	newvecs := make([]*vector.Vector, 0)
	newvecs = append(newvecs, floatvector)
	return CeilFloat64(newvecs, proc)
}

func CeilFloat64(vecs []*vector.Vector, proc *process.Process) (*vector.Vector, error) {
	return generalMathMulti("ceil", vecs, proc, ceil.CeilFloat64)
}

func CeilDecimal64(vecs []*vector.Vector, proc *process.Process) (*vector.Vector, error) {
	scale := vecs[0].GetType().Scale
	if len(vecs) > 1 {
		digit := vector.MustFixedCol[int64](vecs[1])
		if len(digit) > 0 && int32(digit[0]) <= scale-18 {
			return nil, moerr.NewOutOfRange(proc.Ctx, "decimal64", "ceil(decimal64(18,%v),%v)", scale, digit[0])
		}
	}
	cb := func(vs []types.Decimal64, rs []types.Decimal64, digits int64) []types.Decimal64 {
		return ceil.CeilDecimal64(vs, rs, digits, scale)
	}
	return generalMathMulti("ceil", vecs, proc, cb)
}

func CeilDecimal128(vecs []*vector.Vector, proc *process.Process) (*vector.Vector, error) {
	scale := vecs[0].GetType().Scale
	if len(vecs) > 1 {
		digit := vector.MustFixedCol[int64](vecs[1])
		if len(digit) > 0 && int32(digit[0]) <= scale-38 {
			return nil, moerr.NewOutOfRange(proc.Ctx, "decimal128", "ceil(decimal128(38,%v),%v)", scale, digit[0])
		}
	}
	cb := func(vs []types.Decimal128, rs []types.Decimal128, digits int64) []types.Decimal128 {
		return ceil.CeilDecimal128(vs, rs, digits, scale)
	}
	return generalMathMulti("ceil", vecs, proc, cb)
}
