<script type="ts">
  export let name: string | null;
  export let address: string;
</script>

<div>
  {name ? `${name} <${address}>` : address}
</div>
