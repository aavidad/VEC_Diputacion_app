import "./test-preparar-i18n.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { montarVistaBandejaCircuitoDietas } from "./vista-bandeja-circuito.js?v=20261001-ct-a-i18n-v1";
import { cargarTextos } from "../../../comun/textos.js";

const dato = (atributo) => atributo.slice(5).replace(/-([a-z])/g, (_m, letra) => letra.toUpperCase());
class Nodo {
  constructor(documento, etiqueta = "div") { this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {}; this.attrs = {}; this.listeners = {}; this.parent = null; this.textContent = ""; this.value = ""; this.disabled = false; }
  append(...nodos) { this.children.push(...nodos); nodos.forEach((nodo) => { nodo.parent = this; }); }
  replaceChildren(...nodos) { this.children = []; this.append(...nodos); }
  remove() { this.parent?.removeChild(this); }
  removeChild(nodo) { this.children = this.children.filter((hijo) => hijo !== nodo); nodo.parent = null; }
  addEventListener(tipo, listener) { this.listeners[tipo] = listener; } removeEventListener(tipo) { delete this.listeners[tipo]; }
  setAttribute(nombre, valor) { this.attrs[nombre] = String(valor); if (nombre === "name") this.name = String(valor); }
  focus() { this.ownerDocument.activeElement = this; }
  matches(selector) { const m = selector.match(/^\[([^=\]]+)(?:="([^"]*)")?\]$/u); if (!m) return this.tagName === selector; const clave = m[1] === "name" ? "name" : dato(m[1]); const valor = m[1] === "name" ? this.name : this.dataset[clave]; return valor !== undefined && (m[2] === undefined || valor === m[2]); }
  closest(selector) { for (let actual = this; actual; actual = actual.parent) if (actual.matches(selector)) return actual; return null; }
  contains(nodo) { return nodo === this || this.children.some((hijo) => hijo.contains(nodo)); }
  querySelectorAll(selector) { const salida = []; const visitar = (actual) => { if (actual.matches(selector)) salida.push(actual); actual.children.forEach(visitar); }; visitar(this); return salida; }
  querySelector(selector) { return this.querySelectorAll(selector)[0] || null; }
}
function raiz() { const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta) }; return new Nodo(documento, "root"); }
const referencia = "dco_1234567890123456789012";
const fila = { referencia, estado: "enviado_pendiente_revision", version: 3, fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21" };
const documentoLeido = {
  referencia, numero_documento: "VEC-D-2026-000012", fecha_apertura: "2026-09-20T08:00:00.000000Z", estado: "enviado_pendiente_revision", version: 3,
  fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21", hora_inicio: "08:00", hora_fin: "15:30", motivo: "Reunión técnica", codigos_ruta: [], calculo: {},
  documento: { lineas: [
    { tipo: "dieta", concepto: "manutencion", fecha: "2026-09-20", importe_centimos: 2667 },
    { tipo: "kilometraje", ruta_indice: 1, kilometros: "42.5000", importe_centimos: 1105 },
    { tipo: "otro_gasto", concepto: "Aparcamiento", importe_centimos: 650, justificante_ref: "just:ticket-01", justificante_sha256: "a".repeat(64) },
    { tipo: "otro_medio", tipo_gasto: "taxi", catalogo_version: "provisional:otros-gastos:20260925", fecha: "2026-09-21", concepto: "Estación a sede", importe_centimos: 1250, justificante_ref: "ticket:taxi-02", justificante_sha256: "b".repeat(64) },
  ], manutencion_centimos: 2667, alojamiento_tope_centimos: 0, kilometraje_centimos: 1105, otros_centimos: 1900, total_orientativo_centimos: 5672 },
};
const recibo = { referencia: "rcd_1234567890123456789012", version: 4, registrado_en: "2026-09-24T10:00:00.000000Z", repeticion: false };
function texto(nodo) { return [nodo.textContent, ...nodo.children.map(texto)].join(" "); }
const esperar = async () => { await Promise.resolve(); await Promise.resolve(); };

test("sin fuente de competencia lo dice en una línea y no ofrece acciones", async () => {
  const contenedor = raiz();
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { cliente: { listar: async () => ({ items: [], competencia: "sin_fuente" }), decidir: async () => assert.fail("POST inesperado") } });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
  assert.match(panel.querySelector("[data-dietas-circuito-sin-fuente]").textContent, /validadores/u);
  assert.equal(panel.querySelector("[data-dietas-circuito-abrir]"), null);
  assert.equal(panel.querySelector("[data-dietas-circuito-decision]"), null);
  vista.desmontar();
});

