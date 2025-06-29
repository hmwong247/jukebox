<script>
	// component
	import UserProfile from "./lib/components/UserProfile.svelte";
	import CurrentRoom from "./lib/components/CurrentRoom.svelte";
	import MusicPlayer from "./lib/components/MusicPlayer.svelte";
	// import MusicQueue from "./lib/components/MusicQueue.svelte";
	import MusicQueueMeltUI from "./lib/components/MusicQueueMeltUI.svelte";

	// internal library
	import { session } from "./scripts/index.svelte.js";

	// debug
	// $inspect(`app: `, session)
</script>

<main>
	{#if Object.keys(session.userList).length === 0}
		<UserProfile />
	{:else}
		<CurrentRoom />
		<MusicPlayer />
		<!-- <MusicQueue /> -->
		<MusicQueueMeltUI />
	{/if}
</main>

<style>
</style>
