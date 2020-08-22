package quantile

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"testing"

	"github.com/beorn7/perks/quantile"
)

func TestQuantile(t *testing.T) {
	const N, E = 1000, 0.01

	for _, e := range []float64{0.1, 0.01, 0.001, 0.0001} {
		t.Logf("epsilon: %v  => %v size", e, size(e))
	}

	s := New(E)
	s2 := quantile.NewHighBiased(E)

	t.Logf("epsilon: %v  size %v", E, s.Size())

	n := N * s.Size()

	a := make([]float64, n)
	for i := 0; i < n; i++ {
		a[i] = rand.Float64()

		s.Insert(a[i])

		s2.Insert(a[i])
	}

	//	s.Query(0)
	//	t.Logf("compressed %v", s.dump())

	sort.Float64s(a)

	var l1, l2, ms float64
	for q := 0.0001; q <= 1.; q += 0.05 {
		gt := int(q * float64(len(a)))
		v := s.Query(q)
		v2 := s2.Query(q)

		l1 += sqr(v - a[gt])
		l2 += sqr(v2 - a[gt])
		ms++

		over := ""
		if abs(v-a[gt]) >= E {
			over = "  <----"
		}

		t.Logf("q %9.4f => %8.4f  gt %8.4f  diff %8.5f  s2 %8.4f  diff2 %8.5f%v", q, v, a[gt], v-a[gt], v2, v-v2, over)
	}

	l1 = math.Sqrt(l1 / ms)
	l2 = math.Sqrt(l2 / ms)

	t.Logf("l2(me-gt) %.4f   l2(s2-gt) %.4f", l1, l2)
}

func TestPrecision(t *testing.T) {
	const N = 1 << 10
	gt := make([]float64, N)

	for _, e := range []float64{0.1, 0.01} {
		s := New(e)

		for i := 0; i < N; i++ {
			gt[i] = rand.Float64()
			s.Insert(gt[i])

			if i >= 1000 && i&(i-1) == 0 {
				sort.Float64s(gt[:i])

				var d, ms float64

				for q := 0.00001; q <= 1; q += e {
					exp := gt[int(math.Ceil(q*float64(i)))]
					v := s.Query(q)

					if abs(v-exp) > 2*e {
						t.Logf("quantile out of epsilon: q %.4f  v %.4f  gt %.4f  diff %7.4f   eps %.4f  n %7v", q, v, exp, v-exp, e, i)
					} else {
						t.Logf("quantile                 q %.4f  v %.4f  gt %.4f  diff %7.4f   eps %.4f  n %7v", q, v, exp, v-exp, e, i)
					}

					d += sqr(v - exp)
					ms++
				}

				d = math.Sqrt(d / ms)

				if d > e {
					t.Errorf("mean error of epsilon: %.4f   eps %.4f", d, e)
				}
			}
		}
	}
}

func TestDebug(t *testing.T) {
	t.Skip()

	const N, E = 5000, 0.01

	s := New(E)

	t.Logf("eps %v   size %v   iterations %v", E, s.Size(), N)

	n := N * s.Size()
	gt := make([]float64, n)

	check := func(n int) bool {
		sort.Float64s(gt[:n])

		for q := 0.0001; q <= 1.; q += 0.05 {
			i := int(math.Ceil(q * float64(n)))
			v := s.Query(q)

			if abs(v-gt[i]) >= 2*E {
				return false
			}
		}

		return true
	}

	for i := 0; i < n; i++ {
		gt[i] = rand.Float64()

		s.Insert(gt[i])

		if i > 2*s.Size() && !check(i) {
			t.Logf("stop at n %v = %v * %v", i, i/s.Size(), s.Size())
			n = i
			break
		}
	}

	sort.Float64s(gt[:n])

	var l1, ms float64
	for q := 0.0001; q <= 1.; q += 0.0499 {
		i := int(math.Ceil(q * float64(n)))
		v := s.Query(q)

		var p, n float64 = -1, -1
		if i > 0 {
			p = gt[i-1]
		}
		if i+1 < len(gt) {
			n = gt[i+1]
		}

		l1 += sqr(v - gt[i])
		ms++

		over := ""
		if abs(v-gt[i]) >= E {
			over = " <-----"
		}

		t.Logf("q %9.4f => %8.4f  gt %8.4f  diff %8.5f  prev %8.4f  next %8.4f%v", q, v, gt[i], v-gt[i], p, n, over)
	}

	l1 = math.Sqrt(l1 / ms)

	t.Logf("l2(me-gt) %.4f", l1)

	// dump
	f, err := os.Create("/tmp/quantile.dump")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	s.dump(f)

	f2, err := os.Create("/tmp/quantile_gt.dump")
	if err != nil {
		panic(err)
	}
	defer f2.Close()

	fmt.Fprintf(f2, "gt %v\n", s.Size())
	var q float64
	for i := 0; i < s.Size(); i++ {
		q += 1 / float64(s.Size())

		j := int(math.Ceil(q * float64(n)))
		if j == n {
			j--
		}

		fmt.Fprintf(f2, "  %4d: %.4f  q %.4f\n", i, gt[j], q)
	}
}

func BenchmarkQuantile(b *testing.B) {
	b.ReportAllocs()

	s := New(0.01)

	for i := 0; i < b.N; i++ {
		v := rand.Float64()

		s.Insert(v)
	}
}

func BenchmarkBeorn7Quantile(b *testing.B) {
	b.ReportAllocs()

	s := quantile.NewHighBiased(0.01)

	for i := 0; i < b.N; i++ {
		v := rand.Float64()

		s.Insert(v)
	}
}

func sqr(f float64) float64 { return f * f }
