import test from "node:test";
import assert from "node:assert/strict";
import { crearClienteOfertas, crearSuperficieOfertasBolsa, crearTraductorOfertas, mensajeError, validarOfertasBolsa, ESQUEMA_OFERTAS_BOLSA, RUTA_OFERTAS_BOLSA, RUTA_RESOLUCIONES_OFERTA } from "./portal-bolsas-ofertas.js";

function oferta(extra = {}) {
  return {
    oferta_ref: "oferta:1", recibo_ref: "recibo:oferta:1", bolsa_ref: "bolsa:1",
    datos: { categoria: "Auxiliar <b>", centro: "Residencia", fecha_inicio: "2026-10-01", descripcion: "Sustitución" },
    plazo: { regla_ref: "vec.bolsa.reglas:1:b10.plazo_publicacion", ejemplo: false, cantidad: 2 },
    publicada_en: "2026-09-25T10:00:00Z", vence_antes_de: "2026-09-29T22:00:00Z", estado: "abierta",
    disposiciones: [], disposiciones_total: 0, propuesta: null, resolucion: null, ...extra,
  };
}

function respuesta(status, cuerpo) {
  return { ok: status >= 200 && status < 300, status, json: async () => cuerpo };
}

test("el contrato rechaza estados y propuestas ajenos", () => {
  assert.equal(validarOfertasBolsa({ data: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [oferta()] } }).ofertas.length, 1);
  assert.throws(() => validarOfertasBolsa({ data: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [oferta({ estado: "inventado" })] } }));
  assert.throws(() => validarOfertasBolsa({ data: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [oferta({ propuesta: { tipo: "adjudicar" } })] } }));
});

test("el cliente envía solo bolsa, datos y clave, sin identidad", async () => {
  const llamadas = [];
  const cliente = crearClienteOfertas({ fetchImpl: async (ruta, opciones) => { llamadas.push({ ruta, opciones }); return respuesta(201, { data: oferta() }); } });
  const r = await cliente.publicar("bolsa:1", { categoria: "Aux", centro: "Res", fecha_inicio: "2026-10-01", descripcion: "Des" }, "oferta-clave-1");
  assert.equal(r.ok, true);
  assert.equal(llamadas[0].ruta, RUTA_OFERTAS_BOLSA);
  assert.equal(llamadas[0].opciones.headers["Idempotency-Key"], "oferta-clave-1");
  assert.deepEqual(Object.keys(JSON.parse(llamadas[0].opciones.body)), ["bolsa_ref", "datos"]);
  await cliente.resolver("bolsa:1", "oferta:1", null, "clave-resol-1");
  assert.equal(llamadas[1].ruta, RUTA_RESOLUCIONES_OFERTA);
  assert.equal(JSON.parse(llamadas[1].opciones.body).participacion_ref, null);
});

test("los errores del servidor se traducen sin mostrar códigos", () => {
  const t = crearTraductorOfertas();
  assert.match(mensajeError(t, { status: 409, codigo: "propuesta_cambiada" }), /orden ha cambiado/);
  assert.match(mensajeError(t, { status: 503, codigo: "plazo_no_configurado" }), /regla de plazo/);
  assert.match(mensajeError(t, { status: 500, codigo: "x" }), /No se pudo completar/);
});

test("la superficie publica con clave estable al reintentar y escapa los datos", async () => {
  let claves = 0; const enviadas = [];
  let fallar = true;
  const cliente = {
    consultar: async () => ({ ok: true, datos: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [oferta()] } }),
    publicar: async (_b, _d, clave) => { enviadas.push(clave); if (fallar) { fallar = false; return { ok: false, status: 503, codigo: "servicio_no_disponible" }; } return { ok: true, status: 201, oferta: oferta() }; },
    resolver: async () => ({ ok: true, status: 201, oferta: oferta() }),
  };
  const s = crearSuperficieOfertasBolsa({ cliente, generarClave: () => `clave-${++claves}` });
  s.activar("bolsa:1");
  await new Promise((r) => setTimeout(r, 0));
  const html = s.renderizar();
  assert.match(html, /Auxiliar &lt;b&gt;/);
  assert.doesNotMatch(html, /<b>/);
  const datos = { categoria: "Aux", centro: "Res", fecha_inicio: "2026-10-01", descripcion: "Des" };
  const formulario = { closest: () => formulario, reportValidity: () => true };
  globalThis.FormData = class { get(k) { return datos[k] ?? ""; } };
  s.manejarSubmit({ target: formulario, preventDefault() {} });
  await new Promise((r) => setTimeout(r, 0));
  assert.match(s.renderizar(), /Reintentar la misma publicación/);
  s.manejarSubmit({ target: formulario, preventDefault() {} });
  await new Promise((r) => setTimeout(r, 0));
  assert.deepEqual(enviadas, ["clave-1", "clave-1"]);
});

test("la propuesta vencida ofrece confirmar adjudicación o llamamiento directo", async () => {
  const adjudicar = oferta({ oferta_ref: "oferta:2", estado: "pendiente_resolucion", disposiciones_total: 1,
    disposiciones: [{ participacion_ref: "p:3", manifestada_en: "2026-09-26T08:00:00Z", orden_vigente: 2, situacion: "disponible" }],
    propuesta: { tipo: "adjudicar", participacion_ref: "p:3", orden_vigente: 2 } });
  const directo = oferta({ oferta_ref: "oferta:3", estado: "pendiente_resolucion", propuesta: { tipo: "llamamiento_directo" } });
  const resueltas = [];
  const cliente = {
    consultar: async () => ({ ok: true, datos: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [adjudicar, directo] } }),
    publicar: async () => ({ ok: false, status: 500 }),
    resolver: async (_b, o, p) => { resueltas.push([o, p]); return { ok: true, status: 201, oferta: adjudicar }; },
  };
  const s = crearSuperficieOfertasBolsa({ cliente, generarClave: () => "clave-resol-1" });
  s.activar("bolsa:1");
  await new Promise((r) => setTimeout(r, 0));
  const html = s.renderizar();
  assert.match(html, /Adjudicar al orden 2/);
  assert.match(html, /Confirmar adjudicación/);
  assert.match(html, /Pasar a llamamiento directo/);
  const boton = (ofertaRef, participacionRef) => { const b = { disabled: false, dataset: { ofertasAccion: "resolver", ofertaRef, participacionRef } }; b.closest = () => b; return b; };
  s.manejarClick({ target: boton("oferta:2", "p:3") });
  await new Promise((r) => setTimeout(r, 0));
  s.manejarClick({ target: boton("oferta:3", undefined) });
  await new Promise((r) => setTimeout(r, 0));
  assert.deepEqual(resueltas, [["oferta:2", "p:3"], ["oferta:3", null]]);
});
