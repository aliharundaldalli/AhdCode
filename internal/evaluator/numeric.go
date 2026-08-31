package evaluator

import (
	"math"
	"sort"

	"ahdcode/internal/ir"
)

const (
	evalVectorClass = ir.ClassID("builtin:Numeric::class::Vector")
	evalMatrixClass = ir.ClassID("builtin:Numeric::class::Matrix")
	evalVectorField = ir.FieldID("builtin:Numeric::class::Vector::field::values")
	evalMatrixField = ir.FieldID("builtin:Numeric::class::Matrix::field::rows")
)

func (s *Session) numericVector(values []float64) *Instance {
	items := make([]any, len(values))
	for i, v := range values {
		items[i] = v
	}
	return &Instance{Class: evalVectorClass, Fields: map[ir.FieldID]any{evalVectorField: &List{Items: items}}}
}
func (s *Session) numericMatrix(rows [][]float64) *Instance {
	items := make([]any, len(rows))
	for i, row := range rows {
		r := make([]any, len(row))
		for j, v := range row {
			r[j] = v
		}
		items[i] = &List{Items: r}
	}
	return &Instance{Class: evalMatrixClass, Fields: map[ir.FieldID]any{evalMatrixField: &List{Items: items}}}
}
func (s *Session) vectorValues(value any) []float64 {
	v := s.requireInstance(value)
	list := s.requireList(v.Fields[evalVectorField])
	out := make([]float64, len(list.Items))
	for i, x := range list.Items {
		out[i] = x.(float64)
	}
	return out
}
func (s *Session) matrixRows(value any) [][]float64 {
	m := s.requireInstance(value)
	list := s.requireList(m.Fields[evalMatrixField])
	out := make([][]float64, len(list.Items))
	width := -1
	for i, item := range list.Items {
		row := s.requireList(item)
		if width < 0 {
			width = len(row.Items)
		} else if width != len(row.Items) {
			s.raise("NumericError", "matrix rows must have equal lengths")
		}
		out[i] = make([]float64, len(row.Items))
		for j, x := range row.Items {
			out[i][j] = x.(float64)
		}
	}
	if len(out) == 0 || width <= 0 {
		s.raise("NumericError", "matrix requires a non-empty rectangular shape")
	}
	return out
}
func numericFloat(value any) float64 {
	if v, ok := value.(int64); ok {
		return float64(v)
	}
	return value.(float64)
}
func (s *Session) numericBuiltin(name string, args []any) any {
	if len(args) > 0 {
		if receiver, ok := args[0].(*Instance); ok && stringsHasNumericOperation(name) {
			return s.numericOperation(name, receiver, args[1:])
		}
	}
	switch name {
	case "vector":
		list := s.requireList(args[0])
		v := make([]float64, len(list.Items))
		for i, x := range list.Items {
			v[i] = numericFloat(x)
		}
		return s.numericVector(v)
	case "matrix":
		grid := s.requireList(args[0])
		rows := make([][]float64, len(grid.Items))
		for i, item := range grid.Items {
			row := s.requireList(item)
			rows[i] = make([]float64, len(row.Items))
			for j, x := range row.Items {
				rows[i][j] = numericFloat(x)
			}
		}
		return s.numericMatrixValidated(rows)
	case "zeros", "ones":
		if len(args) == 1 {
			n := s.numericSize(args[0].(int64))
			v := make([]float64, n)
			if name == "ones" {
				for i := range v {
					v[i] = 1
				}
			}
			return s.numericVector(v)
		}
		r, c := s.numericSize(args[0].(int64)), s.numericSize(args[1].(int64))
		rows := make([][]float64, r)
		for i := range rows {
			rows[i] = make([]float64, c)
			if name == "ones" {
				for j := range rows[i] {
					rows[i][j] = 1
				}
			}
		}
		return s.numericMatrixValidated(rows)
	case "identity":
		n := s.numericSize(args[0].(int64))
		rows := make([][]float64, n)
		for i := range rows {
			rows[i] = make([]float64, n)
			rows[i][i] = 1
		}
		return s.numericMatrixValidated(rows)
	case "linspace":
		count := s.numericSize(args[2].(int64))
		if count == 0 {
			s.raise("NumericError", "linspace count must be positive")
		}
		start, stop := numericFloat(args[0]), numericFloat(args[1])
		v := make([]float64, count)
		if count == 1 {
			v[0] = start
		} else {
			step := (stop - start) / float64(count-1)
			for i := range v {
				v[i] = start + float64(i)*step
			}
			v[count-1] = stop
		}
		return s.numericVector(v)
	}
	s.raise("Error", "unsupported Numeric operation "+name)
	return nil
}
func stringsHasNumericOperation(name string) bool {
	return len(name) > 7 && (name[:7] == "Vector." || name[:7] == "Matrix.")
}
func (s *Session) numericSize(n int64) int {
	if n < 0 || uint64(n) > uint64(^uint(0)>>1) {
		s.raise("NumericError", "size must be non-negative")
	}
	return int(n)
}
func (s *Session) numericMatrixValidated(rows [][]float64) *Instance {
	if len(rows) == 0 || len(rows[0]) == 0 {
		s.raise("NumericError", "matrix requires a non-empty rectangular shape")
	}
	w := len(rows[0])
	for _, r := range rows {
		if len(r) != w {
			s.raise("NumericError", "matrix rows must have equal lengths")
		}
	}
	return s.numericMatrix(rows)
}
func (s *Session) numericOperation(name string, receiver *Instance, args []any) any {
	if receiver.Class == evalVectorClass {
		v := s.vectorValues(receiver)
		op := name[7:]
		switch op {
		case "length":
			return int64(len(v))
		case "values":
			items := make([]any, len(v))
			for i, x := range v {
				items[i] = x
			}
			return &List{Items: items}
		case "add", "subtract":
			w := s.vectorValues(args[0])
			if len(v) != len(w) {
				s.raise("NumericError", "vector lengths do not match")
			}
			for i := range v {
				if op == "add" {
					v[i] += w[i]
				} else {
					v[i] -= w[i]
				}
			}
			return s.numericVector(v)
		case "scale":
			f := numericFloat(args[0])
			for i := range v {
				v[i] *= f
			}
			return s.numericVector(v)
		case "dot":
			w := s.vectorValues(args[0])
			if len(v) != len(w) {
				s.raise("NumericError", "vector lengths do not match")
			}
			sum := 0.0
			for i := range v {
				sum += v[i] * w[i]
			}
			return sum
		case "abs", "sqrt", "exp", "log":
			for i := range v {
				v[i] = s.numericElement(v[i], op)
			}
			return s.numericVector(v)
		case "sum", "min", "max":
			return s.numericReduction(v, op)
		}
	}
	a := s.matrixRows(receiver)
	op := name[7:]
	switch op {
	case "rowCount":
		return int64(len(a))
	case "columnCount":
		return int64(len(a[0]))
	case "rows":
		return s.requireInstance(s.numericMatrix(a)).Fields[evalMatrixField]
	case "transpose":
		return s.numericMatrix(transpose(a))
	case "add", "subtract":
		b := s.matrixRows(args[0])
		if len(a) != len(b) || len(a[0]) != len(b[0]) {
			s.raise("NumericError", "matrix shapes do not match")
		}
		for i := range a {
			for j := range a[i] {
				if op == "add" {
					a[i][j] += b[i][j]
				} else {
					a[i][j] -= b[i][j]
				}
			}
		}
		return s.numericMatrix(a)
	case "scale":
		f := numericFloat(args[0])
		for i := range a {
			for j := range a[i] {
				a[i][j] *= f
			}
		}
		return s.numericMatrix(a)
	case "matmul":
		return s.numericMatrix(s.matmul(a, s.matrixRows(args[0])))
	case "trace":
		s.requireSquare(a)
		sum := 0.0
		for i := range a {
			sum += a[i][i]
		}
		return sum
	case "determinant":
		s.requireSquare(a)
		response := s.numericHelper(op, a, nil)
		if response.Scalar == nil {
			s.raise("NumericError", "Numeric helper omitted its scalar result")
		}
		return *response.Scalar
	case "rank":
		response := s.numericHelper(op, a, nil)
		if response.Integer == nil {
			s.raise("NumericError", "Numeric helper omitted its Int result")
		}
		return int64(*response.Integer)
	case "inverse":
		s.requireSquare(a)
		return s.numericHelperMatrix(s.numericHelper(op, a, nil))
	case "solve":
		s.requireSquare(a)
		response := s.numericHelper(op, a, s.vectorValues(args[0]))
		if response.Vector == nil {
			s.raise("NumericError", "Numeric helper omitted its Vector result")
		}
		return s.numericVector(response.Vector)
	case "cholesky":
		s.requireSquare(a)
		return s.numericHelperMatrix(s.numericHelper(op, a, nil))
	case "qr":
		return s.numericHelperMatrices(s.numericHelper(op, a, nil), "Q", "R")
	case "lu":
		s.requireSquare(a)
		return s.numericHelperMatrices(s.numericHelper(op, a, nil), "P", "L", "U")
	case "svd":
		return s.numericHelperMatrices(s.numericHelper(op, a, nil), "U", "S", "V")
	case "eigenvalues":
		s.requireSquare(a)
		response := s.numericHelper(op, a, nil)
		if len(response.Complex) == 0 {
			s.raise("NumericError", "Numeric helper omitted its Complex result")
		}
		items := make([]any, len(response.Complex))
		for i, x := range response.Complex {
			items[i] = complex(x.Real, x.Imag)
		}
		return &List{Items: items}
	case "abs", "sqrt", "exp", "log":
		for i := range a {
			for j := range a[i] {
				a[i][j] = s.numericElement(a[i][j], op)
			}
		}
		return s.numericMatrix(a)
	case "sum", "min", "max":
		flat := []float64{}
		for _, r := range a {
			flat = append(flat, r...)
		}
		return s.numericReduction(flat, op)
	}
	s.raise("Error", "unsupported Numeric operation "+name)
	return nil
}
func (s *Session) numericElement(v float64, op string) float64 {
	switch op {
	case "abs":
		return math.Abs(v)
	case "sqrt":
		if v < 0 {
			s.raise("NumericError", "sqrt requires non-negative values")
		}
		return math.Sqrt(v)
	case "log":
		if v <= 0 {
			s.raise("NumericError", "log requires positive values")
		}
		return math.Log(v)
	default:
		return math.Exp(v)
	}
}
func (s *Session) numericReduction(v []float64, op string) float64 {
	if len(v) == 0 && op != "sum" {
		s.raise("NumericError", op+" requires a non-empty value")
	}
	result := 0.0
	if len(v) > 0 && op != "sum" {
		result = v[0]
	}
	for _, x := range v {
		if op == "sum" {
			result += x
		} else if op == "min" && x < result {
			result = x
		} else if op == "max" && x > result {
			result = x
		}
	}
	return result
}
func transpose(a [][]float64) [][]float64 {
	out := make([][]float64, len(a[0]))
	for j := range out {
		out[j] = make([]float64, len(a))
		for i := range a {
			out[j][i] = a[i][j]
		}
	}
	return out
}
func (s *Session) matmul(a, b [][]float64) [][]float64 {
	if len(a[0]) != len(b) {
		s.raise("NumericError", "matrix multiplication shapes do not match")
	}
	out := make([][]float64, len(a))
	for i := range out {
		out[i] = make([]float64, len(b[0]))
		for j := range out[i] {
			for k := range b {
				out[i][j] += a[i][k] * b[k][j]
			}
		}
	}
	return out
}
func cloneRows(a [][]float64) [][]float64 {
	out := make([][]float64, len(a))
	for i := range a {
		out[i] = append([]float64(nil), a[i]...)
	}
	return out
}
func eliminate(input [][]float64) (int, float64) {
	a := cloneRows(input)
	rank := 0
	det := 1.0
	swaps := 0
	for col := 0; col < len(a[0]) && rank < len(a); col++ {
		p := rank
		for i := rank + 1; i < len(a); i++ {
			if math.Abs(a[i][col]) > math.Abs(a[p][col]) {
				p = i
			}
		}
		if math.Abs(a[p][col]) < 1e-12 {
			if len(a) == len(a[0]) {
				det = 0
			}
			continue
		}
		if p != rank {
			a[p], a[rank] = a[rank], a[p]
			swaps++
		}
		pivot := a[rank][col]
		if len(a) == len(a[0]) {
			det *= pivot
		}
		for i := rank + 1; i < len(a); i++ {
			f := a[i][col] / pivot
			for j := col; j < len(a[i]); j++ {
				a[i][j] -= f * a[rank][j]
			}
		}
		rank++
	}
	if swaps%2 != 0 {
		det = -det
	}
	return rank, det
}
func (s *Session) requireSquare(a [][]float64) {
	if len(a) != len(a[0]) {
		s.raise("NumericError", "operation requires a square matrix")
	}
}
func (s *Session) solve(input [][]float64, b []float64) []float64 {
	a := cloneRows(input)
	s.requireSquare(a)
	n := len(a)
	if len(b) != n {
		s.raise("NumericError", "system dimensions do not match")
	}
	for i := range a {
		a[i] = append(a[i], b[i])
	}
	for col := 0; col < n; col++ {
		p := col
		for i := col + 1; i < n; i++ {
			if math.Abs(a[i][col]) > math.Abs(a[p][col]) {
				p = i
			}
		}
		if math.Abs(a[p][col]) < 1e-12 {
			s.raise("NumericError", "matrix is singular")
		}
		a[p], a[col] = a[col], a[p]
		v := a[col][col]
		for j := col; j <= n; j++ {
			a[col][j] /= v
		}
		for i := 0; i < n; i++ {
			if i == col {
				continue
			}
			f := a[i][col]
			for j := col; j <= n; j++ {
				a[i][j] -= f * a[col][j]
			}
		}
	}
	x := make([]float64, n)
	for i := range x {
		x[i] = a[i][n]
	}
	return x
}
func (s *Session) inverse(a [][]float64) [][]float64 {
	s.requireSquare(a)
	n := len(a)
	out := make([][]float64, n)
	for j := 0; j < n; j++ {
		b := make([]float64, n)
		b[j] = 1
		x := s.solve(a, b)
		for i := range out {
			if out[i] == nil {
				out[i] = make([]float64, n)
			}
			out[i][j] = x[i]
		}
	}
	return out
}
func (s *Session) cholesky(a [][]float64) [][]float64 {
	s.requireSquare(a)
	n := len(a)
	l := make([][]float64, n)
	for i := range l {
		l[i] = make([]float64, n)
		for j := 0; j <= i; j++ {
			sum := a[i][j]
			for k := 0; k < j; k++ {
				sum -= l[i][k] * l[j][k]
			}
			if i == j {
				if sum <= 0 {
					s.raise("NumericError", "matrix is not positive definite")
				}
				l[i][j] = math.Sqrt(sum)
			} else {
				l[i][j] = sum / l[j][j]
			}
		}
	}
	return l
}
func (s *Session) qr(a [][]float64) ([][]float64, [][]float64) {
	m, n := len(a), len(a[0])
	q := make([][]float64, m)
	for i := range q {
		q[i] = make([]float64, n)
	}
	r := make([][]float64, n)
	for i := range r {
		r[i] = make([]float64, n)
	}
	for j := 0; j < n; j++ {
		v := make([]float64, m)
		for i := range v {
			v[i] = a[i][j]
		}
		for k := 0; k < j; k++ {
			for i := range v {
				r[k][j] += q[i][k] * v[i]
			}
			for i := range v {
				v[i] -= r[k][j] * q[i][k]
			}
		}
		for _, x := range v {
			r[j][j] += x * x
		}
		r[j][j] = math.Sqrt(r[j][j])
		if r[j][j] < 1e-12 {
			s.raise("NumericError", "QR decomposition failed")
		}
		for i := range v {
			q[i][j] = v[i] / r[j][j]
		}
	}
	return q, r
}
func (s *Session) lu(a [][]float64) ([][]float64, [][]float64, [][]float64) {
	s.requireSquare(a)
	u := cloneRows(a)
	n := len(a)
	p := make([][]float64, n)
	l := make([][]float64, n)
	for i := range p {
		p[i] = make([]float64, n)
		p[i][i] = 1
		l[i] = make([]float64, n)
		l[i][i] = 1
	}
	for k := 0; k < n; k++ {
		pivot := k
		for i := k + 1; i < n; i++ {
			if math.Abs(u[i][k]) > math.Abs(u[pivot][k]) {
				pivot = i
			}
		}
		if math.Abs(u[pivot][k]) < 1e-12 {
			s.raise("NumericError", "LU decomposition failed")
		}
		if pivot != k {
			u[pivot], u[k] = u[k], u[pivot]
			p[pivot], p[k] = p[k], p[pivot]
			for j := 0; j < k; j++ {
				l[pivot][j], l[k][j] = l[k][j], l[pivot][j]
			}
		}
		for i := k + 1; i < n; i++ {
			l[i][k] = u[i][k] / u[k][k]
			for j := k; j < n; j++ {
				u[i][j] -= l[i][k] * u[k][j]
			}
		}
	}
	return p, l, u
}
func jacobi(a [][]float64) ([]float64, [][]float64) {
	a = cloneRows(a)
	n := len(a)
	v := make([][]float64, n)
	for i := range v {
		v[i] = make([]float64, n)
		v[i][i] = 1
	}
	for iter := 0; iter < 100*n*n; iter++ {
		p, q := 0, 0
		largest := 0.0
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				if math.Abs(a[i][j]) > largest {
					largest = math.Abs(a[i][j])
					p, q = i, j
				}
			}
		}
		if largest < 1e-12 {
			break
		}
		angle := .5 * math.Atan2(2*a[p][q], a[q][q]-a[p][p])
		c, ss := math.Cos(angle), math.Sin(angle)
		for i := 0; i < n; i++ {
			x, y := a[i][p], a[i][q]
			a[i][p], a[i][q] = c*x-ss*y, ss*x+c*y
		}
		for j := 0; j < n; j++ {
			x, y := a[p][j], a[q][j]
			a[p][j], a[q][j] = c*x-ss*y, ss*x+c*y
		}
		for i := 0; i < n; i++ {
			x, y := v[i][p], v[i][q]
			v[i][p], v[i][q] = c*x-ss*y, ss*x+c*y
		}
	}
	values := make([]float64, n)
	for i := range values {
		values[i] = a[i][i]
	}
	return values, v
}
func (s *Session) svd(a [][]float64) ([][]float64, [][]float64, [][]float64) {
	ata := s.matmul(transpose(a), a)
	values, v := jacobi(ata)
	order := make([]int, len(values))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool { return values[order[i]] > values[order[j]] })
	vv := make([][]float64, len(v))
	sv := make([][]float64, len(v))
	u := make([][]float64, len(a))
	for i := range vv {
		vv[i] = make([]float64, len(v))
		sv[i] = make([]float64, len(v))
	}
	for i := range u {
		u[i] = make([]float64, len(v))
	}
	for j, old := range order {
		sigma := math.Sqrt(math.Max(0, values[old]))
		sv[j][j] = sigma
		for i := range v {
			vv[i][j] = v[i][old]
		}
		if sigma > 1e-12 {
			for i := range a {
				for k := range v {
					u[i][j] += a[i][k] * vv[k][j]
				}
				u[i][j] /= sigma
			}
		}
	}
	return u, sv, vv
}
func (s *Session) eigen(a [][]float64) []complex128 {
	s.requireSquare(a)
	n := len(a)
	if n == 1 {
		return []complex128{complex(a[0][0], 0)}
	}
	if n == 2 {
		tr := a[0][0] + a[1][1]
		det := a[0][0]*a[1][1] - a[0][1]*a[1][0]
		d := tr*tr - 4*det
		if d >= 0 {
			r := math.Sqrt(d)
			return []complex128{complex((tr+r)/2, 0), complex((tr-r)/2, 0)}
		}
		r := math.Sqrt(-d) / 2
		return []complex128{complex(tr/2, r), complex(tr/2, -r)}
	}
	for i := 0; i < 500; i++ {
		q, r := s.qr(a)
		a = s.matmul(r, q)
	}
	out := make([]complex128, n)
	for i := range out {
		out[i] = complex(a[i][i], 0)
	}
	return out
}
func (s *Session) matrixPair(keys []string, grids [][][]float64) *Pair {
	pair := &Pair{Values: map[any]any{}}
	for i, key := range keys {
		pairSet(pair, key, s.numericMatrix(grids[i]))
	}
	return pair
}
