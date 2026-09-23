const ESQUEMA_PANEL = "vec.bolsa.area-personal.v1";
export const ESQUEMA_MI_BOLSA = "vec.bolsa.mi-bolsa.v1";
const ESQUEMA_RECIBO = "vec.bolsa.area-personal.recibo.v1";
const ESQUEMA_RECIBO_PRESENTACION = "vec.bolsa.area-personal.recibo-demo.v1";
const PATRON_DNI_NIE = /\b(?:[XYZ]\d{7}[A-Z]|\d{8}[A-Z])\b/i;
const PATRON_CORREO = /\b[A-Z0-9._%+-]+@([A-Z0-9.-]+)\b/gi;
const MAXIMO_ELEMENTOS = 200;

export const SITUACIONES_PARTICIPACION_BOLSA = Object.freeze([
  "disponible",
  "ocupado",
  "no_disponible",
  "excluido",
  "renuncia_pendiente",
]);

const SITUACIONES_ACTUALES_MI_BOLSA = Object.freeze([
  "disponible", "no_disponible", "trabajando", "pendiente_incorporacion",
  "renuncia", "excluido", "disponible_desde",
]);

export const RESULTADOS_LLAMAMIENTO = Object.freeze([
  "aceptado",
  "renuncia",
  "sin_respuesta",
  "pendiente",
]);

export const MOTIVOS_PAUSA_DISPONIBILIDAD = Object.freeze([
  "pausa_voluntaria",
  "incorporacion_otro_empleo",
  "enfermedad",
  "otro",
]);

function exigirObjeto(valor, nombre) {
  if (valor === null || typeof valor !== "object" || Array.isArray(valor)) {
    throw new TypeError(`${nombre} debe ser un objeto.`);
  }
  return valor;
}

function exigirCadena(valor, nombre, maximo = 500) {
  if (typeof valor !== "string" || valor.trim() === "" || valor.length > maximo) {
    throw new TypeError(`${nombre} debe ser una cadena no vacía de hasta ${maximo} caracteres.`);
  }
  return valor;
}

function exigirBooleano(valor, nombre) {
  if (typeof valor !== "boolean") throw new TypeError(`${nombre} debe ser booleano.`);
  return valor;
}

function exigirNumero(valor, nombre, { minimo = 0, maximo = 1_000_000 } = {}) {
  if (!Number.isFinite(valor) || valor < minimo || valor > maximo) {
    throw new TypeError(`${nombre} debe ser un número entre ${minimo} y ${maximo}.`);
  }
  return valor;
}

function exigirLista(valor, nombre) {
  if (!Array.isArray(valor) || valor.length > MAXIMO_ELEMENTOS) {
    throw new TypeError(`${nombre} debe ser una lista de hasta ${MAXIMO_ELEMENTOS} elementos.`);
  }
  return valor;
}

function exigirReferencia(valor, nombre, { demostracion = false } = {}) {
  const referencia = exigirCadena(valor, nombre, 100);
  if (!/^[A-Z0-9][A-Z0-9._/:-]*$/i.test(referencia)) throw new TypeError(`${nombre} no es una referencia opaca válida.`);
  if (demostracion && !referencia.startsWith("DEMO-")) throw new TypeError(`${nombre} debe comenzar por DEMO-.`);
  return referencia;
}

function exigirInstante(valor, nombre) {
  const instante = exigirCadena(valor, nombre, 40);
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/.test(instante)) {
    throw new TypeError(`${nombre} debe ser un instante UTC ISO 8601.`);
  }
  return instante;
}

function exigirFechaOInstante(valor, nombre) {
  if (typeof valor !== "string" || valor.trim() === "" || valor.length > 50) {
    throw new TypeError(`${nombre} debe ser una fecha o instante válido.`);
  }
  const esFecha = /^\d{4}-\d{2}-\d{2}$/.test(valor);
  const esInstante = Number.isFinite(Date.parse(valor));
  if (!esFecha && !esInstante) {
    throw new TypeError(`${nombre} no tiene formato de fecha (AAAA-MM-DD) o instante ISO 8601 válido.`);
  }
  return valor;
}

