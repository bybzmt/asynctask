import { writable } from 'svelte/store';

const url = writable({});

function get_base_uri(doc) {
  let baseURI = doc.baseURI;

  if (!baseURI) {
    const baseTags = doc.getElementsByTagName('base');
    baseURI = baseTags.length ? baseTags[0].href : doc.URL;
  }

  return baseURI;
}

function goto(href) {
  const _url = new URL(href, get_base_uri(document));

  history.pushState({}, "", _url.href)
  url.update(() => _url)
}

function href(n) {
  return () => {
    goto(n)
  }
}

export { url, goto, href }
