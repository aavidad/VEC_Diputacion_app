import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { montarModuloRPTPublica } from "./vista-rpt-publica.js";
import { IDIOMA_EFECTIVO_RPT_PUESTOS } from "./i18n-rpt-puestos.js";

function raiz() { class Nodo { constructor(documento, etiqueta = "div") { this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {}; this.listeners = new Map(); this.parent = null; this.textContent = ""; this.atributos = new Map(); this.focused = false; } append(...nodos) { this.children.push(...nodos); nodos.forEach((n) => { n.parent = this; }); } replaceChildren(...nodos) { this.children.forEach((n) => { n.parent = null; }); this.children = []; this.append(...nodos); } removeChild(n) { this.children = this.children.filter((h) => h !== n); n.parent = null; } remove() { this.parent?.removeChild(this); } addEventListener(t, h) { this.listeners.set(t, h); } setAttribute(k, v) { this.atributos.set(k, v); } contains(n) { for (let actual = n; actual; actual = actual.parent) { if (actual === this) return true; } return false; } focus() { let actual = this; while (actual) { if (actual === this.ownerDocument.raiz) { this.focused = true; this.ownerDocument.activeElement = this; return; } actual = actual.parent; } } matches(s) { const k = s.match(/^\[data-([a-z-]+)\]$/u)?.[1]?.replace(/-([a-z])/g, (_m, l) => l.toUpperCase()); return k ? this.dataset[k] !== undefined : false; } querySelector(s) { if (this.matches(s)) return this; for (const h of this.children) { const e = h.querySelector(s); if (e) return e; } return null; } } const d = { createElement: (e) => new Nodo(d, e) }; const salida = new Nodo(d, "root"); d.raiz = salida; return salida; }
function textoNodo(nodo) { return `${nodo.textContent} ${nodo.children.map(textoNodo).join(" ")}`; }
function nodosCon(nodo, clave, salida = []) { if (nodo.dataset[clave] !== undefined) salida.push(nodo); nodo.children.forEach((hijo) => nodosCon(hijo, clave, salida)); return salida; }
function propagarTecla(origen, key) { const evento = { key, cancelado: false, preventDefault() { this.cancelado = true; } }; for (let actual = origen; actual; actual = actual.parent) actual.listeners.get("keydown")?.(evento); return evento; }
function pagina({ vista = "categorias", total = 1, offset = 0, generadoEn = "2026-09-17", items = [{ clave: "administrativo", denominacion: "ADMINISTRATIVO", grupos: ["C1"], escalas: ["AG"], puestos: 57, dotacion: 158, puestos_vinculados: 57, dotacion_vinculada: 158, recuento_coincide: true }] } = {}) { return Object.freeze({ items: Object.freeze(items), total, limit: 25, offset, vista, esquema: "vec.catalogo.rpt.v1", fuente: Object.freeze({ documento: "RPT publicada", importacion: "rpt-v1", generado_en: generadoEn, aviso: "Datos públicos sin ocupantes.", huella_sha256: "a".repeat(64) }), resumen: Object.freeze({ puestos: 842, dotacion: 1714, categorias: 145, centros: 41 }) }); }
test("catálogos ES/EN rotulan centro y generación sin atribuir vigencia", () => {
  const catalogo = (idioma) => JSON.parse(readFileSync(new URL(`../../../textos/${idioma}/personal.json`, import.meta.url), "utf8")).rpt_puestos;
  const es = catalogo("es"), en = catalogo("en");
  assert.deepEqual(Object.keys(es).sort(), Object.keys(en).sort());
  assert.equal(es.centro_codigo, "Código del centro");
  assert.match(es.generacion, /Generación de la proyección: \{valor\}.*no acredita la vigencia/u);
  assert.match(en.generacion, /generation: \{valor\}.*does not establish/u);
});
test("vista RPT separa resumen y fuente, deja ayuda útil tras ? y oculta metadatos técnicos", async () => { const r = raiz(); const llamadas = []; const modulo = await montarModuloRPTPublica({ raiz: r, cliente: { async listar(consulta) { llamadas.push(consulta); return pagina({ total: 26, vista: consulta.vista }); } } }); const contenedor = r.querySelector("[data-personal-rpt-publica]"); assert.ok(contenedor); const resumen = contenedor.querySelector("[data-personal-rpt-publica-resumen]"); assert.match(textoNodo(resumen), /842 puestos/); assert.doesNotMatch(textoNodo(resumen), /Fuente:|ocupantes/); assert.doesNotMatch(textoNodo(resumen), new RegExp("a".repeat(64))); assert.doesNotMatch(textoNodo(contenedor.children[0]), /Portal del Empleado/); const ayuda = contenedor.querySelector("[data-personal-rpt-publica-ayuda]"); const abrirAyuda = contenedor.querySelector("[data-personal-rpt-publica-abrir-ayuda]"); assert.equal(ayuda.hidden, true); assert.equal(abrirAyuda.tagName, "button"); assert.equal(abrirAyuda.textContent, "?"); assert.equal(abrirAyuda.atributos.get("aria-controls"), ayuda.id); assert.equal(abrirAyuda.atributos.get("aria-expanded"), "false"); assert.match(abrirAyuda.atributos.get("aria-label"), /Consulta pública/); assert.doesNotMatch(textoNodo(contenedor), new RegExp("a".repeat(64))); assert.doesNotMatch(textoNodo(contenedor), /rpt-v1|SHA-256|Importación/u); assert.match(textoNodo(ayuda), /Fuente: RPT publicada\./); assert.doesNotMatch(textoNodo(ayuda), /Datos públicos sin ocupantes/u); assert.match(textoNodo(ayuda), /Generación de la proyección: 17 de septiembre de 2026/); assert.match(textoNodo(ayuda), /no acredita la vigencia administrativa/); assert.doesNotMatch(textoNodo(ayuda), /Ejercicio:/); abrirAyuda.focus(); abrirAyuda.listeners.get("click")(); assert.equal(ayuda.hidden, false); assert.equal(abrirAyuda.atributos.get("aria-expanded"), "true"); assert.equal(abrirAyuda.focused, true); const escape = propagarTecla(abrirAyuda, "Escape"); assert.equal(escape.cancelado, true); assert.equal(ayuda.hidden, true); assert.equal(abrirAyuda.atributos.get("aria-expanded"), "false"); assert.equal(abrirAyuda.focused, true); const tabla = contenedor.children.find((n) => n.className === "tabla-contenedor"); assert.equal(tabla.atributos.get("tabindex"), "0"); assert.equal(tabla.atributos.get("role"), "region"); assert.equal(tabla.atributos.get("aria-label"), "Tabla de categorías RPT"); const puestos = contenedor.querySelector("[data-personal-rpt-publica-vista]"); assert.ok(puestos); const siguiente = contenedor.querySelector("[data-personal-rpt-publica-siguiente]"); siguiente.listeners.get("click")(); await Promise.resolve(); assert.equal(llamadas[1].offset, 25); assert.equal(llamadas[0].vista, "categorias"); modulo.desmontar(); });
test("vista RPT no conserva filas en error y desmontar aborta sin DOM tardío", async () => { const r = raiz(); const avisos = []; await montarModuloRPTPublica({ raiz: r, anunciar: (...a) => avisos.push(a), cliente: { async listar() { throw new Error("503"); } } }); assert.equal(avisos.length, 1); assert.match(r.querySelector("[data-personal-rpt-publica]").children.map((n) => n.textContent).join(" "), /No se pudo consultar/); let resolver; let signal; let limpiar; const r2 = raiz(); const montaje = montarModuloRPTPublica({ raiz: r2, registrarDesmontar: (f) => { limpiar = f; }, cliente: { listar(_q, opciones) { signal = opciones.signal; return new Promise((resolve) => { resolver = resolve; }); } } }); await Promise.resolve(); limpiar(); resolver(pagina()); const modulo = await montaje; modulo.desmontar(); assert.equal(signal.aborted, true); assert.equal(r2.querySelector("[data-personal-rpt-publica]"), null); });
test("vista RPT de puestos pinta categoría presente y ausencia consignada", async () => { const r = raiz(); const filas = [{ codigo: "430-101-001", denominacion: "SECRETARIA DE GRUPO", centro_codigo: "101", centro: "GABINETE", delegacion: "PRESIDENCIA", grupos: [], escala: "", categoria_clave: "administrativo", nivel_destino: 17, complemento_especifico_anual_centimos: 1427496, dotacion: 1, tipo: "E", provision: "I" }, { codigo: "430-101-002", denominacion: "SECRETARIA", centro_codigo: "101", centro: "GABINETE", delegacion: "PRESIDENCIA", grupos: [], escala: "", categoria_clave: "", nivel_destino: 17, complemento_especifico_anual_centimos: 1427496, dotacion: 1, tipo: "E", provision: "I" }]; await montarModuloRPTPublica({ raiz: r, cliente: { async listar() { return pagina({ vista: "puestos", total: 2, items: filas }); } } }); const tabla = r.querySelector("[data-personal-rpt-publica]").children.find((n) => n.className === "tabla-contenedor"); nodosCon(tabla, "personalRptPublicaDetalleBoton")[0].listeners.get("click")(); const texto = textoNodo(tabla); assert.match(texto, /administrativo/); assert.match(texto, /No consignada/); assert.doesNotMatch(texto, /undefined/); });
test("puesto separa seis campos de lectura rápida y siete de detalle sin repetirlos; Escape devuelve foco", async () => { const r = raiz(); const filas = [{ codigo: "430-101-001", denominacion: "SECRETARIA", centro_codigo: "101", centro: "GABINETE", delegacion: "PRESIDENCIA", grupos: ["C1"], escala: "AG", categoria_clave: "administrativo", nivel_destino: 17, complemento_especifico_anual_centimos: 1427496, dotacion: 1, tipo: "E", provision: "I" }, { codigo: "430-101-<img>", denominacion: "TÉCNICA <script>", centro_codigo: "102", centro: "CENTRO", delegacion: "RRHH", grupos: ["A2"], escala: "AE", categoria_clave: "tecnico", nivel_destino: 20, complemento_especifico_anual_centimos: 1427496, dotacion: 1, tipo: "F", provision: "C" }]; await montarModuloRPTPublica({ raiz: r, cliente: { async listar() { return pagina({ vista: "puestos", total: 2, items: filas }); } } }); const contenedor = r.querySelector("[data-personal-rpt-publica]"); const tabla = contenedor.children.find((n) => n.className === "tabla-contenedor"); let controles = nodosCon(contenedor, "personalRptPublicaDetalleBoton"); assert.equal(controles.length, 2); assert.equal(controles[1].atributos.get("aria-expanded"), "false"); controles[1].listeners.get("keydown")({ key: " ", preventDefault() {} }); let detalle = contenedor.querySelector("[data-personal-rpt-publica-detalle-fila]"); assert.ok(detalle); assert.match(textoNodo(detalle), /Complemento específico anual/); assert.equal(detalle.children[0].colSpan, 6); const encabezados = tabla.children[0].children[1].children[0].children.map((n) => n.textContent); assert.deepEqual(encabezados, ["Código", "Denominación", "Centro", "Grupos", "Nivel", "Dotación"]); const etiquetasDetalle = detalle.children[0].children[0].children[2].children.filter((n) => n.tagName === "dt").map((n) => n.textContent); assert.deepEqual(etiquetasDetalle, ["Código del centro", "Delegación", "Escala", "Categoría", "Complemento específico anual", "Tipo", "Provisión"]); assert.equal(new Set([...encabezados, ...etiquetasDetalle]).size, 13); assert.match(textoNodo(detalle), /Código del centro\s+102/u); assert.equal(detalle.children[0].children[0].children[2].children.find((n) => n.tagName === "dd").textContent, "102"); assert.match(textoNodo(tabla), /TÉCNICA <script>/); controles = nodosCon(contenedor, "personalRptPublicaDetalleBoton"); assert.equal(controles[1].atributos.get("aria-expanded"), "true"); detalle.parent.listeners.get("keydown")({ key: "Escape", preventDefault() {} }); assert.equal(contenedor.querySelector("[data-personal-rpt-publica-detalle-fila]"), null); controles = nodosCon(contenedor, "personalRptPublicaDetalleBoton"); assert.equal(controles[1].focused, true); controles[1].listeners.get("click")(); assert.ok(contenedor.querySelector("[data-personal-rpt-publica-detalle-fila]")); });

