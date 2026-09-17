/**
 * Atajos del panel de incidencia del expediente. Delegado en el documento para
 * no crecer la vista principal: solo abre y enfoca el historial de actuaciones
 * del mismo contenido; no ejecuta ninguna acción administrativa.
 */
export function abrirHistorialDesde(boton) {
  const contenido = boton.closest(".ct-exp-contenido") ?? boton.ownerDocument;
  const historial = contenido.querySelector(".ct-exp-historial");
  if (!historial) return false;
  historial.open = true;
  historial.scrollIntoView({ block: "start" });
  historial.querySelector("summary")?.focus();
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
