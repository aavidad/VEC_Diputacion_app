/**
 * Componentes HTML puros compartidos por las vistas reales y de presentación.
 * No acceden al DOM, a red ni a almacenamiento del navegador.
 */

export function crearUtilidadesVista({ escaparHTML, numero, claseEstado, encabezadoVista, esPresentacion }) {
  if ([escaparHTML, numero, claseEstado, encabezadoVista, esPresentacion]
    .some((dependencia) => typeof dependencia !== "function")) {
    throw new TypeError("dependencias de vista no válidas");
  }

  function chip(estado) {
    return `<span class="estado-chip ${claseEstado(estado)}">${escaparHTML(estado)}</span>`;
  }

  function botonOperacion(etiqueta, operacion, objetivo, clase = "boton-secundario") {
    const bloqueado = esPresentacion() ? "" : ' disabled aria-disabled="true" title="Capacidad de servidor no conectada"';
    return `<button type="button" class="${clase}" data-accion="operacion-presentacion" data-operacion="${escaparHTML(operacion)}" data-objetivo="${escaparHTML(objetivo)}"${bloqueado}>${escaparHTML(etiqueta)}</button>`;
  }

  function botonBloqueado(etiqueta, motivo) {
    return `<button type="button" class="boton-secundario" data-accion="bloqueo-presentacion" data-motivo="${escaparHTML(motivo)}">${escaparHTML(etiqueta)}</button>`;
  }

  function tabla({ titulo, cabeceras, filas, vacio = "No hay registros para los filtros aplicados." }) {
    const cuerpo = filas.length > 0
      ? filas.map((fila) => `<tr>${fila.map((celda) => `<td>${celda}</td>`).join("")}</tr>`).join("")
      : `<tr><td colspan="${cabeceras.length}" class="vacio-controlado">${escaparHTML(vacio)}</td></tr>`;
    return `<div class="tabla-contenedor"><table class="tabla-datos"><caption>${escaparHTML(titulo)}</caption><thead><tr>${cabeceras.map((cabecera) => `<th scope="col">${escaparHTML(cabecera)}</th>`).join("")}</tr></thead><tbody>${cuerpo}</tbody></table></div>`;
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
    tabla,
  });
}
