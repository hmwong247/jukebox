/**
 * create a websocket on join/create a room
 */

const client = {}

function connectWS(endpoint) {
	client.ws = new WebSocket(endpoint)
	// debug
	return new Promise((resolve, reject) => {
		client.ws.onopen = (event) => {
			console.log("ws open: " + event)
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
				case 1: // broadcast
					updateRoomStatus(msg)
					break
				default:
					break
			}
		}
	})
}

function updateRoomStatus(msg) {
	const payload = msg.Event
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
