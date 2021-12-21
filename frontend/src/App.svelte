<script lang="ts">
	import Connected from "./Connected.svelte";
	const ethereum = (window as any).ethereum;
	let address: string | null = null;
	ethereum.request({ method: "eth_requestAccounts"}).then((accounts) => {
		address = accounts[0];
	})
</script>

<main>
	{#if !address}
	<p>please sign in w/ metamask</p>
	{:else}
	<Connected {address} />
	{/if}
</main>

<style>
	main {
		text-align: center;
		padding: 1em;
		max-width: 240px;
		margin: 0 auto;
	}

	h1 {
		color: #ff3e00;
		font-size: 4em;
		font-weight: 100;
	}

	@media (min-width: 640px) {
		main {
			max-width: none;
		}
	}
</style>