import Msg from './msg.svelte'

function match(routes, url) {
  let uri = url.hash

  if (uri == "") {
    uri = "/"
  } else {
    uri = uri.substring(1)
  }

  let page = routes.map[uri];
  if (page) {
    return page
  }

  if (routes.error) return routes.error

  return null
}

async function loadPage(context, routes, url) {
  let res = match(routes, url)

  if (!res) {
    return {
      status: 404,
      render: Msg,
      props: { msg: "404 Not Found." }
    }
  }

  let p = await res.page()

  let input = {
    url,
    query: url.searchParams,
  }

  let resp;
  resp = p.load ? await p.load(input) : {}

  if (resp instanceof Error) {
    return {
      status: 404,
      render: Msg,
      props: { msg: resp.message }
    }
  }

  return {
    status: resp.status || 200,
    redirect: resp.redirect,
    headers: resp.headers,
    render: p.default,
    props: resp.props,
  }
}

function goto(context) {
  return function (href) {
    let url = context.get('url');

    url.update((_url) => {
      let x = new URL(href, _url);
      history.pushState({}, "", x.href)
      return x
    })
  }
}


export { goto, match, loadPage }
