package ahdruntime

import "testing"

func BenchmarkQRMatrixURL(b *testing.B) {
	for index := 0; index < b.N; index++ {
		if _, problem := AhdQRMatrixText("bench", "https://ahdcode.org/verify?id=AHD-2026-0042", "M"); problem != "" {
			b.Fatal(problem)
		}
	}
}

func BenchmarkQRMatrixLongestAtLevelL(b *testing.B) {
	letters := make([]byte, 2900)
	for index := range letters {
		letters[index] = 'a' + byte(index%26)
	}
	for index := 0; index < b.N; index++ {
		if _, problem := AhdQRMatrixText("bench", string(letters), "L"); problem != "" {
			b.Fatal(problem)
		}
	}
}

func BenchmarkQRPNG1024(b *testing.B) {
	matrix, problem := AhdQRMatrixText("bench", "https://ahdcode.org", "Q")
	if problem != "" {
		b.Fatal(problem)
	}
	for index := 0; index < b.N; index++ {
		if _, problem := AhdQRPNGBytes("bench", matrix, 1024); problem != "" {
			b.Fatal(problem)
		}
	}
}

func BenchmarkBarcodeCode128Pattern(b *testing.B) {
	for index := 0; index < b.N; index++ {
		if _, _, problem := AhdBarcodePatternText("bench", "Code128", "AHD-2026-0042-WAREHOUSE-7"); problem != "" {
			b.Fatal(problem)
		}
	}
}

func BenchmarkLatexQRVectorFragment(b *testing.B) {
	for index := 0; index < b.N; index++ {
		if _, problem := AhdLatexQRText("bench", "https://ahdcode.org/verify?id=42", 3, "H"); problem != "" {
			b.Fatal(problem)
		}
	}
}

func BenchmarkSVGConvertGeneratedQR(b *testing.B) {
	matrix, problem := AhdQRMatrixText("bench", "https://ahdcode.org/verify?id=AHD-2026-0042", "H")
	if problem != "" {
		b.Fatal(problem)
	}
	svg, problem := AhdQRSVGBytes("bench", matrix, 5)
	if problem != "" {
		b.Fatal(problem)
	}
	b.SetBytes(int64(len(svg)))
	for index := 0; index < b.N; index++ {
		if _, problem := AhdSVGConvert("bench", svg); problem != "" {
			b.Fatal(problem)
		}
	}
}
