/** Panel de tipos publicados: el catálogo y cada documento se autorizan en servidor. */
import { crearTraductorContratacionTemporal } from "./i18n.js?v=20261008-alta-rpt-circular-v6";
import { crearClienteBorradoresPublicados } from "./cliente-http-borradores-publicados.js?v=20260928-ppt-503-v5";

const esc = (valor) => String(valor ?? "").replaceAll("&", "&amp;")
  .replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");

export function renderizarBorradoresPublicados({ estado = "cargando", catalogo = null,
  ayudaAbierta = false, mensaje = "", mensajeError = false, ocupado = false } = {}) {
  const traducir = crearTraductorContratacionTemporal();
  const t = (clave, variables) => esc(traducir(`bp_${clave}`, variables));
  const filas = catalogo?.tipos?.map((tipo) => `<tr><th scope="row"><strong>${esc(tipo.etiqueta)}</strong></th>
    <td>${tipo.formatos.map((formato) => `<button type="button" class="boton-secundario" data-bp-descargar="${esc(tipo.clave)}" data-bp-formato="${formato}" ${ocupado ? "disabled" : ""}>${t(formato)}</button>`).join("")}</td></tr>`).join("") ?? "";
  const estadoTexto = estado === "cargando" ? t("cargando") : estado === "denegado" ? t("denegado")
    : estado === "conflicto" ? t("conflicto") : estado === "error" ? t("error") : "";
  return `<section class="panel ct-bp" data-ct-borradores-publicados>
    <header class="cabecera-panel"><div><h3>${t("titulo")}</h3></div>
      <button type="button" class="boton-secundario ct-bp-ayuda-boton" data-bp-ayuda aria-label="${t("ayuda_boton")}" aria-expanded="${ayudaAbierta}" aria-controls="ct-bp-ayuda">?</button></header>
    <p id="ct-bp-ayuda" class="ct-bp-ayuda" ${ayudaAbierta ? "" : "hidden"}>${t("ayuda")}</p>
    <div class="cuerpo-panel">${estadoTexto ? `<p role="${estado === "cargando" ? "status" : "alert"}">${estadoTexto}</p>` : ""}
      ${estado === "lista" && catalogo?.tipos?.length === 0 ? `<p role="status">${t("vacio")}</p>` : ""}
      ${estado === "lista" && filas ? `<div class="ct-bp-tabla" role="region" tabindex="0" aria-label="${t("tipo")}"><table class="tabla-datos"><thead><tr><th scope="col">${t("tipo")}</th><th scope="col">${t("formatos")}</th></tr></thead><tbody>${filas}</tbody></table></div>` : ""}
      ${catalogo && estado === "lista" ? `<p class="ct-bp-catalogo">${t("publicacion_recibo", { recibo: catalogo.procedencia_ref })}</p>` : ""}
      ${ocupado ? `<p role="status">${t("descargando")}</p><button type="button" class="boton-secundario" data-bp-cancelar>${t("cancelar")}</button>` : ""}
      ${mensaje ? `<p class="ct-bp-mensaje${mensajeError ? " ct-bp-mensaje--error" : ""}" role="${mensajeError ? "alert" : "status"}">${esc(mensaje)}</p>` : ""}
      ${["error", "conflicto"].includes(estado) ? `<button type="button" class="boton-secundario" data-bp-reintentar>${t("reintentar")}</button>` : ""}
    </div></section>`;
}

