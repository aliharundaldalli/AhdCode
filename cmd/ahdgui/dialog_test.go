package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func dialogReply(t *testing.T, line string) response {
	t.Helper()
	first, err := readRequest(bufio.NewReader(strings.NewReader(line + "\n")))
	if err != nil {
		t.Fatal(err)
	}
	return runDialog(first)
}

func TestDialogValidationAndScriptedAnswers(t *testing.T) {
	absolute := filepath.Join(t.TempDir(), "report")
	for _, testCase := range []struct {
		line   string
		check  func(response) bool
		reason string
	}{
		{`{"op":"dialog","version":3,"kind":"openFile","headless":true,"answer":{"cancel":true}}`,
			func(r response) bool { return r.OK && r.Cancelled && len(r.Paths) == 0 }, "cancel is not an error"},
		{`{"op":"dialog","version":3,"kind":"openFile","headless":true,"answer":{"paths":["` + absolute + `"]}}`,
			func(r response) bool { return r.OK && len(r.Paths) == 1 && r.Paths[0] == absolute }, "one file"},
		{`{"op":"dialog","version":3,"kind":"openFiles","headless":true,"answer":{"paths":["` + absolute + `","` + absolute + `2"]}}`,
			func(r response) bool { return r.OK && len(r.Paths) == 2 }, "several files"},
		{`{"op":"dialog","version":3,"kind":"openFile","headless":true,"answer":{"paths":["a","b"]}}`,
			func(r response) bool { return !r.OK }, "two paths for one file"},
		{`{"op":"dialog","version":3,"kind":"selectFolder","headless":true,"answer":{"paths":["relative/folder"]}}`,
			func(r response) bool { return !r.OK && strings.Contains(r.Error, "invalid path") }, "relative path"},
		{`{"op":"dialog","version":3,"kind":"saveFile","name":"out.csv","extensions":["csv"],"headless":true,"answer":{"paths":["` + absolute + `"]}}`,
			func(r response) bool { return r.OK && r.Paths[0] == absolute+".csv" }, "save adds the extension"},
		{`{"op":"dialog","version":3,"kind":"saveFile","extensions":["CSV"],"headless":true,"answer":{"paths":["` + absolute + `.csv"]}}`,
			func(r response) bool { return r.OK && r.Paths[0] == absolute+".csv" }, "extension matches without case"},
		{`{"op":"dialog","version":3,"kind":"confirm","text":"Delete?","headless":true,"answer":{"confirm":true}}`,
			func(r response) bool { return r.OK && r.Confirmed != nil && *r.Confirmed }, "confirm yes"},
		{`{"op":"dialog","version":3,"kind":"confirm","text":"Delete?","headless":true,"answer":{}}`,
			func(r response) bool { return r.OK && r.Confirmed != nil && !*r.Confirmed }, "confirm no"},
		{`{"op":"dialog","version":3,"kind":"message","text":"Saved","headless":true,"answer":{}}`,
			func(r response) bool { return r.OK && r.Confirmed != nil }, "message"},
		{`{"op":"dialog","version":3,"kind":"openFile","headless":true,"answer":{"fail":"no display"}}`,
			func(r response) bool { return !r.OK && r.Error == "no display" }, "backend failure"},
		{`{"op":"dialog","version":3,"kind":"openFile","headless":true}`,
			func(r response) bool { return !r.OK }, "no scripted answer"},
		{`{"op":"dialog","version":3,"kind":"openFile","answer":{"cancel":true}}`,
			func(r response) bool { return !r.OK }, "an answer outside headless mode"},
		{`{"op":"dialog","version":2,"kind":"openFile","headless":true,"answer":{"cancel":true}}`,
			func(r response) bool { return !r.OK }, "old protocol"},
		{`{"op":"dialog","version":3,"kind":"print","headless":true,"answer":{"cancel":true}}`,
			func(r response) bool { return !r.OK }, "unknown kind"},
		{`{"op":"dialog","version":3,"kind":"saveFile","name":"../x","headless":true,"answer":{"cancel":true}}`,
			func(r response) bool { return !r.OK }, "a suggested name with a folder"},
		{`{"op":"dialog","version":3,"kind":"openFile","extensions":["*.png"],"headless":true,"answer":{"cancel":true}}`,
			func(r response) bool { return !r.OK }, "a wildcard extension"},
	} {
		if reply := dialogReply(t, testCase.line); !testCase.check(reply) {
			t.Errorf("%s: %+v", testCase.reason, reply)
		}
	}
}

func TestFallbackDialogNavigatesAndChooses(t *testing.T) {
	root := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(root, "data"), 0o755))
	must(os.WriteFile(filepath.Join(root, "data", "sales.csv"), []byte("a"), 0o600))
	must(os.WriteFile(filepath.Join(root, "data", "photo.png"), []byte("a"), 0o600))
	must(os.WriteFile(filepath.Join(root, ".hidden"), []byte("a"), 0o600))
	must(os.WriteFile(filepath.Join(root, "notes.txt"), []byte("a"), 0o600))

	c, err := newDialogController(dialogRequest{kind: dialogOpenFile, extensions: []string{"csv"}}, root)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(c.entries, ",") != "data/" {
		t.Fatalf("filtered listing %v", c.entries)
	}
	_ = c.session.model.selectIndex(c.list, 0)
	c.handle(event{Event: "click", Widget: c.open})
	if c.folder != filepath.Join(root, "data") || strings.Join(c.entries, ",") != "sales.csv" {
		t.Fatalf("entered %s: %v", c.folder, c.entries)
	}
	_ = c.session.model.selectIndex(c.list, 0)
	c.handle(event{Event: "click", Widget: c.open})
	if !c.done || c.result.paths[0] != filepath.Join(root, "data", "sales.csv") {
		t.Fatalf("result %+v", c.result)
	}

	save, _ := newDialogController(dialogRequest{kind: dialogSaveFile, name: "report.csv"}, root)
	save.handle(event{Event: "click", Widget: save.up})
	save.handle(event{Event: "click", Widget: save.cancel})
	if !save.result.cancelled {
		t.Fatal("cancel")
	}
	save, _ = newDialogController(dialogRequest{kind: dialogSaveFile, name: "report.csv"}, root)
	save.handle(event{Event: "click", Widget: save.choose})
	if save.result.paths[0] != filepath.Join(root, "report.csv") {
		t.Fatalf("save %+v", save.result)
	}

	folder, _ := newDialogController(dialogRequest{kind: dialogSelectFolder}, root)
	if strings.Join(folder.entries, ",") != "data/" {
		t.Fatalf("a folder dialog listed files: %v", folder.entries)
	}
	folder.handle(event{Event: "click", Widget: folder.choose})
	if folder.result.paths[0] != root {
		t.Fatalf("folder %+v", folder.result)
	}

	confirm, _ := newDialogController(dialogRequest{kind: dialogConfirm, text: "Delete the row?\nThis cannot be undone."}, root)
	confirm.handle(event{Event: "click", Widget: confirm.cancel})
	if confirm.result.confirmed {
		t.Fatal("Cancel confirmed")
	}
	confirm, _ = newDialogController(dialogRequest{kind: dialogConfirm, text: "Delete?"}, root)
	confirm.handle(event{Event: "click", Widget: confirm.choose})
	if !confirm.result.confirmed {
		t.Fatal("OK did not confirm")
	}
	if reply := dialogResponse(dialogRequest{kind: dialogOpenFile}, dialogResult{cancelled: true}); !reply.OK || !reply.Cancelled {
		t.Fatal("closing the fallback window cancels")
	}
}
