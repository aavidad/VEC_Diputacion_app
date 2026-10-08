import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteInscripcionBolsa } from "./inscripcion-bolsa-api.js";

const instante = "2026-10-09T09:00:00Z";
const bolsa = { bolsa_ref: "bolsa:plaza-42", categoria: "Auxiliar administrativo",
  plazo_inicio: instante, plazo_fin: "2026-10-22T23:59:59Z", requisitos_resumen: "Titulación requerida",
  catalogo_version: 2 };

test("la persona consulta páginas y presenta sin datos de identidad del navegador", async () => {
  const llamadas = [];
  const respuestas = [
    { data: { esquema: "vec.bolsa.inscripciones.abiertas.v1", bolsas: [bolsa], total: 1, cursor_siguiente: null } },
    { data: { esquema: "vec.bolsa.inscripcion.bolsa.v1", bolsa: { ...bolsa,
      requisitos: [{ codigo: "titulo", descripcion: "Título exigido", obligatorio: true }] } } },
    { data: { esquema: "vec.bolsa.inscripcion.recibo.v1", solicitud_ref: "solicitud:42", recibo_ref: "recibo:42",
      bolsa_ref: bolsa.bolsa_ref, estado: "pendiente", version: 1, registrada_en: instante, repetida: false } },
  ];
  const cliente = crearClienteInscripcionBolsa({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return { ok: true, status: 200, json: async () => respuestas.shift() };
  } });
  assert.equal((await cliente.abiertas()).total, 1);
  assert.equal((await cliente.bolsa(bolsa.bolsa_ref)).bolsa.catalogo_version, 2);
  const recibo = await cliente.inscribir({ bolsaRef: bolsa.bolsa_ref, catalogoVersion: 2,
    claveIdempotencia: "77777777-7777-4777-8777-777777777777" });
  assert.equal(recibo.estado, "pendiente");
  assert.equal(llamadas[0].ruta, "/api/vec/bolsa/inscripciones/bolsas-abiertas?limite=20");
  assert.equal(llamadas[1].ruta, "/api/vec/bolsa/inscripciones/bolsas-abiertas/bolsa:plaza-42");
  assert.equal(llamadas[2].ruta, "/api/vec/bolsa/mi-bolsa/inscripciones");
  assert.deepEqual(JSON.parse(llamadas[2].opciones.body), { bolsa_ref: bolsa.bolsa_ref,
    catalogo_version: 2, clave_idempotencia: "77777777-7777-4777-8777-777777777777" });
  assert.ok(llamadas.every(({ opciones }) => opciones.credentials === "same-origin"
    && opciones.mode === "same-origin" && opciones.cache === "no-store"
    && opciones.redirect === "error" && opciones.referrerPolicy === "no-referrer"));
});

test("rechaza respuesta de otra persona o bolsa y conserva el código de error", async () => {
  const solicitud = { solicitud_ref: "solicitud:ajena", recibo_ref: "recibo:1", bolsa_ref: bolsa.bolsa_ref,
    estado: "pendiente", version: 1, registrada_en: instante };
  const cliente = crearClienteInscripcionBolsa({ fetchImpl: async () => ({ ok: true,
    json: async () => ({ data: { esquema: "vec.bolsa.inscripcion.propias.detalle.v1", solicitud } }) }) });
  await assert.rejects(cliente.detallePropio("solicitud:propia"), /distinta/u);
  const denegado = crearClienteInscripcionBolsa({ fetchImpl: async () => ({ ok: false, status: 403 }) });
  await assert.rejects(denegado.propias(), (error) => error.status === 403);
});