test("la generación ISO se localiza y otros textos fuente se conservan sin interpretarlos", async () => {
  for (const [origen, esperado] of [
    ["2026-09-17T10:30:00Z", /17 de septiembre de 2026.*12:30/u],
    ["<fecha fuente 2026-02-30>", /<fecha fuente 2026-02-30>/u],
  ]) {
    const r = raiz();
    await montarModuloRPTPublica({ raiz: r, cliente: { async listar() { return pagina({ generadoEn: origen }); } } });
    const ayuda = r.querySelector("[data-personal-rpt-publica-ayuda]");
    assert.match(textoNodo(ayuda), esperado);
    assert.match(textoNodo(ayuda), /no acredita la vigencia administrativa/u);
    assert.ok(ayuda.children.flatMap((n) => n.children).some((n) => n.tagName === "p" && n.textContent.includes(origen.startsWith("<") ? origen : "Generación")));
  }
});

const esperarRespuesta = async () => { await Promise.resolve(); await Promise.resolve(); };
function enviarBusqueda(r, texto) {
  const entrada = r.querySelector("[data-personal-rpt-publica-busqueda]");
  entrada.value = texto; entrada.focus();
  r.querySelector("[data-personal-rpt-publica-filtros]").listeners.get("submit")({ preventDefault() {} });
}
test("búsqueda aplicada distingue texto sin enviar y quitar vuelve a la primera página conservando vista", async () => {
  const r = raiz(), llamadas = [];
  await montarModuloRPTPublica({ raiz: r, cliente: { async listar(consulta) {
    llamadas.push(consulta); return pagina({ vista: consulta.vista, total: 51, offset: consulta.offset });
  } } });
  assert.equal(r.querySelector("[data-personal-rpt-publica-quitar-busqueda]"), null);
  nodosCon(r, "personalRptPublicaVista")[1].listeners.get("click")(); await esperarRespuesta();
  enviarBusqueda(r, "  administrativo  "); await esperarRespuesta();
  assert.equal(r.querySelector("[data-personal-rpt-publica-busqueda-aplicada]").textContent, "Búsqueda aplicada: administrativo");
  r.querySelector("[data-personal-rpt-publica-busqueda]").value = "sin enviar";
  r.querySelector("[data-personal-rpt-publica-siguiente]").listeners.get("click")(); await esperarRespuesta();
  assert.deepEqual(llamadas.at(-1), { vista: "puestos", q: "administrativo", categoria_clave: "", centro_codigo: "", codigo_puesto: "", limit: 25, offset: 25 });
  assert.equal(r.querySelector("[data-personal-rpt-publica-busqueda]").value, "sin enviar");
  const quitar = r.querySelector("[data-personal-rpt-publica-quitar-busqueda]"); quitar.focus(); quitar.listeners.get("click")();
  await esperarRespuesta();
  assert.deepEqual(llamadas.at(-1), { vista: "puestos", q: "", categoria_clave: "", centro_codigo: "", codigo_puesto: "", limit: 25, offset: 0 });
  assert.equal(r.querySelector("[data-personal-rpt-publica-busqueda-aplicada]"), null);
  assert.equal(r.querySelector("[data-personal-rpt-publica-quitar-busqueda]"), null);
  assert.equal(r.querySelector("[data-personal-rpt-publica-busqueda]").value, "");
  assert.equal(r.ownerDocument.activeElement, r.querySelector("[data-personal-rpt-publica-busqueda]"));
});
test("la respuesta conserva texto y foco del buscador; no roba el foco fuera del buscador", async () => {
  const r = raiz(); let resolver;
  await montarModuloRPTPublica({ raiz: r, cliente: { listar(consulta) {
    return consulta.q === "" ? Promise.resolve(pagina()) : new Promise((resolve) => { resolver = resolve; });
  } } });
  enviarBusqueda(r, "administrativo");
  const entrada = r.querySelector("[data-personal-rpt-publica-busqueda]");
  assert.equal(r.ownerDocument.activeElement, entrada);
  entrada.value = "siguiente búsqueda";
  resolver(pagina()); await esperarRespuesta();
  assert.equal(r.querySelector("[data-personal-rpt-publica-busqueda]"), entrada);
  assert.equal(entrada.value, "siguiente búsqueda"); assert.equal(r.ownerDocument.activeElement, entrada);
  enviarBusqueda(r, "otra");
  const fuera = r.ownerDocument.createElement("button"); r.append(fuera); fuera.focus();
  resolver(pagina()); await esperarRespuesta();
  assert.equal(r.ownerDocument.activeElement, fuera);
});
test("quitar búsqueda sigue disponible tras vacío o error, sin recuperar filas antiguas", async () => {
  for (const fallo of [false, true]) {
    const r = raiz();
    await montarModuloRPTPublica({ raiz: r, cliente: { async listar(consulta) {
      if (consulta.q && fallo) throw new Error("fallo sintético");
      return consulta.q ? pagina({ total: 0, items: [] }) : pagina();
    } } });
    enviarBusqueda(r, "ausente"); await esperarRespuesta();
    assert.doesNotMatch(textoNodo(r), /ADMINISTRATIVO/u);
    const quitar = r.querySelector("[data-personal-rpt-publica-quitar-busqueda]");
    assert.ok(quitar); quitar.focus(); quitar.listeners.get("click")(); await esperarRespuesta();
    assert.match(textoNodo(r), /ADMINISTRATIVO/u);
    assert.equal(r.querySelector("[data-personal-rpt-publica-quitar-busqueda]"), null);
  }
});
test("una búsqueda tardía no reaparece al quitarla y desmontar cancela la consulta vigente", async () => {
  const r = raiz(), pendientes = [];
  const modulo = await montarModuloRPTPublica({ raiz: r, cliente: { listar(consulta, { signal }) {
    if (consulta.q === "") return Promise.resolve(pagina());
    return new Promise((resolve) => pendientes.push({ resolve, signal }));
  } } });
  enviarBusqueda(r, "antigua");
  const quitar = r.querySelector("[data-personal-rpt-publica-quitar-busqueda]");
  quitar.focus(); quitar.listeners.get("click")(); await esperarRespuesta();
  assert.equal(pendientes[0].signal.aborted, true);
  pendientes[0].resolve(pagina({ total: 0, items: [] })); await esperarRespuesta();
  assert.match(textoNodo(r), /ADMINISTRATIVO/u);
  assert.equal(r.querySelector("[data-personal-rpt-publica-busqueda-aplicada]"), null);
  enviarBusqueda(r, "pendiente"); modulo.desmontar();
  assert.equal(pendientes[1].signal.aborted, true);
  pendientes[1].resolve(pagina()); await esperarRespuesta();
  assert.equal(r.querySelector("[data-personal-rpt-publica]"), null);
});

