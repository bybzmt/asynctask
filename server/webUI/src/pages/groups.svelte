<script>
    import Layout from "./lib/layout.svelte";
    import Dialog from "./lib/dialog.svelte";
    import { onMount } from "svelte";
    import { mkUrl, sendJson } from "./lib/base";

    onMount(() => {
        showStatus();
    });

    let Groups = [];
    let editGroup = {};
    let isShow = false;

    async function showStatus() {
        let json = await fetch(mkUrl("api/group/list")).then((t) => t.json());

        Groups = json.Data || [];
    }

    async function groupDel(row) {
        var ok = confirm(`Del Group?\r\nId:${row.Id} Note: ${row.Note}`);
        if (ok) {
            await sendJson(mkUrl("api/group/del"), {
                gid: row.Id,
            });

            await showStatus();
        }
    }

    async function groupAdd() {
        var ok = confirm(`Add Group?`);
        if (ok) {
            await sendJson(mkUrl("api/group/add"), {});

            await showStatus();
        }
    }

    function edit(group) {
        editGroup = group;
        isShow = !isShow;
    }

    async function save() {
        let workerNum = parseInt(editGroup.WorkerNum)

        if (isNaN(workerNum)) {
            alert("workerNum need integer")
            return
        }

        await sendJson(mkUrl("api/group/setConfig"), {
            Id: editGroup.Id,
            Note: editGroup.Note,
            WorkerNum: workerNum,
        });

        isShow = !isShow;

        await showStatus();
    }
</script>

<Layout tab="4">
    <div>
        <table class="m-4 border text-base text-gray-800">
            <thead>
                <tr>
                    <th class="px-2 py-1 border">ID</th>
                    <th class="px-2 py-1 border">备注</th>
                    <th class="px-2 py-1 border">并行数</th>
                    <th class="px-2 py-1 border" colspan="2" />
                </tr>
            </thead>
            <tbody>
                {#each Groups as group}
                    <tr>
                        <td class="px-2 py-1 border text-center">{group.Id}</td>
                        <td class="px-2 py-1 border">{group.Note}</td>
                        <td class="px-2 py-1 border">{group.WorkerNum}</td>
                        <td class="px-2 py-1 border"
                            ><button on:click={() => edit(group)}>编辑</button
                            ></td
                        >
                        <td class="px-2 py-1 border"
                            ><button on:click={() => groupDel(group)}
                                >删除</button
                            ></td
                        >
                    </tr>
                {:else}
                    <tr>
                        <td colspan="5" class="text-center px-2 py-1 border"
                            >empty</td
                        >
                    </tr>
                {/each}
                <tr>
                    <td class="px-2 py-1 text-center"
                        ><button on:click={() => groupAdd()}>添加</button></td
                    >
                    <td colspan="4" />
                </tr>
            </tbody>
        </table>
    </div>
</Layout>

<Dialog bind:isShow>
    <div class="bg-white rounded-lg p-4">
        <div class="grid gap-2">
            <label for="note">Note: </label>
            <input class="border" id="note" bind:value={editGroup.Note} />
            <label for="workerNum">WorkerNum: </label>
            <input
                class="border"
                id="workerNum"
                bind:value={editGroup.WorkerNum}
            />
        </div>
        <div class="flex justify-center mt-2">
            <button class="mx-4" type="button" on:click={save}>确定</button>
            <button
                class="mx-4"
                type="button"
                on:click={() => (isShow = !isShow)}>取消</button
            >
        </div>
    </div>
</Dialog>
