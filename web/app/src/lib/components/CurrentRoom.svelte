<script>
	import { session, API_PATH } from "../../scripts/index.svelte.js";
	import { minidenticon } from "minidenticons";
	import tippy from "tippy.js";

	function copyLink() {
		const link = document.querySelector("#room_id").innerHTML;
		if (navigator.clipboard && window.isSecureContext) {
			navigator.clipboard
				.writeText(link)
				.then((data) => {
					console.log(data);
				})
				.catch((err) => {
					console.log(err);
				});
		} else {
			const textArea = document.createElement("textarea");
			textArea.value = link;
			textArea.style.position = "fixed"; // avoid scrolling to bottom
			document.body.appendChild(textArea);
			textArea.select();

			try {
				document.execCommand("copy");
			} catch (err) {
				console.log(err);
			} finally {
				textArea.remove();
			}
		}
	}

	/* state */

	let hostname = $derived.by(() => {
		// do not index a reactive state
		for (const id in session.userList) {
			if (session.userList[id].host === true) {
				return session.userList[id].name;
			}
		}
	});

	let capacity = $derived.by(() => {
		return Object.keys(session.userList).length;
	});

	/* effect */

	function tooltip(elem, data) {
		$effect(() => {
			const tooltip = tippy(elem, {
				interactive: true,
				duration: 5000,
				content: `${data.name}<br>${data.id}`,
			});

			// return tooltip.destroy;
		});
	}
</script>

<div>
	<article>
		<p style="display:inline">user name:</p>
		<div style="display:inline" id="username">{session.username}</div>
		<br />
		<p style="display:inline">roomd id:</p>
		<div style="display:inline" id="room_id">
			{document.location.origin +
				API_PATH.JOIN +
				"?rid=" +
				session.roomID}
		</div>
		<button style="display:inline" onclick={copyLink}>Copy</button>
		<br />
		<p style="display:inline">host:</p>
		<div style="display:inline" id="room_host">{hostname}</div>
		<br />
		<p style="display:inline">capacity:</p>
		<div style="display:inline" id="room_capacity">{capacity}</div>
		<br />
	</article>

	<section id="room_user_list" class="current-user-list flex overflow-auto">
		{#each Object.entries(session.userList) as [id, val]}
			<div class="flex flex-col items-center bg-gray-300 size-12 m-3">
				<!-- <li {id}>user name: {val.name}<br />id: {id}</li> -->
				<minidenticon-svg
					class="inline-block size-12"
					username={val.name + "-" + id}
					use:tooltip={{ id: id, name: val.name }}
				></minidenticon-svg>
				<p>{val.name}</p>
			</div>
		{/each}
	</section>
</div>
