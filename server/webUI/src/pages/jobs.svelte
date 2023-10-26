<script>
    import Layout from "./lib/layout.svelte";
    import Dialog from "./lib/dialog.svelte";
    import { onMount } from "svelte";
    import { mkUrl, sendJson } from "./lib/base";

    onMount(() => {
        showStatus();
    });

    let editJob;
    let isShow = false;
    let Tasks = [];
    let Groups = [];

    function getGroup(gid) {
        for (let g of Groups) {
            if (g.Id == gid) {
                return g;
            }
        }
        return {};
    }

    async function showStatus() {
        let json = await sendJson(mkUrl("api/task/status"));

        let res = json.Data;

        res.Tasks.sort((a, b) => b.Name.localeCompare(a.Name));

        Groups = res.Groups;
        Tasks = res.Tasks;
    }

    function config(j) {
        editJob = j;
        isShow = !isShow;
    }

    async function save() {
        let Priority = parseInt(editJob.Priority);
        let Parallel = parseInt(editJob.Parallel);

        let json = await sendJson(mkUrl("api/job/setConfig"), {
            Name: editJob.Name,
            Priority: Priority,
            Parallel: Parallel,
        });

        isShow = !isShow;

        await showStatus();
    }

    async function jobEmpty(job) {
        var ok = confirm("Empty Job?\r\nName: " + job.Name);
        if (ok) {
            await sendJson(mkUrl("api/job/empty"), {
                name: job.Name,
            });
            await showStatus();
        }
    }

    async function jobDelIdle(job) {
        var ok = confirm("Del Idle Job?\r\nName: " + job.Name);
        if (ok) {
            await sendJson(mkUrl("api/job/delIdle"), {
                name: job.Name,
            });
            await showStatus();
        }
    }
</script>

<Layout tab="7">
    <div id="jobs">
        <table class="m-4 border text-base text-gray-800">
            <thead>
                <tr>
                    <th class="px-2 py-1 border">名称</th>
                    <th class="px-2 py-1 border">工作组</th>
                    <th class="px-2 py-1 border">WaitNum</th>
                    <th class="px-2 py-1 border">Parallel</th>
                    <th class="px-2 py-1 border">Priority</th>
                    <th class="px-2 py-1 border" />
                    <th class="px-2 py-1 border" />
                    <th class="px-2 py-1 border" />
                </tr>
            </thead>
            <tbody>
                {#each Tasks as j}
                    <tr>
                        <td class="px-2 py-1 border">{j.Name}</td>
                        <td class="px-2 py-1 border"
                            >{j.GroupId +
                                ": " +
                                getGroup(j.GroupId).Note }</td
                        >
                        <td class="px-2 py-1 border">{j.WaitNum}</td>
                        <td class="px-2 py-1 border">{j.Parallel}</td>
                        <td class="px-2 py-1 border">{j.Priority}</td>
                        <td class="px-2 py-1 border"
                            ><button on:click={() => config(j)}>Config</button
                            ></td
                        >
                        <td class="px-2 py-1 border"
                            ><button on:click={() => jobEmpty(j)}>Empty</button
                            ></td
                        >
                        <td class="px-2 py-1 border"
                            ><button on:click={() => jobDelIdle(j)}>Del</button
                            ></td
                        >
                    </tr>
                {:else}
                    <tr
                        ><td colspan="8" class="px-2 py-1 border text-center"
                            >empty</td
                        >
                    </tr>
                {/each}
            </tbody>
        </table>
    </div>
</Layout>

<Dialog bind:isShow>
    <div class="bg-white rounded-lg p-4">
        <div>job: {editJob.Name}</div>
        <br />
        <div class="grid gap-4">
            <label for="parallel">Parallel: </label>
            <input class="border" id="parallel" bind:value={editJob.Parallel} />
            <label for="priority">Priority: </label>
            <input class="border" id="priority" bind:value={editJob.Priority} />
        </div>
        <div class="text-center">
            <button class="m-4" type="button" on:click={save}>确定</button>
            <button
                class="m-4"
                type="button"
                on:click={() => (isShow = !isShow)}>取消</button
            >
        </div>
    </div>
</Dialog>
