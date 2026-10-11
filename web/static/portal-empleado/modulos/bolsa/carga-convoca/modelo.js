/** Lógica de presentación de la carga, sin DOM. La página la decide el servidor. */
export const TAMANO_PAGINA = 50;
export const FILTROS_SERVIDOR = Object.freeze(["todas", "aceptadas", "rechazadas", "con_avisos"]);
const FILTROS_VISTA = Object.freeze({ todas: "todas", aceptadas: "aceptadas", errores: "rechazadas", avisos: "con_avisos" });
const FILTROS_CONTROL = Object.freeze({ todas: "todas", aceptadas: "aceptadas", rechazadas: "errores", con_avisos: "avisos" });
const PARAMETRO_FILTRO = "convoca_estado";
const PARAMETRO_PAGINA = "convoca_pagina";
const MAXIMO_PAGINA = 400;

export function filtroServidor(control) {
  return FILTROS_VISTA[control] ?? "todas";
}

export function filtroControl(filtro) {
  return FILTROS_CONTROL[filtro] ?? "todas";
}

export function leerEstadoRuta(search) {
  const parametros = new URLSearchParams(search);
  const estado = parametros.getAll(PARAMETRO_FILTRO);
  const paginas = parametros.getAll(PARAMETRO_PAGINA);
  const filtro = estado.length === 1 && FILTROS_SERVIDOR.includes(estado[0]) ? estado[0] : "todas";
  const numero = paginas.length === 1 && /^[1-9][0-9]{0,2}$/u.test(paginas[0]) ? Number(paginas[0]) : 1;
  return Object.freeze({ filtro, pagina: numero <= MAXIMO_PAGINA ? numero : 1 });
}

export function escribirEstadoRuta(search, filtro, pagina) {
  const parametros = new URLSearchParams(search);
  parametros.delete(PARAMETRO_FILTRO);
  parametros.delete(PARAMETRO_PAGINA);
  if (filtro !== "todas") parametros.set(PARAMETRO_FILTRO, filtro);
  if (pagina !== 1) parametros.set(PARAMETRO_PAGINA, String(pagina));
  const texto = parametros.toString();
  return texto ? `?${texto}` : "";
}

export function paginaServidor(vista) {
  const total = Math.max(1, Math.ceil(vista.total_filtrado / vista.limite));
  return Object.freeze({ pagina: Math.floor(vista.desplazamiento / vista.limite) + 1, total });
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
