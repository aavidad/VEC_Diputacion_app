import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { obtenerDatosPresentacion } from "../../datos-presentacion.js";
import { crearContextoActorPresentacionDesdeSesion } from "../../identidad/presentacion.js";
import { crearAdaptadorContratacionTemporalPresentacion } from "./adaptador-presentacion.js";
import { crearAdaptadorHTTPExpedientesContratacionTemporal } from "./adaptador-http-expedientes.js";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import { renderizarExpediente } from "./componentes-expedientes.js";
import {
  CAPACIDADES_CONTRATACION_TEMPORAL as CAP,
  validarAuditoriaContratacionTemporal,
  validarComandoActuacion,
  validarCuadroContratacionTemporal,
  validarDocumentosContratacionTemporal,
  validarExpedienteContratacionTemporal,
  validarReciboActuacion,
} from "./contrato-expedientes.js";
import { crearBorradorAlta, crearComandoAlta } from "./contrato.js";
import {
  crearAuditoriaContratacionTemporalPresentacion,
  crearCuadroContratacionTemporalPresentacion,
  crearDocumentosContratacionTemporalPresentacion,
  crearExpedienteContratacionTemporalPresentacion,
} from "./datos-presentacion.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";
import { crearPresentadorExpedientesContratacionTemporal } from "./presentador-expedientes.js";
import {
  crearEjecutorAltaConRefresco,
  montarModuloContratacionTemporal,
  renderizarModuloContratacionTemporal,
} from "./vista-expedientes.js";

function contexto(perfil = "administrador") {
  return crearContextoActorPresentacionDesdeSesion(
    obtenerDatosPresentacion(perfil).sesion,
  );
}

function adaptador(perfil = "administrador") {
  return crearAdaptadorContratacionTemporalPresentacion({
    contextoActor: contexto(perfil),
  });
}

function presentadorDe(fuente, capacidades = fuente.capacidades) {
  return crearPresentadorExpedientesContratacionTemporal({ fuente, capacidades });
}

test("el alta disponible no concede capacidades de consulta", async () => {
  let consultas = 0;
  const fuente = {
    capacidades: [],
    async listar() {
      consultas += 1;
      throw new Error("cuadro todavía no compuesto");
    },
    async obtener() { throw new Error("detalle no compuesto"); },
    async ejecutar() { throw new Error("actuación no compuesta"); },
  };
  const presentador = crearPresentadorExpedientesContratacionTemporal({
    fuente,
    capacidades: [],
    altaDisponible: true,
  });

  presentador.cambiarVista("alta");
  await presentador.cargar();
  assert.equal(consultas, 1);
  assert.equal(presentador.obtenerEstado().vista, "alta");
  assert.equal(presentador.obtenerEstado().carga, "error");
  assert.deepEqual(fuente.capacidades, []);
  assert.throws(
    () => crearPresentadorExpedientesContratacionTemporal({
      fuente, capacidades: [], altaDisponible: "si",
    }),
    /disponibilidad de alta no válida/u,
  );
});

test("un 502 de cuadro conserva el error al navegar y no muestra llamamiento", async () => {
  let consultas = 0;
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async (ruta, opciones) => {
      consultas += 1;
      assert.equal(ruta, "/api/vec/contratacion-temporal/cuadro/consultas");
      assert.equal(opciones.method, "POST");
      return new Response(JSON.stringify({ error: {
        codigo: "resultado_no_confiable",
        clave_i18n: "api.contratacion_temporal.consulta_rrhh.error.resultado_no_confiable",
        correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
      } }), { status: 502, headers: { "Content-Type": "application/json" } });
    },
  });
  const fuente = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente });
  const presentador = crearPresentadorExpedientesContratacionTemporal({
    fuente, capacidades: [], altaDisponible: true,
  });
  const eventos = new Map();
  const raiz = {
    innerHTML: "",
    addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo),
    querySelector: () => null,
    contains: () => true,
  };
  const montaje = await montarModuloContratacionTemporal({
    raiz, presentador, llamamiento: { cliente },
  });
  try {
    for (const vista of ["cuadro", "alta", "cuadro"]) {
      await eventos.get("click")({
        target: { closest: (selector) => selector === "[data-ct-exp-vista]"
          ? { dataset: { ctExpVista: vista } } : null },
        preventDefault() {},
      });
      const estado = presentador.obtenerEstado();
      assert.equal(estado.carga, "error");
      assert.equal(estado.cuadro, null);
      assert.equal(estado.expediente, null);
      assert.equal(estado.mensaje_clave, "estado_error_carga");
      assert.equal(estado.tipo_mensaje, "error");
      assert.match(raiz.innerHTML, /role="alert"/u);
      assert.doesNotMatch(raiz.innerHTML, /Cuadro de contratación temporal actualizado/u);
      assert.doesNotMatch(raiz.innerHTML, /data-ct-exp-llamamiento/u);
      if (vista === "cuadro") {
        assert.match(raiz.innerHTML, /data-ct-exp-accion="reintentar"/u);
      }
    }
    assert.equal(consultas, 1, "navegar no reintenta ni crea un efecto");
    assert.deepEqual(fuente.capacidades, []);
  } finally {
    montaje.desmontar();
  }
});

