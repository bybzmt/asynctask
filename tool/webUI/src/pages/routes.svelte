<script>
    import Layout from "./lib/layout.svelte";
    import { onMount } from "svelte";
    import { sendPost, mkUrl } from "./lib/base";

    onMount(() => {
        showStatus();
    });

    let Routes = [];
    let Groups = {};

    function get(id) {
        return Groups[id] || {};
    }

    async function showStatus() {
        let json = await fetch(mkUrl("api/groups")).then((t) => t.json());

        let tmp = {};
        json.Data.forEach((v) => (tmp[v.Id] = v));
        Groups = tmp;

        json = await fetch(mkUrl("api/routes")).then((t) => t.json());

        Routes = json.Data;
    }

    function routeEdit(row) {}

    function routeDel(row) {
        var ok = confirm(`Del Group?\r\nId:${row.Id} Note: ${row.Note}`);
        if (ok) {
            sendPost(mkUrl("api/route/del"), {
                rid: row.Id,
            });

            showStatus();
        }
    }

    function routeAdd() {
        var ok = confirm(`Add Route?`);
        if (ok) {
            sendPost(mkUrl("api/route/add"));

            showStatus();
        }
    }
</script>

<Layout>
    <div id="tab">
        <a href="#/">Tasks</a>
        <a href="#/routes">Routes</a>
        <a href="#/groups">WorkGroups</a>
    </div>

    <div id="tasks">
        <table>
            <thead>
                <tr>
                    <th>ID</th>
                    <th>备注</th>
                    <th>正则</th>
                    <th>执行组</th>
                    <th>排序</th>
                    <th>优先级</th>
                    <th>并发</th>
                    <th>模式</th>
                    <th>超时</th>
                    <th>CLi</th>
                    <th>Http</th>
                    <th>启用</th>
                    <th />
                    <th />
                </tr>
            </thead>
            <tbody>
                {#each Routes as row}
                    <tr>
                        <td>{row.Id}</td>
                        <td>{row.Note}</td>
                        <td>{row.Match}</td>
                        <td>
                            {#if row.Groups}
                                {#each row.Groups as id}
                                    <div>{id}({get(id).Note})</div>
                                {:else}
                                    <div>empty</div>
                                {/each}
                            {/if}
                        </td>
                        <td>{row.Sort}</td>
                        <td>{row.Priority}</td>
                        <td>{row.Parallel}</td>
                        <td>{row.Mode}</td>
                        <td>{row.Timeout}</td>
                        <td>{row.CmdBase}</td>
                        <td>{row.HttpBase}</td>
                        <td>{row.Used ? "Enable" : "Disable"}</td>
                        <td on:click={() => routeEdit(row)}>编辑</td>
                        <td on:click={() => routeDel(row)}>删除</td>
                    </tr>
                {:else}
                    <tr>
                        <td colspan="14" class="center">empty</td>
                    </tr>
                {/each}
                <tr>
                    <td colspan="13" />
                    <td on:click={() => routeAdd()}>添加</td>
                </tr>
            </tbody>
        </table>
    </div>
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
</style>