test("categoría ambigua abre sólo sus puestos vinculados y conserva aviso", async () => {
  const r = raiz(), llamadas = [];
  const categoria = { clave: "administrativo", denominacion: "ADMINISTRATIVO", grupos: ["C1"], escalas: ["AG"], puestos: 57, dotacion: 158,
    puestos_vinculados: 62, dotacion_vinculada: 163, recuento_coincide: false };
  await montarModuloRPTPublica({ raiz: r, cliente: { async listar(consulta) { llamadas.push(consulta); return pagina({ vista: consulta.vista, total: consulta.vista === "categorias" ? 1 : 0, items: consulta.vista === "categorias" ? [categoria] : [] }); } } });
  assert.match(textoNodo(r), /62/u); assert.match(textoNodo(r), /cifra de la fuente difiere/u);
  const botones = nodosCon(r, "personalRptPublicaEnlace");
  assert.ok(botones.some((b) => b.textContent === "62"));
  botones.find((b) => b.textContent === "62").listeners.get("click")(); await Promise.resolve();
  assert.equal(llamadas.at(-1).vista, "puestos"); assert.equal(llamadas.at(-1).categoria_clave, "administrativo");
  assert.equal(llamadas.at(-1).q, "");
  assert.ok(r.querySelector("[data-personal-rpt-publica-quitar-filtro]"));
});

