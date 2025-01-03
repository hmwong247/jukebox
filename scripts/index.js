// global
const session = {
	sessionID: "",
	roomID: "",
	username: "user",
	userList: {},
}

const API_PATH = {
	NEW_USER: "/api/new-user",
	CREATE: "/api/create",
	SESSION: "/api/session",
	JOIN: "/api/join",
	WEBSOCKET: "/ws",
	LOBBY: "/lobby",
}

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
 * HTMX direct call
 */

function hxGetUserID() {
	return window.localStorage.getItem("userID")
}

/**
 * API calls
 */

async function fetchUserID() {
	if (window.localStorage.getItem("userID") == null) {
		const uid = await fetch(API_PATH.NEW_USER).then(res => { return res.text() }).catch(err => { console.error(err) })
		window.localStorage.setItem("userID", uid)
	}
}

async function fetchRoomID() {
	const path = API_PATH.CREATE + "?sid=" + session.sessionID
	const rid = await fetch(path).then(res => { return res.text() }).catch(err => { console.error(err) })
	session.roomID = rid
}

async function fetchSessionID(form) {
	formData = new FormData(form)
	formData.append("user_id", window.localStorage.getItem("userID"))

	const sid = await fetch(API_PATH.SESSION, {
		method: "POST",
		body: formData
	}).then((res) => {
		return res.text()
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
		return res.text()
	})
	session.sessionID = sid
}

async function connectWebSocket() {
	const origin = document.location.href
	const domain = origin.split('://').pop().split("/").shift()
	const wsPath = "ws://" + domain + API_PATH.WEBSOCKET + "?sid=" + session.sessionID
	await connectWS(wsPath).then(() => {
		console.log("ws connected")
	}).catch((err) => {
		console.error("ws err:" + err)
	})
}

async function requestNewRoom(event, form) {
	// submit user profile and fetch session ID
	event.preventDefault()
	const cfgUsername = document.forms["user_profile"]["cfg_username"].value.trim()
	if (cfgUsername != null && cfgUsername.length) {
		session.username = cfgUsername
	}

	// fetch user ID
	await fetchUserID()

	// fetch session ID
	await fetchSessionID(form)

	// fetch room ID
	await fetchRoomID()

	// connect to the websocket
	await connectWebSocket()

	// render the lobby page
	const lobbyPath = API_PATH.LOBBY + "?sid=" + session.sessionID
	await htmx.ajax("GET", lobbyPath, { target: "#div_swap", })
	history.pushState({}, "", API_PATH.LOBBY)
	swapUsername()
}

async function requestJoinRoom(event, form) {
	// submit user profile and fetch session ID
	event.preventDefault()
	const cfgUsername = document.forms["user_profile"]["cfg_username"].value.trim()
	if (cfgUsername != null && cfgUsername.length) {
		session.username = cfgUsername
	}

	// get room ID
	const queryString = window.location.search;
	const urlParams = new URLSearchParams(queryString);
	session.roomID = urlParams.get("rid")
	console.log(session.roomID)

	// fetch user ID
	await fetchUserID()

	// fetch session ID
	await fetchSessionIDJoin(form)

	// connect to the websocket
	await connectWebSocket()

	// render the lobby page
	const lobbyPath = API_PATH.LOBBY + "?sid=" + session.sessionID
	await htmx.ajax("GET", lobbyPath, { target: "#div_swap", })
	history.pushState({}, "", API_PATH.LOBBY)
	swapUsername()
}


/**
 * UI/UX related functions
 */

function generateInviteLink(code) {
	const origin = document.location.href
	const proto = origin.split('://').shift()
	const domain = origin.split('://').pop().split("/").shift()
	const endpoint = API_PATH.JOIN + "/" + code

	return proto + "://" + domain + endpoint
}

function swapInviteLink() {
	const inner = document.querySelector("#current_room_id").innerHTML
	const head = inner.split(":").shift()
	const code = inner.split(": ").pop()
	if (code != "n/a") {
		document.querySelector("#current_room_id").innerHTML = head + ": " + generateInviteLink(code)
		window.localStorage.setItem("roomID", code)
	}
}

function swapUsername() {
	const inner = document.querySelector("#username").innerHTML
	const prefix = inner.split(": ").shift()
	const username = session.username
	document.querySelector("#username").innerHTML = prefix + ": " + username
}

function updateRoomStatus(msg) {
	const payload = msg.Data
	switch (payload) {
		case "join": {
			const inner = document.querySelector("#room_capacity").innerHTML
			const prefix = inner.split(": ").shift()
			const capacity = parseInt(inner.split(": ").pop()) + 1
			document.querySelector("#room_capacity").innerHTML = prefix + ": " + capacity
			console.log(prefix + capacity)
			// update the userlist
			break
		}
		case "left": {
			const inner = document.querySelector("#room_capacity").innerHTML
			const prefix = inner.split(": ").shift()
			const capacity = parseInt(inner.split(": ").pop()) - 1
			document.querySelector("#room_capacity").innerHTML = prefix + ": " + capacity
			console.log(prefix + capacity)
			break
		}
		case "host": {
			const inner = document.querySelector("#room_host").innerHTML
			const prefix = inner.split(": ").shift()
			const host = msg.Username
			document.querySelector("#room_host").innerHTML = prefix + ": " + host
			console.log(prefix + host)
			break
		}
		default:
			break
	}
}
