/**
 * WebRTC operation with simple-peer, https://github.com/feross/simple-peer
 *
 * the new joiner will be passive for an easier life
 */

var localConn

/** @type {SimplePeer[]} */
var peers = {}

const PEER_CMD = Object.freeze({
	INIT: "init",
	PLAY: "play",
	PAUSE: "pause",
	SKIP: "skip",
})

/**
 * callback when HTMLMediaElement.oncanplay fired
 * the buffer is ensured that the audio is ready
 *
 * @param {!MediaStream} localStream
 * @param {!MediaStreamTrack} captured - new captured MediaStreamTrack
 */
function startSyncPeer(localStream, captured) {
	//for (uuid in peers) {
	//	const msg = { from: session.userID, payload: PEER_CMD.INIT }
	//	peers[uuid].send(JSON.stringify(msg))
	//}

	//if (mp.elem) {
	console.log(`start sync`)
	for (id in peers) {
		peers[id].addTrack(captured, localStream)
		console.log(`added track`)
	}
	//}
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
		config.stream = mp.localStream
	}
	console.log(config)
	conn = new SimplePeer(config)

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
		client.ws.send(JSON.stringify(dm))
	})

	conn.on('connect', () => {
		console.log(`addPeer connect, at ${session.userID}`)
	})

	conn.on('data', data => onpeerdata(data))
	conn.on('stream', stream => onpeerstream(stream))
	conn.on('track', (track, stream) => {
		console.log("addpeer ontrack: " + track, stream)
	})

	peers[msg.UID] = conn
}

// signal(answer) back all the peers
function answerPeer(msg) {
	const from = msg.UID
	if (!peers[from]) {
		conn = new SimplePeer({ initiator: false, trickle: false })

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
			client.ws.send(JSON.stringify(dm))
		})

		conn.on('connect', () => {
			console.log(`answerPeer connect, at ${session.userID}`)
		})

		// datachannel and stream
		conn.on('data', data => onpeerdata(data))
		conn.on('stream', stream => onpeerstream(stream))
		conn.on('track', (track, stream) => {
			console.log("answerpeer ontrack: " + track, stream)
		})

		peers[from] = conn
		console.log(conn)
	}

	peers[from].signal(JSON.parse(msg.Data).Data)
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

/** @param {MediaStream} stream */
function onpeerstream(stream) {
	console.log(`onpeerstream` + stream)
	mp.elem = document.querySelector("#player")
	if ("srcObject" in mp.elem) {
		mp.elem.srcObject = stream
	} else {
		mp.elem.src = URL.createObjectURL(stream)
		// revoke the objecturl
	}
	mp.elem.play()
}
