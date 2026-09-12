import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import { crearAdaptadorHTTPExpedientesContratacionTemporal } from "./adaptador-http-expedientes.js";
import { crearPresentadorExpedientesContratacionTemporal } from "./presentador-expedientes.js";
import { montarModuloContratacionTemporal } from "./vista-expedientes.js";

const expediente_ref = "expediente:ct:montaje";
const recibo = Object.freeze({
  esquema: "vec.contratacion-temporal.incorporacion-ejercicio.recibo.v2", expediente_ref,
  solicitud_personal_ref: "solicitud:personal:montaje", relacion_ref: "relacion:personal:montaje",
  recibo_ref: "recibo:incorporacion:1", actuacion_ref: "actuacion:ct:1", registrada_en: "2026-09-12T10:00:00Z",
  periodo_incorporacion: { desde: "2027-01-01T00:00:00Z", hasta: "2027-03-31T00:00:00Z" },
  version_solicitud_personal: 7, version_actual_expediente: 8, seguimiento_ref: "seguimiento:ct:1",
  version_seguimiento_anterior: 0, version_seguimiento_resultante: 1, auditoria_ref: "auditoria:ct:1",
  outbox_ref: "outbox:ct:1", ejercicio_sintetico: true, firma_oficial: false, eficacia_administrativa: false,
});
const resumen = () => ({ expediente_ref, numero_visible: "2026/CT-0001", version: 8, flujo_ref: "flujo:ct:general", flujo_version: 1, flujo_huella_sha256: "a".repeat(64), fase_clave: "nombramiento", estado_clave: "en_curso", centro_ref: "centro:001", categoria_ref: "categoria:auxiliar", creado_en: "2026-09-03T08:00:00Z", actualizado_en: "2026-09-03T09:00:00Z" });
const detalle = () => ({ esquema: "vec.contratacion-temporal.detalle-rrhh.v1", resumen: resumen(), solicitud: { grupo_subgrupo: "A2", motivo_clave: "sustitucion", periodo_inicio: "2026-09-04T00:00:00Z", periodo_fin: "2026-12-31T00:00:00Z" }, hitos: Array.from({ length: 8 }, (_, indice) => ({ secuencia: indice + 1, version_expediente: indice + 1, accion_clave: "registrar_solicitud", realizada_en: "2026-09-03T09:00:00Z", fase_destino: "solicitud", estado_origen: "en_curso", estado_destino: "en_curso" })) });

function domMontaje() {
  const eventos = new Map(), hijos = [], vigentes = new Set(); let contenedor = null;
  const crearNodo = () => { const eventosNodo = new Map(), eventosBoton = new Map(); const nodo = { innerHTML: "", eventos: eventosNodo, eventosBoton, addEventListener: (tipo, fn) => eventosNodo.set(tipo, fn), removeEventListener: (tipo) => eventosNodo.delete(tipo), replaceChildren() { this.innerHTML = ""; }, append(nodo) { hijos.push(nodo); }, querySelector(selector) { return selector === "[data-ct-exp-reintentar-cierre]" && this.innerHTML.includes("data-ct-exp-reintentar-cierre") ? { addEventListener: (tipo, fn) => eventosBoton.set(tipo, fn) } : null; }, ownerDocument: { createElement: crearNodo } }; return nodo; };
  const raiz = { set innerHTML(html) { vigentes.clear(); contenedor = html.includes("data-ct-exp-incorporacion-ejercicio") ? crearNodo() : null; if (contenedor) vigentes.add(contenedor); }, get innerHTML() { return ""; }, contains: (nodo) => vigentes.has(nodo), querySelector: (selector) => selector === "[data-ct-exp-incorporacion-ejercicio]" ? contenedor : null, querySelectorAll: () => [], addEventListener: (tipo, fn) => eventos.set(tipo, fn), removeEventListener: (tipo) => eventos.delete(tipo) };
  return { raiz, hijos, abrir: async () => { const control = { dataset: { ctExpAbrir: expediente_ref }, closest: (selector) => selector === "[data-ct-exp-abrir]" ? control : null }; vigentes.add(control); await eventos.get("click")({ target: control, preventDefault() {} }); } };
}
const esperar = () => new Promise((resolver) => setImmediate(resolver));

