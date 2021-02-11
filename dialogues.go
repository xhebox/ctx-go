package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Dialogue struct {
	Content Single
	ID      string
	Desc    WideString
}

func (s *Dialogue) Unmarshal(rd io.Reader, desc bool) error {
	if err := s.Content.Unmarshal(rd); err != nil {
		return fmt.Errorf("content -> %w", err)
	}
	var length U32
	if err := length.Unmarshal(rd); err != nil {
		return fmt.Errorf("length -> %w", err)
	}
	buf := U8S(make([]byte, length))
	if err := buf.Unmarshal(rd); err != nil {
		return fmt.Errorf("ID[%d] -> %w", length, err)
	}
	s.ID = string(buf)

	if desc {
		if err := s.Desc.Unmarshal(rd); err != nil {
			return fmt.Errorf("desc -> %w", err)
		}
	}
	return nil
}

func (s *Dialogue) Marshal(wt io.Writer, desc bool) error {
	if err := s.Content.Marshal(wt); err != nil {
		return fmt.Errorf("content -> %w", err)
	}
	length := U32(len(s.ID))
	if err := length.Marshal(wt); err != nil {
		return fmt.Errorf("length -> %w", err)
	}
	buf := U8S(s.ID)
	if err := buf.Marshal(wt); err != nil {
		return fmt.Errorf("ID -> %w", err)
	}

	if desc {
		if err := s.Desc.Marshal(wt); err != nil {
			return fmt.Errorf("desc -> %w", err)
		}
	}
	return nil
}

type Dialogues struct {
	EnableDesc U8
	Dialogues  []Dialogue
}

func (s *Dialogues) Unmarshal(rd io.Reader) error {
	if err := s.EnableDesc.Unmarshal(rd); err != nil {
		return fmt.Errorf("wide -> %w", err)
	}
	var length U32
	if err := length.Unmarshal(rd); err != nil {
		return fmt.Errorf("length -> %w", err)
	}
	s.Dialogues = make([]Dialogue, length)
	for i := range s.Dialogues {
		if err := s.Dialogues[i].Unmarshal(rd, s.EnableDesc != 0); err != nil {
			return fmt.Errorf("[%d/%d] dialogueSet -> %s", i, len(s.Dialogues), err)
		}
	}
	return nil
}

func (s *Dialogues) Marshal(wt io.Writer) error {
	if err := s.EnableDesc.Marshal(wt); err != nil {
		return fmt.Errorf("wide -> %w", err)
	}
	length := U32(len(s.Dialogues))
	if err := length.Marshal(wt); err != nil {
		return fmt.Errorf("length -> %w", err)
	}
	for i := range s.Dialogues {
		if err := s.Dialogues[i].Marshal(wt, s.EnableDesc != 0); err != nil {
			return fmt.Errorf("[%d/%d] dialogueSet -> %s", i, len(s.Dialogues), err)
		}
	}
	return nil
}

func parseDialogues(rd io.Reader, wt io.Writer) error {
	var h Dialogues

	if err := h.Unmarshal(rd); err != nil {
		sb := new(strings.Builder)
		encoder := json.NewEncoder(sb)
		encoder.SetIndent("", "\t")
		encoder.Encode(&h)
		fmt.Printf("%s\n", sb.String())
		return err
	}

	encoder := json.NewEncoder(wt)
	encoder.SetIndent("", "\t")
	return encoder.Encode(&h)
}

func compileDialogues(rd io.Reader, wt io.Writer) error {
	var h Dialogues

	err := json.NewDecoder(rd).Decode(&h)
	if err != nil {
		return err
	}

	return h.Marshal(wt)
}
