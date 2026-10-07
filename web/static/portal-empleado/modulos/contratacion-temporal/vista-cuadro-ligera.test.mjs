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

function raizConBuscador() {
  const raiz = raizFalsa();
  const documento = { activeElement: null };
  let html = "";
  let buscador = null;
  raiz.ownerDocument = documento;
  raiz.contains = (nodo) => nodo === buscador;
  Object.defineProperty(raiz, "innerHTML", {
    get: () => html,
    set: (contenido) => {
      html = contenido;
      buscador = html.includes('name="texto"') ? {
        name: "texto", value: html.match(/name="texto" value="([^"]*)"/u)?.[1] ?? "",
        selectionStart: 0, selectionEnd: 0, selectionDirection: "none",
        closest: (selector) => selector === "[data-ct-exp-filtros-locales]" ? {} : null,
        focus() { documento.activeElement = this; },
        setSelectionRange(inicio, fin, direccion) {
          this.selectionStart = inicio; this.selectionEnd = fin; this.selectionDirection = direccion;
        },
      } : null;
    },
  });
  raiz.querySelector = (selector) => selector === '[data-ct-exp-filtros-locales] [name="texto"]'
    ? buscador : null;
  return { raiz, documento, buscador: () => buscador };
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
  const { raiz, documento, buscador } = raizConBuscador(), solicitudes = [];
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
    const controlInicial = buscador();
    controlInicial.value = "objetivo";
    controlInicial.focus();
    controlInicial.setSelectionRange(2, 6, "forward");
    const formulario = { campos: { texto: "objetivo", fase: "", centro: "", categoria: "", mostrar: "en_tramite" },
      elements: { namedItem: () => ({ value: "" }) } };
    raiz.eventos.get("input")({ type: "input", target: { name: "texto",
      closest: () => formulario } });
    await new Promise((resolver) => setTimeout(resolver, 310));
    assert.equal(solicitudes.length, 2);
    assert.deepEqual(solicitudes[1].filtros, { texto: "objetivo", estado_clave: "", fase_clave: "" });
    assert.equal(solicitudes[1].paginacion.cursor, "");
    assert.match(raiz.innerHTML, /2026\/CT-0001/u);
    assert.doesNotMatch(raiz.innerHTML, /data-ct-pagina="siguiente"/u);
    assert.doesNotMatch(raiz.innerHTML, /data-ct-exp-busqueda-parcial/u,
      "el texto ya filtrado en el servidor no es una búsqueda parcial local");
    assert.equal(documento.activeElement, buscador());
    assert.equal(buscador().value, "objetivo");
    assert.deepEqual([buscador().selectionStart, buscador().selectionEnd,
      buscador().selectionDirection], [2, 6, "forward"]);
    montaje.desmontar();
  } finally { globalThis.FormData = previo; }
});

