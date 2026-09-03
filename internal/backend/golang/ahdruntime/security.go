package ahdruntime

// AhdCode Security standard module runtime.
//
// This file is compiled twice: once as part of the compiler (so ordinary Go
// tooling checks it) and once as generated program source (package clause
// rewritten to main). It therefore depends only on the Go standard library —
// no external modules — and must not import any other AhdCode package.
//
// Argon2id is implemented from scratch using only stdlib primitives (BLAKE2b
// is implemented below; everything else is constant-time comparison from
// crypto/subtle and random bytes from crypto/rand).

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"math/bits"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// BLAKE2b-512 — minimal, allocation-light implementation
// ---------------------------------------------------------------------------
// Reference: RFC 7693 and the BLAKE2 paper.
// Only the 512-bit output variant is implemented since that is all Argon2
// needs internally.

var blake2bIV = [8]uint64{
	0x6a09e667f3bcc908, 0xbb67ae8584caa73b,
	0x3c6ef372fe94f82b, 0xa54ff53a5f1d36f1,
	0x510e527fade682d1, 0x9b05688c2b3e6c1f,
	0x1f83d9abfb41bd6b, 0x5be0cd19137e2179,
}

var blake2bSigma = [12][16]byte{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
	{14, 10, 4, 8, 9, 15, 13, 6, 1, 12, 0, 2, 11, 7, 5, 3},
	{11, 8, 12, 0, 5, 2, 15, 13, 10, 14, 3, 6, 7, 1, 9, 4},
	{7, 9, 3, 1, 13, 12, 11, 14, 2, 6, 5, 10, 4, 0, 15, 8},
	{9, 0, 5, 7, 2, 4, 10, 15, 14, 1, 11, 12, 6, 8, 3, 13},
	{2, 12, 6, 10, 0, 11, 8, 3, 4, 13, 7, 5, 15, 14, 1, 9},
	{12, 5, 1, 15, 14, 13, 4, 10, 0, 7, 6, 3, 9, 2, 8, 11},
	{13, 11, 7, 14, 12, 1, 3, 9, 5, 0, 15, 4, 8, 6, 2, 10},
	{6, 15, 14, 9, 11, 3, 0, 8, 12, 2, 13, 7, 1, 4, 10, 5},
	{10, 2, 8, 4, 7, 6, 1, 5, 15, 11, 9, 14, 3, 12, 13, 0},
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
	{14, 10, 4, 8, 9, 15, 13, 6, 1, 12, 0, 2, 11, 7, 5, 3},
}

type blake2bState struct {
	h   [8]uint64
	t   [2]uint64
	f   [2]uint64
	buf [128]byte
	off int
}

func blake2bNew512(key []byte) *blake2bState {
	s := &blake2bState{}
	// Parameter block
	p := [64]byte{}
	p[0] = 64 // digest length
	p[1] = byte(len(key))
	p[2] = 1 // fanout
	p[3] = 1 // depth
	for i := 0; i < 8; i++ {
		s.h[i] = blake2bIV[i] ^ leUint64(p[i*8:i*8+8])
	}
	if len(key) > 0 {
		var block [128]byte
		copy(block[:], key)
		s.write(block[:])
	}
	return s
}

func leUint64(b []byte) uint64 {
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
}

func putLeUint64(b []byte, v uint64) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
	b[4] = byte(v >> 32)
	b[5] = byte(v >> 40)
	b[6] = byte(v >> 48)
	b[7] = byte(v >> 56)
}

func (s *blake2bState) write(p []byte) {
	for len(p) > 0 {
		room := 128 - s.off
		if room == 0 {
			s.compress(false)
			s.off = 0
		}
		take := len(p)
		if take > 128-s.off {
			take = 128 - s.off
		}
		copy(s.buf[s.off:], p[:take])
		s.off += take
		p = p[take:]
	}
}

