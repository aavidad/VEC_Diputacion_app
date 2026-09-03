import assert from "node:assert/strict";
import test from "node:test";

import {
  crearClienteHTTPContratacionTemporal,
  RUTAS_HTTP_CONTRATACION_TEMPORAL,
} from "./cliente-http.js";

const resumen = Object.freeze({
  expediente_ref: "expediente:ct:001",
  numero_visible: "2026/CT-0001",
  version: 1,
  flujo_ref: "flujo:ct:general",
  flujo_version: 1,
  flujo_huella_sha256: "a".repeat(64),
  fase_clave: "solicitud",
  estado_clave: "pendiente",
  centro_ref: "centro:001",
  categoria_ref: "categoria:auxiliar",
  creado_en: "2026-09-03T08:00:00Z",
  actualizado_en: "2026-09-03T08:00:00Z",
});

function respuesta(datos, estado = 200) {
  return new Response(JSON.stringify(datos), {
    status: estado,
    headers: { "Content-Type": "application/json; charset=utf-8" },
  });
}

test("consulta cuadro y detalle por las rutas reales sin credenciales del navegador", async () => {
  const llamadas = [];
  const fetchImpl = async (ruta, opciones) => {
    llamadas.push({ ruta, opciones, cuerpo: JSON.parse(opciones.body) });
    if (ruta === RUTAS_HTTP_CONTRATACION_TEMPORAL.cuadroRRHH) {
      return respuesta({ data: {
        esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
        generada_en: "2026-09-03T08:05:00Z",
        expedientes: [resumen],
        hay_mas: false,
      } });
    }
    return respuesta({ data: {
      esquema: "vec.contratacion-temporal.detalle-rrhh.v1",
      resumen,
      solicitud: {
        grupo_subgrupo: "A2",
        motivo_clave: "sustitucion",
        periodo_inicio: "2026-09-04T00:00:00Z",
        periodo_fin: "2026-12-31T00:00:00Z",
      },
      hitos: [{
        secuencia: 1,
        version_expediente: 1,
        accion_clave: "registrar_solicitud",
        realizada_en: "2026-09-03T08:00:00Z",
        fase_destino: "solicitud",
        estado_origen: "pendiente",
        estado_destino: "pendiente",
      }],
    } });
  };
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl });
  const pagina = await cliente.consultarCuadroRRHH({
    filtros: { texto: "", estado_clave: "", fase_clave: "" },
    paginacion: { limite: 50, cursor: "" },
  });
  const detalle = await cliente.consultarDetalleRRHH({
    expediente_ref: resumen.expediente_ref,
    version_observada: 0,
  });

  assert.equal(pagina.expedientes[0].expediente_ref, resumen.expediente_ref);
  assert.equal(detalle.resumen.version, 1);
  assert.deepEqual(llamadas.map(({ ruta }) => ruta), [
    "/api/vec/contratacion-temporal/cuadro/consultas",
    "/api/vec/contratacion-temporal/expedientes/consultas",
  ]);
  for (const { opciones } of llamadas) {
    assert.equal(opciones.method, "POST");
    assert.equal(opciones.credentials, "omit");
    assert.equal(opciones.cache, "no-store");
    assert.equal(opciones.redirect, "error");
    assert.equal(opciones.headers.get("content-type"), "application/json");
  }
  assert.deepEqual(llamadas[0].cuerpo, {
    filtros: { texto: "", estado_clave: "", fase_clave: "" },
    paginacion: { limite: 50, cursor: "" },
  });
  assert.deepEqual(llamadas[1].cuerpo, {
    expediente_ref: resumen.expediente_ref,
    version_observada: 0,
  });
});

test("iguala los límites y vocabularios cerrados del contrato Go", async () => {
  const cursor = "A".repeat(43);
  const cuerpos = [];
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async (_ruta, opciones) => {
      cuerpos.push(JSON.parse(opciones.body));
      return respuesta({ data: {
        esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
        generada_en: "2026-09-03T08:05:00Z",
        expedientes: [{ ...resumen, estado_clave: "espera_externa" }],
        hay_mas: true,
        cursor_siguiente: cursor,
      } });
    },
  });
  const solicitud = {
    filtros: {
      texto: "A".repeat(80),
      estado_clave: "espera_externa",
      fase_clave: "solicitud",
    },
    paginacion: { limite: 100, cursor },
  };
  const pagina = await cliente.consultarCuadroRRHH(solicitud);
  assert.equal(pagina.expedientes[0].estado_clave, "espera_externa");
  assert.deepEqual(cuerpos, [solicitud]);

  for (const invalida of [
    {
      ...solicitud,
      filtros: { ...solicitud.filtros, texto: "A".repeat(81) },
    },
    {
      ...solicitud,
      filtros: { ...solicitud.filtros, texto: "correo@invalid.example" },
    },
    {
      ...solicitud,
      filtros: { ...solicitud.filtros, estado_clave: "espera" },
    },
    {
      ...solicitud,
      paginacion: { ...solicitud.paginacion, cursor: "A".repeat(42) + "B" },
    },
  ]) {
    assert.throws(
      () => cliente.consultarCuadroRRHH(invalida),
      /solicitud de cuadro RRHH no válida/,
    );
  }
  assert.equal(cuerpos.length, 1);

  const clienteEstadoDesconocido = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => respuesta({ data: {
      esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
      generada_en: "2026-09-03T08:05:00Z",
      expedientes: [{ ...resumen, estado_clave: "desconocido" }],
      hay_mas: false,
    } }),
  });
  await assert.rejects(clienteEstadoDesconocido.consultarCuadroRRHH({
    filtros: { texto: "", estado_clave: "", fase_clave: "" },
    paginacion: { limite: 50, cursor: "" },
  }), (error) => error?.codigo === "respuesta_incompatible");
});

test("rechaza entradas y proyecciones ambiguas antes de publicarlas", async () => {
  let llamadas = 0;
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => {
      llamadas += 1;
      return respuesta({ data: {
        esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
        generada_en: "2026-09-03T08:05:00Z",
        expedientes: [{ ...resumen, actor_ref: "persona:privada" }],
        hay_mas: false,
      } });
    },
  });
  await assert.rejects(cliente.consultarCuadroRRHH({
    filtros: { texto: "", estado_clave: "", fase_clave: "" },
    paginacion: { limite: 50, cursor: "" },
  }), (error) => error?.codigo === "respuesta_incompatible");
  assert.equal(llamadas, 1);
});

test("acepta el error cerrado propio de consulta RRHH", async () => {
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => respuesta({ error: {
      codigo: "servicio_no_disponible",
      clave_i18n: "api.contratacion_temporal.consulta_rrhh.error.servicio_no_disponible",
      correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
    } }, 503),
  });
  await assert.rejects(cliente.consultarCuadroRRHH({
    filtros: { texto: "", estado_clave: "", fase_clave: "" },
    paginacion: { limite: 50, cursor: "" },
  }), (error) => error?.codigo === "servicio_no_disponible"
    && error.envelopeValido === true);
});
