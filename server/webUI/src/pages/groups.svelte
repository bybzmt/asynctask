<script>
    import Layout from "./lib/layout.svelte";
    import Dialog from "./lib/dialog.svelte";
    import { onMount } from "svelte";
    import { mkUrl, sendJson, sendPost } from "./lib/base";

    onMount(() => {
        showStatus();
    });

    let Groups = [];
    let editGroup = {};
    let isShow = false;

    async function showStatus() {
        let json = await fetch(mkUrl("api/group/status")).then((t) => t.json());

        Groups = json.Data;
    }

    function groupDel(row) {
        var ok = confirm(`Del Group?\r\nId:${row.Id} Note: ${row.Note}`);
        if (ok) {
            sendPost(mkUrl("api/group/del"), {
                gid: row.Id,
            });

            showStatus();
        }
    }

    function groupAdd() {
        var ok = confirm(`Add Group?`);
        if (ok) {
            sendPost(mkUrl("api/group/add"));

            showStatus();
        }
    }

    function edit(group) {
        editGroup = group;
        isShow = !isShow;
    }

    async function save() {
        await sendJson(mkUrl("api/group/setConfig"), editGroup);

        isShow = !isShow;

        showStatus();
    }
</script>

<Layout tab=4>
    <div id="tasks">
        <table>
            <thead>
                <tr>
                    <th>ID</th>
                    <th>备注</th>
                    <th>并行数</th>
                    <th />
                    <th />
                </tr>
            </thead>
            <tbody>
                {#each Groups as group}
                    <tr>
                        <td>{group.Id}</td>
                        <td>{group.Note}</td>
                        <td>{group.WorkerNum}</td>
                        <td
                            ><button on:click={() => edit(group)}>编辑</button
                            ></td
                        >
                        <td
                            ><button on:click={() => groupDel(group)}
                                >删除</button
                            ></td
                        >
                    </tr>
                {:else}
                    <tr>
                        <td colspan="5" class="center2">empty</td>
                    </tr>
                {/each}
                <tr>
                    <td colspan="4" />
                    <td><button on:click={() => groupAdd()}>添加</button></td>
                </tr>
            </tbody>
        </table>
    </div>
</Layout>

<Dialog bind:isShow>
    <div class="box">
        <div class="grid">
            <label for="note">Note: </label>
            <input id="note" bind:value={editGroup.Note} />
            <label for="workerNum">WorkerNum: </label>
            <input id="workerNum" bind:value={editGroup.WorkerNum} />
        </div>
        <div class="center">
            <button type="button" on:click={save}>确定</button>
            <button type="button" on:click={() => (isShow = !isShow)}
                >取消</button
            >
        </div>
    </div>
</Dialog>

<style>
    table {
        margin: 1em;
        border-collapse: collapse;
        border: 1px solid #777;
    }
    table td,
    table th {
        border: 1px solid #777;
        padding: 0px 1em;
    }
    .center2 {
        text-align: center;
    }
    #tab {
        margin: 0 0.5em;
    }
    .box {
        padding: 10px;
        background: #fff;
        border-radius: 10px;
    }
    .grid {
        display: grid;
        gap: 10px;
    }
    .box input {
        border: 1px solid #777;
    }
    .center {
        display: flex;
        justify-content: center;
    }
    .center button {
        margin: 10px;
    }
</style>