function recorrerCadenas(valor, visita) {
  if (typeof valor === "string") {
    visita(valor);
    return;
  }
  if (Array.isArray(valor)) {
    valor.forEach((item) => recorrerCadenas(item, visita));
    return;
  }
  if (valor && typeof valor === "object") {
    Object.values(valor).forEach((item) => recorrerCadenas(item, visita));
  }
}

function validarPrivacidadPresentacion(datos) {
  recorrerCadenas(datos, (cadena) => {
    if (PATRON_DNI_NIE.test(cadena)) throw new TypeError("La presentación no admite DNI o NIE formalmente válidos.");
    PATRON_CORREO.lastIndex = 0;
    for (const coincidencia of cadena.matchAll(PATRON_CORREO)) {
      if (!coincidencia[1].toLowerCase().endsWith(".test")) {
        throw new TypeError("Los correos de presentación deben usar un dominio .test.");
      }
    }
  });
}

function validarListaObjetos(datos, campo, camposCadena = []) {
  exigirLista(datos[campo], campo).forEach((elemento, indice) => {
    const objeto = exigirObjeto(elemento, `${campo}[${indice}]`);
    camposCadena.forEach((nombre) => exigirCadena(objeto[nombre], `${campo}[${indice}].${nombre}`));
  });
}

function congelarProfundo(valor) {
  if (valor && typeof valor === "object" && !Object.isFrozen(valor)) {
    Object.values(valor).forEach(congelarProfundo);
    Object.freeze(valor);
  }
  return valor;
}

export function validarRespuestaMiBolsa(entrada) {
  const datos = structuredClone(exigirObjeto(exigirObjeto(entrada, "respuesta mi-bolsa").data, "respuesta mi-bolsa.data"));
  if (exigirCadena(datos.esquema, "mi-bolsa.esquema", 80) !== ESQUEMA_MI_BOLSA) {
    throw new TypeError("El esquema de mi bolsa no es compatible.");
  }
  exigirInstante(datos.consultada_en, "mi-bolsa.consultada_en");
  exigirLista(datos.participaciones, "mi-bolsa.participaciones").forEach((participacion, indice) => {
    const item = exigirObjeto(participacion, `mi-bolsa.participaciones[${indice}]`);
    exigirReferencia(item.bolsa, `mi-bolsa.participaciones[${indice}].bolsa`);
    exigirCadena(item.categoria, `mi-bolsa.participaciones[${indice}].categoria`, 200);
    exigirNumero(item.version, `mi-bolsa.participaciones[${indice}].version`, { minimo: 1 });
    exigirNumero(item.orden_inicial, `mi-bolsa.participaciones[${indice}].orden_inicial`, { minimo: 1 });
    exigirNumero(item.total_instantanea, `mi-bolsa.participaciones[${indice}].total_instantanea`, { minimo: 0 });
    exigirCadena(item.estado_bolsa, `mi-bolsa.participaciones[${indice}].estado_bolsa`, 80);
    exigirFechaOInstante(item.vigente_desde, `mi-bolsa.participaciones[${indice}].vigente_desde`);
    if (item.vigente_hasta !== null) exigirFechaOInstante(item.vigente_hasta, `mi-bolsa.participaciones[${indice}].vigente_hasta`);
    if (item.situacion_actual !== undefined && item.situacion_actual !== null) {
      const actual = exigirObjeto(item.situacion_actual, `mi-bolsa.participaciones[${indice}].situacion_actual`);
      if (!SITUACIONES_ACTUALES_MI_BOLSA.includes(actual.estado)) throw new TypeError("La situación actual de mi bolsa no es válida.");
      exigirFechaOInstante(actual.desde, `mi-bolsa.participaciones[${indice}].situacion_actual.desde`);
      if (actual.hasta !== null) exigirFechaOInstante(actual.hasta, `mi-bolsa.participaciones[${indice}].situacion_actual.hasta`);
      if (actual.fecha_disponible !== null) exigirFechaOInstante(actual.fecha_disponible, `mi-bolsa.participaciones[${indice}].situacion_actual.fecha_disponible`);
      if ((actual.estado === "disponible_desde") !== (actual.fecha_disponible !== null)) throw new TypeError("La fecha de disponibilidad no corresponde a la situación actual.");
      if (actual.hasta !== null && Date.parse(actual.hasta) < Date.parse(actual.desde)) throw new TypeError("La situación actual termina antes de comenzar.");
    }
  });
  return congelarProfundo(datos);
}

