/**
 * Atajos del panel de incidencia del expediente. Delegado en el documento para
 * no crecer la vista principal: solo abre y enfoca el historial de actuaciones
 * del mismo contenido; no ejecuta ninguna acción administrativa.
 */
export function abrirHistorialDesde(boton) {
  const contenido = boton.closest(".ct-exp-contenido") ?? boton.ownerDocument;
  const historial = contenido.querySelector(".ct-exp-historial");
  if (!historial) return false;
  // El historial legible está a la vista; la tabla de actuaciones, plegada.
  historial.open = true;
  const panel = historial.closest(".ct-exp-ficha-historial") ?? historial;
  panel.scrollIntoView({ block: "start" });
  (panel.querySelector("h3") ?? historial.querySelector("summary"))?.setAttribute?.("tabindex", "-1");
  (panel.querySelector("h3") ?? historial.querySelector("summary"))?.focus();
  return true;
}

let instalado = false;
export function instalarAtajosIncidencia(documento = globalThis.document) {
  if (instalado || !documento?.addEventListener) return;
  instalado = true;
  documento.addEventListener("click", (evento) => {
    const boton = evento.target?.closest?.('[data-ct-exp-accion="abrir-historial"]');
    if (boton && abrirHistorialDesde(boton)) evento.preventDefault();
  });
}

instalarAtajosIncidencia();
