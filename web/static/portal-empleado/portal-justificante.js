/**
 * Justificante de una actuación registrada. La referencia del recibo es opaca:
 * no se muestra en pantalla, solo se ofrece copiarla para el seguimiento. Los
 * textos los aporta cada módulo desde su propio catálogo i18n.
 */

const DOCUMENTOS_CON_COPIA = new WeakSet();

export function renderizarJustificante(referencia, { escapar, etiqueta, copiar, copiado }) {
  if (typeof referencia !== "string" || referencia.trim() === "") return escapar("—");
  return `<span class="justificante-registrado">${escapar(etiqueta)}</span> `
    + `<button type="button" class="boton-terciario boton-copiar-justificante" data-copiar-justificante="${escapar(referencia)}"`
    + ` data-texto-copiado="${escapar(copiado)}">${escapar(copiar)}</button>`;
}

/** Un único oyente por documento copia la referencia del botón pulsado. */
export function instalarCopiaJustificantes(documento = globalThis.document, portapapeles = globalThis.navigator?.clipboard) {
  if (!documento || typeof documento.addEventListener !== "function" || DOCUMENTOS_CON_COPIA.has(documento)) return false;
  DOCUMENTOS_CON_COPIA.add(documento);
  documento.addEventListener("click", (evento) => {
    const boton = evento.target?.closest?.("[data-copiar-justificante]");
    if (!boton || typeof portapapeles?.writeText !== "function") return;
    portapapeles.writeText(boton.dataset.copiarJustificante).then(
      () => { boton.textContent = boton.dataset.textoCopiado || boton.textContent; },
      () => { boton.disabled = true; },
    );
  });
  return true;
}

/** Igual que renderizarJustificante, con los textos del traductor del módulo. */
export function justificanteTraducido(referencia, escapar, t) {
  return renderizarJustificante(referencia, {
    escapar, etiqueta: t("justificante_registrado"), copiar: t("justificante_copiar"), copiado: t("justificante_copiado"),
  });
}

/** Clave de recuperación de un resultado incierto: se ofrece copiarla, no se muestra. */
export function claveRecuperacionTraducida(clave, escapar, t) {
  return renderizarJustificante(clave, {
    escapar, etiqueta: t("clave_recuperacion_preparada"), copiar: t("clave_recuperacion_copiar"), copiado: t("clave_recuperacion_copiada"),
  });
}
