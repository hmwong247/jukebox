// global
const session = {
	sessionID: "",
	roomID: "",
	username: "user",
	userList: {},
	audioArrBuf: {},
	audioContext: new AudioContext(),
}

const API_PATH = {
	NEW_USER: "/api/new-user",
	SESSION: "/api/session",
	CREATE: "/api/create",
	USERS: "/api/users",
	ENQUEUE: "/api/enqueue",
	JOIN: "/join",
	WEBSOCKET: "/ws",
	LOBBY: "/lobby",
	HOME: "/home",
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

/**
 * web audio
 */
async function playAudio() {
	// Create a source node from the buffer
	var source = session.audioContext.createBufferSource()
	source.buffer = await session.audioContext.decodeAudioData(session.audioArrBuf)

	// Connect to the final output node (the speakers)
	source.connect(session.audioContext.destination)

	source.start(0)
}
