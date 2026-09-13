import assert from "node:assert/strict";
import test from "node:test";

import { crearAdaptadorHTTPExpedientesContratacionTemporal } from "./adaptador-http-expedientes.js";
import { renderizarExpediente, solicitudInformeDefinitivoDesdeEstado } from "./componentes-expedientes.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";
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
  assert.equal(cuadro.expedientes[0].fase_actual, "Análisis");
  assert.equal(cuadro.expedientes[0].modalidad, "Bolsa");
  assert.equal(cuadro.expedientes[0].estado, "En espera externa");
  assert.equal(detalle.demostracion, false);
  assert.equal(detalle.cabecera.find(({ clave }) => clave === "motivo").valor, "Sustitución");
  assert.equal(detalle.cabecera.find(({ clave }) => clave === "fase").valor, "Análisis");
  assert.deepEqual(detalle.fases, []);
  assert.deepEqual(detalle.historial, [
    {
      secuencia: 1,
      fecha: "3 sept 2026",
      fase: "Solicitud",
      accion: "Solicitud registrada",
      estado_clave: "pendiente",
      estado: "Pendiente",
      accion_clave: "registrar_solicitud",
      version_expediente: 1,
    },
    {
      secuencia: 2,
      fecha: "3 sept 2026",
      fase: "Análisis",
      accion: "Iniciar analisis",
      estado_clave: "en_curso",
      estado: "En tramitación",
      accion_clave: "iniciar_analisis",
      version_expediente: 2,
    },
  ]);
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

test("traduce las etiquetas conocidas con mensajes inyectados y conserva el fallback", async () => {
  const cliente = clienteFalso([]);
  cliente.consultarCuadroRRHH = async () => ({
    esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
    generada_en: "2026-09-03T09:05:00Z",
    expedientes: [{ ...resumen, fase_clave: "fase_futura", modalidad_clave: "sustitucion" }],
    hay_mas: false,
  });
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente,
    mensajes: { etiqueta_modalidad_sustitucion: "Sustitución del catálogo" },
  });

  const cuadro = await adaptador.listar();

  assert.equal(cuadro.expedientes[0].modalidad, "Sustitución del catálogo");
  assert.equal(cuadro.expedientes[0].fase_actual, "Fase futura");
});

test("muestra la fase de asignación de unidad en castellano", async () => {
  const cliente = clienteFalso([]);
  cliente.consultarCuadroRRHH = async () => ({
    esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
    generada_en: "2026-09-03T09:05:00Z",
    expedientes: [{ ...resumen, fase_clave: "asignacion_unidad" }],
    hay_mas: false,
  });
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente });

  const cuadro = await adaptador.listar();

  assert.equal(cuadro.expedientes[0].fase_actual, "Asignación de unidad");
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

test("muestra el hito de subsanación autorizado sin exponer observaciones", async () => {
  const llamadas = [];
  const cliente = clienteFalso(llamadas);
  cliente.consultarDetalleRRHH = async () => ({
    esquema: "vec.contratacion-temporal.detalle-rrhh.v1",
    resumen: { ...resumen, version: 2, fase_clave: "subsanacion_unidad", estado_clave: "incidencia" },
    solicitud: {
      grupo_subgrupo: "A2", motivo_clave: "sustitucion",
      periodo_inicio: "2026-09-04T00:00:00Z", periodo_fin: "2026-12-31T00:00:00Z",
    },
    hitos: [
      {
        secuencia: 1, version_expediente: 1, accion_clave: "registrar_fiscalizacion",
        realizada_en: "2026-09-03T08:00:00Z", fase_destino: "subsanacion_unidad",
        estado_origen: "en_curso", estado_destino: "incidencia",
      },
      {
        secuencia: 2, version_expediente: 2,
        accion_clave: "contratacion_temporal.subsanacion_reparos.registrar",
        realizada_en: "2026-09-03T09:00:00Z", fase_origen: "subsanacion_unidad",
        fase_destino: "subsanacion_unidad", estado_origen: "incidencia",
        estado_destino: "incidencia",
      },
    ],
    presentacion_flujo: {
      referencia: "flujo-visual:rrhh:temporal",
      fase_actual: "fiscalizacion",
      fases: [
        ["solicitud", "contratacion_temporal.fase.solicitud"],
        ["analisis_rrhh", "contratacion_temporal.fase.analisis_rrhh"],
        ["gestion_bolsa", "contratacion_temporal.fase.gestion_bolsa"],
        ["fiscalizacion", "contratacion_temporal.fase.fiscalizacion"],
        ["obtencion_candidato", "contratacion_temporal.fase.obtencion_candidato"],
        ["nombramiento", "contratacion_temporal.fase.nombramiento"],
        ["incorporacion", "contratacion_temporal.fase.incorporacion"],
        ["seguimiento", "contratacion_temporal.fase.seguimiento"],
      ].map(([clave, clave_i18n], indice) => ({ clave, clave_i18n, orden: indice + 1 })),
    },
  });
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente,
    mensajes: { hito_subsanacion_reparo: "Hito de subsanación del reparo" },
  });
  await adaptador.listar();
  const expediente = await adaptador.obtener(resumen.expediente_ref);
  const html = renderizarExpediente({
    vista: "expediente", carga: "listo", expediente, tarea_ref: "", cuadro: null,
  }, crearTraductorExpedientesContratacion(), "es-ES", "Europe/Madrid");

  assert.equal(expediente.fases.length, 8);
  assert.equal(expediente.fases[3].etiqueta, "Fiscalización");
  assert.equal(expediente.fases[3].estado_clave, "sin_confirmar");
  assert.equal(expediente.historial[1].accion, "Hito de subsanación del reparo");
  assert.match(html, /Hito de subsanación del reparo/u);
  assert.doesNotMatch(JSON.stringify(expediente), /observaciones|retorno_ref|actor_ref|documentos_ref/u);
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

