import test from 'node:test';
import assert from 'node:assert/strict';
import { emailError, nameError, passwordChecks, passwordError, codeError } from '../js/validate.js';
import { formatBytes } from '../js/format.js';

test('jaka lozinka prolazi, slabe ne prolaze', () => {
  assert.equal(passwordError('Str0ng!Passw0rd', 'pera@example.com'), '');
  for (const weak of ['Ab1!', 'alllowercase1!', 'ALLUPPERCASE1!', 'NoDigitsHere!!', 'NoSpecial12345']) {
    assert.notEqual(passwordError(weak, 'pera@example.com'), weak);
    assert.ok(passwordError(weak, 'pera@example.com').length > 0, weak);
  }
});

test('lozinka ne sme da sadrzi deo imejla', () => {
  assert.ok(passwordError('Marko123!Marko', 'marko@example.com').length > 0);
});

test('lozinka duza od 72 bajta je odbijena', () => {
  assert.ok(passwordError('Aa1!' + 'x'.repeat(80), 'a@b.com').length > 0);
});

test('provera uslova vraca sve stavke', () => {
  assert.equal(passwordChecks('x', '').length, 7);
});

test('imejl', () => {
  assert.equal(emailError('pera@example.com'), '');
  for (const bad of ['', 'bez-et.com', 'a@b', 'a b@c.com']) assert.notEqual(emailError(bad), '', bad);
});

test('ime dozvoljava slova sa dijakriticima, ne HTML', () => {
  assert.equal(nameError('Ime', 'Đorđe'), '');
  assert.notEqual(nameError('Ime', '<script>alert(1)</script>'), '');
});

test('kod ima tacno 6 cifara', () => {
  assert.equal(codeError('123456'), '');
  assert.notEqual(codeError('12345'), '');
  assert.notEqual(codeError('12a456'), '');
});

test('formatiranje velicine', () => {
  assert.equal(formatBytes(0), '0 B');
  assert.equal(formatBytes(1024 ** 3), '1.0 GB');
});
