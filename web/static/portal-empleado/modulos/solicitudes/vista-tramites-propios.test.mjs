import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { cargarTextos } from "../../../comun/textos.js";
import { montarVistaTramitesPropios, renderizarVistaTramitesPropios } from "./vista-tramites-propios.js";
import { crearFuenteTramitesPropios } from "./fuente-tramites-propios.js";

const tick = () => new Promise((resolve) => setImmediate(resolve));
const ahora = () => new Date("2026-12-31T23:30:00Z");
const solicitud = (i = 1) => ({ solicitud_ref: `sol:${i}`, nombre: `Permiso ${i}`, estado: "solicitado", version: 1, solicitada_en: "2026-10-01T08:00:00Z", desde: "2026-10-02", hasta: "2026-10-03", pendiente_justificar: false });
const comision = (i = 1) => ({ comision: { referencia: `com:${i}`, numero_documento: `COM-${i}`, estado: "fiscalizada", fecha_inicio: "2026-10-02", fecha_fin: "2026-10-03" }, recibo: { referencia: `operacion:${i}`, version: 1, registrado_en: "2026-10-01T08:00:00Z", repeticion: false } });
const diferida = () => { let resolver; let rechazar; const promesa = new Promise((a, b) => { resolver = a; rechazar = b; }); return { promesa, resolver, rechazar }; };

