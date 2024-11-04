var uuid

// fetch user session
async function getUserSession() {
	if (window.sessionStorage.getItem("userID") == null) {
		uuid = await fetch("/new-user").then(res => { return res.text() })
		window.sessionStorage.setItem("userID", uuid)
	} else {
		uuid = window.sessionStorage.getItem("userID")
	}
	//const current_room = document.querySelector("#current_room").innerHTML
}

async function getCurrentRoom() {
	const apiPath = "/join/" + uuid
	htmx.ajax("GET", apiPath, "#current_room")
}

// init
addEventListener("DOMContentLoaded", async () => {
	await getUserSession()
	getCurrentRoom()
})

// TEST
//(async () => {
//	const uuid = await fetch("/new-user").then(res => { return res.text() })
//	console.log(uuid)
//	window.sessionStorage.setItem("userID", uuid)
//})()
