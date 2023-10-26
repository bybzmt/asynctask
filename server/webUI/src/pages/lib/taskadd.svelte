<script>
    import Dialog from "./dialog.svelte";
    import Task from "./task.svelte";
    import { mkUrl, sendJson } from "./base";

    export let addTask;

    let isShow = false;
    let addTaskTxt = "";
    let resolve;

    addTask = () => {
        isShow = !isShow;

        return new Promise((s, j) => {
            resolve = s;
        });
    };

    async function doAddTask() {
        let task = {};

        try {
            task = JSON.parse(addTaskTxt);
        } catch (e) {
            alert("Task JSON.parse " + e.message);
            return;
        }

        let resp = await sendJson(mkUrl("api/task/add"), task);

        if (resp.Code != 0) {
            alert(resp.Data);
            return;
        }

        resolve(true);
        isShow = !isShow;
    }

    function close() {
        isShow = !isShow;
        resolve(null);
    }
</script>

<Dialog bind:isShow>
    <div class="bg-white rounded-lg p-4 w-[500px]">
        <Task bind:value={addTaskTxt} />

        <div class="text-center mt-2">
            <button class="mx-4" type="button" on:click={doAddTask}>添加</button>
            <button class="mx-4" type="button" on:click={close}>取消</button>
        </div>
    </div>
</Dialog>
