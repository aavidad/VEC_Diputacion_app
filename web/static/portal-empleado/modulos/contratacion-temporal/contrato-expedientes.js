/**
 * Proyecciones cerradas del cuadro y el expediente de contratación temporal.
 *
 * Son contratos de presentación neutrales al transporte. La autoridad,
 * competencia y autorización de cada efecto permanecen siempre en servidor.
 */

export const CAPACIDADES_CONTRATACION_TEMPORAL = Object.freeze({
  consultarCuadro: "contratacion_temporal.cuadro.consultar",
  consultarExpediente: "contratacion_temporal.expediente.consultar",
  consultarDocumentos: "contratacion_temporal.documentos.consultar",
  crearSolicitud: "contratacion_temporal.solicitud.crear",
  enviarAnalisis: "contratacion_temporal.solicitud.enviar_analisis",
  analizar: "contratacion_temporal.analisis.validar",
  decidirCobertura: "contratacion_temporal.cobertura.decidir",
  asignarUnidad: "contratacion_temporal.unidad.asignar",
  prepararInforme: "contratacion_temporal.informe.preparar",
  firmarInforme: "contratacion_temporal.informe.firmar",
  solicitarFiscalizacion: "contratacion_temporal.fiscalizacion.solicitar",
  registrarFiscalizacion: "contratacion_temporal.fiscalizacion.registrar",
  registrarSubsanacion: "contratacion_temporal.subsanacion.registrar",
  prepararLlamamiento: "contratacion_temporal.llamamiento.preparar",
  seleccionarCandidatura: "contratacion_temporal.llamamiento.seleccionar",
  registrarResultadoLlamamiento: "contratacion_temporal.llamamiento.registrar_resultado",
  prepararFormalizacion: "contratacion_temporal.formalizacion.preparar",
  firmarFormalizacion: "contratacion_temporal.formalizacion.firmar",
  confirmarIncorporacion: "contratacion_temporal.incorporacion.confirmar",
  exportarGinpix: "contratacion_temporal.ginpix.exportar",
  enviarGinpix: "contratacion_temporal.ginpix.enviar",
  registrarSeguimiento: "contratacion_temporal.seguimiento.registrar",
  cerrarExpediente: "contratacion_temporal.expediente.cerrar",
  consultarAuditoria: "contratacion_temporal.auditoria.consultar",
});

export const LIMITES_EXPEDIENTES_PRESENTACION = Object.freeze({
  texto: 4000,
  etiqueta: 240,
  referencia: 160,
  indicadores: 32,
  expedientes: 500,
  fases: 32,
  tareas: 128,
  paneles: 32,
  campos: 128,
  columnas: 24,
  filas: 500,
  documentos: 256,
  actuaciones: 1000,
  acciones: 24,
  datosAccion: 128,
});

const ESQUEMA_CUADRO = "vec.contratacion_temporal.cuadro.v1";
const ESQUEMA_EXPEDIENTE = "vec.contratacion_temporal.expediente.v1";
const ESQUEMA_DOCUMENTOS = "vec.contratacion_temporal.documentos.v1";
const ESQUEMA_AUDITORIA = "vec.contratacion_temporal.auditoria.v1";
const ESQUEMA_COMANDO = "vec.contratacion_temporal.actuacion.v1";
const ESQUEMA_RECIBO = "vec.contratacion_temporal.recibo-actuacion.v1";
const PATRON_REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const PATRON_CLAVE = /^[a-z][a-z0-9._-]{1,79}$/u;
const PATRON_NUMERO = /^[0-9]{4}\/[A-Za-z0-9._-]{1,40}$/u;
const PATRON_HUELLA = /^[a-f0-9]{64}$/u;
const TONOS = new Set(["neutro", "informacion", "exito", "aviso", "peligro"]);
const ESTADOS = new Set([
  "pendiente", "en_curso", "espera", "completado", "incidencia", "cancelado",
]);
const TIPOS_PANEL = new Set([
  "datos", "formulario", "comprobaciones", "tabla", "documentos", "aviso",
]);
const CONTROLES = new Set([
  "solo_lectura", "texto", "area", "fecha", "seleccion", "radio", "importe",
]);
const TIPOS_ACCION = new Set(["efecto", "navegacion"]);
const VARIANTES_ACCION = new Set(["primaria", "secundaria", "peligro"]);

