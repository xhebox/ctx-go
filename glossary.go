package main

import (
	"encoding/json"
	"fmt"
	"io"
)

type GlossarySet struct {
	Un1        U32
	Un2        U32
	Category   WideString
	Glossaries []Single
}

func (s *GlossarySet) Unmarshal(rd io.Reader) error {
	if err := s.Un1.Unmarshal(rd); err != nil {
		return fmt.Errorf("un1 -> %w", err)
	}
	if err := s.Un2.Unmarshal(rd); err != nil {
		return fmt.Errorf("un2 -> %w", err)
	}
	if err := s.Category.Unmarshal(rd); err != nil {
		return fmt.Errorf("category -> %w", err)
	}
	var length U32
	if err := length.Unmarshal(rd); err != nil {
		return fmt.Errorf("length -> %w", err)
	}
	s.Glossaries = make([]Single, length)
	for i := range s.Glossaries {
		if err := s.Glossaries[i].Unmarshal(rd); err != nil {
			return fmt.Errorf("[%d/%d] successive -> %w", i, len(s.Glossaries), err)
		}
	}
	return nil
}

func (s *GlossarySet) Marshal(wt io.Writer) error {
	if err := s.Un1.Marshal(wt); err != nil {
		return fmt.Errorf("un1 -> %w", err)
	}
	if err := s.Un2.Marshal(wt); err != nil {
		return fmt.Errorf("un2 -> %w", err)
	}
	if err := s.Category.Marshal(wt); err != nil {
		return fmt.Errorf("category -> %w", err)
	}
	length := U32(len(s.Glossaries))
	if err := length.Marshal(wt); err != nil {
		return fmt.Errorf("length -> %w", err)
	}
	for i := range s.Glossaries {
		if err := s.Glossaries[i].Marshal(wt); err != nil {
			return fmt.Errorf("[%d/%d] successive -> %w", i, len(s.Glossaries), err)
		}
	}
	return nil
}

type Glossary []GlossarySet

func (s *Glossary) Unmarshal(rd io.Reader) error {
	var length U32
	if err := length.Unmarshal(rd); err != nil {
		return fmt.Errorf("length -> %w", err)
	}
	*s = make([]GlossarySet, length)
	for i := range *s {
		if err := (*s)[i].Unmarshal(rd); err != nil {
			return fmt.Errorf("[%d/%d] glossarySet -> %s", i, len(*s), err)
		}
	}
	return nil
}

func (s *Glossary) Marshal(wt io.Writer) error {
	length := U32(len(*s))
	if err := length.Marshal(wt); err != nil {
		return fmt.Errorf("length -> %w", err)
	}
	for i := range *s {
		if err := (*s)[i].Marshal(wt); err != nil {
			return fmt.Errorf("[%d/%d] glossarySet -> %s", i, len(*s), err)
		}
	}
	return nil
}

func parseGlossary(rd io.Reader, wt io.Writer) error {
	var h Glossary

	if err := h.Unmarshal(rd); err != nil {
		return err
	}

	encoder := json.NewEncoder(wt)
	encoder.SetIndent("", "\t")
	return encoder.Encode(&h)
}

func compileGlossary(rd io.Reader, wt io.Writer) error {
	var h Glossary

	err := json.NewDecoder(rd).Decode(&h)
	if err != nil {
		return err
	}

	return h.Marshal(wt)
}
