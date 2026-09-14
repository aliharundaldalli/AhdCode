package ahdruntime

import (
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestUUIDParseRFC9562Vectors(t *testing.T) {
	cases := []struct {
		text      string
		canonical string
		version   int64
	}{
		// RFC 9562 appendix A.3 (version 4) and A.6 (version 7).
		{"919108f7-52d1-4320-9bac-f847db4148a8", "919108f7-52d1-4320-9bac-f847db4148a8", 4},
		{"017F22E2-79B0-7CC3-98C4-DC0C0C07398F", "017f22e2-79b0-7cc3-98c4-dc0c0c07398f", 7},
		{"017f22E2-79b0-7Cc3-98c4-dc0C0c07398F", "017f22e2-79b0-7cc3-98c4-dc0c0c07398f", 7},
		// The zero (Nil) and max UUIDs are ordinary 128-bit values.
		{"00000000-0000-0000-0000-000000000000", "00000000-0000-0000-0000-000000000000", 0},
		{"FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF", "ffffffff-ffff-ffff-ffff-ffffffffffff", 15},
	}
	for _, testCase := range cases {
		if !AhdUUIDIsValid(testCase.text) {
			t.Fatalf("%q is not valid", testCase.text)
		}
		parsed := AhdUUIDParse(AhdClassUUIDError, testCase.text)
		if parsed != testCase.canonical {
			t.Fatalf("parse(%q) = %q, want %q", testCase.text, parsed, testCase.canonical)
		}
		if version := AhdUUIDVersion(AhdClassUUIDError, parsed); version != testCase.version {
			t.Fatalf("version(%q) = %d, want %d", parsed, version, testCase.version)
		}
	}
	if !AhdUUIDIsZero(AhdClassUUIDError, AhdUUIDZero()) || AhdUUIDZero() != "00000000-0000-0000-0000-000000000000" {
		t.Fatal("zero UUID is wrong")
	}
	if AhdUUIDIsZero(AhdClassUUIDError, "ffffffff-ffff-ffff-ffff-ffffffffffff") {
		t.Fatal("max UUID reported as zero")
	}
}

func TestUUIDParseIsStrict(t *testing.T) {
	for _, text := range []string{
		"",
		"017f22e2-79b0-7cc3-98c4-dc0c0c07398",
		"017f22e2-79b0-7cc3-98c4-dc0c0c07398f0",
		"{017f22e2-79b0-7cc3-98c4-dc0c0c07398f}",
		"urn:uuid:017f22e2-79b0-7cc3-98c4-dc0c0c07398f",
		"017f22e279b07cc398c4dc0c0c07398f",
		" 017f22e2-79b0-7cc3-98c4-dc0c0c07398",
		"017f22e2-79b0-7cc3-98c4-dc0c0c07398f\n",
		"017f22e2779b0-7cc3-98c4-dc0c0c07398f",
		"017f22e2-79b07-cc3-98c4-dc0c0c07398f",
		"017f22e2_79b0_7cc3_98c4_dc0c0c07398f",
		"017f22e2-79b0-7cc3-98c4-dc0c0c07398g",
		"017f22e2-79b0-7cc3-98c4-dc0c0c0739ı",
		"+17f22e2-79b0-7cc3-98c4-dc0c0c07398f",
	} {
		if AhdUUIDIsValid(text) {
			t.Fatalf("%q must be invalid", text)
		}
		message := codesRaised(t, func() { AhdUUIDParse(AhdClassUUIDError, text) })
		if message != ahdUUIDTextMessage {
			t.Fatalf("parse(%q) message %q", text, message)
		}
		if text != "" && strings.Contains(message, strings.TrimSpace(text)) {
			t.Fatalf("parse message echoes the rejected text %q", text)
		}
	}
}

func TestUUIDV4Bits(t *testing.T) {
	seen := map[string]bool{}
	for index := 0; index < 4000; index++ {
		text := AhdUUIDV4(AhdClassUUIDError)
		if len(text) != 36 || AhdUUIDParse(AhdClassUUIDError, text) != text {
			t.Fatalf("v4 %q is not canonical", text)
		}
		if text[14] != '4' || !strings.ContainsRune("89ab", rune(text[19])) {
			t.Fatalf("v4 %q has the wrong version or variant", text)
		}
		if seen[text] {
			t.Fatalf("v4 repeated %q", text)
		}
		seen[text] = true
	}
}

// RFC 9562 appendix A.6 uses unix_ts_ms 0x017F22E279B0 (2022-02-22 19:22:22 UTC).
func TestUUIDV7Layout(t *testing.T) {
	base := time.UnixMilli(0x017F22E279B0)
	clock := &ahdUUIDClock{}
	text := ahdUUIDV7At(AhdClassUUIDError, clock, base)
	if !strings.HasPrefix(text, "017f22e2-79b0-7000-") {
		t.Fatalf("v7 at the RFC timestamp = %q", text)
	}
	if !strings.ContainsRune("89ab", rune(text[19])) || AhdUUIDVersion(AhdClassUUIDError, text) != 7 {
		t.Fatalf("v7 %q has the wrong version or variant", text)
	}
	// Half a millisecond later the 12-bit fraction is 2048 (0x800).
	half := ahdUUIDV7At(AhdClassUUIDError, clock, base.Add(500*time.Microsecond))
	if !strings.HasPrefix(half, "017f22e2-79b0-7800-") {
		t.Fatalf("v7 half a millisecond later = %q", half)
	}
}

func TestUUIDV7IsStrictlyIncreasingWithinOneMillisecond(t *testing.T) {
	clock := &ahdUUIDClock{}
	now := time.UnixMilli(0x017F22E279B0)
	previous := ""
	for index := 0; index < 5000; index++ {
		text := ahdUUIDV7At(AhdClassUUIDError, clock, now)
		if text <= previous {
			t.Fatalf("value %d %q is not after %q", index, text, previous)
		}
		previous = text
	}
	// More than 4096 values in one millisecond carry into the next millisecond,
	// running the embedded time ahead of the clock (RFC 9562 section 6.2).
	if !strings.HasPrefix(previous, "017f22e2-79b1-7") {
		t.Fatalf("after 5000 values in one millisecond the embedded time is %q", previous)
	}
}

func TestUUIDV7SurvivesABackwardClock(t *testing.T) {
	clock := &ahdUUIDClock{}
	now := time.UnixMilli(0x017F22E279B0)
	before := ahdUUIDV7At(AhdClassUUIDError, clock, now)
	earlier := ahdUUIDV7At(AhdClassUUIDError, clock, now.Add(-time.Hour))
	if earlier <= before {
		t.Fatalf("after the clock moved back, %q is not after %q", earlier, before)
	}
	if !strings.HasPrefix(earlier, "017f22e2-79b0-7") {
		t.Fatalf("a backward step should continue from the last value, got %q", earlier)
	}
	later := ahdUUIDV7At(AhdClassUUIDError, clock, now.Add(time.Second))
	if later <= earlier || !strings.HasPrefix(later, "017f22e2-7d98-7") {
		t.Fatalf("once the clock catches up the embedded time follows it again, got %q", later)
	}
}

func TestUUIDV7ClockLimits(t *testing.T) {
	for _, now := range []time.Time{time.UnixMilli(-1), time.Unix(-1, 999_999_999), time.UnixMilli(int64(1) << 48)} {
		message := codesRaised(t, func() { ahdUUIDV7At(AhdClassUUIDError, &ahdUUIDClock{}, now) })
		if message != ahdUUIDClockMessage {
			t.Fatalf("clock %v message %q", now, message)
		}
	}
	last := ahdUUIDV7At(AhdClassUUIDError, &ahdUUIDClock{}, time.UnixMilli(int64(1)<<48-1))
	if !strings.HasPrefix(last, "ffffffff-ffff-7") {
		t.Fatalf("the last representable millisecond = %q", last)
	}
	exhausted := &ahdUUIDClock{last: uint64(1)<<60 - 1, issued: true}
	message := codesRaised(t, func() { ahdUUIDV7At(AhdClassUUIDError, exhausted, time.UnixMilli(0x017F22E279B0)) })
	if message != ahdUUIDClockMessage {
		t.Fatalf("exhausted clock message %q", message)
	}
}

func TestUUIDV7ConcurrentGenerationIsUniqueAndOrdered(t *testing.T) {
	const workers, perWorker = 32, 2000
	results := make([][]string, workers)
	var group sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		group.Add(1)
		go func(worker int) {
			defer group.Done()
			values := make([]string, perWorker)
			for index := range values {
				values[index] = AhdUUIDV7(AhdClassUUIDError)
			}
			results[worker] = values
		}(worker)
	}
	group.Wait()
	seen := map[string]bool{}
	for _, values := range results {
		if !sort.StringsAreSorted(values) {
			t.Fatal("one goroutine's version 7 values are not in generation order")
		}
		for _, value := range values {
			if seen[value] {
				t.Fatalf("version 7 repeated %q", value)
			}
			seen[value] = true
		}
	}
}

