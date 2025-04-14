// external library
import htmx from "htmx.org"

// internal library

// global state
const session = $state({
	/** @type {WebSocket?} ws */
	ws: null,
	sessionID: "",
	roomID: "",
	username: "user",
	/** @type {Object.<string, {name: string, host: boolean}>} userList */
	userList: {},
	playlist: [],
	userID: "",
	hostID: "",
});

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

const MSG_TYPE = Object.freeze({
	EVENT_DEBUG: 0,
	EVENT_ROOM: 1,
	EVENT_PEER: 2,
	EVENT_PLAYLIST: 3,
	EVENT_PLAYER: 4,
})


/*
  init
*/
addEventListener("DOMContentLoaded", async () => {
	await onDOMContentLoaded()
})

// this is detached for later use after bootstrap
async function onDOMContentLoaded() {
	//htmx.logAll()
}

/*
  API calls
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
	session.userID = window.localStorage.getItem("userID")
}

/** 
 * @param {HTMLFormElement} form
 */
async function fetchSessionID(form) {
	const formData = new FormData(form)
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

/** 
 * @param {HTMLFormElement} form
 */
async function fetchSessionIDJoin(form) {
	const formData = new FormData(form)
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

function fetchUserList() {
	// store the user list in session, we have to
	return new Promise((resolve, reject) => {
		const path = API_PATH.USERS + "?sid=" + session.sessionID
		fetch(path)
			.then(res => { resolve(res.json()) })
			.catch(err => { reject(err) })
	})
}

/**
 * @param {MouseEvent} event
 * @param {HTMLFormElement} form
 */
async function requestNewRoom(event, form) {
	// submit user profile and fetch session ID
	event.preventDefault()
	const cfgUsername = document.forms["user_profile"]["cfg_username"].value.trim()
	if (cfgUsername != null && cfgUsername.length) {
		session.username = cfgUsername
	} else {
		document.forms["user_profile"]["cfg_username"].value = session.username
	}

	try {
		// fetch user ID
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
	// const lobbyPath = API_PATH.LOBBY + "?sid=" + session.sessionID
	// await htmx.ajax("get", lobbyPath, { target: "#div_swap", })
	// history.pushState({}, "", API_PATH.LOBBY)
	// swapUsername(session.username)
	// swapInviteLink(document.querySelector("#room_id").innerHTML)

	await fetchUserList().then(data => {
		for(const id in data) {
			session.userList[id] = data[id]
		}
	})
	session.hostID = session.userID
}

/**
 * @param {MouseEvent} event
 * @param {HTMLFormElement} form
 */
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
	// const lobbyPath = API_PATH.LOBBY + "?sid=" + session.sessionID
	// await htmx.ajax("get", lobbyPath, { target: "#div_swap", }).catch(err => { console.error(err); return })
	// history.pushState({}, "", API_PATH.LOBBY)
	// swapUsername(session.username)
	// swapInviteLink(document.querySelector("#room_id").innerHTML)

	await fetchUserList().then(data => {
		for(const id in data) {
			session.userList[id] = data[id]
			if (data[id].host === true) {
				session.hostID = id
			}
		}
	})
}

/*
	WebSocket
*/

function connectWebSocket() {
	const wsPath = "ws://" + document.location.host + API_PATH.WEBSOCKET + "?sid=" + session.sessionID
	return new Promise((resolve, reject) => {
		session.ws = new WebSocket(wsPath)

		session.ws.onopen = (event) => {
			console.log("ws open: " + JSON.stringify(event))
			resolve()
		}
		session.ws.onerror = (event) => {
			console.log("ws err: " + JSON.stringify(event))
			reject()
		}
		session.ws.onclose = (event) => {
			console.log("ws close: " + JSON.stringify(event))
			reject()
		}

		session.ws.onmessage = (event) => {
			//console.log("ws recv: " + event.data)
			const msg = JSON.parse(event.data)
			switch (msg.MsgType) {
				case MSG_TYPE.EVENT_ROOM:
					updateRoomStatus(msg)
					updatePeerConnection(msg)
					break
				case MSG_TYPE.EVENT_PEER:
					answerPeer(msg)
					break
				case MSG_TYPE.EVENT_PLAYLIST:
					updatePlaylist(msg)
					break
				case MSG_TYPE.EVENT_PLAYER:
					updateMP(msg)
					break
				default:
					break
			}
		}

	})
}

function updateRoomStatus(msg) {
	const payload = msg.Data
	switch (payload) {
		case "join": {
			session.userList[msg.UID] = { name: msg.Username, host: false }
			// swapRoomCapacity(Object.keys(session.userList).length)
			// swapUserList(msg.UID, msg.Username)
			break
		}
		case "left": {
			delete session.userList[msg.UID]
			// swapRoomCapacity(Object.keys(session.userList).length)
			// swapUserList(msg.UID, msg.Username, false)
			break
		}
		case "host": {
			session.hostID = msg.UID
			session.userList[msg.UID].host = true
			// swapHost(msg.Username)
			break
		}
		default:
			console.warn(`[updateRoomStatus] got unknown payload: ${payload}`)
			break
	}
}

function updatePlaylist(msg) {
	const cmd = msg.Data.Cmd
	switch (cmd) {
		case "add":
			delete msg.Data['Cmd']
			session.playlist.push(msg.Data)
			swapPlaylist(msg.Data)
			break
		case "remove":
			break
		case "swap":
			break
		default:
			break
	}
}

function updateMP(msg) {
	const data = msg.Data
	if (data.OK == true) {
		playAudio()
	}
}

function updatePeerConnection(msg) {
	const evt = msg.Data
	if (evt === "join") {
		addPeer(msg)
	} else if (evt === "left") {
		removePeer(msg)
	}
}

/*
	UI/UX related functions
*/

// redirect to home page
function autoResetPage() {
	htmx.ajax("get", API_PATH.HOME, { target: "#div_swap" }).catch(err => { console.error(err); return })
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

/*
	web audio
*/

const mp = {
	/** @type {HTMLAudioElement | HTMLMediaElement} elem */
	elem: null,
	/** @type {HTMLAudioElement} mozElem */
	mozElem: null,
	/** @type {Boolean} running */
	running: false,
	/** @type {MediaStream} localStream - captured locally */
	localStream: null,
	/** @type {MediaStream} remoteStream - streamed from peer */
	remoteStream: null,
}

function initMP() {
	mp.elem = document.querySelector("#player")
	mp.elem.removeEventListener('loadstart', mploadstart)
	mp.elem.removeEventListener('timeupdate', mptimeupdate)
	mp.elem.removeEventListener('ended', mpended)

	mp.elem.src = API_PATH.STREAM + "?sid=" + session.sessionID
	mp.elem.addEventListener('loadstart', mploadstart)
}

/** 
 * init the MediaStream for the host
 */
function lazyInitMPStream() {
	if (!mp.elem) {
		mp.elem = document.querySelector("#player")
		if (!mp.localStream) {
			mp.localStream = new MediaStream()
		}
	}
}

async function mpcanplay() {
	lazyInitMPStream()
	let stream
	if ("mozCaptureStream" in mp.elem) {
		stream = mp.elem.mozCaptureStream()
		if (!mp.mozElem) {
			mp.mozElem = new Audio()
		}
	} else {
		stream = mp.elem.captureStream()
	}
	const track = stream.getTracks()[0]
	startSyncPeer(mp.localStream, track)
	mp.localStream.addTrack(track)
}

async function mploadstart() {
	console.log(`loadstart`)
	mp.elem.addEventListener('canplay', mpcanplay, { once: true })
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

async function peermpended() {
	console.log(`ended`)
	mp.elem.pause()
	mp.elem.currentTime = 0
	const endedJson = session.playlist.shift()
	swapPlaylist(endedJson, true)

	if (session.playlist.length > 0) {
		playAudioAsPeer()
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
	// host specific event listener
	mp.elem.addEventListener('loadstart', mploadstart) // host will init the play
	mp.elem.addEventListener('timeupdate', mptimeupdate)
	mp.elem.addEventListener('ended', mpended, { once: true })
}

async function playAudioAsPeer() {
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
	mp.elem.addEventListener('ended', peermpended, { once: true })
}

export const api = {
	requestNewRoom,
	requestJoinRoom,
}

// const
export { API_PATH }

// global state
export { session }
