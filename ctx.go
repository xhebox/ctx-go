package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type CtxSingle struct {
	ID  U32
	Str WideString
}

func (i *CtxSingle) Unmarshal(rd io.Reader) error {
	err := i.ID.Unmarshal(rd)
	if err != nil {
		return fmt.Errorf("id -> %w", err)
	}
	err = i.Str.Unmarshal(rd)
	if err != nil {
		return fmt.Errorf("str -> %w", err)
	}
	return nil
}

func (i *CtxSingle) Marshal(wt io.Writer) error {
	err := i.ID.Marshal(wt)
	if err != nil {
		return fmt.Errorf("id -> %w", err)
	}
	err = i.Str.Marshal(wt)
	if err != nil {
		return fmt.Errorf("str -> %w", err)
	}
	return nil
}

func (i *CtxSingle) Size() int {
	return 4 + i.Str.Size()
}

type CtxLangIndex struct {
	Str WideString
	Off U32
}

func (i *CtxLangIndex) Unmarshal(rd io.Reader) error {
	err := i.Str.Unmarshal(rd)
	if err != nil {
		return fmt.Errorf("str -> %w", err)
	}
	err = i.Off.Unmarshal(rd)
	if err != nil {
		return fmt.Errorf("off -> %w", err)
	}
	return nil
}

func (i *CtxLangIndex) Marshal(wt io.Writer) error {
	err := i.Str.Marshal(wt)
	if err != nil {
		return fmt.Errorf("str -> %w", err)
	}
	err = i.Off.Marshal(wt)
	if err != nil {
		return fmt.Errorf("off -> %w", err)
	}
	return nil
}

type Ctx struct {
	Un1        U32
	Un2        U32
	LangIndexs []CtxLangIndex
	Languages  [][]CtxSingle
}

func (s *Ctx) Unmarshal(rd io.Reader) error {
	if err := s.Un1.Unmarshal(rd); err != nil {
		return fmt.Errorf("un1 -> %w", err)
	}

	if err := s.Un2.Unmarshal(rd); err != nil {
		return fmt.Errorf("un2 -> %w", err)
	}

	var length U32
	if err := length.Unmarshal(rd); err != nil {
		return fmt.Errorf("len -> %w", err)
	}

	if length == 0 {
		return nil
	}

	s.LangIndexs = make([]CtxLangIndex, length)
	for k := range s.LangIndexs {
		if err := s.LangIndexs[k].Unmarshal(rd); err != nil {
			return fmt.Errorf("[%d/%d] langindex -> %w", k, len(s.LangIndexs), err)
		}
	}

	if s.LangIndexs[0].Off != 0 {
		return fmt.Errorf("unexpected 0 offset for the first language")
	}

	s.Languages = make([][]CtxSingle, length)
	for k := range s.LangIndexs {
		last := k == len(s.Languages)-1
		off := s.LangIndexs[k].Off
		for {
			if !last && off == s.LangIndexs[k+1].Off {
				break
			}

			var t CtxSingle
			err := t.Unmarshal(rd)
			if err != nil {
				if last && errors.Is(err, io.EOF) {
					return nil
				} else {
					total := U32(0)
					if !last {
						total = s.LangIndexs[k].Off
					}
					return fmt.Errorf("[%d/%d] single [%d - %d] -> %w", k, len(s.LangIndexs), off, total, err)
				}
			}

			s.Languages[k] = append(s.Languages[k], t)
			off += U32(t.Size())
		}
	}

	return nil
}

func (s *Ctx) Marshal(wt io.Writer) error {
	if err := s.Un1.Marshal(wt); err != nil {
		return fmt.Errorf("un1 -> %w", err)
	}

	if err := s.Un2.Marshal(wt); err != nil {
		return fmt.Errorf("un2 -> %w", err)
	}

	if len(s.LangIndexs) != len(s.Languages) {
		return fmt.Errorf("len(langindexes) != len(languages)")
	}

	length := U32(len(s.LangIndexs))
	if err := length.Marshal(wt); err != nil {
		return fmt.Errorf("len -> %w", err)
	}

	if length == 0 {
		return nil
	}

	for k := range s.LangIndexs {
		if err := s.LangIndexs[k].Marshal(wt); err != nil {
			return fmt.Errorf("[%d/%d] langindex -> %w", k, len(s.LangIndexs), err)
		}
	}

	off := U32(0)
	for k := range s.Languages {
		s.LangIndexs[k].Off = off
		for j := range s.Languages[k] {
			if err := s.Languages[k][j].Marshal(wt); err != nil {
				return fmt.Errorf("[%d/%d] single [%d - inf] -> %w", k, len(s.LangIndexs), off, err)
			}
			off += U32(s.Languages[k][j].Size())
		}
	}

	return nil
}

func parseCtx(rd io.Reader, wt io.Writer) error {
	var h Ctx

	if err := h.Unmarshal(rd); err != nil {
		return err
	}

	encoder := json.NewEncoder(wt)
	encoder.SetIndent("", "\t")
	return encoder.Encode(h)
}

func compileCtx(rd io.Reader, wt io.Writer) error {
	var h Ctx

	err := json.NewDecoder(rd).Decode(&h)
	if err != nil {
		return err
	}

	return h.Marshal(wt)
}
