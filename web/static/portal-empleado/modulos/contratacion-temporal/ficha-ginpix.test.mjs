import assert from "node:assert/strict";
import test from "node:test";
import { validarFichaGINPIX } from "./contrato-ficha-ginpix.js";
import { crearFichaGINPIXClienteHTTP, RUTA_FICHA_GINPIX } from "./cliente-http-ficha-ginpix.js";
import { montarFichaGINPIX } from "./ficha-ginpix.js";

const recibo = Object.freeze({ esquema: "vec.contratacion-temporal.incorporacion-ejercicio.recibo.v2", expediente_ref: "expediente:ct:001", solicitud_personal_ref: "solicitud:001", relacion_ref: "relacion:001", recibo_ref: "recibo:001", actuacion_ref: "actuacion:001", registrada_en: "2026-09-10T10:00:00Z", periodo_incorporacion: { desde: "2026-09-11T00:00:00Z", hasta: "2026-09-12T00:00:00Z" }, version_solicitud_personal: 1, version_actual_expediente: 2, seguimiento_ref: "seguimiento:001", version_seguimiento_anterior: 0, version_seguimiento_resultante: 1, auditoria_ref: "auditoria:001", outbox_ref: "outbox:001", ejercicio_sintetico: true, firma_oficial: false, eficacia_administrativa: false });
const ficha = () => ({ esquema: "vec.dipgra.contratacion-temporal.ginpix.fichero.v1", version: 1, metadatos: { esquema_modelo: "vec.dipgra.contratacion-temporal.ginpix.modelo.v1", esquema_mapeo: "vec.dipgra.contratacion-temporal.ginpix.mapeo.v1", esquema_carga: "vec.dipgra.contratacion-temporal.ginpix.carga.v1", version_expediente: 2, expediente_ref: recibo.expediente_ref, incorporacion_ref: recibo.actuacion_ref, procedencia_modelo_ref: recibo.recibo_ref, correlacion_ref: recibo.auditoria_ref, idempotencia_ref: recibo.outbox_ref, huella_modelo_sha256: "a".repeat(64), mapeo_ref: "mapeo:001", mapeo_version: 1, procedencia_mapeo_ref: "procedencia:001", huella_mapeo_sha256: "b".repeat(64), huella_carga_sha256: "c".repeat(64) }, campos: [{ clave: "actuacion_ref", estado: "valor", valor: recibo.actuacion_ref }] });

