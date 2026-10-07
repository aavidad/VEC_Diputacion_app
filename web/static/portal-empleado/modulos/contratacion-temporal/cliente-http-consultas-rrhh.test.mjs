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
      presentacion_flujo: {
        esquema: "vec.contratacion_temporal.presentacion_flujo_rrhh.v1",
        referencia: "flujo:rrhh:presentacion", version: 1,
        clave_i18n: "contratacion_temporal.flujo.rrhh", fase_actual: "solicitud",
        fases: [{clave: "solicitud", orden: 1, clave_i18n: "contratacion_temporal.fase.solicitud"}],
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
  assert.equal(detalle.presentacion_flujo.fase_actual, "solicitud");
  assert.equal(Object.hasOwn(detalle.presentacion_flujo,"huella_sha256"),false);


  assert.equal(pagina.expedientes[0].expediente_ref, resumen.expediente_ref);
  assert.equal(detalle.resumen.version, 1);
  assert.deepEqual(llamadas.map(({ ruta }) => ruta), [
    "/api/vec/contratacion-temporal/cuadro/consultas",
    "/api/vec/contratacion-temporal/expedientes/consultas",
  ]);
  for (const { opciones } of llamadas) {
    assert.equal(opciones.method, "POST");
    assert.equal(opciones.credentials, "same-origin");
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

test("rechaza un getter mutable antes de consultar el cuadro RRHH", () => {
  let lecturas = 0;
  let llamadas = 0;
  const filtros = {
    estado_clave: "",
    fase_clave: "",
  };
  Object.defineProperty(filtros, "texto", {
    enumerable: true,
    get() {
      lecturas += 1;
      return lecturas === 1 ? "" : "correo@invalid.example";
    },
  });
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => {
      llamadas += 1;
      return respuesta({ data: {} });
    },
  });

  assert.throws(() => cliente.consultarCuadroRRHH({
    filtros,
    paginacion: { limite: 50, cursor: "" },
  }), /solicitud de cuadro RRHH no válida/);
  assert.equal(lecturas, 0);
  assert.equal(llamadas, 0);
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

test("el detalle admite observaciones del análisis y rechaza las vacías o desmesuradas", async () => {
  const base = {
    esquema: "vec.contratacion-temporal.detalle-rrhh.v1",
    resumen,
    solicitud: { grupo_subgrupo: "A2", motivo_clave: "sustitucion", periodo_inicio: "2026-09-04T00:00:00Z", periodo_fin: "2026-12-31T00:00:00Z" },
    analisis: {
      modalidad_clave: "sustitucion", categoria_ref: "categoria:desarrollo:a1", causa_clave: "necesidad_temporal",
      periodo_inicio: "2027-01-21T00:00:00Z", periodo_fin: "2027-04-21T00:00:00Z", porcentaje_jornada: 7500,
      resultado_rc: "validada", observaciones: "Necesidad temporal verificada.",
    },
    hitos: [],
  };
  async function consultar(analisis) {
    const fetchImpl = async () => respuesta({ data: { ...base, analisis } });
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl });
    return cliente.consultarDetalleRRHH({ expediente_ref: resumen.expediente_ref, version_observada: 1 });
  }
  const detalle = await consultar(base.analisis);
  assert.equal(detalle.analisis.observaciones, "Necesidad temporal verificada.");
  const { observaciones, ...sinObservaciones } = base.analisis;
  assert.equal(Object.hasOwn((await consultar(sinObservaciones)).analisis, "observaciones"), false);
  for (const invalidas of ["", "x".repeat(4001), 7]) {
    await assert.rejects(consultar({ ...base.analisis, observaciones: invalidas }));
  }
});

test("acepta los totales del conjunto filtrado y rechaza totales incoherentes", async () => {
  const paginaCon = (totales) => async () => respuesta({ data: {
    esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
    generada_en: "2026-09-03T08:05:00Z",
    expedientes: [resumen],
    hay_mas: false,
    totales,
  } });
  const solicitud = {
    filtros: { texto: "", estado_clave: "", fase_clave: "" },
    paginacion: { limite: 50, cursor: "" },
  };
  const validos = { total: 71, en_tramitacion: 70, con_incidencia: 1, en_llamamiento: 0 };
  const pagina = await crearClienteHTTPContratacionTemporal({ fetchImpl: paginaCon(validos) })
    .consultarCuadroRRHH(solicitud);
  assert.deepEqual(pagina.totales, validos);
  for (const invalidos of [
    { total: 1, en_tramitacion: 2, con_incidencia: 0, en_llamamiento: 0 },
    { total: -1, en_tramitacion: 0, con_incidencia: 0, en_llamamiento: 0 },
    { total: 1, en_tramitacion: 0, con_incidencia: 0 },
    { total: 1, en_tramitacion: 0, con_incidencia: 0, en_llamamiento: 0, extra: 1 },
  ]) {
    await assert.rejects(
      crearClienteHTTPContratacionTemporal({ fetchImpl: paginaCon(invalidos) }).consultarCuadroRRHH(solicitud),
    );
  }
});

