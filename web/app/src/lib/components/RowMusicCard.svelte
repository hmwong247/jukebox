<script>
    /** @typedef {import('../../scripts/index.svelte.js').InfoJson} InfoJson */
    let { infoJson } = $props();

    let durationMin = Math.trunc(infoJson.Duration / 60);
    let durationSec = `${Math.trunc(infoJson.Duration % 60)}`.padStart(
        2,
        "0",
    );

    async function fetchThumbnail() {}
</script>

<div class="playlist_card">
    <p class="place-content-center">{infoJson.ID}</p>
    <img class="m-2 w-auto h-16" src={infoJson.Thumbnail} alt="thumbnail" />
    <div class="flex-1 flex-col">
        <p class="text-xl p-2">{infoJson.FullTitle}</p>
        <p class="text-md p-2">{infoJson.Uploader}</p>
    </div>
    <div class="flex flex-end items-center">
        <p class="p-2">{durationMin}:{durationSec}</p>
        <button class="p-2">del</button>
    </div>
</div>
