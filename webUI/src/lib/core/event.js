

/**
 * @param {Event} event
 * @returns {HTMLAnchorElement | SVGAElement | undefined}
 */
function find_anchor(event) {
  const node = event
    .composedPath()
    .find((e) => e instanceof Node && e.nodeName.toUpperCase() === 'A'); // SVG <a> elements have a lowercase name
  return /** @type {HTMLAnchorElement | SVGAElement | undefined} */ (node);
}

/**
 * @param {HTMLAnchorElement | SVGAElement} node
 * @returns {URL}
 */
function get_href(node) {
  return node instanceof SVGAElement
    ? new URL(node.href.baseVal, document.baseURI)
    : new URL(node.href);
}

function owns(url) {
  let base = '/'
  return url.origin === location.origin && url.pathname.startsWith(base);
}

function initEvent(context) {
  window.addEventListener('click', (event) => {

    if (event.button) return;
    if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
    if (event.defaultPrevented) return;


    const a = find_anchor(event);
    if (!a) return;

    if (!a.href) return;

    const url = get_href(a);
    const url_string = url.toString();
    if (url_string === location.href) {
      if (!location.hash) event.preventDefault();
      return;
    }

    // Ignore if tag has
    // 0. 'download' attribute
    // 1. 'rel' attribute includes external
    const rel = (a.getAttribute('rel') || '').split(/\s+/);

    if (a.hasAttribute('download') || (rel && rel.includes('external'))) {
      return;
    }

    // Ignore if <a> has a target
    if (a instanceof SVGAElement ? a.target.baseVal : a.target) return;

    //判断是不是属于同域名的链接
    if (!owns(url)) return;

    const i1 = url_string.indexOf('#');
    const i2 = location.href.indexOf('#');
    const u0 = i1 >= 0 ? url_string.substring(0, i1) : url_string;
    const u1 = i2 >= 0 ? location.href.substring(0, i2) : location.href;

    //history.pushState({}, '', url.href);

    let goto = context.get('goto')
    goto(url.href)

    if (u0 === u1) {
      window.dispatchEvent(new HashChangeEvent('hashchange'));
    }

    event.preventDefault();
  });

  window.addEventListener('popstate', (event) => {
    let _url = new URL(location.href);

    let url = context.get('url')
    url.set(_url);
  });
}

export { initEvent }
