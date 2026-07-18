/**
 * Contratos puros del Portal de Bolsa.
 *
 * La consulta global nunca contiene personas candidatas. Una propuesta de
 * llamamiento solo puede llegar como resultado de su comando específico y se
 * reduce a evaluaciones sin identidad ni datos de contacto.
 */

const LISTAS_PANEL_PRESENTACION = Object.freeze([
  "bolsas", "necesidades_llamamiento", "elaboraciones", "proximos",
  "actividad", "contratos", "reglas", "documentos", "canales", "avisos",
  "solicitudes", "meritos_revision", "criterios_baremo", "ranking",
  "alegaciones", "importaciones", "llamamientos_demo", "comunicaciones_demo",
  "auditoria_eventos", "roles_demo", "configuraciones_demo",
]);

const CAMPOS_PANEL_INTERNO = Object.freeze([
  "esquema", "selector", "origen", "prueba_lectura", "indicadores",
  "convocatorias", "actuaciones_pendientes",
]);
const CAMPOS_SELECTOR = Object.freeze(["clase", "organizacion_ref", "unidad_gestion_ref"]);
const CAMPOS_ORIGEN = Object.freeze(["revision", "actualizada_en", "demostracion"]);
const CAMPOS_PRUEBA_LECTURA = Object.freeze([
  "lectura_ref", "auditoria_ref", "auditoria_secuencia", "decision_ref",
  "huella_decision_sha256", "correlacion_ref", "confirmada_en",
]);
const CAMPOS_INDICADORES = Object.freeze([
  "convocatorias_borrador", "convocatorias_revision", "convocatorias_pendientes_firma",
  "convocatorias_publicadas", "bolsas_activas", "bolsas_suspendidas", "bolsas_agotadas",
  "llamamientos_pendientes", "llamamientos_en_curso", "llamamientos_vencen_hoy",
  "documentos_pendientes_firma", "incidencias_abiertas",
]);
const CAMPOS_CONVOCATORIA = Object.freeze([
  "convocatoria_ref", "categoria_clave", "estado_clave", "plazo_cierra_en",
  "numero_solicitudes", "numero_pendientes",
]);
const CAMPOS_ACTUACION = Object.freeze([
  "actuacion_ref", "recurso_ref", "tipo_clave", "estado_clave",
  "prioridad_clave", "fecha_limite", "numero_elementos",
]);

function esObjeto(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor);
}

function exigirCamposExactos(objeto, campos, nombre, opcionales = []) {
  if (!esObjeto(objeto)) throw new Error(`${nombre} no válido`);
  const permitidos = new Set(campos);
  const opcionalesSet = new Set(opcionales);
  if (Object.keys(objeto).some((campo) => !permitidos.has(campo))
    || campos.some((campo) => !opcionalesSet.has(campo) && !Object.hasOwn(objeto, campo))) {
    throw new Error(`${nombre} no respeta el contrato cerrado`);
  }
}

function exigirCadena(valor, nombre) {
  if (typeof valor !== "string" || valor.trim() === "") throw new Error(`${nombre} no válido`);
  return valor;
}

function exigirEnteroNoNegativo(valor, nombre, maximo = 1_000_000_000) {
  if (!Number.isSafeInteger(valor) || valor < 0 || valor > maximo) throw new Error(`${nombre} no válido`);
  return valor;
}

function exigirInstante(valor, nombre) {
  const patronUTC = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/;
  if (typeof valor !== "string" || !patronUTC.test(valor) || !Number.isFinite(Date.parse(valor))) {
    throw new Error(`${nombre} no válido`);
  }
  return valor;
}

function exigirReferenciaOpaca(valor, prefijo, nombre) {
  const referencia = exigirCadena(valor, nombre);
  if (!referencia.startsWith(prefijo) || !/^[a-z]{3}_[a-z0-9]{16,80}$/.test(referencia)) {
    throw new Error(`${nombre} no válida`);
  }
  return referencia;
}

function exigirClave(valor, nombre) {
  const clave = exigirCadena(valor, nombre);
  if (!/^[a-z][a-z0-9_.-]{1,79}$/.test(clave)) throw new Error(`${nombre} no válida`);
  return clave;
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
  if (!admiteDemostracion) return validarPanelInterno(datos);
  const esquema = "vec.bolsa.panel.presentacion.v1";
  if (datos.esquema !== esquema) throw new Error("versión de contrato del panel no compatible");
  if (Object.hasOwn(datos, "candidatos")) {
    throw new Error("el panel global no admite listados de personas candidatas");
  }
  if (LISTAS_PANEL_PRESENTACION.some((clave) => !Array.isArray(datos[clave]))) {
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
    solicitudes: [...datos.solicitudes],
    meritos_revision: [...datos.meritos_revision],
    criterios_baremo: [...datos.criterios_baremo],
    ranking: [...datos.ranking],
    alegaciones: [...datos.alegaciones],
    importaciones: [...datos.importaciones],
    llamamientos_demo: [...datos.llamamientos_demo],
    comunicaciones_demo: [...datos.comunicaciones_demo],
    auditoria_eventos: [...datos.auditoria_eventos],
    roles_demo: [...datos.roles_demo],
    configuraciones_demo: [...datos.configuraciones_demo],
    auditoria: esObjeto(datos.auditoria) ? { ...datos.auditoria } : {},
  };
}