test("resumen de centros abre agrupación paginada y el centro abre sus puestos", async () => {
  const r = raiz(), llamadas = [];
  const centro = { codigo: "101", denominacion: "CENTRO 101", puestos: 2, dotacion: 5 };
  await montarModuloRPTPublica({ raiz: r, cliente: { async listar(consulta) { llamadas.push(consulta); return pagina({ vista: consulta.vista, total: consulta.vista === "centros" ? 1 : 0, items: consulta.vista === "centros" ? [centro] : [] }); } } });
  nodosCon(r, "personalRptPublicaResumenEnlace").find((b) => b.dataset.personalRptPublicaResumenEnlace === "centros").listeners.get("click")();
  await Promise.resolve();
  assert.equal(llamadas.at(-1).vista, "centros");
  nodosCon(r, "personalRptPublicaEnlace").find((b) => b.textContent === "2").listeners.get("click")();
  await Promise.resolve();
  assert.equal(llamadas.at(-1).vista, "puestos"); assert.equal(llamadas.at(-1).centro_codigo, "101");
});

test("URL de RPT conserva idioma y restaura filtros exactos", async () => {
  const anterior = globalThis.window, llamadas = [];
  const location = { pathname: "/portal-empleado/", search: "?lang=es&rpt_vista=puestos&rpt_categoria=administrativo&rpt_offset=25", hash: "#personal" };
  globalThis.window = { location, history: { replaceState(_a, _b, ruta) { const u = new URL(ruta, "http://vec.local"); location.search = u.search; location.hash = u.hash; } } };
  try {
    const r = raiz();
    const modulo = await montarModuloRPTPublica({ raiz: r, cliente: { async listar(consulta) { llamadas.push(consulta); return pagina({ vista: consulta.vista, total: 0, offset: consulta.offset, items: [] }); } } });
    assert.equal(llamadas[0].categoria_clave, "administrativo"); assert.equal(llamadas[0].offset, 25);
    assert.match(location.search, /lang=es/u); assert.match(location.search, /rpt_categoria=administrativo/u);
    modulo.desmontar();
  } finally { globalThis.window = anterior; }
});