function esRegistro(valor) {
  if (valor === null || typeof valor !== "object" || Array.isArray(valor)) return false;
  const prototipo = Object.getPrototypeOf(valor);
  return prototipo === Object.prototype || prototipo === null;
}

function exigirCamposExactos(registro, campos, nombre) {
  if (!esRegistro(registro)) throw new TypeError(`${nombre} no válido`);
  const recibidos = Object.keys(registro);
  if (recibidos.length !== campos.length
    || recibidos.some((campo) => !campos.includes(campo))
    || campos.some((campo) => !Object.hasOwn(registro, campo))) {
    throw new TypeError(`${nombre} no respeta el contrato cerrado`);
  }
}

function cadena(valor, nombre, maximo = LIMITES_EXPEDIENTES_PRESENTACION.texto) {
  if (typeof valor !== "string" || valor !== valor.trim()
    || valor.normalize("NFC") !== valor || [...valor].length > maximo
    || /[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f-\u009f]/u.test(valor)) {
    throw new TypeError(`${nombre} no válido`);
  }
  return valor;
}

function cadenaNoVacia(valor, nombre, maximo = LIMITES_EXPEDIENTES_PRESENTACION.etiqueta) {
  const resultado = cadena(valor, nombre, maximo);
  if (resultado === "") throw new TypeError(`${nombre} no puede estar vacío`);
  return resultado;
}

function referencia(valor, nombre) {
  const resultado = cadenaNoVacia(
    valor,
    nombre,
    LIMITES_EXPEDIENTES_PRESENTACION.referencia,
  );
  if (!PATRON_REFERENCIA.test(resultado)) throw new TypeError(`${nombre} no válida`);
  return resultado;
}

function clave(valor, nombre) {
  const resultado = cadenaNoVacia(valor, nombre, 80);
  if (!PATRON_CLAVE.test(resultado)) throw new TypeError(`${nombre} no válida`);
  return resultado;
}

function instante(valor, nombre) {
  const resultado = cadenaNoVacia(valor, nombre, 40);
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/u.test(resultado)
    || !Number.isFinite(Date.parse(resultado))) {
    throw new TypeError(`${nombre} no válido`);
  }
  return resultado;
}

function lista(valor, nombre, maximo, transformar) {
  if (!Array.isArray(valor) || valor.length > maximo) {
    throw new TypeError(`${nombre} no válida`);
  }
  return valor.map((elemento, indice) => transformar(elemento, `${nombre}[${indice}]`));
}

function unicos(elementos, campo, nombre) {
  const valores = elementos.map((elemento) => elemento[campo]);
  if (new Set(valores).size !== valores.length) {
    throw new TypeError(`${nombre} contiene duplicados`);
  }
  return elementos;
}

function congelar(valor) {
  if (valor && typeof valor === "object" && !Object.isFrozen(valor)) {
    Object.values(valor).forEach(congelar);
    Object.freeze(valor);
  }
  return valor;
}

function validarIndicador(entrada, nombre) {
  exigirCamposExactos(entrada, ["clave", "etiqueta", "valor", "tono"], nombre);
  if (!TONOS.has(entrada.tono)) throw new TypeError(`${nombre}.tono no válido`);
  return {
    clave: clave(entrada.clave, `${nombre}.clave`),
    etiqueta: cadenaNoVacia(entrada.etiqueta, `${nombre}.etiqueta`),
    valor: cadenaNoVacia(entrada.valor, `${nombre}.valor`, 80),
    tono: entrada.tono,
  };
}

