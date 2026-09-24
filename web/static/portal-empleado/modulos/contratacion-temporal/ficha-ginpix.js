import { validarReciboV2ParaFichaGINPIX } from "./contrato-ficha-ginpix.js";
import { NOMBRE_FICHA_GINPIX } from "./cliente-http-ficha-ginpix.js";
import { escaparHTML } from "./componentes-expedientes.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";
export const CLAVES_I18N_FICHA_GINPIX = Object.freeze(["ficha_ginpix_titulo", "ficha_ginpix_resumen_titulo", "ficha_ginpix_destino", "ficha_ginpix_categoria", "ficha_ginpix_inicio", "ficha_ginpix_fin", "ficha_ginpix_registrada", "ficha_ginpix_recibo", "ficha_ginpix_exportacion_manual", "ficha_ginpix_exportacion_manual_disponible", "ficha_ginpix_transmision", "ficha_ginpix_transmision_pendiente", "ficha_ginpix_nombre_archivo", "ficha_ginpix_descargar", "ficha_ginpix_aviso", "ficha_ginpix_descargando", "ficha_ginpix_error"]);
function contextoResumenGINPIX(valor) {
  if (valor === undefined) return null;
  if (!valor || typeof valor !== "object" || Array.isArray(valor) || Object.getPrototypeOf(valor) !== Object.prototype
    || Object.getOwnPropertySymbols(valor).length || Object.keys(valor).length !== 2
    || !Object.hasOwn(valor, "centro") || !Object.hasOwn(valor, "categoria")
    || ![valor.centro, valor.categoria].every((campo) => typeof campo === "string" && campo.trim().length > 0 && campo.length <= 160)) {
    throw new TypeError("contexto de resumen GINPIX no válido");
  }
  return Object.freeze({ centro: valor.centro, categoria: valor.categoria });
}
function fechaGINPIX(valor, locale, zonaHoraria, incluyeHora = false) {
  const opciones = incluyeHora ? { dateStyle: "medium", timeStyle: "short" } : { dateStyle: "medium" };
  try { return new Intl.DateTimeFormat(locale, { ...opciones, timeZone: zonaHoraria }).format(new Date(valor)); }
  catch { return valor; }
}
export function montarFichaGINPIX({ raiz, cliente, recibo, descargarArchivo, mensajes = {}, resumen, locale = "es-ES", zonaHoraria = "Europe/Madrid" } = {}) {
  if (!raiz?.addEventListener || !raiz?.removeEventListener || !raiz?.replaceChildren || typeof cliente?.descargarFichaGINPIX !== "function" || typeof descargarArchivo !== "function") throw new TypeError("dependencias de ficha GINPIX no válidas");
  const r = validarReciboV2ParaFichaGINPIX(recibo); const contexto = contextoResumenGINPIX(resumen); let activa = true, ocupada = false, controlador; const t = crearTraductorContratacionTemporal(mensajes); const pintar = () => { if (activa) raiz.innerHTML = `<section data-ct-ficha-ginpix><h3>${escaparHTML(t("ficha_ginpix_titulo"))}</h3>${contexto ? `<section class="ct-resumen-documentos" aria-labelledby="ct-ficha-ginpix-resumen"><h4 id="ct-ficha-ginpix-resumen">${escaparHTML(t("ficha_ginpix_resumen_titulo"))}</h4><dl class="ct-resumen"><div><dt>${escaparHTML(t("ficha_ginpix_destino"))}</dt><dd>${escaparHTML(contexto.centro)}</dd></div><div><dt>${escaparHTML(t("ficha_ginpix_categoria"))}</dt><dd>${escaparHTML(contexto.categoria)}</dd></div><div><dt>${escaparHTML(t("ficha_ginpix_inicio"))}</dt><dd><time datetime="${escaparHTML(r.periodo_incorporacion.desde)}">${escaparHTML(fechaGINPIX(r.periodo_incorporacion.desde, locale, "UTC"))}</time></dd></div><div><dt>${escaparHTML(t("ficha_ginpix_fin"))}</dt><dd><time datetime="${escaparHTML(r.periodo_incorporacion.hasta)}">${escaparHTML(fechaGINPIX(r.periodo_incorporacion.hasta, locale, "UTC"))}</time></dd></div><div><dt>${escaparHTML(t("ficha_ginpix_registrada"))}</dt><dd><time datetime="${escaparHTML(r.registrada_en)}">${escaparHTML(fechaGINPIX(r.registrada_en, locale, zonaHoraria, true))}</time></dd></div><div><dt>${escaparHTML(t("ficha_ginpix_recibo"))}</dt><dd><code>${escaparHTML(r.recibo_ref)}</code></dd></div><div><dt>${escaparHTML(t("ficha_ginpix_exportacion_manual"))}</dt><dd>${escaparHTML(t("ficha_ginpix_exportacion_manual_disponible"))}</dd></div><div><dt>${escaparHTML(t("ficha_ginpix_transmision"))}</dt><dd>${escaparHTML(t("ficha_ginpix_transmision_pendiente"))}</dd></div></dl></section>` : ""}<dl class="ct-resumen">${contexto ? "" : `<div><dt>${escaparHTML(t("ficha_ginpix_recibo"))}</dt><dd><code>${escaparHTML(r.recibo_ref)}</code></dd></div>`}<div><dt>${escaparHTML(t("ficha_ginpix_nombre_archivo"))}</dt><dd><code>${escaparHTML(NOMBRE_FICHA_GINPIX)}</code></dd></div></dl><p>${escaparHTML(t("ficha_ginpix_aviso"))}</p><button type="button" data-ct-ficha-ginpix-descargar aria-disabled="false" aria-busy="false">${escaparHTML(t("ficha_ginpix_descargar"))}</button><p data-ct-ficha-ginpix-estado role="status" aria-live="polite"></p></section>`; };
  pintar();
  const boton = raiz.querySelector?.("[data-ct-ficha-ginpix-descargar]");
  const estadoVisible = raiz.querySelector?.("[data-ct-ficha-ginpix-estado]");
  function actualizarEstado(estado) {
    if (!activa || (boton && raiz.contains && !raiz.contains(boton))) return;
    boton?.setAttribute("aria-busy", String(ocupada));
    boton?.setAttribute("aria-disabled", String(ocupada));
    if (estadoVisible) estadoVisible.textContent = estado ? t(estado) : "";
  }
  async function alPulsar(evento) {
    if (!evento.target?.matches?.("[data-ct-ficha-ginpix-descargar]") || ocupada || (raiz.contains && !raiz.contains(evento.target))) return;
    ocupada = true;
    const control = new AbortController(); controlador = control;
    let estado = "ficha_ginpix_descargando";
    actualizarEstado(estado);
    try {
      const archivo = await cliente.descargarFichaGINPIX(r, { signal: control.signal });
      if (activa && !control.signal.aborted) await descargarArchivo(archivo, NOMBRE_FICHA_GINPIX);
      estado = "";
    } catch { if (!control.signal.aborted) estado = "ficha_ginpix_error"; }
    finally {
      ocupada = false;
      if (controlador === control) controlador = null;
      actualizarEstado(estado);
    }
  }
  raiz.addEventListener("click", alPulsar);
  return () => { activa = false; controlador?.abort(); raiz.removeEventListener("click", alPulsar); raiz.replaceChildren(); };
}
