

export function jobSort(sortby) {
    return function(a, b) {
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
    let u = new URL(API_BASE + url, location.href)

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

export async function sendJson(url, data) {
    let he = new Headers();
    he.append("Content-Type", "application/json; charset=utf-8");

    let resp = await fetch(url, {
        method: "POST",
        headers: he,
        body: JSON.stringify(data),
    })

    return resp.json()
}

let timezoneOffset = new Date().getTimezoneOffset() * 60;

export function timeStr(t) {
    if (!t) {
        return "N/A"
    }

    return (new Date((t - timezoneOffset) * 1000))
        .toISOString()
        .substring(0, 19);
}

export function beforSecond(t) {
    if (t > 60*60*24*3) {
        return Math.floor(t/60/60/24) + "d"
    }

    if (t > 60*60*3) {
        return Math.floor(t/60/60) + "h"
    }

    if (t > 60*3) {
        return Math.floor(t/60) + "m"
    }

    if (t == 0) {
        return 'N/A'
    }
    
    return t + "s"
}