function validarResumen(entrada, nombre) {
  exigirCamposExactos(entrada, [
    "expediente_ref", "numero_visible", "centro", "categoria", "modalidad",
    "estado_clave", "estado", "fase_actual", "fecha_solicitud", "responsable",
    "plazo", "version",
  ], nombre);
  if (!ESTADOS.has(entrada.estado_clave)
    || !PATRON_NUMERO.test(entrada.numero_visible)
    || !Number.isSafeInteger(entrada.version) || entrada.version < 1) {
    throw new TypeError(`${nombre} no válido`);
  }
  return {
    expediente_ref: referencia(entrada.expediente_ref, `${nombre}.expediente_ref`),
    numero_visible: entrada.numero_visible,
    centro: cadenaNoVacia(entrada.centro, `${nombre}.centro`),
    categoria: cadenaNoVacia(entrada.categoria, `${nombre}.categoria`),
    modalidad: cadenaNoVacia(entrada.modalidad, `${nombre}.modalidad`),
    estado_clave: entrada.estado_clave,
    estado: cadenaNoVacia(entrada.estado, `${nombre}.estado`),
    fase_actual: cadenaNoVacia(entrada.fase_actual, `${nombre}.fase_actual`),
    fecha_solicitud: cadenaNoVacia(entrada.fecha_solicitud, `${nombre}.fecha_solicitud`, 80),
    responsable: cadenaNoVacia(entrada.responsable, `${nombre}.responsable`),
    plazo: cadenaNoVacia(entrada.plazo, `${nombre}.plazo`, 80),
    version: entrada.version,
  };
}

export function validarCuadroContratacionTemporal(entrada) {
  exigirCamposExactos(
    entrada,
    ["esquema", "demostracion", "generado_en", "indicadores", "expedientes"],
    "cuadro de contratación temporal",
  );
  if (entrada.esquema !== ESQUEMA_CUADRO || typeof entrada.demostracion !== "boolean") {
    throw new TypeError("cuadro de contratación temporal no válido");
  }
  const indicadores = unicos(lista(
    entrada.indicadores,
    "indicadores",
    LIMITES_EXPEDIENTES_PRESENTACION.indicadores,
    validarIndicador,
  ), "clave", "indicadores");
  const expedientes = unicos(lista(
    entrada.expedientes,
    "expedientes",
    LIMITES_EXPEDIENTES_PRESENTACION.expedientes,
    validarResumen,
  ), "expediente_ref", "expedientes");
  return congelar({
    esquema: ESQUEMA_CUADRO,
    demostracion: entrada.demostracion,
    generado_en: instante(entrada.generado_en, "generado_en"),
    indicadores,
    expedientes,
  });
}

function validarFase(entrada, nombre) {
  exigirCamposExactos(entrada, ["fase_ref", "orden", "etiqueta", "estado_clave"], nombre);
  if (!Number.isSafeInteger(entrada.orden) || entrada.orden < 1
    || !ESTADOS.has(entrada.estado_clave)) throw new TypeError(`${nombre} no válida`);
  return {
    fase_ref: referencia(entrada.fase_ref, `${nombre}.fase_ref`),
    orden: entrada.orden,
    etiqueta: cadenaNoVacia(entrada.etiqueta, `${nombre}.etiqueta`),
    estado_clave: entrada.estado_clave,
  };
}

function validarOpcion(entrada, nombre) {
  exigirCamposExactos(entrada, ["clave", "etiqueta"], nombre);
  return {
    clave: cadenaNoVacia(entrada.clave, `${nombre}.clave`, 160),
    etiqueta: cadenaNoVacia(entrada.etiqueta, `${nombre}.etiqueta`),
  };
}

