<script>
	import { session, API_PATH } from "@scripts/index.svelte.js";
	import RowMusicCard from "@components/RowMusicCard.svelte";
	import RowMusicCardLoading from "./RowMusicCardLoading.svelte";

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
					} else {
						console.log("Error:", res.status);
						throw new Error("Failed to submit request");
					}
					return res.text();
				})
				.then((taskID) => {
					// update playlist for loading
					const loading = {
						TaskID: taskID,
						Status: "loading",
						URL: formData.get("post_url"),
					};
					session.queuelist.push(loading);
				})
				.catch((err) => {
					throw err;
				});
		} catch (err) {
			console.error("Fetch error:", err);
		} finally {
			form.reset();
		}
	}
</script>

<div
	id="music_queue"
	class="flex flex-col overflow-auto border rounded-md border-emerald-400"
>
	<form class="music-queue-row p-2" onsubmit={submitURL}>
		<h2 class="text-2xl">Queue</h2>
		<div class="flex gap-1 mt-1">
			<input
				class="flex-1 p-2 border border-blue-600"
				name="post_url"
				id="post_url"
				placeholder="submit an URL"
			/>
			<button class="flex-none p-2" type="submit">add</button>
		</div>
	</form>
	<ul id="room_queue_list" class="music-queue-row">
		<!-- display succesful enqueue -->
		{#each session.playlist as infoJson}
			{#if infoJson.ID != session.playlist[0].ID}
				<li class="hidden" id={infoJson.ID}>
					{JSON.stringify(infoJson)}
				</li>
				<RowMusicCard {infoJson} />
			{/if}
		{/each}
		{#each session.queuelist as task}
			<!-- display loading -->
			{#if task.Status != "ok"}
				<RowMusicCardLoading {task} />
			{/if}
		{/each}
	</ul>
</div>