test("abre el documento con sus líneas y total, exige motivo al devolver y muestra el recibo solo tras la respuesta", async () => {
  const contenedor = raiz(); const llamadas = []; const lecturas = [];
  const cliente = {
    listar: async () => ({ items: [fila], competencia: "acreditada" }),
    documento: async (ref, etapa) => { lecturas.push([ref, etapa]); return documentoLeido; },
    decidir: async (ref, entrada) => { llamadas.push([ref, entrada]); return { comision: { referencia, estado: "devuelta", version: 4 }, recibo }; },
  };
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { cliente, generarClaveIdempotencia: () => "decision-circuito-0001" });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
  assert.equal(panel.querySelector("h2").textContent, "Pendientes de revisar");
  await panel.listeners.click({ target: panel.querySelector("[data-dietas-circuito-abrir]") });
  assert.deepEqual(lecturas, [[referencia, "revision"]]);
  const detalle = panel.querySelector("[data-dietas-circuito-detalle]");
  assert.equal(detalle.hidden, false);
  assert.equal(contenedor.ownerDocument.activeElement, panel.querySelector("[data-dietas-circuito-volver]"));
  assert.equal(panel.querySelectorAll("[data-dietas-circuito-linea]").length, 4);
  assert.match(texto(detalle), /Justificante just:ticket-01/u);
  // La línea D5 muestra su tipo y fecha con rótulos de negocio, sin códigos.
  assert.match(texto(detalle), /Taxi · 21 sept 2026 · Estación a sede/u);
  assert.doesNotMatch(texto(detalle), /otros-gastos|tipo_gasto/u);
  assert.match(texto(panel.querySelector("[data-dietas-circuito-total]")), /56,72/u);
  assert.equal(panel.querySelector('[data-dietas-circuito-decision="aprobar"]').textContent, "Elevar al responsable");
  await panel.listeners.click({ target: panel.querySelector('[data-dietas-circuito-decision="devolver"]') });
  assert.equal(llamadas.length, 0); assert.match(panel.querySelector("[data-dietas-circuito-estado]").textContent, /motivo/u);
  panel.querySelector("[data-dietas-circuito-motivo]").value = "  Falta el ticket del aparcamiento ";
  await panel.listeners.click({ target: panel.querySelector('[data-dietas-circuito-decision="devolver"]') });
  assert.deepEqual(llamadas[0], [referencia, { etapa: "revision", decision: "devolver", motivo: "Falta el ticket del aparcamiento", clave_idempotencia: "decision-circuito-0001", version_esperada: 3 }]);
  assert.match(panel.querySelector("[data-dietas-circuito-estado]").textContent, /Recibo rcd_1234567890123456789012/u);
  assert.equal(panel.querySelector("[data-dietas-circuito-detalle]").hidden, true);
  assert.equal(panel.querySelector("[data-dietas-circuito-abrir]"), null);
  vista.desmontar();
});

test("un resultado incierto conserva clave y decisión para repetir exactamente la misma acción", async () => {
  const contenedor = raiz(); const entradas = []; let intento = 0;
  const cliente = {
    listar: async () => ({ items: [fila], competencia: "acreditada" }), documento: async () => documentoLeido,
    decidir: async (_ref, entrada) => { entradas.push(entrada); if (!intento++) { const error = new Error(); error.resultadoIndeterminado = true; throw error; } return { comision: { referencia, estado: "pendiente_autorizacion", version: 4 }, recibo: { ...recibo, repeticion: true } }; },
  };
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { cliente, generarClaveIdempotencia: () => "decision-circuito-0001" });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
  await panel.listeners.click({ target: panel.querySelector("[data-dietas-circuito-abrir]") });
  await panel.listeners.click({ target: panel.querySelector('[data-dietas-circuito-decision="aprobar"]') });
  assert.match(panel.querySelector("[data-dietas-circuito-estado]").textContent, /confirmado/u);
  assert.equal(panel.querySelector('[data-dietas-circuito-decision="aprobar"]').disabled, true);
  await panel.listeners.click({ target: panel.querySelector("[data-dietas-circuito-reintento]") });
  assert.equal(entradas.length, 2); assert.deepEqual(entradas[1], entradas[0]);
  assert.match(panel.querySelector("[data-dietas-circuito-estado]").textContent, /Ya estaba registrado/u);
  vista.desmontar();
});

