<script>
	import { session, API_PATH } from "@lib/index.svelte.js";
	import RowMusicCard from "@components/RowMusicCard.svelte"

	/**
	 * @param {SubmitEvent} event
	 */
	async function submitURL(event) {
		event.preventDefault();
		const form = /** @type {HTMLFormElement} */ (event.currentTarget);
		const formData = new FormData(form);
		formData.append("user_id", window.localStorage.getItem("userID"));

		try {
			await fetch(API_PATH.ENQUEUE + "?sid=" + session.sessionID, {
				method: "POST",
				body: formData,
			})
				.then((res) => {
					if (res.ok) {
						console.log("Success:", res.status);
						form.reset();
					} else {
						console.log("Error:", res.status);
					}
				})
				.catch((err) => {
					throw err;
				});
		} catch (err) {
			console.error("Fetch error:", err);
		}
	}
</script>

<div id="music_queue" class="flex flex-col border rounded-md border-emerald-400">
	<form class="p-2" onsubmit={submitURL}>
			<h2 class="text-2xl">Queue</h2>
			<div class="flex">
				<input class="flex-1 p-2 border border-blue-600" name="post_url" id="post_url" placeholder="submit an URL" />
				<button class="flex-none p-2" type="submit">add</button>
			</div>
	</form>
	<ul id="room_queue_list">
		{#each session.playlist as infoJson}
			<li class="hidden" id={infoJson.ID}>{JSON.stringify(infoJson)}</li>
			<RowMusicCard {infoJson}/>
		{/each}
	</ul>
</div>
