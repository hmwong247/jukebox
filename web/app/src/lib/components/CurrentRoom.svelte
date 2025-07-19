<script>
	import { session, API_PATH } from "../../scripts/index.svelte.js";
	import Minidenticon from "./Minidenticon.svelte";

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
				content: data.id,
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
			<div
				class="flex flex-col flex-none items-center bg-gray-300 size-12 m-3"
				use:tooltip={{ id: id }}
			>
				<Minidenticon username={val.name + "-" + id} class="size-12" />
				<!-- <li {id}>user name: {val.name}<br />id: {id}</li> -->
				<p>{val.name}</p>
			</div>
		{/each}
	</section>
</div>

<style>
	:global {
		[data-tippy-root] {
			--bg: #666;
			background-color: var(--bg);
			color: white;
			border-radius: 0.2rem;
			padding: 0.2rem 0.6rem;
			filter: drop-shadow(1px 1px 3px rgb(0 0 0 / 0.1));

			* {
				transition: none;
			}
		}

		[data-tippy-root]::before {
			--size: 0.4rem;
			content: "";
			position: absolute;
			left: calc(50% - var(--size));
			top: calc(-2 * var(--size) + 1px);
			border: var(--size) solid transparent;
			border-bottom-color: var(--bg);
		}
	}
</style>
