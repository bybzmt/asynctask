

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

    if (data) {
        for (let k in data) {
            params.set(k, data[k])
        }
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

export async function getStatus() {

    let json = await fetch(mkUrl("api/status")).then((t) => t.json())

    return json.Data
}
