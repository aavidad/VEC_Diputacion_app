import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteJustificante, MAX_PDF_JUSTIFICANTE } from "./mi-bolsa-justificante-cliente.js";
import { enviarPortalMiBolsa, revisarRespuestaPortalMiBolsa } from "./mi-bolsa-portal.js";

const admitido = { carga_id: `ref:${"a".repeat(64)}`, comprobante_ref: `ref:${"b".repeat(64)}`,
  documento_version: 1, recibo_ref: `recibo:${"c".repeat(64)}`, estado: "admitido", registrado_en: "2026-10-15T08:00:00Z" };
const formulario = new FormData();
formulario.set("documento", new Blob(["%PDF-1.4\n"], { type: "application/pdf" }), "parte.pdf");
const archivo = formulario.get("documento");

test("rechaza un PDF superior a 16 MiB antes de llamar a la red", async () => {
  let llamadas = 0;
  const cliente = crearClienteJustificante({ fetchImpl: async () => { llamadas++; } });
  const datos = new FormData();
  datos.set("documento", new Blob([new Uint8Array(MAX_PDF_JUSTIFICANTE + 1)], { type: "application/pdf" }), "grande.pdf");
  await assert.rejects(cliente.subir("bolsa:01", "clave", datos.get("documento")), { codigo: "archivo_demasiado_grande", estado: 413 });
  assert.equal(llamadas, 0);
});

test("traduce un 413 aunque el servidor no devuelva JSON", async () => {
  const cliente = crearClienteJustificante({ fetchImpl: async () => ({ status: 413, json: async () => { throw new Error(); } }) });
  await assert.rejects(cliente.subir("bolsa:01", "clave", archivo), { codigo: "archivo_demasiado_grande", estado: 413 });
});

test("el cliente permite consultar de forma independiente y subir los mismos bytes", async () => {
  const llamadas = [];
  const cliente = crearClienteJustificante({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return opciones.method === "GET" ? { status: 404 }
      : { status: 201, json: async () => ({ data: admitido }) };
  } });
  assert.equal(await cliente.consultar("bolsa:01", "portal-clave:justificante"), null);
  assert.deepEqual(await cliente.subir("bolsa:01", "portal-clave:justificante", archivo), admitido);
  assert.match(llamadas[0].ruta, /bolsa=bolsa%3A01&clave_idempotencia=portal-clave%3Ajustificante/u);
  assert.equal(llamadas[1].opciones.body.get("bolsa"), "bolsa:01");
  assert.equal(llamadas[1].opciones.body.get("clave_idempotencia"), "portal-clave:justificante");
  assert.deepEqual(await llamadas[1].opciones.body.get("documento").arrayBuffer(), await archivo.arrayBuffer());
  assert.equal(Object.hasOwn(llamadas[1].opciones.headers, "Content-Type"), false);
});

test("una consulta fallida o un recibo incompleto nunca se considera admitido", async () => {
  let llamadas = 0;
  const indisponible = crearClienteJustificante({ fetchImpl: async () => {
    llamadas++; return { status: 503, json: async () => ({ error: { codigo: "servicio_no_disponible" } }) };
  } });
  await assert.rejects(indisponible.consultar("bolsa:01", "clave"));
  assert.equal(llamadas, 1);
  const incompleto = crearClienteJustificante({ fetchImpl: async () => ({ status: 200,
    json: async () => ({ data: { ...admitido, documento_version: undefined } }) }) });
  await assert.rejects(incompleto.consultar("bolsa:01", "clave"));
  const refMalformada = crearClienteJustificante({ fetchImpl: async () => ({ status: 200,
    json: async () => ({ data: { ...admitido, comprobante_ref: "ref:documento_externo" } }) }) });
  await assert.rejects(refMalformada.consultar("bolsa:01", "clave"));
});

test("GET distingue admisión, análisis y rechazo por estado HTTP", async () => {
  for (const [status, data] of [
    [200, admitido],
    [202, { carga_id: admitido.carga_id, estado: "recibido" }],
    [422, { carga_id: admitido.carga_id, estado: "rechazado", codigo_rechazo: "archivo_invalido" }],
  ]) {
    const cliente = crearClienteJustificante({ fetchImpl: async () => ({ status, json: async () => ({ data }) }) });
    assert.deepEqual(await cliente.consultar("bolsa:01", "clave"), data);
  }
  const mal = crearClienteJustificante({ fetchImpl: async () => ({ status: 200,
    json: async () => ({ data: { carga_id: admitido.carga_id, estado: "analizando" } }) }) });
  await assert.rejects(mal.consultar("bolsa:01", "clave"));
});

