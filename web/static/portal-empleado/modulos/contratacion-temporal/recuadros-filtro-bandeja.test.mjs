import assert from "node:assert/strict";
import test from "node:test";

import { renderizarCuadro } from "./componentes-expedientes.js";
import { validarCuadroContratacionTemporal } from "./contrato-expedientes.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";
import { crearPresentadorExpedientesContratacionTemporal } from "./presentador-expedientes.js";
import { montarModuloContratacionTemporal } from "./vista-expedientes.js";

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

test("los recuadros con filtro equivalente son botones y los demás siguen como resumen", () => {
  const html = renderizarCuadro({
    cuadro: cuadro(), carga: "listo", filtros: { texto: "", estado: "incidencia", fase: "" },
  }, t);
  for (const [clave, filtro] of [["total", ""], ["pendientes", "pendiente"], ["en_curso", "en_curso"], ["incidencias", "incidencia"]]) {
    const patron = new RegExp(`<button type="button" class="tarjeta-kpi[^"]*" data-ct-exp-indicador="${clave}"\\s+data-ct-exp-filtro-estado="${filtro}"`, "u");
    assert.match(html, patron, clave);
  }
  assert.match(html, /data-ct-exp-indicador="incidencias"\s+data-ct-exp-filtro-estado="incidencia" aria-pressed="true"/u);
  assert.match(html, /data-ct-exp-indicador="total"\s+data-ct-exp-filtro-estado="" aria-pressed="false"/u);
  assert.match(html, /aria-label="Incidencias: 1\. Mostrar en la lista solo estos expedientes"/u);
  assert.match(html, /<article class="tarjeta-kpi" data-ct-exp-indicador="otra">/u);
  assert.doesNotMatch(html, /data-ct-exp-indicador="otra"\s+data-ct-exp-filtro/u);
  assert.match(html, /data-ct-exp-filtro-fase="llamamiento"\s+aria-label="Llamamiento: 1\. Mostrar en la lista los expedientes de esta fase"/u);
  // En cada fila, la pastilla de estado y la fase también filtran la lista.
  assert.match(html, /<button type="button" class="ct-exp-chip ct-fase-incidencia"\s+data-ct-exp-filtro-estado="incidencia"/u);
  assert.match(html, /class="enlace-tabla ct-exp-enlace-fase"\s+data-ct-exp-filtro-fase="solicitud"/u);
});

test("pulsar un recuadro aplica su filtro a la bandeja y lleva a la lista", async () => {
  const eventos = new Map();
  const enfocados = [];
  const listado = { focus() { enfocados.push("listado"); }, scrollIntoView() {} };
  const raiz = {
    innerHTML: "",
    addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo),
    querySelector: (selector) => (selector === ".ct-exp-listado .tabla-contenedor" ? listado : null),
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
    assert.match(raiz.innerHTML, /data-ct-exp-indicador="incidencias"\s+data-ct-exp-filtro-estado="incidencia" aria-pressed="true"/u);
    await pulsar({ ctExpFiltroFase: "llamamiento" });
    assert.deepEqual(llamadas.at(-1), { texto: "", estado: "", fase: "llamamiento" });
    await pulsar({ ctExpFiltroEstado: "" });
    assert.deepEqual(llamadas.at(-1), { texto: "", estado: "", fase: "" });
    assert.deepEqual(enfocados, ["listado", "listado", "listado"]);
  } finally {
    montaje.desmontar();
  }
});
