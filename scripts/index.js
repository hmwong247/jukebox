// global
const session = {
	sessionID: "",
	roomID: "",
	username: "user",
	userList: {},
	playlist: [],
	userID: window.localStorage.getItem("userID"),
	hostID: "",
}

// const
const API_PATH = Object.freeze({
	// api
	NEW_USER: "/api/new-user",
	SESSION: "/api/session",
	CREATE: "/api/create",
	USERS: "/api/users",
	ENQUEUE: "/api/enqueue",
	STREAM: "/api/stream",
	STREAM_END: "/api/streamend",
	STREAM_PRELOAD: "/api/streampreload",
	// other
	JOIN: "/join",
	WEBSOCKET: "/ws",
	LOBBY: "/lobby",
	HOME: "/home",
})

/**
 * init
 */
addEventListener("DOMContentLoaded", async () => {
	await onDOMContentLoaded()
})

// this is detached for later use after bootstrap
async function onDOMContentLoaded() {
	//htmx.logAll()
}

/**
 * API calls
 */

async function fetchUserID() {
	if (window.localStorage.getItem("userID") == null) {
		const uid = await fetch(API_PATH.NEW_USER).then(res => {
			if (res.ok) {
				return res.text()
			} else {
				throw new Error(`fetchUserID, err:${res.statusText}`)
			}
		}).catch(err => {
			throw err
		})
		window.localStorage.setItem("userID", uid)
	}
}

async function fetchSessionID(form) {
	formData = new FormData(form)
	formData.append("user_id", window.localStorage.getItem("userID"))

	const sid = await fetch(API_PATH.SESSION, {
		method: "POST",
		body: formData
	}).then((res) => {
		if (res.ok) {
			return res.text()
		} else {
			throw new Error(`fetchSessionID, err:${res.statusText}`)
		}
	}).catch(err => {
		throw err
	})
	session.sessionID = sid
}

async function fetchSessionIDJoin(form) {
	formData = new FormData(form)
	formData.append("user_id", window.localStorage.getItem("userID"))
	formData.append("room_id", session.roomID)

	const sid = await fetch(API_PATH.SESSION, {
		method: "POST",
		body: formData
	}).then((res) => {
		if (res.ok) {
			return res.text()
		} else {
			throw new Error(`fetchSessionIDJoin, err:${res.statusText}`)
		}
	}).catch(err => {
		throw err
	})
	session.sessionID = sid
}

async function fetchRoomID() {
	const path = API_PATH.CREATE + "?sid=" + session.sessionID
	const rid = await fetch(path).then(res => {
		if (res.ok) {
			return res.text()
		} else {
			throw new Error(`fetchRoomID, err:${res.statusText}`)
		}
	}).catch(err => {
		throw err
	})
	session.roomID = rid
}

async function connectWebSocket() {
	const wsPath = "ws://" + document.location.host + API_PATH.WEBSOCKET + "?sid=" + session.sessionID
	await connectWS(wsPath).then(() => {
		console.log("ws connected")
	}).catch((err) => {
		console.error(`ws err: ${err}`)
	})
}

function fetchUserList() {
	// store the user list in session, we have to
	return new Promise((resolve, reject) => {
		const path = API_PATH.USERS + "?sid=" + session.sessionID
		fetch(path)
			.then(res => { resolve(res.json()) })
			.catch(err => { reject(err) })
	})
}

async function requestNewRoom(event, form) {
	// submit user profile and fetch session ID
	event.preventDefault()
	const cfgUsername = document.forms["user_profile"]["cfg_username"].value.trim()
	if (cfgUsername != null && cfgUsername.length) {
		session.username = cfgUsername
	} else {
		document.forms["user_profile"]["cfg_username"].value = session.username
	}

	// fetch user ID
	try {
		await fetchUserID()

		// fetch session ID
		await fetchSessionID(form)

		// fetch room ID
		await fetchRoomID()

		// connect to the websocket
		await connectWebSocket()
	} catch (err) {
		console.error(err)
		return
	}

	// render the lobby page
	const lobbyPath = API_PATH.LOBBY + "?sid=" + session.sessionID
	await htmx.ajax("GET", lobbyPath, { target: "#div_swap", })
	history.pushState({}, "", API_PATH.LOBBY)
	swapUsername(session.username)
	swapInviteLink(document.querySelector("#room_id").innerHTML)

	fetchUserList().then(data => session.userList = data)
}

async function requestJoinRoom(event, form) {
	// submit user profile and fetch session ID
	event.preventDefault()
	const cfgUsername = document.forms["user_profile"]["cfg_username"].value.trim()
	if (cfgUsername != null && cfgUsername.length) {
		session.username = cfgUsername
	} else {
		document.forms["user_profile"]["cfg_username"].value = session.username
	}

	// get room ID
	const queryString = window.location.search;
	const urlParams = new URLSearchParams(queryString);
	session.roomID = urlParams.get("rid")
	console.log(session.roomID)

	try {
		// fetch user ID
		await fetchUserID()

		// fetch session ID
		await fetchSessionIDJoin(form)

		// connect to the websocket
		await connectWebSocket()
	} catch (err) {
		console.error(err)
		return
	}

	// render the lobby page
	const lobbyPath = API_PATH.LOBBY + "?sid=" + session.sessionID
	await htmx.ajax("GET", lobbyPath, { target: "#div_swap", }).catch(err => { console.error(err); return })
	history.pushState({}, "", API_PATH.LOBBY)
	swapUsername(session.username)
	swapInviteLink(document.querySelector("#room_id").innerHTML)

	fetchUserList().then(data => session.userList = data)
}

