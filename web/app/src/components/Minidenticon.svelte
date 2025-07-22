<script>
	import { minidenticon } from "minidenticons";

	let {
		username,
		// default values from minidenicons
		saturation = 95,
		lightness = 45,
		...others
	} = $props();

	let svgUrl = $derived(
		"data:image/svg+xml;utf8," +
			encodeURIComponent(minidenticon(username, saturation, lightness)),
	);
</script>

<img src={svgUrl} alt={username} {...others} />
