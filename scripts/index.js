// global
const session = {
	sessionID: "",
	roomID: "",
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
		const uuid = await fetch(API_PATH.NEW_USER).then(res => { return res.text() })
		window.localStorage.setItem("userID", uuid)
	}
}

async function requestNewRoom(event, form) {
	// submit user profile and fetch session ID
	event.preventDefault()

	// fetch user ID
	if (window.localStorage.getItem("userID") == null) {
		const uid = await fetch(API_PATH.NEW_USER).then(res => { return res.text() }).catch(err => { console.error(err) })
		window.localStorage.setItem("userID", uid)
	}

	// fetch room ID
	const rid = await fetch(API_PATH.CREATE).then(res => { return res.text() }).catch(err => { console.error(err) })
	session.roomID = rid

	// fetch session ID
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

	// connect to the websocket
	const origin = document.location.href
	const domain = origin.split('://').pop().split("/").shift()
	const wsPath = "ws://" + domain + API_PATH.WEBSOCKET + "?sid=" + session.sessionID
	await connectWS(wsPath).then(() => {
		console.log("ws connected")
	}).catch((err) => {
		console.error("ws err:" + err)
	})

	// render the lobby page
	const lobbyPath = API_PATH.LOBBY + "?rid=" + session.roomID
	await htmx.ajax("GET", lobbyPath, { target: "#div_swap", })
	history.pushState({}, "", API_PATH.LOBBY)
}

async function requestJoinRoom(event, form) {
	// submit user profile and fetch session ID
	event.preventDefault()

	// fetch user ID
	if (window.localStorage.getItem("userID") == null) {
		const uid = await fetch(API_PATH.NEW_USER).then(res => { return res.text() }).catch(err => { console.error(err) })
		window.localStorage.setItem("userID", uid)
	}

	// get room ID
	const queryString = window.location.search
	const queryParam = new URLSearchParams(queryString)
	const rid = queryParam.get("rid")
	session.roomID = rid

	// fetch session ID
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

	// connect to the websocket
	const origin = document.location.href
	const domain = origin.split('://').pop().split("/").shift()
	const wsPath = "ws://" + domain + API_PATH.WEBSOCKET + "?sid=" + session.sessionID
	await connectWS(wsPath).then(() => {
		console.log("ws connected")
	}).catch((err) => {
		console.error("ws err:" + err)
	})

	// render the lobby page
	const lobbyPath = API_PATH.LOBBY + "?rid=" + session.roomID
	await htmx.ajax("GET", lobbyPath, { target: "#div_swap", })
	history.pushState({}, "", API_PATH.LOBBY)
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
			break
		}
		default:
			break
	}
}