function detalleV9() {
  const resumenV9 = { expediente_ref: "expediente:ct:historia-v9", numero_visible: "2026/CT-0009", version: 9,
    flujo_ref: "flujo:ct:desarrollo", flujo_version: 1, flujo_huella_sha256: "a".repeat(64), fase_clave: "nombramiento", estado_clave: "en_curso", centro_ref: "centro:prueba", categoria_ref: "categoria:prueba", creado_en: "2026-09-12T09:00:00Z", actualizado_en: "2026-09-12T10:00:00Z" };
  const acciones = ["registrar_solicitud", "registrar_analisis", "registrar_cobertura", "registrar_asignacion", "registrar_informe_juridico", "registrar_fiscalizacion", "registrar_propuesta_formalizacion", "registrar_resolucion_formalizacion", "contratacion_temporal.anotacion_administrativa.registrar"];
  return { esquema: "vec.contratacion-temporal.detalle-rrhh.v1", resumen: resumenV9,
    solicitud: { grupo_subgrupo: "A2", motivo_clave: "sustitucion", periodo_inicio: "2026-09-12T00:00:00Z", periodo_fin: "2026-12-31T00:00:00Z" },
    hitos: acciones.map((accion_clave, indice) => ({ secuencia: indice + 1, version_expediente: indice + 1, accion_clave, realizada_en: "2026-09-12T10:00:00Z", fase_origen: "nombramiento", fase_destino: "nombramiento", estado_origen: "en_curso", estado_destino: "en_curso" })),
  };
}

function clienteHistoria(detalle) {
  return { async consultarCuadroRRHH() { return { esquema: "vec.contratacion-temporal.cuadro-rrhh.v1", generada_en: detalle.resumen.actualizado_en, expedientes: [detalle.resumen], hay_mas: false }; }, async consultarDetalleRRHH() { return detalle; } };
}

test("proyecta los borradores de propuesta v7 desde la anotación coherente v9", async () => {
  const detalle = detalleV9();
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente: clienteHistoria(detalle) });
  await adaptador.listar();
  const expediente = await adaptador.obtener(detalle.resumen.expediente_ref);
  assert.equal(expediente.version, 9);
  assert.equal(expediente.version_propuesta_documental, 7);
});

test("no proyecta borradores ante historia hueca, cruzada o anotación distinta", async () => {
  for (const alterar of [(d) => { d.hitos[8].secuencia = 10; }, (d) => { d.hitos[8].version_expediente = 8; }, (d) => { d.hitos[7].fase_origen = "fiscalizacion"; }, (d) => { d.hitos[8].accion_clave = "otra_anotacion"; }, (d) => { d.hitos[8].estado_destino = "completado"; }]) {
    const detalle = detalleV9(); alterar(detalle);
    const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente: clienteHistoria(detalle) });
    await adaptador.listar();
    const expediente = await adaptador.obtener(detalle.resumen.expediente_ref);
    assert.equal(Object.hasOwn(expediente, "version_propuesta_documental"), false);
  }
});


