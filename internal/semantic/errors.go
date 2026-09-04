package semantic

const (
	codeUnknownName         = "SEM001"
	codeRedeclaration       = "SEM002"
	codeInvalidType         = "SEM003"
	codeTypeMismatch        = "SEM004"
	codeScopeModifier       = "SEM005"
	codeMissingLocal        = "SEM006"
	codeHiddenGlobal        = "SEM007"
	codeInvalidGlobal       = "SEM008"
	codeConstantAssignment  = "SEM009"
	codeConstantInitializer = "SEM010"
	codeNullableUse         = "SEM011"
	codeConditionType       = "SEM012"
	codeOperatorType        = "SEM013"
	codeInvalidTarget       = "SEM014"
	codeNotCallable         = "SEM015"
	codeCallArguments       = "SEM016"
	codeReturnType          = "SEM017"
	codeMissingReturn       = "SEM018"
	codeInvalidMember       = "SEM019"
	codeConstantRange       = "SEM020"
	codeControlContext      = "SEM021"
	codePendingFeature      = "SEM022"
	codeFunctionInference   = "SEM023"
	codeConflictingFunction = "SEM024"
	codeInvalidOverload     = "SEM025"
	codeNoMatchingOverload  = "SEM026"
	codeAmbiguousOverload   = "SEM027"
	CodeModuleNotFound      = "SEM028"
	CodeExportNotFound      = "SEM029"
	CodeConfidentialAccess  = "SEM030"
	CodeImportCollision     = "SEM031"
	CodeCircularDependency  = "SEM032"
	CodeFailedDependency    = "SEM033"
	CodeNamespaceMember     = "SEM034"
	codeInvalidPairKey      = "SEM035"
	codeDuplicatePairKey    = "SEM036"
	codeCollectionInference = "SEM037"
	codeNullNotAllowed      = "SEM038"
	codeCannotInferType     = "SEM039"
	codeProtocolSlot        = "SEM040"
	codeProtocolSignature   = "SEM041"
	codeInvalidLambda       = "SEM042"
	// The explicit lambda capture rules. A lambda reads an enclosing local
	// only when it lists that name, so each way of getting the list wrong has
	// its own diagnostic rather than a shared generic one.
	codeMissingCapture = "SEM043"
	codeUnknownCapture = "SEM044"
	codeInvalidCapture = "SEM045"
	// require(...) local source composition (v0.14). CodeRequireNotFound,
	// CodeRequireCycle, and CodeRequireInvalidPath are raised by the
	// require-resolution phase in package module, ahead of ordinary semantic
	// analysis; they live here because module already depends on this package
	// for its other cross-file diagnostic codes. CodeRequireNotDeclared is
	// raised from inside the analyzer itself, when an identifier or type
	// resolves to a symbol imported by a bring the current source file never
	// wrote itself.
	CodeRequireNotFound    = "SEM046"
	CodeRequireCycle       = "SEM047"
	CodeRequireInvalidPath = "SEM048"
	CodeRequireNotDeclared = "SEM049"
)
