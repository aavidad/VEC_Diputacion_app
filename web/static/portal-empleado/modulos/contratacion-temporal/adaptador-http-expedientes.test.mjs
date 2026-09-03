import assert from "node:assert/strict";
import test from "node:test";

import { crearAdaptadorHTTPExpedientesContratacionTemporal } from "./adaptador-http-expedientes.js";
import { crearPresentadorExpedientesContratacionTemporal } from "./presentador-expedientes.js";

const resumen = Object.freeze({
  expediente_ref: "expediente:ct:001",
  numero_visible: "2026/CT-0001",
  version: 2,
  flujo_ref: "flujo:ct:general",
  flujo_version: 1,
  flujo_huella_sha256: "a".repeat(64),
  fase_clave: "analisis",
  estado_clave: "espera_externa",
  centro_ref: "centro:001",
  categoria_ref: "categoria:auxiliar",
  modalidad_clave: "bolsa",
  unidad_ref: "unidad:rrhh",
  creado_en: "2026-09-03T08:00:00Z",
  actualizado_en: "2026-09-03T09:00:00Z",
});

function clienteFalso(llamadas) {
  return {
    async consultarCuadroRRHH(solicitud, opciones) {
      llamadas.push({ operacion: "cuadro", solicitud, opciones });
      return {
        esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
        generada_en: "2026-09-03T09:05:00Z",
        expedientes: [resumen],
        hay_mas: false,
      };
    },
    async consultarDetalleRRHH(solicitud, opciones) {
      llamadas.push({ operacion: "detalle", solicitud, opciones });
      return {
        esquema: "vec.contratacion-temporal.detalle-rrhh.v1",
        resumen,
        solicitud: {
          grupo_subgrupo: "A2",
          motivo_clave: "sustitucion",
          periodo_inicio: "2026-09-04T00:00:00Z",
          periodo_fin: "2026-12-31T00:00:00Z",
        },
        analisis: {
          modalidad_clave: "bolsa",
          categoria_ref: "categoria:auxiliar",
          causa_clave: "sustitucion",
          periodo_inicio: "2026-09-04T00:00:00Z",
          periodo_fin: "2026-12-31T00:00:00Z",
          porcentaje_jornada: 10_000,
          resultado_rc: "no_requerida",
        },
        hitos: [
          {
            secuencia: 1,
            version_expediente: 1,
            accion_clave: "registrar_solicitud",
            realizada_en: "2026-09-03T08:00:00Z",
            fase_destino: "solicitud",
            estado_origen: "pendiente",
            estado_destino: "pendiente",
          },
          {
            secuencia: 2,
            version_expediente: 2,
            accion_clave: "iniciar_analisis",
            realizada_en: "2026-09-03T09:00:00Z",
            fase_origen: "solicitud",
            fase_destino: "analisis",
            estado_origen: "pendiente",
            estado_destino: "en_curso",
          },
        ],
      };
    },
  };
}

test("convierte cuadro y detalle del servidor para la pantalla existente", async () => {
  const llamadas = [];
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente: clienteFalso(llamadas),
  });
  assert.deepEqual(adaptador.capacidades, []);
  const cuadro = await adaptador.listar({
    filtros: { texto: "auxiliar", estado: "espera", fase: "analisis" },
  });
  assert.deepEqual(adaptador.capacidades, ["contratacion_temporal.cuadro.consultar"]);
  const detalle = await adaptador.obtener(resumen.expediente_ref);

  assert.equal(cuadro.demostracion, false);
  assert.equal(cuadro.expedientes[0].categoria, "categoria:auxiliar");
  assert.equal(cuadro.expedientes[0].estado_clave, "espera");
  assert.equal(cuadro.expedientes[0].fase_clave, "analisis");
  assert.equal(cuadro.expedientes[0].fase_actual, "Analisis");
  assert.equal(detalle.demostracion, false);
  assert.deepEqual(detalle.fases, []);
  assert.deepEqual(detalle.tareas, []);
  assert.doesNotMatch(
    JSON.stringify(detalle),
    /tarea:hito|Identidad no publicada|Actuación registrada/u,
  );
  assert.deepEqual(adaptador.capacidades, [
    "contratacion_temporal.cuadro.consultar",
    "contratacion_temporal.expediente.consultar",
  ]);
  assert.deepEqual(llamadas.map(({ operacion }) => operacion), ["cuadro", "detalle"]);
  assert.deepEqual(llamadas[0].solicitud, {
    filtros: { texto: "auxiliar", estado_clave: "espera_externa", fase_clave: "analisis" },
    paginacion: { limite: 100, cursor: "" },
  });
  assert.deepEqual(llamadas[1].solicitud, {
    expediente_ref: resumen.expediente_ref,
    version_observada: 2,
  });
});
test("delega el detalle real al servidor y solo lo concede después de consultarlo", async () => {
  const llamadas = [];
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente: clienteFalso(llamadas),
  });
  await adaptador.listar();
  const presentador = crearPresentadorExpedientesContratacionTemporal({
    fuente: adaptador,
    capacidades: adaptador.capacidades,
  });
  assert.deepEqual(adaptador.capacidades, ["contratacion_temporal.cuadro.consultar"]);
  await assert.rejects(
    presentador.cargar({ texto: "A".repeat(81), estado: "", fase: "" }),
    /filtros no válidos/,
  );
  assert.equal(llamadas.length, 1);

  await presentador.cargar({ texto: "", estado: "espera", fase: "analisis" });
  const referencia = presentador.obtenerEstado().cuadro.expedientes[0].expediente_ref;
  await presentador.seleccionarExpediente(referencia);

  assert.equal(presentador.obtenerEstado().expediente.expediente_ref, referencia);
  assert.deepEqual(adaptador.capacidades, [
    "contratacion_temporal.cuadro.consultar",
    "contratacion_temporal.expediente.consultar",
  ]);
  assert.deepEqual(llamadas.map(({ operacion }) => operacion), [
    "cuadro", "cuadro", "detalle",
  ]);
});


test("no permite pedir detalle fuera del último cuadro ni ejecutar efectos", async () => {
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente: clienteFalso([]),
  });
  await assert.rejects(
    adaptador.obtener("expediente:ct:ajeno"),
    /fuera del cuadro consultado/,
  );
  await assert.rejects(
    adaptador.ejecutar({}),
    (error) => error?.codigo === "actuacion_no_disponible",
  );
});

test("rechaza una dependencia parcial antes de publicar el adaptador", () => {
  assert.throws(
    () => crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente: {} }),
    /cliente de expedientes.*no disponible/,
  );
});

test("rechaza estados desconocidos sin degradarlos a pendiente", async () => {
  let consultas = 0;
  const cliente = clienteFalso([]);
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente: {
      ...cliente,
      async consultarCuadroRRHH() {
        consultas += 1;
        return {
          esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
          generada_en: "2026-09-03T09:05:00Z",
          expedientes: [{ ...resumen, estado_clave: "desconocido" }],
          hay_mas: false,
        };
      },
    },
  });
  await assert.rejects(adaptador.listar(), /estado operativo del servidor no válido/);
  assert.deepEqual(adaptador.capacidades, []);
  assert.equal(consultas, 1);
  await assert.rejects(
    adaptador.listar({ filtros: { texto: "", estado: "desconocido", fase: "" } }),
    /filtro de estado visual no válido/,
  );
  assert.equal(consultas, 1);
});
