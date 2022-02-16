// Copyright 2022 Block, Inc.

package sizebinlog

import (
	"context"
	"database/sql"
	"fmt"

	myerr "github.com/go-mysql/errors"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/sqlutil"
)

const (
	DOMAIN = "size.binlog"

	OPT_NO_BINLOGS = "no-binlogs"
	OPT_NO_ACCESS  = "no-access"
)

// Binlog collects metrics for the size.binlog domain. The source is SHOW BINARY LOGS.
type Binlog struct {
	db *sql.DB
	// --
	cols3             bool
	noBinlogs         string
	noBinlogsReported bool
	noAccess          string
	noAccessReported  bool
}

var _ blip.Collector = &Binlog{}

func NewBinlog(db *sql.DB) *Binlog {
	return &Binlog{
		db: db,
		// --
		noBinlogs: "drop", // default value
		noAccess:  "drop", // default value
	}
}

func (c *Binlog) Domain() string {
	return DOMAIN
}

func (c *Binlog) Help() blip.CollectorHelp {
	return blip.CollectorHelp{
		Domain:      DOMAIN,
		Description: "Total size of all binary logs in bytes",
		Options: map[string]blip.CollectorHelpOption{
			OPT_NO_BINLOGS: {
				Name:    OPT_NO_BINLOGS,
				Desc:    "How to handle MySQL error 1381: binary logging not enabled",
				Default: "drop",
				Values: map[string]string{
					"zero":  "Ignore error, report 0 bytes",
					"drop":  "Report error once, don't report metric",
					"error": "Report error every time, don't report metric",
				},
			},
			OPT_NO_ACCESS: {
				Name:    OPT_NO_ACCESS,
				Desc:    "How to handle MySQL error 1227: cccess denied on 'SHOW BINARY LOGS'",
				Default: "drop",
				Values: map[string]string{
					"zero":  "Ignore error, report 0 bytes",
					"drop":  "Report error once, don't report metric",
					"error": "Report error every time, don't report metric",
				},
			},
		},
		Metrics: []blip.CollectorMetric{
			{
				Name: "bytes",
				Type: blip.GAUGE,
				Desc: "Total size of all binary logs in bytes",
			},
		},
	}
}

func (c *Binlog) Prepare(ctx context.Context, plan blip.Plan) (func(), error) {
LEVEL:
	for _, level := range plan.Levels {
		dom, ok := level.Collect[DOMAIN]
		if !ok {
			continue LEVEL // not collected at this level
		}

		// As of MySQL 8.0.14, SHOW BINARY LOGS has 3 cols instead of 2
		if ok, _ := sqlutil.MySQLVersionGTE("8.0.14", c.db, ctx); ok {
			c.cols3 = true
		}

		if val, ok := dom.Options[OPT_NO_BINLOGS]; ok {
			c.noBinlogs = val
		}
		if val, ok := dom.Options[OPT_NO_ACCESS]; ok {
			c.noAccess = val
		}

		// Only need to prepare once because nothing changes: it's all just
		// SHOW BINARY LOGS
		break LEVEL
	}
	return nil, nil
}

func (c *Binlog) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	// Total binlog size might be left a zero on error if option no-binlogs|no-access=zero
	var total float64

	rows, err := c.db.QueryContext(ctx, "SHOW BINARY LOGS")
	if err != nil {
		switch myerr.MySQLErrorCode(err) {
		case 0: // not a MySQL error
			return nil, err
		case 1381: // binary logging not enabled
			switch c.noBinlogs {
			case "zero":
				// continue, report zero value
			case "drop":
				if !c.noBinlogsReported {
					c.noBinlogsReported = true
					return nil, fmt.Errorf("%s (Blip will retry but not report error again)", err)
				}
				return nil, nil
			case "error":
				return nil, err
			}
		case 1227: // acccess denied
			switch c.noAccess {
			case "zero":
				// continue, report zero value
			case "drop":
				if !c.noAccessReported {
					c.noAccessReported = true
					return nil, fmt.Errorf("%s (Blip will retry but not report error again)", err)
				}
				return nil, nil
			case "error":
				return nil, err
			}
		default: // not a MySQL error
			return nil, err
		}
	} else {
		// Access rows only if err==nil, else rows is nil and this code will panic
		defer rows.Close()

		var (
			name string
			val  string
			enc  string
			ok   bool
			n    float64
		)
		for rows.Next() {
			if c.cols3 {
				err = rows.Scan(&name, &val, &enc) // 8.0.14+
			} else {
				err = rows.Scan(&name, &val)
			}
			if err != nil {
				return nil, err
			}
			n, ok = sqlutil.Float64(val)
			if !ok {
				continue
			}
			total += n
		}
	}

	metrics := []blip.MetricValue{{
		Name:  "bytes",
		Value: total,
		Type:  blip.GAUGE,
	}}

	return metrics, nil
}
