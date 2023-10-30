<script>
    export let args = [];

    let val = "";

    function del(i) {
        let t = [];
        for (let x = 0; x < args.length; x++) {
            if (x != i) {
                t.push(args[x]);
            }
        }
        args = t;
    }

    function set() {
        val = val.trim();
        if (val == "") {
            return;
        }

        args.push(val);
        args = args;

        val = "";
    }
</script>

<div class="grid grid-cols-[auto_auto] gap-2">
    {#each args as param, i}
        <div>{param}</div>
        <div class="text-sm"><button on:click={() => del(i)}>删除</button></div>
    {/each}

    <div>
        <input class="border w-full" bind:value={val} placeholder="value" />
    </div>
    <div><button class="text-sm" on:click={() => set()}>添加</button></div>
</div>
