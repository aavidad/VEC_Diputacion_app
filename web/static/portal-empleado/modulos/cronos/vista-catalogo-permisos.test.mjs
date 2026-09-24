import assert from "node:assert/strict";
import { test } from "node:test";
import {
  filtrarCatalogoPermisosCronos,
  montarCatalogoPermisosCronos,
  renderizarCatalogoPermisosCronos,
} from "./vista-catalogo-permisos.js";
import { MENSAJES_CRONOS_C6_ES } from "./i18n-c6.js";

test("la referencia reproduce tipos de la ficha sin cuantías ni saldos personales", () => {
  const html = renderizarCatalogoPermisosCronos();
  assert.equal((html.match(/data-cronos-c6-tipo=/gu) || []).length, 25);
  for (const nombre of ["Asistencia a exámenes", "Bolsa horaria por conciliación", "Horas de médico", "Vacaciones"]) {
    assert.ok(html.includes(nombre), nombre);
  }
  for (const campo of ["Unidad", "Cómputo", "Máximo anual o mensual", "Mínimo", "Quién autoriza", "Justificante exigido"]) {
    assert.ok(html.includes(`<dt>${campo}</dt><dd>Pendiente de confirmación por RRHH</dd>`), campo);
  }
  assert.match(html, /Ficha de requisitos de Cronos, 23\/09\/2026; nombres vistos en WCronos/u);
  assert.match(html, /no habilita la solicitud de permisos/u);
  assert.doesNotMatch(html, /6 días|30 h|22 días|saldo disponible|<form|<button|fetch\(/u);
});

test("la búsqueda admite tildes, muestra vacío y escapa texto del catálogo", () => {
  assert.deepEqual(filtrarCatalogoPermisosCronos("meDIco"), ["horas_medico"]);
  assert.deepEqual(filtrarCatalogoPermisosCronos("  SÁBADOS "), ["compensacion_sabados"]);
  const html = renderizarCatalogoPermisosCronos({ consulta: 'nada"><script>' });
  assert.match(html, /Tipos mostrados: 0/u);
  assert.match(html, /data-cronos-c6-vacio role="status">No hay tipos/u);
  assert.doesNotMatch(html, /<script>/u);
  const modificado = renderizarCatalogoPermisosCronos({ mensajes: {
    ...MENSAJES_CRONOS_C6_ES,
    tipo_vacaciones: '<img src=x onerror="alert(1)">',
  } });
  assert.match(modificado, /&lt;img src=x onerror=&quot;alert\(1\)&quot;&gt;/u);
  assert.doesNotMatch(modificado, /<img/u);
});

test("el montaje filtra localmente, conserva foco del campo y desmonta el oyente", () => {
  const listeners = new Map();
  const filas = [{ dataset: { cronosC6Tipo: "horas_medico" }, hidden: false }, { dataset: { cronosC6Tipo: "vacaciones" }, hidden: false }];
  const estado = { textContent: "" };
  const vacio = { hidden: true };
  let retirado = false;
  const contenedor = {
    innerHTML: "",
    addEventListener(tipo, fn) { listeners.set(tipo, fn); },
    removeEventListener(tipo, fn) { if (listeners.get(tipo) === fn) listeners.delete(tipo); },
    querySelectorAll(selector) { return selector === "[data-cronos-c6-tipo]" ? filas : []; },
    querySelector(selector) { return selector === "[data-cronos-c6-resultados]" ? estado : selector === "[data-cronos-c6-vacio]" ? vacio : null; },
    remove() { retirado = true; },
  };
  const raiz = { ownerDocument: { createElement() { return contenedor; } }, append(nodo) { assert.equal(nodo, contenedor); } };
  let registrado;
  const vista = montarCatalogoPermisosCronos({ raiz, registrarDesmontar(fn) { registrado = fn; } });
  const campo = { value: "médico", matches(selector) { return selector === "[data-cronos-c6-buscar]"; } };
  listeners.get("input")({ target: campo });
  assert.deepEqual(filas.map((fila) => fila.hidden), [false, true]);
  assert.equal(estado.textContent, "Tipos mostrados: 1");
  campo.value = "inexistente";
  listeners.get("input")({ target: campo });
  assert.equal(vacio.hidden, false);
  registrado();
  vista.desmontar();
  assert.equal(listeners.size, 0);
  assert.equal(retirado, true);
});
