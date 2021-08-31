package heartbeat

import (
	"database/sql"

	"github.com/square/blip"
)

type SourceFinder interface {
	Find(blip.ConfigMonitor) (string, error)
}

// --------------------------------------------------------------------------

type StaticSourceList struct {
	sources []string
	db      *sql.DB
}

var _ SourceFinder = &StaticSourceList{}

func NewStaticSourceList(sources []string, db *sql.DB) *StaticSourceList {
	return &StaticSourceList{
		sources: sources,
		db:      db,
	}
}

func (w *StaticSourceList) Find(mon blip.ConfigMonitor) (string, error) {
	return w.sources[0], nil // @todo

}

// --------------------------------------------------------------------------

type AutoSourceFinder struct {
}

var _ SourceFinder = &AutoSourceFinder{}

func NewAutoSourceFinder() *AutoSourceFinder {
	return &AutoSourceFinder{}
}

func (w *AutoSourceFinder) Find(mon blip.ConfigMonitor) (string, error) {
	blip.Debug("auto-finding repl source")
	return "localhost", nil // @todo
}
