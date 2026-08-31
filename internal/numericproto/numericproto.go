// Package numericproto defines the narrow JSON contract between the stdlib-only
// generated runtime and the bundled ahdnumeric Gonum helper.
package numericproto

type Request struct {
	Operation string      `json:"operation"`
	Matrix    [][]float64 `json:"matrix"`
	Vector    []float64   `json:"vector,omitempty"`
}

type Complex struct {
	Real float64 `json:"real"`
	Imag float64 `json:"imag"`
}

type Response struct {
	Error    string                 `json:"error,omitempty"`
	Scalar   *float64               `json:"scalar,omitempty"`
	Integer  *int                   `json:"integer,omitempty"`
	Vector   []float64              `json:"vector,omitempty"`
	Matrix   [][]float64            `json:"matrix,omitempty"`
	Matrices map[string][][]float64 `json:"matrices,omitempty"`
	Complex  []Complex              `json:"complex,omitempty"`
}
