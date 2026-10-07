import assert from "node:assert/strict";
import test from "node:test";
import { montarCuadroContratacionLigero } from "./vista-cuadro-ligera.js";

const fila = Object.freeze({
  expediente_ref: "expediente:ct:001", numero_visible: "2026/CT-0001", version: 2,
  fase_clave: "analisis", estado_clave: "en_curso", centro_ref: "centro:rpt:600",
  categoria_ref: "categoria:auxiliar", creado_en: "2026-10-01T08:00:00Z",
  actualizado_en: "2026-10-01T09:00:00Z",
});

function raizFalsa() {
  const eventos = new Map();
  return { innerHTML: "", eventos, addEventListener(tipo, fn) { eventos.set(tipo, fn); },
    removeEventListener(tipo) { eventos.delete(tipo); }, querySelector() { return null; } };
}

test("la bandeja ligera usa una consulta autorizada y abre detalle solo tras pulsar su referencia", async () => {
  const raiz = raizFalsa(), consultas = [], abiertos = [], errores = [];
  const montaje = await montarCuadroContratacionLigero({
    raiz,
    cliente: { consultarCuadroRRHH: async (solicitud, opciones) => {
      consultas.push({ solicitud, opciones });
      return { generada_en: "2026-10-01T09:00:00Z", expedientes: [fila], hay_mas: false };
    } },
    idioma: "es",
    abrirDetalle: async (datos) => { abiertos.push(datos); },
    mostrarError: (_raiz, datos) => { errores.push(datos); },
  });
  assert.equal(errores.length, 0, errores[0]?.error?.stack);
  assert.equal(consultas.length, 1);
  assert.equal(consultas[0].solicitud.paginacion.limite, 100);
  assert.match(raiz.innerHTML, /2026\/CT-0001/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-exp-vista="alta"/u);
  assert.equal(abiertos.length, 0);
  await raiz.eventos.get("click")({ target: { closest: () => ({ dataset: { ctExpAbrir: fila.expediente_ref } }) } });
  assert.equal(abiertos.length, 1);
  assert.equal(abiertos[0].expedienteRef, fila.expediente_ref);
  assert.equal(abiertos[0].textos.vista, "expediente");
  montaje.desmontar();
  assert.equal(raiz.eventos.size, 0);
});

test("un fallo de consulta conserva acción de reintento y una sola lectura por intento", async () => {
  const raiz = raizFalsa(), errores = [];
  let intentos = 0;
  const montaje = await montarCuadroContratacionLigero({
    raiz, idioma: "en", abrirDetalle: async () => {},
    cliente: { consultarCuadroRRHH: async () => {
      intentos++;
      if (intentos === 1) throw new Error("503 de prueba");
      return { generada_en: "2026-10-01T09:00:00Z", expedientes: [], hay_mas: false };
    } },
    mostrarError: (_raiz, datos) => { errores.push(datos); },
  });
  assert.equal(intentos, 1);
  assert.equal(errores.length, 1);
  await errores[0].reintentar();
  assert.equal(intentos, 2);
  assert.match(raiz.innerHTML, /There are no requests for this profile|No hay peticiones para este perfil/u);
  assert.match(raiz.innerHTML, /data-ct-exp-recargar/u);
  montaje.desmontar();
});

test("un enlace de incidencia consulta el conjunto filtrado y uno de plazo no simula coincidencias", async () => {
  const solicitudes = [], errores = [];
  const cliente = { consultarCuadroRRHH: async (solicitud) => {
    solicitudes.push(solicitud);
    return { generada_en: "2026-10-01T09:00:00Z", expedientes: [], hay_mas: false };
  } };
  const raiz = raizFalsa();
  const montaje = await montarCuadroContratacionLigero({ raiz, cliente, idioma: "es",
    filtroLista: { mostrar: "incidencia" }, abrirDetalle: async () => {},
    mostrarError: (_raiz, datos) => errores.push(datos) });
  assert.equal(solicitudes.length, 1);
  assert.equal(solicitudes[0].filtros.estado_clave, "incidencia");
  assert.equal(errores.length, 0);
  montaje.desmontar();

  const raizPlazo = raizFalsa();
  const otro = await montarCuadroContratacionLigero({ raiz: raizPlazo, cliente, idioma: "es",
    filtroLista: { mostrar: "vence_hoy" }, abrirDetalle: async () => {},
    mostrarError: (_raiz, datos) => errores.push(datos) });
  assert.equal(solicitudes.length, 1, "no consulta ni filtra localmente una promesa de conjunto sin puerto");
  assert.equal(errores.at(-1).error.codigo, "filtro_servidor_no_disponible");
  assert.match(errores.at(-1).mensaje, /no se puede aplicar a todas las páginas/u);
  await errores.at(-1).reintentar();
  assert.equal(solicitudes.length, 2);
  assert.equal(solicitudes[1].filtros.estado_clave, "");
  otro.desmontar();
});

