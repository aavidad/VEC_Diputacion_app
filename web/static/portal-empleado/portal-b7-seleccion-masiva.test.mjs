import test from "node:test";
import assert from "node:assert/strict";
import { consultarSeleccionMasivaBolsa, crearControladorBolsas } from "./portal-bolsas-api.js";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js";

function candidata(orden, estado_clave = "disponible") {
  return { participacion_ref: `participacion:${String(orden).padStart(3, "0")}`, estado_clave, orden };
}
function pagina(candidatos, cursor_siguiente = null, bolsaRef = "bolsa:01") {
  return { ok: true, datos: { generado_en: "2026-09-23T10:00:00Z", bolsa: { bolsa_ref: bolsaRef, total: 120, por_estado: { disponible: 110, no_disponible: 10 }, politica_orden: { politica_ref: "politica:b6", version: 1 } }, candidatos, hay_mas: cursor_siguiente !== null, cursor_siguiente } };
}

test("B7 recorre el cursor completo y ordena por B6 aunque una página posterior tenga mejor turno", async () => {
  const llamadas = [];
  const resultado = await consultarSeleccionMasivaBolsa("bolsa:01", ["disponible", "disponible_desde"], {
    consultar: async (ref, opciones) => {
      llamadas.push({ ref, ...opciones });
      if (!opciones.cursor) return pagina([candidata(40), candidata(3, "no_disponible")], "cursor:02");
      return pagina([candidata(2, "disponible_desde"), candidata(1)], null);
    },
  });
  assert.equal(resultado.ok, true);
  assert.deepEqual(resultado.participaciones, ["participacion:001", "participacion:002", "participacion:040"]);
  assert.equal(resultado.total, 3);
  assert.deepEqual(llamadas.map(({ cursor, limite, estado, texto }) => [cursor, limite, estado, texto]), [["", 100, undefined, undefined], ["cursor:02", 100, undefined, undefined]]);
  assert.ok(llamadas.every(({ ref }) => ref === "bolsa:01"));
});

test("B7 conserva 100 exactas y con 101 sólo permite las primeras 100", async () => {
  for (const cantidad of [100, 101]) {
    const primera = Array.from({ length: 50 }, (_, i) => candidata(cantidad - i));
    const segunda = Array.from({ length: cantidad - 50 }, (_, i) => candidata(cantidad - 50 - i));
    const resultado = await consultarSeleccionMasivaBolsa("bolsa:01", ["disponible"], {
      consultar: async (_ref, { cursor }) => cursor ? pagina(segunda) : pagina(primera, "siguiente"),
    });
    assert.equal(resultado.ok, true);
    assert.equal(resultado.total, cantidad);
    assert.equal(resultado.participaciones.length, 100);
    assert.equal(resultado.participaciones[0], "participacion:001");
    assert.equal(resultado.participaciones.at(-1), "participacion:100");
  }
});

test("B7 rechaza un 403 intermedio y una paginación cambiante sin selección parcial", async () => {
  const denegada = await consultarSeleccionMasivaBolsa("bolsa:01", ["disponible"], {
    consultar: async (_ref, { cursor }) => cursor ? { ok: false, status: 403, mensaje: "Denegado" } : pagina([candidata(1)], "siguiente"),
  });
  assert.deepEqual(denegada, { ok: false, status: 403, mensaje: "Denegado" });
  const cambiada = await consultarSeleccionMasivaBolsa("bolsa:01", ["disponible"], {
    consultar: async (_ref, { cursor }) => cursor ? pagina([candidata(2)], null, "bolsa:02") : pagina([candidata(1)], "siguiente"),
  });
  assert.equal(cambiada.ok, false);
  assert.equal(cambiada.status, 409);
  assert.equal(cambiada.participaciones, undefined);
  const otroInstante = await consultarSeleccionMasivaBolsa("bolsa:01", ["disponible"], {
    consultar: async (_ref, { cursor }) => {
      const respuesta = cursor ? pagina([candidata(2)]) : pagina([candidata(1)], "siguiente");
      if (cursor) respuesta.datos.generado_en = "2026-09-23T10:01:00Z";
      return respuesta;
    },
  });
  assert.equal(otroInstante.status, 409);
});

