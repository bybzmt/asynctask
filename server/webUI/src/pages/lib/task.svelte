<script>
    import { timeStr } from "./base"

    export let value
    export let type = 1

    let isTimed = false
    let strtime = ""
    let errmsg = ""

    const http_val = JSON.stringify(
        {
            url: "http://example/echo",
        },
        null,
        2,
    )

    $: {
        if (value == "" || value == http_val || value == cli_val) {
            value = http_val
        }
    }

    $: {
        if (isTimed) {
            strtime = timeStr(Date.now() / 1000)
        } else {
            strtime = ""
        }
    }

    $: {
        errmsg = ""

        try {
            let task = JSON.parse(value)

            if (strtime) {
                let d = Date.parse(strtime)
                if (!isNaN(d)) {
                    task.timer = d / 1000
                }
            } else {
                delete task.timer
            }

            value = JSON.stringify(task, null, 2)
        } catch (e) {
            errmsg = e.message
        }
    }
</script>

<div>
    <div class="flex gap-4">
        {#if type == 1}
            <label><input class="mr-2" type="checkbox" bind:checked={isTimed} />Timed</label>
            {#if isTimed}
                <input class="border" bind:value={strtime} />
            {/if}
        {/if}
    </div>
    <div class="mt-4 relative">
        <textarea class="border w-full min-h-[200px]" bind:value />
        <div class="my-2 h-4 text-xs text-red-800">{errmsg}</div>
        <pre class="text-xs text-gray-400 absolute top-0 right-0 p-4 pointer-events-none">
url      string
method   string
header   map[string]string
body     base64
runat    uint
timeout  uint
retry    uint
interval uint
status   uint
hold     string

            </pre>
    </div>
</div>
