package main

import (
	"encoding/json"
	"fmt"
	"io"
)

type QuestSet struct {
	FirstQuest       Single
	SuccessiveQuests []Single
}

func (s *QuestSet) Unmarshal(rd io.Reader) error {
	if err := s.FirstQuest.Unmarshal(rd); err != nil {
		return fmt.Errorf("first -> %w", err)
	}
	var length U16
	if err := length.Unmarshal(rd); err != nil {
		return fmt.Errorf("length -> %w", err)
	}
	s.SuccessiveQuests = make([]Single, length)
	for i := range s.SuccessiveQuests {
		if err := s.SuccessiveQuests[i].Unmarshal(rd); err != nil {
			return fmt.Errorf("[%d/%d] successive -> %w", i, len(s.SuccessiveQuests), err)
		}
	}
	return nil
}

func (s *QuestSet) Marshal(wt io.Writer) error {
	if err := s.FirstQuest.Marshal(wt); err != nil {
		return fmt.Errorf("first -> %w", err)
	}

	if len(s.SuccessiveQuests) > (1<<16 - 1) {
		return fmt.Errorf("too much items")
	}
	length := U16(len(s.SuccessiveQuests))
	if err := length.Marshal(wt); err != nil {
		return fmt.Errorf("length -> %w", err)
	}
	for i := range s.SuccessiveQuests {
		if err := s.SuccessiveQuests[i].Marshal(wt); err != nil {
			return fmt.Errorf("[%d/%d] successive -> %w", i, len(s.SuccessiveQuests), err)
		}
	}
	return nil
}

type Quests []QuestSet

func (s *Quests) Unmarshal(rd io.Reader) error {
	var length U32
	if err := length.Unmarshal(rd); err != nil {
		return fmt.Errorf("length -> %w", err)
	}
	*s = make([]QuestSet, length)
	for i := range *s {
		if err := (*s)[i].Unmarshal(rd); err != nil {
			return fmt.Errorf("[%d/%d] questSet -> %s", i, len(*s), err)
		}
	}
	return nil
}

func (s *Quests) Marshal(wt io.Writer) error {
	length := U32(len(*s))
	if err := length.Marshal(wt); err != nil {
		return fmt.Errorf("length -> %w", err)
	}
	for i := range *s {
		if err := (*s)[i].Marshal(wt); err != nil {
			return fmt.Errorf("[%d/%d] questSet -> %s", i, len(*s), err)
		}
	}
	return nil
}

func parseQuests(rd io.Reader, wt io.Writer) error {
	var h Quests

	if err := h.Unmarshal(rd); err != nil {
		return err
	}

	encoder := json.NewEncoder(wt)
	encoder.SetIndent("", "\t")
	return encoder.Encode(&h)
}

func compileQuests(rd io.Reader, wt io.Writer) error {
	var h Quests

	e := json.NewDecoder(rd).Decode(&h)
	if e != nil {
		return e
	}

	return h.Marshal(wt)
}
