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
	const W, D = 32, 5
	const N = 10000 * W * D

	ext := NewExact()
	kll := NewKLL(W, D)
	td := NewTDigest(W*D, TDigestEpsilon(W*D))
	td.Decay = 0.99

	for i := 0; i < N; i++ {
		v := rand()

		ext.Insert(v)
		kll.Insert(v)
		td.Insert(v)
	}

	var dk, dt float64
	var n int

	pt := func(q float64) {
		e := ext.Query(q)

		k := kll.Query(q)
		t := td.Query(q)

		d := math.Abs(e - k)
		dk += d * d

		d = math.Abs(e - t)
		dt += d * d

		tb.Logf("q %.3f: exact %7.3f  kll %7.3f (%6.3f)  tdigest %7.3f (%6.3f)", q, e, k, e-k, t, e-t)

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

	tb.Logf("stddev: kll %.5f  tdigest %.5f", math.Sqrt(dk/float64(n)), math.Sqrt(dt/float64(n)))

	//	tb.Logf("exact dump\n%v", ext.dump())
	tb.Logf("kll dump\n%v", kll.dump())
	tb.Logf("tdigest dump\n%v", td.dump())
}
