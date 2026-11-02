// Copyright 2026 defsub
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

// Draws the Activity page's charts (canvas[data-chart], data from the API)
// with Chart.js, on load and after each page swap.

"use strict";

(function() {
    const barColors = ["#3DDC84", "#f7f6e4", "#9BD0F5"];

    const draw = function() {
	document.querySelectorAll("canvas[data-chart]").forEach(e => {
	    fetch(e.getAttribute("data-chart")).
		then(resp => resp.json()).
		then(data => {
		    const datasets = data.Charts.map(chart => ({label: chart.Label, data: chart.Counts}));
		    datasets.forEach((d, i) => {
			d.backgroundColor = barColors[(datasets.length - 1 - i) % barColors.length];
		    });
		    new Chart(e, {type: "bar", data: {labels: data.Labels, datasets: datasets}});
		}).
		catch(err => console.log("chart: " + err));
	});
    };

    document.addEventListener("DOMContentLoaded", draw);
    document.addEventListener("takeout:page", draw);
})();
