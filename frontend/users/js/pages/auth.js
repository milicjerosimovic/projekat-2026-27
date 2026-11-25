import { api, errorText } from '../api.js';
import { state, setFlash, takeFlash } from '../state.js';
import { h, messageBox, textInput, field, form, withBusy, rulesList } from '../ui.js';
import { link, navigate } from '../router.js';
import { emailError, nameError, passwordChecks, passwordError, codeError } from '../validate.js';

const confirmResults = new Map();
const magicResults = new Set();

function card(title, ...children) {
  return h('section', { class: 'card' }, h('h2', {}, title), ...children);
}

function tokenFromUrl() {
  return new URLSearchParams(location.search).get('token') || '';
}

function passwordRules(passwordInput, emailGetter) {
  const holder = h('div', {});
  const update = () => holder.replaceChildren(rulesList(passwordChecks(passwordInput.value, emailGetter())));
  passwordInput.addEventListener('input', update);
  update();
  return holder;
}

export function registerPage(root) {
  const msg = messageBox();
  const first = textInput('firstName', 'text', { autocomplete: 'given-name', maxlength: 50 });
  const last = textInput('lastName', 'text', { autocomplete: 'family-name', maxlength: 50 });
  const email = textInput('email', 'email', { autocomplete: 'email', maxlength: 254 });
  const password = textInput('password', 'password', { autocomplete: 'new-password' });
  const confirm = textInput('confirmPassword', 'password', { autocomplete: 'new-password' });
  const button = h('button', { type: 'submit', class: 'primary' }, 'Registruj se');

  const onSubmit = async () => {
    msg.clear();
    const problem =
      nameError('Ime', first.value) ||
      nameError('Prezime', last.value) ||
      emailError(email.value) ||
      passwordError(password.value, email.value) ||
      (password.value !== confirm.value ? 'Lozinka i ponovljena lozinka se ne poklapaju' : '');
    if (problem) {
      msg.error(problem);
      return;
    }
    await withBusy(button, async () => {
      const r = await api('POST', '/api/users/register', {
        firstName: first.value.trim(),
        lastName: last.value.trim(),
        email: email.value.trim(),
        password: password.value,
        confirmPassword: confirm.value,
      });
      if (!r.ok) {
        msg.error(errorText(r, 'Registracija nije uspela'));
        return;
      }
      msg.ok(r.data.message || 'Registracija je uspešna. Proverite imejl i potvrdite nalog.');
      password.value = '';
      confirm.value = '';
    });
  };

  root.append(
    card(
      'Registracija',
      msg.el,
      form(
        onSubmit,
        field('Ime', first),
        field('Prezime', last),
        field('Imejl', email),
        field('Lozinka', password),
        passwordRules(password, () => email.value),
        field('Ponovite lozinku', confirm),
        button,
      ),
      h('div', { class: 'links' }, link('/login', 'Već imate nalog? Prijava')),
    ),
  );
}

export function confirmPage(root) {
  const msg = messageBox();
  root.append(card('Potvrda naloga', msg.el, h('div', { class: 'links' }, link('/login', 'Prijava'))));

  const token = tokenFromUrl();
  if (!token) {
    msg.error('Link za potvrdu nije ispravan');
    return;
  }
  const show = (r) => (r.ok ? msg.ok(r.data.message || 'Nalog je aktiviran.') : msg.error(errorText(r, 'Potvrda nije uspela')));
  if (confirmResults.has(token)) {
    show(confirmResults.get(token));
    return;
  }
  api('POST', '/api/users/confirm', { token }).then((r) => {
    confirmResults.set(token, r);
    show(r);
  });
}

function loginErrorText(r) {
  if (r.data && r.data.code === 'not_confirmed') return 'Nalog nije potvrđen. Proverite imejl.';
  if (r.data && r.data.code === 'password_expired') {
    return 'Lozinka je istekla. Prijava je onemogućena dok ne postavite novu lozinku preko "Zaboravljena lozinka" ili "Povrati nalog".';
  }
  return errorText(r, 'Prijava nije uspela');
}

export function loginPage(root) {
  const box = h('section', { class: 'card' });
  root.append(box);
  showCredentials(box, takeFlash());
}

function showCredentials(box, flash) {
  const msg = messageBox();
  if (flash) msg.ok(flash);
  const email = textInput('email', 'email', { autocomplete: 'username', maxlength: 254 });
  const password = textInput('password', 'password', { autocomplete: 'current-password' });
  const button = h('button', { type: 'submit', class: 'primary' }, 'Nastavi');

  const onSubmit = async () => {
    msg.clear();
    const problem = emailError(email.value) || (password.value ? '' : 'Unesite lozinku');
    if (problem) {
      msg.error(problem);
      return;
    }
    await withBusy(button, async () => {
      const r = await api('POST', '/api/users/login', { email: email.value.trim(), password: password.value });
      if (!r.ok) {
        msg.error(loginErrorText(r));
        return;
      }
      showOtp(box, email.value.trim());
    });
  };

  box.replaceChildren(
    h('h2', {}, 'Prijava'),
    msg.el,
    form(onSubmit, field('Imejl', email), field('Lozinka', password), button),
    h(
      'div',
      { class: 'links' },
      link('/register', 'Registracija'),
      link('/forgot-password', 'Zaboravljena lozinka'),
      link('/magic-link', 'Povrati nalog'),
    ),
  );
}

