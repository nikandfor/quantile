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
