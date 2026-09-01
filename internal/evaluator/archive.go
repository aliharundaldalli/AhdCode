package evaluator

// The Archive standard module's REPL implementation. Archive.zip/tar/tarGzip
// are ordinary file-system writes (no external process, no offline-renderer
// staging concern), so unlike Latex.pdf/PDFDocument.save, these run for real
// in the interactive evaluator: it imports ahdruntime and calls the exact
// same AhdArchive* functions the native backend does, so the two paths can
// never diverge in behavior.

import (
	"ahdcode/internal/backend/golang/ahdruntime"
)

func (s *Session) archiveBuiltin(name string, args []any) any {
	switch name {
	case "zip", "tar", "tarGzip":
		output := args[0].(string)
		entries := s.archivePairFrom(args[1])
		switch name {
		case "zip":
			ahdruntime.AhdArchiveZip(output, entries)
		case "tar":
			ahdruntime.AhdArchiveTar(output, entries)
		case "tarGzip":
			ahdruntime.AhdArchiveTarGzip(output, entries)
		}
		return Nothing
	}
	s.raise("Error", "unsupported Archive function "+name)
	return nil
}

// archivePairFrom converts the evaluator's own Pair representation into the
// *AhdPair[string,string] shape ahdruntime's Archive functions expect,
// preserving insertion order -- the same order ahdArchiveEntries treats as
// the archive's deterministic entry order.
func (s *Session) archivePairFrom(value any) *ahdruntime.AhdPair[string, string] {
	pair := s.requirePair(value)
	keys := make([]string, len(pair.Keys))
	values := make([]string, len(pair.Keys))
	for index, key := range pair.Keys {
		text := key.(string)
		keys[index] = text
		values[index] = pair.Values[key].(string)
	}
	return ahdruntime.AhdBuildPair(keys, values)
}
