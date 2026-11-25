import { state } from './state.js';

export async function api(method, path, body) {
  const headers = { Accept: 'application/json' };
  if (body !== undefined) headers['Content-Type'] = 'application/json';
  if (state.token) headers.Authorization = 'Bearer ' + state.token;

  let res;
  try {
    res = await fetch(path, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  } catch {
    return { ok: false, status: 0, data: { error: 'Server nije dostupan' } };
  }

  let data = null;
  if (res.status !== 204) {
    try {
      data = await res.json();
    } catch {
      data = null;
    }
  }
  if (res.status === 401 && state.token) state.clear();
  return { ok: res.ok, status: res.status, data: data || {} };
}

export function errorText(result, fallback) {
  if (result.status === 429) return 'Previše zahteva. Sačekajte malo pa pokušajte ponovo.';
  return (result.data && result.data.error) || fallback;
}
