// Copyright 2024 Block, Inc.

package mock

type Tr struct {
	TranslateFunc func(domain, metric string) string
}

func (tr *Tr) Translate(domain, metric string) string {
	if tr.TranslateFunc != nil {
		return tr.TranslateFunc(domain, metric)
	}

	return metric
}
