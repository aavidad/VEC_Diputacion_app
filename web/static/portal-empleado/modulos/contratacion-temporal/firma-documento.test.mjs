import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearClienteAutoFirma, ErrorAutoFirma, paraPruebas } from "./firma-autofirma.js";
import { crearClienteFirmaDocumento, ErrorFirmaDocumento, RUTA_CONSULTA_FIRMA_DOCUMENTO, RUTA_FIRMA_DOCUMENTO, validarEstadoFirmas } from "./firma-documento-cliente.js";
import { crearAccionesFirma, fusionarEstadoFirmas, renderizarAccionesPaso } from "./circuito-firma-acciones.js";
import { crearTraductorCircuitoFirma } from "./i18n-circuito-firma.js";

const t = crearTraductorCircuitoFirma();
const PDF = new TextEncoder().encode("%PDF-1.7 borrador");
const FIRMADO = new TextEncoder().encode("%PDF-1.7 borrador + firma PAdES");
const temporizadorInmediato = { setTimeout: (f) => setTimeout(f, 0), clearTimeout };

function socketFalso(respuestaFirma, { abrir = true } = {}) {
  const enviados = [];
  class WebSocketFalso {
    constructor(url) {
      this.url = url;
      setTimeout(() => (abrir ? this.onopen?.() : this.onerror?.()), 0);
    }
    send(texto) {
      enviados.push(texto);
      setTimeout(() => this.onmessage?.({ data: texto.startsWith("echo=") ? "OK" : respuestaFirma }), 0);
    }
    close() { this.cerrado = true; }
  }
  return { WebSocketFalso, enviados };
}

test("AutoFirma: lanza el protocolo, saluda y devuelve el PDF firmado", async () => {
  const lanzados = [];
  const { WebSocketFalso, enviados } = socketFalso(`Y2VydA|${paraPruebas.base64URL(FIRMADO)}`);
  const cliente = crearClienteAutoFirma({
    WebSocketImpl: WebSocketFalso, lanzar: (url) => lanzados.push(url), aleatorio: (n) => new Uint8Array(n).fill(3),
    temporizador: temporizadorInmediato, intentos: 2, esperaReintentoMs: 1,
  });
  const firmado = await cliente.firmarPDF(PDF);
  assert.deepEqual([...firmado], [...FIRMADO]);
  assert.match(lanzados[0], /^afirma:\/\/websocket\?v=4&idsession=[A-Za-z0-9]{20}&ports=63117$/u);
  assert.match(enviados[0], /^echo=-idsession=[A-Za-z0-9]{20}@EOF$/u);
  assert.match(enviados[1], /^afirma:\/\/sign\?op=sign&id=.*&format=PAdES&algorithm=SHA256withRSA&dat=[A-Za-z0-9_-]+$/u);
  assert.equal(enviados[1].split("dat=")[1], paraPruebas.base64URL(PDF));
});

test("AutoFirma: cancelación, error de firma, respuesta ajena y sin conexión", async () => {
  const casos = [["CANCEL", "firma_cancelada"], ["SAF_09: Error en la operacion de firma", "firma_fallida"],
    ["solo-un-campo", "respuesta_no_valida"], [`Y2VydA|${paraPruebas.base64URL(new TextEncoder().encode("no es pdf"))}`, "respuesta_no_valida"]];
  for (const [respuesta, codigo] of casos) {
    const { WebSocketFalso } = socketFalso(respuesta);
    const cliente = crearClienteAutoFirma({ WebSocketImpl: WebSocketFalso, lanzar: () => {}, aleatorio: (n) => new Uint8Array(n), temporizador: temporizadorInmediato, intentos: 1, esperaReintentoMs: 1 });
    await assert.rejects(cliente.firmarPDF(PDF), (e) => e instanceof ErrorAutoFirma && e.codigo === codigo);
  }
  const { WebSocketFalso } = socketFalso("", { abrir: false });
  const cliente = crearClienteAutoFirma({ WebSocketImpl: WebSocketFalso, lanzar: () => {}, aleatorio: (n) => new Uint8Array(n), temporizador: temporizadorInmediato, intentos: 2, esperaReintentoMs: 1 });
  await assert.rejects(cliente.firmarPDF(PDF), (e) => e.codigo === "autofirma_no_disponible");
  await assert.rejects(cliente.firmarPDF(new TextEncoder().encode("no pdf")), (e) => e.codigo === "documento_no_valido");
});

