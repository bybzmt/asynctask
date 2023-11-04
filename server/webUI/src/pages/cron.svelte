<script>
    import Layout from "./lib/layout.svelte"
    import Dialog from "./lib/dialog.svelte"
    import Task from "./lib/task.svelte"
    import { onMount } from "svelte"
    import { sendJson, mkUrl, timeStr } from "./lib/base"

    onMount(() => {
        showStatus()
    })

    let cron = []
    let editRow = {}
    let isShow = false

    async function showStatus() {
        let json = await sendJson(mkUrl("api/cron/getConfig"))

        cron = json.Data
        cron.Tasks = cron.Tasks || []
    }

    async function reload() {
        let json = await sendJson(mkUrl("api/cron/reload"))
        if (json.Code != 0) {
            alert(json.Data)
            return
        }
        alert("success")
        await showStatus()
    }

    async function rowDel(row) {
        var ok = confirm(`Del Group?\r\nId:${row.Id} Note: ${row.Note}`)
        if (ok) {
            let cfgs = []
            for (let i = 0; i < cron.Tasks.length; i++) {
                if (row.Id != cron.Tasks[i].Id) {
                    cfgs.push(cron.Tasks[i])
                }
            }

            let resp = await sendJson(mkUrl("api/cron/setConfig"), cfgs)

            if (resp.Code != 0) {
                alert(resp.Data)
                return
            }

            await showStatus()
        }
    }

    function rowAdd() {
        editRow = {
            Minute: "*",
            Hour: "*",
            Day: "*",
            Month: "*",
            Week: "*",
            Note: "",
            Task: "",
        }
        isShow = true
    }

    function rowEdit(row) {
        let t = {}
        t.Id = row.Id || 0
        t.Note = row.Note
        t.Task = JSON.stringify(row.Task)

        let p = row.Cfg.trim().split(" ")
        t.Minute = p.at(0) || "*"
        t.Hour = p.at(1) || "*"
        t.Day = p.at(2) || "*"
        t.Month = p.at(3) || "*"
        t.Week = p.at(4) || "*"

        editRow = t
        isShow = !isShow
    }

    async function save() {
        let row = {}
        row.Id = editRow.Id || 0
        row.Note = editRow.Note

        let cfg = [editRow.Minute, editRow.Hour, editRow.Day, editRow.Month, editRow.Week]

        for (let i = 0; i < cfg.length; i++) {
            cfg[i] = cfg[i].trim() || "*"
        }

        row.Cfg = cfg.join(" ")

        try {
            row.Task = JSON.parse(editRow.Task)
        } catch (e) {
            alert("Task JSON.parse " + e.message)
            return
        }

        let maxId = 0
        let cfgs = []
        for (let i = 0; i < cron.Tasks.length; i++) {
            if (row.Id != cron.Tasks[i].Id) {
                cfgs.push(cron.Tasks[i])
            }
            if (cron.Tasks[i].Id > maxId) {
                maxId = cron.Tasks[i].Id || 0
            }
        }
        if (!(row.Id > 0)) {
            row.Id = maxId + 1
        }
        cfgs.push(row)

        let resp = await sendJson(mkUrl("api/cron/setConfig"), cfgs)

        if (resp.Code != 0) {
            alert(resp.Data)
            return
        }

        isShow = !isShow

        await showStatus()
    }
</script>

<Layout tab="7">
    <div class="m-4 grid gap-y-1 gap-x-2 grid-cols-[auto_auto_auto] w-min text-sm">
        <span>Edit:</span>
        <span>{timeStr(cron.EditAt)}</span>
        <span class="col-start-1">Run :</span>
        <span>{timeStr(cron.RunAt)}</span>
        <button class="whitespace-nowrap" on:click={reload}>reload</button>
    </div>

    <div id="tasks">
        <table class="m-4 border text-base text-gray-800">
            <thead>
                <tr>
                    <th class="px-2 py-1 border">规则</th>
                    <th class="px-2 py-1 border">备注</th>
                    <th class="px-2 py-1 border">任务</th>
                    <th colspan="2" class="px-2 py-1 border" />
                </tr>
            </thead>
            <tbody>
                {#if cron.Tasks}
                    {#each cron.Tasks as row}
                        <tr>
                            <td class="px-2 py-1 border">{row.Cfg}</td>
                            <td class="px-2 py-1 border">{row.Note}</td>
                            <td class="px-2 py-1 border">{JSON.stringify(row.Task)}</td>
                            <td class="px-2 py-1 border"><button on:click={() => rowEdit(row)}>编辑</button></td>
                            <td class="px-2 py-1 border"><button on:click={() => rowDel(row)}>删除</button></td>
                        </tr>
                    {/each}
                {:else}
                    <tr>
                        <td colspan="5" class="border text-center">empty</td>
                    </tr>
                {/if}
                <tr>
                    <td class="text-center"><button on:click={() => rowAdd()}>添加</button></td>
                    <td colspan="4" />
                </tr>
            </tbody>
        </table>
    </div>
</Layout>

<Dialog bind:isShow>
    <div class="bg-white rounded-lg p-4 w-[700px]">
        <div>
            <pre class=" text-gray-400 text-sm">
*: 匹配该字段所有值
/: 表示范围增量
,: 用来分隔同一组中的项目
-: 表示范围
例: */5 10-12 * 1,3  表示每周1周3十点到十二点每五分钟执行一次
</pre>
        </div>
        <div class="grid grid-cols-[auto_auto] gap-4 mt-4">
            <label for="row_cfg">Cfg: </label>
            <div id="row_cfg" class="grid grid-cols-5 gap-y-2 gap-x-4 text-center">
                <input class="border text-center" bind:value={editRow.Minute} />
                <input class="border text-center" bind:value={editRow.Hour} />
                <input class="border text-center" bind:value={editRow.Day} />
                <input class="border text-center" bind:value={editRow.Month} />
                <input class="border text-center" bind:value={editRow.Week} />
                <span>分</span>
                <span>时</span>
                <span>天</span>
                <span>月</span>
                <span>周</span>
            </div>

            <label for="row_note">Note: </label>
            <input class="border" id="row_note" bind:value={editRow.Note} />

            <label for="row_task">Task: </label>
            <Task bind:value={editRow.Task} type={2} />
        </div>
        <div class="text-center mt-2">
            <button class="mx-4" type="button" on:click={save}>确定</button>
            <button class="mx-4" type="button" on:click={() => (isShow = !isShow)}>取消</button>
        </div>
    </div>
</Dialog>
