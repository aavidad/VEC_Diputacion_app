/** Textos castellanos de los apartados de la ficha propia servidos por Personal. */
const MENSAJES_FICHA_PROPIA_ES = Object.freeze({
  fuente_registro: "Registro de Personal",
  estado_relacion_vigente: "Vigente",
  estado_relacion_suspendida: "Suspendida",
  estado_relacion_finalizada: "Finalizada",
  relacion_abierta: "Actualidad",
  estado_servicio_declarado: "Declarado",
  estado_servicio_comprobado: "Comprobado",
  estado_servicio_reconocido: "Reconocido",
  dias_uno: "{total} día",
  dias_otro: "{total} días",
});

const CLAVES = Object.freeze(Object.keys(MENSAJES_FICHA_PROPIA_ES));

export function crearTraductorFichaPropia(catalogo = MENSAJES_FICHA_PROPIA_ES) {
  if (!catalogo || typeof catalogo !== "object"
    || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n de la ficha propia incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n de la ficha propia desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}

/** Días reconocidos con el plural y el formato numérico del castellano. */
export function formatearDiasFichaPropia(total, t = crearTraductorFichaPropia(), locale = "es-ES") {
  if (!Number.isSafeInteger(total) || total < 0) throw new TypeError("días reconocidos no válidos");
  return t(total === 1 ? "dias_uno" : "dias_otro", { total: new Intl.NumberFormat(locale).format(total) });
}
