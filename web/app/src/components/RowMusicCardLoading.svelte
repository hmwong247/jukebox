<script>
	import { session } from "@scripts/index.svelte";
	/** @typedef {import('@scripts/index.svelte.js').InfoJsonTaskStatus} InfoJsonTaskStatus*/

	let { task } = $props();

	function onclick() {
		session.queuelist.forEach((item, index) => {
			if (item.TaskID == task.TaskID) {
				session.queuelist.splice(index, 1);
				return;
			}
		});
	}
</script>

<div class="playlist-card">
	<div class="flex self-end">
		{#if task.Status == "loading"}
			<div class="p-2">spin</div>
		{/if}
		{#if task.Status != "loading" && task.Status != "ok"}
			<button class="p-2" {onclick}>del</button>
		{/if}
	</div>
	<div class="flex flex-col mx-2">
		<h2>{task.URL}</h2>
		<div class="flex gap-2">
			<div>taskID: {task.TaskID}</div>
			<div>taskStatus: {task.Status}</div>
		</div>
	</div>
</div>
