/**
 * WebRTC operation with simple-peer, https://github.com/feross/simple-peer
 *
 * the new joiner will be passive for an easier life
 */

var localConn
var peers = {}



function addPeer(msg) {
	if (msg.UID === session.userID) { return }

	conn = new SimplePeer({ initiator: true, trickle: false })

	conn.on('error', err => {
		console.log(`addPeer error, at ${session.userID}: ${err}`)
	})

	conn.on('signal', data => {
		console.log(`addPeer signal, at ${session.userID}: ${JSON.stringify(data)}`)
		//client.ws.send(JSON.stringify(data))
		const dm = {
			To: msg.UID,
			Data: data,
		}
		client.ws.send(JSON.stringify(dm))
	})

	conn.on('connect', () => {
		console.log(`addPeer connect, at ${session.userID}`)
	})

	conn.on('data', data => {
		console.log(`addPeer data, at ${session.userID}: ${data}`)
	})

	peers[msg.UID] = conn
}

function removePeer(msg) {
	if (peers[msg.UID]) {
		peers[msg.UID].destroy();
		delete peers[msg.UID];
	}
}

function answerPeer(msg) {
	const from = msg.UID
	const to = msg.Data.To
	if (peers[from]) {
		console.log(`peer already connected`)
		return
	}
	//if (to !== session.userID) {
	//	console.log(`peer target is not self`)
	//	return
	//}

	conn = new SimplePeer({ initiator: false, trickle: false })

	conn.on('error', err => {
		console.log(`answerPeer error, at ${session.userID}: ${err}`)
	})

	conn.on('signal', data => {
		console.log(`answerPeer signal, at ${session.userID}: ${JSON.stringify(data)}`)

		const dm = {
			To: from,
			Data: data,
		}
		client.ws.send(JSON.stringify(dm))
	})

	conn.on('connect', () => {
		console.log(`answerPeer connect, at ${session.userID}`)
	})

	conn.on('data', data => {
		console.log(`answerPeerdata, at ${session.userID}: ${data}`)
	})

	conn.signal(JSON.parse(msg.Data).Data)
	peers[msg.UID] = conn
	console.log(conn)
}
