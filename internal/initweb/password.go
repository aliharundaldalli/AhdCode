package initweb

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"ahdcode/internal/backend/golang/ahdruntime"
)

func lineReader(input io.Reader) *bufio.Reader {
	if reader, ok := input.(*bufio.Reader); ok {
		return reader
	}
	return bufio.NewReader(input)
}

func readLine(input io.Reader) (string, error) {
	reader := lineReader(input)
	line, err := reader.ReadString('\n')
	if err != nil && !(err == io.EOF && line != "") {
		if err == io.EOF {
			return "", fmt.Errorf("input is required")
		}
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func readSecret(input io.Reader, output io.Writer) (string, error) {
	if file, ok := input.(*os.File); ok && isTerminalFile(file) {
		restore, err := disableEcho(int(file.Fd()))
		if err == nil {
			defer restore()
			defer fmt.Fprintln(output)
		}
	}
	return readLine(input)
}

func hashPassword(password string) (hash string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if signal, ok := recovered.(*ahdruntime.AhdSignal); ok {
				err = fmt.Errorf("%s", signal.Message)
				return
			}
			panic(recovered)
		}
	}()
	hash = ahdruntime.AhdSecurityPasswordHash(ahdruntime.AhdClassSecurityError, password)
	return hash, nil
}

func verifyPassword(password, encoded string) (ok bool, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if signal, ok := recovered.(*ahdruntime.AhdSignal); ok {
				err = fmt.Errorf("%s", signal.Message)
				return
			}
			panic(recovered)
		}
	}()
	ok = ahdruntime.AhdSecurityPasswordVerify(ahdruntime.AhdClassSecurityError, password, encoded)
	return ok, nil
}
