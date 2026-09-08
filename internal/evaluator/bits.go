package evaluator

import "math/bits"

// The Bits standard module's REPL implementation. It mirrors the native
// backend's ahdruntime/bits.go exactly: same semantics for every operation and
// the same range check on shift and rotate distances.

func (session *Session) bitsBuiltin(name string, args []any) any {
	switch name {
	case "bitAnd":
		return args[0].(int64) & args[1].(int64)
	case "bitOr":
		return args[0].(int64) | args[1].(int64)
	case "bitXor":
		return args[0].(int64) ^ args[1].(int64)
	case "bitNot":
		return ^args[0].(int64)
	case "shiftLeft":
		return args[0].(int64) << session.bitsDistance(args[1].(int64))
	case "shiftRight":
		return args[0].(int64) >> session.bitsDistance(args[1].(int64))
	case "shiftRightUnsigned":
		return int64(uint64(args[0].(int64)) >> session.bitsDistance(args[1].(int64)))
	case "rotateLeft":
		return int64(bits.RotateLeft64(uint64(args[0].(int64)), int(session.bitsDistance(args[1].(int64)))))
	case "rotateRight":
		return int64(bits.RotateLeft64(uint64(args[0].(int64)), -int(session.bitsDistance(args[1].(int64)))))
	case "count":
		return int64(bits.OnesCount64(uint64(args[0].(int64))))
	case "leadingZeros":
		return int64(bits.LeadingZeros64(uint64(args[0].(int64))))
	case "trailingZeros":
		return int64(bits.TrailingZeros64(uint64(args[0].(int64))))
	}
	session.raise("Error", "unsupported Bits function "+name)
	return nil
}

func (session *Session) bitsDistance(distance int64) uint {
	if distance < 0 || distance > 63 {
		session.raise("BitsError", "Bits shift distance must be between 0 and 63")
	}
	return uint(distance)
}
