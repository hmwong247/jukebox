<script>
    import { session, API_PATH } from "@scripts/index.svelte";

    /** @typedef {import('@scripts/index.svelte.js').InfoJson} InfoJson */
    let { infoJson } = $props();

    let durationMin = Math.trunc(infoJson.Duration / 60);
    let durationSec = `${Math.trunc(infoJson.Duration % 60)}`.padStart(2, "0");

    let disableDel = $state(false)

    async function deleteCard() {
        disableDel = true
        const path = `${API_PATH.QUEUE}?sid=${session.sessionID}`;
        await fetch(path, {
            method: "POST",
            body: JSON.stringify({
                Cmd: "del",
                NodeID: infoJson.ID,
            }),
        })
            .then((res) => {
                if (!res.ok) {
                    throw new Error(`delete error:, ${res.status}`);
                }
            })
            .catch((e) => {
                console.error(e);
                disableDel = false
            });
    }
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
    <button class="p-2 flex-initial self-center" onclick={deleteCard} disabled={disableDel}
        >del</button
    >
</div>
