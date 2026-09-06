import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { crearClienteHTTPInformeDefinitivo } from "./cliente-http-informe-definitivo.js";

const solicitud = { expediente_ref: "expediente:ct:sintetico-009", version_observada: 7 };
const pdf = "%PDF-1.7\nBorrador sintético\n%%EOF";
function respuestaPDF(contenido = pdf, cabeceras = {}, status = 200) {
  return new Response(contenido, { status, headers: {
    "Content-Type": "application/pdf",
    "Content-Disposition": 'attachment; filename="informe-definitivo-borrador.pdf"',
    ...cabeceras,
  } });
}
function cliente(respuesta) {
  return crearClienteHTTPInformeDefinitivo({ fetchImpl: async () => respuesta });
}

test("POST de consulta: dos campos ordenados, representación nominal y PDF acotado", async () => {
  const llamadas = [];
  const http = crearClienteHTTPInformeDefinitivo({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return respuestaPDF(pdf, { "Content-Length": String(new TextEncoder().encode(pdf).length) });
  } });
  const blob = await http.descargarInformeDefinitivo(solicitud);
  assert.equal(blob.type, "application/pdf");
  assert.equal(await blob.text(), pdf);
  assert.equal(llamadas.length, 1);
  const { ruta, opciones } = llamadas[0];
  assert.equal(ruta, "/api/vec/contratacion-temporal/expedientes/consultas");
  assert.equal(opciones.method, "POST");
  assert.equal(opciones.body, JSON.stringify(solicitud));
  assert.deepEqual(opciones.headers, {
    "Content-Type": "application/json",
    Accept: "application/pdf; documento=informe-definitivo-desarrollo",
  });
  assert.equal(opciones.mode, "same-origin");
  assert.equal(opciones.credentials, "same-origin");
  assert.equal(opciones.cache, "no-store");
  assert.equal(opciones.redirect, "error");
  assert.equal(opciones.referrerPolicy, "no-referrer");
});

test("rechaza contrato alterado antes de red", async () => {
  let llamadas = 0;
  const http = crearClienteHTTPInformeDefinitivo({ fetchImpl: () => { llamadas += 1; } });
  for (const entrada of [
    { ...solicitud, version_observada: 6 }, { ...solicitud, actor: "rrhh" },
    { ...solicitud, expediente_ref: "no válido" }, { expediente_ref: solicitud.expediente_ref },
  ]) await assert.rejects(http.descargarInformeDefinitivo(entrada), { codigo: "solicitud_no_valida" });
  assert.equal(llamadas, 0);
});

test("no convierte JSON/HTML, nombre ajeno, exceso ni cuerpo truncado en descarga", async () => {
  for (const respuesta of [
    respuestaPDF("<html>error</html>"), respuestaPDF(pdf, { "Content-Type": "application/json" }),
    respuestaPDF(pdf, { "Content-Disposition": 'attachment; filename="otro.pdf"' }),
    respuestaPDF(pdf, { "Content-Length": "2097153" }),
    respuestaPDF(pdf, { "Content-Length": "12" }),
    respuestaPDF(new Uint8Array(2097153)),
  ]) await assert.rejects(cliente(respuesta).descargarInformeDefinitivo(solicitud), { codigo: "resultado_no_confiable" });
});

test("errores JSON cerrados de consulta, incluido 409; no reintenta", async () => {
  for (const [estado, codigo] of [[403, "acceso_denegado"], [409, "documento_no_disponible"], [503, "servicio_no_disponible"]]) {
    const error = { codigo, clave_i18n: `api.contratacion_temporal.consulta_rrhh.error.${codigo}`, correlacion_ref: "corr_" + "a".repeat(32) };
    await assert.rejects(cliente(Response.json({ error }, { status: estado })).descargarInformeDefinitivo(solicitud), {
      codigo, estado, envelopeValido: true, resultadoIndeterminado: false,
    });
    error.clave_i18n = `otra.error.${codigo}`;
    await assert.rejects(cliente(Response.json({ error }, { status: estado })).descargarInformeDefinitivo(solicitud), {
      codigo: "respuesta_error_no_valida",
    });
  }
  await assert.rejects(cliente(respuestaPDF(pdf, {}, 201)).descargarInformeDefinitivo(solicitud));
});

test("cancelación antes de fetch, durante fetch tardío y durante lectura", async () => {
  const previo = new AbortController(); previo.abort();
  let llamadas = 0;
  await assert.rejects(crearClienteHTTPInformeDefinitivo({ fetchImpl: () => { llamadas += 1; } })
    .descargarInformeDefinitivo(solicitud, { signal: previo.signal }), { name: "AbortError" });
  assert.equal(llamadas, 0);
  let completar;
  const controlador = new AbortController();
  const pendiente = crearClienteHTTPInformeDefinitivo({ fetchImpl: () => new Promise((resolve) => { completar = resolve; }) })
    .descargarInformeDefinitivo(solicitud, { signal: controlador.signal });
  controlador.abort();
  await assert.rejects(pendiente, { name: "AbortError" });
  const tardia = respuestaPDF(); completar(tardia);
  await new Promise(setImmediate);
  assert.equal(tardia.bodyUsed, true);
  let cancelado = false;
  const leyendo = new AbortController();
  const lectura = cliente(respuestaPDF(new ReadableStream({
    pull() { leyendo.abort(); }, cancel() { cancelado = true; },
  }))).descargarInformeDefinitivo(solicitud, { signal: leyendo.signal });
  await assert.rejects(lectura, { name: "AbortError" });
  assert.equal(cancelado, true);
});

test("manifiesto activo contiene solo el cliente necesario; sin generador de presentación ni almacenamiento", async () => {
  const manifiesto = await readFile(new URL("../../../../produccion.manifest", import.meta.url), "utf8");
  assert.ok(manifiesto.split("\n").includes("static/portal-empleado/modulos/contratacion-temporal/cliente-http-informe-definitivo.js"));
  const fuente = await readFile(new URL("cliente-http-informe-definitivo.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /recibo-pdf-presentacion|localStorage|sessionStorage|indexedDB|document\.cookie/u);
});
