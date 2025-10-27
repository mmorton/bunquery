package bunquery

import (
	"errors"
	"fmt"
	"hash/crc32"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/uptrace/bun"
)

type Sortable interface {
	GetSortValues(rid uint32) []any
}

const (
	SortAscending  uint8 = 0x00
	SortDescending uint8 = 0x01
	SortUnknown    uint8 = 0xFF
)

type PagingRequest interface {
	GetContinue() string
	GetOrder() string
}

type Sort[M Sortable] struct {
	id    uint32
	model M
	cols  []string // registered columns
	dirs  []uint8  // registered directions
	vals  int      // num values required
	def   bool     // is this the default sort
}

func NewSort[M Sortable](model M) *Sort[M] {
	return &Sort[M]{model: model}
}

func (s *Sort[M]) Column(cols ...string) *Sort[M] {
	for _, col := range cols {
		s.cols = append(s.cols, strings.ToLower(strings.TrimSpace(col)))
	}
	return s
}

func (s *Sort[M]) Direction(dirs ...uint8) *Sort[M] {
	s.dirs = append(s.dirs, dirs...)
	return s
}

func (s *Sort[M]) Order(exprs ...string) *Sort[M] {
	for _, expr := range exprs {
		cols, dirs := parseOrder(expr)
		s.cols = append(s.cols, cols...)
		s.dirs = append(s.dirs, dirs...)
	}
	return s
}

func (s *Sort[M]) Default() *Sort[M] {
	s.def = true
	return s
}

var (
	regs  = map[uint32]any{}
	defs  = map[string]uint32{}
	regsM = sync.RWMutex{}
)

func mid(model any) string {
	return fmt.Sprintf("%T", getUnderlyingPointerType(model))
}

func sid(model any, cols []string) uint32 {
	m := getUnderlyingPointerType(model)
	t := fmt.Sprintf("%T:", m) + strings.Join(cols, ",")
	return crc32.ChecksumIEEE([]byte(t))
}

func (s *Sort[M]) checkAndRegister() (uint32, error) {
	regsM.Lock()
	defer regsM.Unlock()

	s.id = sid(s.model, s.cols)
	s.vals = len(s.cols)

	regs[s.id] = s

	if s.def {
		defs[mid(s.model)] = s.id
	} else {
		if _, ok := defs[mid(s.model)]; !ok {
			defs[mid(s.model)] = s.id
		}
	}

	return s.id, nil
}

func (s *Sort[M]) MustRegister() uint32 {
	if id, err := s.checkAndRegister(); err != nil {
		panic(err)
	} else {
		return id
	}
}

func (s *Sort[M]) Register() (uint32, error) {
	if id, err := s.checkAndRegister(); err != nil {
		return 0, err
	} else {
		return id, nil
	}
}

func (s *Sort[M]) getResolvedDirections(prev *Continuation) []uint8 {
	if prev == nil {
		return nil
	}

	res := make([]uint8, len(s.dirs))
	for i, dir := range s.dirs {
		if i < len(prev.Directions) {
			dir = prev.Directions[i]
		}
		if prev.Reverse {
			if dir == SortAscending {
				res = append(res, SortDescending)
			} else {
				res = append(res, SortAscending)
			}
		} else {
			res = append(res, dir)
		}
	}
	return res
}

func matchSID[M Sortable](rid uint32) (*Sort[M], error) {
	regsM.RLock()
	defer regsM.RUnlock()
	if s, ok := regs[rid]; !ok {
		return nil, errors.New("no sort registered")
	} else if t, ok := s.(*Sort[M]); !ok {
		return nil, errors.New("sort is not correct type")
	} else {
		return t, nil
	}
}

func matchDefaults[M Sortable](model M) (*Sort[M], error) {
	regsM.RLock()
	defer regsM.RUnlock()
	m := getUnderlyingPointerType(model)
	if sid, ok := defs[mid(m)]; !ok {
		return nil, errors.New("no default sort registered")
	} else {
		return matchSID[M](sid)
	}
}

