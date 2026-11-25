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
//
// Live radio plays off the queue: tuning in to a live station follows its
// schedule (/api/radio/{id}/schedule), joining the track on now part way
// through, and leaves the queue as it was. See RADIO.md.

"use strict";

(function() {
    const FourMinutes = 240;      // seconds
    const SaveProgressEvery = 10; // seconds, podcasts
    const MinResume = 10;         // seconds; earlier counts as not started
    const Finished = 120;         // seconds from the end counts as finished
    const OffsetsMaxAge = 10000;  // ms before podcast progress is fetched again
    const RetryKey = "takeout.listens";
    const LiveKey = "takeout.live";   // the live station tuned in, for a reload
    const LiveDrift = 5;              // seconds off the schedule before correcting
    const LiveBehind = 300;           // seconds behind live a resume may stay
    const LiveRefill = 2;             // tracks left before fetching more
    const LivePreload = 30;           // seconds left when the next track loads

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
    let offsetsAt = 0;   // when it was fetched (ms)
    let locationIndex = 0;

    // live radio: {id, name, image, shows, tracks, i, skew, behind, pausedAt,
    // pending}, or null. tracks are the scheduled tracks from the one
    // playing (i), skew the server's clock less this one's (ms).
    let live = null;
    let spare = null;    // {track, el}: the next live track, loading
    let fetchingLive = false;

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
	if (live) {
	    return; // live radio isn't in the queue
	}
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
	postOffset(t.etag, offset, duration);
    };

    const postOffset = function(etag, offset, duration) {
	const o = {ETag: etag, Offset: offset, Duration: duration, Date: new Date().toISOString()};
	if (offsets != null) {
	    offsets[etag] = o;
	}
	return fetch("/api/progress", {
	    method: "POST",
	    body: JSON.stringify({Offsets: [o]}),
	    headers: {"Content-Type": "application/json"},
	    keepalive: true
	}).catch(err => console.log("progress: " + err));
    };

    // Podcast progress, fetched again when it's older than OffsetsMaxAge:
    // the apps save it too. The newest of each episode's offsets wins, so
    // this page's own saves aren't lost to a slower server.
    const loadOffsets = async function() {
	if (offsets == null) {
	    offsets = {};
	}
	if (Date.now() - offsetsAt > OffsetsMaxAge) {
	    offsetsAt = Date.now();
	    try {
		const data = await (await fetch("/api/progress")).json();
		(data.Offsets || []).forEach(o => {
		    const prev = offsets[o.ETag];
		    if (prev == null || Date.parse(o.Date) > Date.parse(prev.Date)) {
			offsets[o.ETag] = o;
		    }
		});
	    } catch (e) {
		offsetsAt = 0;
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
	    background: t.background || "",
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

    // replace the queue with ref and play it, from start or shuffled
    const playRef = async function(ref, start = 0, shuffled = false, fromStart = false) {
	const ok = await patch([
	    {op: "replace", path: "/playlist/track", value: []},
	    {op: "add", path: "/playlist/track/-", value: {"$ref": ref}},
	    ...statePatch(0, 0)
	]);
	if (!ok || queue.length == 0) {
	    return;
	}
	if (shuffled) {
	    shuffle();
	} else {
	    playIndex(Math.min(start, queue.length - 1), fromStart ? -1 : 0);
	}
    };

    // a row in a list (data-list-ref) plays the whole list from that row,
    // as the apps play an album from the track picked
    const playRow = function(e) {
	const list = e.closest("[data-list-ref]");
	if (list && e.hasAttribute("data-track")) {
	    const rows = Array.from(list.querySelectorAll("[data-track]"));
	    playRef(list.getAttribute("data-list-ref"), rows.indexOf(e));
	} else {
	    playRef(e.getAttribute("data-ref"), 0, e.hasAttribute("data-shuffle"),
		    e.hasAttribute("data-from-start"));
	}
    };

    // favorite buttons: data-url is the favorite's API path, aria-pressed
    // whether it is one now
    const favorite = async function(e) {
	const on = e.getAttribute("aria-pressed") == "true";
	try {
	    const resp = await fetch(e.getAttribute("data-url"), {method: on ? "DELETE" : "PUT"});
	    if (resp.ok) {
		e.setAttribute("aria-pressed", String(!on));
		const label = e.querySelector(".label");
		if (label) {
		    label.textContent = on ? "Favorite" : "Unfavorite";
		}
	    }
	} catch (err) {
	    console.log("favorite: " + err);
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

    // ---- live radio

    const isLive = () => live != null;
    const serverNow = () => Date.now() + (live ? live.skew : 0);

    const rememberLive = function(station) {
	try {
	    if (station) {
		localStorage.setItem(LiveKey, JSON.stringify(station));
	    } else {
		localStorage.removeItem(LiveKey);
	    }
	} catch (e) {
	}
    };

    const rememberedLive = function() {
	try {
	    return JSON.parse(localStorage.getItem(LiveKey));
	} catch (e) {
	    return null;
	}
    };

    // a scheduled track, shaped like a queue entry
    const scheduled = function(x, station) {
	const show = (station.Shows || []).find(s => s.Name == x.Show) || {};
	return {
	    creator: x.Artist || "",
	    album: x.Release || "",
	    title: x.Title || "",
	    image: x.Image || show.Image || station.Image || "",
	    background: "",
	    locations: [x.Location],
	    etag: x.ETag || "",
	    kind: "music",
	    start: Date.parse(x.Start),
	    end: Date.parse(x.End),
	    show: x.Show || ""
	};
    };

    // the station's schedule from start (a server time in ms, or now)
    const fetchSchedule = async function(id, start) {
	let url = "/api/radio/" + id + "/schedule";
	if (start) {
	    url += "?start=" + encodeURIComponent(new Date(start).toISOString());
	}
	const sent = Date.now();
	const resp = await fetch(url);
	if (!resp.ok) {
	    throw new Error("schedule " + resp.status);
	}
	const data = await resp.json();
	const skew = data.Time - (sent + Date.now()) / 2;
	const station = (data.Stations || [])[0];
	return {skew: skew, station: station,
		tracks: station ? (station.Tracks || []).map(x => scheduled(x, station)) : []};
    };

    // add the tracks after the last one held
    const appendLive = function(tracks) {
	const last = live.tracks[live.tracks.length - 1];
	const after = last ? last.end - 1000 : -Infinity;
	tracks.forEach(t => {
	    if (t.start >= after && !live.tracks.some(x => x.start == t.start)) {
		live.tracks.push(t);
	    }
	});
    };

    // fetch more of the schedule, from the end of what's held
    const refillLive = async function() {
	if (!live || fetchingLive) {
	    return;
	}
	const station = live;
	const last = live.tracks[live.tracks.length - 1];
	fetchingLive = true;
	try {
	    const got = await fetchSchedule(station.id, last ? last.end : 0);
	    if (live === station) {
		live.skew = got.skew;
		appendLive(got.tracks);
		renderQueue();
		showLive();
	    }
	} catch (e) {
	    console.log("radio: " + e); // keep playing what's held
	} finally {
	    fetchingLive = false;
	}
    };

    // tune in to a live station; play is false after a reload, when the
    // browser won't play until a click
    const tuneIn = async function(id, name, image, play = true) {
	if (current() != null && playing != null && !isLive()) {
	    // leave the queue where it was
	    finishPodcast();
	    saveState();
	}
	dropSpare();
	audio.pause();
	live = {id: id, name: name || "", image: image || "", tracks: [], i: -1, skew: 0,
		behind: false, pausedAt: null, pending: !play};
	rememberLive({id: id, name: name, image: image});
	setSessionActions();
	try {
	    const got = await fetchSchedule(id, 0);
	    if (live == null || live.id != id) {
		return;
	    }
	    live.skew = got.skew;
	    if (got.station) {
		live.name = got.station.Name;
		live.image = got.station.Image || live.image;
	    }
	    live.tracks = got.tracks;
	} catch (e) {
	    console.log("radio: " + e);
	}
	if (live.tracks.length == 0) {
	    flash("Nothing's on " + (live.name || "this station") + " right now");
	    leaveLive();
	    return;
	}
	if (play) {
	    jumpLive();
	} else {
	    // show what's on; a click on play joins it
	    live.i = Math.max(0, onNow());
	    playing = null;
	    showTrack(live.tracks[live.i]);
	}
    };

    // the index of the track on now, or -1
    const onNow = function() {
	const now = serverNow();
	return live.tracks.findIndex(t => t.start <= now && now < t.end);
    };

    // play the track on now, part way through
    const jumpLive = async function() {
	if (!live) {
	    return;
	}
	live.behind = false;
	live.pausedAt = null;
	live.pending = false;
	let i = onNow();
	if (i < 0) {
	    live.tracks = [];
	    await refillLive();
	    i = live ? onNow() : -1;
	}
	if (i < 0) {
	    flash("Couldn't find what's on now");
	    return;
	}
	const t = live.tracks[i];
	playLive(i, Math.max(0, (serverNow() - t.start) / 1000));
    };

    // play live.tracks[i] from offset seconds
    const playLive = function(i, offset) {
	finishPodcast();
	live.tracks = live.tracks.slice(i);
	live.i = 0;
	const t = live.tracks[0];
	if (spare && spare.track === t && offset < 1) {
	    swapSpare();
	} else {
	    dropSpare();
	    resumeAt = offset >= 1 ? offset : 0;
	    resetPlay();
	    start(t);
	}
	// more of the schedule with each track, as the apps do
	refillLive();
    };

    // the next scheduled track: from the top, or at its offset when playback
    // is more than LiveDrift off the schedule
    const nextLive = async function() {
	if (live.tracks.length - 1 - live.i <= LiveRefill) {
	    await refillLive();
	    if (!live) {
		return;
	    }
	}
	const i = live.i + 1;
	if (i >= live.tracks.length) {
	    jumpLive();
	    return;
	}
	const late = (serverNow() - live.tracks[i].start) / 1000;
	if (!live.behind && late > LiveDrift) {
	    jumpLive();
	    return;
	}
	playLive(i, 0);
    };

    // resume after a pause: carry on behind live if it's not too far,
    // otherwise join what's on now
    const resumeLive = function() {
	const t = live.tracks[live.i];
	if (live.pending || t == null || !audio.getAttribute("src")) {
	    jumpLive();
	    return;
	}
	const at = t.start + (audio.currentTime || 0) * 1000;
	const behind = (serverNow() - at) / 1000;
	const left = (t.end - at) / 1000;
	if (behind > Math.max(LiveBehind, left)) {
	    jumpLive();
	    return;
	}
	live.pausedAt = null;
	live.behind = live.behind || behind > LiveDrift;
	showLive();
	audio.play().catch(err => console.log("play: " + err.name));
    };

    // stop the radio and go back to the queue
    const leaveLive = function() {
	if (!live) {
	    return;
	}
	live = null;
	rememberLive(null);
	dropSpare();
	audio.pause();
	audio.removeAttribute("src");
	audio.load();
	playing = null;
	resetPlay();
	setSessionActions();
	const t = current();
	if (t != null) {
	    showTrack(t);
	} else {
	    $("playing").hidden = true;
	    showNowPlaying(false);
	    setAll("queue-title", e => { e.textContent = "Queue"; });
	    setAll("queue-actions", e => { e.hidden = false; });
	    setAll("live-actions", e => { e.hidden = true; });
	    setAll("liveprogress", e => { e.hidden = true; });
	    showLive();
	    renderQueue();
	}
    };

    // load the next live track while this one ends, for a short gap
    const preloadLive = function() {
	const t = live.tracks[live.i + 1];
	if (t == null || (spare && spare.track === t)) {
	    return;
	}
	dropSpare();
	const el = new Audio();
	el.preload = "auto";
	el.src = t.locations[0];
	spare = {track: t, el: el};
    };

    const dropSpare = function() {
	if (spare) {
	    spare.el.removeAttribute("src");
	    spare.el.load();
	    spare = null;
	}
    };

    // play the loaded next track in place of the audio element
    const swapSpare = function() {
	const old = audio;
	unbind(old);
	old.pause();
	old.removeAttribute("src");
	old.load();
	audio = spare.el;
	const t = spare.track;
	spare = null;
	bind(audio);
	resetPlay();
	resumeAt = 0;
	playing = {track: t, type: t.kind};
	showTrack(t);
	lastTime = audio.currentTime || 0;
	play();
    };

    // the parts of the player that differ for live radio
    const showLive = function() {
	setAll("behind", e => { e.hidden = !(live && live.behind); });
	const n = live && live.tracks[live.i + 1];
	setAll("next", e => {
	    e.hidden = !n;
	    e.textContent = n ? "Next: " + [n.creator, n.title].filter(x => x).join(" \u2013 ") : "";
	});
    };

    const setSessionActions = function() {
	if (!("mediaSession" in navigator)) {
	    return;
	}
	const actions = {
	    play: userPlay,
	    pause: pause,
	    nexttrack: isLive() ? null : next,
	    previoustrack: isLive() ? null : prev,
	    seekto: isLive() ? null : d => {
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
    };

    // ---- radio page: what's on each live station

    let radioTimer = null;

    const showOnNow = async function() {
	clearTimeout(radioTimer);
	const lines = document.querySelectorAll("#main [data-radio-now]");
	if (lines.length == 0) {
	    return;
	}
	let data;
	const sent = Date.now();
	try {
	    data = await (await fetch("/api/radio/schedule")).json();
	} catch (e) {
	    console.log("radio: " + e);
	    return;
	}
	const now = data.Time + (Date.now() - sent) / 2;
	let soonest = Infinity;
	const byID = {};
	(data.Stations || []).forEach(st => {
	    const x = (st.Tracks || []).find(x => Date.parse(x.Start) <= now && now < Date.parse(x.End));
	    if (x) {
		byID[st.ID] = x;
		soonest = Math.min(soonest, Date.parse(x.End));
	    }
	});
	lines.forEach(e => {
	    const x = byID[e.getAttribute("data-radio-now")];
	    e.textContent = x ? [x.Artist + " \u2013 " + x.Title, x.Show].filter(s => s).join(" \u00b7 ") : "";
	});
	if (isFinite(soonest)) {
	    radioTimer = setTimeout(showOnNow, Math.max(1000, soonest - now + 1000));
	}
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
	if (live) {
	    live = null;
	    rememberLive(null);
	    dropSpare();
	    setSessionActions();
	}
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
	if (resumeAt < 0) {
	    resumeAt = 0; // start over
	} else if (t.kind == "podcast" && resumeAt == 0) {
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

    // play, from the play button or media keys
    const userPlay = function() {
	if (live) {
	    resumeLive();
	} else {
	    play();
	}
    };

    const toggle = function() {
	if (audio.paused) {
	    userPlay();
	} else {
	    pause();
	}
    };

    const stop = function() {
	if (live) {
	    live = null;
	    rememberLive(null);
	    dropSpare();
	    setSessionActions();
	}
	finishPodcast();
	audio.pause();
	audio.removeAttribute("src");
	audio.load();
	index = -1;
	playing = null;
	resetPlay();
	$("playing").hidden = true;
	showNowPlaying(false);
	if ("mediaSession" in navigator) {
	    navigator.mediaSession.metadata = null;
	}
    };

    const next = function() {
	if (live) {
	    nextLive();
	    return;
	}
	if (index + 1 < queue.length) {
	    playIndex(index + 1);
	} else {
	    stop();
	    saveState();
	}
    };

    const prev = function() {
	if (live) {
	    return;
	}
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

	if (live && !live.pending) {
	    const d = audio.duration;
	    if (isFinite(d) && d - now <= LivePreload) {
		preloadLive();
	    }
	}

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
	if (live && !audio.ended) {
	    live.pausedAt = Date.now();
	}
	showControls();
	showEpisode();
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
	} else if (live) {
	    flash("Couldn't play " + t.title);
	    setTimeout(() => { if (live) { nextLive(); } }, 1500);
	} else {
	    flash("Couldn't play " + t.title);
	    if (index + 1 < queue.length) {
		setTimeout(next, 1500);
	    }
	}
    };

    // ---- display

    // every element showing a part of what's playing (bar and Now Playing)
    const setAll = function(name, fn) {
	document.querySelectorAll('[data-np="' + name + '"]').forEach(fn);
    };

    const showTrack = function(t) {
	const stream = t.kind == "stream";
	const onAir = stream || isLive();
	$("playing").hidden = false;
	setAll("title", e => { e.textContent = t.title; });
	// a station's name is its album; a track's second line is its artist;
	// on live radio the third line is the station and show
	setAll("artist", e => { e.textContent = stream ? t.album || t.creator : t.creator; });
	setAll("album", e => {
	    e.textContent = isLive() ? [live.name, t.show].filter(x => x).join(" \u00b7 ") :
		stream ? "" : t.album;
	});
	setAll("cover", e => {
	    if (t.image) {
		e.src = t.image;
	    } else {
		e.removeAttribute("src");
	    }
	});
	setAll("background", e => {
	    e.hidden = !t.background;
	    if (t.background) {
		e.src = t.background;
	    } else {
		e.removeAttribute("src");
	    }
	});
	setAll("seekable", e => { e.hidden = onAir; });
	setAll("live", e => { e.hidden = !onAir; });
	setAll("queue-title", e => { e.textContent = isLive() ? "Up next on " + live.name : "Queue"; });
	setAll("queue-actions", e => { e.hidden = isLive(); });
	setAll("live-actions", e => { e.hidden = !isLive(); });
	setAll("liveprogress", e => { e.hidden = !isLive(); });
	showLive();
	showControls();
	showProgress();
	renderQueue();
	updateSession(t);
    };

    const showControls = function() {
	const paused = !audio || audio.paused;
	const icon = paused ? "play" : isStream() ? "stop" : "pause";
	setAll("playicon", e => e.setAttribute("href", "#i-" + icon));
	document.querySelectorAll('[data-action="playpause"]').forEach(b => {
	    b.title = paused ? "Play" : isStream() ? "Stop" : "Pause";
	});
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

    let dragging = false; // a seek slider is being dragged

    const showTimes = function(pos, d) {
	setAll("pos", e => { e.textContent = time(pos); });
	setAll("dur", e => { e.textContent = time(d); });
	setAll("time", e => { e.textContent = isFinite(d) ? time(pos) + " / " + time(d) : ""; });
    };

    const showProgress = function() {
	const d = audio.duration;
	const pos = audio.currentTime || 0;
	if (live) {
	    // what's heard, not the schedule, so it's right behind live too
	    const known = isFinite(d) && d > 0;
	    setAll("livebar", e => { e.value = known ? Math.round(pos / d * 1000) : 0; });
	    setAll("remaining", e => { e.textContent = known ? "\u2212" + time(Math.max(0, d - pos)) : ""; });
	    if (known && "mediaSession" in navigator && navigator.mediaSession.setPositionState) {
		try {
		    navigator.mediaSession.setPositionState({duration: d, position: Math.min(pos, d)});
		} catch (e) {
		}
	    }
	    return;
	}
	if (dragging) {
	    return;
	}
	if (isStream() || !isFinite(d) || d <= 0) {
	    document.querySelectorAll("input.seek").forEach(e => { e.value = 0; });
	    showTimes(NaN, NaN);
	    return;
	}
	document.querySelectorAll("input.seek").forEach(e => { e.value = Math.round(pos / d * 1000); });
	showTimes(pos, d);
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

    const clock = ms => new Date(ms).toLocaleTimeString([], {hour: "numeric", minute: "2-digit"});

    const renderQueue = function() {
	if (live) {
	    // what's scheduled; it can't be changed
	    const rows = live.tracks.slice(Math.max(0, live.i)).map((t, k) =>
		'<li class="track scheduled' + (k == 0 ? ' current' : '') + '">' +
		    '<span class="num">' + (k == 0 ?
			'<svg class="playing-icon" aria-label="Playing"><use href="#i-play"></use></svg>' : '') + '</span>' +
		    '<img class="thumb" src="' + esc(t.image) + '" alt="" loading="lazy">' +
		    '<div class="text"><div class="title">' + esc(t.title) + '</div>' +
		    '<div class="sub">' + esc([t.creator, t.show].filter(x => x).join(" \u00b7 ")) + '</div></div>' +
		    '<span class="when">' + esc(clock(t.start - live.skew)) + '</span>' +
		'</li>');
	    $("queue").innerHTML = rows.join("");
	    return;
	}
	const rows = queue.map((t, i) =>
	    '<li class="track' + (i == index ? ' current' : '') + '" data-action="entry" data-index="' + i + '">' +
		'<span class="num">' + (i == index ?
		    '<svg class="playing-icon" aria-label="Playing"><use href="#i-play"></use></svg>' : (i + 1)) + '</span>' +
		'<img class="thumb" src="' + esc(t.image) + '" alt="" loading="lazy">' +
		'<div class="text"><div class="title">' + esc(t.title) + '</div>' +
		'<div class="sub">' + esc([t.creator, t.album].filter(x => x).join(" \u00b7 ")) + '</div></div>' +
		'<a class="add" href="#" data-action="remove" data-index="' + i + '" title="Remove">' +
		'<svg aria-label="Remove"><use href="#i-clear"></use></svg></a>' +
	    '</li>');
	$("queue").innerHTML = rows.length > 0 ? rows.join("") : '<li class="empty">Nothing queued.</li>';
    };

    const showNowPlaying = function(show) {
	$("nowplaying").hidden = !show;
	$("main").hidden = show;
	if (show) {
	    window.scrollTo(0, 0);
	    const row = document.querySelector("#queue .current");
	    if (row) {
		row.scrollIntoView({block: "nearest"});
	    }
	}
    };

    // Now Playing gets a history entry, so Back closes it
    const toggleNowPlaying = function() {
	if (!$("nowplaying").hidden) {
	    if (history.state && history.state.nowplaying) {
		history.back();
	    } else {
		showNowPlaying(false);
	    }
	} else if (current() != null || live) {
	    history.replaceState(Object.assign({}, history.state, {scroll: window.scrollY}), "");
	    history.pushState({scroll: 0, nowplaying: true}, "", location.href);
	    showNowPlaying(true);
	}
    };

    // ---- podcast episode pages: progress, Resume, Start Over, Mark as Played

    const episodeState = function(o) {
	if (o == null) {
	    return {};
	}
	const finished = o.Duration > 0 && o.Offset >= o.Duration - Finished;
	const started = !finished && o.Offset >= MinResume;
	return {finished: finished, started: started};
    };

    const showEpisode = async function() {
	const box = document.querySelector("#main [data-episode]");
	if (box == null) {
	    return;
	}
	const o = (await loadOffsets())[box.getAttribute("data-episode")];
	const st = episodeState(o);
	const page = box.closest(".page") || document;
	const status = page.querySelector("[data-episode-status]");
	if (status) {
	    status.textContent = st.finished ? " \u00b7 Played" :
		st.started ? " \u00b7 " + Math.max(1, Math.round((o.Duration - o.Offset) / 60)) + " min left" : "";
	}
	const bar = page.querySelector("[data-episode-bar]");
	if (bar) {
	    bar.hidden = !st.started;
	    bar.value = st.started && o.Duration > 0 ? o.Offset / o.Duration : 0;
	}
	box.querySelectorAll("[data-play-label]").forEach(e => {
	    e.textContent = st.started ? "Resume" : "Play";
	});
	box.querySelectorAll("[data-if-started]").forEach(e => { e.hidden = !st.started; });
	const mark = box.querySelector('[data-action="mark-played"]');
	if (mark) {
	    // the duration is known once the episode has been played here or elsewhere
	    mark.hidden = !(o && o.Duration > 0);
	    mark.querySelector(".label").textContent = st.finished ? "Mark as Not Played" : "Mark as Played";
	}
    };

    const markPlayed = async function(e) {
	const etag = e.closest("[data-episode]").getAttribute("data-episode");
	const o = (await loadOffsets())[etag];
	if (o && o.Duration > 0) {
	    await postOffset(etag, episodeState(o).finished ? 0 : o.Duration, o.Duration);
	    showEpisode();
	}
    };

    const updateSession = function(t = current()) {
	if (!("mediaSession" in navigator) || t == null) {
	    return;
	}
	navigator.mediaSession.metadata = new MediaMetadata({
	    title: t.title,
	    artist: t.creator,
	    album: isLive() ? [live.name, t.show].filter(x => x).join(" \u00b7 ") : t.album,
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
	case "add-ref": playRow(e); break;
	case "append-ref": addRef(ref, false); break;
	case "prepend-ref": addRef(ref, true); break;
	default:
	    switch (e.getAttribute("data-action")) {
	    case "toggle":
	    case "nowplaying": toggleNowPlaying(); break;
	    case "mark-played": markPlayed(e); break;
	    case "prev": prev(); break;
	    case "favorite": favorite(e); break;
	    case "playpause": toggle(); break;
	    case "next": next(); break;
	    case "shuffle": shuffle(); break;
	    case "clear": clear(); break;
	    case "entry": playIndex(i); break;
	    case "live": tuneIn(e.getAttribute("data-live"), e.getAttribute("data-name"),
				e.getAttribute("data-image")); break;
	    case "jumplive": jumpLive(); break;
	    case "leavelive": leaveLive(); break;
	    case "remove": remove(i); break;
	    default: return;
	    }
	}
	event.preventDefault();
    };

    const onKey = function(event) {
	const tag = document.activeElement && document.activeElement.tagName;
	if (tag == "INPUT" || tag == "TEXTAREA" || event.metaKey || event.ctrlKey || event.altKey ||
	    (current() == null && !live)) {
	    return;
	}
	if (event.key == " ") {
	    toggle();
	} else if (live) {
	    return;
	} else if (event.key == "ArrowRight") {
	    next();
	} else if (event.key == "ArrowLeft") {
	    prev();
	} else {
	    return;
	}
	event.preventDefault();
    };

    // seek sliders: show the time while dragging, seek on release
    const onSeekInput = function(event) {
	const e = event.target;
	const d = audio.duration;
	if (!e.classList || !e.classList.contains("seek") || !isFinite(d)) {
	    return;
	}
	dragging = true;
	document.querySelectorAll("input.seek").forEach(s => { s.value = e.value; });
	showTimes(e.value / 1000 * d, d);
    };

    const onSeekChange = function(event) {
	const e = event.target;
	if (!e.classList || !e.classList.contains("seek")) {
	    return;
	}
	dragging = false;
	const d = audio.duration;
	if (!isStream() && isFinite(d) && d > 0) {
	    audio.currentTime = e.value / 1000 * d;
	    lastTime = audio.currentTime;
	}
    };

    const init = function() {
	audio = $("audio");
	bind(audio);

	document.addEventListener("click", onClick);
	document.addEventListener("keydown", onKey);
	document.addEventListener("takeout:page", () => {
	    showNowPlaying(false);
	    showEpisode();
	    showOnNow();
	});
	// Back or Forward to the same page: Now Playing open or closed
	document.addEventListener("takeout:history", event => {
	    showNowPlaying(!!(event.detail && event.detail.nowplaying) && (current() != null || live != null));
	});
	document.addEventListener("input", onSeekInput);
	document.addEventListener("change", onSeekChange);
	window.addEventListener("pagehide", () => {
	    finishPodcast();
	    if (current() != null && !live && !audio.paused) {
		saveState(true);
	    }
	});
	window.addEventListener("online", retryListens);

	setSessionActions();

	renderQueue();
	fetchQueue().then(() => {
	    // tuned in to a live station before the reload: show what's on
	    const station = rememberedLive();
	    if (station && station.id) {
		tuneIn(station.id, station.name, station.image, false);
	    }
	});
	retryListens();
	showEpisode();
	showOnNow();
    };

    // the audio element's events; a live track change swaps the element
    const events = () => ({
	timeupdate: onTime,
	loadedmetadata: onLoaded,
	play: onPlay,
	pause: onPause,
	ended: next,
	error: onError,
	durationchange: showProgress
    });

    const bind = function(el) {
	for (const [name, fn] of Object.entries(events())) {
	    el.addEventListener(name, fn);
	}
    };

    const unbind = function(el) {
	for (const [name, fn] of Object.entries(events())) {
	    el.removeEventListener(name, fn);
	}
    };

    if (document.readyState == "loading") {
	document.addEventListener("DOMContentLoaded", init);
    } else {
	init();
    }
})();
