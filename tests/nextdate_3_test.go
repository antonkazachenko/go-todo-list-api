package tests

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type nextDate struct {
	date   string
	repeat string
	want   string
}

func TestNextDate(t *testing.T) {
	tbl := []nextDate{
		{"20240126", "", ""},
		{"20240126", "k 34", ""},
		{"20240126", "ooops", ""},
		{"15000156", "y", ""},
		{"ooops", "y", ""},
		{"16890220", "y", `20240220`},
		{"20250701", "y", `20260701`},
		{"20240101", "y", `20250101`},
		{"20231231", "y", `20241231`},
		{"20240229", "y", `20250301`},
		{"20240301", "y", `20250301`},
		{"20240113", "d", ""},
		{"20240113", "d 7", `20240127`},
		{"20240120", "d 20", `20240209`},
		{"20240202", "d 30", `20240303`},
		{"20240320", "d 401", ""},
		{"20231225", "d 12", `20240130`},
		{"20240228", "d 1", "20240229"},
	}
	check := func() {
		for _, v := range tbl {
			urlPath := fmt.Sprintf("api/nextdate?now=20240126&date=%s&repeat=%s",
				url.QueryEscape(v.date), url.QueryEscape(v.repeat))
			get, err := getBody(urlPath)
			assert.NoError(t, err)
			next := strings.TrimSpace(string(get))
			_, err = time.Parse("20060102", next)
			if err != nil && len(v.want) == 0 {
				continue
			}
			assert.Equal(t, v.want, next, `{%q, %q, %q}`,
				v.date, v.repeat, v.want)
		}
	}
	check()
	if !FullNextDate {
		return
	}
	tbl = []nextDate{
		{"20231106", "m 13", "20240213"},
		{"20240120", "m 40,11,19", ""},
		{"20240116", "m 16,5", "20240205"},
		{"20240126", "m 25,26,7", "20240207"},
		{"20240409", "m 31", "20240531"},
		{"20240329", "m 10,17 12,8,1", "20240810"},
		{"20230311", "m 07,19 05,6", "20240507"},
		{"20230311", "m 1 1,2", "20240201"},
		{"20240127", "m -1", "20240131"},
		{"20240222", "m -2", "20240228"},
		{"20240222", "m -2,-3", ""},
		{"20240326", "m -1,-2", "20240330"},
		{"20240201", "m -1,18", "20240218"},
		{"20240125", "w 1,2,3", "20240129"},
		{"20240126", "w 7", "20240128"},
		{"20230126", "w 4,5", "20240201"},
		{"20230226", "w 8,4,5", ""},
	}
	check()
}
