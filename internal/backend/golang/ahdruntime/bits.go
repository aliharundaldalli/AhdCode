package ahdruntime

// AhdCode Bits standard module runtime.
//
// This file is compiled twice: once as part of the compiler and once as
// generated program source, so it depends only on the Go standard library.
//
// AhdCode has no bitwise operators in its grammar; these functions provide the
// same capability as named calls. Every operation is defined on the language's
// only integer type, signed 64-bit Int, and the semantics are stated exactly
// rather than inherited from a host platform:
//
//   - bitAnd/bitOr/bitXor/bitNot operate on the two's-complement bit pattern.
//     They are spelled with a prefix because and, or and not are the
//     language's own logical operators and cannot be used as call names.
//   - shiftLeft and shiftRight are arithmetic: shiftRight keeps the sign bit.
//   - shiftRightUnsigned fills with zeros, treating the value as unsigned.
//   - rotateLeft/rotateRight rotate within 64 bits; no bit is lost.
//
// A shift or rotate distance outside 0..63 raises instead of silently
// producing zero, because "shift by 64" is almost always a bug in the caller's
// arithmetic rather than a request for zero.

import "math/bits"

// AhdBitsAnd returns the bitwise AND of two Ints.
func AhdBitsAnd(left, right int64) int64 { return left & right }

// AhdBitsOr returns the bitwise OR of two Ints.
func AhdBitsOr(left, right int64) int64 { return left | right }

// AhdBitsXor returns the bitwise exclusive OR of two Ints.
func AhdBitsXor(left, right int64) int64 { return left ^ right }

// AhdBitsNot returns the bitwise complement of an Int.
func AhdBitsNot(value int64) int64 { return ^value }

// ahdBitsDistance validates a shift or rotate distance.
func ahdBitsDistance(errorClass *AhdClass, distance int64) uint {
	if distance < 0 || distance > 63 {
		AhdRaiseClass(errorClass, "Bits shift distance must be between 0 and 63")
	}
	return uint(distance)
}

// AhdBitsShiftLeft shifts value left, discarding bits shifted past bit 63.
func AhdBitsShiftLeft(errorClass *AhdClass, value, distance int64) int64 {
	return value << ahdBitsDistance(errorClass, distance)
}

// AhdBitsShiftRight shifts value right, preserving the sign bit
// (arithmetic shift): -8 shifted right by 1 is -4.
func AhdBitsShiftRight(errorClass *AhdClass, value, distance int64) int64 {
	return value >> ahdBitsDistance(errorClass, distance)
}

// AhdBitsShiftRightUnsigned shifts value right filling with zeros, treating
// the bit pattern as unsigned. This is the shift SHA-256 and similar
// algorithms specify.
func AhdBitsShiftRightUnsigned(errorClass *AhdClass, value, distance int64) int64 {
	return int64(uint64(value) >> ahdBitsDistance(errorClass, distance))
}

// AhdBitsRotateLeft rotates the 64-bit pattern left; bits leaving the top
// re-enter at the bottom.
func AhdBitsRotateLeft(errorClass *AhdClass, value, distance int64) int64 {
	return int64(bits.RotateLeft64(uint64(value), int(ahdBitsDistance(errorClass, distance))))
}

// AhdBitsRotateRight rotates the 64-bit pattern right.
func AhdBitsRotateRight(errorClass *AhdClass, value, distance int64) int64 {
	return int64(bits.RotateLeft64(uint64(value), -int(ahdBitsDistance(errorClass, distance))))
}

// AhdBitsCount returns how many bits are set to one.
func AhdBitsCount(value int64) int64 { return int64(bits.OnesCount64(uint64(value))) }

// AhdBitsLeadingZeros returns the number of leading zero bits (0..64).
func AhdBitsLeadingZeros(value int64) int64 { return int64(bits.LeadingZeros64(uint64(value))) }

// AhdBitsTrailingZeros returns the number of trailing zero bits (0..64).
func AhdBitsTrailingZeros(value int64) int64 { return int64(bits.TrailingZeros64(uint64(value))) }
