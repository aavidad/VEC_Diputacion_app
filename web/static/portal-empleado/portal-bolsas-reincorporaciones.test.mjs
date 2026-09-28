import test from "node:test";
import assert from "node:assert/strict";
import { MENSAJES_PORTAL_ES, traducirPortal } from "./portal-i18n.js";
import {
  ESQUEMA_REINCORPORACIONES_TITULAR,
  cargarReincorporacionesTitularFicha,
  consultarReincorporacionesTitular,
  manejarClickReincorporacionesTitular,
  renderizarReincorporacionesTitular,
  rutaReincorporacionesTitular,
} from "./portal-bolsas-reincorporaciones.js";

const escaparHTML = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");
const item = Object.freeze({
  evento_ref: "evento:ct:1", expediente_ref: "expediente:ct:1", relacion_ref: "relacion:1",
  fecha_efectiva: "2026-09-28", recibo_ct_ref: "recibo:ct:1", cese_evento_ref: "cese:1",
  estado: "cese_aplicado", disponible_desde: "2026-10-01", regla_version: 1,
  regla_huella_sha256: "a".repeat(64),
});
const respuesta = (status, body) => ({ ok: status >= 200 && status < 300, status, json: async () => body });
const cuerpo = (items) => ({ data: { esquema: ESQUEMA_REINCORPORACIONES_TITULAR, items } });

test("la consulta usa la ruta canónica y GET sin credenciales persistidas ni caché", async () => {
  assert.equal(rutaReincorporacionesTitular("bolsa:1", "participacion:1"), "/api/vec/bolsa/bolsas/bolsa:1/candidatos/participacion:1/reincorporaciones-titular");
  assert.equal(rutaReincorporacionesTitular("bolsa/ajena", "participacion:1"), null);
  let llamada;
  const resultado = await consultarReincorporacionesTitular("bolsa:1", "participacion:1", {
    fetchImpl: async (ruta, opciones) => { llamada = { ruta, opciones }; return respuesta(200, cuerpo([item])); },
  });
  assert.equal(resultado.ok, true);
  assert.deepEqual(resultado.datos, [item]);
  assert.equal(llamada.opciones.method, "GET");
  assert.equal(llamada.opciones.headers.Accept, "application/json");
  assert.equal(llamada.opciones.credentials, "same-origin");
  assert.equal(llamada.opciones.cache, "no-store");
  assert.equal(llamada.opciones.redirect, "error");
  assert.equal(llamada.opciones.body, undefined);
});

test("falla cerrado ante esquema, fechas, estado o datos inesperados", async () => {
  const invalidos = [
    { data: { esquema: "otro", items: [] } }, cuerpo({}), cuerpo([{ ...item, fecha_efectiva: "2026-02-31" }]),
    cuerpo([{ ...item, estado: "pendiente_cese" }]), cuerpo([{ ...item, recibo_ct_ref: "" }]),
    cuerpo([{ ...item, regla_huella_sha256: "no-es-huella" }]),
  ];
  for (const body of invalidos) {
    const resultado = await consultarReincorporacionesTitular("b", "p", { fetchImpl: async () => respuesta(200, body) });
    assert.equal(resultado.ok, false);
    assert.equal(resultado.mensaje, traducirPortal("reincorporacion_error_contrato"));
  }
});

test("denegación y servicio pendiente se comunican sin reutilizar datos ni revelar errores internos", async () => {
  for (const [status, clave] of [[401, "reincorporacion_error_401"], [403, "reincorporacion_error_403"], [404, "reincorporacion_error_404"], [503, "reincorporacion_error_503"]]) {
    const resultado = await consultarReincorporacionesTitular("b", "p", {
      fetchImpl: async () => respuesta(status, { error: { detalle: "secreto interno" } }),
    });
    assert.equal(resultado.ok, false);
    assert.equal(resultado.mensaje, traducirPortal(clave));
    assert.equal(JSON.stringify(resultado).includes("secreto"), false);
  }
});

test("cancela el resultado antiguo al cambiar de ficha y deja la situación sin tocar", async () => {
  let terminar;
  const actual = { candidato: { participacion_ref: "part:1", estado_clave: "trabajando" } };
  const estado = { bolsaSeleccionada: "bolsa:1", modalFicha: actual };
  let renders = 0;
  const tarea = cargarReincorporacionesTitularFicha(actual, {
    estado, renderizar: () => { renders++; },
    consultar: () => new Promise((resolve) => { terminar = resolve; }),
  });
  assert.equal(actual.reincorporacionesTitular.carga, "cargando");
  actual.controladorReincorporaciones.abort();
  estado.modalFicha = { candidato: { participacion_ref: "part:2" } };
  terminar({ ok: true, datos: [item] });
  await tarea;
  assert.equal(actual.reincorporacionesTitular.carga, "cargando");
  assert.equal(actual.candidato.estado_clave, "trabajando");
  assert.equal(renders, 1);
});

test("la ficha muestra recibo y fecha sin inventar actor ni mutar disponibilidad", () => {
  const html = renderizarReincorporacionesTitular({ estado: { carga: "listo", items: [{ ...item, recibo_ct_ref: "recibo:<script>" }] }, escaparHTML });
  assert.match(html, /recibo:&lt;script&gt;/);
  assert.match(html, /<time datetime="2026-10-01">1\/10\/26<\/time>/);
  assert.match(html, /role="region"/);
  assert.doesNotMatch(html, /actor|data-bolsa-accion="cambiar|<script>/i);
  assert.match(renderizarReincorporacionesTitular({ estado: { carga: "pendiente", error: traducirPortal("reincorporacion_error_404") }, escaparHTML }), /Reintentar consulta/);
  assert.doesNotMatch(renderizarReincorporacionesTitular({ estado: { carga: "denegado" }, escaparHTML }), /data-reincorporacion-accion="reintentar"/);
});

test("catálogo común cubre todos los textos y el control pagina sin llamada de red", () => {
  assert.equal(typeof MENSAJES_PORTAL_ES.reincorporacion_titulo, "string");
  assert.equal(traducirPortal("reincorporacion_mostrando", { desde: 1, hasta: 2, total: 3 }), "Mostrando 1 a 2 de 3");
  const modalFicha = { reincorporacionesTitular: { carga: "listo", items: Array(8).fill(item), pagina: 0 } };
  const estado = { modalFicha };
  let renders = 0;
  const control = { dataset: { reincorporacionAccion: "pagina", pagina: "1" } };
  const evento = { target: { closest: () => control }, preventDefault() {} };
  assert.equal(manejarClickReincorporacionesTitular(evento, { estado, renderizar: () => { renders++; } }), true);
  assert.equal(modalFicha.reincorporacionesTitular.pagina, 1);
  assert.equal(renders, 1);
});
