import assert from "node:assert/strict";
import test from "node:test";

import { crearBorradorAlta } from "../modulos/contratacion-temporal/contrato.js";
import {
  pedir,
  renderizarPeticionCentro,
  validarReciboPeticionCentro,
  registrarOperacionPeticionCentro,
} from "./peticiones-centro.js";

const catalogos = {
  esquema: "vec.contratacion_temporal.catalogos_alta.v1",
  centros: [{ referencia: "cen_sintetico_001", etiqueta: "Centro sintético", contactos: [{ referencia: "con_sintetico_001", etiqueta: "Contacto" }] }],
  categorias: [{ referencia: "cat_sintetica_001", etiqueta: "C2 sintética", grupos_subgrupos: [{ clave: "C2", etiqueta: "C2" }] }],
  motivos: [{ clave: "sustitucion", etiqueta: "Sustitución" }],
  documentos: [],
};

const contexto = { actor: { referencia: "actor:sintetico:001", nombre: "Persona de prueba", cargo: "Cargo sintético", centro: "Centro sintético", puede_presentar: true, puede_ratificar: false }, catalogos };
const peticion = { referencia: "peticion:centro:001", version: 1, estado: "pendiente_ratificacion", configuracion: { solicitante: { actor_ref: "actor:sintetico:001", puesto_ref: "puesto:sintetico:001" } }, solicitud: { centro_ref: "cen_sintetico_001", categoria_ref: "cat_sintetica_001", grupo_subgrupo: "C2", motivo_clave: "sustitucion", detalle: "Necesidad sintética", periodo: { inicio: "2026-09-01T00:00:00Z", fin: "2026-09-30T00:00:00Z" } }, creada_en: "2026-09-06T08:00:00Z" };

test("renderer comparte el formulario de alta y deja claro el circuito previo", () => {
  const html = renderizarPeticionCentro({ contexto, modo: "formulario", estado: { fase: "edicion", disponible: true, ocupado: false, borrador: crearBorradorAlta(), catalogos, errores: {}, mensaje_clave: "estado_disponible", tipo_mensaje: "informacion" } });
  assert.match(html, /data-ct-form/);
  assert.match(html, /Identidades de prueba/);
  assert.match(html, /pendiente de entrada/);
  assert.match(html, /C2/);
});

test("renderer de ratificación muestra todos los datos revisables", () => {
  const html = renderizarPeticionCentro({ contexto: { ...contexto, actor: { ...contexto.actor, puede_presentar: false, puede_ratificar: true } }, modo: "ratificacion", peticion });
  assert.match(html, /Necesidad sintética/);
  assert.match(html, /motivo_ratificacion/);
  assert.match(html, /confirmacion_ratificacion/);
  assert.match(html, /no se firma electrónicamente/);
});

test("pedir conserva cuerpo y certificado exclusivamente en el mismo origen", async () => {
  const anterior = globalThis.fetch;
  const llamadas = [];
  globalThis.fetch = async (ruta, opciones) => { llamadas.push({ ruta, opciones }); return new Response(JSON.stringify({ data: { ok: true } }), { status: 200 }); };
  try { assert.deepEqual(await pedir("/api/prueba", { method: "POST", cuerpo: { clave: "misma" } }), { ok: true }); } finally { globalThis.fetch = anterior; }
  assert.equal(llamadas[0].opciones.credentials, "same-origin");
  assert.equal(llamadas[0].opciones.mode, "same-origin");
  assert.equal(llamadas[0].opciones.redirect, "error");
  assert.deepEqual(JSON.parse(llamadas[0].opciones.body), { clave: "misma" });
});

