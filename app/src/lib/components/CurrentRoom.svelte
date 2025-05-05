<script>
	import { session, API_PATH } from "../../scripts/index.svelte.js";

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

	// state
	let hostname = $derived.by(() => {
		if (session.hostID === null) {
			// console.log(`search host`);
			for (const id in session.userList) {
				if (session.userList[id].host === true) {
					return session.userList[id].name;
				}
			}
		} else {
			// console.log(`index host`);
			return session.userList[session.hostID].name;
		}
	});

	let capacity = $derived.by(() => {
		return Object.keys(session.userList).length;
	});
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

	<ul id="room_user_list" class="current_user_list">
		{#each Object.entries(session.userList) as [id, val]}
			<li {id}>user name: {val.name}<br />id: {id}</li>
		{/each}
	</ul>
</div>

<style>
	.current_user_list {
		width: auto;
		height: 120px;
		overflow: auto;
		background-color: rgb(65, 104, 117);
	}
</style>
