<script type="ts">
  export let address: string;
  export let token: string;

  function sleep(millis: number): Promise<void> {
    return new Promise((r) => setTimeout(r, millis));
  }

  type UnixMillis = number;
  interface Mailbox {
    readonly address: string;
    readonly name?: string;
  }
  interface Message {
    readonly sentAt: UnixMillis;
    readonly from: string;
    readonly to: readonly string[];
    readonly content: string;
  }
  interface SyncMessage {
    readonly seqno: number;
    readonly messages: readonly Message[];
  }
  interface Group {
    readonly members: readonly string[];
  }
  interface GroupedMessages {
    readonly timestamp: UnixMillis;
    readonly group: Group;
    readonly messages: readonly Message[];
  }
  function extractGroup(message: Message): Group {
    const members = new Set([message.from, ...message.to]);
    members.delete(address);
    return { members: members.size === 0 ? [address] : [...members].sort() };
  }
  function groupKey(group: Group): string {
    return group.members.join(":");
  }
  function groupMessages(
    messages: readonly Message[]
  ): readonly GroupedMessages[] {
    const grouped: Map<string, GroupedMessages> = new Map();
    for (const msg of messages) {
      const group = extractGroup(msg);
      const key = groupKey(group);
      const cur = grouped.get(key);
      const updated: GroupedMessages = cur
        ? {
            group,
            messages: [...cur.messages, msg].sort(
              (a, b) => a.sentAt - b.sentAt
            ),
            timestamp: Math.max(cur.timestamp, msg.sentAt),
          }
        : { group, messages: [msg], timestamp: msg.sentAt };
      grouped.set(key, updated);
    }
    // Sort in descending timestamp order.
    return [...grouped.values()].sort((a, b) => b.timestamp - a.timestamp);
  }

  import { ethers } from "ethers";
  const provider = new ethers.providers.Web3Provider((window as any).ethereum);

  let author: Mailbox = { address };
  provider.lookupAddress(address).then((name) => {
    author = { ...author, name };
  });

  const ws = new WebSocket("wss://address-chat-api.fly.dev/ws");
  // const ws = new WebSocket("ws://localhost:8080/ws");
  let messages: readonly Message[] = [];
  $: groupedMessages = groupMessages(messages);
  let selectedGroup: Group | null = null;
  ws.onopen = (evt) => {
    console.log("[OPEN]", evt);
    ws.send(token);
  };
  ws.onmessage = (evt) => {
    console.log("[MESSAGE]", evt);
    try {
      const msg: SyncMessage = JSON.parse(evt.data);
      messages = [...messages, ...msg.messages];
    } catch (e) {
      console.error(e);
    }
  };
  ws.onerror = (evt) => console.log("[ERROR]", evt);
  ws.onclose = (evt) => console.log("[CLOSE]", evt);

  const ADDRESS_REGEX = /^0x[a-fA-F0-9]{40}$/;
  const ENS_REGEX = /^([a-z]+\.)+eth$/;

  async function startConversation(rawRecipients: string) {
    const tokens = rawRecipients.split(/[\s,]+/).filter((t) => t.length > 0);
    console.log("parsing recipients from:", tokens);
    const recipients: Mailbox[] = [];
    for (const token of tokens) {
      if (ENS_REGEX.test(token)) {
        console.log("trying to resolve ENS domain:", token);
        const address: string | null = await Promise.race([
          provider.resolveName(token),
          sleep(1_000).then(() => null),
        ]);
        if (!address) throw new Error(`could not resolve ${token}`);
        recipients.push({ address, name: token });
      } else if (ADDRESS_REGEX.test(token)) {
        const name: string | null = await Promise.race([
          provider.lookupAddress(token),
          sleep(1_000).then(() => null),
        ]);
        recipients.push({ address: token, name });
      } else {
        throw new Error(`invalid recipient: ${token}`);
      }
    }
    ws.send(
      JSON.stringify({
        from: address,
        to: recipients.map((r) => r.address),
        content: "Let's chat!",
      })
    );
  }

  let content = "";
  function contentHandler(evt: KeyboardEvent) {
    if (evt.key !== "Enter") return;
    if (!evt.ctrlKey) return;
    if (!content) return;
    // TODO: update protocol to support multiple recipients
    ws.send(
      JSON.stringify({
        from: address,
        to: selectedGroup.members,
        content,
      })
    );
    content = "";
  }
</script>

<div class="body">
  <div class="leftnav">
    <button
      on:click={() =>
        startConversation(prompt("Input the addresses, comma separated"))}
    >
      New Conversation
    </button>
    {#each groupedMessages as grouped}
      <div class="navitem" on:click={() => (selectedGroup = grouped.group)}>
        {grouped.group.members.join("\n")}
      </div>
    {/each}
  </div>
  {#if selectedGroup}
    <div class="messages">
      {#each groupedMessages.find((grouped) => groupKey(grouped.group) === groupKey(selectedGroup))?.messages ?? [] as m}
        <p>{m.from}: {m.content}</p>
      {/each}
      <textarea
        placeholder="Type message here"
        bind:value={content}
        on:keypress={contentHandler}
      />
    </div>
  {/if}
</div>

<style>
  .body {
    display: flex;
    flex-direction: row;
    height: 800px;
  }
  .leftnav {
    display: flex;
    flex-direction: column;
    width: 400px;
    height: 100%;
    padding: 16px;
    border: solid 1px black;
    overflow: auto;
  }
  .navitem {
    border: solid 1px lightgray;
    height: 80px;
    padding: 4px;
    border-radius: 4px;
    text-align: left;
  }
  .messages {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    padding: 8px;
    width: 680px;
    height: 100%;
    overflow: auto;
  }
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
