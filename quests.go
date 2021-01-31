package main

import (
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf16"

	"github.com/xhebox/bstruct"
)

type String struct {
	L uint32
	R []uint16 `length:"R_length"`
}

func (s *String) Unmarshal() string {
	return string(utf16.Decode(s.R))
}

func (s *String) Marshal(t string) {
	s.R = utf16.Encode([]rune(t))
	s.L = uint32(len(s.R))
}

type QuestSingle struct {
	ID  uint64
	Str [2]String `json:"-"`
	S1  string    `skip:"rw"`
	S2  string    `skip:"rw"`
	Un1 uint32
}

func (s *QuestSingle) Unmarshal() {
	s.S1 = s.Str[0].Unmarshal()
	s.S2 = s.Str[1].Unmarshal()
}

func (s *QuestSingle) Marshal() {
	s.Str[0].Marshal(s.S1)
	s.Str[1].Marshal(s.S2)
}

type QuestItem struct {
	Main QuestSingle
	Len  uint16        `json:"-"`
	Sub  []QuestSingle `length:"Item_length"`
}

func (item *QuestItem) Unmarshal() {
	item.Main.Unmarshal()
	for i := range item.Sub {
		item.Sub[i].Unmarshal()
	}
}

func (item *QuestItem) Marshal() {
	item.Main.Marshal()
	item.Len = uint16(len(item.Sub))
	for i := range item.Sub {
		item.Sub[i].Marshal()
	}
}

type Quests struct {
	Len   uint32      `json:"-"`
	Items []QuestItem `length:"Items_length"`
}

func (q *Quests) Unmarshal() {
	for i := range q.Items {
		q.Items[i].Unmarshal()
	}
}

func (q *Quests) Marshal() {
	q.Len = uint32(len(q.Items))
	for i := range q.Items {
		q.Items[i].Marshal()
	}
}

func parseQuests(rd io.Reader, wt io.Writer) error {
	var h Quests

	ctxType := bstruct.MustNew(h)

	dec := bstruct.NewDecoder()
	dec.Rd = rd

	dec.Runner.Register("R_length", func(s ...interface{}) interface{} {
		return int(s[1].(String).L)
	})

	dec.Runner.Register("Item_length", func(s ...interface{}) interface{} {
		return int(s[1].(QuestItem).Len)
	})

	dec.Runner.Register("Items_length", func(s ...interface{}) interface{} {
		return int(s[1].(Quests).Len)
	})

	if e := dec.Decode(ctxType, &h); e != nil {
		return e
	}

	h.Unmarshal()
	encoder := json.NewEncoder(wt)
	encoder.SetIndent("", "\t")
	return encoder.Encode(&h)
}

func compileQuests(rd io.Reader, wt io.Writer) error {
	var h Quests

	ctxType := bstruct.MustNew(h)

	e := json.NewDecoder(rd).Decode(&h)
	if e != nil {
		return e
	}

	h.Marshal()
	enc := bstruct.NewEncoder()
	enc.Wt = wt
	return enc.Encode(ctxType, &h)
}

func handleQuests(mode string, rd io.Reader, wt io.Writer) error {
	switch mode {
	case "parse":
		e := parseQuests(rd, wt)
		if e != nil {
			return fmt.Errorf("failed to convert ctx to json: %w", e)
		}
	case "compile":
		e := compileQuests(rd, wt)
		if e != nil {
			return fmt.Errorf("failed to convert json to ctx: %w", e)
		}
	}
	return nil
}
