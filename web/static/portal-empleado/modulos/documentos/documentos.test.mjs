import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { createHash } from "node:crypto";
import test from "node:test";
import { crearTraductorDocumentos, MENSAJES_DOCUMENTOS_ES } from "./i18n.js";
import { montarVistaDocumentos, validarArchivoDescarga, validarRespuestaDocumentos } from "./vista.js";
import { crearFuenteDocumentosHTTP } from "./cliente-http.js";

const huella = "a".repeat(64);
const dato = (cambios = {}) => ({ ref: "ref:1111111111111111111111111111111111111111111111111111111111111111", numero_vec: "VEC-2026-1", tipo: "dietas.comision.borrador.v1",
  version: 1, estado_firma: "pendiente_firma", huella, mime: "application/pdf", custodia: "vec", descargable: true, ...cambios });

test("la lista conserva número, firma pendiente y descarga declarada sin promover firma", () => {
  const [documento] = validarRespuestaDocumentos({ estado: "disponible", documentos: [dato()] }).documentos;
  assert.equal(documento.numero, "VEC-2026-1");
  assert.equal(documento.firma, "pendiente_firma");
  assert.equal(documento.descargable, true);
  assert.equal(validarRespuestaDocumentos({ estado: "disponible", documentos: [dato()], siguiente_cursor:"cursor:123" }).siguienteCursor,"cursor:123");
  assert.throws(() => validarRespuestaDocumentos({ estado: "disponible", documentos: [dato({ estado_firma: "firmado" })] }));
  assert.throws(() => validarRespuestaDocumentos({ estado: "disponible", documentos: [dato(), dato()] }));
  assert.throws(() => validarRespuestaDocumentos({ estado: "vacio", documentos: [dato()] }));
  assert.throws(() => validarRespuestaDocumentos({ estado: "vacio", documentos: [], siguiente_cursor:"cursor:123" }));
  assert.throws(() => validarRespuestaDocumentos({ estado: "disponible", documentos: [dato({ huella: "mal" })] }));
});

test("la custodia externa se lista sin descarga aunque el servidor la declare", () => {
  const [externo] = validarRespuestaDocumentos({ estado: "disponible", documentos: [dato({ custodia: "externa", mime: "", descargable: true })] }).documentos;
  assert.equal(externo.custodia, "externa");
  assert.equal(externo.descargable, false);
  assert.equal(externo.huella, huella);
  assert.throws(() => validarRespuestaDocumentos({ estado: "disponible", documentos: [dato({ custodia: "otra" })] }));
  assert.throws(() => validarRespuestaDocumentos({ estado: "disponible", documentos: [dato({ custodia: undefined })] }));
  assert.throws(() => validarRespuestaDocumentos({ estado: "disponible", documentos: [dato({ mime: "" })] }));
  assert.equal(crearTraductorDocumentos()("custodia_externa"), MENSAJES_DOCUMENTOS_ES.custodia_externa);
});

test("el original requiere bytes, nombre seguro y extensión coherente", () => {
  const original = { contenido: new Uint8Array([37, 80, 68, 70]), nombre: "documento-123.pdf", tipo: "application/pdf" };
  assert.equal(validarArchivoDescarga(original), original);
  assert.throws(() => validarArchivoDescarga({ ...original, nombre: "../original.pdf" }));
  assert.throws(() => validarArchivoDescarga({ ...original, contenido: new Uint8Array() }));
  assert.throws(() => validarArchivoDescarga({ ...original, tipo: "text/html" }));
});

