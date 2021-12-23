<script type="ts">
  export let onDelete: () => void;
</script>

<div class="chip">
  <slot />
  {#if onDelete}
    <button class="delete-chip" on:click={onDelete}>❌</button>
  {/if}
</div>

<style>
  .chip {
    position: relative;
    background: lightgray;
    width: fit-content;
    padding: 4px 8px;
    border-radius: 4px;
  }
  .delete-chip {
    display: none;
    position: absolute;
    top: -12px;
    right: -12px;
    font-family: monospace;
    border-radius: 100%;
  }
  .chip:hover > .delete-chip {
    display: inline-block;
    cursor: position;
  }
</style>