export function montarBorradoresPublicados({ raiz, contexto, cliente = crearClienteBorradoresPublicados(),
  entornoDescarga = globalThis, anunciar = () => {} } = {}) {
  if (!raiz?.addEventListener || !raiz?.removeEventListener || !raiz?.replaceChildren
    || typeof cliente?.consultarDisponibles !== "function" || typeof cliente?.descargar !== "function"
    || typeof anunciar !== "function") throw new TypeError("montaje de borradores publicados no válido");
  const t = crearTraductorContratacionTemporal();
  let montado = true;
  let estado = "cargando";
  let catalogo = null;
  let ayudaAbierta = false;
  let mensaje = "";
  let mensajeError = false;
  let ocupado = false;
  let controlador = null;
  let secuencia = 0;
  let urlDocumento = null;
  let revocacion = null;
  const pintar = () => { if (montado) {
    raiz.hidden = estado === "ausente";
    raiz.innerHTML = estado === "ausente" ? "" : renderizarBorradoresPublicados({ estado, catalogo, ayudaAbierta, mensaje, mensajeError, ocupado });
  } };
  const liberarURL = () => { clearTimeout(revocacion); revocacion = null;
    if (urlDocumento) entornoDescarga.URL?.revokeObjectURL?.(urlDocumento); urlDocumento = null; };
  const cancelar = () => { ++secuencia; controlador?.abort(); controlador = null; ocupado = false; };

  async function cargar() {
    cancelar();
    const actual = secuencia;
    controlador = new AbortController();
    const signal = controlador.signal;
    estado = "cargando"; catalogo = null; mensaje = ""; mensajeError = false; pintar();
    try {
      const datos = await cliente.consultarDisponibles(contexto, { signal });
      if (!montado || signal.aborted || secuencia !== actual) return;
      catalogo = datos; estado = "lista";
    } catch (error) {
      if (!montado || signal.aborted || secuencia !== actual) return;
      estado = error?.estado === 404 ? "ausente" : [401, 403].includes(error?.estado) ? "denegado"
        : error?.estado === 409 ? "conflicto" : "error";
    } finally { if (montado && secuencia === actual) { controlador = null; pintar(); } }
  }

  async function descargar(tipo, formato) {
    if (!catalogo || estado !== "lista" || ocupado) return;
    const autorizado = catalogo.tipos.some((item) => item.clave === tipo && item.formatos.includes(formato));
    if (!autorizado) return;
    const actual = ++secuencia;
    controlador?.abort(); controlador = new AbortController();
    const signal = controlador.signal;
    ocupado = true; mensaje = ""; mensajeError = false; pintar();
    try {
      const resultado = await cliente.descargar(contexto, catalogo, tipo, formato, { signal });
      if (!montado || signal.aborted || secuencia !== actual) return;
      if (resultado?.catalogo_ref !== catalogo.catalogo_ref
        || resultado.catalogo_huella_sha256 !== catalogo.catalogo_huella_sha256
        || resultado.procedencia_ref !== catalogo.procedencia_ref) throw new TypeError("procedencia de descarga incompatible");
      const documento = entornoDescarga.documento ?? entornoDescarga.document;
      const urls = entornoDescarga.URL;
      if (!documento?.body || typeof urls?.createObjectURL !== "function"
        || typeof urls?.revokeObjectURL !== "function") throw new TypeError("descarga no disponible");
      liberarURL();
      urlDocumento = urls.createObjectURL(new Blob([resultado.bytes], { type: resultado.mime }));
      const enlace = documento.createElement("a");
      try { enlace.href = urlDocumento; enlace.download = resultado.nombre; enlace.hidden = true;
        documento.body.append(enlace); enlace.click(); }
      finally { enlace.remove(); revocacion = setTimeout(liberarURL, 0); }
      mensaje = `${t("bp_listo", { nombre: resultado.nombre })}. ${t("bp_publicacion_recibo", { recibo: resultado.procedencia_ref })}. ${t("bp_huella", { huella: resultado.huella_sha256 })}`;
      mensajeError = false;
      anunciar(mensaje, "informacion");
    } catch (error) {
      if (!montado || signal.aborted || secuencia !== actual) return;
      if (error?.estado === 409) { estado = "conflicto"; catalogo = null; }
      else if ([401, 403].includes(error?.estado)) { estado = "denegado"; catalogo = null; }
      mensaje = t("bp_descarga_error"); mensajeError = true; anunciar(mensaje, "error");
    } finally { if (montado && secuencia === actual) { controlador = null; ocupado = false; pintar(); } }
  }

  const pulsar = (evento) => {
    const accion = evento.target?.closest?.("[data-bp-ayuda], [data-bp-reintentar], [data-bp-cancelar], [data-bp-descargar]");
    if (!accion || !raiz.contains?.(accion)) return;
    if (accion.hasAttribute("data-bp-ayuda")) { ayudaAbierta = !ayudaAbierta; pintar();
      raiz.querySelector?.("[data-bp-ayuda]")?.focus?.({ preventScroll: true }); }
    else if (accion.hasAttribute("data-bp-reintentar")) void cargar();
    else if (accion.hasAttribute("data-bp-cancelar")) { cancelar(); mensaje = t("bp_cancelada"); mensajeError = false; pintar(); }
    else void descargar(accion.dataset.bpDescargar, accion.dataset.bpFormato);
  };
  raiz.addEventListener("click", pulsar); pintar(); void cargar();
  return Object.freeze({ desmontar() { montado = false; cancelar(); liberarURL();
    raiz.removeEventListener("click", pulsar); raiz.replaceChildren(); } });
}
