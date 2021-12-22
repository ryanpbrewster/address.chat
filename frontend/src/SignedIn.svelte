<script type="ts">
  export let address: string;
  export let token: string;

  import { ethers } from 'ethers';
  const provider = new ethers.providers.Web3Provider((window as any).ethereum);

  const ws = new WebSocket("ws://localhost:8080/ws");
  let authenticatedUntil: number | null = null;
  ws.onopen = (evt) => {
    console.log("[OPEN]", evt);
    ws.send(token);
  };
  ws.onmessage = (evt) => {
    console.log("[MESSAGE]", evt);
    try {
      const msg = JSON.parse(evt.data);
      if (typeof msg.authenticatedUntil === "number") {
        authenticatedUntil = msg.authenticatedUntil || null;
      }
    } catch (e) {
      console.error(e);
    }
  };
  ws.onerror = (evt) => console.log("[ERROR]", evt);
  ws.onclose = (evt) => console.log("[CLOSE]", evt);

  let recipient = "";
  let resolvingName = false;
  async function toHandler(evt: KeyboardEvent) {
    if (evt.key !== "Enter") return;
    if (resolvingName) return;
    const target = evt.target as HTMLInputElement;
    resolvingName = true;
    try {
      recipient = await provider.resolveName(recipient);
    } finally {
      resolvingName = false;
    }
  }
  let content = "";
  function contentHandler(evt: KeyboardEvent) {
    if (evt.ctrlKey && evt.key === "Enter") {
      tryFlush();
    }
  }
  async function tryFlush() {
    if (authenticatedUntil && recipient && content) {
      ws.send(
        JSON.stringify({ from: address, to: recipient, content})
      );
      content = "";
    }
  }
</script>

<div class="center">
<h1>Signed in as {address}</h1>
<input
  type="text"
  placeholder="ryanbrewster.eth"
  bind:value={recipient}
  disabled={!authenticatedUntil || resolvingName}
  on:keypress={toHandler}
/>

<textarea
  placeholder={authenticatedUntil ? "Type message here" : "Connecting...."}
  disabled={!authenticatedUntil}
  bind:value={content}
  on:keypress={contentHandler}
/>
<button disabled={recipient === "" || content === ""}>Send</button>
</div>

<style>
  .center {
    display: flex;
    flex-direction: column;
    max-width: 640px;
  }
</style>