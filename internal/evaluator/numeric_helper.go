package evaluator

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"ahdcode/internal/numericproto"
)

func discoverNumericRuntime() (string, error) {
	name := "ahdnumeric"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	var candidates []string
	if custom := os.Getenv("AHDCODE_NUMERIC_RUNTIME"); custom != "" {
		candidates = append(candidates, custom, filepath.Join(custom, name))
	}
	if executable, err := os.Executable(); err == nil {
		bin := filepath.Dir(executable)
		candidates = append(candidates, filepath.Join(bin, name), filepath.Join(bin, "..", "libexec", "ahdcode", name))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
			return candidate, nil
		}
	}
	return "", errors.New("the Numeric helper (ahdnumeric) was not found; set AHDCODE_NUMERIC_RUNTIME or reinstall AhdCode with the bundled Numeric helper")
}

func (s *Session) numericHelper(operation string, matrix [][]float64, vector []float64) numericproto.Response {
	helper, err := discoverNumericRuntime()
	if err != nil {
		s.raise("NumericError", err.Error())
	}
	dir := filepath.Join(os.TempDir(), "ahdcode", "numeric")
	if err = os.MkdirAll(dir, 0o700); err != nil {
		s.raise("NumericError", "creating temporary directory: "+err.Error())
	}
	file, err := os.CreateTemp(dir, "request-*.json")
	if err != nil {
		s.raise("NumericError", "writing Numeric request: "+err.Error())
	}
	defer os.Remove(file.Name())
	if err = json.NewEncoder(file).Encode(numericproto.Request{Operation: operation, Matrix: matrix, Vector: vector}); err == nil {
		err = file.Close()
	} else {
		_ = file.Close()
	}
	if err != nil {
		s.raise("NumericError", "writing Numeric request: "+err.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	output, runErr := exec.CommandContext(ctx, helper, file.Name()).Output()
	var response numericproto.Response
	decodeErr := json.Unmarshal(output, &response)
	if response.Error != "" {
		s.raise("NumericError", response.Error)
	}
	if ctx.Err() != nil {
		s.raise("NumericError", "Numeric helper timed out")
	}
	if decodeErr != nil {
		s.raise("NumericError", "Numeric helper returned an invalid response")
	}
	if runErr != nil {
		s.raise("NumericError", "Numeric helper failed: "+runErr.Error())
	}
	return response
}

func (s *Session) numericHelperMatrix(response numericproto.Response) *Instance {
	if len(response.Matrix) == 0 {
		s.raise("NumericError", "Numeric helper omitted its Matrix result")
	}
	return s.numericMatrixValidated(response.Matrix)
}

func (s *Session) numericHelperMatrices(response numericproto.Response, keys ...string) any {
	rows := make([][][]float64, len(keys))
	for index, key := range keys {
		value, ok := response.Matrices[key]
		if !ok || len(value) == 0 {
			s.raise("NumericError", "Numeric helper omitted its "+key+" Matrix result")
		}
		rows[index] = value
	}
	return s.matrixPair(keys, rows)
}
