<script>
    import Layout from "./lib/layout.svelte"
    import Dialog from "./lib/dialog.svelte"
    import InputKV from "./lib/inputkv.svelte"
    import InputLine from "./lib/inputline.svelte"
    import { onMount } from "svelte"
    import { sendJson, mkUrl } from "./lib/base"

    onMount(() => {
        showStatus()
    })

    let Routes = []
    let editRow = {}
    let isShow = false

    async function showStatus() {
        let resp = await sendJson(mkUrl("api/taskrule/list"))

        let t = resp.Data || []

        t.sort((a, b) => (a.Sort < b.Sort ? -1 : 1))

        Routes = t
    }

    async function routeDel(row) {
        var ok = confirm(`Del?\r\nPattern:${row.Pattern} Note: ${row.Note}`)
        if (ok) {
            await sendJson(mkUrl("api/taskrule/del"), {
                Type: row.Type,
                Pattern: row.Pattern,
            })

            await showStatus()
        }
    }

    async function routeAdd() {
        editRow = {}
        editRow.Mode = 1
        editRow.Type = 1
        editRow.Note = editRow.Note || ""
        editRow.Sort = editRow.Sort || 0
        editRow.CmdPath = editRow.CmdPath || ""
        editRow.CmdDir = editRow.CmdDir || ""
        editRow.CmdArgs = editRow.CmdArgs || []
        editRow.Timeout = editRow.Timeout || 0
        editRow.Used = false
        isShow = !isShow
    }

    function routeEdit(row) {
        editRow = row
        editRow.Sort = editRow.Sort || 0
        editRow.CmdPath = editRow.CmdPath || ""
        editRow.CmdDir = editRow.CmdDir || ""
        editRow.CmdArgs = editRow.CmdArgs || []
        editRow.Timeout = editRow.Timeout || 0

        if (/cli:\/\//i.test(row.Pattern)) {
            editRow.Mode = 2
        } else {
            editRow.Mode = 1
        }

        isShow = !isShow
    }

    async function save() {
        editRow.Note = editRow.Note.trim()
        editRow.Pattern = editRow.Pattern.trim()
        editRow.CmdPath = editRow.CmdPath.trim()
        editRow.CmdDir = editRow.CmdDir.trim()

        if (editRow.CmdDir != "") {
            if (editRow.CmdDir[0] != "/") {
                alert("CmdDir必需是绝对路径")
                return
            }
        }

        let resp = await sendJson(mkUrl("api/taskrule/put"), editRow)

        if (resp.Code != 0) {
            alert(JSON.stringify(resp))
            return
        }

        isShow = !isShow

        await showStatus()
    }
</script>