async function submitURL(event, form) {
	event.preventDefault()
	formData = new FormData(form)
	formData.append("user_id", window.localStorage.getItem("userID"))

	await fetch(API_PATH.ENQUEUE + "?sid=" + session.sessionID, {
		method: "POST",
		body: formData
	}).then((res) => {
		if (res.ok) {
			console.log(res.status)
		} else {
			console.log(res.status)
		}
	}).catch(err => {
		throw err
	})
}

/**
 * UI/UX related functions
 */

// redirect to home page
function autoResetPage() {
	htmx.ajax("GET", API_PATH.HOME, { target: "#div_swap" }).catch(err => { console.error(err); return })
}

function copyLink() {
	const link = document.querySelector("#room_id").innerHTML
	if (navigator.clipboard && window.isSecureContext) {
		navigator.clipboard.writeText(link)
			.then(data => { console.log(data) })
			.catch(err => { console.log(err) })
	} else {
		const textArea = document.createElement("textarea")
		textArea.value = link
		textArea.style.position = "fixed" // avoid scrolling to bottom
		document.body.appendChild(textArea)
		textArea.select()

		try {
			document.execCommand("copy")
		} catch (err) {
			console.log(err)
		} finally {
			textArea.remove()
		}
	}
}

function swapInviteLink(code) {
	const link = document.location.origin + API_PATH.JOIN + "?rid=" + code
	document.querySelector("#room_id").innerHTML = link
}

function swapUsername(name) {
	document.querySelector("#username").innerHTML = name
}

function swapHost(host) {
	document.querySelector("#room_host").innerHTML = host
}

function swapRoomCapacity(size) {
	document.querySelector("#room_capacity").innerHTML = size
}

function swapUserList(id, username, isAdd = true) {
	if (isAdd) {
		const row = `<li id=\"uid${id}\">user name: ${username}<br>id: ${id}</li>`
		htmx.swap("#room_user_list", row, { swapStyle: "beforeend" })
	} else {
		htmx.remove(htmx.find(`#uid${id}`))
	}
}

// WIP
// id is needed for remove, swap
function swapPlaylist(infojson, shift = false) {
	if (shift) {
		htmx.find(`#pl${infojson.ID}`).remove()
	} else {
		const row = `<li id='pl${infojson.ID}'>${JSON.stringify(infojson)}</li>`
		htmx.swap("#room_queue_list", row, { swapStyle: "beforeend" })
	}
}

/**
 * web audio
 */

const mp = {
	/** @type {HTMLAudioElement} elem */
	elem: null,
	/** @type {Boolean} running */
	running: false,
	/** @type {AudioContext} ctx */
	ctx: null,
	/** @type {MediaStrema} stream */
	stream: null,
	currentSegment: null,
	totalSegement: null,
}

function initMP() {
	mp.elem = document.querySelector("#player")
	mp.elem.removeEventListener('loadstart', mploadstart)
	mp.elem.removeEventListener('timeupdate', mptimeupdate)
	mp.elem.removeEventListener('ended', mpended)

	mp.elem.src = API_PATH.STREAM + "?sid=" + session.sessionID
	mp.elem.addEventListener('loadstart', mploadstart)
}

async function mploadstart() {
	console.log(`loadstart`)
	mp.elem.play()
	mp.running = true
}

async function mptimeupdate() {
	if (mp.elem.currentTime / mp.elem.duration >= 0.5 && session.playlist.length > 1) {
		mp.elem.removeEventListener('timeupdate', mptimeupdate)

		const url = API_PATH.STREAM_PRELOAD + "?sid=" + session.sessionID
		const res = await fetch(url)
		if (res.ok) {
			// const s = await res.json()
			// if (s == false) {

			// }
		}
	}
}

async function mpended() {
	console.log(`ended`)
	mp.elem.pause()
	mp.elem.currentTime = 0
	const url = API_PATH.STREAM_END + "?sid=" + session.sessionID
	const response = fetch(url)
	// .then wait for server to reponse the next audio is ready if the queue is not size of 0

	const endedJson = session.playlist.shift()
	swapPlaylist(endedJson, true)

	if (session.playlist.length > 0) {
		// wait for 20ms to switch audio
		// await new Promise(r => setTimeout(r, 20))
		// if ok
		playAudio()
	} else {
		mp.elem.pause()
		mp.elem.removeAttribute("src")
		mp.elem.load()
		mp.running = false
	}
}


async function playAudio() {
	if (mp.elem && mp.running && mp.elem.buffered.length > 0 && mp.elem.currentTime != 0) {
		// if (mp.elem.buffered.length > 0 && mp.elem.currentTime != 0 && !mp.elem.paused) return
		return
	}
	// 	// implement a retry mechanism
	// const url = API_PATH.STREAM_END + "?sid=" + session.sessionID
	// const serverResp = await fetch(url, {method: "HEAD"}).then(r => r.ok)
	// if(!serverResp) {
	// 	return
	// }
	initMP()
	mp.elem.addEventListener('timeupdate', mptimeupdate)
	mp.elem.addEventListener('ended', mpended, { once: true })
}
