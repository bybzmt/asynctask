<script>
    import Layout from "./lib/layout.svelte";
    import Dialog from "./lib/dialog.svelte";
    import InputKV from "./lib/inputkv.svelte";
    import { onMount } from "svelte";
    import { sendJson, mkUrl } from "./lib/base";

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
            sendJson(mkUrl("api/route/del"), {
                rid: row.Id,
            });

            showStatus();
        }
    }

    function routeAdd() {
        var ok = confirm(`Add Route?`);
        if (ok) {
            sendJson(mkUrl("api/route/add"));

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

<Layout tab="3">
    <div id="tasks">
        <table class="m-4 border text-base text-gray-800">
            <thead>
                <tr>
                    <th class="px-2 py-1 border">ID</th>
                    <th class="px-2 py-1 border">备注</th>
                    <th class="px-2 py-1 border">正则</th>
                    <th class="px-2 py-1 border">组</th>
                    <th class="px-2 py-1 border">排序</th>
                    <th class="px-2 py-1 border">并发</th>
                    <th class="px-2 py-1 border">模式</th>
                    <th class="px-2 py-1 border">超时</th>
                    <th class="px-2 py-1 border">状态</th>
                    <th class="px-2 py-1 border" colspan="2" />
                </tr>
            </thead>
            <tbody>
                {#each Routes as row}
                    <tr>
                        <td class="px-2 py-1 border text-center">{row.Id}</td>
                        <td class="px-2 py-1 border">{row.Note}</td>
                        <td class="px-2 py-1 border">{row.Match}</td>
                        <td class="px-2 py-1 border">{row.GroupId}: {get(row.GroupId).Note}</td>
                        <td class="px-2 py-1 border text-center">{row.Sort}</td>
                        <td class="px-2 py-1 border text-center">{row.Parallel}</td>
                        <td class="px-2 py-1 border">{row.Mode ? "HTTP" : "CLI"}</td>
                        <td class="px-2 py-1 border text-center">{row.Timeout}</td>
                        <td class="px-2 py-1 border">{row.Used ? "Enable" : "Disable"}</td>
                        <td class="px-2 py-1 border"
                            ><button on:click={() => routeEdit(row)}
                                >编辑</button
                            ></td
                        >
                        <td class="px-2 py-1 border"
                            ><button on:click={() => routeDel(row)}>删除</button
                            ></td
                        >
                    </tr>
                {:else}
                    <tr>
                        <td colspan="11" class="px-2 py-1 border text-center">empty</td>
                    </tr>
                {/each}
                <tr>
                    <td class="px-2 py-1 text-center"
                        ><button on:click={() => routeAdd()}>添加</button></td
                    >
                    <td colspan="10" />
                </tr>
            </tbody>
        </table>
    </div>
</Layout>

<Dialog bind:isShow>
    <div class="p-4 bg-white rounded-lg text-gray-800">
        <div class="grid grid-cols-[auto_auto] gap-2">
            <label for="note">Note: </label>
            <input class="border" id="note" bind:value={editRoute.Note} />

            <label for="match">Match: </label>
            <input class="border" id="match" bind:value={editRoute.Match} />

            <label for="sort">Sort: </label>
            <input class="border" type="number" id="sort" bind:value={editRoute.Sort} />

            <label for="groups">执行组: </label>
            <select class="border" bind:value={editRoute.GroupId}>
                {#each Groups as group}
                    <option value={group.Id}>{group.Note}</option>
                {/each}
            </select>

            <label for="Priority">权重系数: </label>
            <input class="border"
                type="number"
                id="Priority"
                bind:value={editRoute.Priority}
            />

            <label for="Parallel">默认并发数: </label>
            <input class="border"
                type="number"
                id="Parallel"
                bind:value={editRoute.Parallel}
            />

            <label for="Timeout">最大执行时间: </label>
            <input class="border" type="number" id="Timeout" bind:value={editRoute.Timeout} />

            <label for="Mode">模式: </label>
            <select class="border" id="Mode" bind:value={editRoute.Mode}>
                <option value={1}>HTTP</option>
                <option value={2}>Cli</option>
            </select>

            {#if editRoute.Mode == 2}
                <label for="CmdBase">CmdBase: </label>
                <input class="border" id="CmdBase" bind:value={editRoute.CmdBase} />

                <label for="CmdDir">CmdDir: </label>
                <input class="border" id="CmdDir" bind:value={editRoute.CmdDir} />

                <label for="CmdEnv">CmdEnv: </label>
                <InputKV bind:kv={editRoute.CmdEnv} />
            {:else}
                <label for="HttpBase">HttpBase: </label>
                <input class="border" id="HttpBase" bind:value={editRoute.HttpBase} />

                <label for="HttpHeader">Header: </label>
                <InputKV bind:kv={editRoute.HttpHeader} />
            {/if}

            <div class="grid grid-cols-2 col-start-2">
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
        <div class="text-center mt-2">
            <button class="mx-4" type="button" on:click={save}>确定</button>
            <button class="mx-4" type="button" on:click={() => (isShow = !isShow)}
                >取消</button
            >
        </div>
    </div>
</Dialog>