test("buscar vuelve a mostrar Siguiente si la respuesta filtrada tiene otra página", async () => {
  const { raiz, documento, buscador } = raizConBuscador(), solicitudes = [];
  const previo = globalThis.FormData;
  globalThis.FormData = class { constructor(formulario) { this.campos = formulario.campos; }
    entries() { return Object.entries(this.campos); } };
  try {
    const montaje = await montarCuadroContratacionLigero({ raiz, idioma: "es",
      abrirDetalle: async () => {}, mostrarError: (_raiz, datos) => { throw datos.error; },
      cliente: { consultarCuadroRRHH: async (solicitud) => {
        solicitudes.push(solicitud);
        return { generada_en: "2026-10-01T09:00:00Z", expedientes: [fila],
          hay_mas: solicitud.filtros.texto === "objetivo" && !solicitud.paginacion.cursor,
          ...(solicitud.filtros.texto === "objetivo" && !solicitud.paginacion.cursor
            ? { cursor_siguiente: "cursor_filtrado" } : {}) };
      } },
    });
    assert.doesNotMatch(raiz.innerHTML, /data-ct-pagina="siguiente"/u);
    buscador().value = "objetivo";
    buscador().focus();
    const formulario = { campos: { texto: "objetivo", fase: "", centro: "", categoria: "", mostrar: "en_tramite" } };
    formulario.elements = { namedItem: () => ({
      set value(valor) { formulario.campos.mostrar = valor; },
      get value() { return formulario.campos.mostrar; },
    }) };
    const buscadorEvento = { name: "texto", closest: () => formulario };
    raiz.eventos.get("input")({ type: "input", target: buscadorEvento });
    raiz.eventos.get("change")({ type: "change", target: buscadorEvento });
    await new Promise((resolver) => setTimeout(resolver, 310));
    assert.equal(solicitudes.length, 2, "input y change del mismo texto hacen una sola lectura");
    assert.match(raiz.innerHTML, /data-ct-pagina="siguiente"/u);
    assert.match(raiz.innerHTML, /lista-parcial/u);
    assert.equal(documento.activeElement, buscador());
    raiz.eventos.get("change")({ type: "change", target: buscadorEvento });
    assert.equal(solicitudes.length, 2, "salir del buscador no repite la lectura ya completada");
    await raiz.eventos.get("click")({ target: { closest: () => ({ dataset: { ctPagina: "siguiente" } }) } });
    assert.equal(solicitudes.length, 3, "pasar de página hace solo su propia lectura");
    assert.equal(solicitudes[2].paginacion.cursor, "cursor_filtrado");
    assert.equal(solicitudes[2].filtros.texto, "objetivo");
    assert.doesNotMatch(raiz.innerHTML, /data-ct-pagina="siguiente"/u);
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

test("v2 usa la lista administrativa exacta del enlace y conserva el cursor sin refiltrar filas", async () => {
  const solicitudes = [], errores = [], enlaces = [], raiz = raizFalsa();
  const montaje = await montarCuadroContratacionLigero({
    raiz, idioma: "es", contratoFiltros: "v2",
    filtroLista: { texto: "objetivo", fase: "solicitud", centro: "centro:rpt:600",
      categoria: "categoria:auxiliar", mostrar: "incidencia" },
    fasesClaveInicial: ["solicitud_registrada", "solicitud"],
    alCambiarFiltroLista: (filtro, opciones) => enlaces.push({ filtro, opciones }),
    abrirDetalle: async () => {}, mostrarError: (_raiz, datos) => errores.push(datos),
    cliente: {
      consultarCuadroRRHH: () => { throw new Error("v1 no debe consultarse"); },
      consultarCuadroRRHHV2: async (solicitud) => {
        solicitudes.push(solicitud);
        const esSiguiente = Boolean(solicitud.paginacion.cursor);
        return { generada_en: "2026-10-01T09:00:00Z", expedientes: [fila],
          hay_mas: !esSiguiente,
          ...(!esSiguiente ? { cursor_siguiente: "cursor_filtrado" } : {}),
          totales: { total: 2 },
          resumen: { por_fase: { solicitud: 1, solicitud_registrada: 1 } },
        };
      },
    },
  });
  assert.equal(errores.length, 0, errores[0]?.error?.stack);
  assert.equal(solicitudes.length, 1, "el enlace directo hace una sola lectura de la página filtrada");
  assert.deepEqual(solicitudes[0].filtros, { texto: "objetivo", centro_ref: "centro:rpt:600",
    categoria_ref: "categoria:auxiliar", estados_clave: ["incidencia"],
    fases_clave: ["solicitud", "solicitud_registrada"] });
  assert.match(raiz.innerHTML, /1 de 2 peticiones/u);
  assert.match(raiz.innerHTML, /2026\/CT-0001/u,
    "una coincidencia del servidor no se oculta por el texto ausente de la fila visible");
  assert.match(raiz.innerHTML, /data-ct-pagina="siguiente"/u);
  await raiz.eventos.get("click")({ target: { closest: () => ({ dataset: { ctPagina: "siguiente" } }) } });
  assert.equal(solicitudes.length, 2);
  assert.equal(solicitudes[1].paginacion.cursor, "cursor_filtrado");
  assert.deepEqual(solicitudes[1].filtros.fases_clave, solicitudes[0].filtros.fases_clave);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-pagina="siguiente"/u);
  await raiz.eventos.get("click")({ target: { closest: () => ({ dataset: { ctExpQuitarFiltro: "texto" } }) } });
  assert.equal(solicitudes.length, 3, "quitar texto con fase conserva la lista exacta y hace una lectura");
  assert.equal(solicitudes.at(-1).paginacion.cursor, "");
  assert.deepEqual(enlaces, [{ filtro: { texto: "", fase: "solicitud", centro: "centro:rpt:600",
    categoria: "categoria:auxiliar", mostrar: "incidencia" },
    opciones: { reemplazar: false, fasesClave: ["solicitud", "solicitud_registrada"] } }]);
  montaje.desmontar();
});

test("v2 no convierte una fase visual sin claves administrativas en filtro vacío", async () => {
  let consultas = 0;
  const errores = [];
  const montaje = await montarCuadroContratacionLigero({ raiz: raizFalsa(), idioma: "es",
    contratoFiltros: "v2", filtroLista: { fase: "solicitud" },
    abrirDetalle: async () => {}, mostrarError: (_raiz, datos) => errores.push(datos),
    cliente: { consultarCuadroRRHH: () => { throw new Error("v1 no debe consultarse"); },
      consultarCuadroRRHHV2: async () => { consultas++; } },
  });
  assert.equal(consultas, 0);
  assert.equal(errores[0].error.codigo, "filtro_servidor_no_disponible");
  montaje.desmontar();
});

test("v2 ignora el resumen tardío de una búsqueda cancelada antes de elegir fase", async () => {
  const raiz = raizFalsa(), solicitudes = [], errores = [];
  let resolverAntigua;
  const pagina = (fase, total) => ({ generada_en: "2026-10-01T09:00:00Z",
    expedientes: [fila], hay_mas: false, totales: { total },
    resumen: { por_fase: { [fase]: total } } });
  const previo = globalThis.FormData;
  globalThis.FormData = class { constructor(formulario) { this.campos = formulario.campos; }
    entries() { return Object.entries(this.campos); } };
  try {
    const montaje = await montarCuadroContratacionLigero({ raiz, idioma: "es", contratoFiltros: "v2",
      abrirDetalle: async () => {}, mostrarError: (_raiz, datos) => errores.push(datos),
      cliente: { consultarCuadroRRHH: () => { throw new Error("v1 no debe consultarse"); },
        consultarCuadroRRHHV2: async (solicitud) => {
          solicitudes.push(solicitud);
          if (solicitud.filtros.texto === "antigua") {
            return new Promise((resolver) => { resolverAntigua = resolver; });
          }
          return solicitud.filtros.texto === "nueva" ? pagina("analisis", 2) : pagina("solicitud", 1);
        } },
    });
    const formulario = { campos: { texto: "antigua", fase: "", centro: "", categoria: "", mostrar: "en_tramite" } };
    const campo = { name: "texto", closest: () => formulario };
    raiz.eventos.get("change")({ type: "change", target: campo });
    await new Promise((resolver) => setTimeout(resolver, 0));
    assert.equal(typeof resolverAntigua, "function");
    formulario.campos.texto = "nueva";
    raiz.eventos.get("change")({ type: "change", target: campo });
    await new Promise((resolver) => setTimeout(resolver, 0));
    resolverAntigua(pagina("solicitud", 9));
    await new Promise((resolver) => setTimeout(resolver, 0));
    formulario.campos.fase = "analisis_rrhh";
    raiz.eventos.get("change")({ type: "change", target: { name: "fase", closest: () => formulario } });
    await new Promise((resolver) => setTimeout(resolver, 0));
    assert.deepEqual(solicitudes.at(-1).filtros.fases_clave, ["analisis"]);
    assert.equal(errores.length, 0, errores[0]?.error?.stack);
    montaje.desmontar();
  } finally { globalThis.FormData = previo; }
});

test("v2 rechaza un plazo sin SQL autorizado antes de consultar el cuadro", async () => {
  const errores = [];
  let consultas = 0;
  const montaje = await montarCuadroContratacionLigero({ raiz: raizFalsa(), idioma: "en",
    contratoFiltros: "v2", filtroLista: { mostrar: "vence_hoy" },
    abrirDetalle: async () => {}, mostrarError: (_raiz, datos) => errores.push(datos),
    cliente: { consultarCuadroRRHH: () => { throw new Error("v1 no debe consultarse"); },
      consultarCuadroRRHHV2: async () => { consultas++; } },
  });
  assert.equal(consultas, 0);
  assert.equal(errores[0].error.codigo, "filtro_servidor_no_disponible");
  montaje.desmontar();
});
