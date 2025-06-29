<script>
	import { session, API_PATH } from "../../scripts/index.svelte.js";
	import { createTooltip } from "@melt-ui/svelte";
	import { writable } from "svelte/store";

	/**
	 * @param {SubmitEvent} event
	 */
	async function submitURL(event) {
		event.preventDefault();
		const form = /** @type {HTMLFormElement} */ (event.currentTarget);
		const formData = new FormData(form);
		formData.append("user_id", window.localStorage.getItem("userID"));

		try {
			const res = await fetch(
				API_PATH.ENQUEUE + "?sid=" + session.sessionID,
				{
					method: "POST",
					body: formData,
				},
			);

			if (res.ok) {
				console.log("Success:", res.status);
				form.reset();
			} else {
				console.log("Error:", res.status);
			}
		} catch (err) {
			console.error("Fetch error:", err);
		}
	}

	// Create MeltUI tooltip for the submit button
	const tooltip = createTooltip({
		positioning: {
			placement: "top",
		},
	});

	const sortKey = writable("id");
	const sortAsc = writable(true);

	function sortBy(key) {
		sortKey.set(key);
		sortAsc.update((asc) => (key === $sortKey ? !asc : true));
	}

	let sortedPlaylist = (() => {
		const playlist = session.playlist || [];
		const key = $sortKey;
		const asc = $sortAsc;
		return [...playlist].sort((a, b) => {
			let va, vb;
			switch (key) {
				case "id":
					va = a.ID ?? a.id ?? 0;
					vb = b.ID ?? b.id ?? 0;
					break;
				case "title":
					va = (a.title || "").toLowerCase();
					vb = (b.title || "").toLowerCase();
					break;
				case "uploader":
					va = (a.uploader || a.artist || "").toLowerCase();
					vb = (b.uploader || b.artist || "").toLowerCase();
					break;
				case "duration":
					va = a.duration || 0;
					vb = b.duration || 0;
					break;
				default:
					va = a[key];
					vb = b[key];
			}
			if (va < vb) return asc ? -1 : 1;
			if (va > vb) return asc ? 1 : -1;
			return 0;
		});
	})();
</script>

<div id="music_queue" class="w-full max-w-2xl mx-auto p-6">
	<div class="bg-gray-800 rounded-lg shadow-lg border border-gray-700">
		<div class="p-6 border-b border-gray-700">
			<h2 class="text-xl font-semibold text-white">Music Queue</h2>
		</div>

		<div class="p-6">
			<form onsubmit={submitURL}>
				<fieldset class="border-none p-0 m-0">
					<div class="flex gap-3 items-end">
						<div class="flex-1">
							<input
								name="post_url"
								id="post_url"
								placeholder="Enter a URL..."
								class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded-md text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors"
								required
							/>
						</div>

						<div use:tooltip.elements.trigger>
							<button
								type="submit"
								class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:ring-offset-gray-800 transition-colors"
							>
								Add to Queue
							</button>
						</div>
					</div>
				</fieldset>
			</form>
		</div>

		<div class="p-6 border-t border-gray-700">
			<h3 class="text-lg font-medium text-white mb-3">Current Queue</h3>
			{#if session.playlist && session.playlist.length > 0}
				<div class="overflow-x-auto rounded-lg">
					<table
						class="min-w-full text-sm text-left text-gray-300 dark:text-gray-300"
					>
						<thead class="bg-gray-700 text-gray-200">
							<tr>
								<th
									class="px-4 py-2 cursor-pointer"
									onclick={() => sortBy("id")}
								>
									ID
								</th>
								<th class="px-4 py-2">Thumbnail</th>
								<th
									class="px-4 py-2 cursor-pointer"
									onclick={() => sortBy("title")}
								>
									Title
								</th>
								<th
									class="px-4 py-2 cursor-pointer"
									onclick={() => sortBy("uploader")}
								>
									uploader
								</th>
								<th
									class="px-4 py-2 cursor-pointer"
									onclick={() => sortBy("duration")}
								>
									duration
								</th>
							</tr>
						</thead>
						<tbody class="bg-gray-800 divide-y divide-gray-700">
							{#each sortedPlaylist as infoJson, index}
								<tr>
									<td class="px-4 py-3 align-middle"
										>{infoJson.ID ??
											infoJson.id ??
											index + 1}</td
									>
									<td class="px-4 py-3 align-middle">
										{#if infoJson.thumbnail}
											<img
												src={infoJson.thumbnail}
												alt="thumbnail"
												class="w-16 h-16 object-cover rounded-lg border border-gray-600"
											/>
										{:else}
											<div
												class="w-16 h-16 flex items-center justify-center bg-gray-700 rounded-lg border border-gray-600 text-gray-400"
											>
												thumbnail
											</div>
										{/if}
									</td>
									<td
										class="px-4 py-3 align-middle text-white"
										>{infoJson.title || "<title>"}</td
									>
									<td class="px-4 py-3 align-middle"
										>{infoJson.uploader ||
											infoJson.artist ||
											"<uploader>"}</td
									>
									<td
										class="px-4 py-3 align-middle text-right"
										>{infoJson.duration || ""}</td
									>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{:else}
				<div class="text-center py-8 text-gray-400">
					<p>No tracks in queue</p>
					<p class="text-sm mt-1">Add a URL above to get started</p>
				</div>
			{/if}
		</div>
	</div>
</div>