test("el control de documentos busca por etapa y fechas acreditadas", async () => {
  const contenedor = raiz(); const consultas = [];
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { control: true, etapas: ["revision", "liquidacion"], etapaInicial: "revision",
    cliente: { listar: async (consulta) => { consultas.push(consulta); return { items: [], competencia: "acreditada" }; }, decidir: async () => assert.fail("POST inesperado") } });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
  assert.equal(panel.querySelector("h2").textContent, "Control de documentos");
  const filtros = panel.querySelector("[data-dietas-circuito-filtros]");
  assert.equal(filtros.hidden, false);
  panel.querySelector("[data-dietas-circuito-etapa]").value = "liquidacion";
  panel.querySelector("[data-dietas-circuito-desde]").value = "2026-09-01";
  panel.querySelector("[data-dietas-circuito-hasta]").value = "2026-09-30";
  await filtros.listeners.submit({ preventDefault() {} });
  assert.deepEqual(consultas.at(-1), { etapa: "liquidacion", limit: 20, fecha_desde: "2026-09-01", fecha_hasta: "2026-09-30" });
  assert.match(texto(panel), /No hay documentos para estas fechas/u);
  assert.throws(() => montarVistaBandejaCircuitoDietas(raiz(), { cliente: { listar() {}, decidir() {} }, etapas: ["otra"] }), TypeError);
  vista.desmontar();
});

test("desmontar aborta la lectura y descarta su respuesta tardía", async () => {
  const contenedor = raiz(); let signal; let resolver;
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { cliente: { listar: (_consulta, opciones) => { signal = opciones.signal; return new Promise((resolve) => { resolver = resolve; }); }, decidir: async () => assert.fail("POST inesperado") } });
  await esperar(); vista.desmontar(); assert.equal(signal.aborted, true); resolver({ items: [fila], competencia: "acreditada" }); await esperar(); assert.equal(contenedor.children.length, 0);
});

test("un reenvío muestra etapa, fecha y motivo de la devolución anterior, sin quién la hizo", async () => {
  const contenedor = raiz();
  const reenviado = { ...documentoLeido, version: 5, devolucion: { etapa: "autorizacion", motivo: "Falta el justificante del taxi", version: 3, devuelta_en: "2026-09-19T09:00:00.123456Z" } };
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { cliente: { listar: async () => ({ items: [fila], competencia: "acreditada" }),
    documento: async () => reenviado, decidir: async () => assert.fail("POST inesperado") } });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
  await panel.listeners.click({ target: panel.querySelector("[data-dietas-circuito-abrir]") });
  const etapa = panel.querySelector('[data-dietas-circuito-reenvio="etapa"]');
  assert.match(texto(etapa), /Reenvío.*Devuelto en Autorización el 19 sept 2026/u);
  assert.match(texto(panel.querySelector('[data-dietas-circuito-reenvio="motivo"]')), /Motivo de la devolución.*Falta el justificante del taxi/u);
  assert.doesNotMatch(texto(panel), /per_|act_|actor/u);
  vista.desmontar();
  const sinReenvio = raiz();
  const otra = montarVistaBandejaCircuitoDietas(sinReenvio, { cliente: { listar: async () => ({ items: [fila], competencia: "acreditada" }), documento: async () => documentoLeido, decidir: async () => assert.fail("POST inesperado") } });
  await esperar(); const panelDos = sinReenvio.querySelector("[data-dietas-bandeja-circuito]");
  await panelDos.listeners.click({ target: panelDos.querySelector("[data-dietas-circuito-abrir]") });
  assert.equal(panelDos.querySelector("[data-dietas-circuito-reenvio]"), null);
  otra.desmontar();
});

test("el motivo de devolución se recorta con el mismo conjunto de blancos que rechaza Go", async () => {
  const contenedor = raiz(); const llamadas = [];
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { generarClaveIdempotencia: () => "decision-circuito-0002", cliente: {
    listar: async () => ({ items: [fila], competencia: "acreditada" }), documento: async () => documentoLeido,
    decidir: async (ref, entrada) => { llamadas.push(entrada); return { comision: { referencia, estado: "devuelta", version: 4 }, recibo }; } } });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
  await panel.listeners.click({ target: panel.querySelector("[data-dietas-circuito-abrir]") });
  panel.querySelector("[data-dietas-circuito-motivo]").value = "\ufeff\u0085 Falta el ticket\u00a0\u0085";
  await panel.listeners.click({ target: panel.querySelector('[data-dietas-circuito-decision="devolver"]') });
  assert.equal(llamadas[0].motivo, "Falta el ticket");
  vista.desmontar();
});

const filaPagina = (numero) => ({ ...fila, referencia: `dco_${String(numero).padStart(22, "0")}` });
const clic = (panel, atributo) => panel.listeners.click({ target: panel.querySelector(`[data-dietas-circuito-${atributo}]`) });
const aplazada = () => { let resolve; let reject; const promise = new Promise((si, no) => { resolve = si; reject = no; }); return { promise, resolve, reject }; };

