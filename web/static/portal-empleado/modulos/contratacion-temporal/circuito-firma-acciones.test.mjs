import assert from "node:assert/strict";
import test from "node:test";
import { crearAccionesFirma, renderizarAccionesPaso } from "./circuito-firma-acciones.js";
import { crearTraductorCircuitoFirma } from "./i18n-circuito-firma.js?v=20261001-ct-a-i18n-v1";

const pdf = new TextEncoder().encode("%PDF-1.7\noriginal con primera firma\n%%EOF");
function escenario(vias = ["certificado_vec", "portafirmas_registro_rrhh"], pasoOrden = 2) {
  const estado = { vista: "expediente", carga: "listo", expediente: { expediente_ref: "expediente:prueba", version: 7, demostracion: false } };
  const circuito = { preflight_compuesto: true, catalogo_ref: "catalogo:prueba", huella_sha256: "a".repeat(64),
    registro: { verificacion: true }, documentos: [{ documento: "resolucion", paso_pendiente: pasoOrden }] };
  const aviso = { textContent: "", focus() {} };
  const campos = new Map();
  const bloque = { innerHTML: "", dataset: { ctFirmaDocumento: "resolucion", ctFirmaPaso: String(pasoOrden) },
    setAttribute() {}, querySelectorAll: () => [],
    querySelector: (s) => s === "[data-ct-firma-resultado]" ? aviso : campos.get(s) ?? null };
  const preflight = { version_expediente: 7, documento: "resolucion", catalogo_ref: "catalogo:prueba", catalogo_huella: "a".repeat(64),
    paso_pendiente: pasoOrden, original_ref: "original:raiz", original_version: 2, vias_disponibles: vias };
  const contador = { originales: 0, firmas: 0, registros: 0, confirmaciones: 0 };
  const recibo = { firma_eficaz: false, recibo_ref: "recibo:prueba", expediente_ref: "expediente:prueba",
    documento: "resolucion", paso_orden: pasoOrden, version_expediente: 7, firma_verificada: true };
  const deps = { obtenerEstado: () => estado, obtenerCircuito: () => circuito, t: crearTraductorCircuitoFirma(),
    clientePreflight: { consultar: async () => preflight },
    obtenerVinculoOriginal: async () => ({ originalRef: "original:raiz", originalVersion: 2, revisionEntradaRef: "revision:firmada-p1", revisionEntradaVersion: 1, revisionEntradaHuella: "d".repeat(64) }),
    obtenerOriginal: async () => { contador.originales++; return pdf; },
    autofirma: { firmarPDF: async (original) => { assert.equal(original, pdf); contador.firmas++; return new TextEncoder().encode("%PDF-1.7\noriginal con primera firma\nsegunda firma\n%%EOF"); } },
    registrarVec: async () => { contador.registros++; return recibo; },
    registrarExterna: async () => { contador.registros++; return recibo; },
    alCambiar: async () => { contador.confirmaciones++; }, crearClave: () => "clave-prueba-123456789" };
  const evento = (accion) => { const b = { dataset: { ctFirmaAccion: accion }, closest: (s) => s === "[data-ct-firma-controles]" ? bloque : b };
    return { target: { closest: () => b } }; };
  return { estado, circuito, bloque, aviso, campos, preflight, contador, recibo, deps, evento };
}
test("solo la composición explícita ofrece consulta, sin firma activa predeterminada", () => {
  const e = escenario(); const documento = { documento: "resolucion", paso_pendiente: 2 };
  const html = renderizarAccionesPaso(e.circuito, documento, { orden: 2 }, e.deps.t);
  assert.match(html, /data-ct-firma-accion="comprobar"/u);
  assert.doesNotMatch(html, /data-ct-firma-accion="certificado_vec"/u);
  assert.doesNotMatch(renderizarAccionesPaso({ ...e.circuito, preflight_compuesto: false }, documento, { orden: 2 }, e.deps.t), /data-ct-firma-accion/u);
  assert.equal(renderizarAccionesPaso({ ...e.circuito, acciones: false }, documento, { orden: 2 }, e.deps.t), "");
});
test("preflight nominal permite sólo las vías devueltas, sin inferirlas del verificador", async () => {
  for (const vias of [[], ["certificado_vec"], ["portafirmas_registro_rrhh"]]) {
    const e = escenario(vias); const acciones = crearAccionesFirma(e.deps);
    await acciones.manejarClic(e.evento("comprobar"));
    assert.equal(e.bloque.innerHTML.includes('data-ct-firma-accion="certificado_vec"'), vias.includes("certificado_vec"));
    assert.equal(e.bloque.innerHTML.includes("data-ct-firma-pdf"), vias.includes("portafirmas_registro_rrhh"));
    await acciones.manejarClic(e.evento(vias.includes("certificado_vec") ? "portafirmas_registro_rrhh" : "certificado_vec"));
    assert.equal(e.contador.registros, 0);
  }
});
test("el paso 2 firma los bytes de la custodia previa y confirma sólo recibo vinculado", async () => {
  const e = escenario(); let enviados;
  e.deps.registrarVec = async (s) => { enviados = s; return e.recibo; };
  const acciones = crearAccionesFirma(e.deps);
  await acciones.manejarClic(e.evento("certificado_vec")); assert.equal(e.contador.firmas, 0);
  await acciones.manejarClic(e.evento("comprobar"));
  await acciones.manejarClic(e.evento("certificado_vec"));
  assert.equal(enviados.originalRef, "original:raiz"); assert.equal(enviados.originalVersion, 2);
  assert.equal(enviados.revisionEntradaRef, "revision:firmada-p1");
  assert.equal(enviados.original, pdf); assert.equal(e.contador.confirmaciones, 1);
  assert.match(e.aviso.textContent, /Firma verificada y registrada/u);
});
test("fallo de confirmación reintenta los mismos bytes y clave sin otra firma", async () => {
  const e = escenario(); const peticiones = [];
  e.deps.registrarVec = async (s) => { peticiones.push(s); if (peticiones.length === 1) throw { codigo: "servicio_no_disponible" }; return e.recibo; };
  const acciones = crearAccionesFirma(e.deps);
  await acciones.manejarClic(e.evento("comprobar")); await acciones.manejarClic(e.evento("certificado_vec"));
  assert.equal(e.contador.confirmaciones, 0); assert.match(e.aviso.textContent, /Reintente con el mismo PDF/u);
  await acciones.manejarClic(e.evento("certificado_vec"));
  assert.equal(e.contador.firmas, 1); assert.equal(peticiones[0], peticiones[1]); assert.equal(e.contador.confirmaciones, 1);
});
test("consulta rota, ejemplo, huella ajena o recibo ajeno cierran efectos y confirmación", async () => {
  for (const alterar of [(e) => { e.estado.expediente.demostracion = true; }, (e) => { e.circuito.registro = null; },
    (e) => { e.preflight.catalogo_huella = "b".repeat(64); }, (e) => { e.preflight.paso_pendiente = 1; },
    (e) => { e.deps.clientePreflight.consultar = async () => { throw { codigo: "acceso_denegado" }; }; }]) {
    const e = escenario(); alterar(e); const a = crearAccionesFirma(e.deps);
    await a.manejarClic(e.evento("comprobar")); await a.manejarClic(e.evento("certificado_vec"));
    assert.equal(e.contador.firmas, 0); assert.equal(e.contador.confirmaciones, 0);
  }
  const e = escenario(); e.recibo.documento = "diligencia"; const a = crearAccionesFirma(e.deps);
  await a.manejarClic(e.evento("comprobar")); await a.manejarClic(e.evento("certificado_vec"));
  assert.equal(e.contador.confirmaciones, 0); assert.doesNotMatch(e.aviso.textContent, /Firma verificada y registrada/u);
});
test("cambio de expediente cancela descarga/firma y descarta respuesta tardía", async () => {
  const e = escenario(); let terminar, signal;
  e.deps.obtenerOriginal = async (_, opciones) => { signal = opciones.signal; return new Promise((r) => { terminar = r; }); };
  const a = crearAccionesFirma(e.deps); await a.manejarClic(e.evento("comprobar"));
  const firma = a.manejarClic(e.evento("certificado_vec"));
  e.estado.expediente.expediente_ref = "expediente:otro"; a.actualizar(); terminar(pdf); await firma;
  assert.equal(signal.aborted, true); assert.equal(e.contador.firmas, 0); assert.equal(e.contador.confirmaciones, 0);
});
test("registro externo usa PDF intacto y conserva clave al reintentar", async () => {
  const e = escenario(["portafirmas_registro_rrhh"]); const peticiones = [];
  const fichero = { size: pdf.length, arrayBuffer: async () => pdf.buffer };
  e.campos.set("[data-ct-firma-pdf]", { files: [fichero] }); e.campos.set("[data-ct-firma-referencia]", { value: "portafirmas:prueba" });
  e.campos.set("[data-ct-firma-fecha]", { value: "2026-10-03T12:00" });
  e.deps.registrarExterna = async (s) => { peticiones.push(s); if (peticiones.length === 1) throw { codigo: "servicio_no_disponible" }; return e.recibo; };
  const a = crearAccionesFirma(e.deps); await a.manejarClic(e.evento("comprobar"));
  await a.manejarClic(e.evento("portafirmas_registro_rrhh")); await a.manejarClic(e.evento("portafirmas_registro_rrhh"));
  assert.deepEqual(peticiones[0].firmado, pdf); assert.equal(peticiones[0], peticiones[1]); assert.equal(e.contador.firmas, 0);
});
test("cancelar tras una confirmación incierta conserva la clave y no afirma ausencia de efecto", async () => {
  const e = escenario(); const peticiones = [];
  e.deps.registrarVec = async (s) => { peticiones.push(s); if (peticiones.length === 1) throw { codigo: "servicio_no_disponible" }; return e.recibo; };
  const a = crearAccionesFirma(e.deps);
  await a.manejarClic(e.evento("comprobar")); await a.manejarClic(e.evento("certificado_vec"));
  await a.manejarClic(e.evento("cancelar"));
  assert.match(e.bloque.innerHTML, /No se ha podido confirmar el registro/u);
  assert.doesNotMatch(e.bloque.innerHTML, /Operación cancelada/u);
  await a.manejarClic(e.evento("comprobar")); await a.manejarClic(e.evento("certificado_vec"));
  assert.equal(peticiones[0], peticiones[1]); assert.equal(e.contador.firmas, 1);
});
test("un fallo al refrescar la vista conserva la confirmación del recibo", async () => {
  const e = escenario(); e.deps.alCambiar = async () => { throw new Error("consulta caída"); };
  const a = crearAccionesFirma(e.deps);
  await a.manejarClic(e.evento("comprobar")); await a.manejarClic(e.evento("certificado_vec"));
  assert.match(e.aviso.textContent, /Firma verificada y registrada/u);
  await a.manejarClic(e.evento("cancelar"));
  assert.match(e.aviso.textContent, /Firma verificada y registrada/u);
});
