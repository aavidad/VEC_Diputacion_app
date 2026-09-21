import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { obtenerDatosPresentacion } from "../../datos-presentacion.js";
import { crearContextoActorPresentacionDesdeSesion } from "../../identidad/presentacion.js";
import { crearAdaptadorContratacionTemporalPresentacion } from "./adaptador-presentacion.js";
import { crearAdaptadorHTTPExpedientesContratacionTemporal } from "./adaptador-http-expedientes.js";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import { renderizarCuadro, renderizarExpediente } from "./componentes-expedientes.js";
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

function expedienteConAccionSinteticaDisponible() {
  const entrada = crearExpedienteContratacionTemporalPresentacion();
  const tarea = entrada.tareas.find(({ tarea_ref }) => tarea_ref === "tarea-solicitud");
  tarea.acciones[0] = {
    ...tarea.acciones[0],
    disponible: true,
    motivo_no_disponible: "",
  };
  return validarExpedienteContratacionTemporal(entrada);
}

test("las tareas operativas cubren todos los hitos funcionales de RRHH", () => {
  const expediente = validarExpedienteContratacionTemporal(
    crearExpedienteContratacionTemporalPresentacion(),
  );
  const matriz = [
    ["tarea-solicitud", "Datos de la petición"],
    ["tarea-analisis", "Comprobación y validación"],
    ["tarea-cobertura", "Procedimiento a seguir"],
    ["tarea-asignacion", "Bandeja de la unidad"],
    ["tarea-informe-juridico", "Borrador y edición gobernada"],
    ["tarea-envio-intervencion", "Datos pendientes del circuito"],
    ["tarea-fiscalizacion", "Modalidad y remisión"],
    ["tarea-subsanacion", "Observaciones, correcciones y evidencias"],
    ["tarea-iniciar-llamamiento", "Historial de llamamientos"],
    ["tarea-seleccion-candidato", "Candidatura propuesta"],
    ["tarea-resultado-llamamiento", "Resumen e historial de la candidatura"],
    ["tarea-traslado-intervencion", "Preparación para formalización"],
    ["tarea-informe-definitivo", "Candidatura, observaciones e historial"],
    ["tarea-formalizacion", "Circuito Portafirmas P4 pendiente"],
    ["tarea-incorporacion", "Proyección autorizada para incorporación"],
    ["tarea-ginpix", "Historial GINPIX"],
    ["tarea-envio-ginpix", "Envío a GINPIX"],
    ["tarea-seguimiento", "Histórico de relación, prórroga y cese"],
  ];
  assert.equal(expediente.tareas.length, matriz.length);
  for (const [referencia, evidencia] of matriz) {
    const tarea = expediente.tareas.find(({ tarea_ref }) => tarea_ref === referencia);
    assert.ok(tarea, `falta ${referencia}`);
    assert.match(JSON.stringify(tarea), new RegExp(evidencia, "u"), referencia);
  }
  const cuadro = validarCuadroContratacionTemporal(
    crearCuadroContratacionTemporalPresentacion(),
  );
  const html = renderizarModuloContratacionTemporal({
    ...estadoVista(expediente),
    vista: "cuadro",
    cuadro,
    expediente: null,
    expediente_ref: "",
    tarea_ref: "",
  });
  assert.match(html, /Mis tareas prioritarias/);
  assert.match(html, /Distribución por fase/);
  assert.match(html, /Registrar nueva petición/);
});

test("la presentación delimita plazos, llamamientos y preparación previa sin inventar efectos", () => {
  const expediente = crearExpedienteContratacionTemporalPresentacion();
  const porReferencia = (referencia) => expediente.tareas.find(
    ({ tarea_ref }) => tarea_ref === referencia,
  );
  const texto = (referencia) => JSON.stringify(porReferencia(referencia));

  assert.match(texto("tarea-subsanacion"), /plazo por definir/u);
  assert.match(texto("tarea-iniciar-llamamiento"), /Escenario sintético/u);
  assert.match(texto("tarea-iniciar-llamamiento"), /Canal de comunicación/u);
  assert.doesNotMatch(texto("tarea-iniciar-llamamiento"), /Correo y teléfono|Renuncia acreditada/u);
  assert.match(texto("tarea-seleccion-candidato"), /Candidatura propuesta/u);
  assert.match(texto("tarea-seleccion-candidato"), /pendiente de validar/u);
  assert.match(texto("tarea-seleccion-candidato"), /no adjudica ni llama automáticamente/u);
  assert.doesNotMatch(texto("tarea-seleccion-candidato"), /Llamar a la primera candidatura/u);
  assert.match(texto("tarea-resultado-llamamiento"), /Respuesta manual sintética/u);
  assert.match(texto("tarea-resultado-llamamiento"), /plazo no evaluado/u);
  assert.doesNotMatch(texto("tarea-resultado-llamamiento"), /dentro de plazo|Entregado/u);

  const traslado = porReferencia("tarea-traslado-intervencion");
  assert.equal(traslado.etiqueta, "Preparación para formalización");
  assert.match(JSON.stringify(traslado), /no existe envío externo/u);
  assert.doesNotMatch(JSON.stringify(traslado), /Trasladado|Enviar a Intervención/u);
  assert.deepEqual(traslado.acciones, []);

  const cuadro = crearCuadroContratacionTemporalPresentacion();
  assert.ok(cuadro.expedientes.every(({ plazo }) => (
    ["Regla pendiente", "Plazo por definir", "Cerrado"].includes(plazo)
  )));
});

