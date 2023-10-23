
export default {
    map: {
        "/": {
            page: () => import('./pages/index.svelte'),
        },
        "/runing": {
            page: () => import('./pages/runing.svelte'),
        },
        "/groups": {
            page: () => import('./pages/groups.svelte'),
        },
        "/routes": {
            page: () => import('./pages/routes.svelte'),
        },
        "/cron": {
            page: () => import('./pages/cron.svelte'),
        },
        "/timed": {
            page: () => import('./pages/timed.svelte'),
        },
    }
}
