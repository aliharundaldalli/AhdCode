package module

import (
	"fmt"
	"strings"
	"testing"
)

func FuzzModuleGraphTerminates(f *testing.F) {
	f.Add([]byte{4, 0, 1, 1, 2, 2, 3})
	f.Add([]byte{3, 0, 1, 1, 2, 2, 0})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}
		count := int(data[0]%8) + 1
		lines := make([][]string, count)
		for index := 1; index+1 < len(data); index += 2 {
			from := int(data[index]) % count
			to := int(data[index+1]) % count
			lines[from] = append(lines[from], fmt.Sprintf("bring M%d", to))
		}
		sources := make(map[string]string, count)
		for index := range count {
			sources[fmt.Sprintf("/M%d.ahd", index)] = strings.Join(lines[index], "\n")
		}
		_, result := compileMemory(t, sources, "/M0.ahd")
		for id, module := range result.Modules {
			if module.AnalyzeCount > 1 {
				t.Fatalf("module %s analyzed %d times", id, module.AnalyzeCount)
			}
			if module.State == Resolving {
				t.Fatalf("module %s remained Resolving", id)
			}
		}
	})
}
