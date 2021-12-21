<script type="ts">
    export let address: string;
    export let challenge: string;

	const ethereum = (window as any).ethereum;
    let token: string | null = null;
    
	ethereum.request({ method: "personal_sign", params: [challenge, address]}).then((signature) => {
        return fetch(`http://localhost:8080/auth/signin`, {
            method: "POST",
            body: JSON.stringify({Address: address, Challenge: challenge, Signature: signature}),
        }).then(async (resp) => {
            console.log(resp);
            token = (await resp.json()).Token;
        })
    });
</script>

{#if !token}
<h1>Signing challenge....</h1>
{:else}
<h1>Signed in as {address}</h1>
{/if}