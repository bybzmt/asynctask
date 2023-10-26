<script>
    export let kv;

    let params = [];

    let key = "";
    let val = "";

    if (kv) {
        for (k in kv) {
            params.push({ key: k, val: kv[k] });
        }
    }

    function del(param) {
        params = misskey(param.key);

        change();
    }

    function misskey(key) {
        let tmp = [];

        for (let row of params) {
            if (row.key != key) {
                tmp.push(row);
            }
        }

        return tmp;
    }

    function change() {
        let tmp = {};

        for (let row of params) {
            tmp[row.key] = row.val;
        }

        kv = tmp;
    }

    function set() {
        key = key.trim();
        if (key == "") {
            return;
        }

        let tmp = misskey(key);
        tmp.push({ key: key, val: val });

        params = tmp;
        key = "";
        val = "";

        change();
    }
</script>

<div class="grid grid-cols-[auto_auto_auto] gap-2">
    {#each params as param}
        <div>{param.key}</div>
        <div>{param.val}</div>
        <div class="text-sm"><button on:click={() => del(param)}>删除</button></div>
    {/each}

    <div><input class="border" bind:value={key} placeholder="key" /></div>
    <div><input class="border" bind:value={val} placeholder="value" /></div>
    <div><button class="text-sm" on:click={() => set()}>添加</button></div>
</div>
