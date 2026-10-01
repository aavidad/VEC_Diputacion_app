import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { cargarTextos } from "../../../comun/textos.js";
import { montarVistaTramitesPropios, renderizarVistaTramitesPropios } from "./vista-tramites-propios.js";

const tick = () => new Promise((resolve) => setImmediate(resolve));
const ahora = () => new Date("2026-12-31T23:30:00Z");
const solicitud = (i = 1) => ({ solicitud_ref: `sol:${i}`, nombre: `Permiso ${i}`, estado: "solicitado", version: 1, solicitada_en: "2026-10-01T08:00:00Z", desde: "2026-10-02", hasta: "2026-10-03", pendiente_justificar: false });
const comision = (i = 1) => ({ comision: { referencia: `com:${i}`, numero_documento: `COM-${i}`, estado: "fiscalizada", fecha_inicio: "2026-10-02", fecha_fin: "2026-10-03" }, recibo: { referencia: `operacion:${i}`, version: 1, registrado_en: "2026-10-01T08:00:00Z", repeticion: false } });
const diferida = () => { let resolver; let rechazar; const promesa = new Promise((a, b) => { resolver = a; rechazar = b; }); return { promesa, resolver, rechazar }; };

function raizFalsa() {
  const eventos = new Map(); const paneles = new Map(); const focos = [];
  const doc = { activeElement: undefined };
  let html = "";
  const patron = /(<section class="panel" data-tramites-panel="(cronos|dietas)"[^>]*>)([\s\S]*?)<\/section>/g;
  function elemento(panel, id) {
    const etiqueta = panel.innerHTML.match(new RegExp(`<[^>]+id="${id}"[^>]*>`))?.[0];
    if (!etiqueta) return undefined;
    return { id, panel, value: etiqueta.match(/value="([^"]*)"/)?.[1] ?? "", focus() { doc.activeElement = this; focos.push(id); } };
  }
  const raiz = {
    eventos, focos, ownerDocument: doc,
    get innerHTML() { return html.replace(patron, (_todo, inicio, bloque) => `${inicio}${paneles.get(bloque).innerHTML}</section>`); },
    set innerHTML(valor) {
      html = valor; paneles.clear();
      for (const [, , bloque, contenido] of html.matchAll(patron)) {
        const panel = { innerHTML: contenido, contains: (el) => el?.panel === panel, querySelector: (sel) => elemento(panel, sel.slice(1)), setAttribute() {} };
        paneles.set(bloque, panel);
      }
    },
    querySelector(sel) { const bloque = sel.match(/^\[data-tramites-panel="(cronos|dietas)"\]$/)?.[1]; return bloque ? paneles.get(bloque) : [...paneles.values()].map((panel) => panel.querySelector(sel)).find(Boolean); },
    replaceChildren() { html = ""; paneles.clear(); },
    addEventListener(tipo, fn) { eventos.set(tipo, fn); },
    removeEventListener(tipo) { eventos.delete(tipo); },
  };
  raiz.click = (bloque, accion) => {
    const id = `tramites-${bloque}-${accion}`;
    const panel = paneles.get(bloque);
    const impedido = panel.innerHTML.match(new RegExp(`id="${id}"[^>]+aria-disabled="(true|false)"`))?.[1];
    raiz.eventos.get("click")({ target: { closest: () => ({ dataset: { tramitesBloque: bloque, tramitesAccion: accion }, getAttribute: () => impedido }) } });
  };
  raiz.anio = (value, tipo = "submit") => {
    const formulario = { elements: { anio: { value } }, matches: () => true };
    if (tipo === "change") raiz.eventos.get(tipo)({ target: { id: "tramites-cronos-anio", closest: () => formulario } });
    else raiz.eventos.get(tipo)({ target: formulario, preventDefault() {} });
  };
  return raiz;
}
const panel = (datos, resto = {}) => ({ visible: true, situacion: "disponible", datos, anio: 2026, pagina: 0, cursores: [], ...resto });

