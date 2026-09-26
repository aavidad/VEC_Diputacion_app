import assert from "node:assert/strict";
import test from "node:test";

import { crearAdaptadorHTTPExpedientesContratacionTemporal } from "./adaptador-http-expedientes.js";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";

const resumen = Object.freeze({
  expediente_ref: "expediente:ct:001", numero_visible: "2026/CT-000001", version: 6,
  flujo_ref: "flujo:ct:general", flujo_version: 1, flujo_huella_sha256: "a".repeat(64),
  fase_clave: "fiscalizacion", estado_clave: "en_curso", centro_ref: "centro:001",
  categoria_ref: "categoria:auxiliar", creado_en: "2026-09-03T08:00:00Z",
  actualizado_en: "2026-09-15T09:00:00Z",
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
const modalidadesCatalogo = Object.freeze([
  { clave: "sustitucion", etiqueta: "Sustitución" },
  { clave: "interinidad_programa_empleo", etiqueta: "Interinidad por programa de empleo" },
]);

test("una modalidad nueva del catálogo se pinta con su etiqueta, no con la clave", async () => {
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente: cliente([{ ...resumen, modalidad_clave: "interinidad_programa_empleo" }]),
    obtenerModalidades: () => Promise.resolve(modalidadesCatalogo),
  });
  const cuadro = await adaptador.listar(filtros);
  assert.equal(cuadro.expedientes[0].modalidad, "Interinidad por programa de empleo");
});

test("la etiqueta del catálogo manda también sobre la traducción conocida", async () => {
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente: cliente([{ ...resumen, modalidad_clave: "relevo" }]),
    obtenerModalidades: () => [{ clave: "relevo", etiqueta: "Jubilación parcial con relevo" }],
  });
  const cuadro = await adaptador.listar(filtros);
  assert.equal(cuadro.expedientes[0].modalidad, "Jubilación parcial con relevo");
});

test("sin catálogo, o si falla o no tiene forma, se pinta como hasta ahora", async () => {
  for (const obtenerModalidades of [
    () => null,
    () => Promise.reject(new Error("sin configuración")),
    () => { throw new Error("sin configuración"); },
    () => [{ clave: "interinidad_programa_empleo", etiqueta: "  " }, "relevo", null],
  ]) {
    const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
      cliente: cliente([
        { ...resumen, modalidad_clave: "vacante" },
        { ...resumen, expediente_ref: "expediente:ct:002", modalidad_clave: "interinidad_programa_empleo" },
        { ...resumen, expediente_ref: "expediente:ct:003" },
      ]),
      obtenerModalidades,
    });
    const cuadro = await adaptador.listar(filtros);
    assert.deepEqual(cuadro.expedientes.map(({ modalidad }) => modalidad),
      ["Vacante", "Interinidad programa empleo", "—"]);
  }
});

test("obtenerModalidades debe ser una función", () => {
  assert.throws(() => crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente: cliente([]), obtenerModalidades: [],
  }), /obtener modalidades/u);
});
