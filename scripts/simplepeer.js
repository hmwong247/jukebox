/**
 * WebRTC operation with simple-peer, https://github.com/feross/simple-peer
 *
 * the new joiner will be passive for an easier life
 */

var localConn
var peers = {}

const PEER_CMD = Object.freeze({
	INIT: "init",
	PLAY: "play",
	PAUSE: "pause",
	SKIP: "skip",
})

function removePeer(msg) {
	if (peers[msg.UID]) {
		peers[msg.UID].destroy();
		delete peers[msg.UID];
	}
}

function addPeer(msg) {
	if (msg.UID === session.userID) { return }

	conn = new SimplePeer({ initiator: true, trickle: false })

	conn.on('error', err => {
		console.log(`addPeer error, at ${session.userID}: ${err}`)

		if (peers[msg.UID]) { delete peers[msg.UID] }
	})

	conn.on('signal', data => {
		//console.log(`addPeer signal, at ${session.userID}: ${JSON.stringify(data)}`)
		const dm = {
			To: msg.UID,
			Data: data,
		}
		client.ws.send(JSON.stringify(dm))
	})

	conn.on('connect', () => {
		console.log(`addPeer connect, at ${session.userID}`)
	})

	conn.on('data', data => onpeerdata(data))

	peers[msg.UID] = conn
}

function answerPeer(msg) {
	const from = msg.UID
	const to = msg.Data.To
	if (!peers[from]) {
		conn = new SimplePeer({ initiator: false, trickle: false })

		conn.on('error', err => {
			console.log(`answerPeer error, at ${session.userID}: ${err}`)

			if (peers[from]) { delete peers[from] }
		})

		conn.on('signal', data => {
			//console.log(`answerPeer signal, at ${session.userID}: ${JSON.stringify(data)}`)

			const dm = {
				To: from,
				Data: data,
			}
			client.ws.send(JSON.stringify(dm))
		})

		conn.on('connect', () => {
			console.log(`answerPeer connect, at ${session.userID}`)
		})

		conn.on('data', data => onpeerdata(data))

		peers[from] = conn
		console.log(conn)
	}

	peers[from].signal(JSON.parse(msg.Data).Data)
}

function startSyncPeer() {
	for (uuid in peers) {
		const msg = { from: session.userID, payload: PEER_CMD.INIT }
		peers[uuid].send(JSON.stringify(msg))
	}
}

/** @param {{to: string, payload: string}} msg */
function onpeerdata(msg) {
	console.log(`onpeerdata, at ${session.userID}: ` + msg)
	const data = JSON.parse(msg.toString())
	const from = data.from
	if (from === session.hostID) {
		const payload = data.payload
		console.log(`payload: ${payload}`)
		switch (payload) {
			case PEER_CMD.INIT:
				playAudioAsPeer()
				break
			case PEER_CMD.PLAY:
				break
			case PEER_CMD.PAUSE:
				break
			case PEER_CMD.STOP:
				break
		}
	}
}
