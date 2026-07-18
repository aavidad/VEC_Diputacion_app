export function escaparHTML(valor) {
  return String(valor ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

export function escaparAtributo(valor) {
  return escaparHTML(valor).replaceAll("`", "&#096;");
}

export function claseEstado(estado = "") {
  const valor = String(estado).toLowerCase();
  if (/rechaz|error|caduc|bloque|cerrad/.test(valor)) return "error";
  if (/acept|valid|registr|disponible|complet|publicad|firmad|abonad/.test(valor)) return "exito";
  if (/pendiente|requiere|plazo|borrador|próxim|revision|revisión/.test(valor)) return "aviso";
  if (/barem|mérito|provisional/.test(valor)) return "merito";
  return "info";
}

export function chip(estado) {
  return `<span class="estado-chip ${claseEstado(estado)}">${escaparHTML(estado)}</span>`;
}

export function encabezadoVista(titulo, descripcion, acciones = "") {
  return `<header class="encabezado-vista"><div><h2>${escaparHTML(titulo)}</h2><p>${escaparHTML(descripcion)}</p></div>${acciones ? `<div class="fila-acciones">${acciones}</div>` : ""}</header>`;
}

export function enlaceRuta(ruta, etiqueta, clase = "enlace-boton", extras = "") {
  return `<a class="${escaparAtributo(clase)}" href="?vista=${encodeURIComponent(ruta)}" data-ruta="${escaparAtributo(ruta)}" ${extras}>${escaparHTML(etiqueta)}</a>`;
}

export function botonOperacion(operacion, etiqueta, { id = "", clase = "boton-primario", descripcion = "" } = {}) {
  return `<button type="button" class="${escaparAtributo(clase)}" data-accion="preparar-operacion" data-operacion="${escaparAtributo(operacion)}" data-id="${escaparAtributo(id)}" data-descripcion="${escaparAtributo(descripcion || etiqueta)}">${escaparHTML(etiqueta)}</button>`;
}

export function panel(titulo, subtitulo, contenido, { estado = "", clase = "" } = {}) {
  return `<section class="panel ${escaparAtributo(clase)}"><header><div><h3>${escaparHTML(titulo)}</h3>${subtitulo ? `<p>${escaparHTML(subtitulo)}</p>` : ""}</div>${estado ? chip(estado) : ""}</header><div class="panel-contenido">${contenido}</div></section>`;
}

export function listaDatos(pares) {
  return `<dl class="dato-lista">${pares.map(([termino, descripcion]) => `<dt>${escaparHTML(termino)}</dt><dd>${descripcion}</dd>`).join("")}</dl>`;
}

export function tabla({ descripcion, columnas, filas, vacio = "No hay registros para mostrar." }) {
  if (!filas.length) return `<div class="estado-vacio"><strong>Sin resultados</strong>${escaparHTML(vacio)}</div>`;
  return `<div class="tabla-contenedor"><table class="tabla-administrativa"><caption>${escaparHTML(descripcion)}</caption><thead><tr>${columnas.map((columna) => `<th scope="col">${escaparHTML(columna)}</th>`).join("")}</tr></thead><tbody>${filas.map((fila) => `<tr>${fila.map((celda, indice) => `<td data-etiqueta="${escaparAtributo(columnas[indice])}">${celda}</td>`).join("")}</tr>`).join("")}</tbody></table></div>`;
}

export function barraProgreso(valor, maximo) {
  const porcentaje = Math.max(0, Math.min(100, maximo > 0 ? (Number(valor) / Number(maximo)) * 100 : 0));
  return `<progress class="barra-progreso" value="${porcentaje.toFixed(1)}" max="100" aria-label="${porcentaje.toFixed(1)} por ciento">${porcentaje.toFixed(1)} %</progress>`;
}

export function formatoPuntos(valor) {
  return new Intl.NumberFormat("es-ES", { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(Number(valor || 0));
}

export function estadoVacio(titulo, detalle, accion = "") {
  return `<div class="estado-vacio"><strong>${escaparHTML(titulo)}</strong><p>${escaparHTML(detalle)}</p>${accion}</div>`;
}

export function notaDemostracion() {
  return `<p class="nota demo"><strong>Recorrido de demostración.</strong> Los datos son sintéticos, las operaciones solo viven en memoria y cada confirmación genera un recibo DEMO sin validez administrativa.</p>`;
}
