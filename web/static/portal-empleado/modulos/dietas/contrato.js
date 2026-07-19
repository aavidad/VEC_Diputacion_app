/** Contrato estable entre la superficie Dietas y cualquier adaptador. */

import { exigirContextoParaModulo } from "../../identidad/contexto-actor.js";

export const ESQUEMA_PANEL_DIETAS = "vec.dietas.portal.v1";
export const ESQUEMA_RECIBO_DIETAS = "vec.documentos.recibo.dietas.v1";
export const CAPACIDAD_CONSULTAR_GASTO = "dietas.gasto.read";
export const CAPACIDAD_GESTIONAR_GASTO = "dietas.gasto.manage";
export const CAPACIDAD_CONSULTAR_RUTA = "dietas.ruta.read";
export const CAPACIDAD_GESTIONAR_RUTA = "dietas.ruta.manage";
export const CAPACIDAD_GESTIONAR_APROBACION = "dietas.aprobacion.manage";
export const CAPACIDAD_CONSULTAR_AUDITORIA = "dietas.audit.read";

const CAPACIDADES_DIETAS = new Set([
  CAPACIDAD_CONSULTAR_GASTO,
  CAPACIDAD_GESTIONAR_GASTO,
  CAPACIDAD_CONSULTAR_RUTA,
  CAPACIDAD_GESTIONAR_RUTA,
  CAPACIDAD_GESTIONAR_APROBACION,
  CAPACIDAD_CONSULTAR_AUDITORIA,
]);
const ESTADOS_DIETAS = new Set([
  "borrador", "pendiente_jefatura", "aprobada", "enviada_rrhh", "enviada_nomina", "pagada",
]);
const ETAPAS_DIETAS = new Set(["borrador", "jefatura", "aprobada", "rrhh", "nomina", "pagada"]);
const SIGUIENTES_ACTUACIONES = new Set([
  "remision_rrhh", "completar_enviar_validacion", "expediente_finalizado", "inclusion_nomina", "revision_jefatura",
]);

export function copiarDietas(valor) {
  return structuredClone(valor);
}

function texto(valor, nombre, maximo = 200) {
  const resultado = String(valor ?? "").trim();
  if (!resultado || resultado.length > maximo || /[\u0000-\u0008\u000B\u000C\u000E-\u001F]/.test(resultado)) {
    throw new Error(`${nombre} no valido`);
  }
  return resultado;
}

export function exigirContextoActorDietas(contextoActor) {
  return exigirContextoParaModulo(contextoActor, "dietas");
}

export function validarCapacidadesDietas(capacidades = []) {
  if (!Array.isArray(capacidades) || capacidades.length > CAPACIDADES_DIETAS.size) {
    throw new Error("capacidades de Dietas no validas");
  }
  const unicas = new Set();
  for (const capacidad of capacidades) {
    if (!CAPACIDADES_DIETAS.has(capacidad) || unicas.has(capacidad)) {
      throw new Error("capacidad de Dietas no valida o repetida");
    }
    unicas.add(capacidad);
  }
  return Object.freeze([...unicas]);
}

export function tieneCapacidadDietas(capacidades, capacidad) {
  return Array.isArray(capacidades) && capacidades.includes(capacidad);
}

export function validarPanelDietas(datos, titularEsperado = "", capacidadesEsperadas = null) {
  if (!datos || typeof datos !== "object" || Array.isArray(datos)
    || datos.esquema !== ESQUEMA_PANEL_DIETAS || !datos.origen || typeof datos.origen !== "object"
    || !Array.isArray(datos.etapas) || !Array.isArray(datos.comisiones) || !datos.politica) {
    throw new Error("panel de Dietas no valido");
  }
  if (datos.origen.demostracion === true && datos.origen.efectos_reales !== false) {
    throw new Error("un panel demostrativo de Dietas no puede declarar efectos reales");
  }
  if (datos.borrador_inicial !== undefined
    && (!datos.borrador_inicial || typeof datos.borrador_inicial !== "object" || Array.isArray(datos.borrador_inicial))) {
    throw new Error("borrador inicial de Dietas no valido");
  }
  if (datos.etapas.length !== ETAPAS_DIETAS.size || datos.etapas.some((etapa) => !ETAPAS_DIETAS.has(etapa))) {
    throw new Error("etapas de Dietas no validas");
  }
  const capacidades = capacidadesEsperadas === null ? null : validarCapacidadesDietas(capacidadesEsperadas);
  const puedeConsultar = capacidades === null || tieneCapacidadDietas(capacidades, CAPACIDAD_CONSULTAR_GASTO);
  const puedeConsultarRutas = capacidades === null || tieneCapacidadDietas(capacidades, CAPACIDAD_CONSULTAR_RUTA);
  if (!puedeConsultar && datos.comisiones.length) {
    throw new Error("el panel contiene expedientes sin capacidad de consulta");
  }
  const referencias = new Set();
  for (const item of datos.comisiones) {
    const referencia = texto(item?.referencia, "referencia de comision", 100);
    if (referencias.has(referencia)) throw new Error("referencia de comision repetida");
    referencias.add(referencia);
    if (!Array.isArray(item.ruta) || !Array.isArray(item.historial)
      || !ESTADOS_DIETAS.has(item.estado) || !SIGUIENTES_ACTUACIONES.has(item.siguiente_actuacion)) {
      throw new Error("comision de Dietas incompleta");
    }
    if (titularEsperado && item.titular_ref !== titularEsperado) {
      throw new Error("el panel de Dietas contiene un expediente ajeno al contexto de actor");
    }
    if (!puedeConsultarRutas && (item.ruta.length || item.kilometros !== null || item.kilometraje_euros !== null)) {
      throw new Error("el panel contiene rutas sin capacidad de consulta");
    }
    if (item.historial.some((evento) => !ESTADOS_DIETAS.has(evento?.estado))) {
      throw new Error("historial de Dietas no valido");
    }
  }
  return copiarDietas(datos);
}

export function validarComandoDietas(comando) {
  if (!comando || typeof comando !== "object" || Array.isArray(comando)) throw new Error("comando de Dietas no valido");
  const tipo = texto(comando.tipo, "tipo de comando", 40);
  if (tipo === "crear_borrador") {
    if (!comando.campos || typeof comando.campos !== "object" || Array.isArray(comando.campos)) {
      throw new Error("campos de comision no validos");
    }
    return Object.freeze({ tipo, campos: Object.freeze({ ...comando.campos }) });
  }
  if (tipo === "enviar_validacion") {
    return Object.freeze({ tipo, referencia: texto(comando.referencia, "referencia", 100) });
  }
  throw new Error("comando de Dietas no permitido");
}
