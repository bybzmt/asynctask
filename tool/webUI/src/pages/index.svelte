<script>
    import Layout from "./lib/layout.svelte";
    import { onMount, onDestroy } from "svelte";
    import {
        jobSort,
        mkUrl,
        taskCancel,
        jobEmpty,
        jobDelIdle,
        jobPriority,
        jobParallel,
        getStatus,
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
    let Runs = [];

    let sortby = 2;
    let tab = 2;
    let sortName = [
        { k: 1, n: "名称" },
        { k: 9, n: "工作组" },
        { k: 2, n: "负载" },
        { k: 3, n: "执行中" },
        { k: 4, n: "己执行" },
        { k: 5, n: "昨天" },
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
        let res = await getStatus();

        console.log(res);

        res.Groups.sort(function (a, b) {
            return a.Id < b.Id ? -1 : 1;
        });

        if (tab == 2 || tab == 3) {
            res.Tasks = res.Tasks.filter(function (task) {
                let ok = () => task.NowNum > 0 || task.WaitNum > 0;

                if (tab == 3) {
                    return !ok();
                }
                return ok;
            });
        }

        res.Tasks.sort(jobSort(sortby));

        Groups = res.Groups;
        Tasks = res.Tasks;
        Runs = res.Runs;
        AllData = res;
    }

    function fmtPriority(val) {
        if (val == 0) {
            return "";
        }

        return val > 0 ? "(+" + val + ")" : "(" + val + ")";
    }
</script>

<Layout>
    <div id="tab">
        <a href="#/">Tasks</a>
        <a href="#/routes">Routes</a>
        <a href="#/groups">WorkGroups</a>
    </div>

    <div id="All">
        <table>
            <thead>
                <tr>
                    <th class="name">工作组</th>
                    <th class="load">负载</th>
                    <th class="now">执行中</th>
                    <th class="run">己执行</th>
                    <th class="old">昨天</th>
                    <th class="wait">队列</th>
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
                    <td>{AllData.OldNum}</td>
                    <td>{AllData.WaitNum}</td>
                </tr>
                {#each Groups as g}
                    <tr>
                        <td>{g.Id} ({g.Note})</td>
                        <td>{Math.round((g.Load / g.Capacity) * 100)}%</td>
                        <td>{g.NowNum} / {g.WorkerNum}</td>
                        <td>{g.RunNum}</td>
                        <td>{g.OldNum}</td>
                        <td />
                    </tr>
                {/each}
            </tbody>
        </table>
    </div>

    <div id="tab">
        <span class="run {tab == 1 ? 'active' : ''}" on:click={() => setTab(1)}
            >runing</span
        >
        <span class="wait {tab == 2 ? 'active' : ''}" on:click={() => setTab(2)}
            >waiting</span
        >
        <span class="idle {tab == 3 ? 'active' : ''}" on:click={() => setTab(3)}
            >idle</span
        >
        <span class="all {tab == 4 ? 'active' : ''}" on:click={() => setTab(4)}
            >all</span
        >
    </div>

    {#if tab == 1}
        <div id="tasks">
            <table>
                <thead>
                    <tr>
                        <th>ID</th>
                        <th class="name">分组</th>
                        <th class="params">任务</th>
                        <th class="time">用时</th>
                    </tr>
                </thead>
                <tbody>
                    {#each Runs as task}
                        <tr>
                            <td
                                ><span on:dblclick={() => taskCancel(task)}
                                    >{task.Id}</span
                                ></td
                            >
                            <td>{task.Name}</td>
                            <td class="params">{task.Task}</td>
                            <td>{task.UseTime / 1000}s</td>
                        </tr>
                    {:else}
                        <tr>
                            <td colspan="4" class="center">empty</td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        </div>
    {/if}

    {#if tab != 1}
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
                            <td>{j.OldNum}</td>
                            <td on:dblclick={() => jobEmpty(j)}>{j.WaitNum}</td>
                            <td>{j.UseTime / 1000}s</td>

                            <td on:dblclick={() => jobPriority(j)}>
                                {j.Score + fmtPriority(j.Priority)}
                            </td>

                            <td>{j.LastTime}s</td>
                            <td>{j.ErrNum}</td>
                        </tr>
                    {:else}
                        <tr><td colspan="11" class="center">empty</td> </tr>
                    {/each}
                </tbody>
            </table>
        </div>
    {/if}
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
    #tab span {
        margin: auto 0.5em;
        color: #777;
    }
    #tab span.active {
        color: black;
        font-weight: bold;
    }
    .params {
        max-width: 500px;
        overflow: hidden;
    }
</style>
