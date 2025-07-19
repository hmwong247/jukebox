<script>
	// component
	import UserProfile from "@components/UserProfile.svelte";
	import CurrentRoom from "@components/CurrentRoom.svelte";
	import MusicPlayer from "@components/MusicPlayer.svelte";
	import MusicQueue from "@components/MusicQueue.svelte";

	// internal library
	import { session } from "@lib/index.svelte.js";

	// debug
	// $inspect(`app: `, session)
</script>

<main>
	{#if Object.keys(session.userList).length === 0}
		<UserProfile />
	{:else}
		<CurrentRoom />
		<MusicPlayer />
		<MusicQueue />
	{/if}
</main>

<style>
</style>