test("doble pulsación del mismo filtro comparte GET y enfoca el error", async () => {
  const r = raiz(), pendientes = [];
  const categoria = { clave: "administrativo", denominacion: "ADMINISTRATIVO", grupos: ["C1"], escalas: ["AG"], puestos: 1, dotacion: 1,
    puestos_vinculados: 1, dotacion_vinculada: 1, recuento_coincide: true };
  await montarModuloRPTPublica({ raiz: r, cliente: { listar(consulta, { signal }) {
    if (consulta.vista === "categorias") return Promise.resolve(pagina({ items: [categoria] }));
    return new Promise((resolve, reject) => pendientes.push({ resolve, reject, signal }));
  } } });
  const boton = nodosCon(r, "personalRptPublicaEnlace")[0];
  boton.listeners.get("click")(); boton.listeners.get("click")();
  assert.equal(pendientes.length, 1); assert.equal(pendientes[0].signal.aborted, false);
  assert.equal(r.ownerDocument.activeElement, r.querySelector("[data-personal-rpt-publica-estado]"));
  pendientes[0].reject(new Error("fuente no disponible")); await esperarRespuesta();
  const error = r.querySelector("[data-personal-rpt-publica-estado]");
  const reintentar = r.querySelector("[data-personal-rpt-publica-reintentar]");
  assert.equal(error.atributos.get("role"), "alert"); assert.equal(r.ownerDocument.activeElement, reintentar);
});

