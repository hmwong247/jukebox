/**
 * create a websocket on join/create a room
 */

const client = {}

function connectWS(endpoint) {
	client.ws = new WebSocket(endpoint)
	// debug
	client.ws.onopen = (event) => {
		console.log("ws open: " + event.data)
	}
	client.ws.onclose = (event) => {
		console.log("ws close: " + event.data)
	}
	client.ws.onerror = (event) => {
		console.log("ws err: " + event.data)
	}
	client.ws.onmessage = (event) => {
		console.log("ws recv: " + event.data)
	}
}

