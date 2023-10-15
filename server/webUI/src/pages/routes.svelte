<script>
    import Layout from "./lib/layout.svelte";
    import Dialog from "./lib/dialog.svelte";
    import InputKV from "./lib/inputkv.svelte";
    import { onMount } from "svelte";
    import { sendPost, sendJson, mkUrl } from "./lib/base";

    onMount(() => {
        showStatus();
    });

    let Routes = [];
    let Groups = {};
    let GroupsKv = {};
    let editRoute = {};
    let isShow = false;
    let addGroupId = 0;

    function get(id) {
        return GroupsKv[id] || {};
    }

    async function showStatus() {
        let json = await fetch(mkUrl("api/group/status")).then((t) => t.json());

        let tmp = {};
        json.Data.forEach((v) => (tmp[v.Id] = v));
        GroupsKv = tmp;
        Groups = json.Data;
        addGroupId = Groups[0].Id;

        json = await fetch(mkUrl("api/routes")).then((t) => t.json());

        Routes = json.Data;
    }

    function routeDel(row) {
        var ok = confirm(`Del Group?\r\nId:${row.Id} Note: ${row.Note}`);
        if (ok) {
            sendPost(mkUrl("api/route/del"), {
                rid: row.Id,
            });

            showStatus();
        }
    }

    function routeAdd() {
        var ok = confirm(`Add Route?`);
        if (ok) {
            sendPost(mkUrl("api/route/add"));

            showStatus();
        }
    }

    function routeEdit(row) {
        editRoute = row;
        isShow = !isShow;
    }

    async function save() {
        editRoute.Note = editRoute.Note.trim();
        editRoute.Match = editRoute.Match.trim();
        editRoute.CmdBase = editRoute.CmdBase.trim();
        editRoute.CmdDir = editRoute.CmdDir.trim();
        editRoute.HttpBase = editRoute.HttpBase.trim();

        if (editRoute.Note == "") {
            alert("Note不能为空");
            return;
        }

        if (editRoute.GroupId == 0) {
            alert("执行组不能为空");
            return;
        }

        if (editRoute.CmdDir != "") {
            if (editRoute.CmdDir[0] != "/") {
                alert("CmdDir必需是绝对路径");
                return;
            }
        }

        await sendJson(mkUrl("api/route/setConfig"), editRoute);

        isShow = !isShow;

        showStatus();
    }
</script>

<Layout tab=3>
    <div id="tasks">
        <table>
            <thead>
                <tr>
                    <th>ID</th>
                    <th>备注</th>
                    <th>正则</th>
                    <th>组ID</th>
                    <th>组Note</th>
                    <th>排序</th>
                    <th>并发</th>
                    <th>模式</th>
                    <th>超时</th>
                    <th>状态</th>
                    <th />
                    <th />
                </tr>
            </thead>
            <tbody>
                {#each Routes as row}
                    <tr>
                        <td>{row.Id}</td>
                        <td>{row.Note}</td>
                        <td>{row.Match}</td>
                        <td>{row.GroupId}</td>
                        <td>{get(row.GroupId).Note}</td>
                        <td>{row.Sort}</td>
                        <td>{row.Parallel}</td>
                        <td>{row.Mode}</td>
                        <td>{row.Timeout}</td>
                        <td>{row.Used ? "Enable" : "Disable"}</td>
                        <td
                            ><button on:click={() => routeEdit(row)}
                                >编辑</button
                            ></td
                        >
                        <td
                            ><button on:click={() => routeDel(row)}>删除</button
                            ></td
                        >
                    </tr>
                {:else}
                    <tr>
                        <td colspan="13" class="center2">empty</td>
                    </tr>
                {/each}
                <tr>
                    <td colspan="11" />
                    <td><button on:click={() => routeAdd()}>添加</button></td>
                </tr>
            </tbody>
        </table>
    </div>
</Layout>

<Dialog bind:isShow>
    <div class="box">
        <div class="grid">
            <label for="note">Note: </label>
            <input id="note" bind:value={editRoute.Note} />

            <label for="match">Match: </label>
            <input id="match" bind:value={editRoute.Match} />

            <label for="sort">Sort: </label>
            <input type="number" id="sort" bind:value={editRoute.Sort} />

            <label for="groups">执行组: </label>
            <select bind:value={editRoute.GroupId}>
                {#each Groups as group}
                    <option value={group.Id}>{group.Note}</option>
                {/each}
            </select>

            <label for="Priority">权重系数: </label>
            <input
                type="number"
                id="Priority"
                bind:value={editRoute.Priority}
            />

            <label for="Parallel">默认并发数: </label>
            <input
                type="number"
                id="Parallel"
                bind:value={editRoute.Parallel}
            />

            <label for="Timeout">最大执行时间: </label>
            <input type="number" id="Timeout" bind:value={editRoute.Timeout} />

            <label for="Mode">模式: </label>
            <select id="Mode" bind:value={editRoute.Mode}>
                <option value={1}>HTTP</option>
                <option value={2}>Cli</option>
            </select>

            {#if editRoute.Mode == 2}
                <label for="CmdBase">CmdBase: </label>
                <input id="CmdBase" bind:value={editRoute.CmdBase} />

                <label for="CmdDir">CmdDir: </label>
                <input id="CmdDir" bind:value={editRoute.CmdDir} />

                <label for="CmdEnv">CmdEnv: </label>
                <InputKV bind:kv={editRoute.CmdEnv} />
            {:else}
                <label for="HttpBase">HttpBase: </label>
                <input id="HttpBase" bind:value={editRoute.HttpBase} />

                <label for="HttpHeader">Header: </label>
                <InputKV bind:kv={editRoute.HttpHeader} />
            {/if}

            <div class="status">
                <div>
                    <input
                        type="radio"
                        id="Used1"
                        value={true}
                        bind:group={editRoute.Used}
                    />
                    <label for="Used1">Enable</label>
                </div>

                <div>
                    <input
                        type="radio"
                        id="Used0"
                        value={false}
                        bind:group={editRoute.Used}
                    />
                    <label for="Used0">Disable</label>
                </div>
            </div>
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
        grid-template-columns: auto auto;
    }
    .status {
        grid-column: 1/3;
        display: flex;
        justify-content: center;
        gap: 20px;
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

    .editGroups {
        display: grid;
        grid-template-columns: auto auto;
    }
</style>
