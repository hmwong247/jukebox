// external library
import SimplePeer from "simple-peer";

/**
 * @typedef {{ID: string, FullTitle: string, Uploader: string, Thumbnail: string, Duration: string}} InfoJson
 */

// global state
const session = $state({
	sessionID: null,
	roomID: null,
	username: "user",
	/** @type {Object.<string, {name: string, host: boolean}>} userList */
	userList: {},
	/** @type {Array.<InfoJson>} */
	playlist: [],
	userID: null,
	hostID: null,
});

// const
const API_PATH = Object.freeze({
	// api
	NEW_USER: "/api/new-user",
	SESSION: "/api/session",
	CREATE: "/api/create",
	USERS: "/api/users",
	PLAYLIST: "/api/playlist",
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


/*==============================================================================
	init
*/
addEventListener("DOMContentLoaded", async () => {
	await onDOMContentLoaded()
})

// this is detached for later use after bootstrap
async function onDOMContentLoaded() {
}

/*==============================================================================
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

function fetchPlaylist() {
	return new Promise((resolve, reject) => {
		const path = API_PATH.PLAYLIST + "?sid=" + session.sessionID
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

	await fetchUserList().then(data => {
		for (const id in data) {
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

	await fetchPlaylist().then(data => {
		session.playlist = data
	})

	// userlist should be the last, we swap compoenet base on the user list
	await fetchUserList().then(data => {
		for (const id in data) {
			session.userList[id] = data[id]
			if (data[id].host === true) {
				session.hostID = id
			}
		}
	})
}

/*==============================================================================
	WebSocket
*/

const ws = $state({
	/** @type {WebSocket?} ws */
	client: null,
})

function connectWebSocket() {
	const wsPath = "ws://" + document.location.host + API_PATH.WEBSOCKET + "?sid=" + session.sessionID
	return new Promise((resolve, reject) => {
		ws.client = new WebSocket(wsPath)

		ws.client.onopen = (event) => {
			console.log("ws open: " + JSON.stringify(event))
			resolve()
		}
		ws.client.onerror = (event) => {
			console.log("ws err: " + JSON.stringify(event))
			reject()
		}
		ws.client.onclose = (event) => {
			console.log("ws close: " + JSON.stringify(event))
			reject()
		}

		ws.client.onmessage = (event) => {
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
			break
		}
		case "left": {
			delete session.userList[msg.UID]
			break
		}
		case "host": {
			session.hostID = msg.UID
			session.userList[msg.UID].host = true
			rtcRestart()
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
		loadAudioAsHost()
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

/*==============================================================================
	UI/UX related functions
*/

/*==============================================================================
	web audio
*/

const mp = $state({
	/** @type {HTMLAudioElement} elem - binded component */
	elem: null,
	/** @type {AudioContext} ctx */
	ctx: null,
	/** @type {MediaElementAudioSourceNode} srcNode */
	srcNode: null,
	/** @type {GainNode} gainNode */
	gainNode: null,
	/** @type {MediaStreamAudioDestinationNode} mediaStreamNode */
	mediaStreamNode: null,
	/** @type {Boolean} running */
	running: false,
	/** @type {MediaStream} hostStream - media stream for hosting */
	hostStream: null,
	/** @type {MediaStream} localStream - media stream from the audio element */
	localStream: null,
	/** @type {MediaStream} remoteStream - media stream from the host */
	remoteStream: null,
	/** @type {MediaStreamTrack} currentTrack */
	currentTrack: null,
})

/** 
 * host only
 * init the MediaStream for the host
 */
function lazyInitMPStream() {
	if (mp.hostStream === null) {
		console.log(`new host ctx`)
		mp.ctx = new AudioContext()
		mp.mediaStreamNode = mp.ctx.createMediaStreamDestination()
		mp.srcNode = mp.ctx.createMediaElementSource(mp.elem);
		mp.srcNode.connect(mp.mediaStreamNode)

		mp.gainNode = mp.ctx.createGain();
		mp.srcNode.connect(mp.gainNode);
		mp.gainNode.connect(mp.ctx.destination);

		mp.hostStream = new MediaStream()
		mp.localStream = mp.mediaStreamNode.stream
	}
}

async function loadAudioAsHost() {
	if (mp.running && mp.elem.buffered.length > 0 && mp.elem.currentTime != 0) {
		if (mp.elem.buffered.length > 0 && mp.elem.currentTime != 0 && !mp.elem.paused) return
		return
	}

	// implement a retry mechanism
	mp.elem.src = API_PATH.STREAM + "?sid=" + session.sessionID
}

async function loadAudioAsPeer(stream) {
	if ("srcObject" in mp.elem) {
		mp.elem.srcObject = stream
	} else {
		console.warn(`Using fallback ObjectURL`)
		mp.elem.src = URL.createObjectURL(stream)
	}

	if (mp.srcNode === null) {
		console.log("new peer ctx");
		mp.ctx = new AudioContext()
		mp.srcNode = mp.ctx.createMediaStreamSource(mp.elem.srcObject);
		mp.gainNode = mp.ctx.createGain();

		mp.srcNode.connect(mp.gainNode);
		mp.gainNode.connect(mp.ctx.destination);
	}

	// update and run mp when recieved PEER_CMD.INIT
	// mp.elem.play()
	// mp.running = true
}

/*==============================================================================
	  WebRTC operation with simple-peer, https://github.com/feross/simple-peer
	  the new joiner will be passive for an easier life
 */

/** @type {Object.<string, SimplePeer.Instance>} */
var peers = {}

const PEER_CMD = Object.freeze({
	INIT: "init",
	PLAY: "play",
	PAUSE: "pause",
	SKIP: "skip",
	NEXT: "next",
	STOP: "stop",
})

/**
 * @param {{from: string, payload: string}} msg
 */
function allPeers(msg) {
	for (const uuid in peers) {
		peers[uuid].send(JSON.stringify(msg));
	}
}

function toPeer(id, msg) {
	for (const uuid in peers) {
		if (uuid === id) {
			peers[uuid].send(JSON.stringify(msg));
		}
	}

}

/**
 * callback when HTMLMediaElement.oncanplay fired
 * the buffer is ensured that the audio is ready
 *
 * @param {!MediaStreamTrack} oldTrack - old captured MediaStreamTrack
 * @param {!MediaStreamTrack} newTrack - new captured MediaStreamTrack
 * @param {!MediaStream} hostStream
 */
function startSyncPeer(oldTrack, newTrack, hostStream) {
	console.log(`start sync`)
	for (const id in peers) {
		if (oldTrack === null) {
			peers[id].addTrack(newTrack, hostStream)
			console.log(`added track`)
		} else {
			peers[id].replaceTrack(oldTrack, newTrack, hostStream)
			console.log(`replaced track`)
		}
	}
}

function startSyncNewPeer(id, track, hostStream) {
	console.log(`sync new peer`)
	const msg = { from: session.userID, payload: PEER_CMD.INIT };
	toPeer(id, msg);
}

function removePeer(msg) {
	if (peers[msg.UID]) {
		peers[msg.UID].destroy();
		delete peers[msg.UID];
	}
}

function addPeer(msg) {
	if (msg.UID === session.userID) { return }

	let config = { initiator: true, trickle: false }
	if (session.userID === session.hostID) {
		lazyInitMPStream()
		config.stream = mp.hostStream
	}
	console.log(config)
	let conn = new SimplePeer(config)

	conn.on('error', err => {
		console.log(`addPeer error, at ${session.userID}: ${err}`)

		if (peers[msg.UID]) {
			peers[msg.UID].destroy()
			delete peers[msg.UID]
		}
	})

	conn.on('signal', data => {
		//console.log(`addPeer signal, at ${session.userID}: ${JSON.stringify(data)}`)
		const dm = {
			To: msg.UID,
			Data: data,
		}
		ws.client.send(JSON.stringify(dm))
	})

	conn.on('connect', () => {
		console.log(`addPeer connect, at ${session.userID}`)
		// sync late joiners
		if (session.userID === session.hostID && mp.running) {
			startSyncNewPeer(msg.UID, mp.currentTrack, mp.hostStream)
		}
	})

	conn.on('data', data => onpeerdata(data))
	conn.on('stream', stream => onpeerstream(stream))
	conn.on('track', (track, stream) => {
		console.log("addpeer ontrack: ", track, stream)
	})

	peers[msg.UID] = conn
}

// signal(answer) back all the peers
function answerPeer(msg) {
	const from = msg.UID
	if (!peers[from]) {
		const conn = new SimplePeer({ initiator: false, trickle: false })

		conn.on('error', err => {
			console.log(`answerPeer error, at ${session.userID}: ${err}`)

			if (peers[from]) {
				peers[from].destroy()
				delete peers[from]
			}
		})

		conn.on('signal', data => {
			//console.log(`answerPeer signal, at ${session.userID}: ${JSON.stringify(data)}`)

			const dm = {
				To: from,
				Data: data,
			}
			ws.client.send(JSON.stringify(dm))
		})

		conn.on('connect', () => {
			console.log(`answerPeer connect, at ${session.userID}`)
		})

		// datachannel and stream
		conn.on('data', data => onpeerdata(data))
		conn.on('stream', stream => onpeerstream(stream))
		conn.on('track', (track, stream) => {
			console.log("answerpeer ontrack: ", track, stream)
		})

		peers[from] = conn
		console.log(conn)
	}

	peers[from].signal(JSON.parse(msg.Data).Data)
}

/*
 * Simple-peer event listeners
 * /

/** @param {{to: string, payload: string}} msg */
function onpeerdata(msg) {
	console.log(`onpeerdata, at ${session.userID}: ` + msg)
	const data = JSON.parse(msg.toString())
	const from = data.from
	if (from === session.hostID) {
		const payload = data.payload
		// console.log(`payload: ${payload}`)
		switch (payload) {
			case PEER_CMD.INIT:
				mp.elem.play()
				mp.running = true
				break
			case PEER_CMD.PLAY:
				mp.elem.play()
				break
			case PEER_CMD.PAUSE:
				mp.elem.pause()
				break
			case PEER_CMD.NEXT:
				mp.elem.pause()
				mp.elem.currentTime = 0
				session.playlist.shift();
				break
			case PEER_CMD.STOP:
				mp.elem.pause()
				mp.elem.currentTime = 0
				session.playlist.shift();
				mp.running = false
				break
		}
	}
}

/** @param {MediaStream} stream */
function onpeerstream(stream) {
	console.log(`onpeerstream`, stream)

	loadAudioAsPeer(stream)
	mp.remoteStream = stream
}

/**
 * this will be run when the host left
 * @TODO: setup a protocol for reestablishing the peer mesh
 */
function rtcRestart() {
	console.log(`rtcRestart`)
	mp.elem.pause()
	mp.running = false;
	if ("srcObject" in mp.elem) {
		mp.elem.srcObject = null
		mp.elem.removeAttribute("srcObject")
	} else {
		URL.revokeObjectURL(mp.elem.src)
		mp.elem.src = null
		mp.elem.removeAttribute("src")
	}
}

/*==============================================================================
	exports module
*/

// const
export { API_PATH, PEER_CMD }

// global state
export { session, ws, mp }

// api
export const api = {
	requestNewRoom,
	requestJoinRoom,
}

// WebRTC
export const rtc = {
	startSyncPeer,
	lazyInitMPStream,
	loadAudioAsHost,
	loadAudioAsPeer,
	allPeers,
}
