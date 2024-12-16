// global
//const session = {
//	userUUID: "", // base64 encoded
//	roomUUID: "", // base64 encoded
//}

const API_PATH = {
	NEW_USER: "/api/new-user",
	JOIN: "/api/join",
	LOBBY: "/lobby",
}

// fetch user session
async function fetchUserID() {
	if (window.localStorage.getItem("userID") == null) {
		const uuid = await fetch(API_PATH.NEW_USER).then(res => { return res.text() })
		window.localStorage.setItem("userID", uuid)
	}
}

//async function getCurrentRoom() {
//	const roomID = window.localStorage.getItem("roomID")
//	if (roomID != undefined && roomID != null) {
//		const path = API_PATH.JOIN + "/" + roomID
//		await htmx.ajax("GET", path, { target: "#current_room", values: { join_userid: hxGetUserID() } })
//	}
//}

/**
 * init
 */
addEventListener("DOMContentLoaded", async () => {
	await onDOMContentLoaded()
})

// this is detached for later use after bootstrap
async function onDOMContentLoaded() {
	//htmx.logAll()

	await fetchUserID()
}

/**
 * HTMX direct call
 */

function hxGetUserID() {
	return window.localStorage.getItem("userID")
}

function hxUpgradeWS() {
	connectWS("ws://homearch:8080/ws")
}

/**
 * helper function
 */

async function submitUserProfile(event, form) {
	event.preventDefault()
	formData = new FormData(form)
	formData.append("user_id", window.localStorage.getItem("userID"))

	const roomID = await fetch(form.action, {
		method: form.method,
		body: formData
	}).then((res) => {
		return res.text()
	})

	console.log(roomID)

	await connectWS("ws://homearch:8080/ws").then(() => {
		console.log("ws connected")
	}).catch(() => {
		console.log("ws err")
	})

	const path = API_PATH.LOBBY + "?rid=" + roomID
	await htmx.ajax("GET", path, { target: "#div_swap", })
	history.pushState({}, "", API_PATH.LOBBY)

	console.log("end")
}

function generateInviteLink(code) {
	const origin = document.location.href
	const proto = origin.split('://').shift()
	const domain = origin.split('://').pop().split("/").shift()
	const endpoint = API_PATH.JOIN + "/" + code

	return proto + "://" + domain + endpoint
}

function swapInviteLink() {
	const inner = document.querySelector("#current_room_id").innerHTML
	const head = inner.split(":").shift()
	const code = inner.split(": ").pop()
	if (code != "n/a") {
		document.querySelector("#current_room_id").innerHTML = head + ": " + generateInviteLink(code)
		window.localStorage.setItem("roomID", code)
	}
}
