// Copyright 2022 Block, Inc.

package mock

type Exporter struct {
	ScrapeFunc func() (string, error)
}

func (e Exporter) Scrape() (string, error) {
	if e.ScrapeFunc != nil {
		return e.ScrapeFunc()
	}
	return "", nil
}
