/** Frontera estable entre la superficie Cronos y cualquier adaptador. */

import { exigirContextoParaModulo } from "../../identidad/contexto-actor.js";

export const CAPACIDAD_CONSULTAR_FICHAJES = "cronos.fichaje.read";
export const CAPACIDAD_REGISTRAR_FICHAJE = "cronos.fichaje.manage";
export const CAPACIDAD_CONSULTAR_HORARIO = "cronos.horario.read";
export const CAPACIDAD_CONSULTAR_PERMISOS = "cronos.permiso.read";
export const CAPACIDAD_SOLICITAR_PERMISO = "cronos.permiso.manage";

const CAPACIDADES_RECONOCIDAS = new Set([
  CAPACIDAD_CONSULTAR_FICHAJES,
  CAPACIDAD_REGISTRAR_FICHAJE,
  CAPACIDAD_CONSULTAR_HORARIO,
  CAPACIDAD_CONSULTAR_PERMISOS,
  CAPACIDAD_SOLICITAR_PERMISO,
]);

const CLAVES_DATOS = Object.freeze([
  "esquema", "demostracion", "actor_ref", "periodo", "actualizado_en",
  "perfil_jornada", "resumen", "fichajes", "saldos", "solicitudes", "historial",
]);
const ESTADOS_REGISTRO = new Set([
  "registrado", "revisado", "simulado", "saldo_demo", "pendiente_responsable",
  "aprobado", "disfrutado", "calculado", "preparado_no_registrado", "sin_registrar",
]);
const TIPOS_FICHAJE = new Set(["entrada", "salida", "inicio_pausa", "fin_pausa"]);
const UNIDADES_PERMISO = new Set(["dia", "minuto"]);
const CAMPOS_FICHAJE = Object.freeze([
  "id", "actor_ref", "instante", "tipo_clave", "canal", "modalidad", "estado_clave", "recibo_ref",
]);
const CAMPOS_SALDO = Object.freeze([
  "id", "nombre", "unidad_clave", "concedido", "solicitado", "aprobado", "disfrutado", "restante", "estado_clave",
]);
const CAMPOS_SOLICITUD = Object.freeze([
  "id", "actor_ref", "tipo", "desde", "hasta", "cantidad_valor", "unidad_clave", "estado_clave", "recibo_ref",
]);
const CAMPOS_HISTORIAL = Object.freeze([
  "id", "actor_ref", "instante", "evento", "detalle", "estado_clave", "recibo_ref",
]);

function esObjeto(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor);
}

function cadena(valor, nombre, maximo = 180) {
  if (typeof valor !== "string" || valor.trim() === "" || valor.length > maximo) {
    throw new Error(`${nombre} no válido`);
  }
  return valor;
}

function referenciaOpaca(valor, nombre) {
  const ref = cadena(valor, nombre, 128);
  if (!/^[A-Za-z0-9][A-Za-z0-9:._-]{5,127}$/.test(ref)) {
    throw new Error(`${nombre} no válida`);
  }
  return ref;
}

function instanteUTC(valor, nombre) {
  const instante = cadena(valor, nombre, 40);
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{3})?Z$/.test(instante)
    || !Number.isFinite(Date.parse(instante))) {
    throw new Error(`${nombre} no válido`);
  }
  return instante;
}

function listaAcotada(valor, nombre, maximo) {
  if (!Array.isArray(valor) || valor.length > maximo) throw new Error(`${nombre} no válida`);
  return valor.map((elemento) => {
    if (!esObjeto(elemento)) throw new Error(`${nombre} contiene un elemento no válido`);
    return { ...elemento };
  });
}

function camposExactos(registro, campos, nombre) {
  const esperados = new Set(campos);
  if (Object.keys(registro).length !== campos.length
    || Object.keys(registro).some((campo) => !esperados.has(campo))
    || campos.some((campo) => !Object.hasOwn(registro, campo))) {
    throw new Error(`${nombre} no respeta el contrato cerrado`);
  }
}

function enteroNoNegativo(valor, nombre) {
  if (!Number.isSafeInteger(valor) || valor < 0) throw new Error(`${nombre} no válido`);
  return valor;
}

function fechaCalendario(valor, nombre) {
  const fecha = cadena(valor, nombre, 10);
  if (!/^\d{4}-\d{2}-\d{2}$/.test(fecha)
    || new Date(`${fecha}T00:00:00Z`).toISOString().slice(0, 10) !== fecha) {
    throw new Error(`${nombre} no válida`);
  }
  return fecha;
}

function exigirReferenciasUnicas(lista, campo, nombre) {
  const vistas = new Set();
  for (const registro of lista) {
    const referencia = referenciaOpaca(registro[campo], `${nombre}.${campo}`);
    if (vistas.has(referencia)) throw new Error(`${nombre} contiene ${campo} repetida`);
    vistas.add(referencia);
  }
}

export function exigirContextoActorCronos(contextoActor) {
  return exigirContextoParaModulo(contextoActor, "cronos");
}

export function validarCapacidadesCronos(capacidades) {
  if (!Array.isArray(capacidades) || capacidades.length > CAPACIDADES_RECONOCIDAS.size) {
    throw new Error("capacidades de Cronos no válidas");
  }
  const normalizadas = [...new Set(capacidades.map((valor) => cadena(valor, "capacidad", 80)))];
  if (normalizadas.some((capacidad) => !CAPACIDADES_RECONOCIDAS.has(capacidad))) {
    throw new Error("capacidad de Cronos desconocida");
  }
  return Object.freeze(normalizadas.sort());
}