func parseOrder(order string) ([]string, []uint8) {
	var cols []string
	var dirs []uint8
	for expr := range strings.SplitSeq(order, ",") {
		idx := strings.Index(expr, " ")
		if idx == -1 {
			cols = append(cols, strings.ToLower(strings.TrimSpace(expr)))
			dirs = append(dirs, SortAscending)
		} else {
			cols = append(cols, strings.TrimSpace(expr[:idx]))
			switch strings.ToUpper(strings.TrimSpace(expr[idx+1:])) {
			case "ASC":
				dirs = append(dirs, SortAscending)
			case "DESC":
				dirs = append(dirs, SortDescending)
			default:
				dirs = append(dirs, SortAscending)
			}
		}
	}
	return cols, dirs
}

type PagerOptions struct {
	ForwardOnly bool
	PageSize    int
}

type PagerOption = func(*PagerOptions)

func WithForwardOnly() PagerOption {
	return func(opts *PagerOptions) {
		opts.ForwardOnly = true
	}
}

func WithPageSize(size int) PagerOption {
	return func(opts *PagerOptions) {
		opts.PageSize = size
	}
}

type Pager[M Sortable] struct {
	from *Sort[M]
	cont *Continuation
	dirs []uint8
	rvrs bool
	opts *PagerOptions
}

func NewOptions(opts ...PagerOption) *PagerOptions {
	res := &PagerOptions{}
	for _, opt := range opts {
		opt(res)
	}
	return res
}

func NewPager[M Sortable](model M, opts ...PagerOption) (*Pager[M], error) {
	if sort, err := matchDefaults(model); err != nil {
		return nil, err
	} else {
		return &Pager[M]{from: sort, dirs: sort.dirs, rvrs: false, opts: NewOptions(opts...)}, nil
	}
}

func NewPagerFromSID[M Sortable](sid uint32, opts ...PagerOption) (*Pager[M], error) {
	if sort, err := matchSID[M](sid); err != nil {
		return nil, err
	} else {
		return &Pager[M]{from: sort, dirs: sort.dirs, rvrs: false, opts: NewOptions(opts...)}, nil
	}
}

func NewPagerFromOrder[M Sortable](model M, order string, opts ...PagerOption) (*Pager[M], error) {
	cols, dirs := parseOrder(order)
	sid := sid(model, cols)
	if sort, err := matchSID[M](sid); err != nil {
		return nil, err
	} else {
		return &Pager[M]{from: sort, dirs: dirs, rvrs: false, opts: NewOptions(opts...)}, nil
	}
}

func NewPagerFromContinuation[M Sortable](token string, opts ...PagerOption) (*Pager[M], error) {
	if cont, err := ParseContinuation(token); err != nil {
		return nil, err
	} else if sort, err := matchSID[M](cont.SID); err != nil {
		return nil, err
	} else {
		return &Pager[M]{from: sort, dirs: sort.getResolvedDirections(cont), rvrs: cont.Reverse, cont: cont, opts: NewOptions(opts...)}, nil
	}
}

func NewPagerFromRequest[M Sortable](model M, req PagingRequest, opts ...PagerOption) (*Pager[M], error) {
	if req.GetContinue() != "" {
		return NewPagerFromContinuation[M](req.GetContinue(), opts...)
	} else if req.GetOrder() != "" {
		return NewPagerFromOrder[M](model, req.GetOrder(), opts...)
	} else {
		return NewPager(model, opts...)
	}
}

func (p *Pager[M]) Compile() func(qry *bun.SelectQuery) *bun.SelectQuery {
	return func(qry *bun.SelectQuery) *bun.SelectQuery {
		// Only have to do this if we have a continuation
		if p.cont != nil {
			qry = qry.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
				for i := range p.from.cols {
					q = q.WhereGroup(" OR ", func(q *bun.SelectQuery) *bun.SelectQuery {
						for j := range i + 1 {
							q = q.Where(
								fmt.Sprintf("%s? %s ?",
									getSortTableAlias(p.from.cols[j]),
									getSortCompOp(p.dirs[j], i, j, len(p.from.cols), p.cont.Include),
								),
								bun.Ident(p.from.cols[j]),
								p.cont.Values[j],
							)
						}
						return q
					})
				}
				return q
			})
		}

		var order []string
		for i, col := range p.from.cols {
			dir := p.from.dirs[i]
			if i < len(p.dirs) {
				dir = p.dirs[i]
			}
			sort := "ASC"
			if dir == SortDescending {
				sort = "DESC"
			}
			order = append(order, fmt.Sprintf("%s %s", col, sort))
		}
		qry = qry.Order(order...)

		if p.opts.PageSize > 0 {
			qry = qry.Limit(p.opts.PageSize)
		}

		return qry
	}
}

