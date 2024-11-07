/**
 * bootstrap new users who are coming from an invite link
 */


// init
addEventListener("DOMContentLoaded", async () => {
	// fetch user session
	if (window.sessionStorage.getItem("userID") == null) {
		const uuid = await fetch("/new-user").then(res => { return res.text() })
		window.sessionStorage.setItem("userID", uuid)
	}

	const origin = document.location.href
	const roomID = origin.split('://').pop().split("/").pop()
	window.sessionStorage.setItem("roomID", roomID)

	// swap document and append js
	await htmx.ajax("GET", "/home", "body")
	let indexScript = document.createElement("script")
	indexScript.src = "/scripts/index.js"
	indexScript.type = "text/javascript"
	indexScript.async = true
	indexScript.onload = async () => {
		await onDOMContentLoaded()
	}
	indexScript.onerror = () => {
		reject(new Error("Failed to load script"))
	}
	document.head.appendChild(indexScript)
})