test("una señal ya abortada no consulta, modifica DOM ni instala eventos", async () => {
  const controlador = new AbortController(), raiz = raizFalsa();
  controlador.abort();
  let consultas = 0;
  const montaje = await montarCuadroContratacionLigero({ raiz, signal: controlador.signal,
    cliente: { consultarCuadroRRHH: async () => { consultas++; } },
    abrirDetalle: async () => {}, mostrarError: () => {} });
  assert.equal(consultas, 0);
  assert.equal(raiz.innerHTML, "");
  assert.equal(raiz.eventos.size, 0);
  montaje.desmontar();
});

test("buscar encuentra una fila fuera de la primera página sin robar foco ni reutilizar cursor", async () => {
  const raiz = raizFalsa(), solicitudes = [];
  const resultados = { outerHTML: "" };
  raiz.querySelector = (selector) => selector === "[data-ct-exp-resultados]" ? resultados : null;
  const cliente = { consultarCuadroRRHH: async (solicitud) => {
    solicitudes.push(solicitud);
    return solicitud.filtros.texto === "objetivo"
      ? { generada_en: "2026-10-01T09:00:00Z", expedientes: [fila], hay_mas: false }
      : { generada_en: "2026-10-01T09:00:00Z", expedientes: [], hay_mas: true,
        cursor_siguiente: "cursor_sintetico_de_pagina" };
  } };
  const previo = globalThis.FormData;
  globalThis.FormData = class { constructor(formulario) { this.campos = formulario.campos; }
    entries() { return Object.entries(this.campos); } };
  try {
    const montaje = await montarCuadroContratacionLigero({ raiz, cliente, idioma: "es",
      abrirDetalle: async () => {}, mostrarError: (_raiz, datos) => { throw datos.error; } });
    const htmlInicial = raiz.innerHTML;
    const formulario = { campos: { texto: "objetivo", fase: "", centro: "", categoria: "", mostrar: "en_tramite" },
      elements: { namedItem: () => ({ value: "" }) } };
    raiz.eventos.get("input")({ type: "input", target: { name: "texto",
      closest: () => formulario } });
    await new Promise((resolver) => setTimeout(resolver, 310));
    assert.equal(solicitudes.length, 2);
    assert.deepEqual(solicitudes[1].filtros, { texto: "objetivo", estado_clave: "", fase_clave: "" });
    assert.equal(solicitudes[1].paginacion.cursor, "");
    assert.match(resultados.outerHTML, /2026\/CT-0001/u);
    assert.equal(raiz.innerHTML, htmlInicial, "no sustituye el buscador mientras se escribe");
    montaje.desmontar();
  } finally { globalThis.FormData = previo; }
});

test("pasar de página usa su cursor y devuelve el foco al título cuando desaparece Siguiente", async () => {
  const raiz = raizFalsa(), solicitudes = [];
  let focoTitulo = 0, tabindex = "";
  raiz.querySelector = (selector) => selector === "#ct-exp-lista-titulo-panel"
    ? { setAttribute: (_nombre, valor) => { tabindex = valor; }, focus: () => { focoTitulo++; } } : null;
  const montaje = await montarCuadroContratacionLigero({ raiz, idioma: "es",
    abrirDetalle: async () => {}, mostrarError: (_raiz, datos) => { throw datos.error; },
    cliente: { consultarCuadroRRHH: async (solicitud) => {
      solicitudes.push(solicitud);
      return solicitud.paginacion.cursor
        ? { generada_en: "2026-10-01T09:00:00Z", expedientes: [], hay_mas: false }
        : { generada_en: "2026-10-01T09:00:00Z", expedientes: [], hay_mas: true,
          cursor_siguiente: "cursor_sintetico_de_pagina" };
    } },
  });
  await raiz.eventos.get("click")({ target: { closest: () => ({ dataset: { ctPagina: "siguiente" } }) } });
  assert.equal(solicitudes.length, 2);
  assert.equal(solicitudes[1].paginacion.cursor, "cursor_sintetico_de_pagina");
  assert.equal(focoTitulo, 1);
  assert.equal(tabindex, "-1");
  montaje.desmontar();
});
