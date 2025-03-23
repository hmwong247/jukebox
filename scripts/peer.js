const PEER_CONFIG = Object.freeze({
	host: "/",
	port: "9000",
	pingInterval: 1000 * 30,
	debug: 3,
})

/** @type {Peer} localCon */
var localConn
var peers = []

function newPeerInterface() {
	const uid = window.localStorage.getItem("userID")
	localConn = new Peer(uid, PEER_CONFIG)

	localConn.on('error', err => {
		console.warn(`localConn error: ${err}`)
	})

	localConn.on('open', () => {
		const uid = window.localStorage.getItem("userID")
		const payload = {
			pid: uid,
		}
		client.ws.send(JSON.stringify(payload))
	})

	localConn.on('connection', peerConn => {
		peerConn.on('error', err => {
			console.warn(`peerConn error: ${err}`)
		})

		peerConn.on('open', () => {
			peers.push(peerConn)
			peerConn.send(`hi from ${localConn.id}`)
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
	if (id === localConn.id) {
		console.log(`same id: ${id}`)
		return
	}
	const peerConn = localConn.connect(id)

	peerConn.on('error', err => {
		console.warn(`peerConn error: ${err}`)
	})

	peerConn.on('open', () => {
		peers.push(peerConn)
		peerConn.send(`hi from ${localConn.id}`)
	})

	peerConn.on('data', (data) => {
		console.log(`peerConn recieved: ${data}`)
	})

	peerConn.on('close', () => {
		console.log(`peerConn close: ${peerConn.connectionId}`)
		peerConn.destroy()
	})
}


