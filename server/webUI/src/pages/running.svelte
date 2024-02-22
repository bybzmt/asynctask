<script>
    import Layout from "./lib/layout.svelte"
    import { onMount, onDestroy } from "svelte"
    import { mkUrl, sendJson } from "./lib/base"

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

    let Runs = []

    async function showStatus() {
        let json = await sendJson(mkUrl("api/task/runing"))

        json.Data.sort((a, b) => (a.StartTime < b.StartTime ? -1 : 1))

        Runs = json.Data
    }

    async function taskCancel(task) {
        var ok = confirm("Cancel Task?\r\nId: " + task.Id + " Name: " + task.Name + " " + task.Task)
        if (ok) {
            await sendJson(mkUrl("api/task/cancel"), {
                Id: task.Id,
            })
        }
    }

    function useTime(t) {
        return (Date.now() - t.StartTime * 1000) / 1000
    }
</script>

<Layout tab="1">
    <div id="tasks">
        <table class="m-4 border text-base text-gray-800">
            <thead>
                <tr>
                    <th class="px-2 py-1 border">ID</th>
                    <th class="px-2 py-1 border">任务组</th>
                    <th class="px-2 py-1 border">任务</th>
                    <th class="px-2 py-1 border">用时</th>
                    <th class="px-2 py-1 border" />
                </tr>
            </thead>
            <tbody>
                {#each Runs as task}
                    <tr>
                        <td class="px-2 py-1 border">{task.Id}</td>
                        <td class="px-2 py-1 border">{task.Job}</td>
                        <td class="px-2 py-1 border">{JSON.stringify(task.Task)}</td>
                        <td class="px-2 py-1 border">{useTime(task)}s</td>
                        <td class="px-2 py-1 border">
                            <button on:dblclick={() => taskCancel(task)}>中止任务</button>
                        </td>
                    </tr>
                {:else}
                    <tr>
                        <td colspan="7" class="px-2 py-1 border text-center">empty</td>
                    </tr>
                {/each}
            </tbody>
        </table>
    </div>
</Layout>
