import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { obtenerDatosPresentacion } from "../../datos-presentacion.js";
import { crearContextoActorPresentacionDesdeSesion } from "../../identidad/presentacion.js";
import { crearAdaptadorContratacionTemporalPresentacion } from "./adaptador-presentacion.js";
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
