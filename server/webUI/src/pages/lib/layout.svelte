<script>
    import "./tailwind.css"

    export let tab = 1
</script>

<div class="text-gray-800">
    <div class="m-4 text-gray-500 flex gap-4">
        <a class:text-gray-900={tab == 1} href="#/task/running">Running</a>
        <a class:text-gray-900={tab == 2} href="#/">Status</a>
        <a class:text-gray-900={tab == 3} href="#/task/history">History</a>
        <a class:text-gray-900={tab == 6} href="#/config">Config</a>
        <a class:text-gray-900={tab == 8} href="#/timed">Timed</a>
        <a class:text-gray-900={tab == 9} href="#/task/add">AddTask</a>
    </div>

    <slot />
</div>
