<script>
    import Layout from "./lib/layout.svelte"
    import { onMount, onDestroy } from "svelte"
    import { sendJson, mkUrl, timeStr } from "./lib/base"

    let timer

    onMount(() => {
        showStatus()

        timer = setInterval(function () {
            showStatus()
        }, 2000)
    })

    onDestroy(() => {
        clearInterval(timer)
    })

    let rows = []

    async function showStatus() {
        let json = await sendJson(mkUrl("api/task/timed"), { Starttime: 0 })

        rows = json.Data || []
    }

    async function rowDel(row) {
        var ok = confirm(`Del timer?\r\nId:${row.Id} name: ${row.name}`)
        if (ok) {
            let json = await sendJson(mkUrl("api/task/timeddel"), {
                TimedID: row.TimedID,
            })
            if (json.Code != 0) {
                alert(json.Data)
                return
            }
            await showStatus()
        }
    }
</script>

<Layout tab="8">
    <div id="tasks">
        <table class="m-4 border text-base text-gray-800">
            <thead>
                <tr>
                    <th class="px-2 py-1 border">运行时间</th>
                    <th class="px-2 py-1 border">任务</th>
                    <th class="px-2 py-1 border" />
                </tr>
            </thead>
            <tbody>
                {#each rows as row}
                    <tr>
                        <td class="px-2 py-1 border">{timeStr(row.timer)}</td>
                        <td class="px-2 py-1 border">{JSON.stringify(row)}</td>
                        <td class="px-2 py-1 border"><button on:click={() => rowDel(row)}>删除</button></td>
                    </tr>
                {:else}
                    <tr>
                        <td colspan="3" class="px-2 py-1 border text-center">empty</td>
                    </tr>
                {/each}
            </tbody>
        </table>
    </div>
</Layout>