func (s *blake2bState) compress(final bool) {
	var v [16]uint64
	for i := 0; i < 8; i++ {
		v[i] = s.h[i]
		v[i+8] = blake2bIV[i]
	}
	// Increment counter
	s.t[0] += uint64(s.off)
	if s.t[0] < uint64(s.off) {
		s.t[1]++
	}
	v[12] ^= s.t[0]
	v[13] ^= s.t[1]
	if final {
		v[14] ^= 0xffffffffffffffff
	}
	var m [16]uint64
	for i := 0; i < 16; i++ {
		m[i] = leUint64(s.buf[i*8 : i*8+8])
	}
	for round := 0; round < 12; round++ {
		sig := blake2bSigma[round]
		blake2bG(&v, 0, 4, 8, 12, m[sig[0]], m[sig[1]])
		blake2bG(&v, 1, 5, 9, 13, m[sig[2]], m[sig[3]])
		blake2bG(&v, 2, 6, 10, 14, m[sig[4]], m[sig[5]])
		blake2bG(&v, 3, 7, 11, 15, m[sig[6]], m[sig[7]])
		blake2bG(&v, 0, 5, 10, 15, m[sig[8]], m[sig[9]])
		blake2bG(&v, 1, 6, 11, 12, m[sig[10]], m[sig[11]])
		blake2bG(&v, 2, 7, 8, 13, m[sig[12]], m[sig[13]])
		blake2bG(&v, 3, 4, 9, 14, m[sig[14]], m[sig[15]])
	}
	for i := 0; i < 8; i++ {
		s.h[i] ^= v[i] ^ v[i+8]
	}
}

func blake2bG(v *[16]uint64, a, b, c, d int, x, y uint64) {
	v[a] = v[a] + v[b] + x
	v[d] = bits.RotateLeft64(v[d]^v[a], -32)
	v[c] = v[c] + v[d]
	v[b] = bits.RotateLeft64(v[b]^v[c], -24)
	v[a] = v[a] + v[b] + y
	v[d] = bits.RotateLeft64(v[d]^v[a], -16)
	v[c] = v[c] + v[d]
	v[b] = bits.RotateLeft64(v[b]^v[c], -63)
}

func (s *blake2bState) sum512() [64]byte {
	// Pad remaining buffer with zeros
	for i := s.off; i < 128; i++ {
		s.buf[i] = 0
	}
	s.compress(true)
	var out [64]byte
	for i := 0; i < 8; i++ {
		putLeUint64(out[i*8:], s.h[i])
	}
	return out
}

// blake2bHash hashes input with optional key and returns 64 bytes.
func blake2bHash(key, input []byte) [64]byte {
	s := blake2bNew512(key)
	s.write(input)
	return s.sum512()
}

// blake2bHashLen hashes input and writes exactly outLen bytes into out
// using the variable-output BLAKE2b construction (H' from Argon2 spec).
func blake2bHashLen(out []byte, input []byte) {
	// H' from RFC 9106 §3.2
	n := len(out)
	if n <= 64 {
		// Single call: encode length as 4-byte LE prefix then hash.
		prefix := []byte{byte(n), byte(n >> 8), byte(n >> 16), byte(n >> 24)}
		s := blake2bNew512(nil)
		s.h[0] = s.h[0]&^0xff | uint64(n) // set digest length in param block
		// Rebuild state with correct output length in parameter block.
		// Simpler: use the keyed API with a crafted parameter block.
		// Easiest correct path: just write len-prefix + input as raw data.
		_ = s
		// Re-do properly: build parameter block with digest_length = n
		var p [64]byte
		p[0] = byte(n)
		p[1] = 0 // no key
		p[2] = 1
		p[3] = 1
		st := &blake2bState{}
		for i := 0; i < 8; i++ {
			st.h[i] = blake2bIV[i] ^ leUint64(p[i*8:i*8+8])
		}
		st.write(prefix)
		st.write(input)
		// finalize with correct digest size
		for i := st.off; i < 128; i++ {
			st.buf[i] = 0
		}
		st.compress(true)
		var full [64]byte
		for i := 0; i < 8; i++ {
			putLeUint64(full[i*8:], st.h[i])
		}
		copy(out, full[:n])
		return
	}
	// Long output: A1 = H(4||τ||x) with 64-byte output, then chain.
	a1Input := make([]byte, 4+len(input))
	a1Input[0] = byte(n)
	a1Input[1] = byte(n >> 8)
	a1Input[2] = byte(n >> 16)
	a1Input[3] = byte(n >> 24)
	copy(a1Input[4:], input)
	prev := blake2bHash(nil, a1Input)

	written := 32
	copy(out, prev[:32])
	for written+64 <= n {
		cur := blake2bHash(nil, prev[:])
		copy(out[written:], cur[:32])
		prev = cur
		written += 32
	}
	// Final block: output remaining bytes
	remaining := n - written
	if remaining > 0 {
		cur := blake2bHash(nil, prev[:])
		copy(out[written:], cur[:remaining])
	}
}

