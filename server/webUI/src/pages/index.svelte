<script>
    import Layout from "./lib/layout.svelte";
    import Dialog from "./lib/dialog.svelte";
    import { onMount, onDestroy } from "svelte";
    import {
        jobSort,
        mkUrl,
        sendJson,
        jobEmpty,
        jobDelIdle,
        jobPriority,
        jobParallel,
    } from "./lib/base";

    let timer;
    onMount(() => {
        showStatus();

        timer = setInterval(function () {
            showStatus();
        }, 1000);
    });

    onDestroy(() => {
        clearInterval(timer);
    });

    let AllData = {};
    let Tasks = [];
    let Groups = [];

    let sortby = 2;
    let tab = 2;
    let sortName = [
        { k: 1, n: "名称" },
        { k: 9, n: "工作组" },
        { k: 2, n: "负载" },
        { k: 3, n: "执行中" },
        { k: 4, n: "己执行" },
        { k: 5, n: "昨天" },
        { k: 10, n: "昨出错" },
        { k: 6, n: "队列" },
        { k: 7, n: "平均时间" },
        { k: 8, n: "优先级" },
    ];

    function getCapacity(gid) {
        for (let g of Groups) {
            if (g.Id == gid) {
                return g.Capacity;
            }
        }
        return 0;
    }

    function setTab(by) {
        tab = by;
    }

    function setSort(by) {
        sortby = sortby == by ? -by : by;
    }

    async function showStatus() {
        let json = await fetch(mkUrl("api/task/status")).then((t) => t.json());

        let res = json.Data;

        res.Groups.sort(function (a, b) {
            return a.Id < b.Id ? -1 : 1;
        });

        if (tab == 2 || tab == 3) {
            res.Tasks = res.Tasks.filter(function (task) {
                let ok = () => task.NowNum > 0 || task.WaitNum > 0;
                return tab == 3 ? !ok() : ok();
            });
        }

        res.Tasks.sort(jobSort(sortby));

        Groups = res.Groups;
        Tasks = res.Tasks;

        res.Capacity = 0;
        res.Load = 0;
        res.NowNum = 0;
        res.RunNum = 0;
        res.ErrNum = 0;
        res.OldRun = 0;
        res.OldErr = 0;
        res.WaitNum = 0;
        res.WorkerNum = 0;

        res.Groups.forEach((g) => {
            res.Load += g.Load;
            res.Capacity += g.Capacity;
            res.NowNum += g.NowNum;
            res.RunNum += g.RunNum;
            res.ErrNum += g.ErrNum;
            res.OldRun += g.OldRun;
            res.OldErr += g.OldErr;
            res.WaitNum += g.WaitNum;
            res.WorkerNum += g.WorkerNum;
        });

        AllData = res;
    }

    function fmtPriority(val) {
        if (val == 0) {
            return "";
        }

        return val > 0 ? "(+" + val + ")" : "(" + val + ")";
    }

    let showAddTask = false;
    let addTaskTxt = "";

    function addTask() {
        addTaskTxt = `{
    "url": "http://g.com",
    "form": {"k":"v"}

    "cmd": "echo",
    "args": ["hellworld"]
}`;
        showAddTask = !showAddTask;
    }

    async function doAddTask() {
        let task = {};

        try {
            task = JSON.parse(addTaskTxt);
        } catch (e) {
            alert("Task JSON.parse " + e.message);
            return;
        }

        let resp = await sendJson(mkUrl("api/task/add"), task);

        if (resp.Code != 0) {
            alert(resp.Data);
            return;
        }

        showAddTask = !showAddTask;
    }
</script>

