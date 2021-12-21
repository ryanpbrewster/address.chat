<script type="ts">
import SignedIn from "./SignedIn.svelte";

    export let address: string;

	const ethereum = (window as any).ethereum;
    
    let challenge: string | null = null;
    fetch(`http://localhost:8080/auth/challenge`, {
        method: "POST",
        body: JSON.stringify({Address: address}),
    }).then(async (resp) => {
        console.log(resp);
        challenge = (await resp.json()).Challenge;
    })
</script>

{#if !challenge}
<h1>Awaiting challenge from server...</h1>
{:else}
<SignedIn {address} {challenge} />
{/if}