test("cambiar de agrupación cancela el GET anterior y pinta solo la última", async () => {
  const r = raiz(), pendientes = [];
  const categoria = { clave: "administrativo", denominacion: "ADMINISTRATIVO", grupos: ["C1"], escalas: ["AG"], puestos: 1, dotacion: 1,
    puestos_vinculados: 1, dotacion_vinculada: 1, recuento_coincide: true };
  await montarModuloRPTPublica({ raiz: r, cliente: { listar(consulta, { signal }) {
    if (consulta.vista === "categorias") return Promise.resolve(pagina({ items: [categoria] }));
    return new Promise((resolve) => pendientes.push({ resolve, signal, vista: consulta.vista }));
  } } });
  nodosCon(r, "personalRptPublicaEnlace")[0].listeners.get("click")();
  nodosCon(r, "personalRptPublicaVista").find((b) => b.dataset.personalRptPublicaVista === "centros").listeners.get("click")();
  assert.deepEqual(pendientes.map((p) => p.vista), ["puestos", "centros"]);
  assert.equal(pendientes[0].signal.aborted, true);
  pendientes[0].resolve(pagina({ vista: "puestos", total: 0, items: [] })); await esperarRespuesta();
  assert.equal(r.querySelector("[data-personal-rpt-publica-tabla]"), null);
  pendientes[1].resolve(pagina({ vista: "centros", total: 0, items: [] })); await esperarRespuesta();
  assert.match(textoNodo(r.querySelector("[data-personal-rpt-publica-tabla]")), /Tabla de centros RPT/u);
});

test("enlace con centro inválido limpia el filtro y permite volver a buscar", async () => {
  const anterior = globalThis.window;
  for (const centro of ["%20", "101%0A", "101%01"]) {
    const location = { pathname: "/portal-empleado/", search: `?lang=es&rpt_vista=puestos&rpt_centro=${centro}`, hash: "#personal" };
    globalThis.window = { location, history: { replaceState(_a, _b, ruta) { const u = new URL(ruta, "http://vec.local"); location.search = u.search; location.hash = u.hash; } } };
    try {
      const r = raiz(), llamadas = [];
      const modulo = await montarModuloRPTPublica({ raiz: r, cliente: { async listar(consulta) { llamadas.push(consulta); return pagina({ vista: consulta.vista, total: 0, items: [] }); } } });
      assert.deepEqual(llamadas[0], { vista: "categorias", q: "", limit: 25, offset: 0, categoria_clave: "", centro_codigo: "", codigo_puesto: "" });
      assert.equal(location.search, "?lang=es");
      assert.match(textoNodo(r), /enlace tenía un filtro no válido/u);
      nodosCon(r, "personalRptPublicaResumenEnlace").find((b) => b.dataset.personalRptPublicaResumenEnlace === "centros").listeners.get("click")();
      await esperarRespuesta();
      assert.equal(llamadas.at(-1).vista, "centros"); assert.equal(llamadas.at(-1).centro_codigo, "");
      assert.doesNotMatch(textoNodo(r), /enlace tenía un filtro no válido/u);
      assert.match(location.search, /rpt_vista=centros/u);
      modulo.desmontar();
    } finally { globalThis.window = anterior; }
  }
});

test("la vista importa sus contratos RPT con la versión propia vigente", () => {
  const fuente = readFileSync(new URL("./vista-rpt-publica.js", import.meta.url), "utf8");
  assert.match(fuente, /i18n-rpt-puestos\.js\?v=20261007-t-rpt-enlaces-v1/u);
  assert.match(fuente, /cliente-http-rpt-publica\.js\?v=20261007-t-rpt-enlaces-v1/u);
});

test("la región RPT declara el idioma efectivo de sus textos", async () => {
  const r = raiz();
  await montarModuloRPTPublica({ raiz: r, cliente: { async listar() { return pagina(); } } });
  assert.equal(r.querySelector("[data-personal-rpt-publica]").lang, IDIOMA_EFECTIVO_RPT_PUESTOS);
});