function validarCampo(entrada, nombre) {
  exigirCamposExactos(entrada, [
    "clave", "etiqueta", "valor", "tono", "control", "obligatorio", "opciones",
  ], nombre);
  if (!TONOS.has(entrada.tono) || !CONTROLES.has(entrada.control)
    || typeof entrada.obligatorio !== "boolean") {
    throw new TypeError(`${nombre} no válido`);
  }
  return {
    clave: clave(entrada.clave, `${nombre}.clave`),
    etiqueta: cadenaNoVacia(entrada.etiqueta, `${nombre}.etiqueta`),
    valor: cadena(entrada.valor, `${nombre}.valor`),
    tono: entrada.tono,
    control: entrada.control,
    obligatorio: entrada.obligatorio,
    opciones: unicos(lista(
      entrada.opciones,
      `${nombre}.opciones`,
      LIMITES_EXPEDIENTES_PRESENTACION.campos,
      validarOpcion,
    ), "clave", `${nombre}.opciones`),
  };
}

function validarColumna(entrada, nombre) {
  exigirCamposExactos(entrada, ["clave", "etiqueta"], nombre);
  return {
    clave: clave(entrada.clave, `${nombre}.clave`),
    etiqueta: cadenaNoVacia(entrada.etiqueta, `${nombre}.etiqueta`),
  };
}

function validarFila(entrada, nombre, columnas) {
  exigirCamposExactos(entrada, ["fila_ref", "celdas"], nombre);
  const celdas = lista(
    entrada.celdas,
    `${nombre}.celdas`,
    LIMITES_EXPEDIENTES_PRESENTACION.columnas,
    (valor, campo) => cadena(valor, campo, LIMITES_EXPEDIENTES_PRESENTACION.etiqueta),
  );
  if (celdas.length !== columnas) throw new TypeError(`${nombre} no coincide con sus columnas`);
  return { fila_ref: referencia(entrada.fila_ref, `${nombre}.fila_ref`), celdas };
}

function validarPanel(entrada, nombre) {
  exigirCamposExactos(entrada, [
    "panel_ref", "tipo", "titulo", "descripcion", "campos", "columnas", "filas",
  ], nombre);
  if (!TIPOS_PANEL.has(entrada.tipo)) throw new TypeError(`${nombre}.tipo no válido`);
  const columnas = unicos(lista(
    entrada.columnas,
    `${nombre}.columnas`,
    LIMITES_EXPEDIENTES_PRESENTACION.columnas,
    validarColumna,
  ), "clave", `${nombre}.columnas`);
  const filas = lista(
    entrada.filas,
    `${nombre}.filas`,
    LIMITES_EXPEDIENTES_PRESENTACION.filas,
    (fila, campo) => validarFila(fila, campo, columnas.length),
  );
  return {
    panel_ref: referencia(entrada.panel_ref, `${nombre}.panel_ref`),
    tipo: entrada.tipo,
    titulo: cadenaNoVacia(entrada.titulo, `${nombre}.titulo`),
    descripcion: cadena(entrada.descripcion, `${nombre}.descripcion`),
    campos: unicos(lista(
      entrada.campos,
      `${nombre}.campos`,
      LIMITES_EXPEDIENTES_PRESENTACION.campos,
      validarCampo,
    ), "clave", `${nombre}.campos`),
    columnas,
    filas: unicos(filas, "fila_ref", `${nombre}.filas`),
  };
}

function validarAccion(entrada, nombre) {
  exigirCamposExactos(entrada, [
    "accion_ref", "etiqueta", "tipo", "variante", "capacidad",
    "confirmacion", "destino_tarea_ref", "disponible", "motivo_no_disponible",
  ], nombre);
  if (!TIPOS_ACCION.has(entrada.tipo) || !VARIANTES_ACCION.has(entrada.variante)
    || typeof entrada.disponible !== "boolean") {
    throw new TypeError(`${nombre} no válida`);
  }
  const capacidad = entrada.tipo === "efecto"
    ? clave(entrada.capacidad, `${nombre}.capacidad`) : cadena(entrada.capacidad, `${nombre}.capacidad`);
  const destino = entrada.tipo === "navegacion"
    ? referencia(entrada.destino_tarea_ref, `${nombre}.destino_tarea_ref`)
    : cadena(entrada.destino_tarea_ref, `${nombre}.destino_tarea_ref`);
  return {
    accion_ref: clave(entrada.accion_ref, `${nombre}.accion_ref`),
    etiqueta: cadenaNoVacia(entrada.etiqueta, `${nombre}.etiqueta`),
    tipo: entrada.tipo,
    variante: entrada.variante,
    capacidad,
    confirmacion: cadena(entrada.confirmacion, `${nombre}.confirmacion`),
    destino_tarea_ref: destino,
    disponible: entrada.disponible,
    motivo_no_disponible: cadena(
      entrada.motivo_no_disponible,
      `${nombre}.motivo_no_disponible`,
      LIMITES_EXPEDIENTES_PRESENTACION.etiqueta,
    ),
  };
}