export function tieneCapacidadCronos(capacidades, capacidad) {
  return validarCapacidadesCronos(capacidades).includes(capacidad);
}

export function validarDatosCronos(datos, contextoActor) {
  const contexto = exigirContextoActorCronos(contextoActor);
  if (!esObjeto(datos) || datos.esquema !== "vec.cronos.area-personal.v1") {
    throw new Error("datos del área personal de Cronos no compatibles");
  }
  if (Object.keys(datos).some((clave) => !CLAVES_DATOS.includes(clave))
    || CLAVES_DATOS.some((clave) => !Object.hasOwn(datos, clave))) {
    throw new Error("datos de Cronos no respetan el contrato cerrado");
  }
  if (typeof datos.demostracion !== "boolean") throw new Error("origen de datos de Cronos no válido");
  if (datos.demostracion !== contexto.demostracion) {
    throw new Error("el origen de los datos no coincide con el contexto compartido");
  }
  if (referenciaOpaca(datos.actor_ref, "referencia de sujeto") !== contexto.actor.actor_ref) {
    throw new Error("los datos de Cronos no pertenecen al actor de la sesión");
  }
  if (!esObjeto(datos.perfil_jornada) || !esObjeto(datos.resumen)) {
    throw new Error("resumen de Cronos incompleto");
  }
  const validado = {
    esquema: datos.esquema,
    demostracion: datos.demostracion,
    actor_ref: contexto.actor.actor_ref,
    periodo: cadena(datos.periodo, "periodo", 80),
    actualizado_en: instanteUTC(datos.actualizado_en, "fecha de actualización"),
    perfil_jornada: { ...datos.perfil_jornada },
    resumen: { ...datos.resumen },
    fichajes: listaAcotada(datos.fichajes, "fichajes", 62),
    saldos: listaAcotada(datos.saldos, "saldos", 30),
    solicitudes: listaAcotada(datos.solicitudes, "solicitudes", 50),
    historial: listaAcotada(datos.historial, "historial", 80),
  };
  validado.fichajes.forEach((registro) => camposExactos(registro, CAMPOS_FICHAJE, "fichaje"));
  validado.saldos.forEach((registro) => camposExactos(registro, CAMPOS_SALDO, "saldo"));
  validado.solicitudes.forEach((registro) => camposExactos(registro, CAMPOS_SOLICITUD, "solicitud"));
  validado.historial.forEach((registro) => camposExactos(registro, CAMPOS_HISTORIAL, "evento de historial"));
  for (const [nombre, lista] of Object.entries({
    saldos: validado.saldos,
    fichajes: validado.fichajes,
    solicitudes: validado.solicitudes,
    historial: validado.historial,
  })) {
    if (nombre !== "saldos" && lista.some((registro) => registro.actor_ref !== contexto.actor.actor_ref)) {
      throw new Error(`${nombre} contiene registros ajenos al actor de la sesión`);
    }
    if (lista.some((registro) => !ESTADOS_REGISTRO.has(registro.estado_clave))) {
      throw new Error(`${nombre} contiene un estado no reconocido`);
    }
  }
  if (validado.fichajes.some((registro) => !TIPOS_FICHAJE.has(registro.tipo_clave))) {
    throw new Error("fichajes contiene un movimiento no reconocido");
  }
  for (const saldo of validado.saldos) {
    if (!UNIDADES_PERMISO.has(saldo.unidad_clave)) throw new Error("saldo contiene una unidad no reconocida");
    for (const campo of ["concedido", "solicitado", "aprobado", "disfrutado", "restante"]) {
      enteroNoNegativo(saldo[campo], `saldo.${campo}`);
    }
    if (saldo.restante !== Math.max(0, saldo.concedido - saldo.solicitado - saldo.aprobado - saldo.disfrutado)) {
      throw new Error("saldo de permiso incoherente");
    }
  }
  for (const solicitud of validado.solicitudes) {
    if (!UNIDADES_PERMISO.has(solicitud.unidad_clave)) throw new Error("solicitud contiene una unidad no reconocida");
    fechaCalendario(solicitud.desde, "solicitud.desde");
    fechaCalendario(solicitud.hasta, "solicitud.hasta");
    if (solicitud.hasta < solicitud.desde || enteroNoNegativo(solicitud.cantidad_valor, "solicitud.cantidad_valor") === 0) {
      throw new Error("solicitud contiene un periodo o cantidad no válido");
    }
  }
  if (validado.fichajes.some((registro) => {
    instanteUTC(registro.instante, "instante de fichaje");
    return Object.hasOwn(registro, "fecha") || Object.hasOwn(registro, "hora");
  })) {
    throw new Error("un fichaje debe usar un instante UTC canónico");
  }
  if (validado.historial.some((registro) => {
    instanteUTC(registro.instante, "instante de historial");
    return Object.hasOwn(registro, "fecha");
  })) {
    throw new Error("el historial debe usar un instante UTC canónico");
  }
  for (const [nombre, lista] of Object.entries({
    fichajes: validado.fichajes,
    saldos: validado.saldos,
    solicitudes: validado.solicitudes,
    historial: validado.historial,
  })) exigirReferenciasUnicas(lista, "id", nombre);
  exigirReferenciasUnicas(validado.fichajes, "recibo_ref", "fichajes");
  exigirReferenciasUnicas(validado.solicitudes, "recibo_ref", "solicitudes");
  exigirReferenciasUnicas(validado.historial, "recibo_ref", "historial");
  return validado;
}
