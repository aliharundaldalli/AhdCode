package lsp

import (
	"strings"
	"testing"
)

const inspectionV270LSPSource = `bring Disk
bring Service
from Disk bring DiskInfo
from Service bring ServiceInfo
disk: DiskInfo := Disk.inspect(path: "/")
unit: ServiceInfo := Service.status(name: "nginx.service")
`

func TestCompletionOffersTheV270InspectionSurfaceOverTheWire(t *testing.T) {
	cases := []struct {
		prefix string
		labels []string
	}{
		{"Disk.", []string{"inspect", "DiskInfo", "DiskError"}},
		{"Service.", []string{"status", "ServiceInfo", "ServiceError"}},
		{"disk.", []string{"path", "totalBytes", "usedBytes", "freeBytes", "availableBytes", "usedPercent"}},
		{"unit.", []string{"name", "activeState", "subState", "running", "enabled"}},
	}
	for _, testCase := range cases {
		source := inspectionV270LSPSource + testCase.prefix
		items := completionAt(t, source, "file:///main.ahd", len(source))
		for _, label := range testCase.labels {
			if !hasCompletionLabel(items, label) {
				t.Fatalf("%s offers no %q; got %#v", testCase.prefix, label, items)
			}
		}
		for _, forbidden := range []string{"start", "stop", "restart", "enable", "disable"} {
			if testCase.prefix == "Service." && hasCompletionLabel(items, forbidden) {
				t.Fatalf("Service offers a mutating %q", forbidden)
			}
		}
	}
	source := "bring "
	items := completionAt(t, source, "file:///main.ahd", len(source))
	for _, module := range []string{"Disk", "Service"} {
		if !hasCompletionLabel(items, module) {
			t.Fatalf("bring offers no %s module", module)
		}
	}
}

func TestHoverAndSignatureHelpForV270InspectionOverTheWire(t *testing.T) {
	source := inspectionV270LSPSource + `Disk.inspect("/tmp")
Service.status("caddy.service")
percent := disk.usedPercent()
`
	hover := hoverAt(t, source, "file:///main.ahd", "Disk.inspect(path", len("Disk.")+1)
	if !strings.Contains(hover.Contents.Value, "path: String") || !strings.Contains(hover.Contents.Value, "DiskInfo") {
		t.Fatalf("Disk.inspect hover = %q", hover.Contents.Value)
	}
	help, found := signatureHelpAt(t, source, "file:///main.ahd", `Service.status("caddy.service")`, len("Service.status("))
	if !found || len(help.Signatures) == 0 || !strings.Contains(help.Signatures[0].Label, "name: String") {
		t.Fatalf("Service.status signature help = %+v, found=%v", help, found)
	}
	hover = hoverAt(t, source, "file:///main.ahd", "disk.usedPercent()", len("disk.")+1)
	if !strings.Contains(hover.Contents.Value, "usedPercent") || !strings.Contains(hover.Contents.Value, "Real") {
		t.Fatalf("usedPercent hover = %q", hover.Contents.Value)
	}
}
