<script>
    import Layout from "./lib/layout.svelte";
    import { onMount } from "svelte";
    import { mkUrl, sendJson, sendPost } from "./lib/base";

    onMount(() => {
        showStatus();
    });

    let Groups = [];

    async function showStatus() {
        let json = await fetch(mkUrl("api/groups")).then((t) => t.json());

        Groups = json.Data;
    }

    function groupNote(row) {
        var txt = prompt("Group: " + row.Id + " Note: ", row.Note);
        if (txt != null && txt != "") {
            sendJson(mkUrl("api/group/setConfig", { gid: row.Id }), {
                Note: txt,
                WorkerNum: row.WorkerNum,
            });
        }
    }

    function groupWorkerNum(row) {
        var txt = prompt("Group: " + row.Id + " WorkerNum: ", row.WorkerNum);
        if (txt != null && txt != "") {
            let workerNum = parseInt(txt);

            sendJson(mkUrl("api/group/setConfig", { gid: row.Id }), {
                Note: row.Note,
                WorkerNum: workerNum,
            });
        }
    }

    function groupDel(row) {
        var ok = confirm(`Del Group?\r\nId:${row.Id} Note: ${row.Note}`);
        if (ok) {
            sendPost(mkUrl("api/group/del"), {
                gid: row.Id,
            });

            showStatus()
        }
    }

    function groupAdd(row) {
        var ok = confirm(`Add Group?`);
        if (ok) {
            sendPost(mkUrl("api/group/add"));

            showStatus()
        }
    }
</script>

<Layout>
    <div id="tab">
        <a href="#/">Tasks</a>
        <a href="#/routes">Routes</a>
        <a href="#/groups">WorkGroups</a>
    </div>

    <div id="tasks">
        <table>
            <thead>
                <tr>
                    <th>ID</th>
                    <th>备注</th>
                    <th>并行数</th>
                    <th />
                </tr>
            </thead>
            <tbody>
                {#each Groups as group}
                    <tr>
                        <td>{group.Id}</td>
                        <td on:click={() => groupNote(group)}>{group.Note}</td>
                        <td on:click={() => groupWorkerNum(group)}
                            >{group.WorkerNum}</td
                        >
                        <td on:click={() => groupDel(group)}>删除</td>
                    </tr>
                {:else}
                    <tr>
                        <td colspan="4" class="center">empty</td>
                    </tr>
                {/each}
                <tr>
                    <td colspan="3" />
                    <td on:click={() => groupAdd()}>添加</td>
                </tr>
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
    #tab {
        margin: 0 0.5em;
    }
</style>
