import { cargarTextos, reintentarTextos } from "../../../comun/textos.js";
import { crearClienteInscripcionesRRHH } from "./inscripcion-rrhh-cliente.js?v=20261009-inscripciones-v1";

const ESTADOS = new Set(["pendiente", "admitida_a_convocatoria", "incorporada", "rechazada"]);
const esc = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
  .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");
const claveNueva = () => `inscripcion-${globalThis.crypto.randomUUID()}`;

export function leerRutaInscripcionesRRHH(busqueda = "") {
  const q = new URLSearchParams(busqueda);
  const estado = q.get("inscripcion_estado") || "pendiente";
  const convocatoria = q.get("inscripcion_convocatoria") || "";
  const cursor = q.get("inscripcion_cursor") || "";
  const selectorCursor = q.get("inscripcion_convocatorias_cursor") || "";
  return Object.freeze({ estado: ESTADOS.has(estado) ? estado : "pendiente",
    convocatoria: convocatoria.length <= 512 ? convocatoria : "", cursor: cursor.length <= 512 ? cursor : "",
    selectorCursor: selectorCursor.length <= 512 ? selectorCursor : "" });
}

export function rutaInscripcionesRRHH(actual, filtro) {
  const url = new URL(actual);
  for (const nombre of ["inscripcion_estado", "inscripcion_convocatoria", "inscripcion_cursor",
    "inscripcion_convocatorias_cursor"]) url.searchParams.delete(nombre);
  if (filtro.estado && filtro.estado !== "pendiente") url.searchParams.set("inscripcion_estado", filtro.estado);
  if (filtro.convocatoria) url.searchParams.set("inscripcion_convocatoria", filtro.convocatoria);
  if (filtro.cursor) url.searchParams.set("inscripcion_cursor", filtro.cursor);
  if (filtro.selectorCursor) url.searchParams.set("inscripcion_convocatorias_cursor", filtro.selectorCursor);
  return `${url.pathname}${url.search}${url.hash}`;
}