function formularioRespuesta() {
  const revision = { hidden: true, focus() {} };
  const edicion = { hidden: false };
  const iniciar = { hidden: false };
  const resumen = { innerHTML: "" };
  const resultado = { textContent: "", focus() {} };
  const confirmar = { disabled: false, hidden: false };
  const volver = { disabled: false };
  const reintentar = { disabled: false, hidden: true };
  const nodos = { "[data-portal-revision]": revision, "[data-portal-edicion]": edicion,
    "[data-portal-iniciar-revision]": iniciar, "[data-portal-revision-datos]": resumen,
    "[data-portal-resultado]": resultado, 'button[type="submit"]': confirmar,
    "[data-portal-revision-volver]": volver, "[data-portal-reintentar]": reintentar };
  return { dataset: { portalMiBolsa: "responder", bolsa: "bolsa:01" }, resultado, reintentar,
    querySelector: (selector) => nodos[selector] || null };
}

function datosRespuesta() {
  const datos = new FormData();
  datos.set("respuesta", "renuncia_justificada");
  datos.set("causa", "enfermedad");
  datos.set("justificante", archivo);
  return datos;
}

test("la renuncia se registra solo después de custodiar un PDF admitido", async () => {
  const form = formularioRespuesta();
  const datos = datosRespuesta();
  await revisarRespuestaPortalMiBolsa(form, datos);
  const rutas = [];
  const recibo = { esquema: "vec.bolsa.mi-bolsa.respuesta.v1", referencia: `respuesta-portal:${"d".repeat(64)}`,
    recibo: `recibo:respuesta-portal:${"e".repeat(64)}`, registrada_en: "2026-10-15T08:00:00Z",
    estado: "firme", repetida: false };
  const fetchImpl = async (ruta, opciones) => {
    rutas.push([ruta, opciones]);
    if (ruta.endsWith("/justificantes")) return { status: 201, json: async () => ({ data: admitido }) };
    return { status: 201, json: async () => ({ data: recibo }) };
  };
  assert.equal(await enviarPortalMiBolsa(form, { datos, fetchImpl }), true);
  assert.deepEqual(rutas.map(([ruta, opciones]) => [ruta, opciones.method]), [
    ["/api/vec/bolsa/mi-bolsa/justificantes", "POST"],
    ["/api/vec/bolsa/mi-bolsa/respuestas", "POST"],
  ]);
  const cuerpo = JSON.parse(rutas[1][1].body);
  assert.equal(cuerpo.justificante_ref, admitido.comprobante_ref);
  assert.equal(cuerpo.justificante_sha256, "e5c62df5dab5c87b6a015ef3d43597074d1eec433b15f51aec63b8582d0e4ab4");
});

test("POST documental 503 o 202 impide la renuncia y reintenta el mismo PDF y clave", async () => {
  for (const status of [503, 202]) {
    const form = formularioRespuesta();
    const datos = datosRespuesta();
    await revisarRespuestaPortalMiBolsa(form, datos);
    const rutas = [];
    const fetchImpl = async (ruta, opciones) => {
      rutas.push({ ruta, opciones });
      return status === 503 ? { status, json: async () => ({ error: { codigo: "servicio_no_disponible" } }) }
        : { status: 202, json: async () => ({ data: { carga_id: admitido.carga_id, estado: "analizando" } }) };
    };
    assert.equal(await enviarPortalMiBolsa(form, { datos, fetchImpl }), false);
    assert.equal(rutas.length, 1);
    assert.equal(rutas[0].ruta, "/api/vec/bolsa/mi-bolsa/justificantes");
    assert.equal(rutas[0].opciones.method, "POST");
    assert.equal(rutas[0].opciones.body.get("documento").name, archivo.name);
    assert.deepEqual(await rutas[0].opciones.body.get("documento").arrayBuffer(), await archivo.arrayBuffer());
    assert.equal(rutas[0].opciones.body.get("clave_idempotencia"), form.dataset.clave + ":justificante");
    assert.equal(form.reintentar.hidden, false);
    assert.equal(await enviarPortalMiBolsa(form, { datos, fetchImpl }), false);
    assert.equal(rutas.length, 1, "sin reintento explícito no duplica el POST");
    assert.equal(await enviarPortalMiBolsa(form, { datos, fetchImpl, reintentoExplicito: true }), false);
    assert.equal(rutas.length, 2);
    assert.equal(rutas[1].opciones.method, "POST");
    assert.equal(rutas[1].opciones.body.get("clave_idempotencia"), rutas[0].opciones.body.get("clave_idempotencia"));
    assert.deepEqual(await rutas[1].opciones.body.get("documento").arrayBuffer(),
      await rutas[0].opciones.body.get("documento").arrayBuffer());
  }
});

