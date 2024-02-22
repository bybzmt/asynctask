<script>
    import Layout from "./lib/layout.svelte"
    import Task from "./lib/task.svelte"
    import { sendJson, mkUrl } from "./lib/base"

    let addTask = ""
    let errmsg = ""
    let taskinfo = ""

    async function doCheck() {
        taskinfo = ""
        errmsg = ""
        try {
            let task = JSON.parse(addTask)

            let resp = await sendJson(mkUrl("api/task/check"), task)

            if (resp.Code != 0) {
                errmsg = resp.Data
                return
            }

            taskinfo = JSON.stringify(resp.Data, null, 2)
        } catch (e) {
            errmsg = e.toString()
        }
    }

    async function doAdd() {
        taskinfo = ""
        errmsg = ""

        try {
            let task = JSON.parse(addTask)

            let resp = await sendJson(mkUrl("api/task/add"), task)

            if (resp.Code != 0) {
                errmsg = resp.Data
                return
            }

            addTask = ""

            alert(JSON.stringify(resp))
        } catch (e) {
            errmsg = e.toString()
        }
    }
</script>

<Layout tab="9">
    <div class="m-8">
        <Task bind:value={addTask} />
        <div class="my-2">
            <button on:click={doCheck}>check</button>
            <button class="mx-4" on:click={doAdd}>add</button>
        </div>

        <div class="my-4 text-xs text-red-800">
            <pre>{errmsg}</pre>
        </div>

        <div class="my-4 text-xs text-gray-800">
            <pre>{taskinfo}</pre>
        </div>
    </div>
</Layout>
