package quantile

import (
	"math"
	"math/rand/v2"
	"sort"
	"testing"
)

func TestCompareUniform(tb *testing.T) {
	src := rand.NewChaCha8([32]byte{})
	r := rand.New(src)

	testCompare(tb, r.Float64)
}

func TestCompareNormal(tb *testing.T) {
	src := rand.NewChaCha8([32]byte{})
	r := rand.New(src)

	testCompare(tb, r.NormFloat64)
}

func testCompare(tb *testing.T, rand func() float64) {
	const W = 1024
	const N = 1000 * W

	ext := NewExact()
	td := NewExtremesBiased(0.01, W)
	//	td := NewExtremesBiased(0.01, 2048)
	td.Decay = 0.99

	tb.Logf("size %v  eps %.5f", td.size, td.Invariant)

	for i := 0; i < N; i++ {
		v := rand()

		ext.Insert(v)
		td.Insert(v)
	}

	var dt float64
	var n int

	pt := func(q float64) {
		e := ext.Query(q)

		t := td.Query(q)

		d := math.Abs(e - t)
		dt += d * d

		tb.Logf("q %.3f: exact %7.3f  tdigest %7.3f (%6.3f)", q, e, t, e-t)

		n++
	}

	qs := []float64{0.5}

	for q := 0.01; q < 0.5; q += q / 2 {
		qs = append(qs, q, 1-q)
	}

	sort.Float64s(qs)

	for _, q := range qs {
		pt(q)
	}

	tb.Logf("stddev: tdigest %.5f", math.Sqrt(dt/float64(n)))

	//	tb.Logf("exact dump\n%v", ext.dump())
	tb.Logf("tdigest dump\n%v", td.dump())
	tb.Logf("tdigest stats: compressions %v / %v,  average reduction %v / %v", td.Compressions, td.BruteCompressions, td.ElementsReduced, td.size)
}
