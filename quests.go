package main

import (
	"encoding/json"
	"io"

	"github.com/xhebox/bstruct"
)

type QuestSingle struct {
	Target StringWithID
	RStr   String `json:"-"`
	Str    string `skip:"rw"`
	Un1    uint32
}

func (s *QuestSingle) Unmarshal() {
	s.Target.Unmarshal()
	s.Str = s.RStr.Unmarshal()
}

func (s *QuestSingle) Marshal() {
	s.Target.Marshal()
	s.RStr.Marshal(s.Str)
}

type QuestSet struct {
	Default    QuestSingle
	Len        uint16        `json:"-"`
	QuestItems []QuestSingle `length:"Items_length"`
}

func (item *QuestSet) Unmarshal() {
	item.Default.Unmarshal()
	for i := range item.QuestItems {
		item.QuestItems[i].Unmarshal()
	}
}

func (item *QuestSet) Marshal() {
	item.Default.Marshal()
	item.Len = uint16(len(item.QuestItems))
	for i := range item.QuestItems {
		item.QuestItems[i].Marshal()
	}
}

type Quests struct {
	Len    uint32     `json:"-"`
	Quests []QuestSet `length:"Sets_length"`
}

func (q *Quests) Unmarshal() {
	for i := range q.Quests {
		q.Quests[i].Unmarshal()
	}
}

func (q *Quests) Marshal() {
	q.Len = uint32(len(q.Quests))
	for i := range q.Quests {
		q.Quests[i].Marshal()
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

	dec.Runner.Register("Items_length", func(s ...interface{}) interface{} {
		return int(s[1].(QuestSet).Len)
	})

	dec.Runner.Register("Sets_length", func(s ...interface{}) interface{} {
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
