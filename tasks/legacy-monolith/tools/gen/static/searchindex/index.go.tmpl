// Package searchindex is the in-process inverted index behind storefront
// search. It is rebuilt from the catalog on boot.
package searchindex

import (
	"sort"
	"strings"
	"sync"
	"unicode"
)

// Tokenize lower-cases s, splits on anything that is not a letter or digit,
// and drops stopwords and one-letter tokens.
func Tokenize(s string) []string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := fields[:0]
	for _, f := range fields {
		if len(f) < 2 || stopwords[f] {
			continue
		}
		out = append(out, f)
	}
	return out
}

// Hit is one search result.
type Hit struct {
	ID    string
	Score int
}

// Index maps tokens to documents.
type Index struct {
	mu       sync.RWMutex
	postings map[string]map[string]int // token -> doc id -> term frequency
	docs     map[string]*Document
}

func NewIndex() *Index {
	return &Index{postings: map[string]map[string]int{}, docs: map[string]*Document{}}
}

// Add indexes d, replacing any previous version.
func (ix *Index) Add(d *Document) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.removeLocked(d.ID)
	cp := d.Clone()
	cp.Status = DocumentStatusIndexed
	ix.docs[d.ID] = cp
	weights := map[string]int{}
	for _, t := range Tokenize(d.Title) {
		weights[t] += 3 // title matches count triple
	}
	for _, t := range Tokenize(d.Body) {
		weights[t]++
	}
	for _, tag := range d.Tags {
		for _, t := range Tokenize(tag) {
			weights[t] += 2
		}
	}
	for t, w := range weights {
		if ix.postings[t] == nil {
			ix.postings[t] = map[string]int{}
		}
		ix.postings[t][d.ID] = w
	}
}

// Remove drops a document.
func (ix *Index) Remove(id string) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.removeLocked(id)
}

func (ix *Index) removeLocked(id string) {
	if _, ok := ix.docs[id]; !ok {
		return
	}
	delete(ix.docs, id)
	for t, p := range ix.postings {
		delete(p, id)
		if len(p) == 0 {
			delete(ix.postings, t)
		}
	}
}

// Len is the number of indexed documents.
func (ix *Index) Len() int {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return len(ix.docs)
}

// Search returns documents matching every query token, best first. Boost is
// added once per matching document.
func (ix *Index) Search(q string, limit int) []Hit {
	toks := Tokenize(q)
	if len(toks) == 0 {
		return nil
	}
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	scores := map[string]int{}
	for i, t := range toks {
		p := ix.postings[t]
		next := map[string]int{}
		for id, w := range p {
			if i == 0 {
				next[id] = w
			} else if s, ok := scores[id]; ok {
				next[id] = s + w
			}
		}
		scores = next
	}
	hits := make([]Hit, 0, len(scores))
	for id, s := range scores {
		hits = append(hits, Hit{ID: id, Score: s + ix.docs[id].Boost})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].ID < hits[j].ID
	})
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}
