import { state } from './state.js';
import { api } from './api.js';
import { h } from './ui.js';
import { setRoutes, getRoutes, onRender, render, navigate, link } from './router.js';
import {
  registerPage,
  confirmPage,
  loginPage,
  forgotPage,
  resetPage,
  magicRequestPage,
  magicLoginPage,
} from './pages/auth.js';
import { accountPage, changePasswordPage } from './pages/account.js';
import { adminPage } from './pages/admin.js';

setRoutes({
  '/login': { page: loginPage },
  '/register': { page: registerPage },
  '/confirm': { page: confirmPage },
  '/forgot-password': { page: forgotPage },
  '/reset-password': { page: resetPage },
  '/magic-link': { page: magicRequestPage },
  '/magic-login': { page: magicLoginPage },
  '/account': { page: accountPage, auth: true },
  '/change-password': { page: changePasswordPage, auth: true },
  '/admin': { page: adminPage, auth: true, admin: true },
});

async function logout() {
  await api('POST', '/api/users/logout');
  navigate('/login', true);
  state.clear();
}

function renderNav() {
  const items = [h('span', { class: 'brand' }, 'Drive')];
  if (state.token) {
    items.push(link('/account', 'Moj nalog'), link('/change-password', 'Promena lozinke'));
    if (state.user && state.user.role === 'A') items.push(link('/admin', 'Administracija'));
    items.push(
      h('span', { class: 'spacer' }),
      h('span', { class: 'who' }, state.user ? state.user.firstName + ' ' + state.user.lastName : ''),
      h('button', { class: 'link', type: 'button', onclick: logout }, 'Odjava'),
    );
  } else {
    items.push(link('/login', 'Prijava'), link('/register', 'Registracija'), link('/magic-link', 'Povrati nalog'));
  }
  document.getElementById('nav').replaceChildren(...items);
}

onRender(renderNav);
state.onChange(() => {
  renderNav();
  const route = getRoutes()[location.pathname];
  if (!state.token && route && route.auth) navigate('/login', true);
});

render();