// ---------------------------------------------------------------------------
// Argon2id — RFC 9106, memory-hard password hashing
// ---------------------------------------------------------------------------

const (
	argon2idMemoryKiB   = 65536 // 64 MiB
	argon2idTime        = 3
	argon2idParallelism = 1
	argon2idSaltLen     = 16
	argon2idKeyLen      = 32
	argon2idVersion     = 19 // 0x13

	argon2BlockSize = 1024 // bytes per block
)

// argon2idHash computes the Argon2id hash of password with the given salt and
// parameters. It uses the pure-stdlib BLAKE2b implementation above.
func argon2idHash(password, salt []byte, memory, time, parallelism uint32, keyLen uint32) []byte {
	// Number of blocks, rounded to parallelism*4
	lanes := parallelism
	segments := uint32(4)
	blocks := memory / (lanes * segments)
	if blocks < 2 {
		blocks = 2
	}
	totalBlocks := blocks * lanes * segments

	// H0: initial hash
	h0 := argon2idH0(password, salt, memory, time, parallelism, keyLen)

	// Allocate memory blocks: totalBlocks * 1024 bytes
	mem := make([]uint64, totalBlocks*128) // 128 uint64 per 1024-byte block

	blockOf := func(idx uint32) []uint64 { return mem[idx*128 : idx*128+128] }
	copyBlock := func(dst, src []uint64) { copy(dst, src) }
	xorBlock := func(dst, src []uint64) {
		for i := range dst {
			dst[i] ^= src[i]
		}
	}

	// Initialize first two blocks of each lane
	var buf [72]byte // h0 (64) + 4-byte index + 4-byte lane
	for lane := uint32(0); lane < lanes; lane++ {
		copy(buf[:64], h0[:])
		buf[64] = 0
		buf[65] = 0
		buf[66] = 0
		buf[67] = 0
		buf[68] = byte(lane)
		buf[69] = byte(lane >> 8)
		buf[70] = byte(lane >> 16)
		buf[71] = byte(lane >> 24)

		idx0 := lane * (blocks * segments)
		block0 := blockOf(idx0)
		argon2idH64(block0, buf[:])

		buf[64] = 1
		buf[65] = 0
		buf[66] = 0
		buf[67] = 0
		idx1 := idx0 + 1
		block1 := blockOf(idx1)
		argon2idH64(block1, buf[:])
	}

	// Fill passes
	for pass := uint32(0); pass < time; pass++ {
		for slice := uint32(0); slice < segments; slice++ {
			for lane := uint32(0); lane < lanes; lane++ {
				argon2idFillSegment(mem, blockOf, copyBlock, xorBlock,
					pass, slice, lane, blocks, lanes, segments, totalBlocks)
			}
		}
	}

	// Finalize: XOR last blocks of each lane into lane 0's last block
	lastIdx := (lanes-1)*(blocks*segments) + (blocks*segments - 1)
	finalBlock := make([]uint64, 128)
	copy(finalBlock, blockOf((blocks*segments)-1)) // lane 0 last block
	for lane := uint32(1); lane < lanes; lane++ {
		laneLastIdx := lane*(blocks*segments) + (blocks*segments - 1)
		_ = lastIdx
		for i := range finalBlock {
			finalBlock[i] ^= blockOf(laneLastIdx)[i]
		}
	}

	// Serialize final block to bytes
	finalBytes := make([]byte, argon2BlockSize)
	for i := 0; i < 128; i++ {
		putLeUint64(finalBytes[i*8:], finalBlock[i])
	}

	// Extract key using H'
	out := make([]byte, keyLen)
	blake2bHashLen(out, finalBytes)
	return out
}

