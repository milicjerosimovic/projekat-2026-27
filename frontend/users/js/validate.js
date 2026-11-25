export function emailError(value) {
  const v = String(value || '').trim();
  if (v.length === 0 || v.length > 254 || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v)) {
    return 'Imejl adresa nije ispravna';
  }
  return '';
}

export function nameError(label, value) {
  const v = String(value || '').trim();
  const length = [...v].length;
  if (length < 1 || length > 50) return label + ' mora imati između 1 i 50 karaktera';
  if (!/^\p{L}[\p{L} .'\-]*$/u.test(v)) {
    return label + ' sme da sadrži samo slova, razmak, tačku, apostrof i crticu';
  }
  return '';
}

export function passwordChecks(password, email) {
  const pw = String(password || '');
  const local = String(email || '').split('@')[0].toLowerCase();
  return [
    { ok: [...pw].length >= 10, text: 'Najmanje 10 karaktera' },
    { ok: new TextEncoder().encode(pw).length <= 72, text: 'Najviše 72 bajta' },
    { ok: /\p{Lu}/u.test(pw), text: 'Veliko slovo' },
    { ok: /\p{Ll}/u.test(pw), text: 'Malo slovo' },
    { ok: /\p{Nd}/u.test(pw), text: 'Cifra' },
    { ok: /[\p{P}\p{S}]/u.test(pw), text: 'Specijalni karakter' },
    { ok: !(local.length >= 3 && pw.toLowerCase().includes(local)), text: 'Ne sadrži deo imejla' },
  ];
}

export function passwordError(password, email) {
  const failed = passwordChecks(password, email).filter((c) => !c.ok);
  if (failed.length === 0) return '';
  return 'Lozinka ne ispunjava uslove: ' + failed.map((c) => c.text.toLowerCase()).join(', ');
}

export function codeError(value) {
  return /^\d{6}$/.test(String(value || '').trim()) ? '' : 'Kod mora imati tačno 6 cifara';
}
