// Copyright 2023 defsub
//
// This file is part of TakeoutFM.
//
// TakeoutFM is free software: you can redistribute it and/or modify it under the
// terms of the GNU Affero General Public License as published by the Free
// Software Foundation, either version 3 of the License, or (at your option)
// any later version.
//
// TakeoutFM is distributed in the hope that it will be useful, but WITHOUT ANY
// WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
// FOR A PARTICULAR PURPOSE.  See the GNU Affero General Public License for
// more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with TakeoutFM.  If not, see <https://www.gnu.org/licenses/>.

package activity

import (
	"fmt"

	"takeoutfm.dev/takeout/lib/date"
	"takeoutfm.dev/takeout/view"
)

func (a *Activity) TrackDayCounts(ctx Context, d date.DateRange) *view.TrackCounts {
	view := &view.TrackCounts{}
	view.Counts = a.TrackCountsByDay(ctx, d.Start, d.End)
	return view
}

func (a *Activity) TrackMonthCounts(ctx Context, d date.DateRange) *view.TrackCounts {
	view := &view.TrackCounts{}
	view.Counts = a.TrackCountsByMonth(ctx, d.Start, d.End)
	return view
}

func (a *Activity) BuildChart(ctx Context, d date.DateRange) *view.TrackCharts {
	charts := &view.TrackCharts{}

	if d.IsYear() {
		prev := d.PreviousYear()
		charts.Labels = []string{
			"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
		}
		charts.AddCounts(
			fmt.Sprintf("%d", prev.Start.Year()),
			a.TrackMonthCounts(ctx, prev))
		charts.AddCounts(
			fmt.Sprintf("%d", d.Start.Year()),
			a.TrackMonthCounts(ctx, d))
	} else if d.IsDay() {
		prev := d.PreviousDay()
		charts.Labels = []string{"Listens"}
		charts.AddCounts(
			prev.Start.Weekday().String(),
			a.TrackDayCounts(ctx, prev))
		charts.AddCounts(
			d.Start.Weekday().String(),
			a.TrackDayCounts(ctx, d))
	} else if d.IsWeek() {
		prev := d.PreviousWeek()
		charts.Labels = []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
		charts.AddCounts(
			fmt.Sprintf("%s %d - %s %d",
				prev.Start.Month().String()[:3], prev.Start.Day(),
				prev.End.Month().String()[:3], prev.End.Day()),
			a.TrackDayCounts(ctx, prev))
		charts.AddCounts(
			fmt.Sprintf("%s %d - %s %d",
				d.Start.Month().String()[:3], d.Start.Day(),
				d.End.Month().String()[:3], d.End.Day()),
			a.TrackDayCounts(ctx, d))
	} else if d.IsMonth() {
		prev := d.PreviousMonth()
		labels := make([]string, 31)
		for i := range 31 {
			labels[i] = fmt.Sprintf("%02d", i+1)
		}
		charts.Labels = labels
		charts.AddCounts(
			fmt.Sprintf("%s %d", prev.Start.Month().String()[:3], prev.Start.Year()),
			a.TrackDayCounts(ctx, prev))
		charts.AddCounts(
			fmt.Sprintf("%s %d", d.Start.Month().String()[:3], d.Start.Year()),
			a.TrackDayCounts(ctx, d))
	} else {
		days := d.DayCount()
		labels := make([]string, days)
		for i := range days {
			d := d.Start.AddDate(0, 0, i)
			labels[i] = fmt.Sprintf("%s %d", d.Month().String()[:3], d.Day())
		}
		charts.Labels = labels
		charts.AddCounts("Listens", a.TrackDayCounts(ctx, d))
	}

	return charts
}
