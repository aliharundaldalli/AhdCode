package ahdruntime

import (
	"math"
	"reflect"
	"testing"
)

func TestMathRoundFloorAndCeil(t *testing.T) {
	if AhdMathRound(3.4) != 3 || AhdMathRound(3.5) != 4 || AhdMathRound(-3.5) != -4 {
		t.Fatal("Math.round does not use half-away-from-zero")
	}
	if got := AhdMathRoundDigits(3.14159, 2); got != 3.14 {
		t.Fatalf("Math.round digits = %g", got)
	}
	if AhdMathFloor(3.9) != 3 || AhdMathFloor(-3.1) != -4 || AhdMathCeil(3.1) != 4 || AhdMathCeil(-3.9) != -3 {
		t.Fatal("Math.floor/ceil results are wrong")
	}
	expectRaise(t, AhdClassDomainError, func() { AhdMathRoundDigits(1.2, -1) })
	expectRaise(t, AhdClassDomainError, func() { AhdMathRoundDigits(1.2, 16) })
	expectRaise(t, AhdClassOverflowError, func() { AhdMathFloor(math.Ldexp(1, 63)) })
	expectRaise(t, AhdClassOverflowError, func() { AhdMathCeil(math.Ldexp(1, 63)) })
}

func TestClassicMathFunctionsAndErrors(t *testing.T) {
	if AhdMathSqrt(25) != 5 || AhdMathSin(0) != 0 || AhdMathCos(0) != 1 || AhdMathExp(0) != 1 {
		t.Fatal("classic Math result is wrong")
	}
	if math.Abs(AhdMathLog(math.E)-1) > 1e-15 || math.Abs(AhdMathLog10(100)-2) > 1e-15 {
		t.Fatal("Math logarithm result is wrong")
	}
	if result := AhdMathTan(0.5); math.IsInf(result, 0) || math.IsNaN(result) {
		t.Fatal("Math.tan exposed a non-finite result")
	}
	expectRaise(t, AhdClassDomainError, func() { AhdMathSqrt(-1) })
	expectRaise(t, AhdClassDomainError, func() { AhdMathLog(0) })
	expectRaise(t, AhdClassDomainError, func() { AhdMathLog(-1) })
	expectRaise(t, AhdClassDomainError, func() { AhdMathLog10(0) })
	expectRaise(t, AhdClassOverflowError, func() { AhdMathExp(1000) })
}

func TestSplitMix64GoldenRandomSequences(t *testing.T) {
	defer AhdMathSeed(ahdMathDefaultSeed)
	tests := []struct {
		seed int64
		want []float64
	}{
		{557, []float64{0.4121990632081577, 0.4686510900868295, 0.5840201876345011, 0.23231713977715018, 0.4016357493295778}},
		{42, []float64{0.7415648787718233, 0.1599103928769201, 0.27860113025513866, 0.34419071652363753, 0.03803016854024621}},
		{-1, []float64{0.8939429202831845, 0.9125972035944532, 0.21948196289526756, 0.4262344494451664, 0.7055706489695709}},
	}
	for _, test := range tests {
		AhdMathSeed(test.seed)
		for index, want := range test.want {
			got := AhdMathRandom()
			if got != want || got < 0 || got >= 1 {
				t.Fatalf("seed %d random %d = %.17g, want %.17g", test.seed, index, got, want)
			}
		}
	}
}

func TestMathSeedResetAndRandomIntContract(t *testing.T) {
	defer AhdMathSeed(ahdMathDefaultSeed)
	AhdMathSeed(42)
	first := AhdMathRandom()
	AhdMathSeed(42)
	if second := AhdMathRandom(); first != second {
		t.Fatal("Math.seed did not reset the shared sequence")
	}
	AhdMathSeed(42)
	withoutSingleton := AhdMathRandom()
	AhdMathSeed(42)
	if AhdMathRandomInt(7, 7) != 7 || AhdMathRandom() != withoutSingleton {
		t.Fatal("a singleton randomInt consumed generator state")
	}
	seenLow, seenHigh := false, false
	AhdMathSeed(42)
	for index := 0; index < 100; index++ {
		value := AhdMathRandomInt(1, 10)
		seenLow = seenLow || value == 1
		seenHigh = seenHigh || value == 10
	}
	if !seenLow || !seenHigh {
		t.Fatal("inclusive randomInt did not reach both tested endpoints")
	}
	for _, bounds := range [][2]int64{{1, 10}, {-10, -1}, {-5, 5}, {math.MinInt64, math.MaxInt64}, {math.MinInt64, math.MinInt64 + 3}, {math.MaxInt64 - 3, math.MaxInt64}} {
		for index := 0; index < 100; index++ {
			value := AhdMathRandomInt(bounds[0], bounds[1])
			if value < bounds[0] || value > bounds[1] {
				t.Fatalf("randomInt(%d, %d) = %d", bounds[0], bounds[1], value)
			}
		}
	}
	expectRaise(t, AhdClassDomainError, func() { AhdMathRandomInt(2, 1) })
}

