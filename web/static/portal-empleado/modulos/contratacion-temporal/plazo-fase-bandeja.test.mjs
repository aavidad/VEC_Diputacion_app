import assert from "node:assert/strict";
import test from "node:test";

import { crearAdaptadorHTTPExpedientesContratacionTemporal } from "./adaptador-http-expedientes.js";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import { renderizarCuadro } from "./componentes-expedientes.js";
import { validarCuadroContratacionTemporal } from "./contrato-expedientes.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";

const t = crearTraductorExpedientesContratacion();
const resumen = Object.freeze({
  expediente_ref: "expediente:ct:001", numero_visible: "2026/CT-0001", version: 6,
  flujo_ref: "flujo:ct:general", flujo_version: 1, flujo_huella_sha256: "a".repeat(64),
  fase_clave: "fiscalizacion", estado_clave: "en_curso", centro_ref: "centro:001",
  categoria_ref: "categoria:auxiliar", creado_en: "2026-09-03T08:00:00Z",
  actualizado_en: "2026-09-15T09:00:00Z",
});
const plazo = Object.freeze({
  ultimo_dia: "2026-09-29", vence_antes_de: "2026-09-29T22:00:00Z", estado: "vencido",
  regla_ref: "vec.contratacion_temporal.reglas:1:c03.plazo_fiscalizacion", regla_ejemplo: true,
});

function cliente(expedientes) {
  return crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => new Response(JSON.stringify({ data: {
      esquema: "vec.contratacion-temporal.cuadro-rrhh.v1", generada_en: "2026-09-30T08:00:00Z",
      expedientes, hay_mas: false,
    } }), { status: 200, headers: { "Content-Type": "application/json; charset=utf-8" } }),
  });
}

const filtros = { filtros: { texto: "", estado: "", fase: "" } };

test("sin plazo_fase la columna Plazo sigue en «—»", async () => {
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente: cliente([resumen]) });
  const cuadro = await adaptador.listar(filtros);
  assert.equal(cuadro.expedientes[0].plazo, "—");
  assert.equal(Object.hasOwn(cuadro.expedientes[0], "plazo_estado"), false);
});

test("con plazo_fase pinta fecha y estado en texto, sin la procedencia de la regla", async () => {
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente: cliente([{ ...resumen, plazo_fase: plazo }]),
  });
  const cuadro = await adaptador.listar(filtros);
  assert.equal(cuadro.expedientes[0].plazo, "29 sept 2026");
  assert.equal(cuadro.expedientes[0].plazo_estado, "vencido");
  const html = renderizarCuadro({
    vista: "cuadro", carga: "listo", filtros: { texto: "", estado: "", fase: "" }, cuadro,
  }, t);
  assert.match(html, /<span class="ct-exp-chip ct-plazo-vencido">29 sept 2026 · Vencido<\/span>/u);
  assert.doesNotMatch(html, /c03\.plazo_fiscalizacion|regla/u);
});

test("un plazo no calculado se dice, sin fecha supuesta", async () => {
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente: cliente([{ ...resumen, plazo_fase: { estado: "no_calculado" } }]),
  });
  const cuadro = await adaptador.listar(filtros);
  assert.equal(cuadro.expedientes[0].plazo, "Sin calcular");
  const html = renderizarCuadro({
    vista: "cuadro", carga: "listo", filtros: { texto: "", estado: "", fase: "" }, cuadro,
  }, t);
  assert.match(html, /<span class="ct-exp-chip ct-plazo-no_calculado">Sin calcular<\/span>/u);
});

test("cada estado del plazo tiene su texto", () => {
  for (const [estado, texto] of [["en_plazo", "En plazo"], ["vence_hoy", "Vence hoy"], ["vencido", "Vencido"]]) {
    assert.equal(t(`plazo_fase_${estado}`), texto);
  }
});

test("un plazo_fase mal formado o un estado desconocido se rechazan", async () => {
  for (const malo of [
    { ...plazo, estado: "casi" },
    { ...plazo, ultimo_dia: "29/09/2026" },
    { ...plazo, extra: 1 },
    { ...plazo, regla_ejemplo: "si" },
    { estado: "no_calculado", ultimo_dia: "2026-09-29" },
  ]) {
    await assert.rejects(cliente([{ ...resumen, plazo_fase: malo }]).consultarCuadroRRHH({
      filtros: { texto: "", estado_clave: "", fase_clave: "" }, paginacion: { limite: 50, cursor: "" },
    }), (error) => error?.codigo === "respuesta_incompatible");
  }
  const base = {
    expediente_ref: "expediente:ct:001", numero_visible: "2026/CT-0001", centro: "C", categoria: "A",
    modalidad: "—", estado_clave: "en_curso", estado: "En curso", fase_actual: "F",
    fecha_solicitud: "2026-09-03T08:00:00Z", responsable: "—", plazo: "29 sept 2026", version: 1,
  };
  const cuadro = (plazoEstado) => ({
    esquema: "vec.contratacion_temporal.cuadro.v1", demostracion: false, generado_en: "2026-09-30T08:00:00Z",
    indicadores: [], expedientes: [{ ...base, plazo_estado: plazoEstado }],
  });
  assert.equal(validarCuadroContratacionTemporal(cuadro("vencido")).expedientes[0].plazo_estado, "vencido");
  assert.throws(() => validarCuadroContratacionTemporal(cuadro("casi")), TypeError);
});
