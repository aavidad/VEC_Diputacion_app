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
    nombreCentro: () => "Centro solicitante",
    nombreCategoria: () => "Auxiliar",
    abrirDetalle: async (datos) => { abiertos.push(datos); },
    mostrarError: (_raiz, datos) => { errores.push(datos); },
  });
  assert.equal(errores.length, 0, errores[0]?.error?.stack);
  assert.equal(consultas.length, 1);
  assert.equal(consultas[0].solicitud.paginacion.limite, 100);
  assert.match(raiz.innerHTML, /2026\/CT-0001/u);
  assert.match(raiz.innerHTML, />Centro solicitante<small>Auxiliar<\/small>/u);
  assert.match(raiz.innerHTML, /aria-label="Abrir el expediente 2026\/CT-0001"/u);
  assert.doesNotMatch(raiz.innerHTML, /title="centro:rpt:600"/u);
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
  assert.match(raiz.innerHTML, /Ninguna petición cumple estos filtros/u);
  assert.match(raiz.innerHTML, /name="mostrar"/u);
  montaje.desmontar();

  const raizPlazo = raizFalsa();
  const otro = await montarCuadroContratacionLigero({ raiz: raizPlazo, cliente, idioma: "es",
    filtroLista: { mostrar: "vence_hoy" }, abrirDetalle: async () => {},
    mostrarError: (_raiz, datos) => errores.push(datos) });
  assert.equal(solicitudes.length, 1, "no consulta ni filtra localmente una promesa de conjunto sin puerto");
  assert.equal(errores.at(-1).error.codigo, "filtro_servidor_no_disponible");
  assert.match(errores.at(-1).mensaje, /No se puede aplicar este filtro/u);
  assert.equal(errores.at(-1).quitarFiltroEtiqueta, "Quitar filtro");
  assert.equal(errores.at(-1).reintentar, undefined);
  assert.equal(solicitudes.length, 1, "la lista sin filtro espera un clic explícito");
  await errores.at(-1).quitarFiltro();
  assert.equal(solicitudes.length, 2);
  assert.deepEqual(solicitudes[1].filtros, { texto: "", estado_clave: "", fase_clave: "" });
  otro.desmontar();
});

test("plazo V2 pide la página y el total del mismo filtro, conserva el cursor y no lee filas ajenas", async () => {
  const raiz = raizFalsa(), solicitudes = [], errores = [];
  const filas = Array.from({ length: 101 }, (_, indice) => ({ ...fila,
    expediente_ref: `expediente:ct:${String(indice + 1).padStart(3, "0")}`,
    numero_visible: `2026/CT-${String(indice + 1).padStart(4, "0")}`,
    plazo_fase: { estado: "vencido", ultimo_dia: "2026-09-30" } }));
  const cliente = {
    consultarCuadroRRHH: async () => { throw new Error("no debe degradar un plazo a V1"); },
    consultarCuadroRRHHV2: async (solicitud) => {
      solicitudes.push(solicitud);
      return { esquema: "vec.contratacion-temporal.cuadro-rrhh.v2",
        generada_en: "2026-10-01T09:00:00Z", expedientes: solicitud.paginacion.cursor ? filas.slice(100) : filas.slice(0, 100),
        hay_mas: !solicitud.paginacion.cursor,
        ...(!solicitud.paginacion.cursor ? { cursor_siguiente: "cursor_plazo" } : {}),
        totales: { total: 101, en_tramitacion: 101, con_incidencia: 0, en_llamamiento: 0 },
        resumen: { en_tramite: 101, con_incidencia: 0, vencidos: 101, vencen_hoy: 0,
          vencen_semana: 0, sin_calcular: 0, por_fase: { analisis: 101 } } };
    },
  };
  const montaje = await montarCuadroContratacionLigero({ raiz, cliente, idioma: "es",
    filtroServidorRuta: { texto: "", estado_clave: "", fase_clave: "", plazo_estado: "vencido" },
    abrirDetalle: async () => {}, mostrarError: (_raiz, datos) => errores.push(datos) });
  assert.equal(errores.length, 0, errores[0]?.error?.stack);
  assert.equal(solicitudes[0].esquema, "vec.contratacion-temporal.cuadro-rrhh.v2");
  assert.deepEqual(solicitudes[0].filtros,
    { texto: "", estado_clave: "", fase_clave: "", plazo_estado: "vencido" });
  assert.equal(solicitudes[0].resumen, true);
  assert.equal((raiz.innerHTML.match(/data-ct-exp-abrir=/gu) ?? []).length, 100);
  assert.match(raiz.innerHTML, /100 de 101 peticiones/u);
  assert.doesNotMatch(raiz.innerHTML, /Recuento parcial/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-exp-busqueda-parcial/u);
  await raiz.eventos.get("click")({ target: { closest: () => ({ dataset: { ctPagina: "siguiente" } }) } });
  assert.equal(solicitudes[1].paginacion.cursor, "cursor_plazo");
  assert.deepEqual(solicitudes[1].filtros, solicitudes[0].filtros);
  assert.equal((raiz.innerHTML.match(/data-ct-exp-abrir=/gu) ?? []).length, 1);
  assert.match(raiz.innerHTML, /1 de 101 peticiones/u);
  montaje.desmontar();
});

