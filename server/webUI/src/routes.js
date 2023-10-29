
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
        "/router": {
            page: () => import('./pages/router.svelte'),
        },
        "/rules": {
            page: () => import('./pages/rules.svelte'),
        },
        "/cron": {
            page: () => import('./pages/cron.svelte'),
        },
        "/timed": {
            page: () => import('./pages/timed.svelte'),
        },
    }
}
