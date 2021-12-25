<script type="ts">
  export let address: string;
  export let token: string;

  interface Mailbox {
    readonly address: string;
    readonly name?: string;
  }
  interface Message {
    readonly from: string;
    readonly to: readonly string[];
    readonly content: string;
  }

  import { ethers } from "ethers";
  import Mailbox from "./Mailbox.svelte";
  import Chip from "./Chip.svelte";
  const provider = new ethers.providers.Web3Provider((window as any).ethereum);

  let author: Mailbox = { address };
  provider.lookupAddress(address).then((name) => {
    console.log("rev lookup", address, "=", name);
    author = { ...author, name };
  });

  // const ws = new WebSocket("wss://address-chat-api.fly.dev/ws");
  const ws = new WebSocket("ws://localhost:8080/ws");
  let authenticatedUntil: number | null = null;
  let messages: readonly Message[] = [];
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
      } else {
        messages = [...messages, msg];
      }
    } catch (e) {
      console.error(e);
    }
  };
  ws.onerror = (evt) => console.log("[ERROR]", evt);
  ws.onclose = (evt) => console.log("[CLOSE]", evt);

  const ADDRESS_REGEX = /^0x[a-fA-F0-9]{40}$/;
  const ENS_REGEX = /^([a-z]+\.)+eth$/;

  let recipients: readonly Mailbox[] = [];
  function deleteRecipient(i: number) {
    recipients = [...recipients.slice(0, i), ...recipients.slice(i + 1)];
  }
  let partialRecipient: string = "";
  async function toHandler(evt: KeyboardEvent) {
    if (evt.key !== "Enter") return;
    let recipient: Mailbox | null = null;
    if (ENS_REGEX.test(partialRecipient)) {
      const address = await provider.resolveName(partialRecipient);
      recipient = address ? { address, name: partialRecipient } : null;
    } else if (ADDRESS_REGEX.test(partialRecipient)) {
      const name = await provider.lookupAddress(partialRecipient);
      recipient = { address: partialRecipient, name };
    }
    if (recipient) {
      recipients = [...recipients, recipient];
      partialRecipient = "";
    }
  }

  let content = "";
  function contentHandler(evt: KeyboardEvent) {
    if (evt.key !== "Enter") return;
    if (!evt.ctrlKey) return;
    tryFlush();
  }
  async function tryFlush() {
    if (authenticatedUntil && recipients.length > 0 && content) {
      // TODO: update protocol to support multiple recipients
      ws.send(
        JSON.stringify({ from: address, to: recipients.map((m) => m.address), content })
      );
      content = "";
    }
  }
</script>

<div>
  {#each messages as m}
  <p>{m.from} -> {m.to}: {m.content}</p>
  {/each}
</div>
<div class="center">
  <table>
    <tbody>
      <tr
        ><td>From:</td><td
          ><Mailbox name={author.name} address={author.address} />
        </td></tr
      >
      <tr
        ><td>To:</td><td>
          <div>
            {#each recipients as recipient, i}
              <Chip onDelete={() => deleteRecipient(i)}
                ><Mailbox
                  name={recipient.name}
                  address={recipient.address}
                /></Chip
              >
            {/each}
            <input
              type="text"
              placeholder="ryanbrewster.eth"
              bind:value={partialRecipient}
              on:keypress={toHandler}
            />
          </div>
        </td></tr
      >
    </tbody>
  </table>

  <textarea
    placeholder={authenticatedUntil ? "Type message here" : "Connecting...."}
    disabled={!authenticatedUntil}
    bind:value={content}
    on:keypress={contentHandler}
  />
  <button
    disabled={partialRecipient.length > 0 ||
      recipients.length === 0 ||
      content.length === 0}>Send</button
  >
</div>

<style>
  .center {
    display: flex;
    flex-direction: column;
    max-width: 640px;
  }
  table {
    text-align: left;
  }
  input {
    width: 30em;
  }
</style>
