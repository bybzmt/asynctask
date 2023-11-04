
export default {
    map: {
        "/": {
            page: () => import('./pages/index.svelte'),
        },
        "/task/running": {
            page: () => import('./pages/running.svelte'),
        },
        "/task/history": {
            page: () => import('./pages/history.svelte'),
        },
        "/task/add": {
            page: () => import('./pages/taskadd.svelte'),
        },
        "/config/groups": {
            page: () => import('./pages/groups.svelte'),
        },
        "/config/taskrules": {
            page: () => import('./pages/taskrules.svelte'),
        },
        "/config/jobrules": {
            page: () => import('./pages/jobrules.svelte'),
        },
        "/cron": {
            page: () => import('./pages/cron.svelte'),
        },
        "/timed": {
            page: () => import('./pages/timed.svelte'),
        },
    }
}
