import { validarReciboV2ParaFichaGINPIX } from "./contrato-ficha-ginpix.js";
import { NOMBRE_FICHA_GINPIX } from "./cliente-http-ficha-ginpix.js";
import { escaparHTML } from "./componentes-expedientes.js";
export const CLAVES_I18N_FICHA_GINPIX = Object.freeze(["ficha_ginpix_titulo", "ficha_ginpix_recibo", "ficha_ginpix_nombre_archivo", "ficha_ginpix_descargar", "ficha_ginpix_aviso", "ficha_ginpix_descargando", "ficha_ginpix_error"]);
const TEXTOS = Object.freeze({
  ficha_ginpix_titulo: "Ficha GINPIX",
  ficha_ginpix_recibo: "Recibo de incorporación confirmado",
  ficha_ginpix_nombre_archivo: "Nombre de archivo",
  ficha_ginpix_descargar: "Descargar ficha GINPIX",
  ficha_ginpix_aviso: "Descarga sintética: no realiza envío, firma ni produce eficacia administrativa.",
  ficha_ginpix_descargando: "Preparando descarga sintética.",
  ficha_ginpix_error: "No se ha podido preparar la descarga.",
});
export function montarFichaGINPIX({ raiz, cliente, recibo, descargarArchivo, mensajes = {} } = {}) {
  if (!raiz?.addEventListener || !raiz?.removeEventListener || !raiz?.replaceChildren || typeof cliente?.descargarFichaGINPIX !== "function" || typeof descargarArchivo !== "function") throw new TypeError("dependencias de ficha GINPIX no válidas");
  const r = validarReciboV2ParaFichaGINPIX(recibo); let activa = true, ocupada = false, controlador; const t = (clave) => typeof mensajes[clave] === "string" ? mensajes[clave] : TEXTOS[clave]; const pintar = (estado = "") => { if (activa) raiz.innerHTML = `<section data-ct-ficha-ginpix><h3>${escaparHTML(t("ficha_ginpix_titulo"))}</h3><dl class="ct-resumen"><div><dt>${escaparHTML(t("ficha_ginpix_recibo"))}</dt><dd><code>${escaparHTML(r.recibo_ref)}</code></dd></div><div><dt>${escaparHTML(t("ficha_ginpix_nombre_archivo"))}</dt><dd><code>${escaparHTML(NOMBRE_FICHA_GINPIX)}</code></dd></div></dl><p>${escaparHTML(t("ficha_ginpix_aviso"))}</p><button type="button" data-ct-ficha-ginpix-descargar${ocupada ? " disabled" : ""}>${escaparHTML(t("ficha_ginpix_descargar"))}</button><p role="status" aria-live="polite">${estado ? escaparHTML(t(estado)) : ""}</p></section>`; };
  async function alPulsar(evento) { if (!evento.target?.matches?.("[data-ct-ficha-ginpix-descargar]") || ocupada) return; ocupada = true; const control = new AbortController(); controlador = control; let estado = "ficha_ginpix_descargando"; pintar(estado); try { const archivo = await cliente.descargarFichaGINPIX(r, { signal: control.signal }); if (activa && !control.signal.aborted) await descargarArchivo(archivo, NOMBRE_FICHA_GINPIX); estado = ""; } catch { if (!control.signal.aborted) estado = "ficha_ginpix_error"; } finally { ocupada = false; if (controlador === control) controlador = null; if (activa) pintar(estado); } }
  raiz.addEventListener("click", alPulsar); pintar(); return () => { activa = false; controlador?.abort(); raiz.removeEventListener("click", alPulsar); raiz.replaceChildren(); };
}
