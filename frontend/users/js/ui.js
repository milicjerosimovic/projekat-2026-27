export function h(tag, attrs, ...children) {
  const el = document.createElement(tag);
  for (const [key, value] of Object.entries(attrs || {})) {
    if (value === false || value === null || value === undefined) continue;
    if (key.startsWith('on') && typeof value === 'function') {
      el.addEventListener(key.slice(2).toLowerCase(), value);
    } else if (value === true) {
      el.setAttribute(key, '');
    } else {
      el.setAttribute(key, String(value));
    }
  }
  for (const child of children.flat()) {
    if (child === null || child === undefined || child === false) continue;
    el.append(child instanceof Node ? child : document.createTextNode(String(child)));
  }
  return el;
}

export function messageBox() {
  const el = h('div', { role: 'alert' });
  const show = (kind, text) => {
    el.className = 'msg ' + kind;
    el.textContent = text;
    el.hidden = false;
  };
  el.hidden = true;
  return {
    el,
    error: (text) => show('error', text),
    ok: (text) => show('ok', text),
    warn: (text) => show('warn', text),
    clear: () => {
      el.hidden = true;
      el.textContent = '';
    },
  };
}

export function textInput(id, type, extra) {
  return h('input', { id, name: id, type, ...(extra || {}) });
}

export function field(labelText, input) {
  return h('div', {}, h('label', { for: input.id }, labelText), input);
}

export function form(onSubmit, ...children) {
  return h(
    'form',
    {
      novalidate: true,
      onsubmit: async (event) => {
        event.preventDefault();
        await onSubmit();
      },
    },
    ...children,
  );
}

export async function withBusy(button, task) {
  button.disabled = true;
  try {
    return await task();
  } finally {
    button.disabled = false;
  }
}

export function rulesList(checks) {
  return h(
    'ul',
    { class: 'rules' },
    checks.map((c) => h('li', { class: c.ok ? 'ok' : '' }, c.text)),
  );
}
