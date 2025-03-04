package kll

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"testing"
)

var (
	benchW = []int{32, 256, 2048}
	benchD = []int{6, 10}
)

func TestKLL(tb *testing.T) {
	const W, D = 32, 10

	s := New(W, D)

	assertKLL(tb, s, 0)

	s.Insert(5)

	assertKLL(tb, s, 0)
	assertKLL(tb, s, 1)
	assertKLL(tb, s, 0.5)

	for i := range W - 1 {
		s.Insert(float64(i))
	}

	assertKLL(tb, s, 0)
	assertKLL(tb, s, 1)
	assertKLL(tb, s, 0.5)

	if tb.Failed() {
		tb.Logf("dump\n%v", s.dump())
	}
}

func BenchmarkKLLInsert(tb *testing.B) {
	for _, W := range benchW {
		for _, D := range benchD {
			tb.Run(fmt.Sprintf("W%d_D%d", W, D), func(tb *testing.B) {
				tb.ReportAllocs()

				src := rand.NewChaCha8([32]byte{})
				r := rand.New(src)

				s := New(W, D)

				for i := 0; i < tb.N; i++ {
					v := r.Float64()
					s.Insert(v)
				}

				assertKLL(tb, s, 0)
				assertKLL(tb, s, 0.1)
				assertKLL(tb, s, 0.5)
				assertKLL(tb, s, 0.9)
				assertKLL(tb, s, 1)

				if tb.Failed() {
					tb.Logf("dump\n%v", s.dump())
				}
			})
		}
	}
}

func BenchmarkKLLQuery(tb *testing.B) {
	for _, W := range benchW {
		for _, D := range benchD {
			tb.Run(fmt.Sprintf("W%d_D%d", W, D), func(tb *testing.B) {
				tb.ReportAllocs()

				src := rand.NewChaCha8([32]byte{})
				r := rand.New(src)

				s := New(W, D)

				for range int(1e6) {
					v := r.Float64()
					s.Insert(v)
				}

				tb.ResetTimer()

				for i := 0; i < tb.N; i++ {
					q := r.Float64()
					_ = s.Query(q)
				}

				assertKLL(tb, s, 0)
				assertKLL(tb, s, 0.1)
				assertKLL(tb, s, 0.5)
				assertKLL(tb, s, 0.9)
				assertKLL(tb, s, 1)

				if tb.Failed() {
					tb.Logf("dump\n%v", s.dump())
				}
			})
		}
	}
}

func assertKLL(tb testing.TB, s *KLL, q float64) {
	tb.Helper()

	v := s.Query(q)
	ext := exactKLL(s, q)

	if math.Abs(v-ext) >= 0.1 {
		tb.Errorf("q %.2f => %.3f  wanted %.3f", q, v, ext)
	}
}

func exactKLL(s *KLL, q float64) float64 {
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
