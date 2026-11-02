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

// Page navigation without reloading, so the player keeps playing. Links and
// the search form fetch only the new page's body (X-Takeout-Fragment) and
// swap it into #main; the address bar and history follow. Anything else
// (other sites, /api, /static, a new tab) is left to the browser.

"use strict";

(function() {
    // paths that aren't web UI pages
    const notPages = /^\/(api|static|img|d|login|link)(\/|$|\.)/;

    let controller = null;

    const isPage = function(url) {
	return url.origin == location.origin && !notPages.test(url.pathname);
    };

    const show = function(html) {
	const doc = new DOMParser().parseFromString(html, "text/html");
	const page = doc.querySelector("[data-title]");
	if (page == null) {
	    return false;
	}
	document.title = page.getAttribute("data-title");
	document.getElementById("main").replaceChildren(page);
	document.dispatchEvent(new CustomEvent("takeout:page"));
	return true;
    };

    // load url into #main; push adds a history entry, scroll is where to go
    const load = async function(url, push, scroll = 0) {
	if (controller != null) {
	    controller.abort();
	}
	controller = new AbortController();
	let resp;
	try {
	    resp = await fetch(url, {
		headers: {"X-Takeout-Fragment": "1"},
		signal: controller.signal
	    });
	} catch (err) {
	    if (err.name != "AbortError") {
		location.assign(url);
	    }
	    return;
	}
	const target = new URL(resp.url);
	if (!resp.ok || !isPage(target)) {
	    // signed out (redirected to the login page), not found, etc.
	    location.assign(resp.ok ? resp.url : url);
	    return;
	}
	const html = await resp.text();
	if (push) {
	    // remember where this page was scrolled to before leaving it
	    history.replaceState({scroll: window.scrollY}, "");
	    history.pushState({scroll: 0}, "", url);
	}
	if (!show(html)) {
	    location.assign(url);
	    return;
	}
	window.scrollTo(0, scroll);
    };

    const onClick = function(event) {
	if (event.defaultPrevented || event.button != 0 ||
	    event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
	    return;
	}
	const a = event.target.closest("a[href]");
	if (a == null || a.target || a.hasAttribute("download") ||
	    a.closest("[data-action], [data-playlist], [data-play]")) {
	    return;
	}
	const url = new URL(a.href);
	if (!isPage(url)) {
	    return;
	}
	event.preventDefault();
	if (url.href != location.href) {
	    load(url.href, true);
	}
    };

    const onSubmit = function(event) {
	const form = event.target;
	if (form.method != "get") {
	    return;
	}
	const url = new URL(form.action);
	if (!isPage(url)) {
	    return;
	}
	event.preventDefault();
	url.search = new URLSearchParams(new FormData(form)).toString();
	load(url.href, true);
    };

    const onPop = function(event) {
	const scroll = (event.state && event.state.scroll) || 0;
	load(location.href, false, scroll);
    };

    history.scrollRestoration = "manual";
    history.replaceState({scroll: 0}, "");
    document.addEventListener("click", onClick);
    document.addEventListener("submit", onSubmit);
    window.addEventListener("popstate", onPop);
})();
