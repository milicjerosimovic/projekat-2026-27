import { api, errorText } from '../api.js';
import { h, messageBox } from '../ui.js';
import { formatBytes, formatDate } from '../format.js';
import { statusLabels } from './account.js';

const PAGE_SIZE = 10;

export function adminPage(root) {
  const msg = messageBox();
  const usersTable = h('div', {});
  const pager = h('div', { class: 'pager' });
  const requestsTable = h('div', {});
  let status = 'PENDING';
  let page = 1;

  const select = h(
    'select',
    {
      id: 'statusFilter',
      onchange: (event) => {
        status = event.target.value;
        loadRequests();
      },
    },
    h('option', { value: 'PENDING', selected: true }, 'Na čekanju'),
    h('option', { value: 'APPROVED' }, 'Odobreni'),
    h('option', { value: 'REJECTED' }, 'Odbijeni'),
    h('option', { value: '' }, 'Svi'),
  );

  async function loadUsers() {
    const r = await api('GET', '/api/users/admin/users?page=' + page + '&size=' + PAGE_SIZE);
    if (!r.ok) {
      msg.error(errorText(r, 'Lista korisnika nije dostupna'));
      return;
    }
    const { items, total } = r.data;
    usersTable.replaceChildren(
      h(
        'table',
        {},
        h('thead', {}, h('tr', {}, ['Ime', 'Prezime', 'Imejl', 'Uloga', 'Status', 'Zauzeto / kvota'].map((t) => h('th', {}, t)))),
        h(
          'tbody',
          {},
          items.map((u) =>
            h(
              'tr',
              {},
              h('td', {}, u.firstName),
              h('td', {}, u.lastName),
              h('td', {}, u.email),
              h('td', {}, u.role === 'A' ? 'Administrator' : 'Korisnik'),
              h('td', {}, h('span', { class: 'tag ' + (u.active ? 'on' : 'off') }, u.active ? 'aktivan' : 'nepotvrđen')),
              h('td', {}, formatBytes(u.usedBytes) + ' / ' + formatBytes(u.quotaBytes)),
            ),
          ),
        ),
      ),
    );
    const pages = Math.max(1, Math.ceil(total / PAGE_SIZE));
    pager.replaceChildren(
      h('button', { class: 'small', type: 'button', disabled: page <= 1, onclick: () => { page -= 1; loadUsers(); } }, 'Prethodna'),
      h('span', {}, 'Strana ' + page + ' od ' + pages + ' (ukupno ' + total + ')'),
      h('button', { class: 'small', type: 'button', disabled: page >= pages, onclick: () => { page += 1; loadUsers(); } }, 'Sledeća'),
    );
  }

  async function decide(request, action, buttons) {
    buttons.forEach((b) => { b.disabled = true; });
    const r = await api('POST', '/api/users/admin/quota-requests/' + encodeURIComponent(request.id) + '/' + action);
    if (!r.ok) msg.error(errorText(r, 'Obrada zahteva nije uspela'));
    else msg.clear();
    await Promise.all([loadRequests(), loadUsers()]);
  }

  async function loadRequests() {
    const r = await api('GET', '/api/users/admin/quota-requests' + (status ? '?status=' + status : ''));
    if (!r.ok) {
      msg.error(errorText(r, 'Lista zahteva nije dostupna'));
      return;
    }
    if (r.data.length === 0) {
      requestsTable.replaceChildren(h('p', {}, 'Nema zahteva za izabrani filter.'));
      return;
    }
    requestsTable.replaceChildren(
      h(
        'table',
        {},
        h('thead', {}, h('tr', {}, ['Korisnik', 'Tražena kvota', 'Status', 'Datum', 'Akcija'].map((t) => h('th', {}, t)))),
        h(
          'tbody',
          {},
          r.data.map((q) => {
            const approve = h('button', { class: 'small', type: 'button' }, 'Odobri');
            const reject = h('button', { class: 'small', type: 'button' }, 'Odbij');
            approve.addEventListener('click', () => decide(q, 'approve', [approve, reject]));
            reject.addEventListener('click', () => decide(q, 'reject', [approve, reject]));
            return h(
              'tr',
              {},
              h('td', {}, q.userEmail),
              h('td', {}, formatBytes(q.requestedBytes)),
              h('td', {}, statusLabels[q.status] || q.status),
              h('td', {}, formatDate(q.createdAt)),
              h('td', {}, q.status === 'PENDING' ? [approve, ' ', reject] : ''),
            );
          }),
        ),
      ),
    );
  }

  root.append(
    h(
      'section',
      { class: 'card wide' },
      h('h2', {}, 'Zahtevi za povećanje kvote'),
      msg.el,
      h('label', { for: 'statusFilter' }, 'Prikaži'),
      select,
      requestsTable,
    ),
    h('section', { class: 'card wide' }, h('h2', {}, 'Korisnici, kvota i zauzeće'), usersTable, pager),
  );

  loadRequests();
  loadUsers();
}