function estadoServidor(extra = {}) {
  return {
    esquema: "vec.contratacion-temporal.estado-firmas-documento.v1", catalogo_ref: "vec.contratacion_temporal.circuito_firma:1",
    huella_sha256: "a".repeat(64), ejemplo: true, firma_eficaz: false, verificacion_disponible: true,
    documentos: [{ documento: "informe_definitivo", etiqueta: "Informe definitivo", paso_pendiente: 2, completo: false, ultima_secuencia: 1,
      original_esperado_sha256: "b".repeat(64),
      pasos: [{ orden: 1, cargo: "Técnico", accion: "firma", devolucion: "vuelve_a_redaccion", estado: "firmado", recibo_ref: "recibo-firma-ct:1", registrada_en: "2026-09-25T10:00:00Z" },
        { orden: 2, cargo: "Jefatura", accion: "firma", devolucion: "vuelve_paso_anterior", estado: "pendiente_firma" }] }],
    ...extra,
  };
}

function respuesta(cuerpo, estado = 200) {
  return new Response(JSON.stringify(cuerpo), { status: estado, headers: { "Content-Type": "application/json; charset=utf-8" } });
}

test("cliente del registro: consulta con contrato exacto y falla cerrado", async () => {
  assert.ok(validarEstadoFirmas(estadoServidor()));
  assert.equal(validarEstadoFirmas(estadoServidor({ firma_eficaz: true })), null);
  assert.equal(validarEstadoFirmas(estadoServidor({ extra: 1 })), null);
  let peticion;
  const cliente = crearClienteFirmaDocumento({ fetchImpl: async (ruta, opciones) => { peticion = { ruta, opciones }; return respuesta({ data: estadoServidor() }); } });
  const estado = await cliente.consultar("expediente:ct:001");
  assert.equal(peticion.ruta, RUTA_CONSULTA_FIRMA_DOCUMENTO);
  assert.equal(peticion.opciones.method, "POST");
  assert.equal(peticion.opciones.credentials, "same-origin");
  assert.equal(estado.documentos[0].paso_pendiente, 2);
  const sinRegistro = crearClienteFirmaDocumento({ fetchImpl: async () => respuesta({ error: { codigo: "recurso_no_encontrado", clave_i18n: "x", correlacion_ref: "corr_no_disponible" } }, 404) });
  assert.equal(await sinRegistro.consultar("expediente:ct:001"), null);
});

