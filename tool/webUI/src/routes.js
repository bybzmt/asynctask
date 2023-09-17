
export default {
    map: {
        "/": {
            page: () => import('./pages/index.svelte'),
        },
        "/groups": {
            page: () => import('./pages/groups.svelte'),
        },
        "/routes": {
            page: () => import('./pages/routes.svelte'),
        },
    }
}
