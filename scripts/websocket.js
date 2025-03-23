/**
 * create a websocket on join/create a room
 */

const client = {}

const MSG_TYPE = Object.freeze({
	EVENT_DEBUG: 0,
	EVENT_ROOM: 1,
	EVENT_PEER: 2,
	EVENT_PLAYLIST: 3,
	EVENT_PLAYER: 4,
})

function connectWS(endpoint) {
	client.ws = new WebSocket(endpoint)
	// debug
	return new Promise((resolve, reject) => {
		client.ws.onopen = (event) => {
			console.log("ws open: " + event)
			newPeerInterface()
			resolve()
		}
		client.ws.onerror = (event) => {
			console.log("ws err: " + event)
			reject()
		}
		client.ws.onclose = (event) => {
			console.log("ws close: " + event)
			autoResetPage()
		}

		client.ws.onmessage = (event) => {
			console.log("ws recv: " + event.data)
			const msg = JSON.parse(event.data)
			switch (msg.MsgType) {
				case MSG_TYPE.EVENT_ROOM:
					updateRoomStatus(msg)
					break
				case MSG_TYPE.EVENT_PEER:
					updatePeerConnection(msg)
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
			session.userList[msg.UID] = msg.Username

			swapRoomCapacity(Object.keys(session.userList).length)
			swapUserList(msg.UID, msg.Username)
			break
		}
		case "left": {
			delete session.userList[msg.UID]

			swapRoomCapacity(Object.keys(session.userList).length)
			swapUserList(msg.UID, msg.Username, false)
			break
		}
		case "host": {
			swapHost(msg.Username)
			break
		}
		default:
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
	const pid = JSON.parse(msg.Data).pid
	if (pid != null) {
		addPeer(pid)
	}
}