test("cliente del registro: firma, devolución y errores con motivo del validador", async () => {
  const recibo = {
    esquema: "vec.contratacion-temporal.recibo-firma-documento.v1", firma_ref: "firma-ct:1", recibo_ref: "recibo-firma-ct:1",
    expediente_ref: "expediente:ct:001", expediente_version: 7, documento: "informe_definitivo", paso_orden: 2, paso_ref: "c:1:p2",
    catalogo_ref: "c:1", catalogo_huella_sha256: "a".repeat(64), secuencia: 2, resultado: "firmado", perfil_ref: "perfil:x",
    registrada_en: "2026-09-25T10:00:00Z", ya_registrada: false, firma_verificada: true, firma_eficaz: false,
    verificacion: { estado: "valida", motivo: "verificada", politica: "p", certificado_sha256: "e".repeat(64), revocacion: "vigente",
      sello_tiempo: "no_presente", original_sha256: "b".repeat(64), firmado_sha256: "c".repeat(64) },
  };
  let cuerpo;
  const cliente = crearClienteFirmaDocumento({ fetchImpl: async (ruta, opciones) => { cuerpo = JSON.parse(opciones.body); assert.equal(ruta, RUTA_FIRMA_DOCUMENTO); return respuesta({ data: recibo }, 201); } });
  const r = await cliente.registrar({ expedienteRef: "expediente:ct:001", version: 7, documento: "informe_definitivo", pasoOrden: 2,
    resultado: "firmado", original: PDF, firmado: FIRMADO, clave: "firma-0123456789abcdef" });
  assert.equal(r.firma_eficaz, false);
  assert.equal(cuerpo.original_base64, Buffer.from(PDF).toString("base64"));
  assert.equal(Object.hasOwn(cuerpo, "actor_ref"), false);
  const rechazo = crearClienteFirmaDocumento({ fetchImpl: async () => respuesta({ error: { codigo: "firma_no_verificada", clave_i18n: "x", motivo: "revocacion_no_acreditada", correlacion_ref: "corr_no_disponible" } }, 422) });
  await assert.rejects(rechazo.registrar({ expedienteRef: "expediente:ct:001", version: 7, documento: "informe_definitivo", pasoOrden: 2,
    resultado: "firmado", original: PDF, firmado: FIRMADO, clave: "firma-0123456789abcdef" }),
  (e) => e instanceof ErrorFirmaDocumento && e.codigo === "firma_no_verificada" && e.motivo === "revocacion_no_acreditada");
  const falsa = crearClienteFirmaDocumento({ fetchImpl: async () => respuesta({ data: { ...recibo, firma_eficaz: true } }, 201) });
  await assert.rejects(falsa.registrar({ expedienteRef: "expediente:ct:001", version: 7, documento: "informe_definitivo", pasoOrden: 2,
    resultado: "firmado", original: PDF, firmado: FIRMADO, clave: "firma-0123456789abcdef" }), (e) => e.codigo === "resultado_no_confiable");
});

function circuitoCatalogo() {
  return Object.freeze({ catalogo_ref: "c:1", huella_sha256: "a".repeat(64), ejemplo: true, documentos: [{ documento: "informe_definitivo", etiqueta: "Informe definitivo", pasos: [
    { orden: 1, cargo: "Técnico", estado: "pendiente_firma" }, { orden: 2, cargo: "Jefatura", estado: "en_espera" }] }] });
}

test("fusiona el estado real y solo ofrece acciones en el paso pendiente", () => {
  const fusion = fusionarEstadoFirmas(circuitoCatalogo(), validarEstadoFirmas(estadoServidor()));
  assert.equal(fusion.documentos[0].pasos[0].estado, "firmado");
  assert.equal(fusion.documentos[0].pasos[1].estado, "pendiente_firma");
  assert.equal(renderizarAccionesPaso(fusion, fusion.documentos[0], fusion.documentos[0].pasos[0], t), "");
  const html = renderizarAccionesPaso(fusion, fusion.documentos[0], fusion.documentos[0].pasos[1], t);
  assert.match(html, /data-ct-firma-accion="firmar"/u);
  assert.match(html, /data-ct-firma-accion="devolver"[^>]*aria-expanded="false"/u);
  assert.match(html, /<label for="ct-firma-motivo-informe_definitivo-2-texto">/u);
  assert.equal(fusionarEstadoFirmas(circuitoCatalogo(), validarEstadoFirmas(estadoServidor({ huella_sha256: "f".repeat(64) }))), null);
  assert.equal(renderizarAccionesPaso(circuitoCatalogo(), circuitoCatalogo().documentos[0], circuitoCatalogo().documentos[0].pasos[0], t), "");
});

function estadoExpediente() {
  const ref = "expediente:ct:001";
  return { vista: "expediente", carga: "listo", expediente_ref: ref, expediente: { expediente_ref: ref, version: 7, demostracion: false },
    cuadro: { demostracion: false, expedientes: [{ expediente_ref: ref, version: 7, fase_clave: "nombramiento", estado_clave: "en_curso" }] } };
}

