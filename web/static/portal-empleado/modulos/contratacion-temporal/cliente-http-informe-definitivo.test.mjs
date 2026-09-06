import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { crearClienteHTTPBorradorRRHH } from "./cliente-http-informe-definitivo.js";

const solicitud = { expediente_ref: "expediente:ct:sintetico-009", version_observada: 7 };
const pdf = "%PDF-1.7\nBorrador sintético\n%%EOF";
const perfiles = [
  { tipo: "informe_definitivo", accept: "application/pdf; documento=informe-definitivo-desarrollo", nombre: "informe-definitivo-borrador.pdf" },
  { tipo: "resolucion", accept: "application/pdf; documento=resolucion-desarrollo", nombre: "resolucion-borrador.pdf" },
  { tipo: "diligencia", accept: "application/pdf; documento=diligencia-desarrollo", nombre: "diligencia-borrador.pdf" },
  { tipo: "toma_posesion", accept: "application/pdf; documento=toma-posesion-desarrollo", nombre: "toma-posesion-borrador.pdf" },
];
function respuestaPDF(contenido = pdf, cabeceras = {}, status = 200) {
  return new Response(contenido, { status, headers: {
    "Content-Type": "application/pdf",
    "Content-Disposition": 'attachment; filename="informe-definitivo-borrador.pdf"',
    ...cabeceras,
  } });
}
function cliente(respuesta) {
  return crearClienteHTTPBorradorRRHH({ fetchImpl: async () => respuesta });
}

test("POST de consulta: dos campos ordenados, representación nominal y PDF acotado", async () => {
  const llamadas = [];
  const http = crearClienteHTTPBorradorRRHH({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return respuestaPDF(pdf, { "Content-Length": String(new TextEncoder().encode(pdf).length) });
  } });
  const blob = await http.descargarBorrador(solicitud);
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
  const http = crearClienteHTTPBorradorRRHH({ fetchImpl: () => { llamadas += 1; } });
  for (const entrada of [
    { ...solicitud, version_observada: 6 }, { ...solicitud, actor: "rrhh" },
    { ...solicitud, expediente_ref: "no válido" }, { expediente_ref: solicitud.expediente_ref },
  ]) await assert.rejects(http.descargarBorrador(entrada), { codigo: "solicitud_no_valida" });
  for (const tipo of ["", "otro", "constructor", "__proto__", null, {}, new String("resolucion")]) {
    await assert.rejects(http.descargarBorrador(solicitud, { tipo }), { codigo: "solicitud_no_valida" });
  }
  assert.equal(llamadas, 0);
});

test("cuatro perfiles nominales usan la misma consulta; no intercambian documentos", async () => {
  for (const { tipo, accept, nombre } of perfiles) {
    const llamadas = [];
    const cabeceras = { "Content-Disposition": `attachment; filename="${nombre}"` };
    const http = crearClienteHTTPBorradorRRHH({ fetchImpl: async (ruta, opciones) => {
      llamadas.push({ ruta, opciones });
      return respuestaPDF(pdf, cabeceras);
    } });
    const blob = await http.descargarBorrador(solicitud, { tipo });
    assert.equal(await blob.text(), pdf);
    assert.equal(blob.type, "application/pdf");
    assert.equal(llamadas.length, 1);
    assert.equal(llamadas[0].ruta, "/api/vec/contratacion-temporal/expedientes/consultas");
    assert.equal(llamadas[0].opciones.method, "POST");
    assert.equal(llamadas[0].opciones.body, JSON.stringify(solicitud));
    assert.deepEqual(llamadas[0].opciones.headers, { "Content-Type": "application/json", Accept: accept });
    for (const ajeno of perfiles.filter((perfil) => perfil.tipo !== tipo)) {
      await assert.rejects(cliente(respuestaPDF(pdf, {
        "Content-Disposition": `attachment; filename="${ajeno.nombre}"`,
      })).descargarBorrador(solicitud, { tipo }), { codigo: "resultado_no_confiable" });
    }
  }
});