function showOtp(box, email) {
  const msg = messageBox();
  msg.ok('Jednokratni kod je poslat na vaš imejl.');
  const code = textInput('code', 'text', { inputmode: 'numeric', maxlength: 6, autocomplete: 'one-time-code' });
  const button = h('button', { type: 'submit', class: 'primary' }, 'Prijavi se');

  const onSubmit = async () => {
    const problem = codeError(code.value);
    if (problem) {
      msg.error(problem);
      return;
    }
    await withBusy(button, async () => {
      const r = await api('POST', '/api/users/login/otp', { email, code: code.value.trim() });
      if (!r.ok) {
        msg.error(errorText(r, 'Kod nije ispravan'));
        return;
      }
      state.set(r.data.token, r.data.user);
      navigate('/account', true);
    });
  };

  box.replaceChildren(
    h('h2', {}, 'Jednokratni kod'),
    msg.el,
    form(onSubmit, field('Kod sa imejla (6 cifara)', code), button),
    h('div', { class: 'links' }, h('a', { href: '/login', onclick: (e) => { e.preventDefault(); showCredentials(box, ''); } }, 'Nazad')),
  );
}

export function forgotPage(root) {
  const msg = messageBox();
  const email = textInput('email', 'email', { autocomplete: 'email', maxlength: 254 });
  const button = h('button', { type: 'submit', class: 'primary' }, 'Pošalji link');
  const onSubmit = async () => {
    msg.clear();
    const problem = emailError(email.value);
    if (problem) {
      msg.error(problem);
      return;
    }
    await withBusy(button, async () => {
      const r = await api('POST', '/api/users/password/forgot', { email: email.value.trim() });
      if (r.ok) msg.ok(r.data.message || 'Ako nalog postoji, poslat je imejl sa linkom.');
      else msg.error(errorText(r, 'Zahtev nije uspeo'));
    });
  };
  root.append(
    card(
      'Zaboravljena lozinka',
      msg.el,
      form(onSubmit, field('Imejl', email), button),
      h('div', { class: 'links' }, link('/login', 'Nazad na prijavu')),
    ),
  );
}

export function resetPage(root) {
  const msg = messageBox();
  const token = tokenFromUrl();
  const password = textInput('newPassword', 'password', { autocomplete: 'new-password' });
  const confirm = textInput('confirmNewPassword', 'password', { autocomplete: 'new-password' });
  const button = h('button', { type: 'submit', class: 'primary' }, 'Postavi lozinku');
  if (!token) msg.error('Link za reset nije ispravan');

  const onSubmit = async () => {
    msg.clear();
    const problem =
      passwordError(password.value, '') ||
      (password.value !== confirm.value ? 'Lozinka i ponovljena lozinka se ne poklapaju' : '');
    if (!token) {
      msg.error('Link za reset nije ispravan');
      return;
    }
    if (problem) {
      msg.error(problem);
      return;
    }
    await withBusy(button, async () => {
      const r = await api('POST', '/api/users/password/reset', {
        token,
        newPassword: password.value,
        confirmNewPassword: confirm.value,
      });
      if (!r.ok) {
        msg.error(errorText(r, 'Reset nije uspeo'));
        return;
      }
      msg.ok(r.data.message || 'Lozinka je promenjena.');
      password.value = '';
      confirm.value = '';
    });
  };

  root.append(
    card(
      'Nova lozinka',
      msg.el,
      form(
        onSubmit,
        field('Nova lozinka', password),
        passwordRules(password, () => ''),
        field('Ponovite lozinku', confirm),
        button,
      ),
      h('div', { class: 'links' }, link('/login', 'Prijava')),
    ),
  );
}

export function magicRequestPage(root) {
  const msg = messageBox();
  const email = textInput('email', 'email', { autocomplete: 'email', maxlength: 254 });
  const button = h('button', { type: 'submit', class: 'primary' }, 'Pošalji magični link');
  const onSubmit = async () => {
    msg.clear();
    const problem = emailError(email.value);
    if (problem) {
      msg.error(problem);
      return;
    }
    await withBusy(button, async () => {
      const r = await api('POST', '/api/users/magic-link/request', { email: email.value.trim() });
      if (r.ok) msg.ok(r.data.message || 'Ako nalog postoji, poslat je imejl sa magičnim linkom.');
      else msg.error(errorText(r, 'Zahtev nije uspeo'));
    });
  };
  root.append(
    card(
      'Povraćaj naloga',
      h('p', {}, 'Unesite imejl i poslaćemo link kojim se prijavljujete bez lozinke.'),
      msg.el,
      form(onSubmit, field('Imejl', email), button),
      h('div', { class: 'links' }, link('/login', 'Nazad na prijavu')),
    ),
  );
}

export function magicLoginPage(root) {
  const msg = messageBox();
  root.append(card('Prijava magičnim linkom', msg.el, h('div', { class: 'links' }, link('/login', 'Prijava'))));

  const token = tokenFromUrl();
  if (!token) {
    msg.error('Magični link nije ispravan');
    return;
  }
  if (magicResults.has(token)) {
    msg.error('Ovaj link je već iskorišćen');
    return;
  }
  magicResults.add(token);
  api('POST', '/api/users/magic-link/login', { token }).then((r) => {
    if (!r.ok) {
      msg.error(errorText(r, 'Prijava nije uspela'));
      return;
    }
    state.set(r.data.token, r.data.user);
    if (r.data.passwordExpired) {
      setFlash('Lozinka je istekla. Postavite novu lozinku.');
      navigate('/change-password', true);
    } else {
      navigate('/account', true);
    }
  });
}
