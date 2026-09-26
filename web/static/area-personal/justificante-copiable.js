/**
 * Referencia opaca copiable del área personal (equivalente a la pieza del
 * portal interno, que esta superficie no sirve). La referencia no se muestra:
 * viaja en el botón y solo se copia al portapapeles.
 */
const DOCUMENTOS = new WeakSet();

export function botonCopiarReferencia(referencia, { escapar, copiar, copiado, descripcion = "" }) {
  if (typeof referencia !== "string" || referencia.trim() === "") return "";
  const aria = descripcion ? ` aria-label="${escapar(descripcion)}"` : "";
  return `<button type="button" class="boton-secundario boton-copiar-justificante" data-copiar-referencia="${escapar(referencia)}"`
    + ` data-texto-copiado="${escapar(copiado)}"${aria}>${escapar(copiar)}</button>`;
}

/** Un único oyente por documento; sin portapapeles el botón queda desactivado. */
export function instalarCopiaReferencias(documento = globalThis.document, portapapeles = globalThis.navigator?.clipboard) {
  if (!documento || typeof documento.addEventListener !== "function" || DOCUMENTOS.has(documento)) return false;
  DOCUMENTOS.add(documento);
  documento.addEventListener("click", (evento) => {
    const boton = evento.target?.closest?.("[data-copiar-referencia]");
    if (!boton) return;
    if (typeof portapapeles?.writeText !== "function") { boton.disabled = true; return; }
    portapapeles.writeText(boton.dataset.copiarReferencia).then(
      () => { boton.textContent = boton.dataset.textoCopiado || boton.textContent; },
      () => { boton.disabled = true; },
    );
  });
  return true;
}
