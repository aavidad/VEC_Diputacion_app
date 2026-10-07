import assert from "node:assert/strict";
import test from "node:test";
import { createHash } from "node:crypto";
import { crearAccionesFirma, renderizarAccionesPaso } from "./circuito-firma-acciones.js";
import { crearTraductorCircuitoFirma } from "./i18n-circuito-firma.js?v=20261001-ct-a-i18n-v1";

const pdf = new TextEncoder().encode("%PDF-1.7\noriginal con primera firma\n%%EOF");
const firmadoVec = new TextEncoder().encode("%PDF-1.7\noriginal con primera firma\nsegunda firma\n%%EOF");
const sha = bytes => createHash("sha256").update(bytes).digest("hex");
// DTO V2 completo: prueba el consumidor, sin acreditar criptografía ni competencia real.
function reciboPrueba(s, externo = false) {
  const h = sha(s.firmado);
  return { esquema: externo ? "vec.contratacion-temporal.registro-firma-externa.v2" : "vec.contratacion-temporal.registro-firma-vec.v2",
    recibo_ref: "recibo:prueba", firma_ref: "firma:prueba", ya_registrada: false,
    expediente_ref: s.expedienteRef, version_expediente: s.version, documento: s.documento,
    paso_orden: s.pasoOrden, paso_ref: "paso:prueba", secuencia: 2, registrada_en: "2026-10-03T10:00:00Z",
    firma_eficaz: false, material_root_sha256: "c".repeat(64),
    documento_custodiado: { expediente_ref: `ref:${"e".repeat(64)}`, documento_ref: `ref:${"d".repeat(64)}`, version: 1, huella_sha256: h },
    verificacion_tecnica: { estado: "valida", motivo: "verificada", politica: "politica:vec:firma:verificacion-autonoma:v2",
      revocacion: "vigente", sello_tiempo: "no_presente", original_sha256: s.originalHuella, firmado_sha256: h },
    revision_pdf: { orden_firma: s.pasoOrden, entrada_sha256: s.revisionEntradaHuella, revision_sha256: h, evidencia_sha256: "f".repeat(64) },
    ...(externo ? { procedencia_portafirmas: { estado: "declarada_por_rrhh", referencia_declarada: s.referenciaPortafirmas, fecha_declarada: s.fechaPortafirmas } } : {}) };
}