test("el cliente POST no acepta permiso del navegador ni transmite credenciales en URL", async () => {
  const peticiones = [];
  const fetchImpl = async (ruta, opciones) => {
    peticiones.push({ ruta, opciones });
    return new Response(JSON.stringify({ data: { estado: "vacio", documentos: [] } }), { status: 200, headers: { "Content-Type": "application/json" } });
  };
  const fuente = crearFuenteDocumentosHTTP({ fetchImpl });
  fuente.seleccionarExpediente("ref:2222222222222222222222222222222222222222222222222222222222222222");
  assert.equal((await fuente.listar()).estado, "vacio");
  assert.equal(peticiones[0].ruta, "/api/vec/documentos/expedientes/consultas");
  assert.deepEqual(JSON.parse(peticiones[0].opciones.body), { expediente_ref: "ref:2222222222222222222222222222222222222222222222222222222222222222", cursor:"", limite:50 });
  assert.equal(peticiones[0].opciones.credentials, "same-origin");
  assert.equal(peticiones[0].opciones.cache, "no-store");
  assert.throws(() => fuente.seleccionarExpediente("../ajeno"));
});

test("denegación y errores no devuelven metadatos", async () => {
  const fuente = crearFuenteDocumentosHTTP({ expedienteRef: "ref:2222222222222222222222222222222222222222222222222222222222222222", fetchImpl: async () => new Response("", { status: 403 }) });
  await assert.rejects(fuente.listar(), (error) => error.codigo === "denegado");
});

test("la descarga conserva los bytes originales y rechaza una huella alterada", async () => {
  const bytes = new TextEncoder().encode("%PDF-1.7\noriginal");
  const huellaReal = createHash("sha256").update(bytes).digest("hex");
  const ref = "ref:" + "1".repeat(64);
  const cuerpos = [];
  const fetchImpl = async (_ruta, opciones) => { cuerpos.push(JSON.parse(opciones.body)); return new Response(bytes, { status:200, headers:{
    "Content-Type":"application/pdf", "Content-Disposition":`attachment; filename="documento.pdf"`,
    "X-Content-SHA256":huellaReal,
  } }); };
  const expediente = "ref:" + "2".repeat(64);
  await assert.rejects(crearFuenteDocumentosHTTP({ fetchImpl }).descargar(ref, { version:1, mime:"application/pdf", huella:huellaReal }),
    (error) => error.codigo === "referencia_invalida");
  const fuente = crearFuenteDocumentosHTTP({ expedienteRef: expediente, fetchImpl });
  const original = await fuente.descargar(ref, { version:1, mime:"application/pdf", huella:huellaReal });
  assert.deepEqual(cuerpos.at(-1), { expediente_ref: expediente, documento_ref: ref, version: 1 });
  assert.deepEqual(original.contenido,bytes);
  assert.equal(original.nombre,"documento.pdf");
  await assert.rejects(fuente.descargar(ref,{ version:1,mime:"application/pdf",huella:"a".repeat(64) }),
    (error) => error.codigo === "respuesta_invalida");
});

// DOM mínimo: basta para montar la vista y recorrer lo que pinta.
function crearDocumentoFalso() {
  const doc = { defaultView: {}, createElement(tag) {
    const nodo = { tagName: tag.toUpperCase(), ownerDocument: doc, children: [], dataset: {}, atributos: {}, oyentes: {}, textoPropio: "", padre: null,
      set textContent(v) { this.textoPropio = String(v); this.children = []; },
      get textContent() { return this.textoPropio + this.children.map((h) => h.textContent).join(""); },
      setAttribute(n, v) { this.atributos[n] = String(v); }, getAttribute(n) { return this.atributos[n] ?? null; },
      append(...hijos) { for (const h of hijos) { h.padre = this; this.children.push(h); } },
      replaceChildren(...hijos) { this.children = []; this.append(...hijos); },
      remove() { if (this.padre) this.padre.children = this.padre.children.filter((h) => h !== this); },
      addEventListener(tipo, fn) { this.oyentes[tipo] = fn; } };
    return nodo;
  } };
  doc.body = doc.createElement("body");
  return doc;
}
const todos = (nodo) => [nodo, ...nodo.children.flatMap(todos)];
const esperar = () => new Promise((resolver) => setImmediate(resolver));
const fuenteFalsa = (documentos) => ({ seleccionados: [], seleccionarExpediente(ref) { this.seleccionados.push(ref); },
  async listar() { return { estado: "disponible", documentos }; }, async descargar() { throw new Error("no usado"); } });

