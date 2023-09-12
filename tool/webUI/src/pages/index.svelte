<script>
    import Layout from "./lib/layout.svelte";
    import { onMount, onDestroy } from "svelte";

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

    let GroupIdx = 0;
    let Group = null;
    let AllData = [];
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
        if (sortby == by) {
            sortby = -by;
        } else {
            sortby = by;
        }
    }

    function taskCancel(task) {
        let params = new URLSearchParams();
        params.set("gid", Group.Id);
        params.set("oid", task.id);

        var ok = confirm(
            "Cancel Task?\r\nName: " + task.Name + " " + params.toString()
        );
        if (ok) {
            let url = API_BASE + "/api/task/cancel?" + params.toString();
            fetch(url)
                .then((t) => t.json())
                .then((json) => {
                    alert(JSON.stringify(json));
                });
        }
    }

    function jobEmpty(job) {
        let params = new URLSearchParams();
        params.set("gid", Group.Id);
        params.set("jid", job.Id);

        var ok = confirm(
            "Empty Job?\r\nName: " + job.Name + " " + params.toString()
        );
        if (ok) {
            let url = API_BASE + "/api/job/empty?name=" + params.toString();
            fetch(url)
                .then((t) => t.json())
                .then((json) => {
                    alert(JSON.stringify(json));
                });
        }
    }

    function jobDelIdle(job) {
        let params = new URLSearchParams();
        params.set("gid", Group.Id);
        params.set("jid", job.Id);

        var ok = confirm(
            "Del Idle Job?\r\nName: " + job.Name + " " + params.toString()
        );
        if (ok) {
            let url = API_BASE + "/api/job/delIdle?name=" + params.toString();
            fetch(url)
                .then((t) => t.json())
                .then((json) => {
                    alert(JSON.stringify(json));
                });
        }
    }

    function jobPriority(job) {
        var txt = prompt("Job: " + job.Name + " Priority: ", job.Priority);
        if (txt != null && txt != "") {
            let params = new URLSearchParams();
            params.set("name", job.Name);

            let url = API_BASE + "/api/job/setConfig?" + params.toString();

            let he = new Headers();
            he.append("Content-Type", "application/json; charset=utf-8");

            job.Priority = parseInt(txt);

            fetch(url, {
                method: "POST",
                headers: he,
                body: JSON.stringify(job),
            })
                .then((t) => t.json())
                .then((json) => {
                    alert(JSON.stringify(json));
                });
        }
    }

    function jobParallel(job) {
        var txt = prompt("Job: " + job.Name + " Parallel: ", job.Parallel);
        if (txt != null && txt != "") {
            let params = new URLSearchParams();
            params.set("name", job.Name);

            let url = API_BASE + "/api/job/setConfig?" + params.toString();

            let he = new Headers();
            he.append("Content-Type", "application/json; charset=utf-8");

            job.Parallel = parseInt(txt);

            fetch(url, {
                method: "POST",
                headers: he,
                body: JSON.stringify(job),
            })
                .then((t) => t.json())
                .then((json) => {
                    alert(JSON.stringify(json));
                });
        }
    }

    function showStatus() {
        let url = API_BASE + "/api/status";
        fetch(url)
            .then((t) => t.json())
            .then((json) => {
                if (json.Data.length == 0) {
                    return;
                }

                if (GroupIdx >= json.Data.length) {
                    GroupIdx = 0;
                }

                let Data = json.Data[GroupIdx];

                Group = Data;
                AllData = Data.All;

                Data.Tasks.sort(function (a, b) {
                    return a.Id.localeCompare(b.Id);
                });
                TasksData = Data.Tasks;

                if (tab == 2 || tab == 3) {
                    Data.Jobs = Data.Jobs.filter(function (job) {
                        if (tab == 3) {
                            return job.NowNum == 0 && job.WaitNum == 0;
                        }
                        return job.NowNum > 0 || job.WaitNum > 0;
                    });
                }

                Data.Jobs.sort(function (a, b) {
                    var x = (function () {
                        switch (Math.abs(sortby)) {
                            case 1:
                                return b.Name.localeCompare(a.Name);
                            case 2:
                                return b.Load != a.Load
                                    ? b.Load - a.Load
                                    : b.Score - a.Score;
                            case 3:
                                return b.NowNum != a.NowNum
                                    ? b.NowNum - a.NowNum
                                    : b.Score - a.Score;
                            case 4:
                                return b.RunNum != a.RunNum
                                    ? b.RunNum - a.RunNum
                                    : b.Score - a.Score;
                            case 5:
                                return b.OldNum != a.OldNum
                                    ? b.OldNum - a.OldNum
                                    : b.Score - a.Score;
                            case 6:
                                return b.WaitNum != a.WaitNum
                                    ? b.WaitNum - a.WaitNum
                                    : b.Score - a.Score;
                            case 7:
                                return b.UseTime != a.UseTime
                                    ? b.UseTime - a.UseTime
                                    : b.Score - a.Score;
                            case 8:
                                return a.Score != b.Score
                                    ? b.Score - a.Score
                                    : b.Name.localeCompare(a.Name);
                        }
                    })();
                    return sortby > 0 ? x : -x;
                });
                JobsData = Data.Jobs;
            });
    }
</script>

<Layout>
    <div id="All">
        <table>
            <thead>
                <tr>
                    <th class="name">名称</th>
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
                    <td>{Math.round(AllData.Load / 100)}%</td>
                    <td>{AllData.NowNum}</td>
                    <td>{AllData.RunNum}</td>
                    <td>{AllData.OldNum}</td>
                    <td>{AllData.WaitNum}</td>
                </tr>
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
                        <th class="name">名称</th>
                        <th class="params">参数</th>
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
                            <td class="params">{JSON.stringify(task.Params)}</td
                            >
                            <td>{task.UseTime / 1000}s</td>
                        </tr>
                    {/each}
                    {#if TasksData.length == 0}
                        <tr>
                            <td colspan="4" class="center">empty</td>
                        </tr>
                    {/if}
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
                                {j.Score +
                                    (j.Priority == 0
                                        ? ""
                                        : j.Priority > 0
                                        ? "(+" + j.Priority + ")"
                                        : "(" + j.Priority + ")")}
                            </td>
                            <td>{j.LastTime}s</td>
                            <td>{j.ErrNum}</td>
                        </tr>
                    {:else}
                        <tr
                            ><td colspan="10" class="center">empty</td>
                        </tr>{/each}
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
