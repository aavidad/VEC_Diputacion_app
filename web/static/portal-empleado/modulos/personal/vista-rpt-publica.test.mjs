import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { montarModuloRPTPublica } from "./vista-rpt-publica.js";

function raiz() { class Nodo { constructor(documento, etiqueta = "div") { this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {}; this.listeners = new Map(); this.parent = null; this.textContent = ""; this.atributos = new Map(); this.focused = false; } append(...nodos) { this.children.push(...nodos); nodos.forEach((n) => { n.parent = this; }); } replaceChildren(...nodos) { this.children.forEach((n) => { n.parent = null; }); this.children = []; this.append(...nodos); } removeChild(n) { this.children = this.children.filter((h) => h !== n); n.parent = null; } remove() { this.parent?.removeChild(this); } addEventListener(t, h) { this.listeners.set(t, h); } setAttribute(k, v) { this.atributos.set(k, v); } contains(n) { for (let actual = n; actual; actual = actual.parent) { if (actual === this) return true; } return false; } focus() { let actual = this; while (actual) { if (actual === this.ownerDocument.raiz) { this.focused = true; this.ownerDocument.activeElement = this; return; } actual = actual.parent; } } matches(s) { const k = s.match(/^\[data-([a-z-]+)\]$/u)?.[1]?.replace(/-([a-z])/g, (_m, l) => l.toUpperCase()); return k ? this.dataset[k] !== undefined : false; } querySelector(s) { if (this.matches(s)) return this; for (const h of this.children) { const e = h.querySelector(s); if (e) return e; } return null; } } const d = { createElement: (e) => new Nodo(d, e) }; const salida = new Nodo(d, "root"); d.raiz = salida; return salida; }
function textoNodo(nodo) { return `${nodo.textContent} ${nodo.children.map(textoNodo).join(" ")}`; }
function nodosCon(nodo, clave, salida = []) { if (nodo.dataset[clave] !== undefined) salida.push(nodo); nodo.children.forEach((hijo) => nodosCon(hijo, clave, salida)); return salida; }
function propagarTecla(origen, key) { const evento = { key, cancelado: false, preventDefault() { this.cancelado = true; } }; for (let actual = origen; actual; actual = actual.parent) actual.listeners.get("keydown")?.(evento); return evento; }
function pagina({ vista = "categorias", total = 1, offset = 0, generadoEn = "2026-09-17", items = [{ clave: "administrativo", denominacion: "ADMINISTRATIVO", grupos: ["C1"], escalas: ["AG"], puestos: 57, dotacion: 158 }] } = {}) { return Object.freeze({ items: Object.freeze(items), total, limit: 25, offset, vista, esquema: "vec.catalogo.rpt.v1", fuente: Object.freeze({ documento: "RPT publicada", importacion: "rpt-v1", generado_en: generadoEn, aviso: "Datos públicos sin ocupantes.", huella_sha256: "a".repeat(64) }), resumen: Object.freeze({ puestos: 842, dotacion: 1714, categorias: 145, centros: 41 }) }); }
test("catálogos ES/EN rotulan centro y generación sin atribuir vigencia", () => {
  const catalogo = (idioma) => JSON.parse(readFileSync(new URL(`../../../textos/${idioma}/personal.json`, import.meta.url), "utf8")).rpt_puestos;
  const es = catalogo("es"), en = catalogo("en");
  assert.deepEqual(Object.keys(es).sort(), Object.keys(en).sort());
  assert.equal(es.centro_codigo, "Código del centro");
  assert.match(es.generacion, /Generación de la proyección: \{valor\}.*no acredita la vigencia/u);
  assert.match(en.generacion, /generation: \{valor\}.*does not establish/u);
});
test("vista RPT separa resumen y fuente, deja ayuda y huella tras ? y conserva la consulta", async () => { const r = raiz(); const llamadas = []; const modulo = await montarModuloRPTPublica({ raiz: r, cliente: { async listar(consulta) { llamadas.push(consulta); return pagina({ total: 26, vista: consulta.vista }); } } }); const contenedor = r.querySelector("[data-personal-rpt-publica]"); assert.ok(contenedor); const resumen = contenedor.querySelector("[data-personal-rpt-publica-resumen]"); assert.match(textoNodo(resumen), /842 puestos/); assert.doesNotMatch(textoNodo(resumen), /Fuente:|ocupantes/); assert.doesNotMatch(textoNodo(resumen), new RegExp("a".repeat(64))); assert.doesNotMatch(textoNodo(contenedor.children[0]), /Portal del Empleado/); const ayuda = contenedor.querySelector("[data-personal-rpt-publica-ayuda]"); const abrirAyuda = contenedor.querySelector("[data-personal-rpt-publica-abrir-ayuda]"); assert.equal(ayuda.hidden, true); assert.equal(abrirAyuda.tagName, "button"); assert.equal(abrirAyuda.textContent, "?"); assert.equal(abrirAyuda.atributos.get("aria-controls"), ayuda.id); assert.equal(abrirAyuda.atributos.get("aria-expanded"), "false"); assert.match(abrirAyuda.atributos.get("aria-label"), /Consulta pública/); assert.match(textoNodo(ayuda), new RegExp("a".repeat(64))); assert.match(textoNodo(ayuda), /Fuente: RPT publicada\. Datos públicos sin ocupantes\./); assert.match(textoNodo(ayuda), /Generación de la proyección: 17 de septiembre de 2026/); assert.match(textoNodo(ayuda), /no acredita la vigencia administrativa/); assert.doesNotMatch(textoNodo(ayuda), /Ejercicio:/); abrirAyuda.focus(); abrirAyuda.listeners.get("click")(); assert.equal(ayuda.hidden, false); assert.equal(abrirAyuda.atributos.get("aria-expanded"), "true"); assert.equal(abrirAyuda.focused, true); const escape = propagarTecla(abrirAyuda, "Escape"); assert.equal(escape.cancelado, true); assert.equal(ayuda.hidden, true); assert.equal(abrirAyuda.atributos.get("aria-expanded"), "false"); assert.equal(abrirAyuda.focused, true); const tabla = contenedor.children.find((n) => n.className === "tabla-contenedor"); assert.equal(tabla.atributos.get("tabindex"), "0"); assert.equal(tabla.atributos.get("role"), "region"); assert.equal(tabla.atributos.get("aria-label"), "Tabla de categorías RPT"); const puestos = contenedor.querySelector("[data-personal-rpt-publica-vista]"); assert.ok(puestos); const siguiente = contenedor.querySelector("[data-personal-rpt-publica-siguiente]"); siguiente.listeners.get("click")(); await Promise.resolve(); assert.equal(llamadas[1].offset, 25); assert.equal(llamadas[0].vista, "categorias"); modulo.desmontar(); });
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
  assert.deepEqual(llamadas.at(-1), { vista: "puestos", q: "administrativo", limit: 25, offset: 25 });
  assert.equal(r.querySelector("[data-personal-rpt-publica-busqueda]").value, "sin enviar");
  const quitar = r.querySelector("[data-personal-rpt-publica-quitar-busqueda]"); quitar.focus(); quitar.listeners.get("click")();
  await esperarRespuesta();
  assert.deepEqual(llamadas.at(-1), { vista: "puestos", q: "", limit: 25, offset: 0 });
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

