package evaluator

import (
	"crypto/rand"
	"encoding/base64"
)

func (session *Session) identityBuiltin(name string, args []any) any {
	_ = args
	switch name {
	case "id":
		return session.identityID()
	}
	session.raise("Error", "unsupported Identity function "+name)
	return nil
}

func (session *Session) identityID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		session.raise("IdentityError", "Identity random generation failed")
	}
	return base64.RawURLEncoding.EncodeToString(raw[:])
}