func TestUUIDEqualsCompareAndStorage(t *testing.T) {
	lower := AhdUUIDParse(AhdClassUUIDError, "017f22e2-79b0-7cc3-98c4-dc0c0c07398f")
	upper := AhdUUIDParse(AhdClassUUIDError, "017F22E2-79B0-7CC3-98C4-DC0C0C07398F")
	other := AhdUUIDParse(AhdClassUUIDError, "919108f7-52d1-4320-9bac-f847db4148a8")
	if !AhdUUIDEquals(AhdClassUUIDError, lower, upper) || AhdUUIDEquals(AhdClassUUIDError, lower, other) {
		t.Fatal("equals is wrong")
	}
	if AhdUUIDCompare(AhdClassUUIDError, lower, other) != -1 || AhdUUIDCompare(AhdClassUUIDError, other, lower) != 1 ||
		AhdUUIDCompare(AhdClassUUIDError, lower, upper) != 0 {
		t.Fatal("compare is wrong")
	}
	for index := 0; index < 500; index++ {
		left, right := AhdUUIDV4(AhdClassUUIDError), AhdUUIDV4(AhdClassUUIDError)
		if AhdUUIDCompare(AhdClassUUIDError, left, right) != int64(strings.Compare(left, right)) {
			t.Fatalf("compare disagrees with text order for %q and %q", left, right)
		}
	}
	for _, corrupted := range []string{"", "not a uuid", "017F22E2-79B0-7CC3-98C4-DC0C0C07398F"} {
		message := codesRaised(t, func() { AhdUUIDVersion(AhdClassUUIDError, corrupted) })
		if message != ahdUUIDStorageMessage {
			t.Fatalf("corrupted storage %q message %q", corrupted, message)
		}
	}
}
