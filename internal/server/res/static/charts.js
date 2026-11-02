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

// Draws the Activity page's charts as SVG bar charts (div[data-chart], data
// from the API), on load and after each page swap. One group of bars per
// label, one bar per period being compared.

"use strict";

(function() {
    const SVG = "http://www.w3.org/2000/svg";
    const W = 800, H = 240;          // drawing size, scaled to fit
    const L = 36, B = 22, T = 8;     // left, bottom and top margins
    const MaxLabels = 16;

    const el = function(name, attrs, parent) {
	const e = document.createElementNS(SVG, name);
	for (const [k, v] of Object.entries(attrs)) {
	    e.setAttribute(k, v);
	}
	if (parent) {
	    parent.appendChild(e);
	}
	return e;
    };

    const text = function(parent, x, y, s, cls) {
	const t = el("text", {x: x, y: y, class: cls}, parent);
	t.textContent = s;
	return t;
    };

    // a round axis maximum: 1, 2, 5, 10, 20, 50, ...
    const niceMax = function(n) {
	let step = 1;
	while (true) {
	    for (const m of [1, 2, 5]) {
		if (m * step >= n) {
		    return m * step;
		}
	    }
	    step *= 10;
	}
    };

    const draw = function(box, data) {
	const labels = data.Labels || [];
	const charts = data.Charts || [];
	const max = niceMax(Math.max(1, ...charts.flatMap(c => c.Counts)));
	const plotW = W - L, plotH = H - B - T;
	const svg = el("svg", {viewBox: `0 0 ${W} ${H}`, class: "bars", role: "img",
			       "aria-label": "Listens per period"});

	// whole-number ticks: max is 1, 2, 5 or a multiple of 5
	const ticks = max <= 5 ? max : 5;
	for (let i = 0; i <= ticks; i++) {
	    const y = T + plotH - plotH * i / ticks;
	    el("line", {x1: L, x2: W, y1: y, y2: y, class: "grid"}, svg);
	    text(svg, L - 6, y + 4, String(max * i / ticks), "axis y");
	}

	const group = plotW / Math.max(1, labels.length);
	const bar = group * 0.8 / Math.max(1, charts.length);
	const every = Math.ceil(labels.length / MaxLabels);
	labels.forEach((label, li) => {
	    const x0 = L + li * group + group * 0.1;
	    charts.forEach((c, ci) => {
		const n = c.Counts[li] || 0;
		const h = plotH * n / max;
		const r = el("rect", {
		    x: x0 + ci * bar, y: T + plotH - h, width: Math.max(1, bar - 1), height: h,
		    class: "bar bar-" + ((charts.length - 1 - ci) % 3)
		}, svg);
		el("title", {}, r).textContent = `${label} ${c.Label}: ${n}`;
	    });
	    if (li % every == 0) {
		text(svg, L + li * group + group / 2, H - 6, label, "axis x");
	    }
	});

	const legend = document.createElement("div");
	legend.className = "legend";
	charts.forEach((c, ci) => {
	    const item = document.createElement("span");
	    item.className = "legend-item bar-" + ((charts.length - 1 - ci) % 3);
	    item.textContent = c.Label;
	    legend.appendChild(item);
	});
	box.replaceChildren(legend, svg);
    };

    const drawAll = function() {
	document.querySelectorAll("[data-chart]").forEach(box => {
	    fetch(box.getAttribute("data-chart")).
		then(resp => resp.json()).
		then(data => draw(box, data)).
		catch(err => console.log("chart: " + err));
	});
    };

    document.addEventListener("DOMContentLoaded", drawAll);
    document.addEventListener("takeout:page", drawAll);
})();
