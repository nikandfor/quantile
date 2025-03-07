package quantile

import "sort"

type TDMulti []*TDigest

func (ss TDMulti) Query(q float64) float64 {
	var res [1]float64

	ss.QueryMulti([]float64{q}, res[:])

	return res[0]
}

func (ss TDMulti) QueryMulti(qs, res []float64) {
	if len(qs) == 0 || len(ss) == 0 {
		for i := range qs {
			res[i] = 0
		}

		return
	}

	copy(res, qs)
	sort.Float64s(res[:len(qs)])

	for _, s := range ss {
		if !s.sorted {
			s.sort()
		}
	}

	var total, sum, prev float64
	var n int

	for _, s := range ss {
		for _, w := range s.w[:s.i] {
			total += float64(w)
		}

		n += s.i
		s.j = 0
	}

	if n == 0 {
		for i := range qs {
			res[i] = 0
		}

		return
	}

	first := func() *TDigest {
		var f *TDigest

		for _, s := range ss {
			if s.j >= s.i {
				continue
			}

			if f == nil || s.v[s.j] < f.v[f.j] {
				f = s
			}
		}

		return f
	}

	var last *TDigest

	qi := 0

	target := res[qi] * total
	prevV := first().v[0]

	for {
		s := first()
		if s == nil {
			break
		}

		cur := sum + 0.5*float64(s.w[s.j])

		//	log.Printf("querymulti %.2f  i %2d  cur %.3f / %.3f  v %.2f  w %.1f", res[qi], s.j, cur, target, s.v[s.j], s.w[s.j])

		if cur >= target {
			l := prev
			r := cur

			switch {
			case target <= l:
				res[qi] = prevV
			case target >= r:
				res[qi] = s.v[s.j]
			default:
				res[qi] = s.interpolate(target, l, r, prevV, s.v[s.j])
			}

			qi++

			if qi == len(qs) {
				break
			}

			target = res[qi] * total

			continue
		}

		sum += float64(s.w[s.j])

		prev = cur
		prevV = s.v[s.j]

		s.j++
		last = s
	}

	for qi < len(qs) {
		res[qi] = last.v[last.i-1]
		qi++
	}
}
