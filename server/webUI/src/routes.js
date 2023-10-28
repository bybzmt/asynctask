
export default {
    map: {
        "/": {
            page: () => import('./pages/index.svelte'),
        },
        "/jobs": {
            page: () => import('./pages/jobs.svelte'),
        },
        "/runing": {
            page: () => import('./pages/runing.svelte'),
        },
        "/groups": {
            page: () => import('./pages/groups.svelte'),
        },
        "/router": {
            page: () => import('./pages/routes.svelte'),
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
