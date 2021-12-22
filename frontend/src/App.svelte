<script lang="ts">
  import SignedIn from "./SignedIn.svelte";

  interface AuthPayload {
    readonly address: string;
    readonly expiresAt: number;
  }

  const ethereum = (window as any).ethereum;
  let address: string | null = null;
  let token: string | null = null;
  async function requestAccounts() {
    const accounts = await ethereum.request({ method: "eth_requestAccounts" });
    address = accounts[0];
  }
  async function signToken(address: string) {
	const payload: AuthPayload = {address, expiresAt: Date.now() + 3_600_000 };
	const body = JSON.stringify(payload);
    const signature = await ethereum.request({
      method: "personal_sign",
      params: [body, address],
    });
	token = JSON.stringify({payload, signature});
  }
</script>

<main>
  {#if !address}
    <button on:click={requestAccounts}>Connect w/ Metamask</button>
  {:else if !token}
    <button on:click={() => signToken(address)}>Sign in as {address}</button>
  {:else}
    <SignedIn {address} {token} />
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
