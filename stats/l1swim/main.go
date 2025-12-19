package main

import (
	"cmp"
	"context"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pgaskin/ottrec-website/pkg/ottrecdl"
	"github.com/pgaskin/ottrec-website/pkg/ottrecidx"
	"github.com/pgaskin/ottrec/schema"
)

var (
	All = flag.Bool("all", false, "not just line 1")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	cl := &ottrecdl.Client{
		Base: "https://data.ottrec.ca/",
	}

	pb, err := cl.Get(ctx, cmp.Or(flag.Arg(0), "latest"), "pb")
	if err != nil {
		panic(err)
	}

	idx, err := new(ottrecidx.Indexer).Load(pb)
	if err != nil {
		panic(err)
	}

	// aquatic facilities within a short (~15 minute fast walk, or ~10 minute
	// bus at least every 20 min) distance of a Line 1 station.
	line1PoolName := []string{
		"plant",
		"jack purcell",
		"champagne",
		"st. laurent",
		"splash",
		"bob macquarrie",
	}
	line1Pool := make([]ottrecidx.FacilityRef, len(line1PoolName))
	for i, name := range line1PoolName {
		for fac := range idx.Data().Facilities() {
			if strings.Contains(strings.ToLower(fac.GetName()), name) {
				line1Pool[i] = fac
				break
			}
		}
	}
	for i, fac := range line1Pool {
		if !fac.Valid() {
			panic(fmt.Errorf("no match for facility %q", line1PoolName[i]))
		}
	}

	var grps []ottrecidx.ScheduleGroupRef
	if *All {
		for grp := range idx.Data().ScheduleGroups() {
			if strings.Contains(strings.ToLower(grp.GetLabel()), "swim") {
				grps = append(grps, grp)
			}
		}
	} else {
		grps = make([]ottrecidx.ScheduleGroupRef, len(line1Pool))
		for i, fac := range line1Pool {
			for grp := range fac.ScheduleGroups() {
				if strings.Contains(strings.ToLower(grp.GetLabel()), "swim") {
					if grps[i].Valid() {
						panic(fmt.Errorf("multiple swim schedule groups for facility %q (%q, %q)", fac.GetName(), grps[i].GetLabel(), grp.GetLabel()))
					}
					grps[i] = grp
				}
			}
		}
		for i, grp := range grps {
			if !grp.Valid() {
				panic(fmt.Errorf("no match for facility %q schedule group %q", line1PoolName[i], "swim"))
			}
		}
	}

	for i, grp := range grps {
		if i != 0 {
			fmt.Println()
		}
		if grp.Schedules().Empty() {
			fmt.Printf("%q :: %q :: %s\n", grp.Facility().GetName(), grp.GetLabel(), "no schedules in group, skipping")
			continue
		}
		for sch := range grp.Schedules() {
			fmt.Println()

			dr, _ := sch.GetDateRange()
			fmt.Printf("%q :: %q :: %s\n", grp.Facility().GetName(), grp.GetLabel(), dr)

			var morn, after, eve []ottrecidx.TimeRef
			for tm := range sch.Times() {
				if strings.Contains(tm.Activity().GetName(), "lane swim") {
					cr, _ := tm.GetRange()
					if cr.Overlaps(schema.MakeClockRange(18, 00, 23, 59)) {
						eve = append(eve, tm)
					} else if cr.Overlaps(schema.MakeClockRange(11, 30, 18, 00)) {
						after = append(after, tm)
					} else {
						morn = append(morn, tm)
					}
				}
			}
			format := func(r []ottrecidx.TimeRef) string {
				var b strings.Builder
				var t time.Duration
				w := make(map[time.Weekday]struct{}, 7)
				for i, x := range r {
					if i != 0 {
						b.WriteString(", ")
					}
					cr, _ := x.GetRange()
					wd, _ := x.GetWeekday()
					w[wd] = struct{}{}
					b.WriteString(wd.String()[:2])
					b.WriteString(" ")
					b.WriteString(cr.Start.Format(false))
					t += (time.Duration(cr.End) - time.Duration(cr.Start)) * time.Minute
				}
				b.WriteString(" (")
				b.WriteString(t.String())
				b.WriteString(" total, ")
				b.WriteString(strconv.Itoa(len(w)))
				b.WriteString(" weekdays)")
				return b.String()
			}

			fmt.Printf("  morning:   %s\n", format(morn))
			fmt.Printf("  afternoon: %s\n", format(after))
			fmt.Printf("  evening:   %s\n", format(eve))
		}
	}
}
