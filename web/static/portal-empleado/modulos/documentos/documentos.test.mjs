import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { createHash } from "node:crypto";
import test from "node:test";
import { crearTraductorDocumentos, MENSAJES_DOCUMENTOS_ES } from "./i18n.js";
import { validarArchivoDescarga, validarRespuestaDocumentos } from "./vista.js";
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
  const fetchImpl = async () => new Response(bytes, { status:200, headers:{
    "Content-Type":"application/pdf", "Content-Disposition":`attachment; filename="documento.pdf"`,
    "X-Content-SHA256":huellaReal,
  } });
  const fuente = crearFuenteDocumentosHTTP({ fetchImpl });
  const original = await fuente.descargar(ref, { version:1, mime:"application/pdf", huella:huellaReal });
  assert.deepEqual(original.contenido,bytes);
  assert.equal(original.nombre,"documento.pdf");
  await assert.rejects(fuente.descargar(ref,{ version:1,mime:"application/pdf",huella:"a".repeat(64) }),
    (error) => error.codigo === "respuesta_invalida");
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