func (p *Pager[M]) getFirstLastSortValues(results []M) ([]any, []any) {
	if len(results) == 0 {
		return []any{}, []any{}
	}
	f := results[0]
	l := results[len(results)-1]
	fv, lv := f.GetSortValues(p.from.id), l.GetSortValues(p.from.id)
	if p.rvrs {
		return lv, fv
	} else {
		return fv, lv
	}
}

func (p *Pager[M]) Map(results []M) ([]M, string, string, error) {
	if len(results) > 0 {
		fv, lv := p.getFirstLastSortValues(results)
		next, err := FormatContinuation(p.from.id, p.dirs, lv, false, false)
		if err != nil {
			return results, "", "", err
		}
		prev, err := FormatContinuation(p.from.id, p.dirs, fv, true, false)
		if err != nil {
			return results, "", "", err
		}

		if p.rvrs {
			// Do we really need to clone?
			temp := make([]M, len(results))
			copy(temp, results)
			slices.Reverse(temp)
			results = temp
		}

		// Zero out tokens that don't make sense.
		if p.rvrs {
			if p.opts.PageSize > 0 && len(results) < p.opts.PageSize {
				prev = ""
			}
		} else {
			if p.opts.PageSize > 0 && len(results) < p.opts.PageSize {
				next = ""
			}
		}

		return results, next, prev, nil
	} else if p.cont != nil {
		// This creates a reflection of the last page (which includes the last result value)
		if p.cont.Reverse {
			if nt, err := FormatContinuation(p.cont.SID, p.cont.Directions, p.cont.Values, false, true); err != nil {
				return results, "", "", err
			} else {
				return results, nt, "", nil
			}
		} else {
			if pt, err := FormatContinuation(p.cont.SID, p.cont.Directions, p.cont.Values, true, true); err != nil {
				return results, "", "", err
			} else {
				return results, "", pt, nil
			}
		}
	}

	return results, "", "", nil
}

// NOTE: all token returns are next, prev
/*
A, Z, D, 20
A, Z, E, 10
B, Y, D, 05
B, Y, D, 25
B, Y, E, 50 <<
---
B, Y, E, 55
---
B, Z, D, 15
B, Z, E, 00
C, X, E, 30

((f0 > v0))
((f0 > v0) OR (f0 = v0 AND f1 > v1))
((f0 > v0) OR (f0 = v0 AND f1 > v1) OR (f0 = v0 AND f1 = v1 AND f2 > v2))
((f0 > v0) OR (f0 = v0 AND f1 > v1) OR (f0 = v0 AND f1 = v1 AND f2 > v2) OR (f0 = v0 AND f1 = v1 AND f2 = v2 AND f3 > v3))

standard forward condition for f0 asc, f1 asc, f2 asc (last row of previous page was v0, v1, v2):
(
	   (f0 > v0)
	OR (f0 = v0 AND f1 > v1)
	OR (f0 = v0 AND f1 = v1 AND f2 > v2)
)

if above query returns no results, the "way back" (reverse direction) token is rooted at v0, v1, v2 and must include this value:
(
	   (f0 < v0)
	OR (f0 = v0 AND f1 < v1)
	OR (f0 = v0 AND f1 = v1 AND f2 <= v2)
)
*/

func getExcIncOp(excOp, incOp string, inc bool) string {
	if inc {
		return incOp
	} else {
		return excOp
	}
}

func getSortCompOp(d uint8, i, j, num int, inc bool) string {
	if i == j {
		if d == SortAscending {
			return getExcIncOp(">", ">=", i == j && i == num-1 && inc) // ">"
		} else {
			return getExcIncOp("<", ">=", i == j && i == num-1 && inc) //"<"
		}
	} else {
		return "="
	}
}

func getSortTableAlias(fld string) string {
	if strings.Contains(fld, ".") {
		return ""
	} else {
		return "?TableAlias."
	}
}

func getUnderlyingPointerType(model any) any {
	t := reflect.TypeOf(model)
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	return reflect.New(t).Interface()
}