test("cada categoría tiene una acción de teclado y dos cifras al mismo destino", async () => {
  const r = raiz(), llamadas = [];
  const categoria = { clave: "administrativo", denominacion: "ADMINISTRATIVO", grupos: ["C1"], escalas: ["AG"], puestos: 57, dotacion: 158,
    puestos_vinculados: 62, dotacion_vinculada: 163, recuento_coincide: false };
  await montarModuloRPTPublica({ raiz: r, cliente: { async listar(q) { llamadas.push(q); return pagina({ vista: q.vista, total: q.vista === "categorias" ? 1 : 0, items: q.vista === "categorias" ? [categoria] : [] }); } } });
  const enlaces = nodosCon(r, "personalRptPublicaEnlace");
  assert.equal(enlaces.length, 3);
  assert.deepEqual(enlaces.filter((b) => b.atributos.get("tabindex") !== "-1").map((b) => b.textContent), ["ADMINISTRATIVO"]);
  assert.match(enlaces[0].atributos.get("aria-label"), /ADMINISTRATIVO: 62 puestos y 163 dotaciones/u);
  assert.deepEqual(enlaces.slice(1).map((b) => b.textContent), ["62", "163"]);
  enlaces[2].listeners.get("click")(); await esperarRespuesta();
  assert.equal(llamadas.at(-1).categoria_clave, "administrativo");
});

test("Reintentar consulta mantiene el filtro y comparte la petición repetida", async () => {
  const r = raiz(), llamadas = [], pendientes = [];
  const modulo = await montarModuloRPTPublica({ raiz: r, cliente: { listar(q, { signal }) {
    llamadas.push(q);
    if (llamadas.length === 1) return Promise.reject(new Error("503"));
    return new Promise((resolve) => pendientes.push({ resolve, signal }));
  } } });
  const reintentar = r.querySelector("[data-personal-rpt-publica-reintentar]");
  assert.equal(r.ownerDocument.activeElement, reintentar);
  reintentar.listeners.get("click")(); reintentar.listeners.get("click")();
  assert.equal(llamadas.length, 2); assert.equal(pendientes[0].signal.aborted, false);
  pendientes[0].resolve(pagina({ total: 0, items: [] })); await esperarRespuesta();
  assert.ok(r.querySelector("[data-personal-rpt-publica-tabla]"));
  modulo.desmontar();
});

test("Atrás restaura filtros y página sin crear entradas nuevas", async () => {
  const anterior = globalThis.window, oyentes = new Map();
  const location = { pathname: "/portal-empleado/", search: "?lang=es", hash: "#personal" };
  const rutas = ["/portal-empleado/?lang=es#personal"]; let indice = 0;
  const aplicar = (ruta) => { const u = new URL(ruta, "http://vec.local"); location.search = u.search; location.hash = u.hash; };
  globalThis.window = { location,
    addEventListener(tipo, fn) { oyentes.set(tipo, fn); }, removeEventListener(tipo, fn) { if (oyentes.get(tipo) === fn) oyentes.delete(tipo); },
    history: {
      replaceState(_a, _b, ruta) { rutas[indice] = ruta; aplicar(ruta); },
      pushState(_a, _b, ruta) { rutas.splice(++indice); rutas.push(ruta); aplicar(ruta); },
    },
  };
  try {
    const r = raiz(), llamadas = [];
    const categoria = { clave: "administrativo", denominacion: "ADMINISTRATIVO", grupos: ["C1"], escalas: ["AG"], puestos: 1, dotacion: 1,
      puestos_vinculados: 1, dotacion_vinculada: 1, recuento_coincide: true };
    const modulo = await montarModuloRPTPublica({ raiz: r, cliente: { async listar(q) { llamadas.push(q); return pagina({ vista: q.vista, total: q.vista === "categorias" ? 1 : 0,
      items: q.vista === "categorias" ? [categoria] : [] }); } } });
    assert.equal(rutas.length, 1);
    nodosCon(r, "personalRptPublicaEnlace")[0].listeners.get("click")(); await esperarRespuesta();
    assert.equal(rutas.length, 2); assert.match(rutas[1], /rpt_categoria=administrativo/u);
    nodosCon(r, "personalRptPublicaResumenEnlace").find((b) => b.dataset.personalRptPublicaResumenEnlace === "centros").listeners.get("click")(); await esperarRespuesta();
    assert.equal(rutas.length, 3); assert.match(rutas[2], /rpt_vista=centros/u);
    aplicar(rutas[--indice]); oyentes.get("popstate")(); await esperarRespuesta();
    assert.equal(rutas.length, 3); assert.equal(llamadas.at(-1).categoria_clave, "administrativo");
    aplicar(rutas[--indice]); oyentes.get("popstate")(); await esperarRespuesta();
    assert.equal(rutas.length, 3); assert.equal(llamadas.at(-1).vista, "categorias");
    assert.match(location.search, /lang=es/u);
    modulo.desmontar(); assert.equal(oyentes.has("popstate"), false);
  } finally { globalThis.window = anterior; }
});

