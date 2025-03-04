package quantile

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
)

var (
	tdbenchW = []int{128, 512, 2048}
	tdbenchE = []float32{0.1, 0.03, 0.01}
)

func TestTDigest(tb *testing.T) {
	const W = 16

	e := NewExact()
	s := NewExtremesBiased(0.1, W)

	assertTDigest(tb, e, s, 0)

	e.Insert(5)
	s.Insert(5)

	assertTDigest(tb, e, s, 0)
	assertTDigest(tb, e, s, 1)
	assertTDigest(tb, e, s, 0.5)

	for i := range W + 1 {
		v := float64(i)
		e.Insert(v)
		s.Insert(v)
	}

	assertTDigest(tb, e, s, 0)
	assertTDigest(tb, e, s, 1)
	assertTDigest(tb, e, s, 0.5)

	if tb.Failed() {
		tb.Logf("dump\n%v", s.dump())
	}
}

func BenchmarkTDigestInsert(tb *testing.B) {
	for _, W := range tdbenchW {
		for _, E := range tdbenchE {
			tb.Run(fmt.Sprintf("W%d_E%.3f", W, E), func(tb *testing.B) {
				tb.ReportAllocs()

				src := rand.NewChaCha8([32]byte{})
				r := rand.New(src)

				//	e := NewExact()
				s := NewExtremesBiased(E, W)

				for i := 0; i < tb.N; i++ {
					v := r.Float64()
					//		e.Insert(v)
					s.Insert(v)
				}

				//	assertTDigest(tb, e, s, 0)
				//	assertTDigest(tb, e, s, 0.1)
				//	assertTDigest(tb, e, s, 0.5)
				//	assertTDigest(tb, e, s, 0.9)
				//	assertTDigest(tb, e, s, 1)

				tb.Logf("stats: compressions %v / %v,  average reduction %v / %v", s.Compressions, s.BruteCompressions, s.ElementsReduced, s.size)

				if tb.Failed() {
					tb.Logf("dump\n%v", s.dump())
				}
			})
		}
	}
}

func BenchmarkTDigestQuery(tb *testing.B) {
	for _, W := range tdbenchW {
		tb.Run(fmt.Sprintf("W%d", W), func(tb *testing.B) {
			tb.ReportAllocs()

			src := rand.NewChaCha8([32]byte{})
			r := rand.New(src)

			e := NewExact()
			s := NewExtremesBiased(0.1, W)

			for range int(1e6) {
				v := r.Float64()
				e.Insert(v)
				s.Insert(v)
			}

			tb.ResetTimer()

			for i := 0; i < tb.N; i++ {
				q := r.Float64()
				_ = s.Query(q)
			}

			assertTDigest(tb, e, s, 0)
			assertTDigest(tb, e, s, 0.1)
			assertTDigest(tb, e, s, 0.5)
			assertTDigest(tb, e, s, 0.9)
			assertTDigest(tb, e, s, 1)

			if tb.Failed() {
				tb.Logf("dump\n%v", s.dump())
			}
		})
	}
}

func assertTDigest[Inv Invariant](tb testing.TB, e *Exact, s *TDigest[Inv], q float64) {
	tb.Helper()

	ext := e.Query(q)
	v := s.Query(q)

	if math.Abs(v-ext) > 0.5001 {
		tb.Errorf("q %.2f => %.3f  wanted %.3f", q, v, ext)
	}
}
