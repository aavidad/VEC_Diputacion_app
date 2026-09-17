/**
 * Pruebas unitarias del módulo de estadísticas de contratación temporal (G16 / C18).
 */

import test from "node:test";
import assert from "node:assert/strict";

import {
  ESQUEMA_ESTADISTICAS,
  PERIODOS_ESTADISTICAS,
  validarSerieEstadisticas,
  validarTotalesEstadisticas,
  validarRespuestaEstadisticas,
  generarCSVEstadisticas,
} from "./contrato-estadisticas.js";

import {
  RUTA_ESTADISTICAS,
  construirUrlEstadisticas,
  consultarEstadisticas,
} from "./cliente-http-estadisticas.js";

import {
  renderizarGraficoSVG,
  renderizarTablaEstadisticas,
  renderizarFormularioFiltros,
  renderizarVistaEstadisticas,
  montarVistaEstadisticas,
} from "./vista-estadisticas.js";

const DATOS_MUESTRA = {
  esquema: ESQUEMA_ESTADISTICAS,
  periodo: "mensual",
  desde: "2026-01-01",
  hasta: "2026-06-30",
  series: [
    {
      inicio: "2026-01-01",
      altas: 12,
      llamamientos: 25,
      formalizaciones: 10,
      cierres: 8,
      incidencias: 2,
    },
    {
      inicio: "2026-02-01",
      altas: 15,
      llamamientos: 30,
      formalizaciones: 14,
      cierres: 12,
      incidencias: 1,
    },
  ],
  totales: {
    altas: 27,
    llamamientos: 55,
    formalizaciones: 24,
    cierres: 20,
    incidencias: 3,
  },
};

const ENVELOPE_VALIDO = {
  data: DATOS_MUESTRA,
};

test("validarRespuestaEstadisticas: valida respuesta correcta y rechaza anomalías o datos personales", () => {
  const v = validarRespuestaEstadisticas(ENVELOPE_VALIDO);
  assert.equal(v.esquema, ESQUEMA_ESTADISTICAS);
  assert.equal(v.series.length, 2);
  assert.equal(v.totales.altas, 27);

  // Envelope ausente
  assert.throws(() => validarRespuestaEstadisticas(DATOS_MUESTRA));

  // Esquema inválido
  assert.throws(() => validarRespuestaEstadisticas({ data: { ...DATOS_MUESTRA, esquema: "invalido" } }));

  // Período inválido
  assert.throws(() => validarRespuestaEstadisticas({ data: { ...DATOS_MUESTRA, periodo: "diario" } }));

  // Fechas inválidas
  assert.throws(() => validarRespuestaEstadisticas({ data: { ...DATOS_MUESTRA, desde: "01/01/2026" } }));
  assert.throws(() => validarRespuestaEstadisticas({ data: { ...DATOS_MUESTRA, desde: "2026-06-30", hasta: "2026-01-01" } }));

  // Rechazo de datos personales o campos extra
  assert.throws(() => validarRespuestaEstadisticas({ data: { ...DATOS_MUESTRA, nombre: "Juan" } }));
  assert.throws(() => validarRespuestaEstadisticas({ data: { ...DATOS_MUESTRA, dni: "12345678A" } }));

  // Totales incongruentes con las series
  const totalesIncongruentes = {
    data: {
      ...DATOS_MUESTRA,
      totales: { altas: 99, llamamientos: 55, formalizaciones: 24, cierres: 20, incidencias: 3 },
    },
  };
  assert.throws(() => validarRespuestaEstadisticas(totalesIncongruentes));
});

test("generarCSVEstadisticas: genera formato delimitado por punto y coma con cabeceras y totales", () => {
  const csv = generarCSVEstadisticas(DATOS_MUESTRA);
  assert.ok(csv.includes("Periodo inicio;Altas;Llamamientos;Formalizaciones;Cierres;Incidencias"));
  assert.ok(csv.includes("2026-01-01;12;25;10;8;2"));
  assert.ok(csv.includes("2026-02-01;15;30;14;12;1"));
  assert.ok(csv.includes("TOTAL;27;55;24;20;3"));
});

