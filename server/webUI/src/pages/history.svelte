<script>
    import Layout from "./lib/layout.svelte"
    import { onMount, onDestroy } from "svelte"
    import { jobSort, mkUrl, sendJson, beforSecond } from "./lib/base"

    let timer
    onMount(() => {
        showStatus()

        timer = setInterval(function () {
            showStatus()
        }, 1000)
    })

    onDestroy(() => {
        clearInterval(timer)
    })

    let AllData = {}
    let Tasks = []
    let Groups = []

    let sortby = 1
    let sortName = [
        { k: 1, n: "名称" },
        { k: 9, n: "工作组" },
        { k: 2, n: "负载" },
        { k: 3, n: "执行中" },
        { k: 4, n: "己执行" },
        { k: 5, n: "昨天" },
        { k: 10, n: "昨错" },
        { k: 6, n: "队列" },
        { k: 7, n: "平均时间" },
        { k: 8, n: "优先级" },
    ]

    function getCapacity(gid) {
        for (let g of Groups) {
            if (g.Id == gid) {
                return g.Capacity
            }
        }
        return 0
    }

    function setSort(by) {
        sortby = sortby == by ? -by : by
    }

    async function showStatus() {
        let json = await sendJson(mkUrl("api/task/status"))

        let res = json.Data

        res.Groups.sort(function (a, b) {
            return a.Id < b.Id ? -1 : 1
        })

        res.Tasks.sort(jobSort(sortby))

        Groups = res.Groups
        Tasks = res.Tasks

        res.Capacity = 0
        res.Load = 0
        res.NowNum = 0
        res.RunNum = 0
        res.ErrNum = 0
        res.OldRun = 0
        res.OldErr = 0
        res.WaitNum = 0
        res.WorkerNum = 0

        res.Groups.forEach(g => {
            res.Load += g.Load
            res.Capacity += g.Capacity
            res.NowNum += g.NowNum
            res.RunNum += g.RunNum
            res.ErrNum += g.ErrNum
            res.OldRun += g.OldRun
            res.OldErr += g.OldErr
            res.WaitNum += g.WaitNum
            res.WorkerNum += g.WorkerNum
        })

        AllData = res
    }

    function fmtPriority(j) {
        if (j.Priority == 0) {
            return j.Score
        }

        return j.Score + (j.Priority > 0 ? "(+" + j.Priority + ")" : "(" + j.Priority + ")")
    }

    async function jobEmpty(job) {
        var ok = confirm("Empty Job?\r\nName: " + job.Name)
        if (ok) {
            await sendJson(mkUrl("api/task/empty"), {
                name: job.Name,
            })
            await showStatus()
        }
    }

    async function jobDelIdle(job) {
        var ok = confirm("Del Idle Job?\r\nName: " + job.Name)
        if (ok) {
            await sendJson(mkUrl("api/task/delIdle"), {
                name: job.Name,
            })
            await showStatus()
        }
    }
</script>

<Layout tab="3">
    <div id="All">
        <table class="m-4 border text-base text-center">
            <thead>
                <tr>
                    <th class="px-2 py-1 border">工作组</th>
                    <th class="px-2 py-1 border">负载</th>
                    <th class="px-2 py-1 border">执行中</th>
                    <th class="px-2 py-1 border">己执行</th>
                    <th class="px-2 py-1 border">出错</th>
                    <th class="px-2 py-1 border">昨天</th>
                    <th class="px-2 py-1 border">昨错</th>
                    <th class="px-2 py-1 border">队列</th>
                    <th class="px-2 py-1 border">定时</th>
                </tr>
            </thead>
            <tbody>
                <tr>
                    <td class="px-2 py-1 border text-left">总体</td>
                    <td class="px-2 py-1 border">{Math.round((AllData.Load / AllData.Capacity) * 100)}%</td>
                    <td class="px-2 py-1 border">{AllData.NowNum} / {AllData.WorkerNum}</td>
                    <td class="px-2 py-1 border">{AllData.RunNum}</td>
                    <td class="px-2 py-1 border">{AllData.ErrNum}</td>
                    <td class="px-2 py-1 border">{AllData.OldRun}</td>
                    <td class="px-2 py-1 border">{AllData.OldErr}</td>
                    <td class="px-2 py-1 border">{AllData.WaitNum}</td>
                    <td class="px-2 py-1 border">{AllData.Timed}</td>
                </tr>
                {#each Groups as g}
                    <tr>
                        <td class="px-2 py-1 border text-left">{g.Id}: {g.Note}</td>
                        <td class="px-2 py-1 border">{Math.round((g.Load / g.Capacity) * 100)}%</td>
                        <td class="px-2 py-1 border">{g.NowNum} / {g.WorkerNum}</td>
                        <td class="px-2 py-1 border">{g.RunNum}</td>
                        <td class="px-2 py-1 border">{g.OldRun}</td>
                        <td class="px-2 py-1 border">{g.OldErr}</td>
                        <td class="px-2 py-1 border">{g.WaitNum}</td>
                        <td class="px-2 py-1 border" />
                    </tr>
                {/each}
            </tbody>
        </table>
    </div>

    <div id="jobs">
        <table class="m-4 border text-base text-gray-800">
            <thead>
                <tr>
                    {#each sortName as d}
                        <th class="px-2 py-1 border" on:click={() => setSort(d.k)}
                            >{d.n}
                            {#if Math.abs(sortby) == d.k}
                                {sortby < 0 ? " ↑" : " ↓"}
                            {/if}
                        </th>
                    {/each}
                    <th class="px-2 py-1 border">上次</th>
                    <th class="px-2 py-1 border">报错</th>

                    <th colspan="2" class="px-2 py-1 border" />
                </tr>
            </thead>
            <tbody>
                {#each Tasks as j}
                    <tr>
                        <td class="px-2 py-1 border">{j.Name}</td>
                        <td class="px-2 py-1 border">{j.GroupId}</td>
                        <td class="px-2 py-1 border">{Math.round((j.Load / getCapacity(j.GroupId)) * 100)}%</td>
                        <td class="px-2 py-1 border">{j.NowNum + "/" + j.Parallel}</td>
                        <td class="px-2 py-1 border">{j.RunNum}</td>
                        <td class="px-2 py-1 border">{j.OldRun}</td>
                        <td class="px-2 py-1 border">{j.OldErr}</td>
                        <td class="px-2 py-1 border">{j.WaitNum}</td>
                        <td class="px-2 py-1 border">{j.UseTime / 1000}s</td>
                        <td class="px-2 py-1 border">{fmtPriority(j)}</td>
                        <td class="px-2 py-1 border">{beforSecond(j.LastTime)}</td>
                        <td class="px-2 py-1 border">{j.ErrNum}</td>
                        <td class="px-2 py-1 border"><button on:click={() => jobEmpty(j)}>Empty</button></td>
                        <td class="px-2 py-1 border"><button on:click={() => jobDelIdle(j)}>Del Stat</button></td>
                    </tr>
                {:else}
                    <tr><td colspan="13" class="px-2 py-1 border text-center">empty</td> </tr>
                {/each}
            </tbody>
        </table>
    </div>
</Layout>
