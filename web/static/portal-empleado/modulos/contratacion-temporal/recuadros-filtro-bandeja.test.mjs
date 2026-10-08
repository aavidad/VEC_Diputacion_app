import assert from "node:assert/strict";
import test from "node:test";

import { renderizarCuadro } from "./componentes-expedientes.js?v=20261001-ct-a-i18n-v1";
import { validarCuadroContratacionTemporal } from "./contrato-expedientes.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js?v=20261001-ct-a-i18n-v1";
import { crearPresentadorExpedientesContratacionTemporal } from "./presentador-expedientes.js?v=20261008-alta-rpt-circular-v5";
import { montarModuloContratacionTemporal } from "./vista-expedientes.js?v=20261008-alta-rpt-circular-v5";

const t = crearTraductorExpedientesContratacion();

function expediente(sufijo, estado_clave, fase_clave, fase_actual) {
  return {
    expediente_ref: `expediente:ct:recuadro-${sufijo}`,
    numero_visible: `2026/CT-00${sufijo}`,
    centro: "centro:rrhh",
    categoria: "categoria:auxiliar",
    modalidad: "Bolsa",
    estado_clave,
    estado: estado_clave,
    fase_clave,
    fase_actual,
    fecha_solicitud: "2026-09-07T08:00:00Z",
    responsable: "—",
    plazo: "—",
    version: 3,
  };
}

function cuadro() {
  return validarCuadroContratacionTemporal({
    esquema: "vec.contratacion_temporal.cuadro.v1",
    demostracion: false,
    generado_en: "2026-09-07T08:00:00Z",
    indicadores: [
      { clave: "total", etiqueta: "Expedientes", valor: "2", tono: "informacion" },
      { clave: "pendientes", etiqueta: "Pendientes", valor: "0", tono: "aviso" },
      { clave: "en_curso", etiqueta: "En curso", valor: "1", tono: "informacion" },
      { clave: "incidencias", etiqueta: "Incidencias", valor: "1", tono: "peligro" },
      { clave: "otra", etiqueta: "Otra cifra", valor: "5", tono: "neutro" },
    ],
    expedientes: [
      expediente("1", "en_curso", "solicitud", "Solicitud"),
      expediente("2", "incidencia", "llamamiento", "Llamamiento"),
    ],
    paginacion: { pagina: 1, cursor_actual: "", cursor_siguiente: "" },
  });
}

test("un filtro pedido al servidor se ve como etiqueta quitable junto a los de pantalla", () => {
  const html = renderizarCuadro({
    cuadro: cuadro(), carga: "listo", filtros: { texto: "", estado: "incidencia", fase: "" },
  }, t, { fase: "obtencion_candidato", mostrar: "atencion" });
  assert.match(html, /data-ct-exp-quitar-filtro="fase"\s+aria-label="Quitar el filtro Obtención del candidato">Obtención del candidato/u);
  assert.match(html, /data-ct-exp-quitar-filtro="mostrar"[^>]*>Requieren atención/u);
  assert.match(html, /data-ct-exp-quitar-filtro="servidor"[^>]*>Estado: Con incidencia/u);
  assert.match(html, /data-ct-exp-quitar-filtro="todos">Quitar todos/u);
  // Solo queda el expediente con incidencia en «Obtención del candidato».
  assert.match(html, /data-ct-exp-abrir="expediente:ct:recuadro-2"/u);
  assert.doesNotMatch(html, /data-ct-exp-abrir="expediente:ct:recuadro-1"/u);
  assert.doesNotMatch(html, /tarjeta-kpi/u);
});

test("pulsar un recuadro aplica su filtro a la bandeja y lleva a la lista", async () => {
  const eventos = new Map();
  const enfocados = [];
  const listado = { focus() { enfocados.push("listado"); }, scrollIntoView() {} };
  const raiz = {
    innerHTML: "",
    addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo),
    querySelector: (selector) => (selector === ".ct-exp-listado .ct-exp-tabla-lista" ? listado : null),
    contains: () => true,
  };
  const llamadas = [];
  const fuente = {
    async listar({ filtros }) { llamadas.push(filtros); return cuadro(); },
    async obtener() { throw new Error("no se debe abrir detalle"); },
    async ejecutar() { throw new Error("no se debe ejecutar efecto"); },
  };
  const presentador = crearPresentadorExpedientesContratacionTemporal({
    fuente, capacidades: ["contratacion_temporal.cuadro.consultar"],
  });
  const montaje = await montarModuloContratacionTemporal({ raiz, presentador });
  const pulsar = (dataset) => {
    const recuadro = { dataset, closest: () => null };
    return eventos.get("click")({
      preventDefault() {},
      target: { closest: (selector) => (selector.includes("data-ct-exp-filtro") ? recuadro : null) },
    });
  };
  try {
    await pulsar({ ctExpFiltroEstado: "incidencia" });
    assert.deepEqual(llamadas.at(-1), { texto: "", estado: "incidencia", fase: "" });
    assert.equal(presentador.obtenerEstado().filtros.estado, "incidencia");
    assert.match(raiz.innerHTML, /data-ct-exp-quitar-filtro="servidor"[^>]*>Estado: Con incidencia/u);
    await pulsar({ ctExpFiltroFase: "llamamiento" });
    assert.deepEqual(llamadas.at(-1), { texto: "", estado: "", fase: "llamamiento" });
    await pulsar({ ctExpFiltroEstado: "" });
    assert.deepEqual(llamadas.at(-1), { texto: "", estado: "", fase: "" });
    assert.deepEqual(enfocados, ["listado", "listado", "listado"]);
  } finally {
    montaje.desmontar();
  }
});