test("la formalización mantiene P4 pendiente sin inventar circuito ni efectos administrativos", () => {
  const expediente = crearExpedienteContratacionTemporalPresentacion();
  const tarea = expediente.tareas.find(({ tarea_ref }) => tarea_ref === "tarea-formalizacion");
  assert.ok(tarea);
  assert.equal(tarea.estado_clave, "en_curso");
  assert.equal(tarea.salida, "");
  assert.equal(tarea.recibo_ref, "");
  assert.equal(tarea.decision_ref, "");
  assert.equal(tarea.acciones.length, 1);
  assert.equal(tarea.acciones[0].accion_ref, "preparar_borrador_firma_demo");
  assert.equal(tarea.acciones[0].disponible, false);
  const texto = JSON.stringify(tarea);
  assert.match(texto, /Pendiente de definición por RRHH/u);
  assert.match(texto, /solo prepara una vista en memoria/i);
  assert.match(texto, /no envía, firma, registra ni genera un recibo administrativo/i);
  assert.doesNotMatch(texto, /Jefatura de Servicio|Órgano competente|fe pública|Enviada al portafirmas|Firmado|orden proceden/i);
});

test("montaje y desmontaje son simétricos y no dejan efectos tras retirar la vista", async () => {
  const eventos = new Map();
  const raiz = {
    innerHTML: "",
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
    },
    querySelector() { return null; },
    contains() { return true; },
  };
  const presentador = presentadorDe(adaptador());
  const montaje = await montarModuloContratacionTemporal({ raiz, presentador });
  assert.equal(eventos.size, 2);
  assert.match(raiz.innerHTML, /Expedientes de contratación/);
  montaje.desmontar();
  assert.equal(eventos.size, 0);
  const estadoAntes = presentador.obtenerEstado();
  const estado = await presentador.cargar();
  assert.strictEqual(estado, estadoAntes);
  assert.strictEqual(presentador.obtenerEstado(), estadoAntes);
});