test("construirUrlEstadisticas y cliente consultarEstadisticas", async () => {
  const url = construirUrlEstadisticas({ periodo: "anual", desde: "2025-01-01", hasta: "2025-12-31" });
  assert.equal(url, "/api/vec/contratacion-temporal/estadisticas?periodo=anual&desde=2025-01-01&hasta=2025-12-31");

  // Mock fetch exitoso
  const mockFetchOk = async (input) => {
    return {
      ok: true,
      status: 200,
      json: async () => ENVELOPE_VALIDO,
    };
  };

  const res = await consultarEstadisticas({ periodo: "mensual" }, { fetchImpl: mockFetchOk });
  assert.equal(res.ok, true);
  assert.equal(res.datos.esquema, ESQUEMA_ESTADISTICAS);
  assert.equal(res.datos.totales.llamamientos, 55);

  // Mock fetch 403
  const mockFetch403 = async () => ({
    ok: false,
    status: 403,
    json: async () => ({ error: "acceso_denegado" }),
  });
  const res403 = await consultarEstadisticas({ periodo: "mensual" }, { fetchImpl: mockFetch403 });
  assert.equal(res403.ok, false);
  assert.equal(res403.status, 403);
  assert.equal(res403.codigo, "acceso_denegado");
});

test("renderizarVistaEstadisticas y componentes HTML/SVG accesibles", () => {
  const html = renderizarVistaEstadisticas({
    estadoEstadisticas: {
      carga: "listo",
      datos: DATOS_MUESTRA,
      error: "",
    },
    filtros: { periodo: "mensual", desde: "2026-01-01", hasta: "2026-06-30" },
  });

  // Verificar que incluye tabla accesible
  assert.ok(html.includes("tabla-datos"));
  assert.ok(html.includes("<caption>"));
  assert.ok(html.includes("<tfoot>"));
  assert.ok(html.includes("TOTALES"));

  // Verificar gráfico SVG
  assert.ok(html.includes("grafico-svg-contenedor"));
  assert.ok(html.includes("<svg"));
  assert.ok(html.includes("Gráfico de evolución temporal de contratación"));
  assert.ok(html.includes("<rect"));

  // Verificar formulario y exportador CSV
  assert.ok(html.includes('data-ct-form="filtros-estadisticas"'));
  assert.ok(html.includes('data-ct-accion="exportar-csv"'));

  // Ausencia de "demo"
  assert.equal(/demo/i.test(html), false, "No debe contener la palabra demo");
});

test("montarVistaEstadisticas: ciclo de vida y montaje con cliente simulado", async () => {
  const eventos = new Map();
  let contenido = "";

  const raiz = {
    get innerHTML() { return contenido; },
    set innerHTML(val) { contenido = val; },
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
    },
    contains() { return true; },
    querySelector(selector) {
      return {
        focus() {},
        scrollIntoView() {},
      };
    },
    replaceChildren() { contenido = ""; },
  };

  const clienteMock = async () => ({
    ok: true,
    datos: DATOS_MUESTRA,
  });

  const anuncios = [];
  const anunciar = (msg) => anuncios.push(msg);
  let csvDescargado = null;
  const descargarCSVImpl = (csv) => { csvDescargado = csv; };

  const inst = montarVistaEstadisticas({
    raiz,
    cliente: clienteMock,
    anunciar,
    descargarCSVImpl,
  });

  // Esperar resolución de carga
  await new Promise((resolve) => setTimeout(resolve, 50));

  assert.ok(contenido.includes("Estadísticas de contratación temporal"));
  assert.ok(contenido.includes("TOTALES"));
  assert.equal(anuncios.includes("Estadísticas actualizadas"), true);

  // Probar exportar CSV mediante evento submit/click
  const botonCSV = {
    dataset: { ctAccion: "exportar-csv" },
    closest: (sel) => (sel === "[data-ct-accion]" ? { dataset: { ctAccion: "exportar-csv" } } : null),
  };
  eventos.get("click")({ target: botonCSV, preventDefault() {} });
  assert.ok(csvDescargado !== null, "Debe haber invocado la descarga de CSV");
  assert.ok(csvDescargado.includes("TOTAL;27;55;24;20;3"));

  // Desmontar
  inst.desmontar();
  assert.equal(contenido, "");
  assert.equal(eventos.size, 0);
});
