package semantic

import (
	"sort"

	"ahdcode/internal/types"
)

const securityModuleID = "builtin:Security"

var securityErrorParent = &types.ClassSymbol{ModuleID: "builtin:core", Name: "Error",
	Parent: &types.ClassSymbol{ModuleID: "builtin:core", Name: "Object"}}
var securityErrorClass = &types.ClassSymbol{
	ModuleID: securityModuleID, Name: "SecurityError",
	Parent: securityErrorParent,
}

// SecurityErrorIdentity exposes the canonical identity to the lowering layer
// without coupling the public module interface to a backend.
func SecurityErrorIdentity() *types.ClassSymbol { return securityErrorClass }

func securityModuleInterface() *ModuleInterface {
	module := standardInterface(securityModuleID, "Security")
	errorSymbol := &Symbol{
		Name: "SecurityError", Kind: ClassSymbol, Class: securityErrorClass,
		Type: types.Class{Symbol: securityErrorClass, Reference: true}, ModuleRoot: true,
		Builtin: true, InitialNull: NonNull, OriginModuleID: securityModuleID,
		Members: make(map[string]*Symbol), Constructor: builtinErrorConstructor(),
	}
	module.Classes[securityModuleID+"\x00SecurityError"] = errorSymbol
	addStandardExport(module, errorSymbol)

	name := func(label string) types.Parameter { return types.Parameter{Name: label, Type: types.String} }

	// Security.passwordHash(password: String) -> String
	addStandardExport(module, standardFunction(securityModuleID, "passwordHash", types.String, name("password")))
	// Security.passwordVerify(password: String, encodedHash: String) -> Bool
	addStandardExport(module, standardFunction(securityModuleID, "passwordVerify", types.Bool, name("password"), name("encodedHash")))
	// Security.token() -> String
	addStandardExport(module, standardFunction(securityModuleID, "token", types.String))
	// Security.secureEqual(expected: String, received: String) -> Bool
	addStandardExport(module, standardFunction(securityModuleID, "secureEqual", types.Bool, name("expected"), name("received")))

	// Digests and message authentication.
	addStandardExport(module, standardFunction(securityModuleID, "sha256", types.String, name("text")))
	addStandardExport(module, standardFunction(securityModuleID, "sha512", types.String, name("text")))
	addStandardExport(module, standardFunction(securityModuleID, "hmacSHA256", types.String, name("key"), name("message")))
	addStandardExport(module, standardFunction(securityModuleID, "hmacVerify", types.Bool, name("key"), name("message"), name("receivedHex")))

	// Encodings.
	addStandardExport(module, standardFunction(securityModuleID, "base64Encode", types.String, name("text")))
	addStandardExport(module, standardFunction(securityModuleID, "base64Decode", types.String, name("encoded")))
	addStandardExport(module, standardFunction(securityModuleID, "base64UrlEncode", types.String, name("text")))
	addStandardExport(module, standardFunction(securityModuleID, "base64UrlDecode", types.String, name("encoded")))
	addStandardExport(module, standardFunction(securityModuleID, "hexEncode", types.String, name("text")))
	addStandardExport(module, standardFunction(securityModuleID, "hexDecode", types.String, name("encoded")))

	// Random material.
	addStandardExport(module, standardFunction(securityModuleID, "randomHex", types.String, types.Parameter{Name: "count", Type: types.Int}))

	// RSA signatures (RS256) and authenticated symmetric encryption.
	addStandardExport(module, standardFunction(securityModuleID, "rsaSignSHA256", types.String, name("privateKeyPem"), name("message")))
	addStandardExport(module, standardFunction(securityModuleID, "rsaVerifySHA256", types.Bool, name("publicKeyPem"), name("message"), name("signature")))
	addStandardExport(module, standardFunction(securityModuleID, "aesEncrypt", types.String, name("keyHex"), name("plaintext")))
	addStandardExport(module, standardFunction(securityModuleID, "aesDecrypt", types.String, name("keyHex"), name("payload")))

	sort.Strings(module.ExportNames)
	return module
}
