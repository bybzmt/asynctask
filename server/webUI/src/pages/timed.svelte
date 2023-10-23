<script>
    import Layout from "./lib/layout.svelte";
    import Dialog from "./lib/dialog.svelte";
    import { onMount } from "svelte";
    import { sendJson, mkUrl } from "./lib/base";

    onMount(() => {
        showStatus();
    });

    let rows = [];

    async function showStatus() {
        let json = await fetch(mkUrl("api/task/timed")).then((t) => t.json());

        rows = json.Data || [];
    }

    async function rowDel(row) {
        var ok = confirm(`Del Group?\r\nId:${row.Id} Note: ${row.Note}`);
        if (ok) {
        }
    }

    let timezoneOffset = new Date().getTimezoneOffset() * 60;

    function timeStr(t) {
        return new Date((t - timezoneOffset) * 1000)
            .toISOString()
            .substring(0, 19);
    }
</script>

<Layout tab="6">
    <div id="tasks">
        <table>
            <thead>
                <tr>
                    <th>运行时间</th>
                    <th>任务</th>
                    <th />
                </tr>
            </thead>
            <tbody>
                {#each rows as row}
                    <tr>
                        <td>{timeStr(row.timer)}</td>
                        <td>{JSON.stringify(row)}</td>
                        <td
                            ><button on:click={() => rowDel(row)}>删除</button
                            ></td
                        >
                    </tr>
                {:else}
                    <tr>
                        <td colspan="3" class="center2">empty</td>
                    </tr>
                {/each}

                <tr>
                    <td colspan="2" />
                    <td><button on:click={() => rowAdd()}>添加</button></td>
                </tr>
            </tbody>
        </table>
    </div>
</Layout>

<style>
    table {
        margin: 1em;
        border-collapse: collapse;
        border: 1px solid #777;
    }
    table td,
    table th {
        border: 1px solid #777;
        padding: 0px 1em;
    }
    .center2 {
        text-align: center;
    }
</style>
