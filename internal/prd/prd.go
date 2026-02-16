package prd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type PRD struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	BranchName  string       `json:"branchName"`
	UserStories []UserStory  `json:"userStories"`
	Metadata    *PRDMetadata `json:"metadata,omitempty"`
}

type UserStory struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Description        string   `json:"description,omitempty"`
	AcceptanceCriteria []string `json:"acceptanceCriteria,omitempty"`
	Priority           int      `json:"priority"`
	Passes             bool     `json:"passes"`
	Notes              string   `json:"notes,omitempty"`
	Iterations         int      `json:"iterations,omitempty"`
}

type PRDMetadata struct {
	UpdatedAt string `json:"updatedAt,omitempty"`
}

func Load(path string) (*PRD, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading prd.json: %w", err)
	}
	var p PRD
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing prd.json: %w", err)
	}
	return &p, nil
}

func (p *PRD) Save(path string) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling prd.json: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

// CurrentStory returns the highest priority user story where passes=false.
// Lower priority number = higher priority. Returns nil if all stories pass.
func (p *PRD) CurrentStory() *UserStory {
	var candidates []UserStory
	for _, s := range p.UserStories {
		if !s.Passes {
			candidates = append(candidates, s)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Priority < candidates[j].Priority
	})
	return &candidates[0]
}

func (p *PRD) CompletedCount() int {
	n := 0
	for _, s := range p.UserStories {
		if s.Passes {
			n++
		}
	}
	return n
}

func (p *PRD) RemainingCount() int {
	return p.TotalCount() - p.CompletedCount()
}

func (p *PRD) TotalCount() int {
	return len(p.UserStories)
}
