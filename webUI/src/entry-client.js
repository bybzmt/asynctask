import { goto, loadPage } from '$src/lib/core';
import { initEvent } from '$src/lib/core/event';
import routes from '$src/routes'
import { writable } from 'svelte/store';

async function render(target) {

    let _url = new URL(location.href);

    let url = writable(_url);

    let session = writable({});

    let context = new Map()

    context.set('url', url)
    context.set('session', session)
    context.set('goto', goto(context))

    let app;

    function onload(page) {
        if (!page) {
            return;
        }

        if (page.redirect) {
            let goto = context.get('goto')
            goto(page.redirect)
            return;
        }

        if (page.headers) {
            for (let key in page.headers) {
                document.cookie = cookie.serialize(key, String(page.header[key]), { sameSite: 'lax', path: "/" })
            }
        }

        if (app instanceof page.render) {
            app.$set(page.props)
        } else {
            if (app) {
                app.$destroy()
            }

            app = new page.render({
                target,
                hydrate: true,
                props: page.props,
                context,
            });
        }
    }

    url.subscribe(url => {
        _url = url
        loadPage(context, routes, url).then(onload)
    });

    initEvent(context)
}


render(document.querySelector("body div"))
