<script>
	import { session, API_PATH } from "../../scripts/index.svelte.js";

/**
 * @param {SubmitEvent} event 
 */
async function submitURL(event) {
	event.preventDefault()
    const form = /** @type {HTMLFormElement} */ (event.currentTarget)
	const formData = new FormData(form) 
	formData.append("user_id", window.localStorage.getItem("userID"))

	await fetch(API_PATH.ENQUEUE + "?sid=" + session.sessionID, {
		method: "POST",
		body: formData
	}).then((res) => {
		if (res.ok) {
			console.log(res.status)
		} else {
			console.log(res.status)
		}
	}).catch(err => {
		throw err
	})
}

</script>

<div>
    <form name="user_queue" onsubmit={submitURL}>
        <fieldset>
            <legend>Queue</legend>
            <div>
                <input name="post_url" id="post_url" placeholder="submit an URL..." />
                <button type="submit" style="display:inline">add to queue</button>
            </div>
        </fieldset>
    </form>
    <ul id="room_queue_list">

    </ul>
</div>

<style>
</style>
