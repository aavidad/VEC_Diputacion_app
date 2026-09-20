/**
 * Estado de entrega visible para superficies que se presentan mientras una
 * capacidad se conecta progresivamente. No representa una autorización ni
 * habilita una acción: devuelve solo un fragmento HTML seguro para inyectar.
 */

import { crearTraductorEstadoEntrega } from "./estado-entrega-i18n.js";

const traducir = crearTraductorEstadoEntrega();

const ESTADOS = Object.freeze({
  conectado: Object.freeze({
    etiqueta: "estado_conectado_etiqueta",
    clase: "conectado",
    encabezado: "estado_conectado_encabezado",
  }),
  visual_pendiente_backend: Object.freeze({
    etiqueta: "estado_pendiente_etiqueta",
    clase: "pendiente",
    encabezado: "estado_pendiente_encabezado",
  }),
  bloqueado_dependencia: Object.freeze({
    etiqueta: "estado_bloqueado_etiqueta",
    clase: "bloqueado",
    encabezado: "estado_bloqueado_encabezado",
  }),
});

const CAMPOS = new Set(["estado", "resumen", "pendientes", "fuente", "conexion"]);
const CAMPOS_FUENTE = new Set(["etiqueta"]);

export function escaparTextoEstadoEntrega(valor) {
  return String(valor)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function esRegistro(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor)
    && Object.getPrototypeOf(valor) === Object.prototype;
}

function texto(campo, valor, { vacio = false } = {}) {
  if (typeof valor !== "string" || valor.length > 500 || (!vacio && valor.trim() === "")) {
    throw new TypeError(`estado de entrega inválido: ${campo}`);
  }
  return valor.trim();
}

function sinCamposDesconocidos(registro, permitidos, nombre) {
  for (const clave of Object.keys(registro)) {
    if (!permitidos.has(clave)) throw new TypeError(`estado de entrega inválido: ${nombre}.${clave}`);
  }
}

function validarFuente(valor) {
  if (!esRegistro(valor)) throw new TypeError("estado de entrega inválido: fuente");
  sinCamposDesconocidos(valor, CAMPOS_FUENTE, "fuente");
  return Object.freeze({ etiqueta: texto("fuente.etiqueta", valor.etiqueta) });
}

/** Valida y normaliza el contrato cerrado del estado visible. */
export function validarEstadoEntrega(entrada) {
  if (!esRegistro(entrada)) throw new TypeError("estado de entrega inválido");
  sinCamposDesconocidos(entrada, CAMPOS, "estado");
  if (!Object.hasOwn(ESTADOS, entrada.estado)) {
    throw new TypeError("estado de entrega inválido: estado");
  }
  if (!Array.isArray(entrada.pendientes) || entrada.pendientes.length > 8) {
    throw new TypeError("estado de entrega inválido: pendientes");
  }
  if (entrada.estado !== "conectado" && entrada.pendientes.length === 0) {
    throw new TypeError("estado de entrega inválido: pendientes vacíos");
  }
  const pendientes = entrada.pendientes.map((item, indice) => texto(`pendientes.${indice}`, item));
  const fuente = entrada.fuente === undefined ? undefined : validarFuente(entrada.fuente);
  const conexion = entrada.conexion === undefined ? undefined : texto("conexion", entrada.conexion);
  // No se admiten enlaces: una fuente es una referencia descriptiva, no una URL
  // que pueda inducir navegación, fuga de datos o una promesa externa.
  return Object.freeze({
    estado: entrada.estado,
    resumen: texto("resumen", entrada.resumen),
    pendientes: Object.freeze(pendientes),
    ...(fuente ? { fuente } : {}),
    ...(conexion ? { conexion } : {}),
  });
}

/** Renderiza un fragmento HTML estático y accesible, sin controles ni enlaces. */
export function renderizarEstadoEntrega(entrada) {
  const estado = validarEstadoEntrega(entrada);
  const meta = ESTADOS[estado.estado];
  const etiquetaEstado = traducir(meta.etiqueta);
  const encabezadoEstado = traducir(meta.encabezado);
  const pendientes = estado.pendientes.length === 0
    ? `<li>${traducir("pendientes_vacios")}</li>`
    : estado.pendientes.map((item) => `<li>${escaparTextoEstadoEntrega(item)}</li>`).join("");
  const fuente = estado.fuente
    ? `<p class="estado-entrega__origen"><strong>${traducir("fuente_etiqueta")}</strong> ${escaparTextoEstadoEntrega(estado.fuente.etiqueta)}</p>`
    : "";
  const conexion = estado.conexion
    ? `<p class="estado-entrega__origen"><strong>${traducir("conexion_etiqueta")}</strong> ${escaparTextoEstadoEntrega(estado.conexion)}</p>`
    : "";
  return `<section class="estado-entrega estado-entrega--${meta.clase}" aria-label="${escaparTextoEstadoEntrega(traducir("aria_estado", { estado: etiquetaEstado }))}">
  <div class="estado-entrega__cabecera">
    <span class="estado-entrega__chip">${etiquetaEstado}</span>
    <h3>${encabezadoEstado}</h3>
  </div>
  <p class="estado-entrega__resumen">${escaparTextoEstadoEntrega(estado.resumen)}</p>
  <h4>${traducir("pendientes_titulo")}</h4>
  <ul class="estado-entrega__pendientes">${pendientes}</ul>
  ${fuente}${conexion}
</section>`;
}

export const ESTADOS_ENTREGA = Object.freeze(Object.keys(ESTADOS));