test("los catálogos reales ES/EN tienen las mismas claves y la vista respeta sus textos", async () => {
  const es = await cargarTextos("tramites-empleado", { idioma: "es" });
  const en = await cargarTextos("tramites-empleado", { idioma: "en" });
  assert.deepEqual(Object.keys(es.seccion("general")), Object.keys(en.seccion("general")));
  assert.equal(en.faltantes.length, 0);
  const estado = { cronos: panel({ solicitudes: [solicitud()] }), dietas: panel({ items: [comision()] }) };
  assert.match(renderizarVistaTramitesPropios(estado, { textos: es }), /Mis trámites/);
  const html = renderizarVistaTramitesPropios(estado, { textos: en });
  assert.match(html, /My requests/); assert.match(html, /Financially reviewed/);
  assert.doesNotMatch(html, /Fiscalizada|paid|pagada|total global|cronolog/i);
  assert.match(html, /href="#cronos-permisos" data-vista="cronos-permisos"/);
  assert.match(html, /href="#dietas" data-vista="dietas"/);
});

test("dos consultas independientes: carga de uno y error del otro no ocultan datos", async () => {
  const raiz = raizFalsa(); const cronos = diferida(); const dietas = diferida();
  montarVistaTramitesPropios({ raiz, ahora, fuente: { consultarCronos: () => cronos.promesa, consultarDietas: () => dietas.promesa } });
  assert.match(raiz.innerHTML, /Cargando permisos/); assert.match(raiz.innerHTML, /Cargando dietas/);
  cronos.resolver({ anio: 2027, solicitudes: [solicitud()] }); await tick();
  assert.match(raiz.innerHTML, /Permiso 1/); assert.match(raiz.innerHTML, /Cargando dietas/);
  dietas.rechazar({ codigo: "red_no_disponible", message: "HTTP 503 secreto" }); await tick();
  assert.match(raiz.innerHTML, /Permiso 1/); assert.match(raiz.innerHTML, /No se han podido consultar las dietas/);
  assert.doesNotMatch(raiz.innerHTML, /HTTP|503|secreto|Mostrando 0/);
});

test("año de Madrid, validación, cancelación y descarte tardío con foco del campo", async () => {
  const raiz = raizFalsa(); const solicitudes = [];
  montarVistaTramitesPropios({ raiz, ahora, fuente: { disponibles: { dietas: false }, consultarCronos: (peticion) => { const d = diferida(); solicitudes.push({ ...peticion, ...d }); return d.promesa; } } });
  assert.equal(solicitudes[0].anio, 2027);
  raiz.anio("1999"); assert.equal(solicitudes.length, 1); assert.match(raiz.innerHTML, /aria-invalid="true"/);
  for (const invalido of ["2101", "2026.5", "2e3", "", "20000"]) raiz.anio(invalido);
  assert.equal(solicitudes.length, 1);
  raiz.anio("2000", "change"); assert.equal(solicitudes[0].signal.aborted, true);
  solicitudes[0].resolver({ anio: 2027, solicitudes: [solicitud(999)] });
  solicitudes[1].resolver({ anio: 2000, solicitudes: [solicitud(2)] }); await tick();
  assert.match(raiz.innerHTML, /Permiso 2/); assert.doesNotMatch(raiz.innerHTML, /Permiso 999/);
  assert.equal(raiz.focos.at(-1), "tramites-cronos-anio");
  raiz.anio("2100"); assert.equal(solicitudes[2].anio, 2100);
});

test("Cronos pagina local de20 sin nueva consulta y año nuevo vuelve a primera página", async () => {
  const raiz = raizFalsa(); let llamadas = 0;
  montarVistaTramitesPropios({ raiz, ahora, fuente: { disponibles: { dietas: false }, consultarCronos: ({ anio }) => { llamadas++; return { anio, solicitudes: Array.from({ length: 21 }, (_, i) => solicitud(i + 1)) }; } } });
  await tick(); assert.match(raiz.innerHTML, /Mostrando 1 a 20 de 21/); assert.doesNotMatch(raiz.innerHTML, /Permiso 21</);
  raiz.querySelector("#tramites-cronos-siguiente").focus(); raiz.click("cronos", "siguiente");
  assert.match(raiz.innerHTML, /Mostrando 21 a 21 de 21/); assert.equal(llamadas, 1);
  assert.equal(raiz.focos.at(-1), "tramites-cronos-siguiente");
  raiz.click("cronos", "siguiente"); assert.equal(llamadas, 1);
  raiz.anio("2025"); await tick(); assert.match(raiz.innerHTML, /Mostrando 1 a 20 de 21/); assert.equal(llamadas, 2);
});