func TestMathRandomIntDeterministicBucketSanity(t *testing.T) {
	defer AhdMathSeed(ahdMathDefaultSeed)
	AhdMathSeed(557)
	counts := make([]int, 7)
	const samples = 70000
	for index := 0; index < samples; index++ {
		counts[AhdMathRandomInt(0, 6)]++
	}
	for bucket, count := range counts {
		if count < 9600 || count > 10400 {
			t.Fatalf("bucket %d count = %d; deterministic sequence is visibly biased", bucket, count)
		}
	}
}

func TestListShuffleUsesDeterministicFisherYates(t *testing.T) {
	defer AhdMathSeed(ahdMathDefaultSeed)

	AhdMathSeed(42)
	values := AhdNewList(int64(1), int64(2), int64(3), int64(4), int64(5))
	alias := values
	values.Shuffle()
	if want := []int64{2, 3, 1, 5, 4}; !reflect.DeepEqual(values.Snapshot(), want) {
		t.Fatalf("seed 42 shuffle = %v, want %v", values.Snapshot(), want)
	}
	if !reflect.DeepEqual(alias.Snapshot(), values.Snapshot()) {
		t.Fatal("a List alias did not observe shuffle's in-place mutation")
	}

	AhdMathSeed(557)
	defaultValues := AhdNewList(int64(1), int64(2), int64(3), int64(4), int64(5))
	defaultValues.Shuffle()
	if want := []int64{5, 2, 3, 1, 4}; !reflect.DeepEqual(defaultValues.Snapshot(), want) {
		t.Fatalf("default-seed shuffle = %v, want %v", defaultValues.Snapshot(), want)
	}

	frozen := AhdNewList(int64(1), int64(2))
	AhdFreeze(frozen)
	AhdMathSeed(42)
	wantAfterFrozen := AhdMathRandom()
	AhdMathSeed(42)
	expectRaise(t, AhdClassConstantError, frozen.Shuffle)
	if got := AhdMathRandom(); got != wantAfterFrozen {
		t.Fatal("a rejected frozen List shuffle consumed Math RNG state")
	}
}

func TestListShuffleSharesMathStateAndSkipsTrivialLists(t *testing.T) {
	defer AhdMathSeed(ahdMathDefaultSeed)

	AhdMathSeed(42)
	expectedAfterEmpty := AhdMathRandom()
	AhdMathSeed(42)
	AhdNewList[int64]().Shuffle()
	if got := AhdMathRandom(); got != expectedAfterEmpty {
		t.Fatal("empty List shuffle consumed Math RNG state")
	}

	AhdMathSeed(42)
	expectedAfterSingleton := AhdMathRandom()
	AhdMathSeed(42)
	AhdNewList(int64(7)).Shuffle()
	if got := AhdMathRandom(); got != expectedAfterSingleton {
		t.Fatal("singleton List shuffle consumed Math RNG state")
	}

	AhdMathSeed(42)
	wantBefore := AhdMathRandom()
	for maximum := int64(4); maximum > 0; maximum-- {
		AhdMathRandomInt(0, maximum)
	}
	wantAfter := AhdMathRandomInt(1, 10)

	AhdMathSeed(42)
	gotBefore := AhdMathRandom()
	values := AhdNewList(int64(1), int64(2), int64(3), int64(4), int64(5))
	values.Shuffle()
	gotAfter := AhdMathRandomInt(1, 10)
	if gotBefore != wantBefore || gotAfter != wantAfter {
		t.Fatalf("shared Math sequence around shuffle = (%v, %d), want (%v, %d)", gotBefore, gotAfter, wantBefore, wantAfter)
	}
}
