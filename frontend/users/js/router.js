import { state } from './state.js';
import { h } from './ui.js';

let routes = {};
let afterRender = () => {};

export function setRoutes(map) {
  routes = map;
}

export function getRoutes() {
  return routes;
}

export function onRender(fn) {
  afterRender = fn;
}

export function navigate(path, replace = false) {
  if (replace) history.replaceState({}, '', path);
  else history.pushState({}, '', path);
  render();
}

export function render() {
  let path = location.pathname;
  if (path === '/') path = state.token ? '/account' : '/login';
  const route = routes[path] || routes['/login'];

  if (route.auth && !state.token) {
    history.replaceState({}, '', '/login');
    return render();
  }
  if (route.admin && !(state.user && state.user.role === 'A')) {
    history.replaceState({}, '', '/account');
    return render();
  }

  const root = document.getElementById('app');
  root.replaceChildren();
  route.page(root);
  afterRender();
}

window.addEventListener('popstate', render);

export function link(path, text) {
  return h(
    'a',
    {
      href: path,
      onclick: (event) => {
        event.preventDefault();
        navigate(path);
      },
    },
    text,
  );
}