function dom(motivo = "") {
  const salida = { textContent: "" };
  const botones = [{ disabled: false }, { disabled: false }];
  const contenedor = {
    querySelector: (s) => (s === "[data-ct-firma-resultado]" ? salida : s === "[data-ct-firma-motivo]" ? { value: motivo } : null),
    querySelectorAll: () => botones,
  };
  const boton = (accion) => ({ dataset: { ctFirmaAccion: accion, ctFirmaDocumento: "informe_definitivo", ctFirmaOrden: "2" },
    closest: (s) => (s === "[data-ct-firma-accion]" ? boton(accion) : contenedor) });
  return { salida, evento: (accion) => ({ target: { closest: () => boton(accion) } }) };
}

test("acciones: firma con AutoFirma y registro; devolución con motivo", async () => {
  const registrados = [];
  let avisos = 0;
  const acciones = crearAccionesFirma({
    obtenerEstado: estadoExpediente, t, alCambiar: async () => { avisos += 1; }, aleatorio: (n) => new Uint8Array(n).fill(1),
    clienteBorrador: { descargarBorrador: async (sol, op) => { assert.deepEqual(sol, { expediente_ref: "expediente:ct:001", version_observada: 7 }); assert.equal(op.tipo, "informe_definitivo"); return new Blob([PDF]); } },
    autofirma: { firmarPDF: async (pdf) => { assert.deepEqual([...pdf], [...PDF]); return FIRMADO; } },
    clienteFirma: { registrar: async (s) => { registrados.push(s); return { recibo_ref: "recibo-firma-ct:9" }; } },
  });
  const firma = dom();
  await acciones.manejarClic(firma.evento("firmar"));
  assert.equal(registrados[0].resultado, "firmado");
  assert.equal(registrados[0].pasoOrden, 2);
  assert.match(registrados[0].clave, /^firma-[0-9a-f]{32}$/u);
  assert.match(firma.salida.textContent, /recibo-firma-ct:9.*sin eficacia|No tiene eficacia administrativa/u);
  const corto = dom("no");
  await acciones.manejarClic(corto.evento("confirmar-devolucion"));
  assert.equal(corto.salida.textContent, t("circuito_firma_motivo_invalido"));
  const devolucion = dom("Falta la fecha de efectos");
  await acciones.manejarClic(devolucion.evento("confirmar-devolucion"));
  assert.equal(registrados[1].resultado, "devuelto");
  assert.equal(registrados[1].motivoDevolucion, "Falta la fecha de efectos");
  assert.equal(avisos, 2);
});

test("acciones: la verificación apagada y el rechazo del validador se explican", async () => {
  for (const [error, clave] of [[new ErrorFirmaDocumento("verificacion_no_disponible"), "circuito_firma_error_verificacion"],
    [new ErrorFirmaDocumento("firma_no_verificada", "integridad_no_valida"), "circuito_firma_error_no_verificada"],
    [new ErrorAutoFirma("autofirma_no_disponible"), "circuito_firma_error_autofirma"]]) {
    const acciones = crearAccionesFirma({
      obtenerEstado: estadoExpediente, t, aleatorio: (n) => new Uint8Array(n),
      clienteBorrador: { descargarBorrador: async () => new Blob([PDF]) },
      autofirma: { firmarPDF: async () => { if (error instanceof ErrorAutoFirma) throw error; return FIRMADO; } },
      clienteFirma: { registrar: async () => { throw error; } },
    });
    const d = dom();
    await acciones.manejarClic(d.evento("firmar"));
    assert.equal(d.salida.textContent, t(clave, { motivo: error.motivo || error.codigo }));
  }
});

test("la ayuda y los textos dicen que la firma no tiene eficacia administrativa", async () => {
  const ayuda = await readFile(new URL("../../portal-i18n-ayuda.js", import.meta.url), "utf8");
  assert.match(ayuda, /ayuda_contenido_404: ".*AutoFirma.*sin eficacia administrativa.*portafirmas corporativo/u);
  assert.match(t("circuito_firma_registrada", { recibo: "r" }), /No tiene eficacia administrativa hasta el portafirmas corporativo/u);
});