test("fila de una plaza anuncia un puesto y una dotación", async () => {
  const r = raiz();
  const categoria = { clave: "auxiliar", denominacion: "AUXILIAR", grupos: ["C2"], escalas: [], puestos: 1, dotacion: 1,
    puestos_vinculados: 1, dotacion_vinculada: 1, recuento_coincide: true };
  await montarModuloRPTPublica({ raiz: r, cliente: { async listar() { return pagina({ items: [categoria] }); } } });
  const principal = nodosCon(r, "personalRptPublicaEnlace").find((b) => b.textContent === "AUXILIAR");
  assert.match(principal.atributos.get("aria-label"), /1 puesto y 1 dotación/u);
});

test("enlace por código conserva filtros exactos y permite quitar sólo ese código", async () => {
  const anterior = globalThis.window, llamadas = [];
  const location = { pathname: "/portal-empleado/", search: "?lang=es&rpt_vista=puestos&rpt_q=admin&rpt_categoria=administrativo&rpt_centro=101&rpt_codigo=217", hash: "#personal" };
  const aplicar = (ruta) => { const u = new URL(ruta, "http://vec.local"); location.search = u.search; location.hash = u.hash; };
  globalThis.window = { location, history: { replaceState(_a, _b, ruta) { aplicar(ruta); }, pushState(_a, _b, ruta) { aplicar(ruta); } } };
  try {
    const r = raiz();
    const modulo = await montarModuloRPTPublica({ raiz: r, cliente: { async listar(q) { llamadas.push(q); return pagina({ vista: q.vista, total: 0, items: [] }); } } });
    assert.equal(llamadas[0].codigo_puesto, "217");
    assert.equal(llamadas[0].categoria_clave, "administrativo"); assert.equal(llamadas[0].centro_codigo, "101"); assert.equal(llamadas[0].q, "admin");
    assert.match(textoNodo(r), /Puesto: 217/u);
    const quitar = nodosCon(r, "personalRptPublicaQuitarFiltro").find((n) => n.dataset.personalRptPublicaQuitarFiltro === "codigo_puesto");
    quitar.listeners.get("click")(); await esperarRespuesta();
    assert.equal(llamadas.at(-1).codigo_puesto, ""); assert.equal(llamadas.at(-1).categoria_clave, "administrativo");
    assert.equal(llamadas.at(-1).centro_codigo, "101"); assert.equal(llamadas.at(-1).q, "admin");
    assert.doesNotMatch(location.search, /rpt_codigo/u); assert.match(location.search, /rpt_categoria=administrativo/u);
    assert.match(location.search, /lang=es/u); assert.equal(location.hash, "#personal");
    modulo.desmontar();
  } finally { globalThis.window = anterior; }
});

test("código no pasa a otra vista o tarjeta y un enlace inválido vuelve al catálogo", async () => {
  const anterior = globalThis.window;
  for (const accion of ["pestana", "tarjeta"]) {
    const llamadas = [], location = { pathname: "/portal-empleado/", search: "?lang=es&rpt_vista=puestos&rpt_codigo=217", hash: "#personal" };
    globalThis.window = { location, history: { replaceState(_a, _b, ruta) { location.search = new URL(ruta, "http://vec.local").search; }, pushState(_a, _b, ruta) { location.search = new URL(ruta, "http://vec.local").search; } } };
    try {
      const r = raiz();
      const modulo = await montarModuloRPTPublica({ raiz: r, cliente: { async listar(q) { llamadas.push(q); return pagina({ vista: q.vista, total: 0, items: [] }); } } });
      const boton = accion === "pestana" ? nodosCon(r, "personalRptPublicaVista").find((n) => n.dataset.personalRptPublicaVista === "centros")
        : nodosCon(r, "personalRptPublicaResumenEnlace").find((n) => n.dataset.personalRptPublicaResumenEnlace === "categorias");
      boton.listeners.get("click")(); await esperarRespuesta();
      assert.equal(llamadas.at(-1).codigo_puesto, ""); assert.doesNotMatch(location.search, /rpt_codigo/u);
      modulo.desmontar();
    } finally { globalThis.window = anterior; }
  }
  for (const codigo of ["217a", "217&rpt_codigo=1217", ""]) {
    const location = { pathname: "/portal-empleado/", search: `?lang=es&rpt_vista=puestos&rpt_codigo=${codigo}`, hash: "#personal" };
    globalThis.window = { location, history: { replaceState(_a, _b, ruta) { location.search = new URL(ruta, "http://vec.local").search; } } };
    try {
      const r = raiz(), llamadas = [];
      const modulo = await montarModuloRPTPublica({ raiz: r, cliente: { async listar(q) { llamadas.push(q); return pagina({ vista: q.vista, total: 0, items: [] }); } } });
      assert.equal(llamadas[0].vista, "categorias"); assert.equal(llamadas[0].codigo_puesto, "");
      assert.match(textoNodo(r), /enlace tenía un filtro no válido/u);
      assert.doesNotMatch(location.search, /rpt_codigo/u);
      modulo.desmontar();
    } finally { globalThis.window = anterior; }
  }
});
