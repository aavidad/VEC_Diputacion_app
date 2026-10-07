import assert from "node:assert/strict";
import test from "node:test";
import { createHash } from "node:crypto";

import { crearClienteFirmaExterna, ErrorFirmaExterna, RUTA_REGISTRO_FIRMA_EXTERNA } from "./firma-externa-cliente.js";
import { crearAccionesFirma, renderizarAccionesPaso } from "./circuito-firma-acciones.js";
import { crearTraductorCircuitoFirma } from "./i18n-circuito-firma.js";

const pdf = new TextEncoder().encode("%PDF-1.7\nobj\n%%EOF");
const solicitud = Object.freeze({
  expedienteRef: "expediente:ct:001", version: 7, documento: "resolucion", pasoOrden: 1,
  originalRef: "ref:original-autorizado", originalVersion: 3, firmado: pdf,
  referenciaPortafirmas: "PF-2026-001", fechaPortafirmas: "2026-10-02T08:15:00Z",
  clave: "firma-0123456789abcdef",
});

function recibo(yaRegistrada = false) {
  return {
    esquema: "vec.contratacion-temporal.registro-firma-externa.v2", recibo_ref: "recibo:firma:001",
    firma_ref: "firma:ct:001", ya_registrada: yaRegistrada, expediente_ref: solicitud.expedienteRef,
    version_expediente: solicitud.version, documento: solicitud.documento, paso_orden: solicitud.pasoOrden,
    paso_ref: "paso:resolucion:1", secuencia: 1, registrada_en: "2026-10-02T08:16:00Z",
    documento_custodiado: { expediente_ref: `ref:${"a".repeat(64)}`, documento_ref: `ref:${"b".repeat(64)}`,
      version: 1, huella_sha256: createHash("sha256").update(pdf).digest("hex") },
    verificacion_tecnica: { estado: "valida", motivo: "verificada", politica: "politica:vec:firma:verificacion-autonoma:v1",
      revocacion: "vigente", sello_tiempo: "no_presente", original_sha256: "d".repeat(64), firmado_sha256: createHash("sha256").update(pdf).digest("hex") },
    procedencia_portafirmas: { estado: "declarada_por_rrhh", referencia_declarada: solicitud.referenciaPortafirmas,
      fecha_declarada: solicitud.fechaPortafirmas }, firma_eficaz: false, material_root_sha256: "e".repeat(64),
    revision_pdf: { orden_firma: 1, entrada_sha256: "d".repeat(64), revision_sha256: createHash("sha256").update(pdf).digest("hex"), evidencia_sha256: "f".repeat(64) },
  };
}

function respuesta(data, status = 201) {
  return new Response(JSON.stringify(data), { status, headers: { "Content-Type": "application/json; charset=utf-8" } });
}

test("el cliente envía solo el PDF firmado y las referencias declaradas al POST interno", async () => {
  let peticion;
  const cliente = crearClienteFirmaExterna({ fetchImpl: async (ruta, opciones) => {
    peticion = { ruta, opciones }; return respuesta({ data: recibo() });
  } });
  const resultado = await cliente.registrar(solicitud);
  assert.equal(resultado.firma_eficaz, false);
  assert.equal(resultado.procedencia_portafirmas.estado, "declarada_por_rrhh");
  assert.equal(peticion.ruta, RUTA_REGISTRO_FIRMA_EXTERNA);
  assert.equal(peticion.opciones.method, "POST");
  assert.equal(peticion.opciones.credentials, "same-origin");
  assert.equal(peticion.opciones.cache, "no-store");
  assert.equal(peticion.opciones.redirect, "error");
  const cuerpo = JSON.parse(peticion.opciones.body);
  assert.deepEqual(Object.keys(cuerpo).sort(), ["expediente_ref", "version_expediente", "documento", "paso_orden",
    "original_ref", "original_version", "firmado_base64", "referencia_portafirmas_declarada",
    "fecha_portafirmas_declarada", "clave_idempotencia"].sort());
  assert.equal(cuerpo.original_ref, solicitud.originalRef);
  assert.equal(cuerpo.original_version, 3);
  assert.equal(cuerpo.firmado_base64, Buffer.from(pdf).toString("base64"));
  assert.equal(Object.hasOwn(cuerpo, "actor_ref"), false);
});

test("replay conserva el recibo y rechaza una respuesta que afirma eficacia o procedencia verificada", async () => {
  const cliente = crearClienteFirmaExterna({ fetchImpl: async () => respuesta({ data: recibo(true) }, 200) });
  assert.equal((await cliente.registrar(solicitud)).ya_registrada, true);
  for (const cambio of [{ firma_eficaz: true }, { procedencia_portafirmas: { ...recibo().procedencia_portafirmas, estado: "verificada" } },
    { ya_registrada: true }, { verificacion_tecnica: { ...recibo().verificacion_tecnica, firmado_sha256: "e".repeat(64) } }]) {
    const rechazado = crearClienteFirmaExterna({ fetchImpl: async () => respuesta({ data: { ...recibo(), ...cambio } }) });
    await assert.rejects(rechazado.registrar(solicitud), (e) => e instanceof ErrorFirmaExterna && e.codigo === "resultado_no_confiable");
  }
});