async function montarPanel({ cierre, registrar, versionCuadro = 8, mensajes = {} } = {}) {
  const dom = domMontaje();
  const clienteLectura = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
    const entrada = JSON.parse(opciones.body);
    const data = ruta.endsWith("/cuadro/consultas") ? { esquema: "vec.contratacion-temporal.cuadro-rrhh.v1", generada_en: "2026-09-03T09:05:00Z", expedientes: [{ ...resumen(), version: versionCuadro }], hay_mas: false } : detalle(entrada.expediente_ref);
    return new Response(JSON.stringify({ data }), { headers: { "content-type": "application/json; charset=utf-8" } });
  } });
  const fuente = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente: clienteLectura }); await fuente.listar();
  const presentador = crearPresentadorExpedientesContratacionTemporal({ fuente, capacidades: fuente.capacidades });
  const cliente = { async prepararIncorporacionEjercicio() { return { esquema: "vec.contratacion-temporal.incorporacion-ejercicio.preparacion.v2", expediente_ref, version_actual_expediente: 8, preparacion: null, recibo }; }, confirmarIncorporacionEjercicio() { assert.fail("no debe enviar incorporación"); }, anotacionAdministrativa: { registrar, recuperar() { assert.fail("no debe recuperar"); } }, consultarPreparacionCierreSinCese: cierre, cerrar() { assert.fail("no debe cerrar"); } };
  const modulo = await montarModuloContratacionTemporal({ raiz: dom.raiz, presentador, llamamiento: { cliente }, confirmarOperacion: () => true, mensajes });
  await dom.abrir(); await esperar(); await esperar();
  return { dom, modulo, presentador };
}

test("montaje: los seis assets nuevos están declarados e importados sin perder resolución", async () => {
  const modulo = new URL("./", import.meta.url);
  const manifiesto = await readFile(new URL("../../../../interno.manifest", modulo), "utf8");
  const esperados = [
    "contrato-anotacion-administrativa.js", "contrato-cierre-administrativo.js",
    "cliente-http-anotacion-administrativa.js", "cliente-http-cierre-administrativo.js",
    "formulario-anotacion-administrativa.js", "formulario-cierre-administrativo.js",
  ];
  for (const asset of esperados) {
    assert.match(manifiesto, new RegExp(`^static/portal-empleado/modulos/contratacion-temporal/${asset}$`, "mu"));
  }
  for (const asset of ["cliente-http-resolucion-formalizacion.js", "formulario-resolucion-formalizacion.js"]) {
    assert.match(manifiesto, new RegExp(`^static/portal-empleado/modulos/contratacion-temporal/${asset}$`, "mu"));
  }
  const [cliente, vista] = await Promise.all(["cliente-http.js", "vista-expedientes.js"].map((asset) => readFile(new URL(asset, modulo), "utf8")));
  for (const asset of esperados.slice(2, 4)) assert.match(cliente, new RegExp(`from "\\./${asset}"`, "u"));
  for (const asset of esperados.slice(4)) assert.match(vista, new RegExp(`from "\\./${asset}"`, "u"));
});

