<script>
    import { onMount } from "svelte";
    import {
        session,
        mp,
        rtc,
        API_PATH,
        PEER_CMD,
    } from "../../scripts/index.svelte.js";

    // bind
    let mpProgress;
    let mpVolume;

    // $inspect(mp).with((t, mp) => {
    //     if (t === "update") {
    //     }
    // });

    let mpPlayButton = $derived(mp.running);
    let isHost = $derived(session.userID === session.hostID ? true : false);
    let mpCurrentTimeMin = $state(0),
        mpCurrentTimeSec = $state("00");
    let mpDurationMin = $state(0),
        mpDurationSec = $state("00");

    $effect(() => {
        if (isHost) {
            mp.elem.volume = 1; // reset the stream source volume
        }
    });

    function togglePlayPause() {
        if (mp.running) {
            if (mp.elem.paused) {
                mp.elem.play();
            } else {
                mp.elem.pause();
            }
        }
    }

    /**
     *
     */

    /*
        audio events
    */

    function onplay() {
        if (mp.ctx && mp.ctx.state === "suspended") {
            mp.ctx.resume();
        }

        if (isHost) {
            const msg = { from: session.userID, payload: PEER_CMD.PLAY };
            rtc.allPeers(msg);
        }
    }

    function onpause() {
        if (isHost) {
            const msg = { from: session.userID, payload: PEER_CMD.PAUSE };
            rtc.allPeers(msg);
        }
    }

    function ontimeupdate() {
        // labels
        mpCurrentTimeMin = Math.trunc(mp.elem.currentTime / 60);
        mpCurrentTimeSec = `${Math.trunc(mp.elem.currentTime % 60)}`.padStart(
            2,
            "0",
        );

        // progress bar
        mpProgress.value = mp.elem.currentTime;
    }

    function mpseek() {
        mp.elem.currentTime = mpProgress.value;
    }

    function mpchangevolume() {
        if (isHost) {
            // host volume is decoupled from the stream
            mp.gainNode.gain.value = mpVolume.value;
        } else {
            mp.gainNode.gain.value = 0;
            mp.elem.volume = mpVolume.value;
        }
    }

    function onloadedmetadata() {
        // labels
        mpDurationMin = Math.trunc(session.playlist[0].Duration / 60);
        mpDurationSec =
            `${Math.trunc(session.playlist[0].Duration % 60)}`.padStart(2, "0");

        // progress bar
        mpProgress.max = session.playlist[0].Duration;
    }

    function mpcanplay() {
        const newTrack = mp.localStream.getTracks()[0];
        rtc.startSyncPeer(mp.currentTrack, newTrack, mp.hostStream);
        if (mp.currentTrack !== null) {
            mp.hostStream.removeTrack(mp.currentTrack);
        }
        mp.hostStream.addTrack(newTrack);
        mp.currentTrack = newTrack;
    }

    async function mpprefetch() {
        if (
            mp.elem.currentTime / mp.elem.duration >= 0.5 &&
            session.playlist.length > 1
        ) {
            mp.elem.removeEventListener("timeupdate", mpprefetch);

            const url = API_PATH.STREAM_PRELOAD + "?sid=" + session.sessionID;
            const res = await fetch(url);
            if (res.ok) {
                // const s = await res.json()
                // if (s == false) {
                // }
            }
        }
    }

    function onloadstart() {
        console.log(`loadstart`);
        if (isHost) {
            mp.elem.addEventListener("canplay", mpcanplay, { once: true });
            mp.elem.addEventListener("timeupdate", mpprefetch);
        }

        // peers will run the mp after they recieved the MediaTrack from host, i.e. onpeerstream
        mp.elem.play();
        mp.running = true;
    }

    function onended() {
        console.log(`ended`);

        if (isHost) {
            mp.elem.pause();
            mp.elem.currentTime = 0;
            mpCurrentTimeMin = 0;
            mpCurrentTimeSec = "00";
            mpDurationMin = 0;
            mpDurationSec = "00";
            mpProgress.value = 0;
            const endedJson = session.playlist.shift();

            // host will fetch the next music
            const url = API_PATH.STREAM_END + "?sid=" + session.sessionID;
            const response = fetch(url);
            // .then wait for server to reponse the next audio is ready if the queue is not size of 0

            if (session.playlist.length > 0) {
                // wait for 20ms to switch audio
                // await new Promise(r => setTimeout(r, 20))
                // if ok
                rtc.loadAudioAsHost();
                const msg = { from: session.userID, payload: PEER_CMD.NEXT };
                rtc.allPeers(msg);
            } else {
                const msg = { from: session.userID, payload: PEER_CMD.STOP };
                rtc.allPeers(msg);

                mp.elem.removeAttribute("src");
                mp.elem.load();
                mp.running = false;
            }
        }
    }
</script>

<div id="mp-wrapper">
    <section class="mp_info">
        <img src="" alt="Thumbnail" />
        <h2>FullTitle</h2>
        <h4>Uploader</h4>
    </section>
    <section class="mp_controls">
        <button onclick={togglePlayPause} disabled={!mp.running}
            >{mpPlayButton}</button
        >
        <span>currentTime: {mpCurrentTimeMin}:{mpCurrentTimeSec}</span>
        <input
            bind:this={mpProgress}
            id="mp_progress"
            type="range"
            min="0"
            value="0"
            disabled={!mp.running}
            oninput={mpseek}
        />
        <span>duration: {mpDurationMin}:{mpDurationSec}</span>
        <input
            bind:this={mpVolume}
            id="mp_volume"
            type="range"
            value="1"
            min="0"
            max="1"
            step="0.01"
            oninput={mpchangevolume}
        />
    </section>
    <audio
        bind:this={mp.elem}
        id="player"
        controls
        preload="none"
        {onplay}
        {onpause}
        {ontimeupdate}
        {onloadedmetadata}
        {onended}
        {onloadstart}
    ></audio>
</div>

<style>
</style>
