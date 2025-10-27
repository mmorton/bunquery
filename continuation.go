package bunquery

import (
	"encoding/base64"
	"fmt"

	"github.com/vmihailenco/msgpack/v5"
)

const (
	flagReverse uint8 = 1 << iota
	flagInclude
)

type rawContinuation struct {
	K uint32
	D uint8
	F uint8
	V []any
}

func decodeRawContinuation(token string) (*rawContinuation, error) {
	b, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}

	c := &rawContinuation{}
	if err := msgpack.Unmarshal(b, c); err != nil {
		return nil, err
	}

	return c, nil
}

func (raw *rawContinuation) Continuation() *Continuation {
	var dirs []uint8
	for i := range 8 {
		dirs = append(dirs, (raw.D>>i)&0x01)
	}
	return &Continuation{
		SID:        raw.K,
		Directions: dirs,
		Values:     raw.V,
		Reverse:    raw.F&flagReverse != 0,
		Include:    raw.F&flagInclude != 0,
	}
}

func (raw *rawContinuation) String() (string, error) {
	if b, err := msgpack.Marshal(raw); err != nil {
		return "", err
	} else {
		return base64.StdEncoding.EncodeToString(b), nil
	}
}

type Continuation struct {
	SID        uint32
	Directions []uint8
	Values     []any
	Reverse    bool
	Include    bool
}

func NewContinuation(sid uint32, directions []uint8, values []any, reverse, include bool) *Continuation {
	return &Continuation{
		SID:        sid,
		Directions: directions,
		Values:     values,
		Reverse:    reverse,
		Include:    include,
	}
}

func (c *Continuation) raw() *rawContinuation {
	d := uint8(0)
	for i, dir := range c.Directions {
		d |= (dir & 0x1) << i
	}
	p := &rawContinuation{
		K: c.SID,
		D: d,
		V: c.Values,
	}
	if c.Reverse {
		p.F |= flagReverse
	}
	if c.Include {
		p.F |= flagInclude
	}
	return p
}

func (c *Continuation) String() (string, error) {
	return c.raw().String()
}

func FormatContinuation(sid uint32, directions []uint8, values []any, reverse, include bool) (string, error) {
	res := NewContinuation(sid, directions, values, reverse, include)
	return res.String()
}

func ParseContinuation(token string) (*Continuation, error) {
	if c, err := decodeRawContinuation(token); err != nil {
		return nil, fmt.Errorf("failed to parse continue token: %w", err)
	} else {
		return c.Continuation(), nil
	}
}
