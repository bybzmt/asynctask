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
    let Groups = [];
    let editRoute = {};
    let isShow = false;

    function get(id) {
        for (let g of Groups) {
            if (g.Id == id) {
                return g;
            }
        }
        return {};
    }

    async function loadGroups() {
        let resp = await sendJson(mkUrl("api/group/list"));
        Groups = resp.Data || [];
    }

    async function showStatus() {
        await loadGroups();

        let resp = await sendJson(mkUrl("api/rule/list"));

        let t = resp.Data || [];

        t.sort((a, b) => (a.Sort < b.Sort ? -1 : 1));

        Routes = t;
    }

    async function routeDel(row) {
        var ok = confirm(`Del?\r\nPattern:${row.Pattern} Note: ${row.Note}`);
        if (ok) {
            await sendJson(mkUrl("api/rule/del"), {
                Type: row.Type,
                Pattern: row.Pattern,
            });

            await showStatus();
        }
    }

    async function routeAdd() {
        editRoute = {};
        isShow = !isShow;
    }

    function routeEdit(row) {
        editRoute = row;
        isShow = !isShow;
    }

    async function save() {
        editRoute.Note = (editRoute.Note || "").trim();
        editRoute.Match = (editRoute.Match || "").trim();
        editRoute.CmdBase = (editRoute.CmdBase || "").trim();
        editRoute.CmdDir = (editRoute.CmdDir || "").trim();
        editRoute.HttpBase = (editRoute.HttpBase || "").trim();

        if (editRoute.CmdDir != "") {
            if (editRoute.CmdDir[0] != "/") {
                alert("CmdDir必需是绝对路径");
                return;
            }
        }

        let resp = await sendJson(mkUrl("api/rule/put"), editRoute);

        if (resp.Code != 0) {
            alert(JSON.stringify(resp));
            return;
        }

        isShow = !isShow;

        await showStatus();
    }
</script>

<Layout tab="8">
    <div id="tasks">
        <table class="m-4 border text-base text-gray-800">
            <thead>
                <tr>
                    <th class="px-2 py-1 border">备注</th>
                    <th class="px-2 py-1 border">类型</th>
                    <th class="px-2 py-1 border">匹配</th>
                    <th class="px-2 py-1 border">执行组</th>
                    <th class="px-2 py-1 border">排序</th>
                    <th class="px-2 py-1 border">并发</th>
                    <th class="px-2 py-1 border">超时</th>
                    <th class="px-2 py-1 border">状态</th>
                    <th class="px-2 py-1 border" colspan="2" />
                </tr>
            </thead>
            <tbody>
                {#each Routes as row}
                    <tr>
                        <td class="px-2 py-1 border">{row.Note}</td>
                        <td class="px-2 py-1 border"
                            >{row.Type == 1 ? "regexp" : "direct"}</td
                        >
                        <td class="px-2 py-1 border">{row.Pattern}</td>
                        <td class="px-2 py-1 border"
                            >{row.GroupId}: {get(row.GroupId).Note}</td
                        >
                        <td class="px-2 py-1 border text-center">{row.Sort}</td>
                        <td class="px-2 py-1 border text-center"
                            >{row.Parallel}</td
                        >
                        <td class="px-2 py-1 border text-center"
                            >{row.Timeout}</td
                        >
                        <td class="px-2 py-1 border"
                            >{row.Used ? "Enable" : "Disable"}</td
                        >
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
                        <td colspan="11" class="px-2 py-1 border text-center"
                            >empty</td
                        >
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

            <span>Type: </span>
            <div>
                <label>
                    <input type="radio" value={0} bind:group={editRoute.Type} />
                    direct</label
                >
                <label>
                    <input type="radio" value={1} bind:group={editRoute.Type} />
                    regexp</label
                >
            </div>

            <label for="match">Pattern: </label>
            <input class="border" id="match" bind:value={editRoute.Pattern} />

            {#if editRoute.Type == 1}
                <label for="sort">Sort: </label>
                <input
                    class="border"
                    type="number"
                    id="sort"
                    bind:value={editRoute.Sort}
                />
            {/if}

            <label for="groups">执行组: </label>
            <select class="border" bind:value={editRoute.GroupId}>
                {#each Groups as group}
                    <option value={group.Id}>{group.Note}</option>
                {/each}
            </select>

            <label for="Priority">权重系数: </label>
            <input
                class="border"
                type="number"
                id="Priority"
                bind:value={editRoute.Priority}
            />

            <label for="Parallel">默认并发数: </label>
            <input
                class="border"
                type="number"
                id="Parallel"
                bind:value={editRoute.Parallel}
            />

            <label for="Timeout">最大执行时间: </label>
            <input
                class="border"
                type="number"
                id="Timeout"
                bind:value={editRoute.Timeout}
            />

            <span>模式: </span>
            <div>
                <label>
                    <input type="radio" value={1} bind:group={editRoute.Mode} />
                    HTTP</label
                >
                <label>
                    <input type="radio" value={2} bind:group={editRoute.Mode} />
                    CLI</label
                >
            </div>

            {#if editRoute.Mode == 2}
                <label for="CmdBase">CmdBase: </label>
                <input
                    class="border"
                    id="CmdBase"
                    bind:value={editRoute.CmdBase}
                />

                <label for="CmdDir">CmdDir: </label>
                <input
                    class="border"
                    id="CmdDir"
                    bind:value={editRoute.CmdDir}
                />

                <label for="CmdEnv">CmdEnv: </label>
                <InputKV bind:kv={editRoute.CmdEnv} />
            {:else}
                <label for="HttpBase">HttpBase: </label>
                <input
                    class="border"
                    id="HttpBase"
                    bind:value={editRoute.HttpBase}
                />

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
            <button
                class="mx-4"
                type="button"
                on:click={() => (isShow = !isShow)}>取消</button
            >
        </div>
    </div>
</Dialog>