export async function montarInscripcionesRRHH({ raiz, cliente = crearClienteInscripcionesRRHH(),
  localizacion = globalThis.location, historial = globalThis.history,
  cargarCatalogo = cargarTextos, reintentarCatalogo = reintentarTextos,
  alDenegacion = () => {}, signal } = {}) {
  if (!raiz?.addEventListener || !raiz?.removeEventListener || !raiz?.replaceChildren
    || !cliente?.convocatorias || !cliente?.listar || !cliente?.detalle || !cliente?.motivos
    || !cliente?.decidir || !cliente?.incorporar) {
    throw new TypeError("superficie no disponible");
  }
  let vivo = true;
  let secuencia = 0;
  let controlador = null;
  let catalogo = null;
  let listado = null;
  let convocatorias = null;
  let seleccion = null;
  let detalle = null;
  let motivos = null;
  let decision = "";
  let recibo = null;
  let estadoVista = "cargando";
  let ayuda = false;
  let filtro = leerRutaInscripcionesRRHH(localizacion.search);
  let intento = null;
  let enviando = false;
  let enfocarLista = false;
  let enfocarRecibo = false;
  const t = (clave, vars = {}) => catalogo.traducir(`rrhh.${clave}`, vars);
  const et = (clave, vars) => esc(t(clave, vars));
  const fecha = (valor) => esc(catalogo.fecha(valor, { dateStyle: "short", timeStyle: "short", timeZone: "Europe/Madrid" }));

  function limpiarDenegacion() {
    ++secuencia; controlador?.abort(); listado = null; convocatorias = null; seleccion = null;
    detalle = null; motivos = null; recibo = null;
    intento = null; estadoVista = "denegada"; alDenegacion(); pintar();
  }
  function pintar() {
    if (!vivo || !catalogo) return;
    if (!filtro.convocatoria) {
      const items = convocatorias?.convocatorias || [];
      const tarjetas = items.map((c) => `<a class="tarjeta-modulo tarjeta-modulo-habilitada" href="${esc(rutaInscripcionesRRHH(localizacion.href, { ...filtro, convocatoria: c.convocatoria_ref, cursor: "" }))}" data-inscripcion-elegir="${esc(c.convocatoria_ref)}"><strong>${esc(c.titulo)}</strong><span>${et("primera_categoria")}: ${esc(c.categorias_resumen)}</span><span class="estado-chip ${c.estado_publicacion === "publicada" ? "exito" : c.estado_publicacion === "sustituida" ? "violeta" : "peligro"}">${et(`publicacion_${c.estado_publicacion}`)}</span><time datetime="${esc(c.plazo_fin)}">${et("fin_plazo")}: ${fecha(c.plazo_fin)}</time>${Date.parse(c.plazo_fin) <= Date.now() ? `<span class="estado-chip aviso">${et("plazo_finalizado")}</span>` : ""}<span>${et("ver_solicitudes")}</span></a>`).join("");
      const aviso = estadoVista === "cargando" ? `<p role="status" aria-busy="true">${et("cargando_convocatorias")}</p>`
        : estadoVista === "denegada" ? `<p role="alert">${et("denegada")}</p>`
          : estadoVista === "error" ? `<p role="alert">${et("error_convocatorias")}</p><button type="button" class="boton-secundario" data-inscripcion-reintentar>${et("reintentar")}</button>`
            : convocatorias?.total === 0 ? `<p>${et("vacio_convocatorias")}</p>` : "";
      const total = convocatorias ? `<a href="${esc(rutaInscripcionesRRHH(localizacion.href, { ...filtro, selectorCursor: "" }))}" data-inscripcion-selector-total>${et("total_convocatorias", { cuenta: catalogo.numero(convocatorias.total) })}</a>` : "";
      const siguiente = convocatorias?.cursor_siguiente ? `<nav aria-label="${et("paginacion_convocatorias")}"><a href="${esc(rutaInscripcionesRRHH(localizacion.href, { ...filtro, selectorCursor: convocatorias.cursor_siguiente }))}" data-inscripcion-selector-siguiente>${et("siguiente_convocatorias")}</a></nav>` : "";
      raiz.innerHTML = `<section class="panel" aria-labelledby="inscripciones-titulo"><header class="cabecera-panel"><div><h2 id="inscripciones-titulo" tabindex="-1">${et("titulo")}</h2></div><button type="button" class="boton-secundario" data-inscripcion-ayuda aria-expanded="${ayuda}" aria-controls="inscripcion-ayuda" aria-label="${et("ayuda_boton")}">?</button></header><div class="cuerpo-panel"><div id="inscripcion-ayuda" ${ayuda ? "" : "hidden"}>${et("ayuda")}</div><h3>${et("elegir_convocatoria")}</h3>${aviso}${total}${tarjetas ? `<div class="rejilla-modulos">${tarjetas}</div>` : ""}${siguiente}</div></section>`;
      return;
    }
    const lista = listado?.solicitudes || [];
    const filas = lista.map((s) => `<tr><th scope="row"><button type="button" class="enlace-tabla" data-inscripcion-abrir="${esc(s.solicitud_ref)}">${esc(s.persona_resumen || t("ver_solicitud"))}</button></th>
      <td><a class="enlace-tabla" href="${esc(rutaInscripcionesRRHH(localizacion.href, { ...filtro, convocatoria: s.convocatoria_ref, cursor: "" }))}" data-inscripcion-convocatoria="${esc(s.convocatoria_ref)}">${esc(s.categoria)}</a></td><td><span class="estado-chip ${s.estado === "pendiente" ? "aviso" : s.estado === "incorporada" ? "exito" : s.estado === "rechazada" ? "peligro" : "info"}">${et(`estado_${s.estado}`)}</span></td><td><time datetime="${esc(s.registrada_en)}">${fecha(s.registrada_en)}</time></td></tr>`).join("");
    const opcionesEstado = [...ESTADOS].map((e) => `<option value="${e}" ${filtro.estado === e ? "selected" : ""}>${et(`estado_${e}`)}</option>`).join("");
    const nombreConvocatoria = listado?.convocatoria_titulo
      || (seleccion?.convocatoria_ref === filtro.convocatoria ? seleccion.titulo : "")
      || t("convocatoria_seleccionada");
    const filtroActivo = filtro.convocatoria ? `<p>${et("filtro_activo")}: ${esc(nombreConvocatoria)}. <a href="${esc(rutaInscripcionesRRHH(localizacion.href, { ...filtro, convocatoria: "", cursor: "" }))}" data-inscripcion-quitar-filtro>${et("cambiar_convocatoria")}</a></p>` : "";
    const estado = estadoVista === "cargando" ? `<p role="status" aria-busy="true">${et("cargando")}</p>`
      : estadoVista === "denegada" ? `<p role="alert">${et("denegada")}</p>`
        : ["error", "conflicto", "falta_acta", "identidad_pendiente", "regla_incompatible", "evidencia_invalida", "plazo_cerrado", "requisitos_pendientes"].includes(estadoVista) ? `<p role="alert">${et(estadoVista === "error" && detalle && decision ? "error_decision" : estadoVista)}</p>${detalle && decision ? "" : `<button type="button" class="boton-secundario" data-inscripcion-reintentar>${et("reintentar")}</button>`}`
          : listado?.total === 0 ? `<p>${et("vacio")}</p>` : "";
    const cuenta = listado ? `<a href="${esc(rutaInscripcionesRRHH(localizacion.href, { ...filtro, cursor: "" }))}" data-inscripcion-total>${et("total", { cuenta: catalogo.numero(listado.total) })}</a>` : "";
    const paginacion = listado?.cursor_siguiente ? `<nav aria-label="${et("paginacion")}"><a href="${esc(rutaInscripcionesRRHH(localizacion.href, { ...filtro, cursor: listado.cursor_siguiente }))}" data-inscripcion-siguiente>${et("siguiente")}</a></nav>` : "";
    let ficha = "";
    if (detalle) {
      const opciones = motivos?.motivos?.map((m) => `<option value="${esc(m.codigo)}" ${intento?.motivoCodigo === m.codigo ? "selected" : ""}>${esc(m.etiqueta)}</option>`).join("") || "";
      const exigeMotivo = decision === "rechazar" || motivos?.motivos?.some((m) => m.obligatorio);
      const selector = decision && motivos?.motivos?.length ? `<label for="inscripcion-motivo">${et("motivo")}</label><select id="inscripcion-motivo" data-inscripcion-motivo ${exigeMotivo ? "required" : ""}><option value="">${et("sin_motivo")}</option>${opciones}</select>` : "";
      const sinMotivos = decision === "rechazar" && !motivos?.motivos?.length;
      const requisitosNoResueltos = detalle.requisitos?.some((r) => r.obligatorio && r.estado !== "cumple") ?? true;
      const requisitos = `<section aria-labelledby="inscripcion-requisitos"><h4 id="inscripcion-requisitos">${et("requisitos")}</h4><ul>${(detalle.requisitos || []).map((r) => `<li>${esc(r.descripcion)} — ${et(`requisito_${r.estado}`)}${r.motivo_etiqueta ? ` · ${esc(r.motivo_etiqueta)}` : ""}${r.hito_etiqueta ? ` · ${esc(r.hito_etiqueta)}` : ""}</li>`).join("")}</ul></section>`;
      const evidencia = decision === "incorporar" ? `<label for="inscripcion-evidencia">${et("evidencia")}</label><input id="inscripcion-evidencia" data-inscripcion-evidencia value="${esc(intento?.evidenciaRef || "")}" required maxlength="512">` : "";
      const confirmar = decision ? `${sinMotivos ? `<p role="status">${et("motivos_no_disponibles")}</p>` : ""}<div class="acciones-vista"><button type="button" class="boton-secundario" data-inscripcion-cancelar>${et("cambiar")}</button><button type="button" class="boton-principal" data-inscripcion-confirmar ${enviando || sinMotivos ? "disabled" : ""}>${et(decision === "admitir" ? "confirmar_admision" : decision === "incorporar" ? "confirmar_incorporacion" : "confirmar_rechazo")}</button></div>`
        : detalle.estado === "pendiente" ? `<div class="acciones-vista">${requisitosNoResueltos ? `<p>${et("requisitos_pendientes")}</p>` : ""}<button type="button" class="boton-principal" data-inscripcion-decidir="admitir" ${requisitosNoResueltos ? "disabled" : ""}>${et("admitir")}</button><button type="button" class="boton-secundario" data-inscripcion-decidir="rechazar">${et("rechazar")}</button></div>`
          : detalle.estado === "admitida_a_convocatoria" ? `<div class="acciones-vista"><button type="button" class="boton-principal" data-inscripcion-decidir="incorporar">${et("incorporar")}</button></div>` : "";
      ficha = `<section class="panel" aria-labelledby="inscripcion-detalle-titulo"><header class="cabecera-panel"><h3 id="inscripcion-detalle-titulo" tabindex="-1">${et("detalle")}</h3><button type="button" class="boton-secundario" data-inscripcion-cerrar>${et("cerrar")}</button></header><div class="cuerpo-panel">
        <dl>${detalle.persona_resumen ? `<dt>${et("persona")}</dt><dd>${esc(detalle.persona_resumen)}</dd>` : ""}<dt>${et("convocatoria")}</dt><dd>${esc(detalle.categoria)}</dd><dt>${et("estado")}</dt><dd>${et(`estado_${detalle.estado}`)}</dd>${detalle.motivo_etiqueta ? `<dt>${et("motivo")}</dt><dd>${esc(detalle.motivo_etiqueta)}</dd>` : ""}<dt>${et("fecha")}</dt><dd>${fecha(detalle.registrada_en)}</dd><dt>${et("plazo")}</dt><dd>${fecha(detalle.plazo_inicio)} – ${fecha(detalle.plazo_fin)}</dd></dl>
        ${requisitos}${decision ? `<h4>${et("revision")}</h4>${selector}${evidencia}` : ""}${confirmar}</div></section>`;
    }
    const confirmacion = recibo ? `<section class="panel" id="inscripcion-recibo" tabindex="-1" role="status"><div class="cuerpo-panel"><p>${et(recibo.estado === "rechazada" ? "rechazada" : recibo.estado === "incorporada" ? "incorporada" : "admitida_a_convocatoria")}</p><p>${et("recibo")}: ${esc(recibo.recibo_ref)}</p><button type="button" class="boton-secundario" data-inscripcion-cerrar-recibo>${et("volver_lista")}</button></div></section>` : "";
    raiz.innerHTML = `<section class="panel" aria-labelledby="inscripciones-titulo"><header class="cabecera-panel"><div><h2 id="inscripciones-titulo" tabindex="-1">${et("titulo")}</h2></div><button type="button" class="boton-secundario" data-inscripcion-ayuda aria-expanded="${ayuda}" aria-controls="inscripcion-ayuda" aria-label="${et("ayuda_boton")}">?</button></header>
      <div class="cuerpo-panel"><div id="inscripcion-ayuda" ${ayuda ? "" : "hidden"}>${et("ayuda")}</div><form data-inscripcion-filtros><label for="inscripcion-estado">${et("estado")}</label><select id="inscripcion-estado" name="estado">${opcionesEstado}</select>
      <button class="boton-secundario" type="submit">${et("filtrar")}</button></form>${filtroActivo}
      ${estado}${cuenta}${listado?.total ? `<div class="tabla-contenedor" role="region" tabindex="0" aria-label="${et("lista")}"><table class="tabla-datos"><thead><tr><th scope="col">${et("persona")}</th><th scope="col">${et("convocatoria")}</th><th scope="col">${et("estado")}</th><th scope="col">${et("fecha")}</th></tr></thead><tbody>${filas}</tbody></table></div>` : ""}${paginacion}</div></section>${ficha}${confirmacion}`;
    if (enfocarRecibo && recibo) raiz.querySelector?.("#inscripcion-recibo")?.focus?.({ preventScroll: true });
  }
  function navegar(nuevo) {
    filtro = nuevo; detalle = null; motivos = null; decision = ""; recibo = null; intento = null;
    enfocarLista = true; enfocarRecibo = false;
    historial.pushState(null, "", rutaInscripcionesRRHH(localizacion.href, filtro));
    void cargarVista();
  }
  function cargarVista() {
    return filtro.convocatoria ? cargarLista() : cargarConvocatorias();
  }
  async function cargarConvocatorias() {
    controlador?.abort(); const orden = ++secuencia; controlador = new AbortController();
    convocatorias = null; listado = null; detalle = null; estadoVista = "cargando"; pintar();
    try {
      const data = await cliente.convocatorias({ cursor: filtro.selectorCursor,
        idioma: catalogo.idioma, signal: controlador.signal });
      if (!vivo || orden !== secuencia || controlador.signal.aborted) return;
      convocatorias = data; estadoVista = "lista"; pintar();
      if (enfocarLista) { enfocarLista = false; raiz.querySelector?.("#inscripciones-titulo")?.focus?.({ preventScroll: true }); }
    } catch (error) {
      if (!vivo || orden !== secuencia || controlador.signal.aborted) return;
      if ([401, 403].includes(error?.estado)) limpiarDenegacion();
      else { estadoVista = "error"; pintar(); }
    }
  }
  async function cargarLista() {
    if (!filtro.convocatoria) return cargarConvocatorias();
    controlador?.abort(); const orden = ++secuencia; controlador = new AbortController();
    listado = null; detalle = null; estadoVista = "cargando"; pintar();
    try {
      const data = await cliente.listar({ ...filtro, idioma: catalogo.idioma, signal: controlador.signal });
      if (!vivo || orden !== secuencia || controlador.signal.aborted) return;
      listado = data; estadoVista = "lista"; pintar();
      enfocarRecibo = false;
      if (enfocarLista) { enfocarLista = false; raiz.querySelector?.("#inscripciones-titulo")?.focus?.({ preventScroll: true }); }
    } catch (error) {
      if (!vivo || orden !== secuencia || controlador.signal.aborted) return;
      if ([401, 403].includes(error?.estado)) limpiarDenegacion();
      else { estadoVista = "error"; pintar(); }
    }
  }
  async function abrir(ref) {
    controlador?.abort(); const orden = ++secuencia; controlador = new AbortController();
    detalle = null; motivos = null; recibo = null; decision = ""; pintar();
    try {
      const data = await cliente.detalle(ref, { idioma: catalogo.idioma, signal: controlador.signal });
      if (!vivo || orden !== secuencia || controlador.signal.aborted) return;
      detalle = data; pintar(); raiz.querySelector?.("#inscripcion-detalle-titulo")?.focus?.({ preventScroll: true });
    } catch (error) {
      if (!vivo || orden !== secuencia || controlador.signal.aborted) return;
      if ([401, 403].includes(error?.estado)) limpiarDenegacion();
      else { estadoVista = "error"; pintar(); }
    }
  }
  async function prepararDecision(elegida) {
    if (!detalle || !(detalle.estado === "pendiente" && ["admitir", "rechazar"].includes(elegida)
      || detalle.estado === "admitida_a_convocatoria" && elegida === "incorporar")) return;
    if (elegida === "admitir" && detalle.requisitos?.some((r) => r.obligatorio && r.estado !== "cumple")) return;
    controlador?.abort(); const orden = ++secuencia; controlador = new AbortController();
    decision = ""; motivos = null;
    try {
      const data = elegida === "incorporar" ? null : await cliente.motivos(elegida,
        { idioma: catalogo.idioma, signal: controlador.signal });
      if (!vivo || orden !== secuencia || controlador.signal.aborted) return;
      motivos = data; decision = elegida; intento = null; pintar();
    } catch (error) {
      if (!vivo || orden !== secuencia || controlador.signal.aborted) return;
      if ([401, 403].includes(error?.estado)) limpiarDenegacion();
      else { estadoVista = "error"; pintar(); }
    }
  }
  async function confirmar() {
    if (!detalle || !decision || enviando || (decision === "rechazar" && !motivos?.motivos?.length)) return;
    const motivoCodigo = raiz.querySelector?.("[data-inscripcion-motivo]")?.value || "";
    const evidenciaRef = raiz.querySelector?.("[data-inscripcion-evidencia]")?.value?.trim() || "";
    if (decision === "incorporar" && !evidenciaRef) { raiz.querySelector?.("[data-inscripcion-evidencia]")?.reportValidity?.(); return; }
    if ((decision === "rechazar" || motivos?.motivos?.some((m) => m.obligatorio)) && !motivoCodigo) {
      raiz.querySelector?.("[data-inscripcion-motivo]")?.reportValidity?.(); return;
    }
    if (motivoCodigo && !motivos?.motivos?.some((m) => m.codigo === motivoCodigo)) return;
    if (!intento || intento.motivoCodigo !== motivoCodigo || intento.evidenciaRef !== evidenciaRef) intento = {
      solicitudRef: detalle.solicitud_ref, decision, motivoCodigo, evidenciaRef,
      versionEsperada: detalle.version, claveIdempotencia: claveNueva() };
    const orden = secuencia; enviando = true; pintar();
    try {
      const data = decision === "incorporar" ? await cliente.incorporar(intento) : await cliente.decidir(intento);
      if (!vivo || orden !== secuencia) return;
      recibo = data; detalle = null; decision = ""; motivos = null; estadoVista = "lista";
      enfocarRecibo = true;
      pintar(); void cargarLista();
    } catch (error) {
      if (!vivo || orden !== secuencia) return;
      if ([401, 403].includes(error?.estado)) limpiarDenegacion();
      else { estadoVista = error?.estado === 409 ? decision === "incorporar"
        ? error.codigo === "vinculo_identidad_pendiente" ? "identidad_pendiente"
          : error.codigo === "acta_pendiente" ? "falta_acta" : "conflicto"
        : "conflicto"
        : error?.estado === 422 ? error.codigo === "plazo_cerrado" ? "plazo_cerrado"
          : error.codigo === "requisito_invalido" ? "requisitos_pendientes"
            : decision === "incorporar" && error.codigo !== "catalogo_cambiado" ? "evidencia_invalida"
              : "regla_incompatible" : "error";
        if (["conflicto", "regla_incompatible", "plazo_cerrado", "requisitos_pendientes"].includes(estadoVista)) {
          detalle = null; decision = ""; motivos = null; intento = null;
        }
        pintar(); }
    } finally { enviando = false; if (vivo && orden === secuencia && !recibo) pintar(); }
  }
  const click = (evento) => {
    const accion = evento.target?.closest?.("[data-inscripcion-abrir], [data-inscripcion-decidir], [data-inscripcion-confirmar], [data-inscripcion-cancelar], [data-inscripcion-cerrar], [data-inscripcion-reintentar], [data-inscripcion-ayuda], [data-inscripcion-siguiente], [data-inscripcion-total], [data-inscripcion-cerrar-recibo], [data-inscripcion-convocatoria], [data-inscripcion-quitar-filtro], [data-inscripcion-elegir], [data-inscripcion-selector-siguiente], [data-inscripcion-selector-total]");
    if (!accion || !raiz.contains(accion)) return;
    if (accion.matches("a")) {
      if ((evento.button != null && evento.button !== 0) || evento.ctrlKey || evento.metaKey
        || evento.shiftKey || evento.altKey) return;
      evento.preventDefault();
    }
    if (accion.dataset.inscripcionAbrir) void abrir(accion.dataset.inscripcionAbrir);
    else if (accion.dataset.inscripcionDecidir) void prepararDecision(accion.dataset.inscripcionDecidir);
    else if (accion.hasAttribute("data-inscripcion-confirmar")) void confirmar();
    else if (accion.hasAttribute("data-inscripcion-cancelar")) { decision = ""; motivos = null; intento = null; pintar(); }
    else if (accion.hasAttribute("data-inscripcion-cerrar")) { const ref = detalle?.solicitud_ref;
      ++secuencia; controlador?.abort(); detalle = null; decision = ""; motivos = null; pintar();
      [...(raiz.querySelectorAll?.("[data-inscripcion-abrir]") || [])]
        .find((control) => control.dataset.inscripcionAbrir === ref)?.focus?.({ preventScroll: true }); }
    else if (accion.hasAttribute("data-inscripcion-reintentar")) void cargarVista();
    else if (accion.hasAttribute("data-inscripcion-ayuda")) { ayuda = !ayuda; pintar(); raiz.querySelector?.("[data-inscripcion-ayuda]")?.focus?.(); }
    else if (accion.hasAttribute("data-inscripcion-siguiente")) navegar({ ...filtro, cursor: listado.cursor_siguiente });
    else if (accion.hasAttribute("data-inscripcion-total")) navegar({ ...filtro, cursor: "" });
    else if (accion.dataset.inscripcionElegir) {
      seleccion = convocatorias?.convocatorias.find((c) => c.convocatoria_ref === accion.dataset.inscripcionElegir) || null;
      navegar({ ...filtro, convocatoria: accion.dataset.inscripcionElegir, cursor: "" });
    }
    else if (accion.hasAttribute("data-inscripcion-selector-siguiente")) navegar({ ...filtro, selectorCursor: convocatorias.cursor_siguiente });
    else if (accion.hasAttribute("data-inscripcion-selector-total")) navegar({ ...filtro, selectorCursor: "" });
    else if (accion.dataset.inscripcionConvocatoria) navegar({ ...filtro, convocatoria: accion.dataset.inscripcionConvocatoria, cursor: "" });
    else if (accion.hasAttribute("data-inscripcion-quitar-filtro")) navegar({ ...filtro, convocatoria: "", cursor: "" });
    else if (accion.hasAttribute("data-inscripcion-cerrar-recibo")) { recibo = null; enfocarRecibo = false; pintar();
      raiz.querySelector?.("#inscripciones-titulo")?.focus?.({ preventScroll: true }); }
  };
  const submit = (evento) => {
    if (!evento.target?.matches?.("[data-inscripcion-filtros]")) return;
    evento.preventDefault(); const form = new FormData(evento.target);
    navegar({ ...filtro, estado: form.get("estado"), cursor: "" });
  };
  const pop = () => { filtro = leerRutaInscripcionesRRHH(localizacion.search);
    if (seleccion?.convocatoria_ref !== filtro.convocatoria) seleccion = null;
    enfocarLista = true; void cargarVista(); };
  const desmontar = () => {
    if (!vivo) return;
    vivo = false; ++secuencia; controlador?.abort();
    raiz.removeEventListener("click", click); raiz.removeEventListener("submit", submit);
    globalThis.removeEventListener?.("popstate", pop); signal?.removeEventListener?.("abort", desmontar);
    raiz.replaceChildren();
  };
  const montaje = Object.freeze({ desmontar });
  raiz.addEventListener("click", click); raiz.addEventListener("submit", submit);
  globalThis.addEventListener?.("popstate", pop);
  if (signal?.aborted) desmontar();
  else signal?.addEventListener?.("abort", desmontar, { once: true });
  try {
    try { catalogo = await cargarCatalogo("bolsa-inscripcion-rrhh"); }
    catch { catalogo = await reintentarCatalogo("bolsa-inscripcion-rrhh"); }
  } catch (error) { if (!vivo) return montaje; throw error; }
  if (vivo) void cargarVista();
  return montaje;
}
