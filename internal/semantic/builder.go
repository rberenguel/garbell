package semantic

import "sync"

const windowSize = 10

// Builder accumulates token co-occurrence statistics across multiple documents.
type Builder struct {
	mu          sync.Mutex
	termFreq    map[string]int
	coOccur     map[string]map[string]int
	totalTokens int
}

// NewBuilder returns an initialised Builder.
func NewBuilder() *Builder {
	return &Builder{
		termFreq: make(map[string]int),
		coOccur:  make(map[string]map[string]int),
	}
}

// AddDocument tokenises text and updates term-frequency and co-occurrence counts.
// It is safe to call concurrently from multiple goroutines.
func (b *Builder) AddDocument(text string) {
	tokens := Tokenize(text)
	if len(tokens) == 0 {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.totalTokens += len(tokens)

	for i, tok := range tokens {
		b.termFreq[tok]++

		start := i - windowSize
		if start < 0 {
			start = 0
		}
		end := i + windowSize
		if end >= len(tokens) {
			end = len(tokens) - 1
		}

		if b.coOccur[tok] == nil {
			b.coOccur[tok] = make(map[string]int)
		}
		for j := start; j <= end; j++ {
			if j == i {
				continue
			}
			b.coOccur[tok][tokens[j]]++
		}
	}
}
