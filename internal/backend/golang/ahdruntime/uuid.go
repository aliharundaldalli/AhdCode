package ahdruntime

// UUID (v1.4.0) generates, parses, and compares RFC 9562 UUIDs.
//
// A UUIDValue is carried as its canonical lowercase 36-character text, which
// every operation below validates before use. Only the Go standard library is
// involved -- crypto/rand for random bits and time for the version 7 clock --
// so the generated workspace stays dependency-free and a source build keeps
// working on the Go version go.mod names (Go 1.27's own uuid package is not
// used, and its parser accepts forms AhdCode rejects).
//
// UUIDs are identifiers, never secrets. Security.token() is for credentials,
// and Identity.id() remains the opaque public identifier Web starters use.

import (
	"bytes"
	"crypto/rand"
	"sync"
	"time"
)

// The interactive evaluator raises UUIDError through this runtime, so the class
// needs a constructor in the compiler process too. A generated program raises
// through its own generated descriptor instead.
func init() {
	AhdRegisterError(AhdClassUUIDError, func(message string) AhdInstance {
		instance := &ahdModuleError{message: message}
		instance.AhdSetClass(AhdClassUUIDError)
		return instance
	})
}

const (
	ahdUUIDZeroText       = "00000000-0000-0000-0000-000000000000"
	ahdUUIDTextMessage    = "UUID text must be 36 characters in the form xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
	ahdUUIDRandomMessage  = "UUID random generation failed"
	ahdUUIDClockMessage   = "UUID v7 requires a system clock between 1970 and the year 10889"
	ahdUUIDStorageMessage = "UUIDValue storage is corrupted"
	// ahdUUIDMillisecondLimit is the first Unix millisecond that no longer fits
	// the 48-bit unix_ts_ms field of a version 7 UUID (the year 10889).
	ahdUUIDMillisecondLimit = int64(1) << 48
	ahdUUIDHexDigits        = "0123456789abcdef"
)

// ahdUUIDFormat renders 128 bits in the canonical lowercase hex-and-dash form.
func ahdUUIDFormat(raw [16]byte) string {
	var text [36]byte
	position := 0
	for index, octet := range raw {
		if index == 4 || index == 6 || index == 8 || index == 10 {
			text[position] = '-'
			position++
		}
		text[position] = ahdUUIDHexDigits[octet>>4]
		text[position+1] = ahdUUIDHexDigits[octet&0x0f]
		position += 2
	}
	return string(text[:])
}

func ahdUUIDHexValue(character byte) (byte, bool) {
	switch {
	case character >= '0' && character <= '9':
		return character - '0', true
	case character >= 'a' && character <= 'f':
		return character - 'a' + 10, true
	case character >= 'A' && character <= 'F':
		return character - 'A' + 10, true
	}
	return 0, false
}

// ahdUUIDDecode accepts exactly the RFC 9562 hex-and-dash form: 36 characters,
// hyphens at offsets 8, 13, 18, and 23, hexadecimal digits of either case
// everywhere else. Braces, a urn:uuid: prefix, whitespace, and the 32-digit
// form are rejected. Every 128-bit value is accepted, whatever its version or
// variant.
func ahdUUIDDecode(text string) ([16]byte, bool) {
	var raw [16]byte
	if len(text) != 36 {
		return raw, false
	}
	octet := 0
	for index := 0; index < len(text); {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if text[index] != '-' {
				return raw, false
			}
			index++
			continue
		}
		high, highValid := ahdUUIDHexValue(text[index])
		low, lowValid := ahdUUIDHexValue(text[index+1])
		if !highValid || !lowValid {
			return raw, false
		}
		raw[octet] = high<<4 | low
		octet++
		index += 2
	}
	return raw, true
}

// ahdUUIDClock issues the 60-bit version 7 clock values of one process: 48 bits
// of Unix milliseconds followed by a 12-bit sub-millisecond fraction (RFC 9562
// section 6.2, method 3). Each value is strictly greater than the one issued
// before it, so the UUIDs one process generates sort in generation order even
// when the wall clock steps backwards; the embedded time then continues from
// the last issued value until the clock catches up (section 6.2 permits this).
type ahdUUIDClock struct {
	mutex  sync.Mutex
	last   uint64
	issued bool
}

var ahdUUIDProcessClock ahdUUIDClock

