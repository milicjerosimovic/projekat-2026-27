const listeners = new Set();
let flash = '';

export const state = {
  token: null,
  user: null,
  set(token, user) {
    this.token = token;
    this.user = user;
    listeners.forEach((f) => f());
  },
  clear() {
    this.token = null;
    this.user = null;
    listeners.forEach((f) => f());
  },
  onChange(f) {
    listeners.add(f);
  },
};

export function setFlash(text) {
  flash = text;
}

export function takeFlash() {
  const text = flash;
  flash = '';
  return text;
}