test("avanza sin acumular documentos, vuelve consultando el cursor anterior y permite regresar a la primera página", async () => {
  const contenedor = raiz(); const consultas = [];
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { cliente: {
    listar: async (consulta) => {
      consultas.push(consulta);
      const numero = consulta.cursor === "pagina-3" ? 3 : consulta.cursor === "pagina-2" ? 2 : 1;
      return { items: [filaPagina(numero)], competencia: "acreditada", ...(numero < 3 ? { siguiente_cursor: `pagina-${numero + 1}` } : {}) };
    }, decidir: async () => assert.fail("POST inesperado"),
  } });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
  assert.equal(panel.querySelector("[data-dietas-circuito-anterior]"), null);
  assert.match(panel.querySelector("[data-dietas-circuito-pagina]").textContent, /Página 1 · 1 documento en esta página/u);
  await clic(panel, "siguiente"); await clic(panel, "siguiente");
  assert.equal(panel.querySelectorAll("[data-dietas-circuito-comision]").length, 1);
  assert.equal(panel.querySelector("[data-dietas-circuito-comision]").dataset.dietasCircuitoComision, filaPagina(3).referencia);
  assert.equal(panel.querySelector("[data-dietas-circuito-siguiente]"), null);
  assert.match(panel.querySelector("[data-dietas-circuito-pagina]").textContent, /Página 3/u);
  assert.equal(contenedor.ownerDocument.activeElement, panel.querySelector("[data-dietas-circuito-pagina]"));
  await clic(panel, "anterior");
  assert.equal(consultas.at(-1).cursor, "pagina-2");
  assert.equal(panel.querySelector("[data-dietas-circuito-comision]").dataset.dietasCircuitoComision, filaPagina(2).referencia);
  await clic(panel, "primera");
  assert.deepEqual(consultas.at(-1), { etapa: "revision", limit: 20 });
  assert.match(panel.querySelector("[data-dietas-circuito-pagina]").textContent, /Página 1/u);
  assert.equal(panel.querySelector("[data-dietas-circuito-anterior]"), null);
  assert.equal(consultas.length, 5);
  vista.desmontar();
});

test("conserva página y filtros al abrir y cerrar un documento; una nueva búsqueda empieza en la primera página", async () => {
  const contenedor = raiz(); const consultas = []; const lecturas = [];
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { control: true, etapas: ["revision", "autorizacion"], cliente: {
    listar: async (consulta) => { consultas.push(consulta); return { items: [fila], competencia: "acreditada", siguiente_cursor: "pagina-siguiente" }; },
    documento: async (ref, etapa) => { lecturas.push([ref, etapa]); return documentoLeido; }, decidir: async () => assert.fail("POST inesperado"),
  } });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
  const filtros = panel.querySelector("[data-dietas-circuito-filtros]");
  panel.querySelector("[data-dietas-circuito-desde]").value = "2026-09-01";
  panel.querySelector("[data-dietas-circuito-hasta]").value = "2026-09-30";
  await filtros.listeners.submit({ preventDefault() {} }); await clic(panel, "siguiente");
  const consultaPagina = { etapa: "revision", limit: 20, fecha_desde: "2026-09-01", fecha_hasta: "2026-09-30", cursor: "pagina-siguiente" };
  assert.deepEqual(consultas.at(-1), consultaPagina);
  await clic(panel, "abrir"); await clic(panel, "volver");
  assert.deepEqual(lecturas, [[referencia, "revision"]]);
  assert.equal(consultas.length, 3);
  assert.match(panel.querySelector("[data-dietas-circuito-pagina]").textContent, /Página 2/u);
  assert.equal(contenedor.ownerDocument.activeElement, panel.querySelector("[data-dietas-circuito-abrir]"));
  await vista.recargar(); assert.deepEqual(consultas.at(-1), consultaPagina);
  panel.querySelector("[data-dietas-circuito-etapa]").value = "autorizacion";
  panel.querySelector("[data-dietas-circuito-desde]").value = "2026-09-10";
  await filtros.listeners.submit({ preventDefault() {} });
  assert.deepEqual(consultas.at(-1), { etapa: "autorizacion", limit: 20, fecha_desde: "2026-09-10", fecha_hasta: "2026-09-30" });
  assert.equal(panel.querySelector("[data-dietas-circuito-anterior]"), null);
  assert.match(panel.querySelector("[data-dietas-circuito-pagina]").textContent, /Página 1/u);
  vista.desmontar();
});