test("los errores HTTP de V2 no abren V1 y un 503 reintenta el mismo filtro", async () => {
  for (const estado of [400, 401, 403, 503]) {
    const raiz = raizFalsa(), errores = [];
    let lecturasV1 = 0, lecturasV2 = 0;
    const montaje = await montarCuadroContratacionLigero({ raiz, idioma: "es",
      filtroServidorRuta: { texto: "", estado_clave: "", fase_clave: "", plazo_estado: "vence_hoy" },
      cliente: { consultarCuadroRRHH: async () => { lecturasV1++; },
        consultarCuadroRRHHV2: async () => {
          lecturasV2++;
          throw Object.assign(new Error("fallo V2"), { estado });
        } },
      abrirDetalle: async () => {}, mostrarError: (_raiz, datos) => errores.push(datos) });
    assert.equal(lecturasV1, 0, estado);
    assert.equal(lecturasV2, 1, estado);
    assert.match(errores[0].error.message, /fallo V2/u);
    if (estado === 400) {
      assert.equal(errores[0].quitarFiltroEtiqueta, "Quitar filtro");
      assert.equal(typeof errores[0].quitarFiltro, "function");
      assert.equal(errores[0].reintentar, undefined);
    } else assert.equal(errores[0].quitarFiltro, undefined, estado);
    assert.equal(raiz.innerHTML, "");
    if (estado === 503) {
      await errores[0].reintentar();
      assert.equal(lecturasV1, 0);
      assert.equal(lecturasV2, 2);
    }
    montaje.desmontar();
  }
});

test("la primera página V2 habilita filtros de plazo sin sondeo adicional", async () => {
  const raiz = raizFalsa();
  let v1 = 0, v2 = 0;
  const montaje = await montarCuadroContratacionLigero({ raiz, idioma: "es",
    cliente: {
      consultarCuadroRRHH: async () => {
        v1++;
        return { esquema: "vec.contratacion-temporal.cuadro-rrhh.v2",
          generada_en: "2026-10-08T10:00:00Z", expedientes: [], hay_mas: false,
          totales: { total: 0, en_tramitacion: 0, con_incidencia: 0, en_llamamiento: 0 },
          resumen: { en_tramite: 0, con_incidencia: 0, vencidos: 0, vencen_hoy: 0,
            vencen_semana: 0, sin_calcular: 0, por_fase: {} } };
      },
      consultarCuadroRRHHV2: async () => { v2++; throw new Error("sin filtro pedido"); },
    },
    abrirDetalle: async () => {}, mostrarError: (_raiz, datos) => { throw datos.error; } });
  assert.equal(v1, 1);
  assert.equal(v2, 0);
  assert.match(raiz.innerHTML, /value="vencidos"/u);
  assert.match(raiz.innerHTML, /value="vencen_semana"/u);
  montaje.desmontar();
});

