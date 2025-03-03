package quantile

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"testing"
)

func TestQuantileFirst(tb *testing.T) {
	const W, D = 32, 10

	s := New(W, D)

	assertInEpsilon(tb, s, 0)

	s.Insert(5)

	assertInEpsilon(tb, s, 0)
	assertInEpsilon(tb, s, 1)
	assertInEpsilon(tb, s, 0.5)

	for i := range W - 1 {
		s.Insert(float64(i))
	}

	assertInEpsilon(tb, s, 0)
	assertInEpsilon(tb, s, 1)
	assertInEpsilon(tb, s, 0.5)

	if tb.Failed() {
		for l := range D {
			st, end := s.startEnd(l)

			tb.Logf("dump l %2x: %v", l, s.v[st:end])
		}
	}
}

func BenchmarkInsert(tb *testing.B) {
	for _, W := range []int{32, 256, 2048} {
		for _, D := range []int{10, 12, 14} {
			tb.Run(fmt.Sprintf("W%d_D%d", W, D), func(tb *testing.B) {
				tb.ReportAllocs()

				src := rand.NewChaCha8([32]byte{})
				r := rand.New(src)

				s := New(W, D)

				for tb.Loop() {
					v := r.Float64()
					s.Insert(v)
				}

				assertInEpsilon(tb, s, 0)
				assertInEpsilon(tb, s, 0.1)
				assertInEpsilon(tb, s, 0.5)
				assertInEpsilon(tb, s, 0.9)
				assertInEpsilon(tb, s, 1)

				if tb.Failed() {
					for l := range D {
						st, end := s.startEnd(l)

						tb.Logf("dump l %2x: %.3f", l, s.v[st:end])
					}
				}
			})
		}
	}
}

func BenchmarkQuery(tb *testing.B) {
	for _, W := range []int{32, 256, 2048} {
		for _, D := range []int{10, 12, 14} {
			tb.Run(fmt.Sprintf("W%d_D%d", W, D), func(tb *testing.B) {
				tb.ReportAllocs()

				src := rand.NewChaCha8([32]byte{})
				r := rand.New(src)

				s := New(W, D)

				for range W * D * D * D * D {
					v := r.Float64()
					s.Insert(v)
				}

				tb.ResetTimer()

				for tb.Loop() {
					q := r.Float64()
					_ = s.Query(q)
				}

				assertInEpsilon(tb, s, 0)
				assertInEpsilon(tb, s, 0.1)
				assertInEpsilon(tb, s, 0.5)
				assertInEpsilon(tb, s, 0.9)
				assertInEpsilon(tb, s, 1)

				if tb.Failed() {
					for l := range D {
						st, end := s.startEnd(l)

						tb.Logf("dump l %2x: %.3f", l, s.v[st:end])
					}
				}
			})
		}
	}
}

func assertInEpsilon(tb testing.TB, s *Stream, q float64) {
	tb.Helper()

	v := s.Query(q)
	ext := exact(s, q)

	if math.Abs(v-ext) >= 0.1 {
		tb.Errorf("q %.2f => %.3f  wanted %.3f", q, v, ext)
	}
}

func exact(s *Stream, q float64) float64 {
	a := make([]float64, 0, s.width*s.depth)

	for l := range s.depth {
		st, end := s.startEnd(l)

		a = append(a, s.v[st:end]...)
	}

	if len(a) == 0 {
		return 0
	}

	sort.Float64s(a)

	i := int(q * float64(len(a)))

	//	log.Printf("exact %.3f: %.3f  at %d/%d  %.3f <> %.3f  a: %.3f", q, a[i], i, len(a), a[i-5:i], a[i:i+5], a)

	if i == len(a) {
		i--
	}

	return a[i]
}