test("un filtro nuevo aborta la página pendiente y descarta sus filas o errores tardíos", async () => {
  for (const falla of [false, true]) {
    const contenedor = raiz(); const pendiente = aplazada(); let signal; let llamadas = 0;
    const vista = montarVistaBandejaCircuitoDietas(contenedor, { control: true, cliente: {
      listar: async (consulta, opciones) => {
        llamadas++;
        if (consulta.cursor) { signal = opciones.signal; return pendiente.promise; }
        return { items: [filaPagina(llamadas)], competencia: "acreditada", siguiente_cursor: "pagina-obsoleta" };
      }, decidir: async () => assert.fail("POST inesperado"),
    } });
    await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
    const avance = clic(panel, "siguiente");
    assert.equal(panel.querySelector("[data-dietas-circuito-contenido]").attrs["aria-busy"], "true");
    assert.equal(panel.querySelector("[data-dietas-circuito-abrir]"), null);
    panel.querySelector("[data-dietas-circuito-desde]").value = "2026-10-01";
    await panel.querySelector("[data-dietas-circuito-filtros]").listeners.submit({ preventDefault() {} });
    assert.equal(signal.aborted, true);
    if (falla) pendiente.reject(new Error("fallo tardío"));
    else pendiente.resolve({ items: [filaPagina(99)], competencia: "acreditada", siguiente_cursor: "otro-cursor" });
    await avance;
    assert.equal(panel.querySelector("[data-dietas-circuito-comision]").dataset.dietasCircuitoComision, filaPagina(3).referencia);
    assert.match(panel.querySelector("[data-dietas-circuito-pagina]").textContent, /Página 1/u);
    assert.equal(panel.querySelector("[data-dietas-circuito-estado]").textContent, "");
    assert.equal(panel.querySelector("[data-dietas-circuito-contenido]").attrs["aria-busy"], "false");
    vista.desmontar();
  }
});

test("un fallo al avanzar permite repetir la misma página o regresar, sin mostrar filas antiguas ni un recuento ficticio", async () => {
  const contenedor = raiz(); const consultas = []; let intentos = 0;
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { cliente: {
    listar: async (consulta) => {
      consultas.push(consulta);
      if (consulta.cursor && !intentos++) throw new Error("sin conexión");
      return { items: consulta.cursor ? [] : [fila], competencia: "acreditada", ...(!consulta.cursor ? { siguiente_cursor: "pagina-2" } : {}) };
    }, decidir: async () => assert.fail("POST inesperado"),
  } });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
  await clic(panel, "siguiente");
  assert.equal(panel.querySelector("[data-dietas-circuito-comision]"), null);
  assert.equal(panel.querySelector("[data-dietas-circuito-pagina]").textContent, "");
  assert.match(panel.querySelector("[data-dietas-circuito-estado]").textContent, /No se ha podido consultar/u);
  assert.ok(panel.querySelector("[data-dietas-circuito-anterior]"));
  assert.equal(contenedor.ownerDocument.activeElement, panel.querySelector("[data-dietas-circuito-estado]"));
  await clic(panel, "reintentar");
  assert.equal(consultas.at(-1).cursor, "pagina-2");
  assert.match(panel.querySelector("[data-dietas-circuito-pagina]").textContent, /Página 2 · 0 documentos en esta página/u);
  await clic(panel, "anterior");
  assert.equal(consultas.at(-1).cursor, undefined);
  assert.equal(panel.querySelectorAll("[data-dietas-circuito-comision]").length, 1);
  vista.desmontar();
});

test("sin fuente tras avanzar retira navegación y documentos, incluso si la respuesta contiene cursor o filas", async () => {
  for (const error of [false, true]) {
    const contenedor = raiz();
    const vista = montarVistaBandejaCircuitoDietas(contenedor, { cliente: {
      listar: async (consulta) => {
        if (!consulta.cursor) return { items: [fila], competencia: "acreditada", siguiente_cursor: "pagina-2" };
        if (error) throw { codigo: "competencia_sin_fuente" };
        return { items: [filaPagina(2)], competencia: "sin_fuente", siguiente_cursor: "pagina-3" };
      }, decidir: async () => assert.fail("POST inesperado"),
    } });
    await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
    await clic(panel, "siguiente");
    assert.ok(panel.querySelector("[data-dietas-circuito-sin-fuente]"));
    for (const atributo of ["abrir", "anterior", "primera", "siguiente", "reintentar"]) assert.equal(panel.querySelector(`[data-dietas-circuito-${atributo}]`), null);
    vista.desmontar();
  }
});