function raizFalsa() {
  const eventos = new Map(); const paneles = new Map(); const focos = [];
  const cuerpo = { id: "body" }; const doc = { activeElement: cuerpo };
  let aviso;
  let html = "";
  const patron = /(<section class="panel" data-tramites-panel="(cronos|dietas)"[^>]*>)([\s\S]*?)<\/section>/g;
  function elemento(panel, id) {
    const etiqueta = panel.innerHTML.match(new RegExp(`<[^>]+id="${id}"[^>]*>`))?.[0];
    if (!etiqueta) return undefined;
    const disabled = /\sdisabled(?:\s|>)/.test(etiqueta);
    return { id, panel, disabled, value: etiqueta.match(/value="([^"]*)"/)?.[1] ?? "", getAttribute: (nombre) => etiqueta.match(new RegExp(`${nombre}="([^"]*)"`))?.[1], focus() { if (!disabled) { doc.activeElement = this; focos.push(id); } } };
  }
  const raiz = {
    eventos, focos, ownerDocument: doc,
    get innerHTML() { return html.replace(patron, (_todo, inicio, bloque) => `${inicio}${paneles.get(bloque).innerHTML}</section>`).replace(/(<p id="tramites-autenticacion-estado"[^>]*>)[\s\S]*?<\/p>/, (_todo, inicio) => `${aviso?.hidden ? inicio : inicio.replace(" hidden", "")}${aviso?.textContent ?? ""}</p>`); },
    set innerHTML(valor) {
      html = valor; paneles.clear();
      aviso = { id: "tramites-autenticacion-estado", hidden: true, textContent: "", focus() { if (!this.hidden) { doc.activeElement = this; focos.push(this.id); } } };
      for (const [, , bloque, contenido] of html.matchAll(patron)) {
        let actual = contenido;
        const panel = { get innerHTML() { return actual; }, set innerHTML(nuevo) { if (doc.activeElement?.panel === panel) doc.activeElement = cuerpo; actual = nuevo; }, contains: (el) => el?.panel === panel, querySelector: (sel) => elemento(panel, sel.slice(1)), setAttribute() {} };
        paneles.set(bloque, panel);
      }
    },
    contains(el) { return el === aviso || [...paneles.values()].some((panel) => panel.contains(el)); },
    querySelector(sel) { if (sel === "#tramites-autenticacion-estado") return aviso; const bloque = sel.match(/^\[data-tramites-panel="(cronos|dietas)"\]$/)?.[1]; return bloque ? paneles.get(bloque) : [...paneles.values()].map((panel) => panel.querySelector(sel)).find(Boolean); },
    replaceChildren() { html = ""; paneles.clear(); aviso = undefined; doc.activeElement = cuerpo; },
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

test("corregir el año ya consultado en vuelo limpia el error sin repetir la petición", async () => {
  const raiz = raizFalsa(); const pendiente = diferida(); let llamadas = 0;
  montarVistaTramitesPropios({ raiz, ahora, fuente: { disponibles: { dietas: false }, consultarCronos: () => {
    llamadas++; return pendiente.promesa;
  } } });
  raiz.anio("1999"); assert.match(raiz.innerHTML, /aria-invalid="true"/);
  raiz.anio("2027"); assert.equal(llamadas, 1); assert.doesNotMatch(raiz.innerHTML, /aria-invalid="true"/);
  pendiente.resolver({ anio: 2027, solicitudes: [solicitud()] }); await tick();
  assert.match(raiz.innerHTML, /Permiso 1/); assert.doesNotMatch(raiz.innerHTML, /aria-invalid="true"|Introduce un año/);
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

test("reintentar Dietas conserva la página solicitada tras un fallo", async () => {
  const raiz = raizFalsa(); const pedidos = []; let fallar = true;
  montarVistaTramitesPropios({ raiz, ahora, fuente: { disponibles: { cronos: false }, consultarDietas: ({ cursor }) => {
    pedidos.push(cursor);
    if (!cursor) return { items: [comision()], siguiente_cursor: "cursor-2" };
    if (fallar) { fallar = false; throw { codigo: "red_no_disponible" }; }
    return { items: [comision(2)] };
  } } });
  await tick(); raiz.click("dietas", "siguiente"); await tick();
  assert.match(raiz.innerHTML, /Reintentar/); assert.doesNotMatch(raiz.innerHTML, /COM-1/);
  raiz.click("dietas", "consultar"); await tick();
  assert.deepEqual(pedidos, [undefined, "cursor-2", "cursor-2"]);
  assert.match(raiz.innerHTML, /COM-2/); assert.match(raiz.innerHTML, /Página 2/);
});

test("Dietas permite volver desde una página vacía sin convertirla en fallo", async () => {
  const raiz = raizFalsa(); const pedidos = [];
  montarVistaTramitesPropios({ raiz, ahora, fuente: { disponibles: { cronos: false }, consultarDietas: ({ cursor }) => {
    pedidos.push(cursor); return cursor ? { items: [] } : { items: [comision()], siguiente_cursor: "cursor-2" };
  } } });
  await tick(); raiz.click("dietas", "siguiente"); await tick();
  assert.match(raiz.innerHTML, /No hay comisiones/); assert.match(raiz.innerHTML, /Página 2/);
  raiz.click("dietas", "anterior"); await tick();
  assert.deepEqual(pedidos, [undefined, "cursor-2", undefined]); assert.match(raiz.innerHTML, /COM-1/);
});

test("respuesta tardía conserva el año en edición y no roba foco al otro panel", async () => {
  const raiz = raizFalsa(); const cronos = diferida(); const dietas = diferida();
  montarVistaTramitesPropios({ raiz, ahora, fuente: { consultarCronos: () => cronos.promesa, consultarDietas: () => dietas.promesa } });
  const entrada = raiz.querySelector("#tramites-cronos-anio"); entrada.focus(); entrada.value = "2031";
  cronos.resolver({ anio: 2027, solicitudes: [solicitud()] }); await tick();
  assert.equal(raiz.ownerDocument.activeElement.id, "tramites-cronos-anio");
  assert.equal(raiz.ownerDocument.activeElement.value, "2031");
  const focos = raiz.focos.length;
  dietas.resolver({ items: [comision()] }); await tick();
  assert.equal(raiz.focos.length, focos); assert.equal(raiz.ownerDocument.activeElement.value, "2031");
});

test("errores mínimos, denegación y relación ambigua no muestran registros previos", async () => {
  for (const codigo of ["acceso_denegado", "autenticacion_requerida", "relacion_ambigua", "sin_empleado", "fuente_no_configurada"]) {
    const raiz = raizFalsa(); let fallar = false;
    montarVistaTramitesPropios({ raiz, ahora, fuente: { disponibles: { cronos: false }, consultarDietas: () => { if (fallar) throw { codigo, message: "detalle interno" }; return { items: [comision()] }; } } });
    await tick(); fallar = true; raiz.click("dietas", "consultar"); await tick();
    assert.doesNotMatch(raiz.innerHTML, /COM-1|operacion:1|detalle interno/);
    if (codigo === "relacion_ambigua") assert.match(raiz.innerHTML, /Abre Mis dietas para elegir/);
    if (codigo.includes("denegado")) assert.match(raiz.innerHTML, /No tienes acceso/);
    if (codigo === "autenticacion_requerida") assert.match(raiz.innerHTML, /Identifícate de nuevo/);
    if (codigo === "fuente_no_configurada") assert.match(raiz.innerHTML, /no está disponible/);
  }
});

test("autenticación requerida purga los dos paneles y descarta la otra respuesta tardía", async () => {
  for (const origen of ["cronos", "dietas"]) {
    const raiz = raizFalsa(); const pendientes = { cronos: diferida(), dietas: diferida() };
    const señales = []; let llamadas = 0; let inicial = true;
    const consultar = (bloque) => ({ anio, signal }) => {
      llamadas++; señales.push(signal);
      return inicial ? bloque === "cronos" ? { anio, solicitudes: [solicitud()] } : { items: [comision()], siguiente_cursor: "cursor-2" } : pendientes[bloque].promesa;
    };
    const vista = montarVistaTramitesPropios({ raiz, ahora, fuente: { consultarCronos: consultar("cronos"), consultarDietas: consultar("dietas") } });
    await tick(); assert.match(raiz.innerHTML, /Permiso 1/); assert.match(raiz.innerHTML, /COM-1/);
    inicial = false;
    // Conservar datos visibles del otro módulo al llegar la primera denegación.
    raiz.click(origen, "consultar");
    pendientes[origen].rechazar({ codigo: "autenticacion_requerida" }); await tick();
    assert.doesNotMatch(raiz.innerHTML, /Permiso 1|COM-1|operacion:1/);
    assert.equal(señales.every((signal) => signal.aborted), true);
    assert.equal((raiz.innerHTML.match(/Identifícate de nuevo/g) ?? []).length, 1);
    const antes = llamadas;
    raiz.click("cronos", "consultar"); raiz.click("dietas", "consultar"); raiz.anio("2025");
    assert.equal(llamadas, antes); vista.desmontar();
  }
  for (const origen of ["cronos", "dietas"]) {
    const raiz = raizFalsa(); const pendientes = { cronos: diferida(), dietas: diferida() }; const señales = [];
    montarVistaTramitesPropios({ raiz, ahora, fuente: {
      consultarCronos: ({ signal }) => { señales.push(signal); return pendientes.cronos.promesa; },
      consultarDietas: ({ signal }) => { señales.push(signal); return pendientes.dietas.promesa; },
    } });
    pendientes[origen].rechazar({ codigo: "autenticacion_requerida" }); await tick();
    assert.equal(señales.every((signal) => signal.aborted), true);
    pendientes[origen === "cronos" ? "dietas" : "cronos"].resolver({ anio: 2027, solicitudes: [solicitud(999)], items: [comision(999)] });
    await tick(); assert.doesNotMatch(raiz.innerHTML, /Permiso 999|COM-999|operacion:999/);
    assert.equal((raiz.innerHTML.match(/Identifícate de nuevo/g) ?? []).length, 1);
  }
});

test("denegar permiso de un módulo conserva los datos y las lecturas del otro", async () => {
  const raiz = raizFalsa(); let llamadas = 0;
  const vista = montarVistaTramitesPropios({ raiz, ahora, fuente: {
    consultarCronos: ({ anio }) => { llamadas++; return { anio, solicitudes: [solicitud()] }; },
    consultarDietas: () => { throw { codigo: "acceso_denegado" }; },
  } });
  await tick(); assert.match(raiz.innerHTML, /Permiso 1/); assert.match(raiz.innerHTML, /No tienes acceso/);
  raiz.anio("2025"); await tick(); assert.equal(llamadas, 2); assert.match(raiz.innerHTML, /Permiso 1/);
  vista.desmontar();
});

test("Dietas sin autenticación lleva el foco del año deshabilitado al aviso y anuncia una sola vez", async () => {
  const raiz = raizFalsa(); const dietas = diferida(); const cronos = diferida(); const anuncios = [];
  const vista = montarVistaTramitesPropios({ raiz, ahora, anunciar: (...args) => anuncios.push(args), fuente: {
    consultarCronos: () => cronos.promesa, consultarDietas: () => dietas.promesa,
  } });
  raiz.querySelector("#tramites-cronos-anio").focus();
  dietas.rechazar({ codigo: "autenticacion_requerida", message: "dato reservado" }); await tick();
  const anio = raiz.querySelector("#tramites-cronos-anio");
  assert.equal(anio.disabled, true); anio.focus();
  assert.equal(raiz.ownerDocument.activeElement.id, "tramites-autenticacion-estado");
  assert.match(raiz.innerHTML, /id="tramites-autenticacion-estado" tabindex="-1">Identifícate de nuevo/);
  assert.deepEqual(anuncios, [["Identifícate de nuevo para consultar tus trámites.", "error"]]);
  cronos.rechazar({ codigo: "autenticacion_requerida" }); await tick();
  assert.equal(anuncios.length, 1); assert.doesNotMatch(raiz.innerHTML, /dato reservado/);
  vista.desmontar();
});

test("la purga de autenticación conserva el foco fuera de la vista", async () => {
  const raiz = raizFalsa(); const dietas = diferida(); const exterior = { id: "menu-portal" };
  const vista = montarVistaTramitesPropios({ raiz, ahora, fuente: { disponibles: { cronos: false }, consultarDietas: () => dietas.promesa } });
  raiz.ownerDocument.activeElement = exterior;
  dietas.rechazar({ codigo: "autenticacion_requerida" }); await tick();
  assert.equal(raiz.ownerDocument.activeElement, exterior);
  vista.desmontar();
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
  const antiguos = new Map(raiz.eventos);
  assert.equal(limpieza, vista.desmontar); limpieza(); limpieza();
  assert.equal(señales.length, 2); assert.equal(señales.every((signal) => signal.aborted), true);
  d.resolver({ anio: 2027, solicitudes: [solicitud()], items: [comision()] }); await tick();
  antiguos.get("submit")({ target: { matches: () => true, elements: { anio: { value: "2025" } } }, preventDefault() {} });
  antiguos.get("click")({ target: { closest: () => ({ dataset: { tramitesBloque: "cronos", tramitesAccion: "consultar" }, getAttribute: () => "false" }) } });
  assert.equal(señales.length, 2);
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

test("el justificante muestra su propia versión, referencia y fecha sin tomar la versión de la comisión", async () => {
  const item = comision(); item.comision.version = 7; item.recibo.version = 3;
  for (const idioma of ["es", "en"]) {
    const textos = await cargarTextos("tramites-empleado", { idioma });
    const html = renderizarVistaTramitesPropios({ dietas: panel({ items: [item] }) }, { textos });
    assert.ok(html.includes(`<th scope="col">${textos.traducir("general.justificante_operacion")}</th>`));
    const detalle = html.match(/<details>([\s\S]*?)<\/details>/)?.[1];
    assert.ok(detalle.includes(`<dt>${textos.traducir("general.version_justificante")}</dt><dd>3</dd>`));
    assert.match(detalle, /operacion:1/); assert.doesNotMatch(detalle, /<dd>7<\/dd>|registro oficial|firmado|notificaci[oó]n/i);
    const fecha = new Intl.DateTimeFormat(textos.localizacion, { dateStyle: "short", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(item.recibo.registrado_en));
    assert.ok(detalle.includes(`<dt>${textos.traducir("general.fecha_operacion")}</dt><dd>${fecha}</dd>`));
    assert.equal(item.recibo.version, 3); assert.equal(item.comision.version, 7);
  }
});

test("sin justificante no fabrica versión ni detalle desde los datos de la comisión", () => {
  const item = comision(); item.comision.version = 7; delete item.recibo;
  const html = renderizarVistaTramitesPropios({ dietas: panel({ items: [item] }) });
  assert.match(html, /Justificante de operación/); assert.match(html, /<td\b[^>]*>No consta<\/td>/);
  assert.doesNotMatch(html, /<details>|Versión del justificante|<dd>7<\/dd>/);
});

test("el catálogo conserva las claves de justificante para una vista anterior ya abierta", async () => {
  for (const idioma of ["es", "en"]) {
    const textos = await cargarTextos("tramites-empleado", { idioma });
    for (const [anterior, actual] of [["registro", "justificante_operacion"], ["ver_registro", "ver_justificante"], ["fecha_registro", "fecha_operacion"]]) {
      assert.equal(textos.traducir(`general.${anterior}`), textos.traducir(`general.${actual}`));
      assert.ok(textos.traducir(`general.${anterior}`).trim());
    }
    assert.ok(textos.traducir("general.version_justificante").trim());
    assert.equal(textos.faltantes.length, 0);
  }
});

test("Mis trámites muestra la devolución propia sin confundir sus versiones ni interpretar el motivo como HTML", async () => {
  const original = { items: [{ comision: { referencia: `dco_${"a".repeat(22)}`, version: 7,
    estado: "devuelta", fecha_inicio: "2026-10-10", fecha_fin: "2026-10-11", motivo: "Contexto que no se proyecta",
    devolucion: { etapa: "revision", motivo: "Revisar <img src=x onerror=alert(1)> el recorrido.",
      version: 3, devuelta_en: "2026-10-03T10:00:00.123456Z" } },
    recibo: { referencia: `rcd_${"b".repeat(22)}`, version: 6, registrado_en: "2026-10-01T10:00:00.123456Z", repeticion: false } }] };
  const datos = await crearFuenteTramitesPropios({ listarComisiones: async () => original }).consultarDietas();
  for (const idioma of ["es", "en"]) {
    const textos = await cargarTextos("tramites-empleado", { idioma });
    const html = renderizarVistaTramitesPropios({ dietas: panel(datos) }, { textos });
    const detalle = html.match(/<details data-tramites-devolucion>([\s\S]*?)<\/details>/)?.[1];
    assert.ok(detalle.includes(`<dt>${textos.traducir("general.devolucion_version")}</dt><dd>3</dd>`));
    assert.ok(detalle.includes(textos.traducir("general.devolucion_etapa_revision")));
    assert.match(detalle, /&lt;img src=x onerror=alert\(1\)&gt;/);
    assert.match(detalle, /href="#dietas" data-vista="dietas"/);
    assert.doesNotMatch(html, /<img|onerror="|Contexto que no se proyecta|plazo|vencimiento/i);
    const justificante = html.match(/<details>([\s\S]*?)<\/details>/)?.[1];
    assert.ok(justificante.includes(`<dt>${textos.traducir("general.version_justificante")}</dt><dd>6</dd>`));
    const fecha = new Intl.DateTimeFormat(textos.localizacion, { dateStyle: "short", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(original.items[0].comision.devolucion.devuelta_en));
    assert.ok(detalle.includes(fecha));
  }
  assert.equal(datos.items[0].comision.version, 7);
  assert.equal(datos.items[0].comision.devolucion.motivo, original.items[0].comision.devolucion.motivo);
});

test("la caducidad de Cronos retira también el motivo personal de devolución de Dietas", async () => {
  const raiz = raizFalsa(); const cronos = diferida(); const item = comision();
  item.comision.devolucion = { etapa: "revision", motivo: "Completa el recorrido de Ana Molina.", version: 3, devuelta_en: "2026-10-03T10:00:00.123456Z" };
  montarVistaTramitesPropios({ raiz, ahora, fuente: {
    consultarCronos: () => cronos.promesa, consultarDietas: async () => ({ items: [item] }),
  } });
  await tick(); assert.match(raiz.innerHTML, /Completa el recorrido de Ana Molina/);
  cronos.rechazar({ codigo: "autenticacion_requerida" }); await tick();
  assert.doesNotMatch(raiz.innerHTML, /Ana Molina|data-tramites-devolucion/);
  assert.match(raiz.innerHTML, /Identifícate de nuevo/);
});

test("hoja nueva no usa almacenamiento, identidad cliente, red ni CSS propio", async () => {
  const codigo = await readFile(new URL("vista-tramites-propios.js", import.meta.url), "utf8");
  assert.doesNotMatch(codigo, /localStorage|sessionStorage|indexedDB|document\.cookie|\bfetch\s*\(|empleado_ref|persona_ref|style=/);
  assert.match(codigo, /ZONA_HORARIA_PORTAL/);
});