test("presentación queda aislada de red, cookies, storage y manifiestos productivos", async () => {
  const directorio = new URL("./", import.meta.url);
  const [adaptadorFuente, datosFuente, presentadorFuente, vistaFuente, interno, produccion] =
    await Promise.all([
      readFile(new URL("adaptador-presentacion.js", directorio), "utf8"),
      readFile(new URL("datos-presentacion.js", directorio), "utf8"),
      readFile(new URL("presentador-expedientes.js", directorio), "utf8"),
      readFile(new URL("vista-expedientes.js", directorio), "utf8"),
      readFile(new URL("../../../../interno.manifest", directorio), "utf8"),
      readFile(new URL("../../../../produccion.manifest", directorio), "utf8"),
    ]);
  const candidato = `${adaptadorFuente}\n${datosFuente}\n${presentadorFuente}\n${vistaFuente}`;
  assert.doesNotMatch(
    candidato,
    /\b(?:fetch|XMLHttpRequest|WebSocket|EventSource)\s*\(|document\.cookie|localStorage|sessionStorage|indexedDB/i,
  );
  assert.doesNotMatch(`${interno}\n${produccion}`, /adaptador-presentacion\.js|datos-presentacion\.js/);
  for (const neutro of [
    "contrato-expedientes.js", "presentador-expedientes.js", "vista-expedientes.js",
    "componentes-expedientes.js", "expedientes.css",
  ]) {
    assert.match(interno, new RegExp(neutro.replace(".", "\\.")));
    assert.match(produccion, new RegExp(neutro.replace(".", "\\.")));
  }
});

test("la presentación no transmite GINPIX ni altera el expediente al llegar al envío", async () => {
  const fuente = adaptador();
  const referencia = "exp-demo-contratacion-005487";
  const expediente = await fuente.obtener(referencia);

  const tareaEnvio = expediente.tareas.find(({ tarea_ref: actual }) => actual === "tarea-envio-ginpix");
  const accionEnvio = tareaEnvio.acciones.find(({ accion_ref: actual }) => actual === "enviar_ginpix");
  assert.equal(accionEnvio.capacidad, CAP.enviarGinpix);
  assert.equal(accionEnvio.disponible, false);
  const versionAntes = expediente.version;
  const auditoriaAntes = await fuente.obtenerAuditoria(referencia);

  await assert.rejects(
    fuente.ejecutar(comandoDe(expediente, tareaEnvio, accionEnvio)),
    /no está disponible|conector corporativo/u,
  );

  const despues = await fuente.obtener(referencia);
  assert.equal(despues.version, versionAntes);
  assert.deepEqual(await fuente.obtenerAuditoria(referencia), auditoriaAntes);
});

test("la vista de alta no renderiza subcabecera redundante ni bloque de llamamiento", () => {
  const estado = {
    vista: "alta",
    carga: "listo",
    tipo_mensaje: "informacion",
    mensaje_clave: "estado_inicial",
    cuadro: null,
    expediente: null,
  };
  const html = renderizarModuloContratacionTemporal(estado, {
    altaDisponible: true,
    llamamientoDisponible: true,
  });
  assert.doesNotMatch(html, /ct-exp-subcabecera/u);
  assert.doesNotMatch(html, /Nueva petición de personal/u);
  assert.doesNotMatch(html, /data-ct-exp-llamamiento/u);
  assert.match(html, /data-ct-exp-alta/u);
});

test("abrir otro expediente sitúa foco y scroll en su cabecera una sola vez", async () => {
  const fuente = adaptador();
  const presentador = presentadorDe(fuente);
  await presentador.cargar();
  const eventos = new Map();
  const atributos = new Map();
  let focos = 0;
  let desplazamientos = 0;
  const cabecera = {
    setAttribute(nombre, valor) { atributos.set(nombre, valor); },
    focus() { focos += 1; },
    scrollIntoView(opciones) {
      desplazamientos += 1;
      assert.deepEqual(opciones, { block: "nearest", inline: "nearest" });
    },
  };
  const raiz = {
    innerHTML: "",
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) { eventos.delete(tipo); },
    querySelector(selector) {
      return selector === ".ct-exp-cabecera-expediente h3" ? cabecera : null;
    },
    contains() { return true; },
  };
  const referencia = presentador.obtenerEstado().cuadro.expedientes[1].expediente_ref;
  const control = {
    dataset: { ctExpAbrir: referencia },
    closest(selector) { return selector === "[data-ct-exp-abrir]" ? control : null; },
  };
  const montaje = await montarModuloContratacionTemporal({ raiz, presentador });
  try {
    await eventos.get("click")({ target: control, preventDefault() {} });
    assert.equal(presentador.obtenerEstado().expediente_ref, referencia);
    assert.equal(atributos.get("tabindex"), "-1");
    assert.equal(focos, 1);
    assert.equal(desplazamientos, 1);
  } finally {
    montaje.desmontar();
  }
});

test("un fallo al abrir se renderiza desde el estado del presentador y no mueve el foco", async () => {
  const base = adaptador();
  const cuadro = await base.listar();
  const fuente = {
    capacidades: [CAP.consultarCuadro, CAP.consultarExpediente],
    async listar() { return cuadro; },
    async obtener() { throw new Error("detalle no disponible"); },
    async ejecutar() { throw new Error("actuación no disponible"); },
  };
  const presentador = presentadorDe(fuente);
  const eventos = new Map();
  let focos = 0;
  const raiz = {
    innerHTML: "",
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) { eventos.delete(tipo); },
    querySelector(selector) {
      return selector === ".ct-exp-cabecera-expediente h3" ? { focus() { focos += 1; } } : null;
    },
    contains() { return true; },
  };
  const referencia = cuadro.expedientes[0].expediente_ref;
  const control = {
    dataset: { ctExpAbrir: referencia },
    closest(selector) { return selector === "[data-ct-exp-abrir]" ? control : null; },
  };
  const montaje = await montarModuloContratacionTemporal({ raiz, presentador });
  try {
    await eventos.get("click")({ target: control, preventDefault() {} });
    assert.equal(presentador.obtenerEstado().carga, "error");
    assert.match(raiz.innerHTML, /No se pudo cargar el expediente. Reintente desde el cuadro/u);
    assert.match(raiz.innerHTML, /role="alert"/u);
    assert.equal(focos, 0);
  } finally {
    montaje.desmontar();
  }
});