test("el documento tardío no altera una nueva apertura de la misma comisión tras cambiar de consulta", async () => {
  const contenedor = raiz(); const primeraLectura = aplazada(); let lecturas = 0; let signal;
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { control: true, cliente: {
    listar: async () => ({ items: [fila], competencia: "acreditada" }),
    documento: (_ref, _etapa, opciones) => { if (!lecturas++) { signal = opciones.signal; return primeraLectura.promise; } return Promise.resolve({ ...documentoLeido, motivo: "Documento actualizado" }); },
    decidir: async () => assert.fail("POST inesperado"),
  } });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
  const primeraApertura = clic(panel, "abrir");
  panel.querySelector("[data-dietas-circuito-desde]").value = "2026-09-01";
  await panel.querySelector("[data-dietas-circuito-filtros]").listeners.submit({ preventDefault() {} });
  assert.equal(signal.aborted, true); await clic(panel, "abrir");
  primeraLectura.resolve(documentoLeido); await primeraApertura;
  assert.match(texto(panel.querySelector("[data-dietas-circuito-detalle]")), /Documento actualizado/u);
  assert.doesNotMatch(texto(panel.querySelector("[data-dietas-circuito-detalle]")), /Reunión técnica/u);
  vista.desmontar();
});

test("el recuento y los controles usan el catálogo inglés real", async () => {
  const textos = await cargarTextos("dietas", { idioma: "en" }); const contenedor = raiz();
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { traducir: (clave, variables) => textos.traducir(`circuito.${clave}`, variables), cliente: {
    listar: async (consulta) => ({ items: consulta.cursor ? [fila] : [fila, filaPagina(2)], competencia: "acreditada", ...(!consulta.cursor ? { siguiente_cursor: "pagina-2" } : {}) }),
    decidir: async () => assert.fail("POST inesperado"),
  } });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
  assert.equal(panel.querySelector("[data-dietas-circuito-pagina]").textContent, "Page 1 · 2 documents on this page");
  assert.equal(panel.querySelector("[data-dietas-circuito-siguiente]").textContent, "Next");
  await clic(panel, "siguiente");
  assert.equal(panel.querySelector("[data-dietas-circuito-pagina]").textContent, "Page 2 · 1 document on this page");
  assert.equal(panel.querySelector("[data-dietas-circuito-anterior]").textContent, "Previous");
  assert.equal(panel.querySelector("[data-dietas-circuito-primera]").textContent, "First page");
  vista.desmontar();
});

test("una página demorada conserva el foco que la persona ha movido a un filtro, también ante error o sin fuente", async () => {
  for (const resultado of ["pagina", "error", "sin_fuente"]) {
    const contenedor = raiz(); const lectura = aplazada();
    const vista = montarVistaBandejaCircuitoDietas(contenedor, { control: true, cliente: {
      listar: (consulta) => consulta.cursor ? lectura.promise : Promise.resolve({ items: [fila], competencia: "acreditada", siguiente_cursor: "pagina-2" }),
      decidir: async () => assert.fail("POST inesperado"),
    } });
    await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
    panel.querySelector("[data-dietas-circuito-siguiente]").focus();
    const avance = clic(panel, "siguiente");
    const filtro = panel.querySelector("[data-dietas-circuito-desde]"); filtro.focus(); filtro.value = "2026-10-01";
    if (resultado === "error") lectura.reject(new Error("sin conexión"));
    else lectura.resolve({ items: resultado === "pagina" ? [filaPagina(2)] : [], competencia: resultado === "pagina" ? "acreditada" : "sin_fuente" });
    await avance;
    assert.equal(contenedor.ownerDocument.activeElement, filtro, resultado);
    assert.equal(filtro.value, "2026-10-01");
    vista.desmontar();
  }
});

test("un documento demorado conserva el foco que la persona ha movido a un filtro al completar o fallar su lectura", async () => {
  for (const falla of [false, true]) {
    const contenedor = raiz(); const lectura = aplazada();
    const vista = montarVistaBandejaCircuitoDietas(contenedor, { control: true, cliente: {
      listar: async () => ({ items: [fila], competencia: "acreditada" }), documento: () => lectura.promise,
      decidir: async () => assert.fail("POST inesperado"),
    } });
    await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
    panel.querySelector("[data-dietas-circuito-abrir]").focus();
    const apertura = clic(panel, "abrir");
    assert.equal(contenedor.ownerDocument.activeElement, panel.querySelector("[data-dietas-circuito-volver]"));
    const filtro = panel.querySelector("[data-dietas-circuito-hasta]"); filtro.focus(); filtro.value = "2026-10-31";
    if (falla) lectura.reject(new Error("sin conexión")); else lectura.resolve(documentoLeido);
    await apertura;
    assert.equal(contenedor.ownerDocument.activeElement, filtro);
    assert.equal(filtro.value, "2026-10-31");
    vista.desmontar();
  }
});