test("vista RPT conserva el código exacto y permite quitarlo sin perder la búsqueda", async () => {
  const r = raiz(), llamadas = [];
  const puesto = { codigo: "217", denominacion: "TÉCNICO", centro_codigo: "101", centro: "CENTRO",
    delegacion: "PRESIDENCIA", grupos: [], escala: "", categoria_clave: "", nivel_destino: 17,
    complemento_especifico_anual_centimos: 0, dotacion: 1, tipo: "E", provision: "I" };
  await montarModuloRPTPublica({ raiz: r, codigoPuesto: "217", cliente: { async listar(consulta) {
    llamadas.push(consulta);
    return pagina({ vista: consulta.vista, total: consulta.vista === "puestos" ? 1 : 0,
      items: consulta.vista === "puestos" ? [puesto] : [] });
  } } });
  assert.equal(llamadas[0].codigo_puesto, "217");
  assert.equal(llamadas[0].vista, "puestos");
  assert.match(textoNodo(r.querySelector("[data-personal-rpt-publica-codigo-aplicado]")), /Puesto: 217/u);
  r.querySelector("[data-personal-rpt-publica-quitar-codigo]").listeners.get("click")();
  await esperarRespuesta();
  assert.equal(llamadas.at(-1).codigo_puesto, "");
  assert.equal(llamadas.at(-1).vista, "puestos");
  assert.equal(r.querySelector("[data-personal-rpt-publica-codigo-aplicado]"), null);
  await assert.rejects(() => montarModuloRPTPublica({ raiz: raiz(), codigoPuesto: "217 ",
    cliente: { listar: async () => pagina() } }), TypeError);
});