test("incidencia conserva el mismo total autorizado a través de 100 más 1 filas paginadas", async () => {
  const raiz = raizFalsa(), solicitudes = [];
  const filas = Array.from({ length: 101 }, (_, indice) => ({ ...fila,
    expediente_ref: `expediente:ct:${String(indice + 1).padStart(3, "0")}`,
    numero_visible: `2026/CT-${String(indice + 1).padStart(4, "0")}`,
    estado_clave: "incidencia" }));
  const totales = { total: 101, en_tramitacion: 0, con_incidencia: 101, en_llamamiento: 0 };
  const resumen = { en_tramite: 101, con_incidencia: 101, vencidos: 0, vencen_hoy: 0,
    vencen_semana: 0, sin_calcular: 0, por_fase: { analisis: 101 } };
  const montaje = await montarCuadroContratacionLigero({ raiz, idioma: "es",
    filtroLista: { mostrar: "incidencia" }, abrirDetalle: async () => {},
    mostrarError: (_raiz, datos) => { throw datos.error; },
    cliente: { consultarCuadroRRHH: async (solicitud) => {
      solicitudes.push(solicitud);
      assert.deepEqual(solicitud.filtros, { texto: "", estado_clave: "incidencia", fase_clave: "" });
      return { generada_en: "2026-10-01T09:00:00Z", totales, resumen,
        expedientes: solicitud.paginacion.cursor ? filas.slice(100) : filas.slice(0, 100),
        hay_mas: !solicitud.paginacion.cursor,
        ...(!solicitud.paginacion.cursor ? { cursor_siguiente: "cursor_de_incidentes" } : {}) };
    } },
  });
  assert.equal(solicitudes.length, 1);
  assert.equal(solicitudes[0].paginacion.limite, 100);
  assert.equal((raiz.innerHTML.match(/data-ct-exp-abrir=/gu) ?? []).length, 100);
  assert.match(raiz.innerHTML, /100 de 101 peticiones/u);
  await raiz.eventos.get("click")({ target: { closest: () => ({ dataset: { ctPagina: "siguiente" } }) } });
  assert.equal(solicitudes.length, 2);
  assert.equal(solicitudes[1].paginacion.cursor, "cursor_de_incidentes");
  assert.equal((raiz.innerHTML.match(/data-ct-exp-abrir=/gu) ?? []).length, 1);
  assert.match(raiz.innerHTML, /1 de 101 peticiones/u);
  montaje.desmontar();
});

test("quitar incidencia y buscar por prefijo notifican solo filtros confirmados por el servidor", async () => {
  const raiz = raizFalsa(), solicitudes = [], aplicados = [];
  const previo = globalThis.FormData;
  globalThis.FormData = class { constructor(formulario) { this.campos = formulario.campos; }
    entries() { return Object.entries(this.campos); } };
  try {
    const montaje = await montarCuadroContratacionLigero({ raiz, idioma: "es",
      filtroServidorRuta: { texto: "", estado_clave: "incidencia", fase_clave: "" },
      alCambiarFiltroLista: (filtro) => aplicados.push({ ...filtro }),
      abrirDetalle: async () => {}, mostrarError: (_raiz, datos) => { throw datos.error; },
      cliente: { consultarCuadroRRHH: async (solicitud) => {
        solicitudes.push(solicitud);
        return { generada_en: "2026-10-01T09:00:00Z", expedientes: [fila], hay_mas: false };
      } },
    });
    assert.deepEqual(aplicados, [{ texto: "", estado_clave: "incidencia", fase_clave: "" }]);
    assert.match(raiz.innerHTML, /name="mostrar"[^>]*>[\s\S]*?value="incidencia" selected/u);
    assert.doesNotMatch(raiz.innerHTML, /name="centro"|name="categoria"|Vencen hoy/u);
    await raiz.eventos.get("click")({ target: { closest: () => ({ dataset: { ctExpQuitarFiltro: "mostrar" } }) } });
    assert.equal(solicitudes.length, 2);
    assert.deepEqual(aplicados.at(-1), { texto: "", estado_clave: "", fase_clave: "" });
    const formulario = { campos: { texto: "2026/CT", fase: "", mostrar: "todas" } };
    raiz.eventos.get("change")({ type: "change", target: { closest: () => formulario } });
    await new Promise((resolver) => setImmediate(resolver));
    assert.equal(solicitudes.length, 3);
    assert.deepEqual(aplicados.at(-1), { texto: "2026/CT", estado_clave: "", fase_clave: "" });
    assert.equal(aplicados.length, 3, "una notificación por respuesta válida, sin POST añadido");
    montaje.desmontar();
  } finally { globalThis.FormData = previo; }
});

