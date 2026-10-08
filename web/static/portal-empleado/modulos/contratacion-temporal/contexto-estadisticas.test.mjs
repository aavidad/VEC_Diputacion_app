import test from "node:test";
import assert from "node:assert/strict";
import { renderizarContextoEstadisticas, renderizarVistaEstadisticas, montarVistaEstadisticas } from "./vista-estadisticas.js?v=20261008-alta-rpt-circular-v5";

const datos = Object.freeze({ esquema: "vec.ct.estadisticas.v1", periodo: "mensual",
  desde: "2026-01-15", hasta: "2026-06-08", zona_horaria: "Europe/Madrid", corte_global: 73,
  series: [], totales: { altas: 0, llamamientos: 0, formalizaciones: 0, cierres: 0, incidencias: 0 } });

test("el contexto visible usa las fechas y agrupación del servidor aunque los filtros estén vacíos o editados", () => {
  for (const filtros of [{}, { periodo: "semanal", desde: "2024-01-01", hasta: "2024-02-01" }]) {
    const antes = structuredClone(datos);
    const html = renderizarVistaEstadisticas({ estadoEstadisticas: { carga: "listo", datos }, filtros });
    const contexto = html.match(/<dl data-ct-contexto-estadisticas>[\s\S]*?<\/dl>/u)?.[0];
    assert.ok(contexto);
    assert.match(contexto, /datetime="2026-01-15"/u);
    assert.match(contexto, /datetime="2026-06-08"/u);
    assert.match(contexto, /<dd>Mensual<\/dd>/u);
    assert.match(contexto, /<dd>Europe\/Madrid<\/dd>/u);
    assert.match(contexto, /<dd>73<\/dd>/u);
    assert.doesNotMatch(contexto, /2024-|Semanal|data-ct-ayuda/u);
    assert.deepEqual(datos, antes);
  }
});

test("los metadatos opcionales ausentes no se infieren y el corte cero se muestra", () => {
  const { zona_horaria, corte_global, ...sinContexto } = datos;
  const html = renderizarContextoEstadisticas(sinContexto);
  assert.equal((html.match(/No comunicado/gu) ?? []).length, 2);
  assert.doesNotMatch(html, /Europe\/Madrid|<dd>73<\/dd>/u);
  assert.match(renderizarContextoEstadisticas({ ...datos, corte_global: 0 }), /<dd>0<\/dd>/u);
  assert.equal(renderizarContextoEstadisticas(null), "");
});

test("la zona recibida se escapa y la fecha calendario no cambia de día", () => {
  const html = renderizarContextoEstadisticas({ ...datos, zona_horaria: '<img src=x onerror="alert(1)">' });
  assert.doesNotMatch(html, /<img/u);
  assert.match(html, /&lt;img/u);
  assert.match(html, /datetime="2026-01-15">15 ene 2026<\/time>/u);
  assert.doesNotMatch(html, /14 ene 2026/u);
});

test("carga, error y denegación no muestran el contexto de una consulta anterior", () => {
  for (const carga of ["cargando", "error", "denegado"]) {
    const html = renderizarVistaEstadisticas({ estadoEstadisticas: { carga, datos, error: "Sin conexión" }, filtros: {} });
    assert.doesNotMatch(html, /data-ct-contexto-estadisticas/u);
  }
});

test("el montaje muestra el contexto de la respuesta incluso si la ayuda no está disponible", async () => {
  const raiz = { innerHTML: "", addEventListener() {}, removeEventListener() {} };
  const vista = montarVistaEstadisticas({ raiz, cliente: async () => ({ ok: true, datos }),
    cargarFicha: async () => { throw new Error("ayuda_no_disponible"); } });
  await Promise.resolve();
  assert.match(raiz.innerHTML, /data-ct-contexto-estadisticas/u);
  assert.deepEqual(vista.obtenerFiltros(), { periodo: "mensual", desde: "", hasta: "" });
  assert.deepEqual(vista.obtenerEstado().datos, datos);
  vista.desmontar();
});
