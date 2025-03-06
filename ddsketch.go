package quantile

// Based on clickhouse implementation.
//
// https://github.com/ClickHouse/ClickHouse/blob/master/src/AggregateFunctions/DDSketch.h

import (
	"fmt"
	"io"
	"math"
	"strings"
)

type (
	DDLog struct {
		pos, neg ddstorage
		zeros    float64

		gamma, multiplier        float64
		minPossible, maxPossible float64
	}

	ddstorage struct {
		bins   []float32
		offset int
		total  float64
	}
)

func NewDDLog(relAcc float64) *DDLog {
	gamma := (1 + relAcc) / (1 - relAcc)

	return &DDLog{
		gamma:       gamma,
		multiplier:  1 / math.Log(gamma),
		minPossible: math.SmallestNonzeroFloat64 * gamma,
		maxPossible: math.MaxFloat64 / gamma,
	}
}

func (s *DDLog) Query(q float64) float64 {
	total := s.neg.total + s.zeros + s.pos.total
	if total == 0 {
		return 0
	}

	if q <= 0 {
		switch {
		case s.neg.total != 0:
			return -s.unkey(s.neg.offset + len(s.neg.bins))
		case s.zeros != 0:
			return 0
		default:
			return s.unkey(s.pos.offset)
		}
	}
	if q >= 1 {
		switch {
		case s.pos.total != 0:
			return s.unkey(s.pos.offset + len(s.pos.bins))
		case s.zeros != 0:
			return 0
		default:
			return -s.unkey(s.neg.offset)
		}
	}

	target := q * total

	var limit float64
	b := &s.pos

	//	log.Printf("query %.3f %6.1f of %6.1f  (%.1f + %.1f + %.1f)", q, target, total, s.neg.total, s.zeros, s.pos.total)

	switch {
	case target < s.neg.total:
		b = &s.neg
		limit = s.neg.total - target // going from 0 to -Inf
	case total-target <= s.pos.total:
		limit = target - (s.neg.total + s.zeros)
	default:
		return 0
	}

	var cum float64
	i := 0

	for i < len(b.bins) {
		cum += float64(b.bins[i])
		if cum > limit {
			break
		}

		i++
	}

	v := s.unkey(b.offset + i)

	//	log.Printf("got index %3d + %3d of %3d  with cum %.3f of %.3f   v %9.3f", b.offset, i, len(b.bins), cum, limit, v)

	if b == &s.neg {
		v = -v
	}

	return v
}

func (s *DDLog) Insert(v float64) {
	s.InsertWeight(v, 1)
}

func (s *DDLog) InsertWeight(v float64, w float32) {
	b := &s.pos

	if v < 0 {
		v = -v
		b = &s.neg
	}

	if v < s.minPossible {
		s.zeros += float64(w)
		return
	}

	key := s.key(v)

	//	log.Printf("insert %.3v  key %d  off %d  bins %d", v, key, b.offset, len(b.bins))

	if len(b.bins) == 0 {
		b.offset = key
	}
	if key < b.offset {
		low := key &^ 0x7

		b.bins = append(make([]float32, b.offset-low), b.bins...)
		b.offset = low
	}
	if key >= b.offset+cap(b.bins) {
		end := b.offset + cap(b.bins)

		b.bins = append(b.bins[:cap(b.bins)], make([]float32, key-end+1)...)
	}
	if key >= b.offset+len(b.bins) {
		b.bins = b.bins[:key-b.offset+1]
	}

	b.bins[key-b.offset] += w
	b.total += float64(w)
}

func (s *DDLog) key(v float64) int {
	return int(math.Log(v) * s.multiplier)
}

func (s *DDLog) unkey(key int) float64 {
	return math.Exp(float64(key) / s.multiplier)
}

func (s *DDLog) dump() string {
	var b strings.Builder

	fmt.Fprintf(&b, "dd totals (neg+zero+pos): %.2f + %.2f + %.2f\n", s.neg.total, s.zeros, s.pos.total)

	if s.neg.total != 0 {
		fmt.Fprintf(&b, "negative (off %3d)\n", s.neg.offset)
		s.dumpBins(&b, &s.neg)
	}
	if s.pos.total != 0 {
		fmt.Fprintf(&b, "positive (off %3d)\n", s.pos.offset)
		s.dumpBins(&b, &s.pos)
	}

	return b.String()
}

func (s *DDLog) dumpBins(w io.Writer, b *ddstorage) {
	for i, wg := range b.bins {
		if wg == 0 {
			continue
		}

		fmt.Fprintf(w, "i %3d  v %.3f  w %.1f\n", b.offset+i, s.unkey(b.offset+i), wg)
	}
}
