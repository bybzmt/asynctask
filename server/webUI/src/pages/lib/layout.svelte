<script>
    import "./tailwind.css";

    export let tab = 1;
</script>

<div class="text-gray-800">
    <div class="m-4 text-gray-500 flex gap-4">
        <a class:text-gray-900={tab == 1} href="#/runing">Runing</a>
        <a class:text-gray-900={tab == 2} href="#/">Tasks</a>
        <a class:text-gray-900={tab == 7} href="#/jobs">Jobs</a>
        <a class:text-gray-900={tab == 3} href="#/router">Router</a>
        <a class:text-gray-900={tab == 8} href="#/rules">Rules</a>
        <a class:text-gray-900={tab == 4} href="#/groups">Groups</a>
        <a class:text-gray-900={tab == 5} href="#/cron">Cron</a>
        <a class:text-gray-900={tab == 6} href="#/timed">Timed</a>
    </div>

    <slot />
</div>