test("el recibo exige tipos estrictos en referencias, custodia y verificación", async () => {
  const mutaciones = [
    { recibo_ref: null }, { firma_ref: 1234567890123456 }, { paso_ref: ["paso:resolucion:1"] },
    { documento_custodiado: { ...recibo().documento_custodiado, expediente_ref: [`ref:${"a".repeat(64)}`] } },
    { documento_custodiado: { ...recibo().documento_custodiado, huella_sha256: ["c".repeat(64)] } },
    { verificacion_tecnica: { ...recibo().verificacion_tecnica, original_sha256: ["d".repeat(64)] } },
    { verificacion_tecnica: { ...recibo().verificacion_tecnica, politica: null } },
    { verificacion_tecnica: { ...recibo().verificacion_tecnica, revocacion: null } },
    { verificacion_tecnica: { ...recibo().verificacion_tecnica, sello_tiempo: {} } },
  ];
  for (const cambio of mutaciones) {
    const cliente = crearClienteFirmaExterna({ fetchImpl: async () => respuesta({ data: { ...recibo(), ...cambio } }) });
    await assert.rejects(cliente.registrar(solicitud), (e) => e instanceof ErrorFirmaExterna && e.codigo === "resultado_no_confiable");
  }
});

test("denegación, conflicto y red caída no producen un recibo aparente", async () => {
  const casos = [[403, "acceso_denegado"], [409, "conflicto"], [422, "firma_no_verificada"]];
  for (const [status, codigo] of casos) {
    const cliente = crearClienteFirmaExterna({ fetchImpl: async () => respuesta({ error: {
      codigo, clave_i18n: `api.contratacion_temporal.registro_firma_externa.error.${codigo}`, correlacion_ref: "corr:001",
    } }, status) });
    await assert.rejects(cliente.registrar(solicitud), (e) => e instanceof ErrorFirmaExterna && e.codigo === codigo);
  }
  const sinRed = crearClienteFirmaExterna({ fetchImpl: async () => { throw new TypeError("red"); } });
  await assert.rejects(sinRed.registrar(solicitud), (e) => e.codigo === "servicio_no_disponible");
});

test("sin original autorizado, fecha válida o PDF firmado no hace peticiones", async () => {
  let llamadas = 0;
  const cliente = crearClienteFirmaExterna({ fetchImpl: async () => { llamadas += 1; return respuesta({ data: recibo() }); } });
  for (const cambio of [{ originalRef: "" }, { originalVersion: 0 }, { firmado: new Uint8Array([1, 2, 3]) },
    { fechaPortafirmas: "2026-10-02T10:15:00+02:00" }, { referenciaPortafirmas: " PF-2026-001" }]) {
    await assert.rejects(cliente.registrar({ ...solicitud, ...cambio }), (e) => e.codigo === "contenido_no_valido");
  }
  assert.equal(llamadas, 0);
});

const t = crearTraductorCircuitoFirma();

test("sin preflight R5, ningún DTO sintético abre la vista ni emite el POST externo", async () => {
  const html = renderizarAccionesPaso({ registro: { verificacion: true, externo: {
    permitido: true, original_ref: solicitud.originalRef, original_version: 3,
  } } }, { documento: "resolucion", paso_pendiente: 1 }, { orden: 1 }, t);
  assert.match(html, /Firmar en PRUEBA<\/button>/u);
  assert.match(html, /<button[^>]*disabled[^>]*>Firmar en PRUEBA/u);
  assert.match(html, /<button[^>]*disabled[^>]*>Devolver en PRUEBA/u);
  assert.doesNotMatch(html, /data-ct-firma-accion|type="file"/u);
  let llamados = 0;
  const acciones = crearAccionesFirma({ clienteExterno: { registrar: async () => { llamados++; } },
    clienteFirma: { registrar: async () => { llamados++; } },
    autofirma: { firmarPDF: async () => { llamados++; } } });
  await acciones.manejarClic({ target: { closest: () => ({ dataset: { ctFirmaAccion: "firmar" } }) } });
  await acciones.manejarClic({ target: { closest: () => ({ dataset: { ctFirmaAccion: "confirmar-externo" } }) } });
  assert.equal(llamados, 0);
});


test("V2 exige revisión incremental y rechaza recibos legacy o revisión de otro paso", async () => {
  for (const cambiar of [(d) => { d.esquema = "vec.contratacion-temporal.registro-firma-externa.v1"; },
    (d) => { delete d.revision_pdf; }, (d) => { d.revision_pdf.orden_firma = 2; },
    (d) => { d.revision_pdf.entrada_sha256 = "a".repeat(64); }, (d) => { d.revision_pdf.revision_sha256 = "a".repeat(64); }]) {
    const d = recibo(); cambiar(d);
    await assert.rejects(crearClienteFirmaExterna({ fetchImpl: async () => respuesta({ data: d }) }).registrar(solicitud), (e) => e.codigo === "resultado_no_confiable");
  }
});