test("navegar sin resultado o cancelar una carga no anuncia un cuadro actualizado", async () => {
  let rechazar;
  const fuente = {
    listar: () => new Promise((_, reject) => { rechazar = reject; }),
    obtener() {}, ejecutar() {},
  };
  const presentador = presentadorDe(fuente, [CAP.consultarCuadro]);
  presentador.cambiarVista("cuadro");
  assert.equal(presentador.obtenerEstado().carga, "inicial");
  assert.equal(presentador.obtenerEstado().mensaje_clave, "estado_inicial");
  const carga = presentador.cargar();
  presentador.cambiarVista("cuadro");
  assert.equal(presentador.obtenerEstado().carga, "inicial");
  assert.equal(presentador.obtenerEstado().mensaje_clave, "estado_lectura_cancelada");
  rechazar(new Error("consulta sintética cancelada"));
  await carga;
  assert.equal(presentador.obtenerEstado().carga, "inicial");
  const denegado = presentadorDe(fuente, []);
  denegado.cambiarVista("cuadro");
  assert.equal(denegado.obtenerEstado().carga, "denegado");
  assert.equal(denegado.obtenerEstado().mensaje_clave, "estado_denegado");
});

function estadoVista(expediente, tareaRef = expediente.tareas[0].tarea_ref) {
  return {
    vista: "expediente",
    carga: "listo",
    cuadro: null,
    expediente,
    documentos: null,
    auditoria: null,
    expediente_ref: expediente.expediente_ref,
    tarea_ref: tareaRef,
    filtros: { texto: "", estado: "", fase: "" },
    ocupado: false,
    actualizacion_pendiente: false,
    recibo: null,
    mensaje_clave: "estado_expediente_listo",
    tipo_mensaje: "informacion",
  };
}

function comandoDe(expediente, tarea, accion) {
  return validarComandoActuacion({
    esquema: "vec.contratacion_temporal.actuacion.v1",
    expediente_ref: expediente.expediente_ref,
    version_esperada: expediente.version,
    tarea_ref: tarea.tarea_ref,
    accion_ref: accion.accion_ref,
    datos: {},
  });
}

