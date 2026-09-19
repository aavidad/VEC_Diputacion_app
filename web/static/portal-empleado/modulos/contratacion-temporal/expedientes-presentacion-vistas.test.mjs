import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { obtenerDatosPresentacion } from "../../datos-presentacion.js";
import { crearContextoActorPresentacionDesdeSesion } from "../../identidad/presentacion.js";
import { crearAdaptadorContratacionTemporalPresentacion } from "./adaptador-presentacion.js";
import { renderizarExpediente } from "./componentes-expedientes.js";
import {
  CAPACIDADES_CONTRATACION_TEMPORAL as CAP,
  validarComandoActuacion,
  validarExpedienteContratacionTemporal,
  validarAuditoriaContratacionTemporal,
  validarCuadroContratacionTemporal,
} from "./contrato-expedientes.js";
import { crearBorradorAlta, crearComandoAlta } from "./contrato.js";
import {
  crearAuditoriaContratacionTemporalPresentacion,
  crearCuadroContratacionTemporalPresentacion,
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

test("el alta crea un expediente nuevo mínimo sin heredar candidato ni documentos", async () => {
  const fuente = adaptador();
  const catalogos = fuente.obtenerCatalogosAlta();
  const base = crearBorradorAlta();
  const borrador = {
    ...base,
    centro_ref: catalogos.centros[0].referencia,
    contacto_ref: catalogos.centros[0].contactos[0].referencia,
    categoria_ref: catalogos.categorias[0].referencia,
    grupo_subgrupo: catalogos.categorias[0].grupos_subgrupos[0].clave,
    motivo_clave: catalogos.motivos[0].clave,
    detalle: "Necesidad sintética para validar el alta coherente.",
    inicio: "2026-08-15",
    fin: "2027-04-14",
    documentos_adjuntos: [catalogos.documentos[0].referencia],
  };
  const comando = crearComandoAlta(
    borrador,
    catalogos,
    "12345678-1234-4abc-8def-1234567890ab",
  );
  const recibo = await fuente.registrarSolicitud(comando);
  const cuadro = await fuente.listar();
  assert.equal(cuadro.expedientes[0].expediente_ref, recibo.expediente_ref);
  const detalle = await fuente.obtener(recibo.expediente_ref);
  const documentos = await fuente.obtenerDocumentos(recibo.expediente_ref);
  const auditoria = await fuente.obtenerAuditoria(recibo.expediente_ref);
  assert.equal(detalle.tareas[0].estado_clave, "en_curso");
  assert.ok(detalle.tareas.slice(1).every(({ estado_clave }) => estado_clave === "pendiente"));
  assert.doesNotMatch(JSON.stringify(detalle), /CAND-DEMO|fiscalización favorable/i);
  assert.equal(documentos.documentos.length, 0);
  assert.equal(auditoria.actuaciones.length, 1);
});

test("el alta ejecuta un solo efecto y conserva su recibo sin refresco automático", async () => {
  const recibo = Object.freeze({
    expediente_ref: "expediente:ct:real:001",
    numero_visible: "2026/CT-0001",
    version: 1,
    recibo_ref: "recibo:ct:real:001",
    confirmada_en: "2026-09-04T07:55:00Z",
  });
  let altas = 0;
  let refrescos = 0;
  const ejecutar = async () => {
    altas += 1;
    return recibo;
  };
  const presentador = {
    async cargar() {
      refrescos += 1;
      throw new Error("cuadro todavía no compuesto");
    },
  };
  const ejecutarConRefresco = crearEjecutorAltaConRefresco(ejecutar, presentador);

  assert.deepEqual(await ejecutarConRefresco({}, {}), recibo);
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(altas, 1);
  assert.equal(refrescos, 0);
});

test("el alta no inicia un refresco pendiente que pueda retirar el análisis", async () => {
  const recibo = Object.freeze({
    expediente_ref: "expediente:ct:real:pendiente",
    numero_visible: "2026/CT-0002",
    version: 1,
    recibo_ref: "recibo:ct:real:pendiente",
    confirmada_en: "2026-09-04T08:05:00Z",
  });
  const refrescoPendiente = new Promise(() => {});
  const eventos = [];
  let montajes = 0;
  let refrescos = 0;
  const ejecutarConRefresco = crearEjecutorAltaConRefresco(
    async () => {
      eventos.push("alta");
      return recibo;
    },
    {
      cargar() {
        refrescos += 1;
        eventos.push("refresco");
        return refrescoPendiente;
      },
    },
    (confirmado) => {
      montajes += 1;
      eventos.push("analisis");
      assert.deepEqual(confirmado, recibo);
    },
  );

  const resultado = await Promise.race([
    ejecutarConRefresco({}, {}),
    new Promise((_, reject) => setImmediate(() => {
      reject(new Error("el alta quedó bloqueada por el refresco"));
    })),
  ]);

  assert.deepEqual(resultado, recibo);
  await new Promise((resolve) => setImmediate(resolve));
  assert.deepEqual(eventos, ["alta", "analisis"]);
  assert.equal(montajes, 1);
  assert.equal(refrescos, 0);
});

test("un refresco satisfactorio conserva la vista de alta", async () => {
  const presentador = presentadorDe(adaptador());
  await presentador.cargar();
  presentador.cambiarVista("alta");
  await presentador.cargar();
  assert.equal(presentador.obtenerEstado().vista, "alta");
});

test("HTML escapa contenido, bloquea históricos y expone semántica accesible", () => {
  const entrada = crearExpedienteContratacionTemporalPresentacion();
  entrada.cabecera[0].valor = '<img src=x onerror="alert(1)">';
  const expediente = validarExpedienteContratacionTemporal(entrada);
  const t = crearTraductorExpedientesContratacion();
  const estado = estadoVista(expediente, "tarea-analisis");
  const html = renderizarModuloContratacionTemporal(estado);
  assert.match(html, /&lt;img src=x onerror=&quot;alert\(1\)&quot;&gt;/);
  assert.doesNotMatch(html, /<img src=x|style="/);
  assert.match(html, /aria-labelledby="ct-exp-titulo"/);
  assert.match(html, /aria-current="step"/);
  assert.match(html, /Vista histórica o de consulta/);
  assert.match(html, /<select[^>]+disabled/);
  const htmlComponente = renderizarExpediente(estado, t, "es-ES", "Europe/Madrid");
  assert.match(htmlComponente, /<nav class="ct-exp-tareas" aria-label=/);
  assert.match(htmlComponente, /<details class="ct-exp-detalle-tecnico">/);
  assert.doesNotMatch(htmlComponente, /<details class="ct-exp-detalle-tecnico" open/);
  assert.match(htmlComponente, /Metadatos técnicos del expediente/);
});

test("el identificador completo puede envolver y los paneles vacíos no ocultan auditoría", async () => {
  const css = await readFile(new URL("./expedientes.css", import.meta.url), "utf8");
  assert.match(css, /\.ct-exp-cabecera-expediente > div\s*\{\s*min-width: 0;\s*\}/u);
  assert.match(css, /\.ct-exp-cabecera-expediente h3\s*\{\s*overflow-wrap: anywhere;\s*\}/u);
  const expediente = validarExpedienteContratacionTemporal({
    ...crearExpedienteContratacionTemporalPresentacion(),
    numero_visible: "2026/CT-" + "b".repeat(32),
    fases: [], tareas: [],
  });
  const estado = estadoVista(expediente, "");
  const html = renderizarModuloContratacionTemporal(estado);
  assert.ok(html.includes(`<h3>${expediente.numero_visible}</h3>`));
  assert.match(html, /ct-exp-cabecera-expediente/u);
  assert.doesNotMatch(html, /class="ct-exp-(?:progreso|tareas|tramitacion)"/u);
  const auditoria = validarAuditoriaContratacionTemporal(
    crearAuditoriaContratacionTemporalPresentacion(),
  );
  const htmlAuditoria = renderizarModuloContratacionTemporal({
    ...estado, vista: "auditoria", auditoria,
  });
  assert.ok(auditoria.actuaciones.length > 0);
  assert.match(htmlAuditoria, /ct-exp-tabla-auditoria/u);
  for (const actuacion of auditoria.actuaciones) {
    assert.ok(htmlAuditoria.includes(actuacion.fecha));
  }
});

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
    ["tarea-envio-intervencion", "Vista previa y circuito de firma"],
    ["tarea-fiscalizacion", "Modalidad y remisión"],
    ["tarea-subsanacion", "Observaciones, correcciones y evidencias"],
    ["tarea-iniciar-llamamiento", "Historial de llamamientos"],
    ["tarea-seleccion-candidato", "Candidatura seleccionada"],
    ["tarea-resultado-llamamiento", "Resumen e historial de la candidatura"],
    ["tarea-traslado-intervencion", "Tarjeta minimizada de candidatura"],
    ["tarea-informe-definitivo", "Candidatura, observaciones e historial"],
    ["tarea-formalizacion", "Subpasos de formalización"],
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
  let expediente = await fuente.obtener(referencia);

  async function completar(tareaRef, accionRef) {
    const tarea = expediente.tareas.find(({ tarea_ref: actual }) => actual === tareaRef);
    const accion = tarea.acciones.find(({ accion_ref: actual }) => actual === accionRef);
    await fuente.ejecutar(comandoDe(expediente, tarea, accion));
    expediente = await fuente.obtener(referencia);
  }

  await completar("tarea-formalizacion", "generar_documentos_formalizacion");
  await completar("tarea-formalizacion", "enviar_firma_formalizacion");
  await completar("tarea-incorporacion", "confirmar_incorporacion");
  await completar("tarea-ginpix", "generar_fichero_ginpix");

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
