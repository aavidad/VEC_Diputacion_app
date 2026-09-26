/**
 * Incorporación acreditada en el panel de seguimiento (dudas 11 y 12 de RRHH):
 * confirmación de GINPIX por RRHH, número del cierre tomado de ella y
 * confirmación de la incorporación por el centro. Solo pinta; el servidor
 * decide y revalida todo.
 */

/** Etiqueta del tipo de documento acreditativo; sin traducción, la clave. */
export function etiquetaDocumento(tipo, t) {
  const clave = `documento_${tipo}`;
  const texto = t(clave);
  return texto === clave ? tipo : texto;
}

/** Filas del resumen: confirmación del centro y de GINPIX, si se componen. */
export function filasIncorporacionAcreditada(estado, opciones, t, fecha) {
  if (opciones?.confirmacion_ginpix !== true) return [];
  const centro = estado.confirmacion_centro;
  const ginpix = estado.ginpix;
  return [
    [t("centro_confirmacion"), centro
      ? t("centro_confirmada", { fecha: fecha(centro.fecha_incorporacion), documento: etiquetaDocumento(centro.documento_tipo, t) })
      : t("pendiente")],
    [t("ginpix"), ginpix ? t("ginpix_confirmado", { numero: ginpix.ginpix_numero, fecha: fecha(ginpix.ginpix_confirmada_en) }) : t("pendiente")],
  ];
}

/** Se ofrece la confirmación de GINPIX con incorporación y sin confirmar. */
export function ofrecerConfirmacionGINPIX(estado, opciones) {
  return opciones?.confirmacion_ginpix === true && Boolean(estado.incorporacion) && !estado.ginpix && !estado.cierre;
}

export function formularioConfirmacionGINPIX({ t, escapar, deshabilitado }) {
  return `<form class="ct-seg-cese-form" data-ct-seg-form="ginpix" aria-labelledby="ct-seg-ginpix-titulo" novalidate>
      <h4 id="ct-seg-ginpix-titulo">${escapar(t("ginpix_titulo"))}</h4>
      <div class="ct-seg-cese-rejilla">
      <label class="ct-campo"><span>${escapar(t("cierre_ginpix_numero"))}</span><input type="text" name="ginpix_numero" required maxlength="64" autocomplete="off" ${deshabilitado}></label>
      <label class="ct-campo"><span>${escapar(t("cierre_ginpix_fecha"))}</span><input type="date" name="ginpix_confirmada_en" required ${deshabilitado}></label>
      <label class="ct-campo ct-seg-cese-ancho"><span>${escapar(t("ginpix_observaciones"))}</span><textarea name="observaciones" rows="2" maxlength="2000" ${deshabilitado}></textarea></label>
      </div>
      <div class="ct-acciones"><button type="submit" class="boton-primario" ${deshabilitado}>${escapar(t("ginpix_enviar"))}</button></div></form>`;
}

/**
 * Parte del cierre relativa a GINPIX con la incorporación acreditada:
 * `{ ofrecer, html }`. Si la regla exige GINPIX y no hay confirmación, el
 * cierre no se ofrece y se explica por qué; si la hay, su número se muestra
 * sin poder cambiarse.
 */
export function cierreConGINPIXConfirmado({ opciones, estado, t, escapar, fecha }) {
  if (!opciones.condiciones_cierre.includes("ginpix_confirmado")) return { ofrecer: true, html: "" };
  if (!estado.ginpix) return { ofrecer: false, html: `<p class="ct-seg-cese-motivo" role="note">${escapar(t("cierre_requiere_ginpix"))}</p>` };
  return {
    ofrecer: true,
    html: `<dl class="ct-exp-fase-datos" data-ct-seg-ginpix-cierre><div><dt>${escapar(t("cierre_ginpix_numero"))}</dt><dd>${escapar(estado.ginpix.ginpix_numero)}</dd></div>
      <div><dt>${escapar(t("cierre_ginpix_fecha"))}</dt><dd>${escapar(fecha(estado.ginpix.ginpix_confirmada_en))}</dd></div></dl>`,
  };
}
