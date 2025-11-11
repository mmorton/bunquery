package bunquery

import "database/sql"

type QueryOpts struct {
	Mods []QueryMod
}

type QueryOpt func(*QueryOpts)

func (opt QueryOpt) Apply(opts any) {
	switch opts := opts.(type) {
	case *QueryOpts:
		opt(opts)
	}
}

func WithMods(mods ...QueryMod) QueryOpt {
	return func(o *QueryOpts) {
		if o.Mods == nil {
			o.Mods = make([]QueryMod, 0, len(mods))
		}
		o.Mods = append(o.Mods, mods...)
	}
}

type MutationOpts struct {
	QueryOpts
	TxOptions *sql.TxOptions
}

type MutationOpt func(*MutationOpts)

func (opt MutationOpt) Apply(opts any) {
	switch opts := opts.(type) {
	case *MutationOpts:
		opt(opts)
	}
}

type AnyOpt interface {
	Apply(opts any)
}

func WithTxOptions(txOptions *sql.TxOptions) MutationOpt {
	return func(o *MutationOpts) {
		o.TxOptions = txOptions
	}
}

func NewQueryOpts(opts ...AnyOpt) *QueryOpts {
	queryOpts := &QueryOpts{}
	for _, opt := range opts {
		opt.Apply(queryOpts)
	}
	return queryOpts
}

func NewMutationOpts(opts ...AnyOpt) *MutationOpts {
	mutationOpts := &MutationOpts{}
	for _, opt := range opts {
		opt.Apply(mutationOpts)
	}
	return mutationOpts
}