test("B7 ignora el filtro B5 al iniciar y consulta la bolsa completa por estados propios", async () => {
  const escuchas = {};
  const llamadas = [];
  const form = { estados: ["disponible"] };
  const documento = {
    addEventListener(tipo, fn) { escuchas[tipo] = fn; },
    querySelector(selector) {
      if (selector === '[aria-current="step"]') return { focus() {} };
      return selector === '[data-bolsa-form="b7-paso2"]' ? form : null;
    },
  };
  const FormDataOriginal = globalThis.FormData;
  globalThis.FormData = class { getAll(clave) { return clave === "estado" ? form.estados : []; } };
  try {
    const estado = { bolsaSeleccionada: "bolsa:01", filtrosBolsa: { estado: "no_disponible", texto: "nombre" }, datosCandidatos: { carga: "listo", datos: pagina([candidata(5, "no_disponible")]).datos } };
    crearControladorBolsas({ estado, renderizar: () => {}, navegar: () => {}, documento, obtenerFuenteLectura: () => ({
      consultarCandidatosBolsa: async (_ref, opciones) => { llamadas.push(opciones); return pagina([candidata(1)]); },
    }) }).instalar();
    const click = (accion) => escuchas.click({ preventDefault() {}, target: { closest(selector) { return selector === "[data-bolsa-accion]" ? { dataset: { bolsaAccion: accion } } : null; } } });
    click("iniciar-b7");
    assert.equal(estado.filtrosBolsa.estado, "");
    assert.equal(estado.filtrosBolsa.texto, "");
    escuchas.submit({ preventDefault() {}, target: { closest(selector) { return selector === '[data-bolsa-form="b7-paso1"]' ? {} : null; } } });
    await new Promise(setImmediate);
    click("b7-seleccionar-todas");
    await new Promise(setImmediate);
    assert.deepEqual(estado.filtrosBolsa.nuevo_llamamiento.participaciones, ["participacion:001"]);
    assert.ok(llamadas.every((opciones) => (opciones.estado === "" || opciones.estado === undefined) && (opciones.texto === "" || opciones.texto === undefined)));
    assert.equal(llamadas.at(-1).limite, 100);
  } finally {
    globalThis.FormData = FormDataOriginal;
  }
});

test("B7 descarta una respuesta tardía tras iniciar otra selección", async () => {
  const escuchas = {};
  const solicitudes = [];
  const form = { estados: ["disponible"] };
  const documento = {
    addEventListener(tipo, fn) { escuchas[tipo] = fn; },
    querySelector(selector) { return selector === '[data-bolsa-form="b7-paso2"]' ? form : null; },
  };
  const FormDataOriginal = globalThis.FormData;
  globalThis.FormData = class {
    constructor(elemento) { assert.equal(elemento, form); }
    getAll(clave) { return clave === "estado" ? form.estados : []; }
  };
  try {
    const flujo = { paso: 2, estados: ["disponible"], participaciones: [], seleccion_total: false };
    const estado = { bolsaSeleccionada: "bolsa:01", filtrosBolsa: { estado: "", texto: "", nuevo_llamamiento: flujo } };
    const controlador = crearControladorBolsas({ estado, renderizar: () => {}, navegar: () => {}, documento, obtenerFuenteLectura: () => ({
      consultarCandidatosBolsa: (_ref, _opciones, { signal }) => new Promise((resolver) => solicitudes.push({ resolver, signal })),
    }) });
    controlador.instalar();
    const boton = { dataset: { bolsaAccion: "b7-seleccionar-todas" } };
    const click = () => escuchas.click({ preventDefault() {}, target: { closest(selector) { return selector === "[data-bolsa-accion]" ? boton : null; } } });
    click();
    assert.equal(flujo.consultando, true);
    click();
    assert.equal(solicitudes[0].signal.aborted, true);
    solicitudes[1].resolver(pagina([candidata(2)]));
    await new Promise(setImmediate);
    solicitudes[0].resolver(pagina([candidata(1)]));
    await new Promise(setImmediate);
    assert.deepEqual(flujo.participaciones, ["participacion:002"]);
    assert.equal(flujo.seleccion_total, true);
    assert.equal(flujo.consultando, false);
    escuchas.submit({ preventDefault() {}, target: { closest(selector) { return selector === '[data-bolsa-form="b7-paso2"]' ? form : null; } } });
    assert.equal(flujo.paso, 3);
    assert.deepEqual(flujo.participaciones, ["participacion:002"]);
  } finally {
    globalThis.FormData = FormDataOriginal;
  }
});

test("B7 cancela la selección en curso al cambiar los estados del filtro", async () => {
  const escuchas = {};
  let resolverConsulta;
  const form = { estados: ["disponible"] };
  const documento = {
    addEventListener(tipo, fn) { escuchas[tipo] = fn; },
    querySelector(selector) { return selector === '[data-bolsa-form="b7-paso2"]' ? form : null; },
  };
  const FormDataOriginal = globalThis.FormData;
  globalThis.FormData = class { getAll(clave) { return clave === "estado" ? form.estados : []; } };
  try {
    const flujo = { paso: 2, estados: ["disponible"], participaciones: [], seleccion_total: false };
    const estado = { bolsaSeleccionada: "bolsa:01", filtrosBolsa: { estado: "", texto: "", nuevo_llamamiento: flujo } };
    crearControladorBolsas({ estado, renderizar: () => {}, navegar: () => {}, documento, obtenerFuenteLectura: () => ({
      consultarCandidatosBolsa: () => new Promise((resolver) => { resolverConsulta = resolver; }),
    }) }).instalar();
    const boton = { dataset: { bolsaAccion: "b7-seleccionar-todas" } };
    escuchas.click({ preventDefault() {}, target: { closest(selector) { return selector === "[data-bolsa-accion]" ? boton : null; } } });
    form.estados = ["disponible_desde"];
    escuchas.change({ target: { name: "estado", closest(selector) { return selector === '[data-bolsa-form="b7-paso2"]' ? form : null; } } });
    resolverConsulta(pagina([candidata(1)]));
    await new Promise(setImmediate);
    assert.deepEqual(flujo.participaciones, []);
    assert.deepEqual(flujo.estados, ["disponible_desde"]);
    assert.equal(flujo.consultando, false);
  } finally {
    globalThis.FormData = FormDataOriginal;
  }
});