test("al autorizar o devolver conserva todo el envío incierto, bloquea la navegación y evita dobles envíos", async () => {
  for (const decision of ["aprobar", "devolver"]) {
    const contenedor = raiz(); const primera = aplazada(); const segunda = aplazada();
    const entradas = []; const consultas = []; const documentos = []; let claves = 0;
    const vista = montarVistaBandejaCircuitoDietas(contenedor, { control: true, etapas: ["autorizacion", "revision"], etapaInicial: "autorizacion",
      generarClaveIdempotencia: () => { claves++; return "decision-responsable-0001"; }, cliente: {
        listar: async (consulta) => { consultas.push(consulta); return { items: [fila, filaPagina(2)], competencia: "acreditada", siguiente_cursor: "pagina-2" }; },
        documento: async (ref, etapa) => { documentos.push([ref, etapa]); return documentoLeido; },
        decidir: (ref, entrada) => { entradas.push([ref, entrada]); return entradas.length === 1 ? primera.promise : segunda.promise; },
      } });
    await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]");
    const otroDocumento = panel.querySelectorAll("[data-dietas-circuito-abrir]")[1];
    const paginaSiguiente = panel.querySelector("[data-dietas-circuito-siguiente]");
    await clic(panel, "abrir");
    const motivo = "Falta el justificante del taxi";
    panel.querySelector("[data-dietas-circuito-motivo]").value = motivo;
    const boton = panel.querySelector(`[data-dietas-circuito-decision="${decision}"]`); boton.focus();
    const envio = panel.listeners.click({ target: boton });
    assert.equal(contenedor.ownerDocument.activeElement, panel.querySelector("[data-dietas-circuito-estado]"));
    const comprobarBloqueo = async () => {
      for (const atributo of ["etapa", "desde", "hasta"]) assert.equal(panel.querySelector(`[data-dietas-circuito-${atributo}]`).disabled, true);
      assert.equal(panel.querySelector('[data-dietas-circuito-decision="aprobar"]').disabled, true);
      assert.equal(panel.querySelector('[data-dietas-circuito-decision="devolver"]').disabled, true);
      assert.equal(panel.querySelector("[data-dietas-circuito-volver]").disabled, true);
      assert.equal(panel.querySelector("[data-dietas-circuito-motivo]").readOnly, true);
      assert.equal(panel.querySelector("[data-dietas-circuito-motivo]").value, motivo);
      // Incluso eventos de controles retirados o enviados por código quedan cerrados.
      panel.querySelector("[data-dietas-circuito-etapa]").value = "revision";
      await panel.querySelector("[data-dietas-circuito-filtros]").listeners.submit({ preventDefault() {} });
      for (const target of [otroDocumento, paginaSiguiente, boton]) await panel.listeners.click({ target });
      await clic(panel, "volver"); await vista.recargar();
      assert.equal(consultas.length, 1); assert.equal(documentos.length, 1);
    };
    await comprobarBloqueo(); assert.equal(entradas.length, 1);
    primera.reject({ resultadoIndeterminado: true }); await envio;
    const reintento = panel.querySelector("[data-dietas-circuito-reintento]");
    assert.equal(contenedor.ownerDocument.activeElement, reintento);
    await comprobarBloqueo();
    const repeticion = panel.listeners.click({ target: reintento });
    assert.equal(panel.querySelector("[data-dietas-circuito-reintento]").disabled, true);
    await panel.listeners.click({ target: reintento });
    assert.equal(entradas.length, 2); assert.equal(claves, 1);
    assert.deepEqual(entradas[1], [referencia, { etapa: "autorizacion", decision, motivo, clave_idempotencia: "decision-responsable-0001", version_esperada: 3 }]);
    assert.deepEqual(entradas[1], entradas[0]);
    segunda.resolve({ comision: { referencia, estado: "pendiente_liquidacion", version: 4 }, recibo: { ...recibo, repeticion: true } }); await repeticion;
    assert.match(panel.querySelector("[data-dietas-circuito-estado]").textContent, /Ya estaba registrado/u);
    assert.equal(contenedor.ownerDocument.activeElement, panel.querySelector("[data-dietas-circuito-estado]"));
    assert.equal(panel.querySelector("[data-dietas-circuito-etapa]").disabled, false);
    vista.desmontar();
  }
});