test("Dietas sigue cursor y vuelve usando sólo la pila en memoria; conserva foco", async () => {
  const raiz = raizFalsa(); const pedidos = [];
  montarVistaTramitesPropios({ raiz, ahora, fuente: { disponibles: { cronos: false }, consultarDietas: ({ cursor }) => { pedidos.push(cursor); return cursor ? { items: [comision(2)] } : { items: [comision()], siguiente_cursor: "cursor-2" }; } } });
  await tick(); raiz.querySelector("#tramites-dietas-siguiente").focus(); raiz.click("dietas", "siguiente");
  assert.equal(raiz.focos.at(-1), "tramites-dietas-siguiente"); await tick();
  assert.match(raiz.innerHTML, /COM-2/); assert.match(raiz.innerHTML, /Página 2/);
  raiz.click("dietas", "anterior"); await tick();
  assert.deepEqual(pedidos, [undefined, "cursor-2", undefined]); assert.match(raiz.innerHTML, /COM-1/);
});

test("errores mínimos, denegación y relación ambigua no muestran registros previos", async () => {
  for (const codigo of ["acceso_denegado", "autenticacion_requerida", "relacion_ambigua", "sin_empleado", "fuente_no_configurada"]) {
    const raiz = raizFalsa(); let fallar = false;
    montarVistaTramitesPropios({ raiz, ahora, fuente: { disponibles: { cronos: false }, consultarDietas: () => { if (fallar) throw { codigo, message: "detalle interno" }; return { items: [comision()] }; } } });
    await tick(); fallar = true; raiz.click("dietas", "consultar"); await tick();
    assert.doesNotMatch(raiz.innerHTML, /COM-1|operacion:1|detalle interno/);
    if (codigo === "relacion_ambigua") assert.match(raiz.innerHTML, /Abre Mis dietas para elegir/);
    if (codigo.includes("denegado") || codigo === "autenticacion_requerida") assert.match(raiz.innerHTML, /No tienes acceso/);
    if (codigo === "fuente_no_configurada") assert.match(raiz.innerHTML, /no está disponible/);
  }
});

test("vacío, fuente ausente y bloques deshabilitados se distinguen del fallo", async () => {
  const vacio = raizFalsa(); montarVistaTramitesPropios({ raiz: vacio, ahora, fuente: { consultarCronos: ({ anio }) => ({ anio, solicitudes: [] }), consultarDietas: () => ({ items: [] }) } });
  await tick(); assert.match(vacio.innerHTML, /No hay solicitudes/); assert.match(vacio.innerHTML, /No hay comisiones/);
  const sinFuente = raizFalsa(); montarVistaTramitesPropios({ raiz: sinFuente, ahora }); await tick();
  assert.match(sinFuente.innerHTML, /no está disponible/); assert.doesNotMatch(sinFuente.innerHTML, /No hay solicitudes|No hay comisiones/);
  const sinBloques = raizFalsa(); let llamadas = 0;
  montarVistaTramitesPropios({ raiz: sinBloques, ahora, fuente: { disponibles: { cronos: false, dietas: false }, consultarCronos: () => llamadas++, consultarDietas: () => llamadas++ } });
  assert.equal(llamadas, 0); assert.match(sinBloques.innerHTML, /No hay consultas disponibles/); assert.doesNotMatch(sinBloques.innerHTML, /data-tramites-panel/);
});

test("reintento funciona y mantiene el foco; datos malos producen error y nunca vacío", async () => {
  const raiz = raizFalsa(); let llamada = 0;
  montarVistaTramitesPropios({ raiz, ahora, fuente: { disponibles: { cronos: false }, consultarDietas: () => ++llamada === 1 ? { items: [{ comision: {} }] } : { items: [comision()] } } });
  await tick(); assert.match(raiz.innerHTML, /Reintentar/); assert.doesNotMatch(raiz.innerHTML, /No hay comisiones/);
  raiz.querySelector("#tramites-dietas-consultar").focus(); raiz.click("dietas", "consultar"); await tick();
  assert.match(raiz.innerHTML, /COM-1/); assert.equal(raiz.focos.at(-1), "tramites-dietas-consultar");
});