test("contrato: acepta solo el fichero V1 vinculado al recibo V2", () => {
  assert.equal(validarFichaGINPIX(ficha(), recibo).metadatos.incorporacion_ref, recibo.actuacion_ref);
  for (const cambio of [{ expediente_ref: "expediente:ajeno" }, { incorporacion_ref: "actuacion:ajena" }, { procedencia_modelo_ref: "recibo:ajeno" }, { version_expediente: 3 }]) assert.throws(() => validarFichaGINPIX({ ...ficha(), metadatos: { ...ficha().metadatos, ...cambio } }, recibo));
});
test("cliente: delega GET, límite y cabeceras nominales al transporte inyectado", async () => {
  let orden; const contenido = new Uint8Array([123, 125]); const cliente = crearFichaGINPIXClienteHTTP({ validarOpciones: (x) => x ?? {}, descargar: async (x) => { orden = x; return { tipo: "application/json", nombre: "ficha-ginpix-ejercicio.json", json: ficha(), contenido }; } });
  assert.equal((await cliente.descargarFichaGINPIX(recibo)).contenido, contenido); assert.equal(orden.ruta, `${RUTA_FICHA_GINPIX}?expediente_ref=${encodeURIComponent(recibo.expediente_ref)}`); assert.equal(orden.metodo, "GET"); assert.equal(orden.maximoRespuesta, 512 * 1024); assert.deepEqual(orden.errores, ["acceso_denegado", "recibo_no_confirmado", "servicio_no_disponible"]);
});
test("controlador: identifica la ficha con su recibo y nombre antes de descargar los bytes", async () => {
  const eventos = new Map(); const raiz = { innerHTML: "", addEventListener: (k, v) => eventos.set(k, v), removeEventListener() {}, replaceChildren() {} }; let descargado;
  montarFichaGINPIX({ raiz, recibo, mensajes: { ficha_ginpix_aviso: "<aviso>" }, cliente: { descargarFichaGINPIX: async () => ({ json: ficha(), contenido: new Uint8Array([123, 125]) }) }, descargarArchivo: async (archivo, nombre) => { descargado = [archivo, nombre]; } });
  assert.match(raiz.innerHTML, /<h3>Ficha GINPIX<\/h3>/u); assert.match(raiz.innerHTML, /<dl class="ct-resumen">/u); assert.match(raiz.innerHTML, /<dt>Recibo de incorporación confirmado<\/dt><dd><code>recibo:001<\/code><\/dd>/u); assert.match(raiz.innerHTML, /<dt>Nombre de archivo<\/dt><dd><code>ficha-ginpix-ejercicio\.json<\/code><\/dd>/u); assert.match(raiz.innerHTML, /&lt;aviso&gt;/u); await eventos.get("click")({ target: { matches: () => true } }); assert.equal(descargado[1], "ficha-ginpix-ejercicio.json"); assert.ok(descargado[0].contenido instanceof Uint8Array);
});
test("controlador: el resumen final agrupa sólo centro y categoría ya validados con el recibo", () => {
  const eventos = new Map(); const raiz = { innerHTML: "", addEventListener: (k, v) => eventos.set(k, v), removeEventListener() {}, replaceChildren() {} };
  montarFichaGINPIX({ raiz, recibo, resumen: { centro: "Centro sintético", categoria: "Categoría sintética" },
    cliente: { descargarFichaGINPIX: async () => ({ json: ficha(), contenido: new Uint8Array([123, 125]) }) }, descargarArchivo: async () => {}, locale: "es-ES", zonaHoraria: "UTC" });
  assert.match(raiz.innerHTML, /Resumen final para GINPIX/u);
  assert.match(raiz.innerHTML, /<dt>Destino<\/dt><dd>Centro sintético<\/dd>/u);
  assert.match(raiz.innerHTML, /<dt>Categoría<\/dt><dd>Categoría sintética<\/dd>/u);
  assert.match(raiz.innerHTML, /Inicio de incorporación/u);
  assert.match(raiz.innerHTML, /recibo:001/u);
  assert.match(raiz.innerHTML, /Envío externo no conectado; la ficha de carga manual está disponible\./u);
  assert.doesNotMatch(raiz.innerHTML, /enviado a GINPIX/u);
});
test("controlador: rechaza un resumen ajeno o incompleto antes de montar", () => {
  const raiz = { addEventListener() {}, removeEventListener() {}, replaceChildren() {} };
  for (const resumen of [null, { centro: "Centro" }, { centro: "Centro", categoria: "Categoría", extra: true }]) {
    assert.throws(() => montarFichaGINPIX({ raiz, recibo, resumen, cliente: { descargarFichaGINPIX() {} }, descargarArchivo() {} }), /contexto de resumen GINPIX/u);
  }
});
test("controlador: el desmontaje conserva la cancelación aunque falle la descarga", async () => {
  const eventos = new Map(); const raiz = { innerHTML: "", addEventListener: (k, v) => eventos.set(k, v), removeEventListener() {}, replaceChildren() { this.innerHTML = ""; } }; let rechazar, marcarInicio; const iniciado = new Promise((resolve) => { marcarInicio = resolve; });
  const desmontar = montarFichaGINPIX({ raiz, recibo, cliente: { descargarFichaGINPIX: async () => ({ json: ficha(), contenido: new Uint8Array([123, 125]) }) }, descargarArchivo: () => new Promise((_, reject) => { rechazar = reject; marcarInicio(); }) });
  const pulsacion = eventos.get("click")({ target: { matches: () => true } }); await iniciado; desmontar(); rechazar(new Error("fallo tardío")); await pulsacion;
  assert.equal(raiz.innerHTML, "");
});
