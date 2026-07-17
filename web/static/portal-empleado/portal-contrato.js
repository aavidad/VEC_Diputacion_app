/**
 * Contratos puros del Portal de Bolsa.
 *
 * La consulta global nunca contiene personas candidatas. Una propuesta de
 * llamamiento solo puede llegar como resultado de su comando específico y se
 * reduce a evaluaciones sin identidad ni datos de contacto.
 */

const LISTAS_PANEL = Object.freeze([
  "bolsas", "necesidades_llamamiento", "elaboraciones", "proximos",
  "actividad", "contratos", "reglas", "documentos", "canales", "avisos",
]);

function esObjeto(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor);
}

export function extraerDatosEnvelopeCanonico(envelope) {
  if (!esObjeto(envelope) || !Object.hasOwn(envelope, "data") || !esObjeto(envelope.data)) {
    throw new Error("la API debe responder con el envelope canónico {data:{...}}");
  }
  if (Object.hasOwn(envelope, "esquema") || Object.hasOwn(envelope, "bolsas") || Object.hasOwn(envelope, "demostracion")) {
    throw new Error("se ha rechazado una respuesta raw en la raíz");
  }
  return envelope.data;
}

export function validarPanelBolsa(datos, admiteDemostracion = false) {
  if (!esObjeto(datos)) throw new Error("respuesta del panel no válida");
  const esquema = admiteDemostracion ? "vec.bolsa.panel.presentacion.v1" : "vec.bolsa.panel.v1";
  if (datos.esquema !== esquema) throw new Error("versión de contrato del panel no compatible");
  if (datos.demostracion === true && !admiteDemostracion) {
    throw new Error("la API interna no puede responder con datos de demostración");
  }
  if (Object.hasOwn(datos, "candidatos")) {
    throw new Error("el panel global no admite listados de personas candidatas");
  }
  if (LISTAS_PANEL.some((clave) => !Array.isArray(datos[clave]))) {
    throw new Error("respuesta del panel incompleta");
  }
  return {
    esquema: String(datos.esquema),
    demostracion: datos.demostracion === true,
    sesion: esObjeto(datos.sesion) ? { ...datos.sesion } : null,
    indicadores: esObjeto(datos.indicadores) ? { ...datos.indicadores } : {},
    distribucion_global: esObjeto(datos.distribucion_global) ? { ...datos.distribucion_global } : {},
    series: esObjeto(datos.series) ? { ...datos.series } : {},
    avisos: [...datos.avisos],
    capacidades: esObjeto(datos.capacidades) ? { ...datos.capacidades } : {},
    configuracion_llamamiento: esObjeto(datos.configuracion_llamamiento) ? { ...datos.configuracion_llamamiento } : {},
    catalogos_llamamiento: esObjeto(datos.catalogos_llamamiento) ? { ...datos.catalogos_llamamiento } : {},
    bolsas: [...datos.bolsas],
    necesidades_llamamiento: [...datos.necesidades_llamamiento],
    elaboraciones: [...datos.elaboraciones],
    proximos: [...datos.proximos],
    actividad: [...datos.actividad],
    contratos: [...datos.contratos],
    reglas: [...datos.reglas],
    documentos: [...datos.documentos],
    canales: [...datos.canales],
    auditoria: esObjeto(datos.auditoria) ? { ...datos.auditoria } : {},
  };
}

export function validarPropuestaLlamamiento(datos, admiteDemostracion = false) {
  if (!esObjeto(datos)) throw new Error("propuesta de llamamiento no válida");
  const esquema = admiteDemostracion
    ? "vec.bolsa.propuesta-llamamiento.presentacion.v1"
    : "vec.bolsa.propuesta-llamamiento.v1";
  if (datos.esquema !== esquema || !Array.isArray(datos.evaluaciones)) {
    throw new Error("contrato de propuesta de llamamiento no compatible");
  }
  if (datos.demostracion === true && !admiteDemostracion) {
    throw new Error("la API interna no puede responder con una propuesta de demostración");
  }
  const camposPropuesta = new Set(["esquema", "demostracion", "id", "necesidad_id", "estado", "version_bolsa", "version_regla", "fecha_corte", "personas_incluidas", "evaluaciones"]);
  const camposEvaluacion = new Set(["secuencia", "resultado", "puntuacion", "regla", "fundamento"]);
  if (Object.keys(datos).some((campo) => !camposPropuesta.has(campo))
    || datos.evaluaciones.some((item) => !esObjeto(item) || Object.keys(item).some((campo) => !camposEvaluacion.has(campo)))) {
    throw new Error("la propuesta visible no admite identidad ni contacto");
  }
  if (!datos.necesidad_id || !datos.id || !Number.isFinite(Number(datos.personas_incluidas))) {
    throw new Error("propuesta de llamamiento incompleta");
  }
  if (datos.evaluaciones.some((item) => !Number.isFinite(Number(item.secuencia)) || !Number.isFinite(Number(item.puntuacion)))) {
    throw new Error("evaluación de propuesta no válida");
  }
  return {
    esquema: String(datos.esquema),
    demostracion: datos.demostracion === true,
    id: String(datos.id || ""),
    necesidad_id: String(datos.necesidad_id || ""),
    estado: String(datos.estado || ""),
    version_bolsa: String(datos.version_bolsa || ""),
    version_regla: String(datos.version_regla || ""),
    fecha_corte: String(datos.fecha_corte || ""),
    personas_incluidas: Number(datos.personas_incluidas || 0),
    evaluaciones: datos.evaluaciones.map((item) => ({
      secuencia: Number(item.secuencia || 0),
      resultado: String(item.resultado || ""),
      puntuacion: Number(item.puntuacion || 0),
      regla: String(item.regla || ""),
      fundamento: String(item.fundamento || ""),
    })),
  };
}
