import { api, errorText } from '../api.js';
import { state, setFlash, takeFlash } from '../state.js';
import { h, messageBox, textInput, field, form, withBusy, rulesList } from '../ui.js';
import { link, navigate } from '../router.js';
import { formatBytes, formatDate } from '../format.js';
import { passwordChecks, passwordError } from '../validate.js';

const GIB = 1024 ** 3;

export const statusLabels = {
  PENDING: 'čeka obradu',
  APPROVED: 'odobren',
  REJECTED: 'odbijen',
};

function timeLeftText(ms) {
  const minutes = Math.floor(ms / 60000);
  if (minutes < 60) return Math.max(minutes, 1) + ' min';
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return hours + ' h';
  return Math.floor(hours / 24) + ' dana';
}

export function accountPage(root) {
  const box = h('section', { class: 'card' }, h('h2', {}, 'Moj nalog'));
  const msg = messageBox();
  const info = h('div', {});
  box.append(msg.el, info);

  const quotaCard = h('section', { class: 'card' });
  root.append(box, quotaCard);

  api('GET', '/api/users/me').then((r) => {
    if (!r.ok) {
      if (state.token) msg.error(errorText(r, 'Podaci o nalogu nisu dostupni'));
      return;
    }
    const u = r.data;
    const left = new Date(u.passwordExpiresAt).getTime() - Date.now();
    if (left <= 0) msg.error('Lozinka je istekla. Promenite je što pre.');
    else if (left <= 7 * 24 * 3600 * 1000) msg.warn('Lozinka ističe za ' + timeLeftText(left) + '. Promenite je na vreme.');

    info.append(
      h(
        'dl',
        {},
        h('dt', {}, 'Ime i prezime'),
        h('dd', {}, u.firstName + ' ' + u.lastName),
        h('dt', {}, 'Imejl'),
        h('dd', {}, u.email),
        h('dt', {}, 'Uloga'),
        h('dd', {}, u.role === 'A' ? 'Administrator' : 'Redovan korisnik'),
        h('dt', {}, 'Skladište'),
        h('dd', {}, h('progress', { value: u.usedBytes, max: u.quotaBytes }), h('div', {}, formatBytes(u.usedBytes) + ' / ' + formatBytes(u.quotaBytes))),
        h('dt', {}, 'Lozinka važi do'),
        h('dd', {}, formatDate(u.passwordExpiresAt)),
      ),
      h('div', { class: 'links' }, link('/change-password', 'Promena lozinke')),
    );
    renderQuota(quotaCard, u);
  });
}

function renderQuota(card, user) {
  const msg = messageBox();
  const list = h('div', {});
  const gb = textInput('requestedGb', 'number', { min: '0', step: '0.5', inputmode: 'decimal' });
  const button = h('button', { type: 'submit', class: 'primary' }, 'Pošalji zahtev');

  const loadRequests = async () => {
    const r = await api('GET', '/api/users/quota-requests');
    if (!r.ok) {
      list.replaceChildren();
      return;
    }
    if (r.data.length === 0) {
      list.replaceChildren(h('p', {}, 'Nemate podnetih zahteva.'));
      return;
    }
    list.replaceChildren(
      h(
        'table',
        {},
        h('thead', {}, h('tr', {}, h('th', {}, 'Tražena kvota'), h('th', {}, 'Status'), h('th', {}, 'Datum'))),
        h(
          'tbody',
          {},
          r.data.map((q) =>
            h('tr', {}, h('td', {}, formatBytes(q.requestedBytes)), h('td', {}, statusLabels[q.status] || q.status), h('td', {}, formatDate(q.createdAt))),
          ),
        ),
      ),
    );
  };

  const onSubmit = async () => {
    msg.clear();
    const value = Number(gb.value);
    if (!Number.isFinite(value) || value <= 0) {
      msg.error('Unesite željenu kvotu u GB');
      return;
    }
    const bytes = Math.round(value * GIB);
    if (bytes <= user.quotaBytes) {
      msg.error('Tražena kvota mora biti veća od trenutne (' + formatBytes(user.quotaBytes) + ')');
      return;
    }
    await withBusy(button, async () => {
      const r = await api('POST', '/api/users/quota-requests', { requestedBytes: bytes });
      if (!r.ok) {
        msg.error(errorText(r, 'Zahtev nije uspeo'));
        return;
      }
      msg.ok('Zahtev je poslat administratoru.');
      gb.value = '';
      await loadRequests();
    });
  };

  card.replaceChildren(
    h('h2', {}, 'Zahtev za povećanje kvote'),
    h('p', {}, 'Trenutna kvota: ' + formatBytes(user.quotaBytes)),
    msg.el,
    form(onSubmit, field('Željena kvota (GB)', gb), button),
    h('h3', {}, 'Moji zahtevi'),
    list,
  );
  loadRequests();
}

export function changePasswordPage(root) {
  const msg = messageBox();
  const flash = takeFlash();
  if (flash) msg.warn(flash);

  const current = textInput('currentPassword', 'password', { autocomplete: 'current-password' });
  const next = textInput('newPassword', 'password', { autocomplete: 'new-password' });
  const confirm = textInput('confirmNewPassword', 'password', { autocomplete: 'new-password' });
  const button = h('button', { type: 'submit', class: 'primary' }, 'Promeni lozinku');

  const rules = h('div', {});
  const updateRules = () =>
    rules.replaceChildren(rulesList(passwordChecks(next.value, state.user ? state.user.email : '')));
  next.addEventListener('input', updateRules);
  updateRules();

  const onSubmit = async () => {
    msg.clear();
    const problem =
      (current.value ? '' : 'Unesite trenutnu lozinku') ||
      passwordError(next.value, state.user ? state.user.email : '') ||
      (next.value !== confirm.value ? 'Nova lozinka i ponovljena lozinka se ne poklapaju' : '');
    if (problem) {
      msg.error(problem);
      return;
    }
    await withBusy(button, async () => {
      const r = await api('POST', '/api/users/password/change', {
        currentPassword: current.value,
        newPassword: next.value,
        confirmNewPassword: confirm.value,
      });
      if (!r.ok) {
        msg.error(errorText(r, 'Promena lozinke nije uspela'));
        return;
      }
      setFlash('Lozinka je promenjena. Prijavite se ponovo.');
      navigate('/login', true);
      state.clear();
    });
  };

  root.append(
    h(
      'section',
      { class: 'card' },
      h('h2', {}, 'Promena lozinke'),
      msg.el,
      form(
        onSubmit,
        field('Trenutna lozinka', current),
        field('Nova lozinka', next),
        rules,
        field('Ponovite novu lozinku', confirm),
        button,
      ),
      h('div', { class: 'links' }, link('/account', 'Nazad na nalog')),
    ),
  );
}
