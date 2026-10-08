/** Lógica de presentación de la carga, sin DOM: filtros, páginas y mensajes. */
export const TAMANO_PAGINA = 50;
export const FILTROS = Object.freeze(["todas", "aceptadas", "errores", "avisos"]);

export function filtrarFilas(filas, filtro) {
  if (filtro === "aceptadas") return filas.filter((f) => f.estado === "aceptada");
  if (filtro === "errores") return filas.filter((f) => f.estado === "rechazada");
  if (filtro === "avisos") return filas.filter((f) => f.avisos.length > 0);
  return filas;
}

export function paginar(lista, pagina, tamano = TAMANO_PAGINA) {
  const total = Math.max(1, Math.ceil(lista.length / tamano));
  const actual = Math.min(Math.max(1, pagina), total);
  return { pagina: actual, total, elementos: lista.slice((actual - 1) * tamano, actual * tamano) };
}

/** «Primer apellido Segundo apellido, Nombre», como en las listas de RRHH. */
export function nombrePersona(fila) {
  const apellidos = [fila.primer_apellido, fila.segundo_apellido].filter(Boolean).join(" ");
  return [apellidos, fila.nombre].filter(Boolean).join(", ");
}

/** Clave estable del catálogo para el campo que llega del acta («Primer Apellido»). */
export function claveCampo(campo) {
  return String(campo).normalize("NFD").replace(/\p{M}/gu, "").toLowerCase().replace(/[^a-z0-9]+/gu, "_").replace(/^_|_$/gu, "");
}

function existe(textos, seccion, clave) {
  return Object.hasOwn(textos.mensajes?.[seccion] ?? {}, clave);
}

function entrada(textos, seccion, clave, variables = {}) {
  return textos.traducir(`${seccion}.${existe(textos, seccion, clave) ? clave : "otro"}`, variables);
}

export function textoIncidencia(textos, incidencia) {
  const campo = entrada(textos, "campos", claveCampo(incidencia.campo));
  return entrada(textos, "codigos", incidencia.codigo, { campo });
}

export function textoAviso(textos, aviso) {
  return entrada(textos, "avisos", aviso);
}

export function textoBloqueo(textos, bloqueo) {
  return entrada(textos, "bloqueos", bloqueo);
}

/** Mensaje en llano para un fallo; nunca códigos HTTP ni técnicos. */
export function textoError(textos, error) {
  const codigo = error?.codigo;
  if (typeof codigo === "string" && existe(textos, "errores", codigo)) return textos.traducir(`errores.${codigo}`);
  if (error?.estado === 401) return textos.traducir("errores.autenticacion_requerida");
  if (error?.estado === 403) return textos.traducir("errores.acceso_denegado");
  return textos.traducir("errores.servicio_no_disponible");
}

/** Clave «categoria:rpt:<clave>» → clave que pide el servidor. */
export function claveCategoria(referencia) {
  const prefijo = "categoria:rpt:";
  return typeof referencia === "string" && referencia.startsWith(prefijo) ? referencia.slice(prefijo.length) : "";
}