test("una respuesta de renuncia incierta conserva la referencia admitida sin resubir PDF", async () => {
  const form = formularioRespuesta();
  const datos = datosRespuesta();
  await revisarRespuestaPortalMiBolsa(form, datos);
  const llamadas = [];
  const recibo = { esquema: "vec.bolsa.mi-bolsa.respuesta.v1", referencia: `respuesta-portal:${"d".repeat(64)}`,
    recibo: `recibo:respuesta-portal:${"e".repeat(64)}`, registrada_en: "2026-10-15T08:00:00Z",
    estado: "firme", repetida: true };
  const fetchImpl = async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    if (ruta.endsWith("/justificantes")) return { status: 201, json: async () => ({ data: admitido }) };
    return llamadas.length === 2
      ? { status: 503, json: async () => ({ error: { codigo: "servicio_no_disponible" } }) }
      : { status: 200, json: async () => ({ data: recibo }) };
  };
  assert.equal(await enviarPortalMiBolsa(form, { datos, fetchImpl }), false);
  assert.deepEqual(llamadas.map(({ ruta }) => ruta), [
    "/api/vec/bolsa/mi-bolsa/justificantes", "/api/vec/bolsa/mi-bolsa/respuestas",
  ]);
  assert.equal(await enviarPortalMiBolsa(form, { datos, fetchImpl, reintentoExplicito: true }), true);
  assert.deepEqual(llamadas.map(({ ruta }) => ruta), [
    "/api/vec/bolsa/mi-bolsa/justificantes", "/api/vec/bolsa/mi-bolsa/respuestas",
    "/api/vec/bolsa/mi-bolsa/respuestas",
  ]);
  assert.equal(llamadas[2].opciones.body, llamadas[1].opciones.body);
  assert.equal(JSON.parse(llamadas[2].opciones.body).justificante_ref, admitido.comprobante_ref);
});

test("un justificante rechazado permite corregir el archivo sin enviar la respuesta", async () => {
  const form = formularioRespuesta();
  const datos = datosRespuesta();
  await revisarRespuestaPortalMiBolsa(form, datos);
  let llamadas = 0;
  const fetchImpl = async () => {
    llamadas++;
    return { status: 422, json: async () => ({ data: {
      carga_id: admitido.carga_id, estado: "rechazado", codigo_rechazo: "archivo_invalido",
    } }) };
  };
  assert.equal(await enviarPortalMiBolsa(form, { datos, fetchImpl }), false);
  assert.equal(llamadas, 1);
  assert.equal(form.reintentar.hidden, true);
  assert.match(form.resultado.textContent, /otro PDF/u);
});

test("POST documental 422 tampoco registra la renuncia", async () => {
  const form = formularioRespuesta();
  const datos = datosRespuesta();
  await revisarRespuestaPortalMiBolsa(form, datos);
  const metodos = [];
  const fetchImpl = async (_ruta, opciones) => {
    metodos.push(opciones.method);
    return { status: 422, json: async () => ({ data: {
        carga_id: admitido.carga_id, estado: "rechazado", codigo_rechazo: "archivo_invalido",
      } }) };
  };
  assert.equal(await enviarPortalMiBolsa(form, { datos, fetchImpl }), false);
  assert.deepEqual(metodos, ["POST"]);
  assert.match(form.resultado.textContent, /otro PDF/u);
});
