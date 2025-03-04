package kll

import (
	"math"
	"math/rand/v2"
	"sort"
	"testing"

	"nikand.dev/go/quantile"
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
	const W, D = 32, 5
	const N = 10000 * W * D

	ext := quantile.NewExact()
	kll := New(W, D)

	for i := 0; i < N; i++ {
		v := rand()

		ext.Insert(v)
		kll.Insert(v)
	}

	var dk float64
	var n int

	pt := func(q float64) {
		e := ext.Query(q)

		k := kll.Query(q)

		d := math.Abs(e - k)
		dk += d * d

		tb.Logf("q %.3f: exact %7.3f  kll %7.3f (%6.3f)", q, e, k, e-k)

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

	tb.Logf("stddev: kll %.5f", math.Sqrt(dk/float64(n)))

	//	tb.Logf("exact dump\n%v", ext.dump())
	tb.Logf("kll dump\n%v", kll.dump())
}