function validarPanelInterno(datos) {
  if (datos.esquema !== "vec.bolsa.panel.interno.v1") {
    throw new Error("versión de contrato del panel no compatible");
  }
  if (Object.hasOwn(datos, "candidatos")) {
    throw new Error("el panel global no admite listados de personas candidatas");
  }
  exigirCamposExactos(datos, CAMPOS_PANEL_INTERNO, "panel interno");
  exigirCamposExactos(datos.selector, CAMPOS_SELECTOR, "selector", ["unidad_gestion_ref"]);
  exigirCamposExactos(datos.origen, CAMPOS_ORIGEN, "origen");
  exigirCamposExactos(datos.prueba_lectura, CAMPOS_PRUEBA_LECTURA, "prueba de lectura");
  exigirCamposExactos(datos.indicadores, CAMPOS_INDICADORES, "indicadores");

  const clase = exigirCadena(datos.selector.clase, "clase del selector");
  if (clase !== "organizacion" && clase !== "unidad_gestion") {
    throw new Error("clase del selector no válida");
  }
  exigirReferenciaOpaca(datos.selector.organizacion_ref, "org_", "referencia de organización");
  if (clase === "organizacion" && Object.hasOwn(datos.selector, "unidad_gestion_ref")) {
    throw new Error("un selector de organización no admite unidad de gestión");
  }
  if (clase === "unidad_gestion") {
    exigirReferenciaOpaca(datos.selector.unidad_gestion_ref, "uni_", "referencia de unidad");
  }

  if (datos.origen.demostracion !== false) {
    throw new Error("la API interna no puede responder con datos de demostración");
  }
  exigirReferenciaOpaca(datos.origen.revision, "rev_", "revisión de origen");
  exigirInstante(datos.origen.actualizada_en, "fecha de actualización");

  exigirReferenciaOpaca(datos.prueba_lectura.lectura_ref, "lec_", "referencia de lectura");
  exigirReferenciaOpaca(datos.prueba_lectura.auditoria_ref, "aud_", "referencia de auditoría");
  exigirEnteroNoNegativo(datos.prueba_lectura.auditoria_secuencia, "secuencia de auditoría", Number.MAX_SAFE_INTEGER);
  if (datos.prueba_lectura.auditoria_secuencia === 0) throw new Error("secuencia de auditoría no válida");
  exigirCadena(datos.prueba_lectura.decision_ref, "referencia de decisión");
  if (!/^[a-f0-9]{64}$/.test(exigirCadena(datos.prueba_lectura.huella_decision_sha256, "huella de decisión"))) {
    throw new Error("huella de decisión no válida");
  }
  exigirCadena(datos.prueba_lectura.correlacion_ref, "referencia de correlación");
  exigirInstante(datos.prueba_lectura.confirmada_en, "fecha de confirmación");

  for (const campo of CAMPOS_INDICADORES) {
    exigirEnteroNoNegativo(datos.indicadores[campo], `indicador ${campo}`);
  }
  if (!Array.isArray(datos.convocatorias) || datos.convocatorias.length > 40
    || !Array.isArray(datos.actuaciones_pendientes) || datos.actuaciones_pendientes.length > 80) {
    throw new Error("respuesta del panel incompleta");
  }

  const referenciasConvocatorias = new Set();
  const convocatorias = datos.convocatorias.map((item) => {
    exigirCamposExactos(item, CAMPOS_CONVOCATORIA, "convocatoria", ["plazo_cierra_en"]);
    exigirReferenciaOpaca(item.convocatoria_ref, "cnv_", "referencia de convocatoria");
    exigirClave(item.categoria_clave, "clave de categoría");
    exigirClave(item.estado_clave, "clave de estado");
    if (Object.hasOwn(item, "plazo_cierra_en")) exigirInstante(item.plazo_cierra_en, "cierre de plazo");
    exigirEnteroNoNegativo(item.numero_solicitudes, "número de solicitudes");
    exigirEnteroNoNegativo(item.numero_pendientes, "número de pendientes");
    if (item.numero_pendientes > item.numero_solicitudes) {
      throw new Error("una convocatoria no puede tener más pendientes que solicitudes");
    }
    if (referenciasConvocatorias.has(item.convocatoria_ref)) throw new Error("convocatoria repetida");
    referenciasConvocatorias.add(item.convocatoria_ref);
    return { ...item };
  });
  const referenciasActuaciones = new Set();
  const actuacionesPendientes = datos.actuaciones_pendientes.map((item) => {
    exigirCamposExactos(item, CAMPOS_ACTUACION, "actuación pendiente", ["fecha_limite"]);
    exigirReferenciaOpaca(item.actuacion_ref, "act_", "referencia de actuación");
    if (!/^[a-z]{3}_[a-z0-9]{16,80}$/.test(exigirCadena(item.recurso_ref, "referencia de recurso"))) {
      throw new Error("referencia de recurso no válida");
    }
    exigirClave(item.tipo_clave, "tipo de actuación");
    exigirClave(item.estado_clave, "estado de actuación");
    exigirClave(item.prioridad_clave, "prioridad de actuación");
    if (Object.hasOwn(item, "fecha_limite")) exigirInstante(item.fecha_limite, "fecha límite");
    exigirEnteroNoNegativo(item.numero_elementos, "número de elementos");
    if (referenciasActuaciones.has(item.actuacion_ref)) throw new Error("actuación repetida");
    referenciasActuaciones.add(item.actuacion_ref);
    return { ...item };
  });

  return {
    esquema: datos.esquema,
    selector: { ...datos.selector },
    origen: { ...datos.origen },
    prueba_lectura: { ...datos.prueba_lectura },
    indicadores: { ...datos.indicadores },
    convocatorias,
    actuaciones_pendientes: actuacionesPendientes,
  };
}