function validarTarea(entrada, nombre) {
  exigirCamposExactos(entrada, [
    "tarea_ref", "orden", "fase_ref", "etiqueta", "descripcion", "estado_clave",
    "estado", "unidad", "responsable", "entrada", "salida", "tiempo",
    "recibo_ref", "decision_ref", "paneles", "acciones",
  ], nombre);
  if (!Number.isSafeInteger(entrada.orden) || entrada.orden < 1
    || !ESTADOS.has(entrada.estado_clave)) throw new TypeError(`${nombre} no válida`);
  return {
    tarea_ref: referencia(entrada.tarea_ref, `${nombre}.tarea_ref`),
    orden: entrada.orden,
    fase_ref: referencia(entrada.fase_ref, `${nombre}.fase_ref`),
    etiqueta: cadenaNoVacia(entrada.etiqueta, `${nombre}.etiqueta`),
    descripcion: cadenaNoVacia(entrada.descripcion, `${nombre}.descripcion`),
    estado_clave: entrada.estado_clave,
    estado: cadenaNoVacia(entrada.estado, `${nombre}.estado`),
    unidad: cadenaNoVacia(entrada.unidad, `${nombre}.unidad`),
    responsable: cadenaNoVacia(entrada.responsable, `${nombre}.responsable`),
    entrada: cadenaNoVacia(entrada.entrada, `${nombre}.entrada`, 80),
    salida: cadena(entrada.salida, `${nombre}.salida`, 80),
    tiempo: cadenaNoVacia(entrada.tiempo, `${nombre}.tiempo`, 80),
    recibo_ref: entrada.recibo_ref === ""
      ? "" : referencia(entrada.recibo_ref, `${nombre}.recibo_ref`),
    decision_ref: entrada.decision_ref === ""
      ? "" : referencia(entrada.decision_ref, `${nombre}.decision_ref`),
    paneles: unicos(lista(
      entrada.paneles,
      `${nombre}.paneles`,
      LIMITES_EXPEDIENTES_PRESENTACION.paneles,
      validarPanel,
    ), "panel_ref", `${nombre}.paneles`),
    acciones: unicos(lista(
      entrada.acciones,
      `${nombre}.acciones`,
      LIMITES_EXPEDIENTES_PRESENTACION.acciones,
      validarAccion,
    ), "accion_ref", `${nombre}.acciones`),
  };
}

function validarDocumento(entrada, nombre) {
  exigirCamposExactos(entrada, [
    "documento_ref", "titulo", "tipo", "version", "estado", "firma",
    "fecha", "descarga_disponible",
  ], nombre);
  if (!Number.isSafeInteger(entrada.version) || entrada.version < 1
    || typeof entrada.descarga_disponible !== "boolean") {
    throw new TypeError(`${nombre} no válido`);
  }
  return {
    documento_ref: referencia(entrada.documento_ref, `${nombre}.documento_ref`),
    titulo: cadenaNoVacia(entrada.titulo, `${nombre}.titulo`),
    tipo: cadenaNoVacia(entrada.tipo, `${nombre}.tipo`, 80),
    version: entrada.version,
    estado: cadenaNoVacia(entrada.estado, `${nombre}.estado`, 80),
    firma: cadenaNoVacia(entrada.firma, `${nombre}.firma`, 160),
    fecha: cadenaNoVacia(entrada.fecha, `${nombre}.fecha`, 80),
    descarga_disponible: entrada.descarga_disponible,
  };
}

