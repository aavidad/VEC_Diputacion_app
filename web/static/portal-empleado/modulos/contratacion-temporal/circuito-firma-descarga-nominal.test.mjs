import assert from "node:assert/strict";
import test from "node:test";
import { createHash, webcrypto } from "node:crypto";
import { cargarTextos } from "../../../comun/textos.js";
import { crearGestorCircuitoFirma, validarCircuitoFirma } from "./circuito-firma.js";
import { crearClienteDescargaFirmaV2, RUTA_DESCARGA_FIRMA_V2 } from "./cliente-http-descarga-firma-v2.js";

const esperarTurno = () => new Promise((resolver) => setTimeout(resolver, 0));
const pdf = new TextEncoder().encode("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n%%EOF\n");
const sha = createHash("sha256").update(pdf).digest("hex");
const expedienteRef = "expediente:ct:" + "a".repeat(64);
const firmaRef = "firma:nominal:r5";
const custodiaRef = "ref:" + "b".repeat(64);
const textos = await cargarTextos("contratacion-temporal-firma", { idioma: "es", porDefecto: "es" });

function circuito() {
  return validarCircuitoFirma({ esquema: "vec.contratacion_temporal.circuito_firma.v1",
    catalogo_ref: "vec.contratacion_temporal.circuito_firma:1", huella_sha256: "c".repeat(64),
    ejemplo: true, firma_eficaz: false, portafirmas: { conectado: false, motivo: "conexion_pendiente" },
    documentos: [{ documento: "resolucion", etiqueta: "Resolución de nombramiento", pasos: [{
      orden: 1, cargo: "Jefatura", perfil_ref: "perfil:ct:jefatura_servicio_rrhh", accion: "firma",
      condicion: "borrador_generado", habilita: "remision_intervencion", devolucion: "vuelve_a_redaccion",
      sustitucion: "suplente_designado", estado: "pendiente_firma",
      referencia: "vec.contratacion_temporal.circuito_firma:1:resolucion.p1",
    }] }] });
}
function lecturaR5(catalogo) {
  return { expediente_ref: expedienteRef, version_expediente: 7, documento: "resolucion",
    garantia_autenticacion: "sustancial", firmas: [{ firma_ref: firmaRef, recibo_ref: "recibo:firma:r5",
      paso_orden: 1, secuencia: 1, catalogo_ref: catalogo.catalogo_ref, catalogo_huella: catalogo.huella_sha256,
      resultado: "firmado", via: "certificado_vec", revision_pdf: { revision_sha256: sha },
      documento_custodiado: { expediente_ref: "ref:" + "d".repeat(64), documento_ref: custodiaRef,
        version: 2, huella_sha256: sha }, registrada_en: "2026-10-14T10:00:00Z" }] };
}
function datosBoton(html) {
  const boton = /<button[^>]*data-ct-descargar-firmado[^>]*>/su.exec(html)?.[0];
  assert.ok(boton, "la lectura R5 muestra el botón firmado");
  const dataset = { ctDescargarFirmado: "" };
  for (const [, clave, valor] of boton.matchAll(/data-ct-([a-z-]+)="([^"]*)"/gu)) {
    const nombre = "ct" + clave.split("-").map((pieza) => pieza[0].toUpperCase() + pieza.slice(1)).join("");
    dataset[nombre] = valor;
  }
  return dataset;
}
async function montar({ fetchDescarga, alDocumentoGeneral = () => { throw new Error("Documentos high no debe usarse"); } }) {
  let estado = { vista: "documentos", carga: "listo", expediente_ref: expedienteRef,
    expediente: { expediente_ref: expedienteRef, version: 7, demostracion: false },
    documentos: { expediente_ref: expedienteRef, version: 7, demostracion: false },
    cuadro: { demostracion: false, expedientes: [] } };
  const catalogo = circuito();
  assert.ok(catalogo);
  const aviso = { textContent: "" };
  const enlaces = [];
  let manejar;
  const zona = { innerHTML: "", querySelector: (selector) => selector === "#ct-firma-descarga-aviso" ? aviso : null };
  const raiz = { querySelector: (selector) => selector === "[data-ct-exp-firmados]" ? zona : null,
    contains: () => true,
    addEventListener: (tipo, funcion) => { if (tipo === "click") manejar = funcion; },
    removeEventListener() {} };
  const entornoDescarga = { Blob, URL: { createObjectURL: () => "blob:nominal", revokeObjectURL() {} },
    document: { body: { append(enlace) { enlaces.push(enlace); } },
      createElement: () => ({ click() { this.pulsado = true; }, remove() {} }) } };
  const gestor = crearGestorCircuitoFirma({ raiz, obtenerEstado: () => estado, cargarTextos: async () => textos,
    cliente: { obtenerCircuitoConEstado: async () => ({ estado: "disponible", circuito: catalogo }) },
    clienteFirmasV2: { consultar: async () => lecturaR5(catalogo) },
    clienteDescargaFirmaV2: crearClienteDescargaFirmaV2({ fetchImpl: fetchDescarga, cryptoImpl: webcrypto }),
    crearDocumentos: alDocumentoGeneral, entornoDescarga });
  gestor.montarSiProcede(estado);
  await esperarTurno(); await esperarTurno();
  const dataset = datosBoton(zona.innerHTML);
  assert.equal(dataset.ctFirmadoModo, "nominal");
  assert.equal(dataset.ctFirmadoTipo, "resolucion");
  assert.equal(dataset.ctFirmadoFirmaRef, firmaRef);
  assert.equal(dataset.ctFirmadoCustodiaRef, custodiaRef);
  assert.equal(dataset.ctFirmadoCustodiaVersion, "2");
  assert.equal(dataset.ctFirmadoHuella, sha);
  assert.equal(dataset.ctFirmadoDocumento, undefined, "no lleva la terna Documentos high");
  const fila = { querySelector: (selector) => selector === ".ct-fase-firma-documento strong"
    ? { textContent: "Resolución de nombramiento" } : null };
  const boton = { dataset, setAttribute() {}, removeAttribute() {},
    closest: (selector) => selector === "[data-ct-exp-firmados] [data-ct-descargar-firmado]" ? boton
      : selector === "[data-ct-exp-firmados]" ? zona : selector === ".ct-fase-firma-fila" ? fila : null };
  return { gestor, get estado() { return estado; }, cambiarEstado(nuevo) { estado = nuevo; gestor.montarSiProcede(nuevo); },
    pulsar() { assert.equal(typeof manejar, "function"); manejar({ target: boton, preventDefault() {} }); }, enlaces, aviso };
}
function respuestaValida() {
  return new Response(JSON.stringify({ data: { esquema: "vec.contratacion-temporal.descarga-firma-r5.v2",
    pdf_base64: Buffer.from(pdf).toString("base64"), expediente_ref: expedienteRef, version_observada: 7,
    documento: "resolucion", firma_ref: firmaRef, recibo_ref: "recibo:firma:r5", documento_ref: custodiaRef,
    documento_version: 2, huella_sha256: sha } }), { status: 200, headers: { "Content-Type": "application/json" } });
}

