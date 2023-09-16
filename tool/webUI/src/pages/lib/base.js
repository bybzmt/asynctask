

export function jobSort(sortby) {
    return function (a, b) {
        var x = (function() {
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
    }
}

export function mkUrl(url, data) {
    let u = new URL(url, location.href)

    for (let k in data) {
        u.searchParams.set(k, data[k])
    }

    return u.toString()
}

export function sendPost(url, data) {
    let params = new URLSearchParams();

    for (let k in data) {
        params.set(k, data[k])
    }

    fetch(url, {
        method: "POST",
        body: params,
    })
        .then((t) => t.json())
        .then((json) => {
            alert(JSON.stringify(json));
        });
}

export function sendJson(url, data) {
    let he = new Headers();
    he.append("Content-Type", "application/json; charset=utf-8");

    fetch(url, {
        method: "POST",
        headers: he,
        body: JSON.stringify(data),
    })
        .then((t) => t.json())
        .then((json) => {
            alert(JSON.stringify(json));
        });
}

export function taskCancel(task) {
    var ok = confirm(
        "Cancel Task?\r\nName: " + task.Name + " " + task.Task
    );
    if (ok) {
        sendPost(mkUrl("api/task/cancel"), {
            gid: task.Gid,
            tid: task.Name,
        });
    }
}

export function jobEmpty(job) {
    var ok = confirm("Empty Job?\r\nName: " + job.Name);
    if (ok) {
        sendPost(mkUrl("api/job/empty"), {
            gid: job.Gid,
            name: job.Name,
        });
    }
}

export function jobDelIdle(job) {
    var ok = confirm("Del Idle Job?\r\nName: " + job.Name);
    if (ok) {
        sendPost(mkUrl("api/job/delIdle"), {
            gid: job.Gid,
            name: job.Name,
        });
    }
}

export function jobPriority(job) {
    var txt = prompt("Job: " + job.Name + " Priority: ", job.Priority);
    if (txt != null && txt != "") {
        let Priority = parseInt(txt);

        sendJson(
            mkUrl("api/job/setConfig", {
                gid: gid,
                name: job.Name,
            }),
            {
                Priority: Priority,
                Parallel: job.Parallel,
            }
        );
    }
}

export function jobParallel(job) {
    var txt = prompt("Job: " + job.Name + " Parallel: ", job.Parallel);
    if (txt != null && txt != "") {
        let Parallel = parseInt(txt);

        sendJson(
            mkUrl("api/job/setConfig", {
                gid: gid,
                name: job.Name,
            }),
            {
                Priority: job.Priority,
                Parallel: Parallel,
            }
        );
    }
}

export async function getStatus(GroupId) {

    let json = await fetch(mkUrl("api/status")).then((t) => t.json())

    let jobs = new Map();
    let tasks = [];
    let groups = [];

    let all = {
        Capacity: 0,
        Load: 0,
        RunNum: 0,
        OldNum: 0,
        NowNum: 0,
        WaitNum: 0,
        WorkerNum: 0,
    };

    json.Data.forEach((g) => {
        if (GroupId != 0 && g.Id != GroupId) {
            return;
        }

        all.Capacity += g.Capacity;
        all.Load += g.Load;
        all.RunNum += g.RunNum;
        all.OldNum += g.OldNum;
        all.NowNum += g.NowNum;
        all.WaitNum += g.WaitNum;
        all.WorkerNum += g.WorkerNum;

        groups.push({
            Id: g.Id,
            Note: g.Note,
            Load: g.Load,
            RunNum: g.RunNum,
            OldNum: g.OldNum,
            NowNum: g.NowNum,
            WaitNum: g.WaitNum,
            WorkerNum: g.WorkerNum,
        });

        g.Tasks.forEach((t) => {
            t.Gid = g.Id;
            tasks.push(t);
        });

        g.Jobs.forEach((j) => {
            j.Gid = g.Id;

            if (jobs.has(j.Name)) {
                j2 = jobs.get(j.Name);
                j2.Load += j.Load;
                j2.RunNum += j.RunNum;
                j2.OldNum += j.OldNum;
                j2.NowNum += j.NowNum;
                j2.ErrNum += j.ErrNum;
                j2.WaitNum += j.WaitNum;
                j2.Parallel += j.Parallel;
            } else {
                jobs.set(j.Name, j);
            }
        });
    });

    let out = [];

    jobs.forEach((v) => out.push(v));

    return {
        all,
        tasks,
        groups,
        jobs: out,
    }
}