test("una fase exacta puede contener terminadas: el listado usa su total, no el recuento activo de Inicio", async () => {
  const raiz = raizFalsa(), solicitudes = [];
  const activa = { ...fila, fase_clave: "nombramiento", estado_clave: "en_curso" };
  const terminada = { ...fila, expediente_ref: "expediente:ct:002", numero_visible: "2026/CT-0002",
    fase_clave: "nombramiento", estado_clave: "completado" };
  const montaje = await montarCuadroContratacionLigero({ raiz, idioma: "es",
    filtroServidorRuta: { texto: "", estado_clave: "", fase_clave: "nombramiento" },
    abrirDetalle: async () => {}, mostrarError: (_raiz, datos) => { throw datos.error; },
    cliente: { consultarCuadroRRHH: async (solicitud) => {
      solicitudes.push(solicitud);
      return { generada_en: "2026-10-01T09:00:00Z", expedientes: [activa, terminada], hay_mas: false,
        totales: { total: 2, en_tramitacion: 1, con_incidencia: 0, en_llamamiento: 0 },
        resumen: { en_tramite: 1, con_incidencia: 0, vencidos: 0, vencen_hoy: 0,
          vencen_semana: 0, sin_calcular: 0, por_fase: { nombramiento: 1 } } };
    } },
  });
  assert.deepEqual(solicitudes[0].filtros, { texto: "", estado_clave: "", fase_clave: "nombramiento" });
  assert.match(raiz.innerHTML, /<h3 class="ct-exp-lista-titulo">Peticiones<\/h3>/u);
  assert.match(raiz.innerHTML, /2 de 2 peticiones/u);
  assert.equal((raiz.innerHTML.match(/data-ct-exp-abrir=/gu) ?? []).length, 2);
  assert.match(raiz.innerHTML, /value="nombramiento" selected/u);
  const fases = raiz.innerHTML.match(/<select name="fase">([\s\S]*?)<\/select>/u)?.[1] ?? "";
  assert.deepEqual([...fases.matchAll(/<option value="([^"]*)"/gu)].map((m) => m[1]),
    ["", "solicitud", "analisis", "preparacion", "fiscalizacion", "llamamiento", "nombramiento", "incorporacion", "cierre"]);
  const estados = raiz.innerHTML.match(/<select name="mostrar">([\s\S]*?)<\/select>/u)?.[1] ?? "";
  assert.deepEqual([...estados.matchAll(/<option value="([^"]*)"/gu)].map((m) => m[1]),
    ["todas", "incidencia", "espera"]);
  assert.match(raiz.innerHTML, /placeholder="Prefijo del número de expediente"/u);
  assert.doesNotMatch(raiz.innerHTML, /1 petición en trámite/u);
  montaje.desmontar();
});