function validarActuacion(entrada, nombre) {
  exigirCamposExactos(entrada, [
    "actuacion_ref", "fecha", "fase", "accion", "actor", "unidad",
    "estado", "observaciones", "documento_ref",
  ], nombre);
  return {
    actuacion_ref: referencia(entrada.actuacion_ref, `${nombre}.actuacion_ref`),
    fecha: cadenaNoVacia(entrada.fecha, `${nombre}.fecha`, 80),
    fase: cadenaNoVacia(entrada.fase, `${nombre}.fase`),
    accion: cadenaNoVacia(entrada.accion, `${nombre}.accion`),
    actor: cadenaNoVacia(entrada.actor, `${nombre}.actor`),
    unidad: cadenaNoVacia(entrada.unidad, `${nombre}.unidad`),
    estado: cadenaNoVacia(entrada.estado, `${nombre}.estado`, 80),
    observaciones: cadena(entrada.observaciones, `${nombre}.observaciones`),
    documento_ref: entrada.documento_ref === ""
      ? "" : referencia(entrada.documento_ref, `${nombre}.documento_ref`),
  };
}

export function validarExpedienteContratacionTemporal(entrada) {
  exigirCamposExactos(entrada, [
    "esquema", "demostracion", "expediente_ref", "numero_visible", "version",
    "flujo_ref", "flujo_version", "flujo_huella", "cabecera", "fases", "tareas",
  ], "expediente de contratación temporal");
  if (entrada.esquema !== ESQUEMA_EXPEDIENTE || typeof entrada.demostracion !== "boolean"
    || !PATRON_NUMERO.test(entrada.numero_visible)
    || !Number.isSafeInteger(entrada.version) || entrada.version < 1
    || !Number.isSafeInteger(entrada.flujo_version) || entrada.flujo_version < 1
    || typeof entrada.flujo_huella !== "string" || !PATRON_HUELLA.test(entrada.flujo_huella)) {
    throw new TypeError("expediente de contratación temporal no válido");
  }
  const fases = unicos(lista(
    entrada.fases,
    "fases",
    LIMITES_EXPEDIENTES_PRESENTACION.fases,
    validarFase,
  ), "fase_ref", "fases");
  const referenciasFases = new Set(fases.map(({ fase_ref: valor }) => valor));
  const tareas = unicos(lista(
    entrada.tareas,
    "tareas",
    LIMITES_EXPEDIENTES_PRESENTACION.tareas,
    validarTarea,
  ), "tarea_ref", "tareas");
  if (tareas.some((tarea) => !referenciasFases.has(tarea.fase_ref))) {
    throw new TypeError("una tarea referencia una fase inexistente");
  }
  const referenciasTareas = new Set(tareas.map(({ tarea_ref: valor }) => valor));
  if (tareas.some((tarea) => tarea.acciones.some(
    (accion) => accion.tipo === "navegacion"
      && !referenciasTareas.has(accion.destino_tarea_ref),
  ))) throw new TypeError("una acción referencia una tarea inexistente");
  return congelar({
    esquema: ESQUEMA_EXPEDIENTE,
    demostracion: entrada.demostracion,
    expediente_ref: referencia(entrada.expediente_ref, "expediente_ref"),
    numero_visible: entrada.numero_visible,
    version: entrada.version,
    flujo_ref: referencia(entrada.flujo_ref, "flujo_ref"),
    flujo_version: entrada.flujo_version,
    flujo_huella: entrada.flujo_huella,
    cabecera: unicos(lista(
      entrada.cabecera,
      "cabecera",
      LIMITES_EXPEDIENTES_PRESENTACION.campos,
      validarCampo,
    ), "clave", "cabecera"),
    fases,
    tareas,
  });
}

