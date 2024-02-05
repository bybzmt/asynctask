
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
        "/config": {
            page: () => import('./pages/config.svelte'),
        },
        "/timed": {
            page: () => import('./pages/timed.svelte'),
        },
    }
}