export function validarDatosAreaPersonal(entrada, { presentacionEsperada = false } = {}) {
  const datos = structuredClone(exigirObjeto(entrada, "datos"));
  const meta = exigirObjeto(datos.meta, "meta");
  if (exigirCadena(meta.esquema, "meta.esquema", 80) !== ESQUEMA_PANEL) {
    throw new TypeError("El esquema del área personal no es compatible.");
  }
  const presentacion = exigirBooleano(meta.presentacion, "meta.presentacion");
  if (presentacion !== presentacionEsperada) throw new TypeError("El origen no coincide con el modo solicitado.");
  exigirCadena(meta.origen, "meta.origen", 120);
  exigirInstante(meta.generado_en, "meta.generado_en");

  const sesion = exigirObjeto(datos.sesion, "sesion");
  exigirCadena(sesion.nombre_visible, "sesion.nombre_visible", 100);
  exigirCadena(sesion.iniciales, "sesion.iniciales", 4);
  exigirCadena(sesion.metodo, "sesion.metodo", 100);
  exigirReferencia(sesion.persona_ref, "sesion.persona_ref", { demostracion: presentacion });

  const resumen = exigirObjeto(datos.resumen, "resumen");
  ["acciones_pendientes", "convocatorias_abiertas", "solicitudes_activas", "mensajes_no_leidos"]
    .forEach((campo) => exigirNumero(resumen[campo], `resumen.${campo}`, { maximo: 10_000 }));
  exigirNumero(resumen.puntuacion_provisional, "resumen.puntuacion_provisional", { maximo: 10_000 });

  const perfil = exigirObjeto(datos.perfil, "perfil");
  exigirReferencia(perfil.referencia, "perfil.referencia", { demostracion: presentacion });
  ["nombre_visible", "identificador_visible", "correo", "telefono", "domicilio", "estado_verificacion"]
    .forEach((campo) => exigirCadena(perfil[campo], `perfil.${campo}`));

  validarListaObjetos(datos, "plazos", ["id", "dia", "mes", "titulo", "detalle", "estado"]);
  validarListaObjetos(datos, "convocatorias", ["id", "referencia", "titulo", "categoria", "estado", "plazo", "descripcion"]);
  validarListaObjetos(datos, "meritos", ["id", "tipo", "titulo", "detalle", "estado", "documento_ref"]);
  validarListaObjetos(datos, "solicitudes", ["id", "convocatoria_id", "referencia", "titulo", "estado", "actualizado"]);
  validarListaObjetos(datos, "baremo", ["id", "nombre", "detalle", "estado"]);
  validarListaObjetos(datos, "llamamientos", ["id", "bolsa", "puesto", "plazo", "estado"]);
  datos.llamamientos.forEach((item, indice) => {
    if (item.canal !== undefined && item.canal !== null) {
      exigirCadena(item.canal, `llamamientos[${indice}].canal`, 100);
    }
    if (item.comunicado_en !== undefined && item.comunicado_en !== null) {
      exigirFechaOInstante(item.comunicado_en, `llamamientos[${indice}].comunicado_en`);
    }
    if (item.resultado_clave !== undefined && item.resultado_clave !== null) {
      if (!RESULTADOS_LLAMAMIENTO.includes(item.resultado_clave)) {
        throw new TypeError(`llamamientos[${indice}].resultado_clave no reconocido en el catálogo: ${item.resultado_clave}`);
      }
    }
  });
  validarListaObjetos(datos, "subsanaciones", ["id", "solicitud_ref", "motivo", "plazo", "estado"]);
  validarListaObjetos(datos, "alegaciones", ["id", "solicitud_ref", "asunto", "estado", "fecha"]);
  validarListaObjetos(datos, "mensajes", ["id", "asunto", "resumen", "fecha", "estado"]);
  validarListaObjetos(datos, "certificados", ["id", "tipo", "descripcion", "estado"]);
  validarListaObjetos(datos, "documentos", ["id", "nombre", "tipo", "fecha", "estado"]);
  validarListaObjetos(datos, "actividad", ["id", "titulo", "detalle", "fecha", "actor"]);
  exigirObjeto(datos.disponibilidad, "disponibilidad");
  exigirBooleano(datos.disponibilidad.disponible, "disponibilidad.disponible");
  exigirCadena(datos.disponibilidad.estado, "disponibilidad.estado");
  if (datos.disponibilidad.estado_clave !== undefined && datos.disponibilidad.estado_clave !== null) {
    if (!SITUACIONES_PARTICIPACION_BOLSA.includes(datos.disponibilidad.estado_clave)) {
      throw new TypeError(`disponibilidad.estado_clave no reconocido en el catálogo: ${datos.disponibilidad.estado_clave}`);
    }
  }
  if (datos.disponibilidad.estado_desde !== undefined && datos.disponibilidad.estado_desde !== null) {
    exigirFechaOInstante(datos.disponibilidad.estado_desde, "disponibilidad.estado_desde");
  }
  if (datos.disponibilidad.disponible_desde !== undefined && datos.disponibilidad.disponible_desde !== null) {
    exigirFechaOInstante(datos.disponibilidad.disponible_desde, "disponibilidad.disponible_desde");
  }
  if (datos.disponibilidad.motivo_visible !== undefined && datos.disponibilidad.motivo_visible !== null) {
    exigirCadena(datos.disponibilidad.motivo_visible, "disponibilidad.motivo_visible", 500);
  }
  if (datos.posicion !== undefined && datos.posicion !== null) {
    const pos = exigirObjeto(datos.posicion, "posicion");
    exigirCadena(pos.bolsa, "posicion.bolsa", 200);
    exigirCadena(pos.categoria, "posicion.categoria", 200);
    exigirNumero(pos.orden, "posicion.orden", { minimo: 1, maximo: 1_000_000 });
    exigirNumero(pos.total, "posicion.total", { minimo: 0, maximo: 1_000_000 });
    exigirNumero(pos.puntuacion, "posicion.puntuacion", { minimo: 0, maximo: 10_000 });
    exigirFechaOInstante(pos.vigente_desde, "posicion.vigente_desde");
  }
  exigirLista(datos.ayuda, "ayuda").forEach((item, indice) => {
    exigirCadena(exigirObjeto(item, `ayuda[${indice}]`).pregunta, `ayuda[${indice}].pregunta`);
    exigirCadena(item.respuesta, `ayuda[${indice}].respuesta`, 2_000);
  });
  const capacidades = exigirObjeto(datos.capacidades, "capacidades");
  Object.entries(capacidades).forEach(([nombre, valor]) => exigirBooleano(valor, `capacidades.${nombre}`));
  if (datos.preferencias_notificacion !== undefined) {
    const preferencias = exigirObjeto(datos.preferencias_notificacion, "preferencias_notificacion");
    Object.entries(preferencias).forEach(([nombre, valor]) => exigirBooleano(valor, `preferencias_notificacion.${nombre}`));
  }
  if (datos.resultado_autobaremo !== undefined) {
    const resultado = exigirObjeto(datos.resultado_autobaremo, "resultado_autobaremo");
    exigirReferencia(resultado.convocatoria_id, "resultado_autobaremo.convocatoria_id", { demostracion: presentacion });
    exigirLista(resultado.meritos_ids, "resultado_autobaremo.meritos_ids")
      .forEach((id, indice) => exigirReferencia(id, `resultado_autobaremo.meritos_ids[${indice}]`, { demostracion: presentacion }));
    exigirNumero(resultado.puntos, "resultado_autobaremo.puntos", { maximo: 10_000 });
    exigirInstante(resultado.calculado_en, "resultado_autobaremo.calculado_en");
  }

  if (presentacion) validarPrivacidadPresentacion(datos);
  return congelarProfundo(datos);
}

