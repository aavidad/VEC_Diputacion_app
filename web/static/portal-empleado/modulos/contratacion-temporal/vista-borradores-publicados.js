/** Panel de tipos publicados: el catálogo y cada documento se autorizan en servidor. */
import { crearTraductorContratacionTemporal } from "./i18n.js?v=20261008-alta-rpt-circular-v6";
import { crearClienteBorradoresPublicados } from "./cliente-http-borradores-publicados.js?v=20261009-borradores-fase-v1";

const esc = (valor) => String(valor ?? "").replaceAll("&", "&amp;")
  .replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");

export function renderizarBorradoresPublicados({ estado = "cargando", catalogo = null,
  ayudaAbierta = false, mensaje = "", mensajeError = false, tipoAvisado = "", ocupado = false } = {}) {
  const traducir = crearTraductorContratacionTemporal();
  const t = (clave, variables) => esc(traducir(`bp_${clave}`, variables));
  // Un tipo que el expediente todavía no permite preparar se ve, pero sus
  // botones quedan inactivos (siguen enfocables para que se lea el motivo).
  // Cada botón nombra su documento y, si toca, por qué no está disponible.
  // El aviso de un documento se pinta junto a sus botones.
  const filas = catalogo?.tipos?.map((tipo) => {
    const listo = tipo.disponible !== false;
    const idTitulo = `ct-bp-documento-${esc(tipo.clave)}`;
    const idEstado = `ct-bp-estado-${esc(tipo.clave)}`;
    const descrito = listo ? idTitulo : `${idTitulo} ${idEstado}`;
    const aviso = mensaje && !mensajeError && tipoAvisado === tipo.clave
      ? `<p class="ct-bp-mensaje ct-bp-mensaje--aviso" role="status">${esc(mensaje)}</p>` : "";
    return `<li class="ct-exp-documento${listo ? "" : " ct-bp-pendiente"}">
    <div class="ct-exp-documento-principal"><strong id="${idTitulo}">${esc(tipo.etiqueta)}</strong>
      <span class="ct-exp-chip" id="${idEstado}">${t(listo ? "estado_borrador" : "estado_pendiente")}</span></div>
    <div class="ct-exp-borradores-acciones">${tipo.formatos.map((formato) => `<button type="button" class="boton-secundario" data-bp-descargar="${esc(tipo.clave)}" data-bp-formato="${formato}"${listo ? "" : ' aria-disabled="true"'} aria-describedby="${descrito}" ${ocupado ? "disabled" : ""}>${t(formato === "docx" ? "ficha_docx" : formato)}</button>`).join("")}</div>${aviso}
  </li>`;
  }).join("") ?? "";
  const estadoTexto = estado === "cargando" ? t("cargando") : estado === "denegado" ? t("denegado")
    : estado === "conflicto" ? t("conflicto") : estado === "error" ? t("error") : "";
  return `<section class="ct-bp" data-ct-borradores-publicados aria-labelledby="ct-bp-titulo">
    <header class="cabecera-panel"><div><h4 id="ct-bp-titulo">${t("ficha_titulo")}</h4></div>
      <button type="button" class="boton-secundario ct-bp-ayuda-boton" data-bp-ayuda aria-label="${t("ayuda_boton")}" aria-expanded="${ayudaAbierta}" aria-controls="ct-bp-ayuda">?</button></header>
    <p id="ct-bp-ayuda" class="ct-bp-ayuda" ${ayudaAbierta ? "" : "hidden"}>${t("ayuda")}</p>
    <div class="cuerpo-panel">${estadoTexto ? `<p role="${estado === "cargando" ? "status" : "alert"}">${estadoTexto}</p>` : ""}
      ${estado === "lista" && catalogo?.tipos?.length === 0 ? `<p role="status">${t("vacio")}</p>` : ""}
      ${estado === "lista" && filas ? `<ul class="ct-exp-documentos-lista" aria-label="${t("tipo")}">${filas}</ul>` : ""}
      ${catalogo && estado === "lista" ? `<p class="ct-bp-catalogo">${t("publicacion_recibo", { recibo: catalogo.procedencia_ref })}</p>` : ""}
      ${ocupado ? `<p role="status">${t("descargando")}</p><button type="button" class="boton-secundario" data-bp-cancelar>${t("cancelar")}</button>` : ""}
      ${mensaje && (mensajeError || !tipoAvisado) ? `<p class="ct-bp-mensaje${mensajeError ? " ct-bp-mensaje--error" : ""}" role="${mensajeError ? "alert" : "status"}">${esc(mensaje)}</p>` : ""}
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
  let tipoAvisado = "";
  let ocupado = false;
  let controlador = null;
  let secuencia = 0;
  let urlDocumento = null;
  let revocacion = null;
  const pintar = () => { if (montado) {
    raiz.hidden = false;
    raiz.innerHTML = renderizarBorradoresPublicados({ estado, catalogo, ayudaAbierta, mensaje, mensajeError, tipoAvisado, ocupado });
  } };
  const liberarURL = () => { clearTimeout(revocacion); revocacion = null;
    if (urlDocumento) entornoDescarga.URL?.revokeObjectURL?.(urlDocumento); urlDocumento = null; };
  const cancelar = () => { ++secuencia; controlador?.abort(); controlador = null; ocupado = false; };

  async function cargar() {
    cancelar();
    const actual = secuencia;
    controlador = new AbortController();
    const signal = controlador.signal;
    estado = "cargando"; catalogo = null; mensaje = ""; mensajeError = false; tipoAvisado = ""; pintar();
    try {
      const datos = await cliente.consultarDisponibles(contexto, { signal });
      if (!montado || signal.aborted || secuencia !== actual) return;
      catalogo = datos; estado = "lista";
    } catch (error) {
      if (!montado || signal.aborted || secuencia !== actual) return;
      estado = [401, 403].includes(error?.estado) ? "denegado"
        : error?.estado === 409 ? "conflicto" : "error";
    } finally { if (montado && secuencia === actual) { controlador = null; pintar(); } }
  }

  async function descargar(tipo, formato) {
    if (!catalogo || estado !== "lista" || ocupado) return;
    const elegido = catalogo.tipos.find((item) => item.clave === tipo && item.formatos.includes(formato));
    if (!elegido) return;
    if (elegido.disponible === false) { avisarNoDisponible(tipo); pintar(); enfocar(tipo, formato); return; }
    const actual = ++secuencia;
    controlador?.abort(); controlador = new AbortController();
    const signal = controlador.signal;
    ocupado = true; mensaje = ""; mensajeError = false; tipoAvisado = ""; pintar();
    let reenfocar = false;
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
      mensajeError = false; tipoAvisado = "";
      anunciar(mensaje, "informacion");
    } catch (error) {
      if (!montado || signal.aborted || secuencia !== actual) return;
      if (error?.codigo === "documento_no_disponible") {
        // No es un cambio del expediente: la lista sigue valiendo y ese tipo
        // queda marcado como todavía no disponible.
        catalogo = Object.freeze({ ...catalogo, tipos: Object.freeze(catalogo.tipos.map((item) => item.clave === tipo
          ? Object.freeze({ ...item, disponible: false }) : item)) });
        avisarNoDisponible(tipo); reenfocar = true;
        return;
      }
      if (error?.estado === 409) { estado = "conflicto"; catalogo = null; }
      else if ([401, 403].includes(error?.estado)) { estado = "denegado"; catalogo = null; }
      mensaje = t("bp_descarga_error"); mensajeError = true; tipoAvisado = ""; anunciar(mensaje, "error");
    } finally {
      if (montado && secuencia === actual) { controlador = null; ocupado = false; pintar(); if (reenfocar) enfocar(tipo, formato); }
    }
  }

  // Repintar sustituye los botones: se devuelve el foco al pulsado.
  const enfocar = (tipo, formato) => [...(raiz.querySelectorAll?.("[data-bp-descargar]") ?? [])]
    .find((boton) => boton.dataset?.bpDescargar === tipo && boton.dataset?.bpFormato === formato)
    ?.focus?.({ preventScroll: true });

  function avisarNoDisponible(tipo) {
    mensaje = t("bp_no_disponible"); mensajeError = false; tipoAvisado = tipo;
    anunciar(mensaje, "informacion");
  }

  const pulsar = (evento) => {
    const accion = evento.target?.closest?.("[data-bp-ayuda], [data-bp-reintentar], [data-bp-cancelar], [data-bp-descargar]");
    if (!accion || !raiz.contains?.(accion)) return;
    if (accion.hasAttribute("data-bp-ayuda")) { ayudaAbierta = !ayudaAbierta; pintar();
      raiz.querySelector?.("[data-bp-ayuda]")?.focus?.({ preventScroll: true }); }
    else if (accion.hasAttribute("data-bp-reintentar")) void cargar();
    else if (accion.hasAttribute("data-bp-cancelar")) { cancelar(); mensaje = t("bp_cancelada"); mensajeError = false; tipoAvisado = ""; pintar(); }
    else void descargar(accion.dataset.bpDescargar, accion.dataset.bpFormato);
  };
  raiz.addEventListener("click", pulsar); pintar(); void cargar();
  return Object.freeze({ desmontar() { montado = false; cancelar(); liberarURL();
    raiz.removeEventListener("click", pulsar); raiz.replaceChildren(); } });
}