export function validarDocumentosContratacionTemporal(entrada) {
  exigirCamposExactos(entrada, [
    "esquema", "demostracion", "expediente_ref", "version", "documentos",
  ], "índice documental de contratación temporal");
  if (entrada.esquema !== ESQUEMA_DOCUMENTOS || typeof entrada.demostracion !== "boolean"
    || !Number.isSafeInteger(entrada.version) || entrada.version < 1) {
    throw new TypeError("índice documental de contratación temporal no válido");
  }
  return congelar({
    esquema: ESQUEMA_DOCUMENTOS,
    demostracion: entrada.demostracion,
    expediente_ref: referencia(entrada.expediente_ref, "expediente_ref"),
    version: entrada.version,
    documentos: unicos(lista(
      entrada.documentos,
      "documentos",
      LIMITES_EXPEDIENTES_PRESENTACION.documentos,
      validarDocumento,
    ), "documento_ref", "documentos"),
  });
}

export function validarAuditoriaContratacionTemporal(entrada) {
  exigirCamposExactos(entrada, [
    "esquema", "demostracion", "expediente_ref", "version", "actuaciones",
  ], "auditoría de contratación temporal");
  if (entrada.esquema !== ESQUEMA_AUDITORIA || typeof entrada.demostracion !== "boolean"
    || !Number.isSafeInteger(entrada.version) || entrada.version < 1) {
    throw new TypeError("auditoría de contratación temporal no válida");
  }
  return congelar({
    esquema: ESQUEMA_AUDITORIA,
    demostracion: entrada.demostracion,
    expediente_ref: referencia(entrada.expediente_ref, "expediente_ref"),
    version: entrada.version,
    actuaciones: unicos(lista(
      entrada.actuaciones,
      "actuaciones",
      LIMITES_EXPEDIENTES_PRESENTACION.actuaciones,
      validarActuacion,
    ), "actuacion_ref", "actuaciones"),
  });
}

export function validarComandoActuacion(entrada) {
  exigirCamposExactos(entrada, [
    "esquema", "expediente_ref", "version_esperada", "tarea_ref", "accion_ref", "datos",
  ], "comando de actuación");
  if (entrada.esquema !== ESQUEMA_COMANDO
    || !Number.isSafeInteger(entrada.version_esperada) || entrada.version_esperada < 1
    || !esRegistro(entrada.datos)
    || Object.keys(entrada.datos).length > LIMITES_EXPEDIENTES_PRESENTACION.datosAccion) {
    throw new TypeError("comando de actuación no válido");
  }
  const datos = {};
  for (const [campo, valor] of Object.entries(entrada.datos)) {
    datos[clave(campo, `datos.${campo}`)] = cadena(valor, `datos.${campo}`);
  }
  return congelar({
    esquema: ESQUEMA_COMANDO,
    expediente_ref: referencia(entrada.expediente_ref, "expediente_ref"),
    version_esperada: entrada.version_esperada,
    tarea_ref: referencia(entrada.tarea_ref, "tarea_ref"),
    accion_ref: clave(entrada.accion_ref, "accion_ref"),
    datos,
  });
}

export function validarReciboActuacion(entrada) {
  exigirCamposExactos(entrada, [
    "esquema", "recibo_ref", "expediente_ref", "numero_visible", "version",
    "actuacion", "estado_resultante", "registrada_en",
  ], "recibo de actuación");
  if (entrada.esquema !== ESQUEMA_RECIBO || !PATRON_NUMERO.test(entrada.numero_visible)
    || !Number.isSafeInteger(entrada.version) || entrada.version < 1) {
    throw new TypeError("recibo de actuación no válido");
  }
  return congelar({
    esquema: ESQUEMA_RECIBO,
    recibo_ref: referencia(entrada.recibo_ref, "recibo_ref"),
    expediente_ref: referencia(entrada.expediente_ref, "expediente_ref"),
    numero_visible: entrada.numero_visible,
    version: entrada.version,
    actuacion: cadenaNoVacia(entrada.actuacion, "actuacion"),
    estado_resultante: cadenaNoVacia(entrada.estado_resultante, "estado_resultante"),
    registrada_en: instante(entrada.registrada_en, "registrada_en"),
  });
}
