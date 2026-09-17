/**
 * Pantalla de cada fase del expediente. Al pulsar una fase del raíl se muestra
 * un panel con los datos de la cabecera que pertenecen a esa fase y sus
 * actuaciones del historial; «Mostrar todo» vuelve a la vista completa. Trabaja
 * sobre el HTML ya renderizado y no ejecuta ninguna acción administrativa.
 */
const TEXTO = Object.freeze({
  fase: "Fase",
  datos: "Datos de la fase",
  sinDatos: "Esta fase todavía no tiene datos registrados en el expediente.",
  actuaciones: "Actuaciones de la fase",
  sinActuaciones: "Sin actuaciones registradas en esta fase.",
  mostrarTodo: "Mostrar todo el expediente",
  documentos: "Documentos",
});

function escapar(valor) {
  return String(valor ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "\"": "&quot;", "'": "&#39;" }[c]));
}

export function construirPanelFase(contenido, boton) {
  const clave = boton.dataset.ctExpFaseVer;
  const item = boton.closest("li");
  const etiqueta = item?.querySelector("span:not(.ct-exp-numero-fase)")?.textContent?.trim() ?? clave;
  const estado = item?.querySelector("small")?.textContent?.trim() ?? "";
  const orden = item?.querySelector(".ct-exp-numero-fase")?.textContent?.trim() ?? "";
  const campos = [...contenido.querySelectorAll(`[data-ct-exp-campo-fase="${clave}"]`)];
  const hitos = [...contenido.querySelectorAll("[data-ct-exp-hito-fase]")]
    .filter((fila) => fila.dataset.ctExpHitoFase === etiqueta);
  const panel = document.createElement("section");
  panel.className = `ct-exp-fase-panel ${item?.className ?? ""}`.trim();
  panel.setAttribute("aria-live", "polite");
  panel.innerHTML = `<header class="ct-exp-fase-panel-cabecera">
      <div><p class="sobrelinea">${escapar(TEXTO.fase)} ${escapar(orden)}</p><h3>${escapar(etiqueta)}</h3><p>${escapar(estado)}</p></div>
      <div class="ct-exp-fase-panel-acciones">
        <button type="button" class="boton-secundario" data-ct-exp-vista="documentos">${escapar(TEXTO.documentos)}</button>
        <button type="button" class="boton-secundario" data-ct-exp-fase-cerrar>${escapar(TEXTO.mostrarTodo)}</button>
      </div>
    </header>
    <h4>${escapar(TEXTO.datos)}</h4>
    ${campos.length === 0 ? `<p>${escapar(TEXTO.sinDatos)}</p>` : `<dl class="ct-exp-fase-datos">${campos.map((c) => c.outerHTML).join("")}</dl>`}
    <h4>${escapar(TEXTO.actuaciones)}</h4>
    ${hitos.length === 0 ? `<p>${escapar(TEXTO.sinActuaciones)}</p>` : `<div class="tabla-contenedor"><table class="tabla-datos"><tbody>${hitos.map((h) => h.outerHTML).join("")}</tbody></table></div>`}`;
  return panel;
}

export function mostrarFase(boton) {
  const contenido = boton.closest(".ct-exp-contenido") ?? boton.ownerDocument;
  const rail = boton.closest(".ct-exp-progreso");
  if (!rail) return false;
  contenido.querySelector(".ct-exp-fase-panel")?.remove();
  for (const otro of rail.querySelectorAll("[data-ct-exp-fase-ver]")) otro.setAttribute("aria-pressed", "false");
  boton.setAttribute("aria-pressed", "true");
  const panel = construirPanelFase(contenido, boton);
  rail.insertAdjacentElement("afterend", panel);
  panel.querySelector("h3")?.setAttribute("tabindex", "-1");
  panel.querySelector("h3")?.focus();
  return true;
}

export function cerrarFase(boton) {
  const contenido = boton.closest(".ct-exp-contenido") ?? boton.ownerDocument;
  contenido.querySelector(".ct-exp-fase-panel")?.remove();
  for (const otro of contenido.querySelectorAll("[data-ct-exp-fase-ver]")) otro.setAttribute("aria-pressed", "false");
  return true;
}

let instalado = false;
export function instalarPantallasFase(documento = globalThis.document) {
  if (instalado || !documento?.addEventListener) return;
  instalado = true;
  documento.addEventListener("click", (evento) => {
    const ver = evento.target?.closest?.("[data-ct-exp-fase-ver]");
    if (ver) { evento.preventDefault(); mostrarFase(ver); return; }
    const cerrar = evento.target?.closest?.("[data-ct-exp-fase-cerrar]");
    if (cerrar) { evento.preventDefault(); cerrarFase(cerrar); }
  });
}

instalarPantallasFase();
