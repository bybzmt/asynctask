<script>
    import Layout from "./lib/layout.svelte";
    import Dialog from "./lib/dialog.svelte";
    import { onMount } from "svelte";
    import { sendJson, mkUrl } from "./lib/base";

    onMount(() => {
        showStatus();
    });

    let cron = [];
    let editRow = {};
    let isShow = false;

    async function showStatus() {
        let json = await fetch(mkUrl("api/cron/getConfig")).then((t) =>
            t.json()
        );

        cron = json.Data;
    }

    async function rowDel(row) {
        var ok = confirm(`Del Group?\r\nId:${row.Id} Note: ${row.Note}`);
        if (ok) {
            let cfgs = [];
            for (let i = 0; i < cron.length; i++) {
                if (row.Id != cron[i].Id) {
                    cfgs.push(cron[i]);
                }
            }

            let resp = await sendJson(mkUrl("api/cron/setConfig"), cfgs);

            if (resp.Code != 0) {
                alert(resp.Data);
                return;
            }

            showStatus();
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
            Task: `{
    "url": "http://g.com",
    "form": {"k":"v"}

    "cmd": "echo",
    "args": ["hellworld"]
}`,
        };
        isShow = true;
    }

    function rowEdit(row) {
        let t = {};
        t.Id = row.Id || 0;
        t.Node = row.Note;
        t.Task = JSON.stringify(row.Task);

        let p = row.Cfg.trim().split(" ");
        t.Minute = p.at(0) || "*";
        t.Hour = p.at(1) || "*";
        t.Day = p.at(2) || "*";
        t.Month = p.at(3) || "*";
        t.Week = p.at(4) || "*";

        editRow = t;
        isShow = !isShow;
    }

    async function save() {
        let row = {};
        row.Id = row.Id;
        row.Node = editRow.Note;

        let cfg = [
            editRow.Minute,
            editRow.Hour,
            editRow.Day,
            editRow.Month,
            editRow.Week,
        ];

        for (let i = 0; i < cfg.length; i++) {
            cfg[i] = cfg[i].trim() || "*";
        }

        row.Cfg = cfg.join(" ");

        try {
            row.Task = JSON.parse(editRow.Task);
        } catch (e) {
            alert("Task JSON.parse " + e.message);
            return;
        }

        let maxId = 0;
        let cfgs = [];
        for (let i = 0; i < cron.length; i++) {
            if (cfg.Id != cron[i].Id) {
                cfgs.push(cron[i]);
            }
            if (cron[i].Id > maxId) {
                maxId = cron[i].Id || 0;
            }
        }
        if (!(row.Id > 0)) {
            row.Id = maxId + 1;
        }
        cfgs.push(row);

        let resp = await sendJson(mkUrl("api/cron/setConfig"), cfgs);

        if (resp.Code != 0) {
            alert(resp.Data);
            return;
        }

        isShow = !isShow;

        showStatus();
    }
</script>

<Layout tab="5">
    <div id="tasks">
        <table>
            <thead>
                <tr>
                    <th>规则</th>
                    <th>备注</th>
                    <th>任务</th>
                    <th />
                    <th />
                </tr>
            </thead>
            <tbody>
                {#if cron.Tasks}
                    {#each cron.Tasks as row}
                        <tr>
                            <td>{row.Cfg}</td>
                            <td>{row.Note}</td>
                            <td>{JSON.stringify(row.Task)}</td>
                            <td
                                ><button on:click={() => rowEdit(row)}
                                    >编辑</button
                                ></td
                            >
                            <td
                                ><button on:click={() => rowDel(row)}
                                    >删除</button
                                ></td
                            >
                        </tr>
                    {/each}
                {:else}
                    <tr>
                        <td colspan="5" class="center2">empty</td>
                    </tr>
                {/if}
                <tr>
                    <td colspan="4" />
                    <td><button on:click={() => rowAdd()}>添加</button></td>
                </tr>
            </tbody>
        </table>
    </div>
</Layout>

<Dialog bind:isShow>
    <div class="box">
        <div>
            <pre>
*: 匹配该字段所有值
/: 表示范围增量
,: 用来分隔同一组中的项目
-: 表示范围
例: */5 10-12 * 1,3  表示每周1周3十点到十二点每五分钟执行一次
</pre>
        </div>
        <div class="grid">
            <label for="row_cfg">Cfg: </label>
            <div id="row_cfg">
                <input bind:value={editRow.Minute} />
                <input bind:value={editRow.Hour} />
                <input bind:value={editRow.Day} />
                <input bind:value={editRow.Month} />
                <input bind:value={editRow.Week} />
                <span>分</span>
                <span>时</span>
                <span>天</span>
                <span>月</span>
                <span>周</span>
            </div>

            <label for="row_note">Note: </label>
            <input id="row_note" bind:value={editRow.Note} />

            <label for="row_task">Task: </label>
            <textarea id="row_task" bind:value={editRow.Task} />
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
    pre {
        font-size: 12px;
        color: #aaa;
        margin: 10px;
    }

    .box {
        padding: 10px;
        background: #fff;
        border-radius: 10px;
        width: 500px;
    }
    .grid {
        display: grid;
        gap: 10px;
        grid-template-columns: auto auto;
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
    #row_cfg {
        display: grid;
        gap: 2px 2px;
        grid-template-columns: repeat(5, 1fr);
        text-align: center;
    }
    #row_cfg input {
        width: 100%;
        text-align: center;
    }
    #row_cfg span {
        font-size: 12px;
        color: #aaa;
    }
    textarea {
        border: 1px solid #777;
        min-height:200px;
    }
</style>
