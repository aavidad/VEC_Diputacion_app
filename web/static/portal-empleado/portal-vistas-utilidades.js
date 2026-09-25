import { formatearFechaPortal, traducirBolsaInterna } from "./portal-i18n.js?v=20260925-e10-v1";

/**
 * Componentes HTML puros compartidos por las vistas de consulta.
 * No acceden al DOM, a red ni a almacenamiento del navegador.
 */

export function crearUtilidadesVista({ escaparHTML, numero, claseEstado, encabezadoVista }) {
  if ([escaparHTML, numero, claseEstado, encabezadoVista]
    .some((dependencia) => typeof dependencia !== "function")) {
    throw new TypeError("dependencias de vista no válidas");
  }

  function chip(estado) {
    return `<span class="estado-chip ${claseEstado(estado)}">${escaparHTML(estado)}</span>`;
  }

  function tabla({
    titulo,
    cabeceras,
    filas,
    vacio = traducirBolsaInterna("tabla_sin_registros"),
    clavesColumnas = [],
    prioridadColumnas = "",
  }) {
    if (clavesColumnas.length > 0 && clavesColumnas.length !== cabeceras.length) {
      throw new TypeError("las claves de columna deben corresponder con todas las cabeceras");
    }
    if (prioridadColumnas && !["estado", "estado-acciones"].includes(prioridadColumnas)) {
      throw new TypeError("prioridad de columnas no válida");
    }
    const filaInvalida = filas.findIndex((fila) => !Array.isArray(fila) || fila.length !== cabeceras.length);
    if (filaInvalida >= 0) {
      throw new TypeError(`la fila ${filaInvalida + 1} debe contener una celda por cada cabecera`);
    }
    const atributoColumna = (indice) => clavesColumnas[indice]
      ? ` data-columna="${escaparHTML(clavesColumnas[indice])}"`
      : "";
    const cuerpo = filas.length > 0
      ? filas.map((fila) => `<tr>${fila.map((celda, indice) => `<td${atributoColumna(indice)}>${celda}</td>`).join("")}</tr>`).join("")
      : `<tr><td colspan="${cabeceras.length}" class="vacio-controlado">${escaparHTML(vacio)}</td></tr>`;
    const clasePrioridad = prioridadColumnas ? ` tabla-contenedor--prioritaria tabla-contenedor--${prioridadColumnas}` : "";
    const atributosRegion = prioridadColumnas
      ? ` tabindex="0" role="region" aria-label="${escaparHTML(traducirBolsaInterna("tabla_region_operativa", { titulo }))}" data-tabla-prioritaria="${prioridadColumnas}"`
      : "";
    const claseTabla = prioridadColumnas ? ` tabla-datos--prioritaria tabla-datos--${prioridadColumnas}` : "";
    return `<div class="tabla-contenedor${clasePrioridad}"${atributosRegion}><table class="tabla-datos${claseTabla}"><caption>${escaparHTML(titulo)}</caption><thead><tr>${cabeceras.map((cabecera, indice) => `<th scope="col"${atributoColumna(indice)}>${escaparHTML(cabecera)}</th>`).join("")}</tr></thead><tbody>${cuerpo}</tbody></table></div>`;
  }

  function kpi(sigla, valor, etiqueta) {
    return `<article class="tarjeta-kpi"><span class="icono-kpi" aria-hidden="true">${escaparHTML(sigla)}</span><div><strong class="valor-kpi">${escaparHTML(valor)}</strong><span class="etiqueta-kpi">${escaparHTML(etiqueta)}</span></div></article>`;
  }

  function campo(label, control, ayuda = "") {
    const controlSeguro = String(control).replace(/<(input|select|textarea)\b/, '<$1 disabled aria-disabled="true"');
    return `<label class="campo"><span>${escaparHTML(label)}</span>${controlSeguro}${ayuda ? `<small>${escaparHTML(ayuda)}</small>` : ""}</label>`;
  }

  return Object.freeze({
    campo,
    chip,
    encabezadoVista,
    escaparHTML,
    kpi,
    numero,
    fecha: formatearFechaPortal,
    tabla,
  });
}
