<script>
    import "./tailwind.css";
</script>

<div>
    <div id="tab">
        <a class="span" href="#/runing">Runing</a>
        <a class="span" href="#/">Tasks</a>
        <a class="span" href="#/routes">Routes</a>
        <a class="span" href="#/groups">WorkGroups</a>
    </div>

    <slot />
</div>

<style>
    #tab {
        margin: 0 0.5em;
    }
    #tab .span {
        margin: auto 0.5em;
        color: #777;
    }
</style>

