<script>
    import Layout from "./lib/layout.svelte";
    import { onMount } from "svelte";
    import { sendJson, mkUrl } from "./lib/base";

    onMount(() => {
        showStatus();
    });

    let router = "";

    async function showStatus() {
        let json = await sendJson(mkUrl("api/router/get"));

        router = (json.Data || []).join("\n");
    }

    async function save() {
        let str = router.split(/\n|\r/);

        let routes = [];
        let chk = new Map();

        for (let s of str) {
            let x = s.trim();
            if (x.length > 0) {
                if (chk.has(x)) {
                    alert(x + " duplicate");
                    return;
                }

                chk.set(x, 1);
                routes.push(x);
            }
        }

        let resp = await sendJson(mkUrl("api/router/set"), routes);

        alert(JSON.stringify(resp));

        if (resp.Code == 0) {
            await showStatus();
        }
    }
</script>

<Layout tab="3">
    <div class="m-4">
        <div>
            <textarea
                class="border min-w-[500px] min-h-[200px]"
                bind:value={router}
            />
        </div>
        <div class="my-2"><button on:click={() => save()}>Save</button></div>

        <div class="my-4 text-gray-500">
            router决定什么任务能运行<br />

            example:<br />
            ^https?://xxx<br />
            ^cli://xxx
        </div>
    </div>
</Layout>
