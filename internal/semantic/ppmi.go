package semantic

import (
	"math"
	"sort"
)

type scoredTerm struct {
	term  string
	score float64
}

// BuildThesaurus computes PPMI scores for all token pairs and returns a map
// from each token to its top maxSynonyms most co-occurring neighbours.
//
// PPMI(x,y) = max(0, log2( P(x,y) / (P(x) * P(y)) ))
// where:
//   P(x)   = termFreq[x] / totalTokens
//   P(x,y) = coOccur[x][y] / (totalTokens * windowSize)
func (b *Builder) BuildThesaurus(maxSynonyms int) map[string][]string {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.totalTokens == 0 {
		return nil
	}

	total := float64(b.totalTokens)
	window := float64(windowSize)
	thesaurus := make(map[string][]string)

	for x, neighbors := range b.coOccur {
		px := float64(b.termFreq[x]) / total
		if px == 0 {
			continue
		}

		var scored []scoredTerm
		for y, cnt := range neighbors {
			py := float64(b.termFreq[y]) / total
			if py == 0 {
				continue
			}
			pxy := float64(cnt) / (total * window)
			pmi := math.Log2(pxy / (px * py))
			if pmi > 0 {
				scored = append(scored, scoredTerm{term: y, score: pmi})
			}
		}

		if len(scored) == 0 {
			continue
		}

		sort.Slice(scored, func(i, j int) bool {
			return scored[i].score > scored[j].score
		})

		limit := maxSynonyms
		if limit > len(scored) {
			limit = len(scored)
		}
		syns := make([]string, limit)
		for i := 0; i < limit; i++ {
			syns[i] = scored[i].term
		}
		thesaurus[x] = syns
	}

	return thesaurus
}
