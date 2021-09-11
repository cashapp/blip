package size

import (
	"fmt"
	"strings"

	"github.com/square/blip/collect"
)

func DataSizeQuery(set map[string]string, def collect.Help) (string, error) {
	cols := ""
	groupBy := ""
	if val := set[OPT_TOTAL]; val == "only" {
		cols = "\"\" AS db"
	} else {
		cols = "table_schema AS db"
		groupBy = " GROUP BY 1"
	}
	cols += ", SUM(data_length+index_length) AS bytes"

	like := false
	if val := set[OPT_LIKE]; val == "yes" {
		like = true
	}

	where := ""
	if include := set[OPT_INCLUDE]; include != "" {
		o := collect.ObjectList(include, "'")
		if like {
			for i := range o {
				o[i] = "table_schema LIKE " + o[i]
			}
			where = strings.Join(o, " OR ")
		} else {
			where = fmt.Sprintf("table_schema IN (%s)", strings.Join(o, ","))
		}
	} else {
		exclude := set[OPT_EXCLUDE]
		if exclude == "" {
			exclude = def.Options[OPT_EXCLUDE].Default
		}
		o := collect.ObjectList(exclude, "'")
		if like {
			for i := range o {
				o[i] = "table_schema NOT LIKE " + o[i]
			}
			where = strings.Join(o, " AND ")
		} else {
			where = fmt.Sprintf("table_schema NOT IN (%s)", strings.Join(o, ","))
		}
	}

	return "SELECT " + cols + " FROM information_schema.tables WHERE " + where + groupBy, nil
}