export function validarRecibo(entrada, { presentacionEsperada = false } = {}) {
  const recibo = structuredClone(exigirObjeto(entrada, "recibo"));
  const esquemaEsperado = presentacionEsperada ? ESQUEMA_RECIBO_PRESENTACION : ESQUEMA_RECIBO;
  if (exigirCadena(recibo.esquema, "recibo.esquema", 80) !== esquemaEsperado) {
    throw new TypeError("El esquema del recibo no es compatible.");
  }
  if (exigirBooleano(recibo.presentacion, "recibo.presentacion") !== presentacionEsperada) {
    throw new TypeError("El recibo no corresponde al modo activo.");
  }
  exigirReferencia(recibo.referencia, "recibo.referencia", { demostracion: presentacionEsperada });
  exigirCadena(recibo.accion, "recibo.accion", 80);
  exigirReferencia(recibo.objetivo, "recibo.objetivo", { demostracion: presentacionEsperada });
  exigirCadena(recibo.resultado, "recibo.resultado", 80);
  exigirCadena(recibo.actor, "recibo.actor", 100);
  exigirInstante(recibo.fecha, "recibo.fecha");
  exigirCadena(recibo.advertencia, "recibo.advertencia", 300);
  if (recibo.estado_clave !== undefined && recibo.estado_clave !== null) {
    if (!SITUACIONES_PARTICIPACION_BOLSA.includes(recibo.estado_clave)) {
      throw new TypeError(`recibo.estado_clave no reconocido en el catálogo: ${recibo.estado_clave}`);
    }
  }
  if (recibo.disponible_desde !== undefined && recibo.disponible_desde !== null) {
    exigirFechaOInstante(recibo.disponible_desde, "recibo.disponible_desde");
  }
  return congelarProfundo(recibo);
}

