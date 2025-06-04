<script>
	import { api, session } from "../../scripts/index.svelte.js";

	function onsubmit(event) {
		if (window.location.href.includes("join?rid=")) {
			api.requestJoinRoom(event, this);
		} else {
			api.requestNewRoom(event, this);
		}
	}

</script>

<form name="user_profile" {onsubmit}>
	<fieldset>
		<legend>Your Profile</legend>
		<div>
			<label for="cfg_username">User name: </label>
			<input
				name="cfg_username"
				id="cfg_username"
				placeholder="(optional)"
			/>
		</div>
		<br />
		<button type="submit">Continue</button>
	</fieldset>
</form>
