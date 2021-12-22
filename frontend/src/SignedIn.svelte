<script type="ts">
import App from "./App.svelte";

  export let address: string;
  export let token: string;

  const ws = new WebSocket("wss://address-chat-api.fly.dev/ws");
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

  function keypressHandler(evt: KeyboardEvent) {
    if (evt.ctrlKey && evt.key === "Enter") {
      const target = evt.target as HTMLTextAreaElement;
      ws.send(
        JSON.stringify({ from: address, to: address, content: target.value })
      );
      target.value = "";
    }
  }
</script>

<h1>Signed in as {address}</h1>
<textarea
  placeholder={authenticatedUntil ? "Type message here" : "Connecting...."}
  disabled={!authenticatedUntil}
  on:keypress={keypressHandler}
/>
