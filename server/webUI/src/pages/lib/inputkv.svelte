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

<div class="table">
    <div>Key</div>
    <div>Value</div>
    <div />

    {#each params as param}
        <div>{param.key}</div>
        <div>{param.val}</div>
        <div><button on:click={() => del(param)}>删除</button></div>
    {/each}

    <div><input bind:value={key} /></div>
    <div><input bind:value={val} /></div>
    <div><button on:click={() => set()}>添加</button></div>
</div>

<style>
    .table {
        display: grid;
        grid-template-columns: auto auto auto;
    }
    .table input {
        border-bottom: 1px solid #777;
    }
    button {
        font-size: 12px;
        padding: 3px 5px;
    }
</style>
