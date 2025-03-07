package quantile

import (
	"math/rand/v2"
	"testing"
)

func TestTDMulti(tb *testing.T) {
	const W, N = 32 * 4, 1000

	split := []float64{0.3, 0.6, 0.9, 1}

	ss := make(TDMulti, len(split))

	for i := range ss {
		ss[i] = NewTDExtremesBiased(0.05, W)
	}

	src := rand.NewChaCha8([32]byte{})
	r := rand.New(src)

	testCompare(tb, r.Float64, ss)

	for _, s := range ss {
		//	tb.Logf("dump %d\n%v", j, s.dump())
		tb.Logf("stats: compressions %v / %v,  average reduction %.2f / %v", s.Compressions, s.BruteCompressions, s.ElementsReduced, s.size)
	}
}

func TestTDMulti10(tb *testing.T) {
	e := NewExact()
	ss := TDMulti{
		NewTDExtremesBiased(0.01, 16),
		NewTDExtremesBiased(0.01, 16),
		NewTDExtremesBiased(0.01, 16),
		NewTDExtremesBiased(0.01, 16),
	}

	for i, v := range []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1} {
		e.Insert(v)

		if i < 5 {
			ss[0].Insert(v)
		} else {
			ss[1].Insert(v)
		}
	}

	qs := []float64{0., 0.01, 0.1, 0.5, 0.9, 0.99, 1}

	for _, q := range qs {
		assertEqual(tb, e, ss, q, 0.02)
	}

	res := make([]float64, len(qs))

	ss.QueryMulti(qs, res)

	tb.Logf("multi: %v -> %v", qs, res)
}

func (s TDMulti) Insert(v float64) {
	step := 1 / float64(len(s))
	t := step
	i := 0

	for i < len(s) && v >= t {
		t += step
		i++
	}

	if i >= len(s) {
		i--
	}

	s[i].Insert(v)
}
