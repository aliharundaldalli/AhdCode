package module

import (
	"fmt"

	"ahdcode/internal/framework"
)

// frameworkIdentity resolves one bring target against AhdCode's bundled
// first-party AhdCode source modules.
//
// Two rules, and only two:
//
//   - An application file reaches a bundled module only when that module is
//     public. Today that is `bring Web` and nothing else, so the framework's
//     internal decomposition stays invisible to applications and is free to
//     change without breaking them.
//   - A bundled module reaches any bundled module. That is how the framework
//     is written in more than one file without exposing those files.
//
// The lookup deliberately sits after the built-in modules and before the
// filesystem resolver, which is the same precedence HTTP and HTML already
// have: a first-party module name is reserved, so a sibling Web.ahd in an
// application never silently shadows the framework.
func frameworkIdentity(importer SourceIdentity, name string) (SourceIdentity, bool) {
	if !framework.Has(name) {
		return SourceIdentity{}, false
	}
	if !importer.Framework && !framework.IsPublic(name) {
		return SourceIdentity{}, false
	}
	return SourceIdentity{
		ID:        ModuleID(framework.ModuleID(name)),
		Name:      name,
		Path:      framework.VirtualPath(name),
		Framework: true,
	}, true
}

// frameworkIsolationError explains why a bundled module's bring did not
// resolve. Framework sources never read the application's source tree, so a
// name they cannot resolve is a defect in the framework itself, not something
// the application can fix by adding a file.
func frameworkIsolationError(name string) error {
	return fmt.Errorf("bundled first-party module %s is not a built-in module or another bundled module", name)
}
