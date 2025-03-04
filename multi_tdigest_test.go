package quantile

import (
	"math"
	"math/rand/v2"
	"sort"
	"testing"
)

func TestMulti(tb *testing.T) {
	const W, N = 32 * 5, 1000

	split := []float64{0.3, 0.6, 0.9, 1}

	ss := make([]*TDigest, len(split))

	for i := range ss {
		ss[i] = NewExtremesBiased(0.05, W)
	}

	src := rand.NewChaCha8([32]byte{})
	r := rand.New(src)

	ext := NewExact()

	for range N {
		v := r.Float64()

		ext.Insert(v)

		for j, s := range split {
			if v <= s {
				ss[j].Insert(v)
				break
			}
		}
	}

	var dt float64
	var n int

	pt := func(q float64) {
		e := ext.Query(q)

		t := Query(q, ss...)

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

	tb.Logf("stddev  %.5f", math.Sqrt(dt/float64(n)))

	for _, s := range ss {
		tb.Logf("dump\n%v", s.dump())
		tb.Logf("stats: compressions %v / %v,  average reduction %v / %v", s.Compressions, s.BruteCompressions, s.ElementsReduced, s.size)
	}
}