function escenario(vias = ["certificado_vec", "portafirmas_registro_rrhh"], pasoOrden = 2) {
  const estado = { vista: "expediente", carga: "listo", expediente: { expediente_ref: "expediente:prueba", version: 7, demostracion: false } };
  const circuito = { preflight_compuesto: true, catalogo_ref: "catalogo:prueba", huella_sha256: "a".repeat(64),
    registro: { verificacion: true }, documentos: [{ documento: "resolucion", paso_pendiente: pasoOrden }] };
  const aviso = { textContent: "", focus() {} };
  const campos = new Map();
  const bloque = { innerHTML: "", dataset: { ctFirmaDocumento: "resolucion", ctFirmaPaso: String(pasoOrden) },
    setAttribute() {}, querySelectorAll: () => [],
    querySelector: (s) => s === "[data-ct-firma-resultado]" ? aviso : campos.get(s) ?? null };
  const preflight = { esquema: "vec.contratacion-temporal.preflight-firma.v2",
    entrada_documento_ref: pasoOrden > 1 ? "revision:firmada-p1" : "original:raiz",
    entrada_documento_version: pasoOrden > 1 ? 1 : 2, entrada_documento_sha256: createHash("sha256").update(pdf).digest("hex"), version_expediente: 7, documento: "resolucion", catalogo_ref: "catalogo:prueba", catalogo_huella: "a".repeat(64),
    paso_pendiente: pasoOrden, original_ref: "original:raiz", original_version: 2, vias_disponibles: vias };
  const contador = { originales: 0, firmas: 0, registros: 0, confirmaciones: 0 };
  const recibo = reciboPrueba({ expedienteRef: "expediente:prueba", version: 7, documento: "resolucion",
    pasoOrden, originalHuella: sha(pdf), revisionEntradaHuella: sha(pdf), firmado: firmadoVec });
  const deps = { obtenerEstado: () => estado, obtenerCircuito: () => circuito, t: crearTraductorCircuitoFirma(),
    clientePreflight: { consultar: async () => preflight },
    obtenerVinculoOriginal: async () => ({ originalRef: "original:raiz", originalVersion: 2, originalHuella: createHash("sha256").update(pdf).digest("hex"), revisionEntradaRef: "revision:firmada-p1", revisionEntradaVersion: 1, revisionEntradaHuella: "d".repeat(64) }),
    obtenerOriginal: async () => { contador.originales++; return pdf; },
    autofirma: { firmarPDF: async (original) => { assert.equal(original, pdf); contador.firmas++; return new TextEncoder().encode("%PDF-1.7\noriginal con primera firma\nsegunda firma\n%%EOF"); } },
    registrarVec: async () => { contador.registros++; return recibo; },
    registrarExterna: async (s) => { contador.registros++; return reciboPrueba(s, true); },
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
  assert.deepEqual(enviados.original, pdf); assert.equal(e.contador.confirmaciones, 1);
  assert.match(e.aviso.textContent, /Firma verificada y registrada/u);
});
test("fallo de confirmación reintenta los mismos bytes y clave sin otra firma", async () => {
  const e = escenario(); const peticiones = [];
  e.deps.registrarVec = async (s) => { peticiones.push(s); if (peticiones.length === 1) throw { codigo: "servicio_no_disponible" }; return e.recibo; };
  const acciones = crearAccionesFirma(e.deps);
  await acciones.manejarClic(e.evento("comprobar")); await acciones.manejarClic(e.evento("certificado_vec"));
  assert.equal(e.contador.confirmaciones, 0); assert.match(e.aviso.textContent, /Reintente con el mismo PDF/u);
  await acciones.manejarClic(e.evento("certificado_vec"));
  assert.equal(e.contador.firmas, 1); assert.deepEqual(peticiones[0], peticiones[1]); assert.equal(e.contador.confirmaciones, 1);
});
test("consulta rota, ejemplo, huella ajena o recibo ajeno cierran efectos y confirmación", async () => {
  for (const alterar of [(e) => { e.estado.expediente.demostracion = true; },
    (e) => { e.bloque.dataset.ctFirmaPaso = "NaN"; },
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
  e.deps.registrarExterna = async (s) => { peticiones.push(s); if (peticiones.length === 1) throw { codigo: "servicio_no_disponible" }; return reciboPrueba(s, true); };
  const a = crearAccionesFirma(e.deps); await a.manejarClic(e.evento("comprobar"));
  await a.manejarClic(e.evento("portafirmas_registro_rrhh")); await a.manejarClic(e.evento("portafirmas_registro_rrhh"));
  assert.deepEqual(peticiones[0].firmado, pdf); assert.deepEqual(peticiones[0], peticiones[1]); assert.equal(e.contador.firmas, 0);
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
  assert.deepEqual(peticiones[0], peticiones[1]); assert.equal(e.contador.firmas, 1);
});
test("un fallo al refrescar la vista conserva la confirmación del recibo", async () => {
  const e = escenario(); e.deps.alCambiar = async () => { throw new Error("consulta caída"); };
  const a = crearAccionesFirma(e.deps);
  await a.manejarClic(e.evento("comprobar")); await a.manejarClic(e.evento("certificado_vec"));
  assert.match(e.aviso.textContent, /Firma verificada y registrada/u);
  await a.manejarClic(e.evento("cancelar"));
  assert.match(e.aviso.textContent, /Firma verificada y registrada/u);
});

test("el paso nominal V2 prevalece sobre un panel CT118 antiguo sin inferir otro estado", async () => {
  const e = escenario(["certificado_vec"], 1);
  e.circuito.registro = null;
  e.preflight.paso_pendiente = 2;
  e.preflight.entrada_documento_ref = "revision:firmada-p1";
  e.preflight.entrada_documento_version = 1;
  e.recibo.paso_orden = 2; e.recibo.revision_pdf.orden_firma = 2;
  let enviada;
  e.deps.registrarVec = async (sol) => { enviada = sol; return e.recibo; };
  const a = crearAccionesFirma(e.deps);
  await a.manejarClic(e.evento("comprobar")); await a.manejarClic(e.evento("certificado_vec"));
  assert.equal(enviada.pasoOrden, 2);
  assert.equal(enviada.originalRef, "original:raiz");
  assert.equal(enviada.revisionEntradaRef, "revision:firmada-p1");
  assert.equal(e.circuito.documentos[0].paso_pendiente, 1);
  assert.equal(e.contador.confirmaciones, 1);
});

test("una operación externa incierta avisa y conserva su vía antes de otra firma", async () => {
  const e = escenario(); const fichero = { size: pdf.length, arrayBuffer: async () => pdf.buffer };
  e.campos.set("[data-ct-firma-pdf]", { files: [fichero] });
  e.campos.set("[data-ct-firma-referencia]", { value: "portafirmas:prueba" });
  e.campos.set("[data-ct-firma-fecha]", { value: "2026-10-03T12:00" });
  const recibidas = [];
  e.deps.registrarExterna = async (s) => { recibidas.push(s); if (recibidas.length === 1) throw { codigo: "servicio_no_disponible" }; return reciboPrueba(s, true); };
  const acciones = crearAccionesFirma(e.deps);
  await acciones.manejarClic(e.evento("comprobar")); await acciones.manejarClic(e.evento("portafirmas_registro_rrhh"));
  e.aviso.textContent = "";
  await acciones.manejarClic(e.evento("certificado_vec"));
  assert.match(e.aviso.textContent, /Reintente con el mismo PDF/u); assert.equal(e.contador.firmas, 0);
  await acciones.manejarClic(e.evento("portafirmas_registro_rrhh"));
  assert.deepEqual(recibidas[0], recibidas[1]); assert.equal(e.contador.confirmaciones, 1);
});

test("el formulario externo identifica y enfoca el primer campo obligatorio ausente", async () => {
  const e = escenario(["portafirmas_registro_rrhh"]);
  const campos = ["[data-ct-firma-pdf]", "[data-ct-firma-referencia]", "[data-ct-firma-fecha]"];
  for (const selector of campos) {
    const atributos = new Map();
    e.campos.set(selector, { files: [], value: "", atributos, setAttribute(k,v) { atributos.set(k,v); },
      removeAttribute(k) { atributos.delete(k); }, focus() { this.enfocado = true; } });
  }
  const acciones = crearAccionesFirma(e.deps); await acciones.manejarClic(e.evento("comprobar"));
  for (const selector of campos) {
    await acciones.manejarClic(e.evento("portafirmas_registro_rrhh"));
    const campo = e.campos.get(selector);
    assert.equal(campo.atributos.get("aria-invalid"), "true"); assert.equal(campo.enfocado, true);
    if (selector.includes("-pdf")) campo.files = [{ size: pdf.length, arrayBuffer: async () => pdf.buffer }];
    else if (selector.includes("referencia")) campo.value = "portafirmas:prueba";
    assert.equal(e.contador.registros, 0);
  }
});

test("tras 503 externo conserva material, metadatos y clave pese a cambios por JS", async () => {
  for (const cambio of ["pdf", "referencia", "fecha"]) {
    const e = escenario(["portafirmas_registro_rrhh"]); const peticiones = [];
    let bytes = pdf.slice(); let claves = 0;
    const fichero = { size: pdf.length, arrayBuffer: async () => bytes.buffer };
    const campoPDF = { tagName: "INPUT", files: [fichero] };
    const referencia = { tagName: "INPUT", value: "portafirmas:prueba" };
    const fecha = { tagName: "INPUT", value: "2026-10-03T12:00" };
    e.campos.set("[data-ct-firma-pdf]", campoPDF); e.campos.set("[data-ct-firma-referencia]", referencia); e.campos.set("[data-ct-firma-fecha]", fecha);
    e.bloque.querySelectorAll = (selector) => selector === "button,input" ? [campoPDF, referencia, fecha] : [];
    e.deps.crearClave = () => { claves++; return "clave-prueba-123456789"; };
    e.deps.registrarExterna = async (s) => { peticiones.push(s); if (peticiones.length === 1) throw { codigo: "servicio_no_disponible" }; return reciboPrueba(s, true); };
    const a = crearAccionesFirma(e.deps);
    await a.manejarClic(e.evento("comprobar")); await a.manejarClic(e.evento("portafirmas_registro_rrhh"));
    assert.ok([campoPDF, referencia, fecha].every(c => c.disabled));
    if (cambio === "pdf") bytes[10] ^= 1;
    if (cambio === "referencia") referencia.value = "portafirmas:otra";
    if (cambio === "fecha") fecha.value = "2026-10-04T12:00";
    // Un caller puede saltarse disabled; el controlador sigue rechazando el material distinto.
    for (const c of [campoPDF, referencia, fecha]) c.disabled = false;
    await a.manejarClic(e.evento("portafirmas_registro_rrhh"));
    assert.equal(peticiones.length, 1, cambio); assert.equal(claves, 1, cambio); assert.equal(e.contador.confirmaciones, 0);
    assert.match(e.aviso.textContent, /Reintente con el mismo PDF/u);
    bytes = pdf.slice(); referencia.value = "portafirmas:prueba"; fecha.value = "2026-10-03T12:00";
    await a.manejarClic(e.evento("portafirmas_registro_rrhh"));
    assert.deepEqual(peticiones[0], peticiones[1]); assert.equal(claves, 1); assert.equal(e.contador.confirmaciones, 1);
  }
});

test("el último puerto no confirma recibos legacy ni bytes distintos aunque el DTO V2 sea coherente", async () => {
  for (const alterar of [s => ({ firma_eficaz: false, recibo_ref: "recibo:legacy", expediente_ref: s.expedienteRef,
      version_expediente: s.version, documento: s.documento, paso_orden: s.pasoOrden, firma_verificada: true }),
    s => { const otro = s.firmado.slice(); otro[10] ^= 1; return reciboPrueba({ ...s, firmado: otro }); }]) {
    const e = escenario(["certificado_vec"]);
    e.deps.registrarVec = async s => alterar(s);
    const a = crearAccionesFirma(e.deps); await a.manejarClic(e.evento("comprobar")); await a.manejarClic(e.evento("certificado_vec"));
    assert.equal(e.contador.confirmaciones, 0); assert.doesNotMatch(e.aviso.textContent, /Firma verificada y registrada/u);
  }
});

test("un puerto que modifica su copia no cambia el material privado del reintento", async () => {
  const e = escenario(["certificado_vec"]); const enviadas = [];
  e.deps.registrarVec = async s => {
    enviadas.push(s.firmado.slice());
    if (enviadas.length === 1) { s.firmado[10] ^= 1; throw { codigo: "servicio_no_disponible" }; }
    return reciboPrueba(s);
  };
  const a = crearAccionesFirma(e.deps); await a.manejarClic(e.evento("comprobar")); await a.manejarClic(e.evento("certificado_vec"));
  await a.manejarClic(e.evento("certificado_vec"));
  assert.deepEqual(enviadas[0], enviadas[1]); assert.equal(e.contador.firmas, 1); assert.equal(e.contador.confirmaciones, 1);
});