// Resumen de la portada (CT-000184): solo se pide con resumen: true y la
// respuesta solo admite recuentos enteros cuyo reparto por fase suma «en trámite».
test("el resumen de la portada se pide con un indicador cerrado y se valida", async () => {
  const resumenValido = { en_tramite: 3, con_incidencia: 1, vencidos: 1, vencen_hoy: 0, vencen_semana: 2, sin_calcular: 0,
    por_fase: { solicitud: 2, fiscalizacion: 1 } };
  const pagina = (resumenPortada) => ({ data: { esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
    generada_en: "2026-09-03T08:05:00Z", expedientes: [], hay_mas: false, resumen: resumenPortada } });
  let cuerpoEnviado = null;
  const cliente = (resumenPortada) => crearClienteHTTPContratacionTemporal({
    fetchImpl: async (_ruta, opciones) => { cuerpoEnviado = JSON.parse(opciones.body); return respuesta(pagina(resumenPortada)); },
  });
  const solicitud = { filtros: { texto: "", estado_clave: "", fase_clave: "" }, paginacion: { limite: 1, cursor: "" }, resumen: true };
  const leida = await cliente(resumenValido).consultarCuadroRRHH(solicitud);
  assert.equal(cuerpoEnviado.resumen, true);
  assert.deepEqual(leida.resumen, resumenValido);
  await assert.rejects(async () => cliente(resumenValido).consultarCuadroRRHH({ ...solicitud, resumen: false }));
  await assert.rejects(async () => cliente(resumenValido).consultarCuadroRRHH({ ...solicitud, resumen: "si" }));
  for (const malo of [
    { ...resumenValido, por_fase: { solicitud: 1 } },
    { ...resumenValido, vencidos: -1 },
    { ...resumenValido, con_incidencia: 4 },
    { ...resumenValido, expediente_ref: "e:1" },
    { ...resumenValido, por_fase: { "Fase mala": 3 } },
  ]) {
    await assert.rejects(async () => cliente(malo).consultarCuadroRRHH(solicitud), undefined, JSON.stringify(malo));
  }
});

test("v2 envía filtros cerrados sin mezclar campos viejos y exige recuentos del mismo corte", async () => {
  const cuerpos = [];
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (_ruta, opciones) => {
    cuerpos.push(JSON.parse(opciones.body));
    return respuesta({ data: { esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
      generada_en: "2026-09-03T08:05:00Z", expedientes: [resumen], hay_mas: false,
      totales: { total: 1, en_tramitacion: 1, con_incidencia: 0, en_llamamiento: 0 },
      resumen: { en_tramite: 1, con_incidencia: 0, vencidos: 0, vencen_hoy: 0,
        vencen_semana: 0, sin_calcular: 0, por_fase: { solicitud: 1 } },
    } });
  } });
  const solicitud = { filtros: { texto: "", centro_ref: "centro:001", categoria_ref: "categoria:auxiliar",
    estados_clave: ["pendiente", "en_curso"], fases_clave: ["solicitud", "solicitud_registrada"] },
  paginacion: { limite: 50, cursor: "" }, resumen: true };
  const pagina = await cliente.consultarCuadroRRHHV2(solicitud);
  assert.equal(pagina.totales.total, 1);
  assert.deepEqual(cuerpos, [solicitud]);
  for (const filtros of [
    { ...solicitud.filtros, estado_clave: "pendiente" },
    { ...solicitud.filtros, estados_clave: ["pendiente", "pendiente"] },
    { ...solicitud.filtros, fases_clave: ["solicitud", "solicitud"] },
    { ...solicitud.filtros, centro_ref: "<script>" },
    { ...solicitud.filtros, estados_clave: Array(7).fill("pendiente") },
  ]) {
    await assert.rejects(async () => cliente.consultarCuadroRRHHV2({ ...solicitud, filtros }));
  }
  assert.equal(cuerpos.length, 1, "un filtro inválido se rechaza antes de POST");
  await assert.rejects(async () => cliente.consultarCuadroRRHH(solicitud), "v1 no acepta filtros v2");
});