export function validarPayloadCambiarDisponibilidad(payload) {
  const obj = exigirObjeto(payload, "payload de cambio de disponibilidad");
  const disponible = exigirBooleano(obj.disponible, "disponible");
  if (!disponible) {
    if (obj.motivo_clave !== undefined && obj.motivo_clave !== null) {
      if (!MOTIVOS_PAUSA_DISPONIBILIDAD.includes(obj.motivo_clave)) {
        throw new TypeError(`motivo_clave no reconocido en el catálogo: ${obj.motivo_clave}`);
      }
    }
    if (obj.motivo_texto !== undefined && obj.motivo_texto !== null) {
      if (typeof obj.motivo_texto !== "string" || obj.motivo_texto.length > 500) {
        throw new TypeError("motivo_texto debe ser una cadena de hasta 500 caracteres.");
      }
    }
    if (obj.hasta !== undefined && obj.hasta !== null && obj.hasta !== "") {
      if (!/^\d{4}-\d{2}-\d{2}$/.test(obj.hasta)) {
        throw new TypeError("hasta debe tener formato AAAA-MM-DD.");
      }
    }
  }
  return obj;
}

export function esModoPresentacion(parametros = new URLSearchParams()) {
  const selectores = parametros.getAll("presentacion");
  if (selectores.length > 1) throw new TypeError("El selector de presentación es ambiguo.");
  return selectores.length === 1 && selectores[0] === "rrhh";
}

export const CONTRATO_AREA_PERSONAL = Object.freeze({
  esquemaPanel: ESQUEMA_PANEL,
  esquemaRecibo: ESQUEMA_RECIBO,
  esquemaReciboPresentacion: ESQUEMA_RECIBO_PRESENTACION,
  situacionesParticipacion: SITUACIONES_PARTICIPACION_BOLSA,
  resultadosLlamamiento: RESULTADOS_LLAMAMIENTO,
  motivosPausaDisponibilidad: MOTIVOS_PAUSA_DISPONIBILIDAD,
});
