/**
 * create a websocket on join/create a room
 */

const client = {}

function connectWS(endpoint) {
	client.ws = new WebSocket(endpoint)
	// debug
	return new Promise((resolve, reject) => {
		client.ws.onopen = (event) => {
			console.log("ws open: " + event.data)
			resolve()
		}
		client.ws.onerror = (event) => {
			console.log("ws err: " + event.data)
			reject()
		}
		client.ws.onclose = (event) => {
			console.log("ws close: " + event.data)
		}
		client.ws.onmessage = (event) => {
			console.log("ws recv: " + event.data)
			const msg = JSON.parse(event.data)
			console.log(msg.MsgType)
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

