// global
//const session = {
//	userUUID: "", // base64 encoded
//	roomUUID: "", // base64 encoded
//}

// fetch user session
async function getUserID() {
	if (window.sessionStorage.getItem("userID") == null) {
		const uuid = await fetch("/new-user").then(res => { return res.text() })
		window.sessionStorage.setItem("userID", uuid)
	}
}

async function getCurrentRoom() {
	const roomID = window.sessionStorage.getItem("roomID")
	if (roomID != undefined && roomID != null) {
		const apiPath = "/join/" + roomID
		await htmx.ajax("GET", apiPath, { target: "#current_room", values: { join_userid: hxGetUserID() } })
	}
}

// init
addEventListener("DOMContentLoaded", async () => {
	await onDOMContentLoaded()
})

async function onDOMContentLoaded() {
	//htmx.logAll()
	await getUserID()
	await getCurrentRoom()
}

/*
 * HTMX direct call
 */

function hxGetUserID() {
	return window.sessionStorage.getItem("userID")
}

/*
 * helper function
 */

function generateInviteLink(code) {
	const origin = document.location.href
	const proto = origin.split('://').shift()
	const domain = origin.split('://').pop().split("/").shift()
	const endpoint = "/join/" + code

	return proto + "://" + domain + endpoint
}

function swapInviteLink() {
	const inner = document.querySelector("#current_room_id").innerHTML
	const head = inner.split(":").shift()
	const code = inner.split(": ").pop()
	if (code != "n/a") {
		document.querySelector("#current_room_id").innerHTML = head + ": " + generateInviteLink(code)
		window.sessionStorage.setItem("roomID", code)
	}
}