// argon2idH0 computes the initial 64-byte hash H0 from all inputs.
func argon2idH0(password, salt []byte, memory, time, parallelism, keyLen uint32) [64]byte {
	// H0 = H(||p||τ||m||t||v||y||||password||||salt||||key||||data||)
	// For our use: key and data are empty; y=2 for Argon2id.
	buf := make([]byte, 0, 10*4+len(password)+len(salt))
	buf = appendLeUint32(buf, parallelism)
	buf = appendLeUint32(buf, keyLen)
	buf = appendLeUint32(buf, memory)
	buf = appendLeUint32(buf, time)
	buf = appendLeUint32(buf, uint32(argon2idVersion))
	buf = appendLeUint32(buf, 2) // type=2 for Argon2id
	buf = appendLeUint32(buf, uint32(len(password)))
	buf = append(buf, password...)
	buf = appendLeUint32(buf, uint32(len(salt)))
	buf = append(buf, salt...)
	buf = appendLeUint32(buf, 0) // key length = 0
	buf = appendLeUint32(buf, 0) // data length = 0
	return blake2bHash(nil, buf)
}

func appendLeUint32(b []byte, v uint32) []byte {
	return append(b, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

// argon2idH64 fills a 1024-byte block (as 128 uint64s) using H'(input).
func argon2idH64(block []uint64, input []byte) {
	var outBytes [argon2BlockSize]byte
	blake2bHashLen(outBytes[:], input)
	for i := 0; i < 128; i++ {
		block[i] = leUint64(outBytes[i*8 : i*8+8])
	}
}

func argon2idFillSegment(
	mem []uint64,
	blockOf func(uint32) []uint64,
	copyBlock func(dst, src []uint64),
	xorBlock func(dst, src []uint64),
	pass, slice, lane, blocks, lanes, segments, totalBlocks uint32,
) {
	startIdx := uint32(0)
	if pass == 0 && slice == 0 {
		startIdx = 2
	}

	laneBase := lane * (blocks * segments)
	curOffset := laneBase + slice*blocks + startIdx

	for idx := startIdx; idx < blocks; idx++ {
		curIdx := laneBase + slice*blocks + idx

		var prevIdx uint32
		if curIdx == laneBase {
			prevIdx = laneBase + (blocks*segments - 1)
		} else {
			prevIdx = curIdx - 1
		}

		// Compute reference block index
		refLane, refIdx := argon2idRefBlock(mem, blockOf, pass, slice, lane, idx, blocks, lanes, segments, prevIdx)
		_ = refLane

		cur := blockOf(curIdx)
		prev := blockOf(prevIdx)
		ref := blockOf(refIdx)

		var tmp [128]uint64
		argon2idCompress(tmp[:], prev, ref)

		if pass == 0 {
			copyBlock(cur, tmp[:])
		} else {
			xorBlock(cur, tmp[:])
		}
		_ = curOffset
	}
}

func argon2idRefBlock(
	mem []uint64,
	blockOf func(uint32) []uint64,
	pass, slice, lane, idx, blocks, lanes, segments, prevIdx uint32,
) (uint32, uint32) {
	// Determine the pseudo-random value from the first 64 bits of G(prev, prev)
	var tmp [128]uint64
	prev := blockOf(prevIdx)
	argon2idCompress(tmp[:], prev, prev)
	j1 := uint32(tmp[0] & 0xffffffff)
	j2 := uint32(tmp[0] >> 32)

	// Select reference lane
	var refLane uint32
	if pass == 0 && slice == 0 {
		refLane = lane
	} else {
		refLane = j2 % lanes
	}

	// Reference set size
	refAreaSize := uint32(0)
	if pass == 0 {
		if slice == 0 {
			refAreaSize = idx - 1
		} else if refLane == lane {
			refAreaSize = slice*blocks + idx - 1
		} else {
			refAreaSize = slice*blocks - 1
			if idx == 0 {
				refAreaSize--
			}
		}
	} else {
		if refLane == lane {
			refAreaSize = blocks*segments - blocks + idx - 1
		} else {
			refAreaSize = blocks*segments - blocks - 1
			if idx == 0 {
				refAreaSize--
			}
		}
	}

	startPos := uint32(0)
	if pass != 0 && slice != segments-1 {
		startPos = (slice + 1) * blocks
	}

	// Map j1 to an index within the reference set
	x := uint64(j1) * uint64(j1) >> 32
	y := uint64(refAreaSize) * x >> 32
	z := uint64(refAreaSize) - 1 - y

	refIndex := (startPos + uint32(z)) % (blocks * segments)
	return refLane, refLane*(blocks*segments) + refIndex
}

// argon2idCompress fills dst with G(x, y) where x=prev, y=ref.
// Uses the Argon2 compression function (BLAKE2b-based rounds on 8x8 matrices).
func argon2idCompress(dst, prev, ref []uint64) {
	var r [128]uint64
	for i := range r {
		r[i] = prev[i] ^ ref[i]
	}
	copy(dst, r[:])

	// Apply BLAKE2b-based G permutation on 8x8 sub-blocks.
	// Process 8 rows of 16 uint64 each.
	for i := 0; i < 8; i++ {
		argon2idBlamka(
			&dst[16*i], &dst[16*i+1], &dst[16*i+2], &dst[16*i+3],
			&dst[16*i+4], &dst[16*i+5], &dst[16*i+6], &dst[16*i+7],
			&dst[16*i+8], &dst[16*i+9], &dst[16*i+10], &dst[16*i+11],
			&dst[16*i+12], &dst[16*i+13], &dst[16*i+14], &dst[16*i+15],
		)
	}
	// Process 8 columns of 16 uint64 (non-contiguous).
	for i := 0; i < 8; i++ {
		argon2idBlamka(
			&dst[2*i], &dst[2*i+1], &dst[2*i+16], &dst[2*i+17],
			&dst[2*i+32], &dst[2*i+33], &dst[2*i+48], &dst[2*i+49],
			&dst[2*i+64], &dst[2*i+65], &dst[2*i+80], &dst[2*i+81],
			&dst[2*i+96], &dst[2*i+97], &dst[2*i+112], &dst[2*i+113],
		)
	}
	for i := range dst {
		dst[i] ^= r[i]
	}
}

// argon2idBlamka applies the quarter-round G function to 16 uint64 values.
// This is the Argon2 variant of BLAKE2b's G using the fBlaMka MUL operation.
func argon2idBlamka(v0, v1, v2, v3, v4, v5, v6, v7, v8, v9, v10, v11, v12, v13, v14, v15 *uint64) {
	argon2idG16(v0, v4, v8, v12)
	argon2idG16(v1, v5, v9, v13)
	argon2idG16(v2, v6, v10, v14)
	argon2idG16(v3, v7, v11, v15)
	argon2idG16(v0, v5, v10, v15)
	argon2idG16(v1, v6, v11, v12)
	argon2idG16(v2, v7, v8, v13)
	argon2idG16(v3, v4, v9, v14)
}

func argon2idG16(a, b, c, d *uint64) {
	*a = *a + *b + 2*(*a&0xffffffff)*(*b&0xffffffff)
	*d = bits.RotateLeft64(*d^*a, -32)
	*c = *c + *d + 2*(*c&0xffffffff)*(*d&0xffffffff)
	*b = bits.RotateLeft64(*b^*c, -24)
	*a = *a + *b + 2*(*a&0xffffffff)*(*b&0xffffffff)
	*d = bits.RotateLeft64(*d^*a, -16)
	*c = *c + *d + 2*(*c&0xffffffff)*(*d&0xffffffff)
	*b = bits.RotateLeft64(*b^*c, -63)
}

// ---------------------------------------------------------------------------
// PHC string format helpers
// ---------------------------------------------------------------------------

// phcEncode produces the PHC string for an argon2id hash.
func phcEncode(salt, hash []byte, memory, time, parallelism uint32) string {
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2idVersion,
		memory, time, parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
}

// phcDecode parses a PHC string and returns its components.
// Returns an error string if malformed.
func phcDecode(encoded string) (memory, time, parallelism uint32, salt, hash []byte, errMsg string) {
	parts := strings.Split(encoded, "$")
	// $argon2id$v=19$m=65536,t=3,p=1$<salt>$<hash>
	// → ["", "argon2id", "v=19", "m=65536,t=3,p=1", "<salt>", "<hash>"]
	if len(parts) != 6 || parts[0] != "" {
		return 0, 0, 0, nil, nil, "Security password hash is malformed"
	}
	if parts[1] != "argon2id" {
		return 0, 0, 0, nil, nil, "Security password hash uses an unsupported algorithm"
	}
	// Version
	if !strings.HasPrefix(parts[2], "v=") {
		return 0, 0, 0, nil, nil, "Security password hash is malformed"
	}
	version, err := strconv.ParseUint(parts[2][2:], 10, 32)
	if err != nil || version != argon2idVersion {
		return 0, 0, 0, nil, nil, "Security password hash uses an unsupported algorithm"
	}
	// Parameters
	params := parts[3]
	paramMap := make(map[string]uint64)
	for _, kv := range strings.Split(params, ",") {
		eq := strings.IndexByte(kv, '=')
		if eq < 0 {
			return 0, 0, 0, nil, nil, "Security password hash is malformed"
		}
		val, perr := strconv.ParseUint(kv[eq+1:], 10, 32)
		if perr != nil {
			return 0, 0, 0, nil, nil, "Security password hash is malformed"
		}
		paramMap[kv[:eq]] = val
	}
	mVal, mOk := paramMap["m"]
	tVal, tOk := paramMap["t"]
	pVal, pOk := paramMap["p"]
	if !mOk || !tOk || !pOk {
		return 0, 0, 0, nil, nil, "Security password hash is malformed"
	}
	memory = uint32(mVal)
	time = uint32(tVal)
	parallelism = uint32(pVal)
	// Bounds checks (before running Argon2)
	if memory < 8192 || memory > 262144 {
		return 0, 0, 0, nil, nil, "Security password hash has unsafe parameters"
	}
	if time < 1 || time > 10 {
		return 0, 0, 0, nil, nil, "Security password hash has unsafe parameters"
	}
	if parallelism < 1 || parallelism > 16 {
		return 0, 0, 0, nil, nil, "Security password hash has unsafe parameters"
	}
	// Decode salt and hash
	salt, serr := base64.RawStdEncoding.DecodeString(parts[4])
	if serr != nil {
		return 0, 0, 0, nil, nil, "Security password hash is malformed"
	}
	hash, herr := base64.RawStdEncoding.DecodeString(parts[5])
	if herr != nil {
		return 0, 0, 0, nil, nil, "Security password hash is malformed"
	}
	if len(salt) < 8 || len(salt) > 64 {
		return 0, 0, 0, nil, nil, "Security password hash has unsafe parameters"
	}
	if len(hash) < 16 || len(hash) > 64 {
		return 0, 0, 0, nil, nil, "Security password hash has unsafe parameters"
	}
	return memory, time, parallelism, salt, hash, ""
}

// ---------------------------------------------------------------------------
// Public AhdCode Security functions
// ---------------------------------------------------------------------------

const securityMaxPasswordBytes = 1 << 20 // 1 MiB

// AhdSecurityPasswordHash hashes password with Argon2id and returns a PHC string.
func AhdSecurityPasswordHash(errorClass *AhdClass, password string) string {
	if len(password) > securityMaxPasswordBytes {
		AhdRaiseClass(errorClass, "Security password input is too large")
	}
	salt := make([]byte, argon2idSaltLen)
	if _, err := rand.Read(salt); err != nil {
		AhdRaiseClass(errorClass, "Security random token generation failed")
	}
	hash := argon2idHash([]byte(password), salt,
		argon2idMemoryKiB, argon2idTime, argon2idParallelism, argon2idKeyLen)
	return phcEncode(salt, hash, argon2idMemoryKiB, argon2idTime, argon2idParallelism)
}

// AhdSecurityPasswordVerify verifies a password against a PHC-encoded Argon2id hash.
// Returns false for wrong password; raises SecurityError for malformed/unsafe hashes.
func AhdSecurityPasswordVerify(errorClass *AhdClass, password, encodedHash string) bool {
	memory, time, parallelism, salt, storedHash, errMsg := phcDecode(encodedHash)
	if errMsg != "" {
		AhdRaiseClass(errorClass, errMsg)
	}
	candidate := argon2idHash([]byte(password), salt, memory, time, parallelism, uint32(len(storedHash)))
	return subtle.ConstantTimeCompare(candidate, storedHash) == 1
}

// AhdSecurityToken generates a cryptographically secure random token.
// Returns 43 characters of unpadded base64url (32 random bytes).
func AhdSecurityToken(errorClass *AhdClass) string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		AhdRaiseClass(errorClass, "Security random token generation failed")
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

// AhdSecuritySecureEqual compares two strings in constant time.
// Intended for fixed-length secret comparison (tokens, HMACs).
func AhdSecuritySecureEqual(expected, received string) bool {
	return subtle.ConstantTimeCompare([]byte(expected), []byte(received)) == 1
}