test("montaje: un GET de cierre tardío no recrea el panel tras desmontar", async () => {
  const dom = domMontaje(); let resolverCierre; let lecturasCierre = 0;
  const clienteLectura = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
    const entrada = JSON.parse(opciones.body);
    const data = ruta.endsWith("/cuadro/consultas") ? { esquema: "vec.contratacion-temporal.cuadro-rrhh.v1", generada_en: "2026-09-03T09:05:00Z", expedientes: [resumen()], hay_mas: false } : detalle(entrada.expediente_ref);
    return new Response(JSON.stringify({ data }), { headers: { "content-type": "application/json; charset=utf-8" } });
  } });
  const fuente = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente: clienteLectura }); await fuente.listar();
  const presentador = crearPresentadorExpedientesContratacionTemporal({ fuente, capacidades: fuente.capacidades });
  const cliente = { async prepararIncorporacionEjercicio() { return { esquema: "vec.contratacion-temporal.incorporacion-ejercicio.preparacion.v2", expediente_ref, version_actual_expediente: 8, preparacion: null, recibo }; }, confirmarIncorporacionEjercicio() { assert.fail("no debe enviar incorporación"); }, anotacionAdministrativa: { registrar() { assert.fail("no debe anotar"); }, recuperar() { assert.fail("no debe recuperar"); } }, async consultarPreparacionCierreSinCese() { lecturasCierre++; return new Promise((resolver) => { resolverCierre = resolver; }); }, cerrar() { assert.fail("no debe cerrar"); } };
  const modulo = await montarModuloContratacionTemporal({ raiz: dom.raiz, presentador, llamamiento: { cliente }, confirmarOperacion: () => true });
  await dom.abrir(); await esperar(); await esperar();
  assert.equal(lecturasCierre, 1);
  const hijosAntes = dom.hijos.length;
  modulo.desmontar(); resolverCierre({ preparacion: null }); await esperar(); await esperar();
  assert.equal(dom.hijos.length, hijosAntes);
  assert.ok(dom.hijos.every((hijo) => hijo.innerHTML === ""));
});

test("montaje: anotación v8→v9 conserva su recibo y deshabilita cierre si el detalle sigue obsoleto", async () => {
  const reciboAnotacion = { operacion: "registrar_anotacion_administrativa", organizacion_ref: "organizacion:ct:1", expediente_ref, version_anterior: 8, version_resultante: 9, fase_resultante: "nombramiento", estado_resultante: "en_curso", seguimiento_original: { seguimiento_ref: recibo.seguimiento_ref, version_seguimiento: 1, huella_raiz_seguimiento_sha256: "a".repeat(64) }, recibo_ref: "recibo:anotacion:9", auditoria_ref: "auditoria:ct:9", evento_ref: "evento:ct:9", actor_ref: "actor:ct:1", registrada_en: "2026-09-12T10:00:00Z" };
  const panel = await montarPanel({ cierre: async () => ({ preparacion: null }), registrar: async () => reciboAnotacion, versionCuadro: 8 });
  await panel.dom.hijos[0].eventos.get("submit")({ preventDefault() {}, target: { elements: { observaciones: { value: "Anotación sintética v9" } } } });
  await esperar(); await esperar();
  assert.match(panel.dom.hijos[0].innerHTML, /recibo:anotacion:9/u);
  assert.match(panel.dom.hijos[1].innerHTML, /detalle mostrado \(v8\) está obsoleto/u);
  assert.doesNotMatch(panel.dom.hijos[1].innerHTML, /data-ct-cierre-administrativo-form/u);
  panel.modulo.desmontar();
});

