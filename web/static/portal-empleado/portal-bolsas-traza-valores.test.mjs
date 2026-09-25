import test from "node:test";
import assert from "node:assert/strict";
import { renderizarTrazaValores, validarCambiosTraza } from "./portal-bolsas-traza-valores.js";
import { consultarOperacionesSituacion, renderizarOperacionesSituacion } from "./portal-bolsas-operaciones.js";

const escapar = (v) => String(v ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");
const cambio = (campo, valor_anterior, valor_nuevo) => ({ instante: "2026-09-25T08:00:00.123456Z", recibo_ref: "recibo:1", campo, valor_anterior, valor_nuevo, actor: "per_rrhh" });

test("petición RRHH p.4: la traza admite solo valores minimizados", () => {
  assert.deepEqual(validarCambiosTraza(undefined), []);
  assert.ok(validarCambiosTraza([cambio("situacion", "disponible", "no_disponible"), cambio("telefono_1", null, "version:1")]));
  assert.equal(validarCambiosTraza([cambio("correo", "a@ejemplo.es", "b@ejemplo.es")]), null, "un correo en claro no es válido");
  assert.equal(validarCambiosTraza([cambio("telefono_2", "version:1", "600000000")]), null);
  assert.equal(validarCambiosTraza([cambio("situacion", "disponible", "inventada")]), null);
  assert.equal(validarCambiosTraza([cambio("otro", null, "x")]), null);
  assert.equal(validarCambiosTraza([cambio("situacion", null, null)]), null);
  assert.equal(validarCambiosTraza("x"), null);
});

test("petición RRHH p.4: la ficha pinta valor anterior y nuevo en una tabla accesible", () => {
  const html = renderizarTrazaValores({ escaparHTML: escapar, cambios: [
    cambio("situacion", "disponible", "no_disponible"),
    cambio("fecha_disponible", null, "2026-10-01T00:00:00.000000Z"),
    cambio("correo", "version:1", "version:2"),
  ] });
  assert.match(html, /<caption>Cambios registrados con valor anterior y nuevo<\/caption>/);
  assert.match(html, /<th scope="col">Valor anterior<\/th><th scope="col">Valor nuevo<\/th>/);
  assert.match(html, /<th scope="row">Situación<\/th><td>Disponible<\/td><td>No disponible<\/td>/);
  assert.match(html, /<th scope="row">Correo<\/th><td>Versión 1<\/td><td>Versión 2<\/td>/);
  assert.match(html, /<th scope="row">Disponible desde<\/th><td>—<\/td><td>1\/10\/26/);
  assert.match(html, /<time datetime="2026-09-25T08:00:00.123456Z">/);
  assert.match(renderizarTrazaValores({ escaparHTML: escapar, cambios: [] }), /No hay cambios registrados/);
  const muchos = Array.from({ length: 8 }, () => cambio("situacion", "disponible", "no_disponible"));
  const segunda = renderizarTrazaValores({ escaparHTML: escapar, cambios: muchos, pagina: 1 });
  assert.match(segunda, /Mostrando 7 a 8 de 8/);
  assert.match(segunda, /data-b8-accion="pagina-traza" data-pagina="0"/);
});

test("petición RRHH p.4: el historial de la ficha incluye los cambios y rechaza un claro", async () => {
  const respuesta = (cambios) => async () => ({ ok: true, status: 200, json: async () => ({ data: { esquema: "vec.bolsa.rrhh.operaciones_situacion.v1", items: [], cambios } }) });
  const bien = await consultarOperacionesSituacion("bolsa:1", "participacion:1", { fetchImpl: respuesta([cambio("situacion", "disponible", "trabajando")]) });
  assert.equal(bien.ok, true);
  assert.equal(bien.cambios.length, 1);
  const mal = await consultarOperacionesSituacion("bolsa:1", "participacion:1", { fetchImpl: respuesta([cambio("correo", null, "persona@ejemplo.es")]) });
  assert.equal(mal.ok, false);
  const html = renderizarOperacionesSituacion({ candidato: { estado_clave: "disponible" }, estado: { carga: "listo", items: [], cambios: bien.cambios } });
  assert.match(html, /<h4>Cambios registrados<\/h4>/);
  assert.match(html, /<td>Disponible<\/td><td>Trabajando<\/td>/);
});
