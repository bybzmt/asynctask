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
        <table>
            <thead>
                <tr>
                    <th>ID</th>
                    <th>工作组</th>
                    <th class="name">任务组</th>
                    <th>模式</th>
                    <th class="params">任务</th>
                    <th class="time">用时</th>
                    <th />
                </tr>
            </thead>
            <tbody>
                {#each Runs as task}
                    <tr>
                        <td>{task.Id}</td>
                        <td>{task.Group} ({getGroup(task.Group).Note})</td>
                        <td>{task.Name}</td>
                        <td>{task.Mode}</td>
                        <td class="params">{task.Task}</td>
                        <td>{task.UseTime / 1000}s</td>
                        <td>
                            <button
                                class="cancel"
                                on:dblclick={() => taskCancel(task)}
                                >中止任务</button
                            >
                        </td>
                    </tr>
                {:else}
                    <tr>
                        <td colspan="7" class="center">empty</td>
                    </tr>
                {/each}
            </tbody>
        </table>
    </div>
</Layout>

<style>
    table {
        margin: 1em;
        border-collapse: collapse;
    }
    table td,
    table th {
        border: 1px solid #777;
        padding: 0px 1em;
    }
    .center {
        text-align: center;
    }
    .params {
        max-width: 500px;
        overflow: hidden;
    }

    .cancel {
        font-size: 12px;
    }
</style>