test("las filas de preparación y cierre conservan el rótulo del filtro en ES y EN", async () => {
  for (const [idioma, faseEtiqueta, preparacion, cierre] of [
    ["es", "Fase", "Preparación", "Cierre"], ["en", "Stage", "Preparation", "Closure"],
  ]) {
    const raiz = raizFalsa();
    const montaje = await montarCuadroContratacionLigero({ raiz, idioma,
      abrirDetalle: async () => {}, mostrarError: (_raiz, datos) => { throw datos.error; },
      cliente: { consultarCuadroRRHH: async () => ({ generada_en: "2026-10-01T09:00:00Z", hay_mas: false,
        expedientes: [{ ...fila, fase_clave: "preparacion" },
          { ...fila, expediente_ref: "expediente:ct:002", numero_visible: "2026/CT-0002", fase_clave: "cierre" }] }) },
    });
    assert.match(raiz.innerHTML, new RegExp(`<td data-etiqueta="${faseEtiqueta}">${preparacion}<\\/td>`, "u"));
    assert.match(raiz.innerHTML, new RegExp(`<td data-etiqueta="${faseEtiqueta}">${cierre}`, "u"));
    assert.doesNotMatch(raiz.innerHTML, new RegExp(`<td data-etiqueta="${faseEtiqueta}">(?:preparacion|cierre)`, "u"));
    montaje.desmontar();
  }
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
    const formulario = { campos: { texto: "objetivo", fase: "", centro: "", categoria: "", mostrar: "todas" },
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
    const formulario = { campos: { texto: "objetivo", fase: "", centro: "", categoria: "", mostrar: "todas" } };
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

test("el título y resultado de la lista usan el total autorizado, no las filas de la página", async () => {
  const raiz = raizFalsa(), solicitudes = [];
  const totales = { total: 5071, en_tramitacion: 100, con_incidencia: 0, en_llamamiento: 0 };
  const resumen = { en_tramite: 5071, con_incidencia: 0, vencidos: 0, vencen_hoy: 0,
    vencen_semana: 0, sin_calcular: 0, por_fase: { solicitud: 5071 } };
  const montaje = await montarCuadroContratacionLigero({ raiz, idioma: "es",
    abrirDetalle: async () => {}, mostrarError: (_raiz, datos) => { throw datos.error; },
    cliente: { consultarCuadroRRHH: async (solicitud) => {
      solicitudes.push(solicitud);
      return { generada_en: "2026-10-01T09:00:00Z", totales, resumen,
        expedientes: solicitud.paginacion.cursor ? [fila] : [fila, { ...fila,
          expediente_ref: "expediente:ct:002", numero_visible: "2026/CT-0002" }],
        hay_mas: !solicitud.paginacion.cursor,
        ...(!solicitud.paginacion.cursor ? { cursor_siguiente: "cursor_sintetico_de_pagina" } : {}) };
    } },
  });
  assert.match(raiz.innerHTML, /<h3 class="ct-exp-lista-titulo">Peticiones<\/h3>/u);
  assert.match(raiz.innerHTML, /2 de 5071 peticiones/u);
  assert.equal(solicitudes[0].resumen, true, "el resumen viaja en la misma POST de la página");
  assert.doesNotMatch(raiz.innerHTML, /2 peticiones en trámite/u);
  await raiz.eventos.get("click")({ target: { closest: () => ({ dataset: { ctPagina: "siguiente" } }) } });
  assert.equal(solicitudes[1].paginacion.cursor, "cursor_sintetico_de_pagina");
  assert.match(raiz.innerHTML, /<h3 class="ct-exp-lista-titulo">Peticiones<\/h3>/u);
  assert.match(raiz.innerHTML, /1 de 5071 peticiones/u);
  montaje.desmontar();
});

test("sin total del servidor no presenta el tamaño de una página como total del conjunto", async () => {
  const raiz = raizFalsa();
  const montaje = await montarCuadroContratacionLigero({ raiz, idioma: "es",
    abrirDetalle: async () => {}, mostrarError: (_raiz, datos) => { throw datos.error; },
    cliente: { consultarCuadroRRHH: async (solicitud) => ({ generada_en: "2026-10-01T09:00:00Z",
      expedientes: [fila], hay_mas: !solicitud.paginacion.cursor,
      ...(!solicitud.paginacion.cursor ? { cursor_siguiente: "cursor_sintetico_de_pagina" } : {}) }) },
  });
  assert.match(raiz.innerHTML, /Expedientes de peticiones de personal temporal/u);
  assert.doesNotMatch(raiz.innerHTML, /1 petición en trámite/u);
  assert.match(raiz.innerHTML, /Peticiones mostradas: 1/u);
  assert.doesNotMatch(raiz.innerHTML, /1 de 1 peticiones/u);
  await raiz.eventos.get("click")({ target: { closest: () => ({ dataset: { ctPagina: "siguiente" } }) } });
  assert.match(raiz.innerHTML, /Peticiones mostradas: 1/u);
  assert.doesNotMatch(raiz.innerHTML, /1 de 1 peticiones/u);
  montaje.desmontar();
});

test("si llegan totales sin el resumen pedido, la lista ofrece error en vez de un recuento ambiguo", async () => {
  const raiz = raizFalsa(), errores = [];
  const montaje = await montarCuadroContratacionLigero({ raiz, idioma: "es",
    abrirDetalle: async () => {}, mostrarError: (_raiz, datos) => errores.push(datos),
    cliente: { consultarCuadroRRHH: async () => ({ generada_en: "2026-10-01T09:00:00Z",
      expedientes: [fila], hay_mas: false,
      totales: { total: 2, en_tramitacion: 1, con_incidencia: 0, en_llamamiento: 0 } }) },
  });
  assert.equal(errores.length, 1);
  assert.equal(raiz.innerHTML, "");
  montaje.desmontar();
});