<Layout tab="4">
    <div id="tasks">
        <table class="m-4 border text-base text-gray-800">
            <thead>
                <tr>
                    <th class="px-2 py-1 border">类型</th>
                    <th class="px-2 py-1 border">匹配</th>
                    <th class="px-2 py-1 border">备注</th>
                    <th class="px-2 py-1 border">排序</th>
                    <th class="px-2 py-1 border">超时</th>
                    <th class="px-2 py-1 border">状态</th>
                    <th class="px-2 py-1 border" colspan="2" />
                </tr>
            </thead>
            <tbody>
                {#each Routes as row}
                    {#if row.Type != 1}
                        <tr>
                            <td class="px-2 py-1 border">{row.Type == 1 ? "regexp" : "direct"}</td>
                            <td class="px-2 py-1 border">{row.Pattern}</td>
                            <td class="px-2 py-1 border">{row.Note}</td>
                            <td class="px-2 py-1 border text-center">{row.Sort}</td>
                            <td class="px-2 py-1 border text-center">{row.Timeout}</td>
                            <td class="px-2 py-1 border">{row.Used ? "Enable" : "Disable"}</td>
                            <td class="px-2 py-1 border"><button on:click={() => routeEdit(row)}>编辑</button></td>
                            <td class="px-2 py-1 border"><button on:click={() => routeDel(row)}>删除</button></td>
                        </tr>
                    {/if}
                {:else}
                    <tr>
                        <td colspan="8" class="px-2 py-1 border text-center">empty</td>
                    </tr>
                {/each}
                <tr>
                    <td class="px-2 py-1 text-center"><button on:click={() => routeAdd()}>添加</button></td>
                    <td colspan="7" />
                </tr>
            </tbody>
        </table>

        <table class="m-4 border text-base text-gray-800">
            <thead>
                <tr>
                    <th class="px-2 py-1 border">类型</th>
                    <th class="px-2 py-1 border">匹配</th>
                    <th class="px-2 py-1 border">备注</th>
                    <th class="px-2 py-1 border">排序</th>
                    <th class="px-2 py-1 border">超时</th>
                    <th class="px-2 py-1 border">状态</th>
                    <th class="px-2 py-1 border" colspan="2" />
                </tr>
            </thead>
            <tbody>
                {#each Routes as row}
                    {#if row.Type == 1}
                        <tr>
                            <td class="px-2 py-1 border">{row.Type == 1 ? "regexp" : "direct"}</td>
                            <td class="px-2 py-1 border">{row.Pattern}</td>
                            <td class="px-2 py-1 border">{row.Note}</td>
                            <td class="px-2 py-1 border text-center">{row.Sort}</td>
                            <td class="px-2 py-1 border text-center">{row.Timeout}</td>
                            <td class="px-2 py-1 border">{row.Used ? "Enable" : "Disable"}</td>
                            <td class="px-2 py-1 border"><button on:click={() => routeEdit(row)}>编辑</button></td>
                            <td class="px-2 py-1 border"><button on:click={() => routeDel(row)}>删除</button></td>
                        </tr>
                    {/if}
                {:else}
                    <tr>
                        <td colspan="8" class="px-2 py-1 border text-center">empty</td>
                    </tr>
                {/each}
                <tr>
                    <td class="px-2 py-1 text-center"><button on:click={() => routeAdd()}>添加</button></td>
                    <td colspan="7" />
                </tr>
            </tbody>
        </table>
    </div>
</Layout>

<Dialog bind:isShow>
    <div class="p-4 bg-white rounded-lg text-gray-800">
        <div class="grid grid-cols-[auto_auto] gap-2">
            <label for="note">Note: </label>
            <input class="border" id="note" bind:value={editRow.Note} />

            <span>Type: </span>
            <div>
                <label class="mr-4"
                    ><input class="mr-2" type="radio" value={0} bind:group={editRow.Type} />direct</label>
                <label class="mr-4"
                    ><input class="mr-2" type="radio" value={1} bind:group={editRow.Type} />regexp</label>
            </div>
            {#if editRow.Type == 1}
                <label for="sort">Sort: </label>
                <input class="border" type="number" id="sort" bind:value={editRow.Sort} />
            {/if}

            <label for="match">Pattern: </label>
            <input
                class="border"
                id="match"
                bind:value={editRow.Pattern}
                placeholder={editRow.Mode == 2 ? "^cli://xxx" : "^https?://xxx"} />

            <div>Rewrite:</div>

            <div class="flex">
                <input class="border w-full" bind:value={editRow.RewriteReg} />
                <div class="mx-2">=&gt;</div>
                <input class="border w-full" bind:value={editRow.RewriteRepl} />
            </div>

            <label for="Timeout">Timeout: </label>
            <input class="border" type="number" id="Timeout" bind:value={editRow.Timeout} />

            <span>模式: </span>
            <div>
                <label class="mr-4"><input class="mr-2" type="radio" value={1} bind:group={editRow.Mode} />HTTP</label>
                <label class="mr-4"><input class="mr-2" type="radio" value={2} bind:group={editRow.Mode} />CLI</label>
            </div>

            {#if editRow.Mode == 2}
                <label for="CmdPath">CmdPath: </label>
                <input class="border" id="CmdPath" bind:value={editRow.CmdPath} />

                <label for="CmdArgs">CmdArgs: </label>
                <div>
                    <InputLine bind:args={editRow.CmdArgs} />
                    <div class="text-gray-400">Next Arg: Original Cmd</div>
                    <div class="text-gray-400">Next Args: Original Args</div>
                </div>

                <label for="CmdDir">CmdDir: </label>
                <input class="border" id="CmdDir" bind:value={editRow.CmdDir} />

                <label for="CmdEnv">CmdEnv: </label>
                <InputKV bind:kv={editRow.CmdEnv} />
            {:else}
                <label for="HttpHeader">Header: </label>
                <InputKV bind:kv={editRow.HttpHeader} />
            {/if}

            <div class="grid grid-cols-2 col-start-2">
                <div>
                    <input type="radio" id="Used1" value={true} bind:group={editRow.Used} />
                    <label for="Used1">Enable</label>
                </div>

                <div>
                    <input type="radio" id="Used0" value={false} bind:group={editRow.Used} />
                    <label for="Used0">Disable</label>
                </div>
            </div>
        </div>
        <div class="text-center mt-2">
            <button class="mx-4" type="button" on:click={save}>确定</button>
            <button class="mx-4" type="button" on:click={() => (isShow = !isShow)}>取消</button>
        </div>
    </div>
</Dialog>