test("R5 sustancial descarga el PDF por CT con la selección visible y sin Documentos high", async () => {
  let peticion;
  const vista = await montar({ fetchDescarga: async (ruta, opciones) => {
    peticion = { ruta, opciones }; return respuestaValida();
  } });
  vista.pulsar(); await esperarTurno(); await esperarTurno();
  assert.equal(peticion.ruta, RUTA_DESCARGA_FIRMA_V2);
  assert.deepEqual(JSON.parse(peticion.opciones.body), { expediente_ref: expedienteRef,
    version_observada: 7, documento: "resolucion", firma_ref: firmaRef });
  assert.equal(vista.enlaces.length, 1);
  assert.equal(vista.enlaces[0].pulsado, true);
  assert.match(vista.enlaces[0].download, /\.pdf$/u);
  assert.match(vista.aviso.textContent, /PDF firmado descargado/u);
  vista.gestor.retirar();
});

test("403 nominal avisa sin usar la descarga general", async () => {
  let generales = 0;
  const vista = await montar({ fetchDescarga: async () => new Response(null, { status: 403 }),
    alDocumentoGeneral: () => { generales += 1; throw new Error("fallback prohibido"); } });
  vista.pulsar(); await esperarTurno(); await esperarTurno();
  assert.equal(generales, 0);
  assert.equal(vista.enlaces.length, 0);
  assert.match(vista.aviso.textContent, /No tiene permiso para descargar este PDF firmado/u);
  vista.gestor.retirar();
});

test("salir de la ficha o retirar el bloque aborta la descarga sin crear enlace", async () => {
  for (const modo of ["cambiar", "retirar"]) {
    let resolver; let signal;
    const vista = await montar({ fetchDescarga: async (_ruta, opciones) => {
      signal = opciones.signal;
      return new Promise((fin) => { resolver = fin; });
    } });
    vista.pulsar(); await esperarTurno();
    assert.ok(signal && !signal.aborted);
    const avisoAntes = vista.aviso.textContent;
    assert.match(avisoAntes, /Descargando el PDF firmado/u);
    if (modo === "cambiar") vista.cambiarEstado({ ...vista.estado, expediente_ref: "expediente:ct:" + "e".repeat(64),
      expediente: { ...vista.estado.expediente, expediente_ref: "expediente:ct:" + "e".repeat(64) } });
    else vista.gestor.retirar();
    assert.equal(signal.aborted, true);
    resolver(respuestaValida());
    await esperarTurno(); await esperarTurno();
    assert.equal(vista.enlaces.length, 0, modo);
    assert.equal(vista.aviso.textContent, avisoAntes, `${modo}: la ficha anterior no muestra un error nuevo`);
    assert.doesNotMatch(vista.aviso.textContent, /No se ha podido descargar el PDF firmado/u);
    if (modo === "cambiar") vista.gestor.retirar();
  }
});
