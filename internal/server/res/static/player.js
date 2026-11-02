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

// The audio player. The queue is the server's playlist (/api/playlist), as in
// the other TakeoutFM apps, so it carries over between them. Music listens
// go to /api/activity (4 minutes in or half way, as the apps count them),
// podcast positions to /api/progress, and neither for radio streams. Each
// entry's kind comes from its location, since a queue can mix them.

"use strict";

(function() {
    const FourMinutes = 240;      // seconds
    const SaveProgressEvery = 10; // seconds, podcasts
    const MinResume = 10;         // seconds; earlier counts as not started
    const Finished = 120;         // seconds from the end counts as finished
    const RetryKey = "takeout.listens";

    let audio = null;
    let queue = [];      // {creator, album, title, image, locations, etag, kind}
    let index = -1;
    let resumeAt = 0;    // position to start the current track at
    let unlocked = false;

    // what the audio element is playing, which stays put while the queue changes
    let playing = null;  // {track, type}, type being the track's kind

    // this play of it
    let started = null;  // Date it started
    let played = 0;      // seconds actually heard
    let lastTime = 0;
    let listened = false;
    let savedOffset = -1;
    let sinceSave = 0;
    let offsets = null;  // podcast progress from the server, by etag
    let locationIndex = 0;

    const $ = id => document.getElementById(id);

    // ---- server calls

    // resolves to true when the server took the change, false otherwise
    const patch = function(ops, keepalive = false) {
	return fetch("/api/playlist", {
	    method: "PATCH",
	    body: JSON.stringify(ops),
	    headers: {"Content-Type": "application/json-patch+json"},
	    keepalive: keepalive
	}).then(async resp => {
	    if (resp.status == 200) {
		load(await resp.json());
	    }
	    // 204 when the tracks didn't change, only the index or position
	    return resp.ok;
	}).catch(err => {
	    console.log("playlist: " + err);
	    return false;
	});
    };

    const statePatch = function(i, position) {
	return [
	    {op: "replace", path: "/index", value: i},
	    {op: "replace", path: "/position", value: Math.floor(position)}
	];
    };

    const saveState = function(keepalive = false) {
	const pos = audio && !isStream() ? audio.currentTime || 0 : 0;
	patch(statePatch(index, pos), keepalive);
    };

    const postListens = function(events) {
	return fetch("/api/activity", {
	    method: "POST",
	    body: JSON.stringify({TrackEvents: events}),
	    headers: {"Content-Type": "application/json"}
	}).then(resp => resp.ok);
    };

    // listens that couldn't be sent wait in localStorage for the next try
    const pendingListens = function() {
	try {
	    return JSON.parse(localStorage.getItem(RetryKey)) || [];
	} catch (e) {
	    return [];
	}
    };

    const setPendingListens = function(events) {
	try {
	    if (events.length == 0) {
		localStorage.removeItem(RetryKey);
	    } else {
		localStorage.setItem(RetryKey, JSON.stringify(events.slice(-500)));
	    }
	} catch (e) {
	}
    };

    const sendListen = function(event) {
	const events = pendingListens().concat([event]);
	postListens(events).then(ok => {
	    setPendingListens(ok ? [] : events);
	}).catch(() => setPendingListens(events));
    };

    const retryListens = function() {
	const events = pendingListens();
	if (events.length > 0) {
	    postListens(events).then(ok => {
		if (ok) {
		    setPendingListens([]);
		}
	    }).catch(() => {});
	}
    };

    const saveProgress = function(t, offset, duration) {
	offset = Math.floor(offset);
	duration = Math.floor(duration) || 0;
	if (!t || !t.etag || offset == savedOffset) {
	    return;
	}
	savedOffset = offset;
	const o = {ETag: t.etag, Offset: offset, Duration: duration, Date: new Date().toISOString()};
	if (offsets != null) {
	    offsets[t.etag] = o;
	}
	fetch("/api/progress", {
	    method: "POST",
	    body: JSON.stringify({Offsets: [o]}),
	    headers: {"Content-Type": "application/json"},
	    keepalive: true
	}).catch(err => console.log("progress: " + err));
    };

    const loadOffsets = async function() {
	if (offsets == null) {
	    offsets = {};
	    try {
		const data = await (await fetch("/api/progress")).json();
		(data.Offsets || []).forEach(o => {
		    const prev = offsets[o.ETag];
		    if (prev == null || o.Date > prev.Date) {
			offsets[o.ETag] = o;
		    }
		});
	    } catch (e) {
		console.log("progress: " + e);
	    }
	}
	return offsets;
    };

    // ---- queue

    // music, podcast or stream (anything not served by Takeout)
    const kind = function(locations) {
	const l = (locations && locations[0]) || "";
	if (l.startsWith("/api/tracks/")) {
	    return "music";
	} else if (l.startsWith("/api/episodes/")) {
	    return "podcast";
	}
	return "stream";
    };

    const load = function(data) {
	const tracks = (data.playlist && data.playlist.track) || [];
	queue = tracks.map(t => ({
	    creator: t.creator || "",
	    album: t.album || "",
	    title: t.title || "",
	    image: t.image || "",
	    locations: t.location || [],
	    etag: (t.identifier && t.identifier[0]) || "",
	    kind: kind(t.location)
	}));
	index = Math.min(data.index, queue.length - 1);
	renderQueue();
    };

    const fetchQueue = async function() {
	try {
	    const data = await (await fetch("/api/playlist")).json();
	    load(data);
	    if (index >= 0 && queue.length > 0) {
		resumeAt = data.position || 0;
		showTrack(queue[index]);
	    }
	} catch (e) {
	    console.log("playlist: " + e);
	}
    };

    // is the playing item (or else the queue) a live stream
    const isStream = function() {
	const t = playing ? playing.track : current();
	return t != null && t.kind == "stream";
    };
    const current = () => (index >= 0 && index < queue.length) ? queue[index] : null;

    // replace the queue with ref and play it
    const playRef = async function(ref) {
	const ok = await patch([
	    {op: "replace", path: "/playlist/track", value: []},
	    {op: "add", path: "/playlist/track/-", value: {"$ref": ref}},
	    ...statePatch(0, 0)
	]);
	if (ok && queue.length > 0) {
	    playIndex(0);
	}
    };

    const addRef = function(ref, next) {
	const at = next && queue.length > 0 ? String(index + 1) : "-";
	if (queue.length == 0) {
	    playRef(ref);
	    return;
	}
	patch([{op: "add", path: "/playlist/track/" + at, value: {"$ref": ref}}]);
	flash(next ? "Playing next" : "Added to the queue");
    };

    const remove = function(i) {
	const ops = [{op: "remove", path: "/playlist/track/" + i}];
	if (i < index) {
	    index--;
	    ops.push(...statePatch(index, audio.currentTime || 0));
	} else if (i == index) {
	    stop();
	    ops.push(...statePatch(-1, 0));
	}
	patch(ops);
    };

    const clear = function() {
	stop();
	patch([{op: "replace", path: "/playlist/track", value: []}, ...statePatch(-1, 0)]);
    };

    // shuffle the server's queue with moves, then play from the top
    const shuffle = function() {
	const order = queue.map((_, i) => i);
	for (let i = order.length - 1; i > 0; i--) {
	    const j = Math.floor(Math.random() * (i + 1));
	    [order[i], order[j]] = [order[j], order[i]];
	}
	const positions = queue.map((_, i) => i); // where each original track is now
	const ops = [];
	order.forEach((orig, k) => {
	    const from = positions.indexOf(orig);
	    if (from != k) {
		ops.push({op: "move", from: "/playlist/track/" + from, path: "/playlist/track/" + k});
		positions.splice(from, 1);
		positions.splice(k, 0, orig);
	    }
	});
	ops.push(...statePatch(0, 0));
	patch(ops).then(() => playIndex(0));
    };

    // ---- playback

    const resetPlay = function() {
	started = null;
	played = 0;
	lastTime = 0;
	listened = false;
	savedOffset = -1;
	sinceSave = 0;
	locationIndex = 0;
    };

    const playIndex = function(i, position = 0) {
	if (i < 0 || i >= queue.length) {
	    stop();
	    return;
	}
	finishPodcast();
	index = i;
	resumeAt = position;
	resetPlay();
	start(queue[i]);
	saveState();
    };

    const start = async function(t) {
	playing = {track: t, type: t.kind};
	showTrack(t);
	if (t.kind == "podcast" && resumeAt == 0) {
	    const o = (await loadOffsets())[t.etag];
	    if (o && o.Offset >= MinResume && (o.Duration <= 0 || o.Offset < o.Duration - Finished)) {
		resumeAt = o.Offset;
	    }
	}
	audio.src = t.locations[locationIndex] || "";
	play();
    };

    const play = function() {
	if (!audio.src && current()) {
	    start(current());
	    return;
	}
	audio.play().catch(err => {
	    // a newer load interrupted this one, or the browser wants a click first
	    if (err.name != "AbortError") {
		console.log("play: " + err.name);
	    }
	});
    };

    const pause = function() {
	audio.pause();
    };

    const toggle = function() {
	if (audio.paused) {
	    play();
	} else {
	    pause();
	}
    };

    const stop = function() {
	finishPodcast();
	audio.pause();
	audio.removeAttribute("src");
	audio.load();
	index = -1;
	playing = null;
	resetPlay();
	$("playing").style.display = "none";
	showQueue(false);
	if ("mediaSession" in navigator) {
	    navigator.mediaSession.metadata = null;
	}
    };

    const next = function() {
	if (index + 1 < queue.length) {
	    playIndex(index + 1);
	} else {
	    stop();
	    saveState();
	}
    };

    const prev = function() {
	if (!isStream() && audio.currentTime > 5) {
	    audio.currentTime = 0;
	} else if (index > 0) {
	    playIndex(index - 1);
	}
    };

    // save where a podcast episode got to; played to its end, it's finished
    const finishPodcast = function() {
	if (playing && playing.type == "podcast" && audio && audio.getAttribute("src")) {
	    const d = audio.duration;
	    saveProgress(playing.track, audio.ended ? d : audio.currentTime, d);
	}
    };

    // ---- audio events

    const onTime = function() {
	if (playing == null) {
	    return;
	}
	const t = playing.track;
	const now = audio.currentTime;
	const delta = now - lastTime;
	lastTime = now;
	if (!audio.paused && delta > 0 && delta < 2) {
	    // count only time heard, not seeks
	    played += delta;
	    sinceSave += delta;
	}
	showProgress();

	if (playing.type == "music" && !listened && t.etag) {
	    const d = audio.duration;
	    if (played >= FourMinutes || (d > 0 && played >= d / 2)) {
		listened = true;
		sendListen({Date: (started || new Date()).toISOString(), ETag: t.etag});
	    }
	} else if (playing.type == "podcast" && sinceSave >= SaveProgressEvery) {
	    sinceSave = 0;
	    saveProgress(t, now, audio.duration);
	}
    };

    const onLoaded = function() {
	if (resumeAt > 0 && !isStream()) {
	    audio.currentTime = resumeAt;
	}
	resumeAt = 0;
	lastTime = audio.currentTime;
	updateSession();
    };

    const onPlay = function() {
	if (started == null) {
	    started = new Date();
	}
	lastTime = audio.currentTime;
	showControls();
    };

    const onPause = function() {
	showControls();
	if (playing && !audio.ended && audio.getAttribute("src")) {
	    finishPodcast();
	    saveState();
	}
    };

    const onError = function() {
	if (playing == null || !audio.getAttribute("src")) {
	    return;
	}
	const t = playing.track;
	// try a station's other streams before giving up
	if (locationIndex + 1 < t.locations.length) {
	    locationIndex++;
	    audio.src = t.locations[locationIndex];
	    play();
	    return;
	}
	if (playing.type == "stream") {
	    flash("This station can't play here; try the TakeoutFM app");
	    audio.pause();
	} else {
	    flash("Couldn't play " + t.title);
	    if (index + 1 < queue.length) {
		setTimeout(next, 1500);
	    }
	}
    };

    // ---- display

    const showTrack = function(t) {
	$("playing").style.display = "block";
	$("np-title").textContent = t.title;
	$("np-artist").textContent = isStream() ? t.album || t.creator : t.creator;
	const cover = $("np-cover");
	if (t.image) {
	    cover.src = t.image;
	} else {
	    cover.removeAttribute("src");
	}
	$("progressbar").classList.toggle("live", isStream());
	showControls();
	showProgress();
	renderQueue();
	updateSession(t);
    };

    const showControls = function() {
	const paused = !audio || audio.paused;
	$("np-playpause").src = paused ? "/static/play_arrow-white-36dp.svg" : "/static/pause-white-36dp.svg";
	$("np-playpause").alt = paused ? "Play" : "Pause";
	if ("mediaSession" in navigator) {
	    navigator.mediaSession.playbackState = paused ? "paused" : "playing";
	}
    };

    const time = function(s) {
	if (!isFinite(s)) {
	    return "";
	}
	s = Math.floor(s);
	const m = Math.floor(s / 60);
	return m + ":" + String(s % 60).padStart(2, "0");
    };

    const showProgress = function() {
	const d = audio.duration;
	const pos = audio.currentTime || 0;
	if (isStream() || !isFinite(d) || d <= 0) {
	    $("progress").style.width = isStream() ? "100%" : "0%";
	    $("np-duration").textContent = isStream() ? "Live" : "";
	    return;
	}
	$("progress").style.width = (pos / d * 100) + "%";
	$("np-duration").textContent = time(pos) + " / " + time(d);
	if ("mediaSession" in navigator && navigator.mediaSession.setPositionState) {
	    try {
		navigator.mediaSession.setPositionState({duration: d, position: Math.min(pos, d)});
	    } catch (e) {
	    }
	}
    };

    let flashTimer = null;
    const flash = function(message) {
	const e = $("np-message");
	e.textContent = message;
	e.hidden = false;
	clearTimeout(flashTimer);
	flashTimer = setTimeout(() => { e.hidden = true; }, 4000);
    };

    const esc = function(s) {
	return String(s ?? "").replace(/[&<>"']/g, c => ({
	    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;"
	})[c]);
    };

    const renderQueue = function() {
	const rows = queue.map((t, i) =>
	    '<div class="parent' + (i == index ? ' playing' : '') + '">' +
		'<div><img data-action="entry" data-index="' + i + '" class="np-cover clickable" src="' +
		esc(t.image) + '" alt="" loading="lazy"></div>' +
		'<div class="left clickable" data-action="entry" data-index="' + i + '">' +
		'<div class="parent2">' +
		'<div class="track-title">' + esc(t.title) + '</div>' +
		'<div class="track-artist">' + esc(t.creator + (t.album ? " ~ " + t.album : "")) + '</div>' +
		'</div></div>' +
		'<div class="separator"></div>' +
		'<div class="right"><img class="np-control clickable" src="/static/clear-white-24dp.svg"' +
		' data-action="remove" data-index="' + i + '" alt="Remove"></div>' +
		'</div>');
	$("playlist").innerHTML =
	    '<h2>Queue</h2>' +
	    '<img class="np-control clickable" src="/static/shuffle-white-24dp.svg" data-action="shuffle" alt="Shuffle">' +
	    '<img class="np-control clickable" src="/static/clear-white-24dp.svg" data-action="clear" alt="Clear">' +
	    (rows.length > 0 ? rows.join("") : '<p class="empty">Nothing queued.</p>');
    };

    const showQueue = function(show) {
	$("playlist").style.display = show ? "block" : "none";
	$("main").style.display = show ? "none" : "block";
    };

    const toggleQueue = function() {
	showQueue($("playlist").style.display == "none");
    };

    const updateSession = function(t = current()) {
	if (!("mediaSession" in navigator) || t == null) {
	    return;
	}
	navigator.mediaSession.metadata = new MediaMetadata({
	    title: t.title,
	    artist: t.creator,
	    album: t.album,
	    artwork: t.image ? [{src: t.image}] : []
	});
    };

    // ---- input

    // the first click lets the audio element play later, after a fetch
    const unlock = function() {
	if (!unlocked) {
	    unlocked = true;
	    if (!audio.getAttribute("src")) {
		audio.play().catch(() => {});
		audio.pause();
	    }
	}
    };

    const onClick = function(event) {
	const e = event.target.closest("[data-playlist], [data-action]");
	if (e == null) {
	    return;
	}
	unlock();
	const i = parseInt(e.getAttribute("data-index"));
	const ref = e.getAttribute("data-ref");
	switch (e.getAttribute("data-playlist")) {
	case "add-ref": playRef(ref); break;
	case "append-ref": addRef(ref, false); break;
	case "prepend-ref": addRef(ref, true); break;
	default:
	    switch (e.getAttribute("data-action")) {
	    case "toggle": toggleQueue(); break;
	    case "playpause": toggle(); break;
	    case "next": next(); break;
	    case "shuffle": shuffle(); break;
	    case "clear": clear(); break;
	    case "entry": playIndex(i); break;
	    case "remove": remove(i); break;
	    default: return;
	    }
	}
	event.preventDefault();
    };

    const onKey = function(event) {
	const tag = document.activeElement && document.activeElement.tagName;
	if (tag == "INPUT" || tag == "TEXTAREA" || event.metaKey || event.ctrlKey || event.altKey ||
	    current() == null) {
	    return;
	}
	if (event.key == " ") {
	    toggle();
	} else if (event.key == "ArrowRight") {
	    next();
	} else if (event.key == "ArrowLeft") {
	    prev();
	} else {
	    return;
	}
	event.preventDefault();
    };

    const onSeek = function(event) {
	const d = audio.duration;
	if (isStream() || !isFinite(d) || d <= 0) {
	    return;
	}
	const bar = $("progressbar").getBoundingClientRect();
	audio.currentTime = (event.clientX - bar.left) / bar.width * d;
	lastTime = audio.currentTime;
    };

    const init = function() {
	audio = $("audio");
	audio.addEventListener("timeupdate", onTime);
	audio.addEventListener("loadedmetadata", onLoaded);
	audio.addEventListener("play", onPlay);
	audio.addEventListener("pause", onPause);
	audio.addEventListener("ended", next);
	audio.addEventListener("error", onError);
	audio.addEventListener("durationchange", showProgress);

	document.addEventListener("click", onClick);
	document.addEventListener("keydown", onKey);
	document.addEventListener("takeout:page", () => showQueue(false));
	$("progressbar").addEventListener("click", onSeek);
	window.addEventListener("pagehide", () => {
	    finishPodcast();
	    if (current() != null && !audio.paused) {
		saveState(true);
	    }
	});
	window.addEventListener("online", retryListens);

	if ("mediaSession" in navigator) {
	    const actions = {
		play: play,
		pause: pause,
		nexttrack: next,
		previoustrack: prev,
		seekto: d => {
		    if (!isStream()) {
			audio.currentTime = d.seekTime;
		    }
		}
	    };
	    for (const [action, handler] of Object.entries(actions)) {
		try {
		    navigator.mediaSession.setActionHandler(action, handler);
		} catch (e) {
		}
	    }
	}

	renderQueue();
	fetchQueue();
	retryListens();
    };

    if (document.readyState == "loading") {
	document.addEventListener("DOMContentLoaded", init);
    } else {
	init();
    }
})();
