import test from "node:test";
import assert from "node:assert/strict";
import { cargarFichaIndicadores } from "../analitica/ficha-indicadores.js";
import { montarVistaEstadisticas, renderizarVistaEstadisticas } from "./vista-estadisticas.js?v=20261008-alta-rpt-circular-v5";

const ficha = await cargarFichaIndicadores();
const datos = { esquema: "vec.ct.estadisticas.v1", corte_global: 73, periodo: "mensual", series: [], totales: {} };

test("la vista existente ofrece ayuda nativa ligada al esquema y corte recibido sin cambiar datos", () => {
  const estado = { carga: "listo", datos };
  const antes = structuredClone(datos);
  const html = renderizarVistaEstadisticas({ estadoEstadisticas: estado, filtros: {}, fichaIndicadores: ficha });
  assert.match(html, /<details data-ct-ayuda-indicadores><summary class="boton-secundario boton-icono" aria-label=/);
  assert.match(html, /Corte de la respuesta consultada: 73/);
  assert.match(html, /vec.ct.estadisticas.v1/);
  assert.doesNotMatch(html, /data-analitica-cerrar/);
  assert.deepEqual(datos, antes);
  for (const malo of [{ ...datos, esquema: "otro" }, { ...datos, corte_global: -1 }, { ...datos, corte_global: undefined }]) {
    const resultado = renderizarVistaEstadisticas({ estadoEstadisticas: { carga: "listo", datos: malo }, filtros: {}, fichaIndicadores: ficha });
    assert.match(resultado, /no coincide con el formato o el corte/);
    assert.doesNotMatch(resultado, /Corte de la respuesta consultada:/);
  }
  const pendiente = renderizarVistaEstadisticas({ estadoEstadisticas: { carga: "cargando" }, filtros: {}, fichaIndicadores: ficha });
  assert.match(pendiente, /Todavía no hay una respuesta compatible/);
});

test("el montaje real pasa su respuesta a la ficha de ayuda", async () => {
  const raiz = { innerHTML: "", addEventListener() {}, removeEventListener() {} };
  const vista = montarVistaEstadisticas({ raiz, fichaIndicadores: ficha,
    cliente: async () => ({ ok: true, datos }) });
  await Promise.resolve();
  assert.match(raiz.innerHTML, /Corte de la respuesta consultada: 73/);
  assert.deepEqual(vista.obtenerEstado().datos, datos);
  vista.desmontar();
  assert.equal(raiz.innerHTML, "");
});

test("la ayuda tardía no sustituye el formulario que RRHH está editando", async () => {
  let entregarFicha;
  const fichaPendiente = new Promise((resolver) => { entregarFicha = resolver; });
  const hueco = { innerHTML: "" };
  const raiz = { innerHTML: "", addEventListener() {}, removeEventListener() {},
    querySelector: () => hueco };
  const vista = montarVistaEstadisticas({ raiz,
    cliente: () => new Promise(() => {}), cargarFicha: () => fichaPendiente });
  raiz.innerHTML += "valor-escrito-por-rrhh";
  entregarFicha(ficha);
  await fichaPendiente;
  await Promise.resolve();
  assert.match(raiz.innerHTML, /valor-escrito-por-rrhh/);
  assert.match(hueco.innerHTML, /data-ct-ayuda-indicadores/);
  vista.desmontar();
});