test("resultado incierto conserva señal para reintento y el recibo se liga al objetivo", async () => {
  const anterior = globalThis.fetch;
  let intentos = 0;
  globalThis.fetch = async () => { intentos += 1; throw new DOMException("timeout", "AbortError"); };
  try { await assert.rejects(pedir("/api/prueba"), (error) => error.indeterminado === true); } finally { globalThis.fetch = anterior; }
  assert.equal(intentos, 1);
  assert.doesNotThrow(() => validarReciboPeticionCentro({ recibo_ref: "recibo:sintetico:001", peticion_ref: "peticion:centro:001", version: 1, estado: "pendiente_ratificacion", actor_ref: "actor:sintetico:001", registrado_en: "2026-09-06T08:00:01Z", estado_local: "registrado" }, { peticionRef: "peticion:centro:001", version: 1, estado: "pendiente_ratificacion", actorRef: "actor:sintetico:001" }));
});

test("un fallo de transporte o un recibo inválido no permiten crear otra operación", async () => {
  const comando = { operacion: "presentar", clave_idempotencia: "f3134ee2-61af-467d-aa58-dc71f07553b6", solicitud: peticion.solicitud };
  for (const fallo of [new TypeError("conexión perdida"), { status: 503 }, null]) {
    const cliente = async () => { if (fallo) throw fallo; return { recibo_ref: "incorrecto" }; };
    await assert.rejects(registrarOperacionPeticionCentro(cliente, comando, contexto.actor.referencia), (e) => e.indeterminado === true);
  }
  for (const status of [400, 403, 409]) {
    await assert.rejects(registrarOperacionPeticionCentro(async () => { throw { status }; }, comando, contexto.actor.referencia), (e) => e.status === status && !e.indeterminado);
  }
  const enviados = [];
  const cliente = async (_ruta, opciones) => {
    enviados.push(opciones.cuerpo);
    return { recibo_ref: "recibo:peticion-centro:001", peticion_ref: `peticion:centro:${comando.clave_idempotencia}`, version: 1,
      estado: "pendiente_ratificacion", actor_ref: contexto.actor.referencia, registrado_en: "2026-09-06T08:00:01Z", estado_local: "replay_confirmado" };
  };
  await registrarOperacionPeticionCentro(cliente, comando, contexto.actor.referencia);
  await registrarOperacionPeticionCentro(cliente, comando, contexto.actor.referencia);
  assert.deepEqual(enviados, [comando, comando]);
});

test("JSON ilegible con HTTP 200 es resultado incierto, no rechazo confirmado", async () => {
  const anterior = globalThis.fetch;
  globalThis.fetch = async () => new Response("{incompleto", { status: 200 });
  try { await assert.rejects(pedir("/api/prueba", { method: "POST", cuerpo: {} }), (e) => e.indeterminado); }
  finally { globalThis.fetch = anterior; }
});

test("detalle incluye crédito y documentos; ratificada no ofrece otra ratificación", () => {
  const completa = structuredClone(peticion);
  completa.solicitud.rc = { existe: true, numero: "RC-SINTETICA", fecha: "2026-09-06T00:00:00Z", importe: { centimos: 123456, moneda: "EUR" }, documento_ref: "doc:rc:001" };
  completa.solicitud.documentos_adjuntos = ["doc:adjunto:001"];
  const ctx = { ...contexto, actor: { ...contexto.actor, puede_presentar: false, puede_ratificar: true } };
  const html = renderizarPeticionCentro({ contexto: ctx, peticion: completa });
  assert.match(html, /RC-SINTETICA/);
  assert.match(html, /doc:rc:001/);
  assert.match(html, /doc:adjunto:001/);
  assert.match(html, /data-accion="abrir-ratificacion"/);
  completa.version = 2; completa.estado = "ratificada";
  assert.doesNotMatch(renderizarPeticionCentro({ contexto: ctx, peticion: completa }), /data-accion="abrir-ratificacion"/);
});

test("un recibo confirmado permite volver a la bandeja sin repetir la presentación", () => {
  const html = renderizarPeticionCentro({ contexto, recibo: { registrado_en: "2026-09-06T08:00:01Z" } });
  assert.match(html, /Operación registrada/);
  assert.match(html, /data-accion="recargar"/);
  assert.match(html, /data-accion="nueva"/);
});