test("B7 muestra página local, consulta pendiente, límite y confirmación de cantidad exacta", () => {
  const flujo = { paso: 2, estados: ["disponible"], participaciones: ["participacion:001"], totalElegibles: 101, consultando: true, error: "" };
  const bolsa = { bolsa_ref: "bolsa:01", categoria: "Auxiliar", tipo_lista: "ordinaria", vigente_desde: "2026-09-01", total: 120, por_estado: { disponible: 101 } };
  const presentador = crearPresentadorPanelInterno({
    claseEstado: () => "info", etiquetaClave: (valor) => valor,
    encabezadoVista: (_seccion, titulo, descripcion, acciones = "") => `<header><h2>${titulo}</h2><p>${descripcion}</p>${acciones}</header>`,
    escaparHTML: (valor) => String(valor ?? ""), numero: (valor) => String(valor ?? 0),
    obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }), tituloVista: (valor) => valor,
    obtenerDatosCandidatosBolsa: () => ({ carga: "listo", datos: { bolsa, candidatos: [candidata(1)], contactos: [] }, error: "" }),
    obtenerEstadoCandidatos: () => ({ nuevo_llamamiento: flujo }),
  });
  let html = presentador.renderizarVista("bolsa-candidatos");
  assert.match(html, /1 ocupan turno en esta página/);
  assert.match(html, /aria-busy="true"/);
  assert.match(html, /Seleccionar todas las que cumplen el filtro<\/button>/);
  assert.match(html, /aria-label="Ayuda sobre el límite de selección">\?<\/summary>/);
  assert.match(html, /Configurar llamamiento<\/button>/);
  assert.match(html, /Consultando todas las páginas/);
  flujo.consultando = false;
  flujo.participaciones = Array.from({ length: 100 }, (_, i) => candidata(i + 1).participacion_ref);
  html = presentador.renderizarVista("bolsa-candidatos");
  assert.match(html, /100 seleccionadas.*de 101 candidaturas/);
  flujo.paso = 4;
  flujo.configuracion = { referencia: "NEC-01", centro: "Centro", modalidad: "Sustitución", plazo: "provisional" };
  html = presentador.renderizarVista("bolsa-candidatos");
  assert.match(html, /data-cantidad="100"/);
  assert.match(html, /Confirmo la emisión para exactamente 100 candidatos/);
  assert.match(html, /100 seleccionados de 101 elegibles/);
});

test("B7 devuelve al paso de selección un 422 de orden vigente sin recibo de éxito", async () => {
  const escuchas = {};
  const form = { dataset: { cantidad: "1" } };
  const documento = { addEventListener(tipo, fn) { escuchas[tipo] = fn; }, querySelector() { return null; } };
  const FormDataOriginal = globalThis.FormData;
  const fetchOriginal = globalThis.fetch;
  let enviado;
  globalThis.FormData = class { get(clave) { return clave === "confirmacion" ? "on" : null; } };
  globalThis.fetch = async (_url, opciones) => {
    enviado = JSON.parse(opciones.body);
    return { status: 422, json: async () => ({ error: { codigo: "orden_desactualizado" } }) };
  };
  try {
    const flujo = { paso: 4, estados: ["disponible"], participaciones: ["participacion:001"], configuracion: { referencia: "NEC-01" }, recibo: "", error: "" };
    const estado = { bolsaSeleccionada: "bolsa:01", filtrosBolsa: { estado: "", texto: "", nuevo_llamamiento: flujo } };
    crearControladorBolsas({ estado, renderizar: () => {}, navegar: () => {}, documento }).instalar();
    escuchas.submit({ preventDefault() {}, target: { closest(selector) { return selector === '[data-bolsa-form="b7-paso4"]' ? form : null; } } });
    await new Promise(setImmediate);
    assert.deepEqual(enviado.participaciones, ["participacion:001"]);
    assert.equal(flujo.paso, 2);
    assert.deepEqual(flujo.participaciones, []);
    assert.equal(flujo.recibo, "");
    assert.match(flujo.error, /selecci[oó]n ya no respeta la bolsa|orden vigente/i);
  } finally {
    globalThis.FormData = FormDataOriginal;
    globalThis.fetch = fetchOriginal;
  }
});
