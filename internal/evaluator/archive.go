package evaluator

// The Archive standard module's REPL implementation. Archive.zip/tar/tarGzip
// are ordinary file-system writes (no external process, no offline-renderer
// staging concern), so unlike Latex.pdf/PDFDocument.save, these run for real
// in the interactive evaluator: it imports ahdruntime and calls the exact
// same ArchiveZip/Tar/TarGzip core the native backend's AhdArchiveZip/Tar/
// TarGzip wrappers call, so the two paths can never diverge in validation,
// ordering, or writing behavior.
//
// That shared core reports failure by returning a Go error rather than by
// panicking (unlike the AhdArchiveZip/Tar/TarGzip wrappers themselves, which
// exist only for generated native programs and raise via the panic-based
// ahdruntime.AhdRaiseClass -- a mechanism that requires the generated
// program's own error-class registration, which the REPL never runs). Here,
// the evaluator converts that returned error into its own catchable
// ArchiveError with s.raise, exactly like every other module's evaluator
// builtin does. It must not call the panicking AhdArchive* wrappers
// directly: doing so previously let a Go panic escape the evaluator's own
// error boundary instead of becoming a catchable AhdCode error.
import (
	"ahdcode/internal/backend/golang/ahdruntime"
)

func (s *Session) archiveBuiltin(name string, args []any) any {
	switch name {
	case "zip", "tar", "tarGzip":
		output := args[0].(string)
		entries := s.archivePairFrom(args[1])
		var err error
		switch name {
		case "zip":
			err = ahdruntime.ArchiveZip(output, entries)
		case "tar":
			err = ahdruntime.ArchiveTar(output, entries)
		case "tarGzip":
			err = ahdruntime.ArchiveTarGzip(output, entries)
		}
		if err != nil {
			s.raise("ArchiveError", err.Error())
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
