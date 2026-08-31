// Command ahdnumeric is the bundled Gonum-backed helper for advanced Numeric operations.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"

	"ahdcode/internal/numericproto"
	"gonum.org/v1/gonum/mat"
)

func main() {
	response := run()
	_ = json.NewEncoder(os.Stdout).Encode(response)
	if response.Error != "" {
		os.Exit(1)
	}
}
func run() (response numericproto.Response) {
	defer func() {
		if recovered := recover(); recovered != nil {
			response = numericproto.Response{Error: fmt.Sprintf("numeric backend failure: %v", recovered)}
		}
	}()
	if len(os.Args) != 2 {
		return numericproto.Response{Error: "usage: ahdnumeric <request-file>"}
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		return failure(err)
	}
	var request numericproto.Request
	if err = json.Unmarshal(data, &request); err != nil {
		return failure(err)
	}
	return calculate(request)
}
func failure(err error) numericproto.Response {
	if err == nil {
		return numericproto.Response{}
	}
	return numericproto.Response{Error: err.Error()}
}
func dense(rows [][]float64) (*mat.Dense, error) {
	if len(rows) == 0 || len(rows[0]) == 0 {
		return nil, errors.New("matrix requires a non-empty rectangular shape")
	}
	r, c := len(rows), len(rows[0])
	data := make([]float64, 0, r*c)
	for _, row := range rows {
		if len(row) != c {
			return nil, errors.New("matrix rows must have equal lengths")
		}
		data = append(data, row...)
	}
	return mat.NewDense(r, c, data), nil
}
func grid(value mat.Matrix) [][]float64 {
	r, c := value.Dims()
	out := make([][]float64, r)
	for i := 0; i < r; i++ {
		out[i] = make([]float64, c)
		for j := 0; j < c; j++ {
			out[i][j] = value.At(i, j)
		}
	}
	return out
}
func square(a *mat.Dense) error {
	r, c := a.Dims()
	if r != c {
		return errors.New("operation requires a square matrix")
	}
	return nil
}
func calculate(req numericproto.Request) numericproto.Response {
	a, err := dense(req.Matrix)
	if err != nil {
		return failure(err)
	}
	r, c := a.Dims()
	switch req.Operation {
	case "determinant":
		if err = square(a); err != nil {
			return failure(err)
		}
		value := mat.Det(a)
		return numericproto.Response{Scalar: &value}
	case "inverse":
		if err = square(a); err != nil {
			return failure(err)
		}
		var inverse mat.Dense
		if err = inverse.Inverse(a); err != nil {
			return failure(errors.New("matrix is singular"))
		}
		return numericproto.Response{Matrix: grid(&inverse)}
	case "solve":
		if err = square(a); err != nil {
			return failure(err)
		}
		if len(req.Vector) != r {
			return failure(errors.New("system dimensions do not match"))
		}
		var result mat.VecDense
		if err = result.SolveVec(a, mat.NewVecDense(r, req.Vector)); err != nil {
			return failure(errors.New("system is singular or unsolvable"))
		}
		return numericproto.Response{Vector: append([]float64(nil), result.RawVector().Data...)}
	case "rank":
		var svd mat.SVD
		if !svd.Factorize(a, mat.SVDThin) {
			return failure(errors.New("SVD factorization failed"))
		}
		rank := svd.Rank(1e-12)
		return numericproto.Response{Integer: &rank}
	case "lu":
		if err = square(a); err != nil {
			return failure(err)
		}
		var lu mat.LU
		lu.Factorize(a)
		var l, u mat.TriDense
		lu.LTo(&l)
		lu.UTo(&u)
		pivots := lu.RowPivots(nil)
		p := mat.NewDense(r, r, nil)
		for i, j := range pivots {
			p.Set(i, j, 1)
		}
		return numericproto.Response{Matrices: map[string][][]float64{"P": grid(p), "L": grid(&l), "U": grid(&u)}}
	case "qr":
		var qr mat.QR
		qr.Factorize(a)
		var q, rr mat.Dense
		qr.QTo(&q)
		qr.RTo(&rr)
		return numericproto.Response{Matrices: map[string][][]float64{"Q": grid(&q), "R": grid(&rr)}}
	case "cholesky":
		if err = square(a); err != nil {
			return failure(err)
		}
		sym := mat.NewSymDense(r, nil)
		for i := 0; i < r; i++ {
			for j := 0; j <= i; j++ {
				if math.Abs(a.At(i, j)-a.At(j, i)) > 1e-12 {
					return failure(errors.New("Cholesky requires a symmetric matrix"))
				}
				sym.SetSym(i, j, a.At(i, j))
			}
		}
		var chol mat.Cholesky
		if !chol.Factorize(sym) {
			return failure(errors.New("matrix is not positive definite"))
		}
		var l mat.TriDense
		chol.LTo(&l)
		return numericproto.Response{Matrix: grid(&l)}
	case "svd":
		var svd mat.SVD
		if !svd.Factorize(a, mat.SVDFull) {
			return failure(errors.New("SVD factorization failed"))
		}
		var u, v mat.Dense
		svd.UTo(&u)
		svd.VTo(&v)
		values := svd.Values(nil)
		s := mat.NewDense(r, c, nil)
		for i, value := range values {
			s.Set(i, i, value)
		}
		return numericproto.Response{Matrices: map[string][][]float64{"U": grid(&u), "S": grid(s), "V": grid(&v)}}
	case "eigenvalues":
		if err = square(a); err != nil {
			return failure(err)
		}
		var eigen mat.Eigen
		if !eigen.Factorize(a, mat.EigenNone) {
			return failure(errors.New("eigenvalue factorization failed"))
		}
		values := eigen.Values(nil)
		result := make([]numericproto.Complex, len(values))
		for i, value := range values {
			result[i] = numericproto.Complex{Real: real(value), Imag: imag(value)}
		}
		return numericproto.Response{Complex: result}
	}
	_ = c
	return failure(errors.New("unknown numeric operation"))
}
