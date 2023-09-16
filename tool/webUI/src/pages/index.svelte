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

    let GroupId = 0;
    let Groups = [];
    let AllData = {};

    let JobsData = [];
    let TasksData = [];
    let sortby = 2;
    let tab = 2;
    let sortName = [
        { k: 1, n: "名称" },
        { k: 2, n: "负载" },
        { k: 3, n: "执行中" },
        { k: 4, n: "己执行" },
        { k: 5, n: "昨天" },
        { k: 6, n: "队列" },
        { k: 7, n: "平均时间" },
        { k: 8, n: "优先级" },
    ];

    function setTab(by) {
        tab = by;
    }

    function setSort(by) {
        sortby = sortby == by ? -by : by;
    }

    function selectGroup(id) {
        GroupId = id;
    }

    async function showStatus() {
        let res = await getStatus(GroupId)

        AllData = res.all

        Groups = res.groups

        res.tasks.sort(function (a, b) {
            return a.Id.localeCompare(b.Id);
        })

        TasksData = res.tasks

        if (tab == 2 || tab == 3) {
            res.jobs = res.jobs.filter(function (job) {
                if (tab == 3) {
                    return job.NowNum == 0 && job.WaitNum == 0;
                }
                return job.NowNum > 0 || job.WaitNum > 0;
            });
        }

        res.jobs.sort(jobSort(sortby))

        JobsData = res.jobs
    }

    function fmtPriority(val) {
        if (val == 0) {
            return "";
        }

        return val > 0 ? "(+" + val + ")" : "(" + val + ")";
    }
</script>

<Layout>
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
                    <th />
                </tr>
            </thead>
            <tbody>
                <tr on:click={() => selectGroup(0)}>
                    <td>总体</td>
                    <td>{Math.round(AllData.Load / AllData.Capacity)}%</td>
                    <td>{AllData.NowNum} / {AllData.WorkerNum}</td>
                    <td>{AllData.RunNum}</td>
                    <td>{AllData.OldNum}</td>
                    <td>{AllData.WaitNum}</td>
                    <td
                        >{#if GroupId == 0}√{/if}</td
                    >
                </tr>
                {#each Groups as g}
                    <tr on:click={() => selectGroup(g.Id)}>
                        <td>{g.Id} ({g.Note})</td>
                        <td>{Math.round(g.Load / AllData.Capacity)}%</td>
                        <td>{g.NowNum} / {g.WorkerNum}</td>
                        <td>{g.RunNum}</td>
                        <td>{g.OldNum}</td>
                        <td>{g.WaitNum}</td>
                        <td
                            >{#if GroupId == g.Id}√{/if}</td
                        >
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
                    {#each TasksData as task}
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
                    {#each JobsData as j}
                        <tr>
                            <td on:dblclick={() => jobDelIdle(j)}>{j.Name}</td>
                            <td>{j.Load / 100}%</td>
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
                        <tr><td colspan="10" class="center">empty</td> </tr>
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
