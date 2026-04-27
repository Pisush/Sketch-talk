package questions

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Question is a single audience question.
type Question struct {
	Text      string
	Asker     string // optional name/handle
	CreatedAt time.Time
}

// Store holds all questions submitted during a session.
type Store struct {
	mu        sync.RWMutex
	questions []Question
}

// NewStore returns an empty Store.
func NewStore() *Store {
	return &Store{}
}

// Add records a new question.
func (s *Store) Add(text, asker string) Question {
	q := Question{
		Text:      strings.TrimSpace(text),
		Asker:     strings.TrimSpace(asker),
		CreatedAt: time.Now(),
	}
	s.mu.Lock()
	s.questions = append(s.questions, q)
	s.mu.Unlock()
	return q
}

// All returns a snapshot of all questions.
func (s *Store) All() []Question {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Question, len(s.questions))
	copy(out, s.questions)
	return out
}

// Count returns the number of questions.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.questions)
}

// Export writes all questions to a text file.
func (s *Store) Export(path string) error {
	qs := s.All()
	if len(qs) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("=== Audience Questions ===\n\n")
	for i, q := range qs {
		ts := q.CreatedAt.Format("15:04:05")
		who := q.Asker
		if who == "" {
			who = "Anonymous"
		}
		sb.WriteString(fmt.Sprintf("[%s] Q%d — %s\n%s\n\n", ts, i+1, who, q.Text))
	}

	return os.WriteFile(path, []byte(sb.String()), 0644)
}