test("rechaza motivos inválidos antes de decidir también al aprobar y conserva el campo para corregir", async () => {
  const contenedor = raiz(); let llamadas = 0; let claves = 0;
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { generarClaveIdempotencia: () => { claves++; return "decision-responsable-0002"; }, cliente: {
    listar: async () => ({ items: [fila], competencia: "acreditada" }), documento: async () => documentoLeido,
    decidir: async () => { llamadas++; throw { codigo: "conflicto_estado" }; },
  } });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]"); await clic(panel, "abrir");
  for (const decision of ["aprobar", "devolver"]) {
    for (const motivo of ["é".repeat(301), "😀".repeat(151), "a".repeat(601), "Falta\njustificante", "Falta\u0000justificante", "Falta\u007fjustificante", ...(decision === "devolver" ? ["", "é", "\ufeff\u0085 "] : [])]) {
      panel.querySelector("[data-dietas-circuito-motivo]").value = motivo;
      await panel.listeners.click({ target: panel.querySelector(`[data-dietas-circuito-decision="${decision}"]`) });
      const campo = panel.querySelector("[data-dietas-circuito-motivo]");
      assert.equal(campo.value, motivo); assert.equal(contenedor.ownerDocument.activeElement, campo);
      assert.equal(campo.attrs["aria-invalid"], "true");
      const errorCampo = panel.querySelector("[data-dietas-circuito-motivo-error]");
      assert.equal(campo.attrs["aria-describedby"], errorCampo.id);
      assert.equal(errorCampo.textContent, panel.querySelector("[data-dietas-circuito-estado]").textContent);
      assert.match(panel.querySelector("[data-dietas-circuito-estado]").textContent, /motivo/u);
      assert.equal(llamadas, 0); assert.equal(claves, 0);
    }
  }
  for (const motivo of ["漢", "😀", "é".repeat(300)]) {
    panel.querySelector("[data-dietas-circuito-motivo]").value = motivo;
    await panel.listeners.click({ target: panel.querySelector('[data-dietas-circuito-decision="devolver"]') });
    assert.equal(panel.querySelector("[data-dietas-circuito-motivo]").value, motivo);
  }
  assert.equal(llamadas, 3); vista.desmontar();
});

test("la retirada de la fuente en la decisión cierra documento y acciones", async () => {
  const contenedor = raiz();
  const vista = montarVistaBandejaCircuitoDietas(contenedor, { generarClaveIdempotencia: () => "decision-responsable-0003", cliente: {
    listar: async () => ({ items: [fila], competencia: "acreditada" }), documento: async () => documentoLeido,
    decidir: async () => { throw { codigo: "competencia_sin_fuente" }; },
  } });
  await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]"); await clic(panel, "abrir");
  await panel.listeners.click({ target: panel.querySelector('[data-dietas-circuito-decision="aprobar"]') });
  assert.ok(panel.querySelector("[data-dietas-circuito-sin-fuente]"));
  assert.equal(panel.querySelector("[data-dietas-circuito-decision]"), null); assert.equal(panel.querySelector("[data-dietas-circuito-abrir]"), null);
  vista.desmontar();
});


test("el error junto al motivo distingue aprobación inválida de devolución incompleta en ambos catálogos", async () => {
  for (const idioma of ["es", "en"]) {
    const textos = await cargarTextos("dietas", { idioma }); const contenedor = raiz(); let llamadas = 0;
    const vista = montarVistaBandejaCircuitoDietas(contenedor, { traducir: (clave, variables) => textos.traducir(`circuito.${clave}`, variables),
      generarClaveIdempotencia: () => "decision-responsable-0004", cliente: {
        listar: async () => ({ items: [fila], competencia: "acreditada" }), documento: async () => documentoLeido,
        decidir: async () => { llamadas++; throw { codigo: "conflicto_estado" }; },
      } });
    await esperar(); const panel = contenedor.querySelector("[data-dietas-bandeja-circuito]"); await clic(panel, "abrir");
    for (const motivo of ["é".repeat(301), "Falta\njustificante"]) {
      panel.querySelector("[data-dietas-circuito-motivo]").value = motivo;
      await panel.listeners.click({ target: panel.querySelector('[data-dietas-circuito-decision="aprobar"]') });
      assert.equal(panel.querySelector("[data-dietas-circuito-motivo-error]").textContent, textos.traducir("circuito.circuito_motivo_invalido"));
      assert.equal(llamadas, 0);
    }
    panel.querySelector("[data-dietas-circuito-motivo]").value = "";
    await panel.listeners.click({ target: panel.querySelector('[data-dietas-circuito-decision="devolver"]') });
    assert.equal(panel.querySelector("[data-dietas-circuito-motivo-error]").textContent, textos.traducir("circuito.circuito_motivo_devolucion_incompleto"));
    assert.equal(llamadas, 0);
    // Una aprobación válida limpia el error anterior aunque el servidor devuelva conflicto.
    await panel.listeners.click({ target: panel.querySelector('[data-dietas-circuito-decision="aprobar"]') });
    assert.equal(llamadas, 1);
    assert.equal(panel.querySelector("[data-dietas-circuito-motivo]").attrs["aria-invalid"], "false");
    assert.equal(panel.querySelector("[data-dietas-circuito-motivo]").attrs["aria-describedby"], undefined);
    assert.equal(panel.querySelector("[data-dietas-circuito-motivo-error]"), null);
    vista.desmontar();
  }
});