func (clock *ahdUUIDClock) next(now time.Time) (uint64, bool) {
	milliseconds := now.UnixMilli()
	if milliseconds < 0 || milliseconds >= ahdUUIDMillisecondLimit {
		return 0, false
	}
	fraction := uint64(now.Nanosecond()%1_000_000) * 4096 / 1_000_000
	value := uint64(milliseconds)<<12 | fraction
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	if clock.issued && value <= clock.last {
		value = clock.last + 1
	}
	if value >= uint64(ahdUUIDMillisecondLimit)<<12 {
		return 0, false
	}
	clock.last = value
	clock.issued = true
	return value, true
}

// AhdUUIDV4 returns a version 4 UUID: 122 bits from crypto/rand.
func AhdUUIDV4(class *AhdClass) string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		AhdRaiseClass(class, ahdUUIDRandomMessage)
	}
	raw[6] = raw[6]&0x0f | 0x40
	raw[8] = raw[8]&0x3f | 0x80
	return ahdUUIDFormat(raw)
}

// AhdUUIDV7 returns a version 7 UUID from the process clock.
func AhdUUIDV7(class *AhdClass) string {
	return ahdUUIDV7At(class, &ahdUUIDProcessClock, time.Now())
}

// ahdUUIDV7At lays out RFC 9562 section 5.7: unix_ts_ms (48 bits), ver (4),
// rand_a (12, the sub-millisecond fraction), var (2), rand_b (62 random bits).
func ahdUUIDV7At(class *AhdClass, clock *ahdUUIDClock, now time.Time) string {
	value, valid := clock.next(now)
	if !valid {
		AhdRaiseClass(class, ahdUUIDClockMessage)
	}
	var raw [16]byte
	if _, err := rand.Read(raw[8:]); err != nil {
		AhdRaiseClass(class, ahdUUIDRandomMessage)
	}
	milliseconds := value >> 12
	for index := 0; index < 6; index++ {
		raw[index] = byte(milliseconds >> (40 - 8*uint(index)))
	}
	raw[6] = 0x70 | byte(value>>8)&0x0f
	raw[7] = byte(value)
	raw[8] = raw[8]&0x3f | 0x80
	return ahdUUIDFormat(raw)
}

// AhdUUIDParse validates text and returns its canonical lowercase form. The
// message never echoes the rejected text.
func AhdUUIDParse(class *AhdClass, text string) string {
	raw, valid := ahdUUIDDecode(text)
	if !valid {
		AhdRaiseClass(class, ahdUUIDTextMessage)
	}
	return ahdUUIDFormat(raw)
}

// AhdUUIDIsValid reports whether UUID.parse would accept text. It never raises.
func AhdUUIDIsValid(text string) bool {
	_, valid := ahdUUIDDecode(text)
	return valid
}

// AhdUUIDZero returns the all-zero UUID (RFC 9562 calls it the Nil UUID).
func AhdUUIDZero() string {
	return ahdUUIDZeroText
}

// ahdUUIDStored decodes a UUIDValue's hidden canonical text.
func ahdUUIDStored(class *AhdClass, data string) [16]byte {
	raw, valid := ahdUUIDDecode(data)
	if !valid || ahdUUIDFormat(raw) != data {
		AhdRaiseClass(class, ahdUUIDStorageMessage)
	}
	return raw
}

func AhdUUIDString(class *AhdClass, data string) string {
	ahdUUIDStored(class, data)
	return data
}

// AhdUUIDVersion is the 4-bit version field as written, 0 through 15.
func AhdUUIDVersion(class *AhdClass, data string) int64 {
	raw := ahdUUIDStored(class, data)
	return int64(raw[6] >> 4)
}

func AhdUUIDIsZero(class *AhdClass, data string) bool {
	return ahdUUIDStored(class, data) == [16]byte{}
}

func AhdUUIDEquals(class *AhdClass, left, right string) bool {
	return ahdUUIDStored(class, left) == ahdUUIDStored(class, right)
}

// AhdUUIDCompare orders two UUIDs by their big-endian bytes (RFC 9562 section
// 6.11), which is also their lowercase text order.
func AhdUUIDCompare(class *AhdClass, left, right string) int64 {
	leftRaw := ahdUUIDStored(class, left)
	rightRaw := ahdUUIDStored(class, right)
	return int64(bytes.Compare(leftRaw[:], rightRaw[:]))
}