<Layout tab="2">
    <div id="All">
        <table>
            <thead>
                <tr>
                    <th class="name">工作组</th>
                    <th class="load">负载</th>
                    <th class="now">执行中</th>
                    <th class="run">己执行</th>
                    <th class="old">昨天</th>
                    <th class="old">昨出错</th>
                    <th class="wait">队列</th>
                    <th>定时</th>
                </tr>
            </thead>
            <tbody>
                <tr>
                    <td>总体</td>
                    <td
                        >{Math.round(
                            (AllData.Load / AllData.Capacity) * 100
                        )}%</td
                    >
                    <td>{AllData.NowNum} / {AllData.WorkerNum}</td>
                    <td>{AllData.RunNum}</td>
                    <td>{AllData.OldRun}</td>
                    <td>{AllData.OldErr}</td>
                    <td>{AllData.WaitNum}</td>
                    <td>{AllData.Timed}</td>
                </tr>
                {#each Groups as g}
                    <tr>
                        <td>{g.Id} ({g.Note})</td>
                        <td>{Math.round((g.Load / g.Capacity) * 100)}%</td>
                        <td>{g.NowNum} / {g.WorkerNum}</td>
                        <td>{g.RunNum}</td>
                        <td>{g.OldRun}</td>
                        <td>{g.OldErr}</td>
                        <td>{g.WaitNum}</td>
                        <td />
                    </tr>
                {/each}
            </tbody>
        </table>
    </div>

    <div id="tab">
        <button
            class="wait {tab == 2 ? 'active' : ''}"
            on:click={() => setTab(2)}>waiting</button
        >
        <button
            class="idle {tab == 3 ? 'active' : ''}"
            on:click={() => setTab(3)}>idle</button
        >
        <button
            class="all {tab == 4 ? 'active' : ''}"
            on:click={() => setTab(4)}>all</button
        >
        <button on:click={() => addTask()}>add</button>
    </div>

    <div id="jobs">
        <table>
            <thead>
                <tr>
                    {#each sortName as d}
                        <th on:click={() => setSort(d.k)}
                            >{d.n}
                            {#if Math.abs(sortby) == d.k}
                                {sortby < 0 ? " ↑" : " ↓"}
                            {/if}
                        </th>
                    {/each}
                    <th>上次</th>
                    <th>报错</th>
                </tr>
            </thead>
            <tbody>
                {#each Tasks as j}
                    <tr>
                        <td on:dblclick={() => jobDelIdle(j)}>{j.Name}</td>

                        <td>{j.GroupId}</td>

                        <td
                            >{Math.round(
                                (j.Load / getCapacity(j.GroupId)) * 100
                            )}%</td
                        >

                        <td on:dblclick={() => jobParallel(j)}
                            >{j.NowNum + "/" + j.Parallel}</td
                        >

                        <td>{j.RunNum}</td>
                        <td>{j.OldRun}</td>
                        <td>{j.OldErr}</td>
                        <td on:dblclick={() => jobEmpty(j)}>{j.WaitNum}</td>
                        <td>{j.UseTime / 1000}s</td>

                        <td on:dblclick={() => jobPriority(j)}>
                            {j.Score + fmtPriority(j.Priority)}
                        </td>

                        <td>{j.LastTime}s</td>
                        <td>{j.ErrNum}</td>
                    </tr>
                {:else}
                    <tr><td colspan="12" class="center">empty</td> </tr>
                {/each}
            </tbody>
        </table>
    </div>
</Layout>

<Dialog bind:isShow={showAddTask}>
    <div class="box">
        <div>
            <textarea id="row_task" bind:value={addTaskTxt} />
        </div>
        <div class="center">
            <button type="button" on:click={doAddTask}>添加</button>
            <button type="button" on:click={() => (showAddTask = !showAddTask)}
                >取消</button
            >
        </div>
    </div>
</Dialog>

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
    #tab button {
        margin: auto 0.5em;
        color: #777;
    }
    #tab .active {
        color: black;
        font-weight: bold;
    }
    .box {
        padding: 10px;
        background: #fff;
        border-radius: 10px;
        width: 500px;
    }
    textarea {
        width: 100%;
        border: 1px solid #777;
        min-height: 200px;
    }
</style>
