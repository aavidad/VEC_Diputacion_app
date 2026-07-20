/**
 * Componentes HTML puros compartidos por las vistas reales y de presentación.
 * No acceden al DOM, a red ni a almacenamiento del navegador.
 */

export function crearUtilidadesVista({ escaparHTML, numero, claseEstado, encabezadoVista, esPresentacion, operacionPermitida }) {
  if ([escaparHTML, numero, claseEstado, encabezadoVista, esPresentacion, operacionPermitida]
    .some((dependencia) => typeof dependencia !== "function")) {
    throw new TypeError("dependencias de vista no válidas");
  }

  function chip(estado) {
    return `<span class="estado-chip ${claseEstado(estado)}">${escaparHTML(estado)}</span>`;
  }

  function botonOperacion(etiqueta, operacion, objetivo, clase = "boton-secundario") {
    const esDemo = esPresentacion();
    const permitido = esDemo && operacionPermitida(operacion);
    const titulo = esDemo
      ? "El perfil de presentación no permite esta operación"
      : "Capacidad de servidor no conectada";
    const bloqueado = permitido ? "" : ` disabled aria-disabled="true" title="${escaparHTML(titulo)}"`;
    return `<button type="button" class="${clase}" data-accion="operacion-presentacion" data-comando="${escaparHTML(operacion)}" data-operacion="${escaparHTML(operacion)}" data-objetivo="${escaparHTML(objetivo)}"${bloqueado}>${escaparHTML(etiqueta)}</button>`;
  }

  function botonBloqueado(etiqueta, motivo) {
    return `<button type="button" class="boton-secundario" data-accion="bloqueo-presentacion" data-motivo="${escaparHTML(motivo)}">${escaparHTML(etiqueta)}</button>`;
  }

  function tabla({
    titulo,
    cabeceras,
    filas,
    vacio = "No hay registros para los filtros aplicados.",
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
      ? ` tabindex="0" role="region" aria-label="Tabla operativa: ${escaparHTML(titulo)}" data-tabla-prioritaria="${prioridadColumnas}"`
      : "";
    const claseTabla = prioridadColumnas ? ` tabla-datos--prioritaria tabla-datos--${prioridadColumnas}` : "";
    return `<div class="tabla-contenedor${clasePrioridad}"${atributosRegion}><table class="tabla-datos${claseTabla}"><caption>${escaparHTML(titulo)}</caption><thead><tr>${cabeceras.map((cabecera, indice) => `<th scope="col"${atributoColumna(indice)}>${escaparHTML(cabecera)}</th>`).join("")}</tr></thead><tbody>${cuerpo}</tbody></table></div>`;
  }

  function kpi(sigla, valor, etiqueta) {
    return `<article class="tarjeta-kpi"><span class="icono-kpi" aria-hidden="true">${escaparHTML(sigla)}</span><div><strong class="valor-kpi">${escaparHTML(valor)}</strong><span class="etiqueta-kpi">${escaparHTML(etiqueta)}</span></div></article>`;
  }

  function fuentePresentacion() {
    return esPresentacion()
      ? '<span class="estado-chip info">Datos sintéticos · Memoria volátil</span>'
      : '<span class="estado-chip neutro">Fuente real no conectada</span>';
  }

  function avisoPresentacion(texto = "Este recorrido simula la actuación sin efectos administrativos ni comunicaciones externas.") {
    return esPresentacion()
      ? `<section class="nota-seguridad" aria-label="Alcance de la presentación"><strong>Modo presentación.</strong> ${escaparHTML(texto)}</section>`
      : '<section class="nota-pendiente" aria-label="Capacidad real no disponible"><strong>Funcionalidad no conectada.</strong> La misma pantalla queda visible, pero sus acciones permanecen deshabilitadas hasta que el servidor conceda capacidad explícita y aporte datos autorizados.</section>';
  }

  function campo(label, control, ayuda = "") {
    const controlSeguro = esPresentacion()
      ? control
      : String(control).replace(/<(input|select|textarea)\b/, '<$1 disabled aria-disabled="true"');
    return `<label class="campo"><span>${escaparHTML(label)}</span>${controlSeguro}${ayuda ? `<small>${escaparHTML(ayuda)}</small>` : ""}</label>`;
  }

  return Object.freeze({
    avisoPresentacion,
    botonBloqueado,
    botonOperacion,
    campo,
    chip,
    encabezadoVista,
    escaparHTML,
    fuentePresentacion,
    kpi,
    numero,
    operacionPermitida,
    tabla,
  });
}