test("el espacio operativo separa tareas y distribución en paneles legibles", async () => {
  const [estilos, tema] = await Promise.all([
    readFile(new URL("./expedientes-operativo.css", import.meta.url), "utf8"),
    readFile(new URL("../../portal.css", import.meta.url), "utf8"),
  ]);
  assert.match(
    estilos,
    /\.ct-exp-mis-tareas,\s*\n\.ct-exp-distribucion\s*\{[\s\S]*border:[^;]+;[\s\S]*background:/u,
  );
  assert.match(estilos, /\.ct-exp-operativo\s*\{[\s\S]*grid-template-columns:/u);
  for (const token of [
    "--portal-espacio-1", "--portal-espacio-2", "--portal-espacio-3",
    "--portal-espacio-4", "--portal-radio-md", "--portal-radio-lg",
    "--portal-sombra-sm", "--portal-tinta-suave",
  ]) {
    assert.match(tema, new RegExp(`${token}:`), `${token} debe proceder del tema común`);
  }
});

test("los cuatro contratos rechazan extras, duplicados, cruces y valores no canónicos", () => {
  const cuadro = crearCuadroContratacionTemporalPresentacion();
  const expediente = crearExpedienteContratacionTemporalPresentacion();
  const documentos = crearDocumentosContratacionTemporalPresentacion();
  const auditoria = crearAuditoriaContratacionTemporalPresentacion();
  assert.equal(validarCuadroContratacionTemporal(cuadro).expedientes.length, 5);
  assert.equal(validarExpedienteContratacionTemporal(expediente).tareas.length, 18);
  assert.equal(validarDocumentosContratacionTemporal(documentos).documentos.length, 7);
  assert.equal(validarAuditoriaContratacionTemporal(auditoria).actuaciones.length, 8);
  assert.throws(() => validarCuadroContratacionTemporal({ ...cuadro, secreto: "x" }), /cerrado/);
  assert.throws(() => validarExpedienteContratacionTemporal({
    ...expediente,
    tareas: [...expediente.tareas, expediente.tareas[0]],
  }), /duplicados/);
  assert.throws(() => validarExpedienteContratacionTemporal({
    ...expediente,
    tareas: expediente.tareas.map((tarea, indice) => (
      indice === 0 ? { ...tarea, fase_ref: "fase-inexistente" } : tarea
    )),
  }), /fase inexistente/);
  assert.throws(() => validarDocumentosContratacionTemporal({
    ...documentos,
    documentos: documentos.documentos.map((documento, indice) => (
      indice === 0 ? { ...documento, extra: true } : documento
    )),
  }), /cerrado/);
  assert.throws(() => validarAuditoriaContratacionTemporal({
    ...auditoria,
    actuaciones: [...auditoria.actuaciones, auditoria.actuaciones[0]],
  }), /duplicados/);
  assert.throws(() => validarComandoActuacion({
    esquema: "vec.contratacion_temporal.actuacion.v1",
    expediente_ref: expediente.expediente_ref,
    version_esperada: expediente.version,
    tarea_ref: expediente.tareas[13].tarea_ref,
    accion_ref: expediente.tareas[13].acciones[0].accion_ref,
    datos: { campo: { anidado: true } },
  }), /no válido/);
  assert.throws(() => validarReciboActuacion({
    esquema: "vec.contratacion_temporal.recibo-actuacion.v1",
    recibo_ref: "rec-prueba-001",
    expediente_ref: expediente.expediente_ref,
    numero_visible: expediente.numero_visible,
    version: 13,
    actuacion: "Prueba",
    estado_resultante: "Registrado",
    registrada_en: "2026-07-23T10:00:00Z",
    token: "prohibido",
  }), /cerrado/);
});

test("filtro, selección y proyecciones segregadas conservan referencia y versión", async () => {
  const fuente = adaptador();
  const presentador = presentadorDe(fuente);
  await presentador.cargar({ texto: "Secretaría", estado: "", fase: "" });
  let estado = presentador.obtenerEstado();
  assert.equal(estado.cuadro.expedientes.length, 1);
  const referencia = estado.cuadro.expedientes[0].expediente_ref;
  await presentador.seleccionarExpediente(referencia, "documentos");
  estado = presentador.obtenerEstado();
  assert.equal(estado.expediente.expediente_ref, referencia);
  assert.equal(estado.documentos.expediente_ref, referencia);
  assert.equal(estado.documentos.version, estado.expediente.version);
  assert.equal(estado.auditoria, null);
  await presentador.seleccionarExpediente(referencia, "auditoria");
  estado = presentador.obtenerEstado();
  assert.equal(estado.auditoria.expediente_ref, referencia);
  assert.equal(estado.auditoria.version, estado.expediente.version);
  assert.equal(estado.documentos, null);
  await presentador.cargar({ texto: "sin coincidencias", estado: "", fase: "" });
  estado = presentador.obtenerEstado();
  assert.equal(estado.vista, "cuadro");
  assert.equal(estado.expediente_ref, "");
  assert.equal(estado.expediente, null);
});

test("cuadro y detalle son coherentes para las cinco referencias sintéticas", async () => {
  const fuente = adaptador();
  const cuadro = await fuente.listar();
  for (const resumen of cuadro.expedientes) {
    const detalle = await fuente.obtener(resumen.expediente_ref);
    assert.equal(detalle.numero_visible, resumen.numero_visible);
    assert.equal(detalle.version, resumen.version);
    const activas = detalle.tareas.filter((tarea) => (
      ["en_curso", "espera", "incidencia"].includes(tarea.estado_clave)
    ));
    if (resumen.estado_clave === "completado") {
      assert.equal(activas.length, 0);
      assert.ok(detalle.tareas.every(({ estado_clave: clave }) => clave === "completado"));
      assert.ok(detalle.fases.every(({ estado_clave: clave }) => clave === "completado"));
    } else {
      assert.equal(activas.length, 1, resumen.numero_visible);
      assert.equal(activas[0].estado_clave, resumen.estado_clave);
      const fase = detalle.fases.find(({ fase_ref }) => fase_ref === activas[0].fase_ref);
      assert.equal(fase.estado_clave, resumen.estado_clave);
    }
  }
});

test("RBAC se proyecta en HTML y vuelve a imponerse dentro del adaptador", async () => {
  const fuenteAdmin = adaptador("administrador");
  const fuenteTecnica = adaptador("tecnico");
  const referencia = (await fuenteAdmin.listar()).expedientes[0].expediente_ref;
  const presentadorAdmin = presentadorDe(fuenteAdmin);
  const presentadorTecnica = presentadorDe(fuenteTecnica);
  await Promise.all([presentadorAdmin.cargar(), presentadorTecnica.cargar()]);
  await Promise.all([
    presentadorAdmin.seleccionarExpediente(referencia),
    presentadorTecnica.seleccionarExpediente(referencia),
  ]);
  for (const presentador of [presentadorAdmin, presentadorTecnica]) {
    presentador.seleccionarTarea("tarea-formalizacion");
  }
  const t = crearTraductorExpedientesContratacion();
  const htmlAdmin = renderizarExpediente(
    presentadorAdmin.obtenerEstado(), t, "es-ES", "Europe/Madrid",
  );
  const htmlTecnica = renderizarExpediente(
    presentadorTecnica.obtenerEstado(), t, "es-ES", "Europe/Madrid",
  );
  assert.match(htmlAdmin, /data-ct-exp-efecto="enviar_firma_formalizacion"[\s\S]*?>Enviar a firma electrónica/);
  assert.doesNotMatch(
    htmlAdmin,
    /data-ct-exp-efecto="enviar_firma_formalizacion"[\s\S]{0,300}?disabled/,
  );
  assert.match(
    htmlTecnica,
    /data-ct-exp-efecto="enviar_firma_formalizacion"[\s\S]{0,300}?disabled/,
  );
  assert.match(htmlTecnica, /perfil activo no tiene concedida esta actuación/);
  const detalle = await fuenteTecnica.obtener(referencia);
  const tarea = detalle.tareas.find(({ tarea_ref }) => tarea_ref === "tarea-formalizacion");
  const accion = tarea.acciones.find(({ accion_ref }) => accion_ref === "enviar_firma_formalizacion");
  await assert.rejects(fuenteTecnica.ejecutar(comandoDe(detalle, tarea, accion)), /Acceso denegado/);
});

test("una transición emite recibo, añade auditoría y no puede repetirse", async () => {
  const fuente = adaptador();
  const resumen = (await fuente.listar()).expedientes[0];
  const antes = await fuente.obtener(resumen.expediente_ref);
  const auditoriaAntes = await fuente.obtenerAuditoria(resumen.expediente_ref);
  const tarea = antes.tareas.find(({ tarea_ref }) => tarea_ref === "tarea-formalizacion");
  const accion = tarea.acciones.find(
    ({ accion_ref }) => accion_ref === "generar_documentos_formalizacion",
  );
  const comando = comandoDe(antes, tarea, accion);
  const recibo = await fuente.ejecutar(comando);
  const despues = await fuente.obtener(resumen.expediente_ref);
  const auditoriaDespues = await fuente.obtenerAuditoria(resumen.expediente_ref);
  assert.equal(recibo.version, antes.version + 1);
  assert.equal(despues.version, recibo.version);
  const tareaDespues = despues.tareas.find(({ tarea_ref }) => tarea_ref === tarea.tarea_ref);
  assert.equal(tareaDespues.recibo_ref, recibo.recibo_ref);
  assert.ok(tareaDespues.decision_ref);
  assert.equal(
    tareaDespues.acciones.find(({ accion_ref }) => accion_ref === accion.accion_ref).disponible,
    false,
  );
  assert.equal(auditoriaDespues.actuaciones.length, auditoriaAntes.actuaciones.length + 1);
  await assert.rejects(fuente.ejecutar({
    ...comando,
    version_esperada: recibo.version,
  }), /no está disponible/);
});

test("el presentador rechaza recibos cruzados y conserva el éxito si falla la recarga", async () => {
  const expediente = validarExpedienteContratacionTemporal(
    crearExpedienteContratacionTemporalPresentacion(),
  );
  const cuadro = validarCuadroContratacionTemporal(crearCuadroContratacionTemporalPresentacion());
  const tarea = expediente.tareas[13];
  const accion = tarea.acciones[0];
  let lecturas = 0;
  const fuenteCruzada = {
    listar: async () => cuadro,
    obtener: async () => expediente,
    ejecutar: async () => validarReciboActuacion({
      esquema: "vec.contratacion_temporal.recibo-actuacion.v1",
      recibo_ref: "rec-prueba-cruzado",
      expediente_ref: expediente.expediente_ref,
      numero_visible: "2026/CT-99999",
      version: expediente.version + 1,
      actuacion: accion.etiqueta,
      estado_resultante: "Registrado",
      registrada_en: "2026-07-23T10:00:00Z",
    }),
  };
  const cruzado = presentadorDe(fuenteCruzada, [
    CAP.consultarCuadro, CAP.consultarExpediente, CAP.prepararFormalizacion,
  ]);
  await cruzado.cargar();
  await cruzado.seleccionarExpediente(expediente.expediente_ref);
  cruzado.seleccionarTarea(tarea.tarea_ref);
  await cruzado.ejecutarActuacion({ accionRef: accion.accion_ref });
  assert.equal(cruzado.obtenerEstado().mensaje_clave, "estado_error_actuacion");
  assert.equal(cruzado.obtenerEstado().recibo, null);

  const fuenteSinRefresco = {
    ...fuenteCruzada,
    obtener: async () => {
      lecturas += 1;
      if (lecturas > 1) throw new Error("detalle privado");
      return expediente;
    },
    ejecutar: async () => validarReciboActuacion({
      esquema: "vec.contratacion_temporal.recibo-actuacion.v1",
      recibo_ref: "rec-prueba-valido",
      expediente_ref: expediente.expediente_ref,
      numero_visible: expediente.numero_visible,
      version: expediente.version + 1,
      actuacion: accion.etiqueta,
      estado_resultante: "Registrado",
      registrada_en: "2026-07-23T10:00:00Z",
    }),
  };
  const sinRefresco = presentadorDe(fuenteSinRefresco, [
    CAP.consultarCuadro, CAP.consultarExpediente, CAP.prepararFormalizacion,
  ]);
  await sinRefresco.cargar();
  await sinRefresco.seleccionarExpediente(expediente.expediente_ref);
  sinRefresco.seleccionarTarea(tarea.tarea_ref);
  await sinRefresco.ejecutarActuacion({ accionRef: accion.accion_ref });
  assert.equal(sinRefresco.obtenerEstado().recibo.recibo_ref, "rec-prueba-valido");
  assert.equal(sinRefresco.obtenerEstado().actualizacion_pendiente, true);
  assert.equal(
    sinRefresco.obtenerEstado().mensaje_clave,
    "estado_confirmada_actualizacion_pendiente",
  );
});

test("mutex y cancelación impiden doble efecto y dejan resultado indeterminado visible", async () => {
  const expediente = validarExpedienteContratacionTemporal(
    crearExpedienteContratacionTemporalPresentacion(),
  );
  const cuadro = validarCuadroContratacionTemporal(crearCuadroContratacionTemporalPresentacion());
  let ejecuciones = 0;
  const fuente = {
    listar: async () => cuadro,
    obtener: async () => expediente,
    ejecutar: async (_comando, { signal }) => {
      ejecuciones += 1;
      return new Promise((resolve, reject) => {
        signal.addEventListener("abort", () => reject(
          new DOMException("cancelada", "AbortError"),
        ), { once: true });
      });
    },
  };
  const presentador = presentadorDe(fuente, [
    CAP.consultarCuadro, CAP.consultarExpediente, CAP.prepararFormalizacion,
  ]);
  await presentador.cargar();
  await presentador.seleccionarExpediente(expediente.expediente_ref);
  presentador.seleccionarTarea("tarea-formalizacion");
  const primera = presentador.ejecutarActuacion({
    accionRef: "generar_documentos_formalizacion",
  });
  const segunda = presentador.ejecutarActuacion({
    accionRef: "generar_documentos_formalizacion",
  });
  presentador.cancelar();
  await Promise.all([primera, segunda]);
  assert.equal(ejecuciones, 1);
  assert.equal(presentador.obtenerEstado().actualizacion_pendiente, true);
  assert.equal(presentador.obtenerEstado().mensaje_clave, "estado_cancelado");
});

test("cancelar una lectura no fabrica un efecto indeterminado", async () => {
  const expediente = validarExpedienteContratacionTemporal(
    crearExpedienteContratacionTemporalPresentacion(),
  );
  const fuente = {
    listar: async ({ signal }) => new Promise((_resolve, reject) => {
      signal.addEventListener("abort", () => reject(
        new DOMException("cancelada", "AbortError"),
      ), { once: true });
    }),
    obtener: async () => expediente,
    ejecutar: async () => {
      throw new Error("no debe ejecutarse");
    },
  };
  const presentador = presentadorDe(fuente, [CAP.consultarCuadro]);
  const carga = presentador.cargar();
  presentador.cancelar();
  await carga;
  assert.equal(presentador.obtenerEstado().actualizacion_pendiente, false);
  assert.equal(presentador.obtenerEstado().resultado_indeterminado, false);
  assert.equal(
    presentador.obtenerEstado().mensaje_clave,
    "estado_lectura_cancelada",
  );
});

test("un efecto indeterminado bloquea cualquier repetición hasta recuperar estado", async () => {
  const expediente = validarExpedienteContratacionTemporal(
    crearExpedienteContratacionTemporalPresentacion(),
  );
  const cuadro = validarCuadroContratacionTemporal(
    crearCuadroContratacionTemporalPresentacion(),
  );
  let ejecuciones = 0;
  const fuente = {
    listar: async () => cuadro,
    obtener: async () => expediente,
    ejecutar: async () => {
      ejecuciones += 1;
      const error = new Error("detalle privado");
      error.resultadoIndeterminado = true;
      error.reintentoPermitido = false;
      throw error;
    },
  };
  const presentador = presentadorDe(fuente, [
    CAP.consultarCuadro, CAP.consultarExpediente, CAP.prepararFormalizacion,
  ]);
  await presentador.cargar();
  await presentador.seleccionarExpediente(expediente.expediente_ref);
  presentador.seleccionarTarea("tarea-formalizacion");
  await presentador.ejecutarActuacion({
    accionRef: "generar_documentos_formalizacion",
  });
  assert.equal(presentador.obtenerEstado().actualizacion_pendiente, true);
  assert.equal(
    presentador.obtenerEstado().mensaje_clave,
    "estado_resultado_indeterminado",
  );
  await presentador.ejecutarActuacion({
    accionRef: "generar_documentos_formalizacion",
  });
  assert.equal(ejecuciones, 1);
  assert.equal(
    presentador.obtenerEstado().mensaje_clave,
    "estado_resultado_indeterminado",
  );
  await presentador.seleccionarExpediente(expediente.expediente_ref);
  assert.equal(presentador.obtenerEstado().resultado_indeterminado, true);
  assert.equal(presentador.obtenerEstado().actualizacion_pendiente, true);
  assert.equal(
    presentador.obtenerEstado().mensaje_clave,
    "estado_resultado_indeterminado",
  );
  presentador.cambiarVista("cuadro");
  assert.equal(
    presentador.obtenerEstado().mensaje_clave,
    "estado_resultado_indeterminado",
  );
});
