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

// The sign-in and Link a device pages: messages from the server's redirect
// (?failed, ?linked), a code from the URL (?code=, as a QR code would give),
// and on Link, no username and password when this browser is signed in.

"use strict";

(function() {
    const params = new URLSearchParams(location.search);

    document.querySelectorAll("[data-show-if]").forEach(e => {
	e.hidden = !params.has(e.getAttribute("data-show-if"));
    });

    // codes are upper case; phones like to type lower
    const code = document.getElementById("code");
    if (code) {
	const value = (params.get("code") || "").toUpperCase().replace(/[^1-9A-Z]/g, "");
	if (value) {
	    code.value = value.slice(0, 6);
	}
	code.addEventListener("input", () => {
	    const at = code.selectionStart;
	    code.value = code.value.toUpperCase();
	    code.setSelectionRange(at, at);
	});
    }

    // signed in already: the server links with this browser's session
    const credentials = document.querySelector("[data-credentials]");
    if (credentials) {
	fetch("/api/index", {redirect: "manual"}).then(resp => {
	    if (resp.ok) {
		credentials.hidden = true;
		credentials.querySelectorAll("input").forEach(i => { i.required = false; });
		document.querySelectorAll("[data-signed-in]").forEach(e => { e.hidden = false; });
	    }
	}).catch(() => {});
    }
})();