test("no convierte JSON/HTML, nombre ajeno, exceso ni cuerpo truncado en descarga", async () => {
  for (const { tipo, nombre } of perfiles) {
    const nominales = { "Content-Disposition": `attachment; filename="${nombre}"` };
    for (const respuesta of [
      respuestaPDF("<html>error</html>", nominales), respuestaPDF(pdf, { ...nominales, "Content-Type": "application/json" }),
      respuestaPDF(pdf, { "Content-Disposition": 'attachment; filename="otro.pdf"' }),
      respuestaPDF(pdf, { ...nominales, "Content-Length": "2097153" }),
      respuestaPDF(pdf, { ...nominales, "Content-Length": "12" }),
      respuestaPDF(new Uint8Array(2097153), nominales),
    ]) await assert.rejects(cliente(respuesta).descargarBorrador(solicitud, { tipo }), { codigo: "resultado_no_confiable" });
  }
});

test("errores JSON cerrados de consulta, incluido 409; no reintenta", async () => {
  for (const [estado, codigo] of [[403, "acceso_denegado"], [409, "documento_no_disponible"], [503, "servicio_no_disponible"]]) {
    for (const { tipo } of perfiles) {
      const error = { codigo, clave_i18n: `api.contratacion_temporal.consulta_rrhh.error.${codigo}`, correlacion_ref: "corr_" + "a".repeat(32) };
      await assert.rejects(cliente(Response.json({ error }, { status: estado })).descargarBorrador(solicitud, { tipo }), {
        codigo, estado, envelopeValido: true, resultadoIndeterminado: false,
      });
      error.clave_i18n = `otra.error.${codigo}`;
      await assert.rejects(cliente(Response.json({ error }, { status: estado })).descargarBorrador(solicitud, { tipo }), {
        codigo: "respuesta_error_no_valida",
      });
    }
  }
  await assert.rejects(cliente(respuestaPDF(pdf, {}, 201)).descargarBorrador(solicitud));
});

test("cancelación antes de fetch, durante fetch tardío y durante lectura", async () => {
  const previo = new AbortController(); previo.abort();
  let llamadas = 0;
  await assert.rejects(crearClienteHTTPBorradorRRHH({ fetchImpl: () => { llamadas += 1; } })
    .descargarBorrador(solicitud, { signal: previo.signal }), { name: "AbortError" });
  assert.equal(llamadas, 0);
  let completar;
  const controlador = new AbortController();
  const pendiente = crearClienteHTTPBorradorRRHH({ fetchImpl: () => new Promise((resolve) => { completar = resolve; }) })
    .descargarBorrador(solicitud, { signal: controlador.signal });
  controlador.abort();
  await assert.rejects(pendiente, { name: "AbortError" });
  const tardia = respuestaPDF(); completar(tardia);
  await new Promise(setImmediate);
  assert.equal(tardia.bodyUsed, true);
  let cancelado = false;
  const leyendo = new AbortController();
  const lectura = cliente(respuestaPDF(new ReadableStream({
    pull() { leyendo.abort(); }, cancel() { cancelado = true; },
  }))).descargarBorrador(solicitud, { signal: leyendo.signal });
  await assert.rejects(lectura, { name: "AbortError" });
  assert.equal(cancelado, true);
});

test("manifiesto activo contiene solo el cliente necesario; sin generador de presentación ni almacenamiento", async () => {
  const manifiesto = await readFile(new URL("../../../../produccion.manifest", import.meta.url), "utf8");
  assert.ok(manifiesto.split("\n").includes("static/portal-empleado/modulos/contratacion-temporal/cliente-http-informe-definitivo.js"));
  const fuente = await readFile(new URL("cliente-http-informe-definitivo.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /recibo-pdf-presentacion|localStorage|sessionStorage|indexedDB|document\.cookie/u);
});
