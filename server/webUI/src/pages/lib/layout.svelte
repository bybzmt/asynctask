<script>
    import "./tailwind.css";

    export let tab = 1;
</script>

<div>
    <div class="tab">
        <a class="{tab == 1 ? 'active' : ''}" href="#/runing">Runing</a>
        <a class="{tab == 2 ? 'active' : ''}" href="#/">Tasks</a>
        <a class="{tab == 3 ? 'active' : ''}" href="#/routes">Routes</a>
        <a class="{tab == 4 ? 'active' : ''}" href="#/groups">Groups</a>
        <a class="{tab == 5 ? 'active' : ''}" href="#/cron">Cron</a>
    </div>

    <slot />
</div>

<style>
    .tab {
        margin: 0 0.5em;
    }
    .tab a {
        margin: auto 0.5em;
        color: #777;
    }
    .tab a.active {
        color: black;
        font-weight: bold;
    }
</style>