test("ofrece documentos de la propuesta actual tras subsanar y refiscalizar", async () => {
  const detalle = detalleV9();
  const accionesPosteriores = ["contratacion_temporal.subsanacion_reparo.registrar",
    "registrar_fiscalizacion", "registrar_propuesta_formalizacion"];
  detalle.hitos = detalle.hitos.map((hito, i) => i < 6 ? hito
    : { ...hito, accion_clave: accionesPosteriores[i - 6] });
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente: clienteHistoria(detalle) });
  const cuadro = await adaptador.listar();
  const expediente = await adaptador.obtener(detalle.resumen.expediente_ref);
  assert.equal(expediente.version_propuesta_documental, 9);
  const estado = { vista: "expediente", carga: "listo", expediente, cuadro,
    expediente_ref: expediente.expediente_ref };
  assert.deepEqual(solicitudInformeDefinitivoDesdeEstado(estado), {
    expediente_ref: expediente.expediente_ref, version_observada: 9,
  });
  assert.equal(solicitudInformeDefinitivoDesdeEstado({ ...estado, actualizacion_pendiente: true }), null);
  for (const alterar of [
    (d) => { d.hitos[3].version_expediente = 3; },
    (d) => { d.hitos[8] = { ...d.hitos[8], accion_clave: accionesPosteriores[1] }; },
  ]) {
    const invalido = structuredClone(detalle); alterar(invalido);
    const otro = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente: clienteHistoria(invalido) });
    const cuadroInvalido = await otro.listar();
    const expedienteInvalido = await otro.obtener(invalido.resumen.expediente_ref);
    assert.equal(Object.hasOwn(expedienteInvalido, "version_propuesta_documental"), false);
    assert.equal(solicitudInformeDefinitivoDesdeEstado({ ...estado,
      expediente: expedienteInvalido, cuadro: cuadroInvalido }), null);
  }
});

test("conserva acceso a propuesta v9 desde resolución v10 y anotación v11", async () => {
  const detalle = detalleV9();
  ["contratacion_temporal.subsanacion_reparos.registrar", "registrar_fiscalizacion", "registrar_propuesta_formalizacion"]
    .forEach((accion_clave, i) => { detalle.hitos[i + 6].accion_clave = accion_clave; });
  for (const accion_clave of ["registrar_resolucion_formalizacion", "contratacion_temporal.anotacion_administrativa.registrar"]) {
    const version = detalle.hitos.length + 1;
    detalle.hitos.push({ ...detalle.hitos.at(-1), accion_clave, secuencia: version, version_expediente: version });
    detalle.resumen.version = version;
    const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente: clienteHistoria(detalle) });
    const cuadro = await adaptador.listar();
    const expediente = await adaptador.obtener(detalle.resumen.expediente_ref);
    const estado = { vista: "expediente", carga: "listo", expediente, cuadro,
      expediente_ref: expediente.expediente_ref };
    assert.equal(expediente.version_propuesta_documental, 9);
    assert.deepEqual(solicitudInformeDefinitivoDesdeEstado(estado), {
      expediente_ref: expediente.expediente_ref, version_observada: version,
    });
    for (const alterar of [
      (d) => { d.hitos[9] = { ...d.hitos[9], accion_clave: "resolucion_ajena" }; },
      (d) => { d.hitos[8].version_expediente = 7; },
      (d) => { d.hitos[9].estado_destino = "completado"; },
    ]) {
      const invalido = structuredClone(detalle); alterar(invalido);
      const otro = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente: clienteHistoria(invalido) });
      const cuadroInvalido = await otro.listar();
      const expedienteInvalido = await otro.obtener(invalido.resumen.expediente_ref);
      assert.equal(Object.hasOwn(expedienteInvalido, "version_propuesta_documental"), false);
      assert.equal(solicitudInformeDefinitivoDesdeEstado({ ...estado,
        expediente: expedienteInvalido, cuadro: cuadroInvalido }), null);
    }
  }
});


test("distingue el período solicitado del revisado por RRHH sin sustituir el antecedente", async () => {
  const cliente = clienteFalso([]);
  const consultar = cliente.consultarDetalleRRHH;
  let conAnalisis = true;
  cliente.consultarDetalleRRHH = async (...args) => {
    const datos = await consultar(...args);
    datos.solicitud.periodo_inicio = "2027-01-01T00:00:00Z";
    datos.solicitud.periodo_fin = "2027-03-31T00:00:00Z";
    if (!conAnalisis) delete datos.analisis;
    return datos;
  };
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente });
  await adaptador.listar({ filtros: { texto: "", estado: "", fase: "" } });
  const detalle = await adaptador.obtener(resumen.expediente_ref);
  const solicitado = detalle.cabecera.find(c => c.clave === "periodo");
  const analizado = detalle.cabecera.find(c => c.clave === "periodo_analizado");
  assert.equal(solicitado.etiqueta, "Período solicitado");
  assert.equal(analizado.etiqueta, "Período analizado por RRHH");
  assert.match(solicitado.valor, /2027/u);
  assert.match(analizado.valor, /2026/u);
  assert.equal(detalle.analisis_previo.periodo.inicio, "2026-09-04T00:00:00Z");
  assert.equal(detalle.analisis_previo.porcentaje_jornada, 10000);
  assert.equal(Object.hasOwn(detalle.analisis_previo, "entrada_rc"), false);
  conAnalisis = false;
  const sinAnalisis = await adaptador.obtener(resumen.expediente_ref);
  assert.deepEqual(sinAnalisis.cabecera.find(c => c.clave === "periodo"), solicitado);
  assert.equal(sinAnalisis.cabecera.some(c => c.clave === "periodo_analizado"), false);
  assert.equal(Object.hasOwn(sinAnalisis, "analisis_previo"), false);
});
