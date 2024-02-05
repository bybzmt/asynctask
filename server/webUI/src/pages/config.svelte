<script>
    import Layout from "./lib/layout.svelte"
    import { onMount } from "svelte"
    import { mkUrl } from "./lib/base"

    onMount(() => {
        showStatus()
    })

    let config = null

    async function showStatus() {
        let json = await fetch(mkUrl("api/config")).then(t => t.json())

        config = JSON.stringify(json.Data, null, 2)
    }
</script>

<Layout tab="6">
    <div class="p-4 text-sm">
        <pre>{config}</pre>
    </div>
</Layout>
