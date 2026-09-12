import assert from "node:assert/strict";
import test from "node:test";

import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";

const expediente_ref = "expediente:ct:cierre";
const seguimiento_ref = "seguimiento:ct:cierre";
const clave_idempotencia = "4c31dbbc-487c-4cbb-a391-b0e94978d272";
const respuesta = (body, status = 200) => new Response(JSON.stringify(body), { status, headers: { "content-type": "application/json; charset=utf-8" } });
const preparacion = {
  expediente_ref, seguimiento_ref, version_actual: 1, estado_actual: "vigente",
  acciones: [{ transicion_clave: "cerrar_administrativamente_sin_cese", motivos: [{ motivo_clave: "sin_cese" }] }],
  preparada_en: "2026-09-12T10:00:00.000000Z",
};

test("cierre: la preparación GET queda ligada a expediente y seguimiento solicitados", async () => {
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
    assert.equal(opciones.method, "GET"); assert.equal(opciones.body, undefined);
    assert.equal(ruta, "/api/vec/contratacion-temporal/seguimiento/cerrar-sin-cese/preparacion?expediente_ref=expediente%3Act%3Acierre&seguimiento_ref=seguimiento%3Act%3Acierre");
    return respuesta({ data: preparacion });
  } });
  const resultado = await cliente.consultarPreparacionCierreSinCese({ expediente_ref, seguimiento_ref });
  assert.deepEqual(resultado.preparacion, { expediente_ref, seguimiento_ref, version_esperada: 1, motivos: ["sin_cese"] });
  const cruzado = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => respuesta({ data: { ...preparacion, seguimiento_ref: "seguimiento:ct:ajeno" } }) });
  await assert.rejects(cruzado.consultarPreparacionCierreSinCese({ expediente_ref, seguimiento_ref }));
});

test("cierre: POST entrega recibo y conserva denegación o conflicto como resultado determinado", async () => {
  const solicitud = { expediente_ref, seguimiento_ref, version_esperada: 1, clave_idempotencia, transicion_clave: "cerrar_administrativamente_sin_cese", motivo_clave: "sin_cese" };
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
    assert.equal(ruta, "/api/vec/contratacion-temporal/seguimiento/cerrar-sin-cese"); assert.equal(opciones.method, "POST");
    assert.deepEqual(JSON.parse(opciones.body), solicitud);
    return respuesta({ data: { recibo_ref: "recibo:cierre:1", version_seguimiento: 2 } }, 201);
  } });
  assert.deepEqual(await cliente.cerrar(solicitud), { recibo_ref: "recibo:cierre:1", version_seguimiento: 2 });
  for (const [estado, codigo] of [[401, "autenticacion_requerida"], [403, "acceso_denegado"], [409, "version_en_conflicto"]]) {
    const rechazado = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => respuesta({ error: { codigo, clave_i18n: `api.contratacion_temporal.cierre_administrativo.error.${codigo}`, correlacion_ref: "corr_no_disponible" } }, estado) });
    await assert.rejects(rechazado.cerrar(solicitud), (error) => error.estado === estado && error.codigo === codigo && error.envelopeValido && !error.resultadoIndeterminado);
  }
});
