import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js";

/** Extensión D1 del catálogo común de Dietas. */
export const MENSAJES_D1_DIETAS_ES = Object.freeze({
  d1_titulo: "Acceso por papel",
  d1_subtitulo: "Tareas previstas en el circuito de Dietas",
  d1_limite: "El acceso a cada tarea requiere la identidad común de VEC y una concesión V3 positiva. Sin una proyección autorizada, los papeles se muestran como no configurados.",
  d1_ayuda_etiqueta: "Ayuda sobre los papeles de Dietas",
  d1_ayuda: "Los papeles describen responsabilidades del procedimiento. Esta vista no cambia de identidad ni concede permisos. Una tarea solo puede marcarse disponible tras una comprobación autorizada para la persona y el ámbito concretos.",
  d1_estado_disponible: "Disponible",
  d1_estado_no_configurado: "No configurado",
  d1_empleado: "Empleado",
  d1_empleado_tarea: "Preparar una comisión de servicio y consultar los borradores propios.",
  d1_administrativo: "Administrativo del servicio",
  d1_administrativo_tarea: "Revisar documentos enviados y devolverlos con motivo cuando corresponda.",
  d1_responsable: "Responsable de centro",
  d1_responsable_tarea: "Autorizar la comisión o devolverla con un motivo.",
  d1_rrhh: "RRHH",
  d1_rrhh_tarea: "Revisar importes y liquidar la comisión según las reglas aprobadas.",
  d1_intervencion: "Intervención",
  d1_intervencion_tarea: "Fiscalizar la liquidación o devolverla con un motivo.",
});

export function crearTraductorD1Dietas(traducirBase = crearTraductorDietas(MENSAJES_DIETAS_ES)) {
  if (typeof traducirBase !== "function") throw new TypeError("traductor de Dietas no disponible");
  return (clave, variables = {}) => {
    if (!Object.hasOwn(MENSAJES_D1_DIETAS_ES, clave)) return traducirBase(clave, variables);
    try {
      const traducido = traducirBase(clave, variables);
      if (typeof traducido === "string" && traducido !== clave) return traducido;
    } catch { /* El catálogo común puede preceder a esta extensión. */ }
    return MENSAJES_D1_DIETAS_ES[clave].replace(/\{([a-z_]+)\}/gu, (_texto, variable) => String(variables[variable] ?? ""));
  };
}