test("desmontar cancela ambos paneles y descarta respuestas que ignoran abort", async () => {
  const raiz = raizFalsa(); const d = diferida(); const señales = []; let limpieza;
  const consulta = ({ signal }) => { señales.push(signal); return d.promesa; };
  const vista = montarVistaTramitesPropios({ raiz, ahora, registrarDesmontar: (fn) => limpieza = fn, fuente: { consultarCronos: consulta, consultarDietas: consulta } });
  assert.equal(limpieza, vista.desmontar); limpieza(); limpieza();
  assert.equal(señales.length, 2); assert.equal(señales.every((signal) => signal.aborted), true);
  d.resolver({ anio: 2027, solicitudes: [solicitud()], items: [comision()] }); await tick();
  assert.equal(raiz.innerHTML, ""); assert.equal(raiz.eventos.size, 0);
});

test("fechas civiles no cambian de día, instantes usan Madrid y fechas imposibles no se normalizan", () => {
  const s = { ...solicitud(), desde: "2026-02-30", hasta: "2026-03-01", solicitada_en: "2026-12-31T23:30:00Z" };
  const html = renderizarVistaTramitesPropios({ cronos: panel({ solicitudes: [s] }) });
  assert.match(html, /No consta – 1\/3\/26/); assert.match(html, /1\/1\/27, 0:30/);
  const falta = { ...solicitud(), solicitada_en: "2026-10-01" };
  assert.match(renderizarVistaTramitesPropios({ cronos: panel({ solicitudes: [falta] }) }), /<td>No consta<\/td>/);
});

test("todos los estados de origen se traducen, el desconocido no llega sin filtrar", async () => {
  const textos = await cargarTextos("tramites-empleado", { idioma: "en" });
  const cronos = ["solicitado", "pendiente_administracion", "concedido", "denegado", "cancelado"].map((estado) => ({ ...solicitud(), estado }));
  const dietas = ["borrador", "eliminado", "enviado_pendiente_revision", "pendiente_autorizacion", "pendiente_liquidacion", "pendiente_fiscalizacion", "fiscalizada", "devuelta", "estado_inyectado"].map((estado) => ({ ...comision(), comision: { ...comision().comision, estado } }));
  const html = renderizarVistaTramitesPropios({ cronos: panel({ solicitudes: cronos }), dietas: panel({ items: dietas }) }, { textos });
  for (const estado of cronos) assert.match(html, new RegExp(textos.traducir(`general.cronos_estado_${estado.estado}`)));
  assert.match(html, /Status unavailable/); assert.doesNotMatch(html, /estado_inyectado/);
});

test("escapa datos y traducciones, referencias sólo en details y no crea recibo Cronos", () => {
  const s = { ...solicitud(), nombre: '<script>alert("x")</script>' };
  const d = comision(); d.comision.numero_documento = '<img onerror="x">'; d.recibo.referencia = '<svg onload="x">';
  const html = renderizarVistaTramitesPropios({ cronos: panel({ solicitudes: [s] }), dietas: panel({ items: [d] }) });
  assert.match(html, /&lt;script&gt;/); assert.match(html, /&lt;img onerror=&quot;x&quot;&gt;/);
  assert.match(html, /<details>[\s\S]*&lt;svg onload=&quot;x&quot;&gt;[\s\S]*<\/details>/);
  assert.doesNotMatch(html, /<script|<svg|<img|sol:1|com:1|download=/);
  const traducido = renderizarVistaTramitesPropios({}, { t: () => '<script>x</script>' }); assert.doesNotMatch(traducido, /<script/);
});

test("hoja nueva no usa almacenamiento, identidad cliente, red ni CSS propio", async () => {
  const codigo = await readFile(new URL("vista-tramites-propios.js", import.meta.url), "utf8");
  assert.doesNotMatch(codigo, /localStorage|sessionStorage|indexedDB|document\.cookie|\bfetch\s*\(|empleado_ref|persona_ref|style=/);
  assert.match(codigo, /ZONA_HORARIA_PORTAL/);
});