test("sin expediente recibido por navegación no hay campo ni consulta", async () => {
  const doc = crearDocumentoFalso();
  const fuente = fuenteFalsa([dato()]);
  montarVistaDocumentos({ raiz: doc.body, fuente });
  await esperar();
  const nodos = todos(doc.body);
  assert.equal(nodos.some((n) => ["INPUT", "FORM"].includes(n.tagName)), false);
  assert.equal(fuente.seleccionados.length, 0);
  assert.equal(nodos.find((n) => n.getAttribute("role") === "status").textContent, MENSAJES_DOCUMENTOS_ES.sin_expediente);
  const conBasura = crearDocumentoFalso();
  montarVistaDocumentos({ raiz: conBasura.body, fuente, expedienteRef: "../ajeno" });
  await esperar();
  assert.equal(fuente.seleccionados.length, 0);
});

test("solo se pinta la descarga de lo descargable y la huella externa queda visible", async () => {
  const doc = crearDocumentoFalso();
  const expediente = "ref:" + "3".repeat(64);
  const externa = "b".repeat(64);
  const fuente = fuenteFalsa([
    dato(),
    dato({ ref: "ref:" + "4".repeat(64), numero_vec: "VEC-2026-2", descargable: false }),
    dato({ ref: "ref:" + "5".repeat(64), numero_vec: "VEC-2026-3", custodia: "externa", mime: "", huella: externa }),
  ]);
  montarVistaDocumentos({ raiz: doc.body, fuente, expedienteRef: expediente });
  await esperar();
  assert.deepEqual(fuente.seleccionados, [expediente]);
  const nodos = todos(doc.body);
  assert.equal(nodos.some((n) => n.tagName === "INPUT"), false);
  const botones = nodos.filter((n) => n.tagName === "BUTTON" && n.textContent === MENSAJES_DOCUMENTOS_ES.descargar);
  assert.equal(botones.length, 1);
  assert.equal(botones[0].getAttribute("aria-label"), "Descargar original del documento VEC-2026-1");
  const filas = nodos.filter((n) => n.tagName === "TR");
  assert.equal(todos(filas[2]).some((n) => n.tagName === "BUTTON"), false, "sin botón deshabilitado perpetuo");
  const detalle = nodos.find((n) => n.tagName === "DETAILS" && n.className === "documentos-huella");
  const resumen = detalle.children.find((n) => n.tagName === "SUMMARY");
  assert.equal(resumen.textContent, `Huella ${"b".repeat(12)}…`);
  assert.equal(detalle.children.find((n) => n.tagName === "CODE").textContent, externa);
  assert.equal(nodos.some((n) => n.title), false, "la huella no depende de un title");
});

test("i18n y vista solo muestran ayuda tras el signo de interrogación", async () => {
  const vista = await readFile(new URL("./vista.js", import.meta.url), "utf8");
  const css = await readFile(new URL("./documentos.css", import.meta.url), "utf8");
  assert.equal(crearTraductorDocumentos()("tipo_comision"), "Comisión de servicio");
  assert.equal(crearTraductorDocumentos()("firma_pendiente_firma"), "Pendiente de firma");
  assert.match(vista, /"summary", "\?"/u);
  assert.doesNotMatch(vista, /localStorage|sessionStorage|document\.cookie|datos-sinteticos|datos-presentacion/iu);
  assert.doesNotMatch(css, /#[0-9a-f]{3,8}\b/iu);
  for (const clave of ["no_configurado", "cargando", "disponible", "vacio", "denegado", "error", "tipo_comision"]) assert.equal(typeof MENSAJES_DOCUMENTOS_ES[clave], "string");
});
