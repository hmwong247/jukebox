<script>
    /** @typedef {import('@scripts/index.svelte.js').InfoJson} InfoJson */
    let { infoJson } = $props();

    let durationMin = Math.trunc(infoJson.Duration / 60);
    let durationSec = `${Math.trunc(infoJson.Duration % 60)}`.padStart(2, "0");
</script>

<div class="playlist-card">
    <p class="place-content-center">{infoJson.ID}</p>
    <img class="m-2 w-auto h-16" src={infoJson.Thumbnail} alt="thumbnail" />
    <div class="flex-1 flex-col min-w-[10%]">
        <p class="text-xl p-1 overflow-hidden truncate text-ellipsis">
            {infoJson.FullTitle}
        </p>
        <p
            class="text-md text-light p-1 overflow-hidden truncate text-ellipsis"
        >
            {infoJson.Uploader}
        </p>
    </div>
    <p class="p-2 flex-initial self-center">
        {durationMin}:{durationSec}
    </p>
    <button class="p-2 flex-initial self-center">del</button>
</div>
