/**
 * Justificante de una actuación registrada. La referencia del recibo es opaca:
 * no se muestra en pantalla, solo se ofrece copiarla para el seguimiento. Los
 * textos los aporta cada módulo desde su propio catálogo i18n.
 */

const DOCUMENTOS_CON_COPIA = new WeakSet();

export function renderizarJustificante(referencia, { escapar, etiqueta, copiar, copiado, descripcion = "" }) {
  if (typeof referencia !== "string" || referencia.trim() === "") return escapar("—");
  const rotulo = etiqueta ? `<span class="justificante-registrado">${escapar(etiqueta)}</span> ` : "";
  const aria = descripcion ? ` aria-label="${escapar(descripcion)}"` : "";
  return `${rotulo}<button type="button" class="boton-terciario boton-copiar-justificante" data-copiar-justificante="${escapar(referencia)}"`
    + ` data-texto-copiado="${escapar(copiado)}"${aria}>${escapar(copiar)}</button>`;
}

/**
 * Referencia interna de una fila (convocatoria, actuación, persona que actuó):
 * no se muestra, viaja en el botón para copiarla. `descripcion` nombra la fila
 * para el lector de pantalla, porque todos los botones dicen lo mismo.
 */
export function referenciaCopiableTraducida(referencia, escapar, t, descripcion = "") {
  return renderizarJustificante(referencia, {
    escapar, etiqueta: "", copiar: t("justificante_copiar"), copiado: t("justificante_copiado"), descripcion,
  });
}

/**
 * Un valor es una referencia interna cuando no es texto para personas: sin
 * espacios y con prefijo técnico (`per_x`, `recibo:1`), UUID o huella hexadecimal.
 */
export function esReferenciaInterna(valor) {
  if (typeof valor !== "string") return false;
  const texto = valor.trim();
  if (texto === "" || /\s/u.test(texto)) return false;
  return /^[a-z][a-z0-9-]*[_:]/u.test(texto)
    || /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/iu.test(texto)
    || /^[0-9a-f]{32,}$/iu.test(texto);
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

/**
 * Quien registró una actuación: si llega como referencia de identidad se
 * nombra por su papel y la referencia se ofrece para copiar; un nombre escrito
 * por una persona se muestra tal cual.
 */
export function actorTraducido(valor, escapar, t) {
  if (!esReferenciaInterna(valor)) return escapar(typeof valor === "string" && valor.trim() ? valor : "—");
  return `${escapar(t("actor_rrhh"))} ${referenciaCopiableTraducida(valor, escapar, t, t("actor_copiar_aria"))}`;
}
