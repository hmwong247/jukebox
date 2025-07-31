<script>
	import { session, TASK_STATUS_STR } from "@scripts/index.svelte";
	import { Spinner } from "flowbite-svelte";

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
	<div class="flex items-center">
		{#if task.Status == TASK_STATUS_STR.LOADING}
			<Spinner color="red" />
		{/if}
		{#if task.Status != TASK_STATUS_STR.LOADING && task.Status != TASK_STATUS_STR.OK}
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
