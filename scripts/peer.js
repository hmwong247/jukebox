const PEER_CONFIG = Object.freeze({
	host: "/",
	port: "9000",
	pingInterval: 1000 * 30,
	debug: 3,
})

/** @type {Peer} localCon */
var conn
var peers = []

function newPeerInterface() {
	const uid = window.localStorage.getItem("userID")
	conn = new Peer(uid, PEER_CONFIG)

	conn.on('error', err => {
		console.warn(`localConn error: ${err}`)
	})

	conn.on('open', () => {
		const uid = window.localStorage.getItem("userID")
		const payload = {
			pid: uid,
		}
		client.ws.send(JSON.stringify(payload))
	})

	conn.on('connection', peerConn => {
		peerConn.on('error', err => {
			console.warn(`peerConn error: ${err}`)
		})

		peerConn.on('open', () => {
			peers.push(peerConn)
			peerConn.send(`hi from ${conn.id}`)
		})

		peerConn.on('data', (data) => {
			console.log(`peerConn recieved: ${data}`)
		})

		peerConn.on('close', () => {
			console.log(`peerConn close: ${peerConn.connectionId}`)
			peerConn.destroy()
		})
	})
}

function addPeer(id) {
	if (id === conn.id) {
		console.log(`same id: ${id}`)
		return
	}
	const peerConn = conn.connect(id)

	peerConn.on('error', err => {
		console.warn(`peerConn error: ${err}`)
	})

	peerConn.on('open', () => {
		peers.push(peerConn)
		peerConn.send(`hi from ${conn.id}`)
	})

	peerConn.on('data', (data) => {
		console.log(`peerConn recieved: ${data}`)
	})

	peerConn.on('close', () => {
		console.log(`peerConn close: ${peerConn.connectionId}`)
		peerConn.destroy()
	})
}


