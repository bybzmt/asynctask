<script>
    import Layout from "./lib/layout.svelte";
    import { onMount, onDestroy } from "svelte";
    import { mkUrl, taskCancel } from "./lib/base";

    let timer;
    onMount(() => {
        getGroups();
        showStatus();

        timer = setInterval(function () {
            getGroups();
            showStatus();
        }, 1000);
    });

    onDestroy(() => {
        clearInterval(timer);
    });

    let Groups = [];
    let Runs = [];

    async function getGroups() {
        let json = await fetch(mkUrl("api/group/status")).then((t) => t.json());

        Groups = json.Data;
    }

    function getGroup(gid) {
        for (let g of Groups) {
            if (g.Id == gid) {
                return g;
            }
        }
        return {};
    }

    async function showStatus() {
        let json = await fetch(mkUrl("api/task/runing")).then((t) => t.json());

        Runs = json.Data.sort((a, b) => a.Id > b.Id);
    }
</script>

<Layout tab=1>
    <div id="tasks">
        <table class="m-4 border text-base text-gray-800">
            <thead>
                <tr>
                    <th class="px-2 py-1 border">ID</th>
                    <th class="px-2 py-1 border">工作组</th>
                    <th class="px-2 py-1 border">任务组</th>
                    <th class="px-2 py-1 border">模式</th>
                    <th class="px-2 py-1 border">任务</th>
                    <th class="px-2 py-1 border">用时</th>
                    <th />
                </tr>
            </thead>
            <tbody>
                {#each Runs as task}
                    <tr>
                        <td class="px-2 py-1 border">{task.Id}</td>
                        <td class="px-2 py-1 border">{task.Group} ({getGroup(task.Group).Note})</td>
                        <td class="px-2 py-1 border">{task.Name}</td>
                        <td class="px-2 py-1 border">{task.Mode}</td>
                        <td class="px-2 py-1 border">{task.Task}</td>
                        <td class="px-2 py-1 border">{task.UseTime / 1000}s</td>
                        <td class="px-2 py-1 border">
                            <button
                                on:dblclick={() => taskCancel(task)}
                                >中止任务</button
                            >
                        </td>
                    </tr>
                {:else}
                    <tr>
                        <td colspan="7"  class="px-2 py-1 border text-center">empty</td>
                    </tr>
                {/each}
            </tbody>
        </table>
    </div>
</Layout>