test("montaje: un GET inicial tardío no vence el bloqueo monotónico tras anotación v9", async () => {
  let resolverCierre;
  const reciboAnotacion = { operacion: "registrar_anotacion_administrativa", organizacion_ref: "organizacion:ct:1", expediente_ref, version_anterior: 8, version_resultante: 9, fase_resultante: "nombramiento", estado_resultante: "en_curso", seguimiento_original: { seguimiento_ref: recibo.seguimiento_ref, version_seguimiento: 1, huella_raiz_seguimiento_sha256: "a".repeat(64) }, recibo_ref: "recibo:anotacion:tardia", auditoria_ref: "auditoria:ct:9", evento_ref: "evento:ct:9", actor_ref: "actor:ct:1", registrada_en: "2026-09-12T10:00:00Z" };
  const panel = await montarPanel({ cierre: async () => new Promise((resolver) => { resolverCierre = resolver; }), registrar: async () => reciboAnotacion, versionCuadro: 8 });
  await panel.dom.hijos[0].eventos.get("submit")({ preventDefault() {}, target: { elements: { observaciones: { value: "Anotación sintética v9 con GET pendiente" } } } });
  await esperar(); await esperar();
  resolverCierre({ preparacion: { expediente_ref, seguimiento_ref: recibo.seguimiento_ref, version_esperada: 1, motivos: ["sin_cese"] } });
  await esperar(); await esperar();
  assert.match(panel.dom.hijos[0].innerHTML, /recibo:anotacion:tardia/u);
  assert.match(panel.dom.hijos[1].innerHTML, /detalle mostrado \(v8\) está obsoleto/u);
  assert.doesNotMatch(panel.dom.hijos[1].innerHTML, /data-ct-cierre-administrativo-form/u);
  panel.modulo.desmontar();
});

test("montaje: error de lectura de cierre es visible y el reintento conserva la recuperación", async () => {
  let lecturas = 0;
  const panel = await montarPanel({ cierre: async () => { lecturas++; throw new Error("GET sintético no disponible"); }, registrar: async () => { assert.fail("no debe anotar"); } });
  assert.equal(lecturas, 1);
  assert.match(panel.dom.hijos[1].innerHTML, /No se pudo recuperar la preparación del cierre/u);
  assert.match(panel.dom.hijos[1].innerHTML, /data-ct-exp-reintentar-cierre/u);
  await panel.dom.hijos[1].eventosBoton.get("click")(); await esperar();
  assert.equal(lecturas, 2);
  assert.match(panel.dom.hijos[1].innerHTML, /No se pudo recuperar la preparación del cierre/u);
  assert.match(panel.dom.hijos[1].innerHTML, /data-ct-exp-reintentar-cierre/u);
  panel.modulo.desmontar();
});

test("montaje: doble clic de recuperación comparte una única lectura pendiente", async () => {
  let lecturas = 0, resolverReintento;
  const panel = await montarPanel({ cierre: async () => {
    lecturas++;
    if (lecturas === 1) throw new Error("GET inicial sintético no disponible");
    return new Promise((resolver) => { resolverReintento = resolver; });
  }, registrar: async () => { assert.fail("no debe anotar"); } });
  const reintentar = panel.dom.hijos[1].eventosBoton.get("click");
  const primera = reintentar();
  const segunda = reintentar();
  assert.equal(lecturas, 2);
  resolverReintento({ preparacion: { expediente_ref, seguimiento_ref: recibo.seguimiento_ref, version_esperada: 1, motivos: ["sin_cese"] } });
  await Promise.all([primera, segunda]); await esperar();
  assert.equal(lecturas, 2);
  assert.match(panel.dom.hijos[1].innerHTML, /data-ct-cierre-administrativo-form/u);
  panel.modulo.desmontar();
});

test("montaje: entrega sobrescrituras CT explícitas a anotación y cierre sin combinar catálogos", async () => {
  const panel = await montarPanel({ cierre: async () => ({ preparacion: null }), registrar: async () => { assert.fail("sin anotación"); }, mensajes: {
    anotacion_titulo: "Anotación <montada>", cierre_titulo: "Cierre <montado>", cierre_estado_sin_preparacion: "Recuperación <montada>",
  } });
  assert.match(panel.dom.hijos[0].innerHTML, /Anotación &lt;montada&gt;/u);
  assert.match(panel.dom.hijos[1].innerHTML, /Cierre &lt;montado&gt;/u);
  assert.match(panel.dom.hijos[1].innerHTML, /Recuperación &lt;montada&gt;/u);
  assert.doesNotMatch(panel.dom.hijos[0].innerHTML + panel.dom.hijos[1].innerHTML, /<montada>|<montado>/u);
  panel.modulo.desmontar();
});
