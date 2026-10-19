import { cargarTextos, reintentarTextos } from "./textos.js";
import { crearClienteAlegaciones, objetoAlegacion } from "./alegaciones-cliente.js?v=VEC_WEB_VERSION";

const REF = /^[A-Za-z0-9][A-Za-z0-9:._-]{2,255}$/u;

/** Ambas superficies usan los mismos recibos; RRHH nunca monta el formulario. */
export async function montarAlegaciones({ contenedor, modo = "propias", escaparHTML,
  cliente = crearClienteAlegaciones(), signal, localizacion = globalThis.location,
  historial = globalThis.history, pendiente = null,
  cargarCatalogo = cargarTextos, reintentarCatalogo = reintentarTextos } = {}) {
  if (!contenedor?.addEventListener || typeof escaparHTML !== "function" || !["propias", "rrhh"].includes(modo)) {
    throw new TypeError("alegaciones_superficie_invalida");
  }
  const rrhh = modo === "rrhh";
  const query = new URLSearchParams(localizacion?.search || "");
  let ambito = rrhh && ["convocatoria", "bolsa"].includes(query.get("alega_tipo")) && REF.test(query.get("alega_ref") || "")
    ? { tipo: query.get("alega_tipo"), referencia: query.get("alega_ref"), titulo: "" } : null;
  let vista = rrhh ? (ambito ? "lista" : "ambitos") : pendiente ? "formulario" : "lista";
  let catalogo, vivo = true, secuencia = 0, lectura = null, enviando = false, ayuda = false;
  let pagina = null, estado = "cargando", seleccion = null, ficha = null, recibo = null;
  let fundamento = pendiente?.orden?.fundamento || "";
  let intencion = rrhh ? null : pendiente;
  let errorFormulario = intencion ? "respuesta_incierta" : "";
  const doc = contenedor.ownerDocument;
  const estilo = doc.createElement("link"); estilo.rel = "stylesheet";
  estilo.href = "/comun/alegaciones.css?v=VEC_WEB_VERSION"; doc.head.append(estilo);
  const esc = escaparHTML;
  const t = (clave, vars = {}) => catalogo.traducir(`alegaciones.${clave}`, vars);
  const et = (clave, vars) => esc(t(clave, vars));
  const fecha = (v) => esc(catalogo.fecha(v, { dateStyle: "short", timeStyle: "short", timeZone: "Europe/Madrid" }));
  const boton = (clave, accion, clase = "boton-secundario") => `<button type="button" class="${clase}" data-alegacion-accion="${accion}" ${estado === "cargando" || enviando ? "disabled" : ""}>${et(clave)}</button>`;

  function ruta(cambios = {}) {
    const url = new URL(localizacion.href);
    for (const [clave, valor] of Object.entries(cambios)) {
      if (!valor) url.searchParams.delete(clave); else url.searchParams.set(clave, valor);
    }
    return url.pathname + url.search + url.hash;
  }
  function conservarRuta(cambios) { historial?.pushState?.(null, "", ruta(cambios)); }
  function tituloSeleccion(titulo, tipo) {
    return ["bolsa", "acto_participacion"].includes(tipo) && /^\d{4}-\d{2}-\d{2}[T ]/u.test(titulo || "") && Number.isFinite(Date.parse(titulo))
      ? t("bolsa_fecha", { fecha: catalogo.fecha(titulo, { dateStyle: "short", timeStyle: "short", timeZone: "Europe/Madrid" }) }) : titulo;
  }
  function textoObjeto(o) { return `${et(o.tipo)} · ${et("version")} ${esc(catalogo.numero(o.version_acto))}`; }
  function pintarRecibo(a) {
    return `<section class="alegaciones-ficha" aria-labelledby="alegaciones-ficha-titulo"><h3 id="alegaciones-ficha-titulo" tabindex="-1">${et("recibo_titulo")}</h3>
      <span class="estado-chip exito">${et("presentada")}</span><dl><dt>${et("objeto")}</dt><dd>${textoObjeto(a.objeto)}</dd>
      <dt>${et("fecha")}</dt><dd><time datetime="${esc(a.presentada_en)}">${fecha(a.presentada_en)}</time></dd>
      <dt>${et("referencia")}</dt><dd class="alegaciones-referencia">${esc(a.recibo_ref)}</dd></dl>
      ${rrhh ? `<p>${et("persona")}: ${a.persona_resumen ? esc(a.persona_resumen) : et("nombre_no_disponible")}</p>` : ""}
      <h4>${et("fundamento")}</h4><p class="alegaciones-fundamento">${esc(a.fundamento)}</p>${boton("cerrar_recibo", "cerrar-ficha")}</section>`;
  }
  function pintar() {
    if (!vivo || !catalogo) return;
    let contenido = "";
    const ocupado = estado === "cargando";
    const error = !["listo", "vacio", "cargando"].includes(estado);
    const aviso = ocupado ? `<p role="status" aria-busy="true">${et(vista === "formulario" ? "cargando_objetos" : "cargando")}</p>`
      : error ? `<p role="alert">${et(estado)}</p>${boton("reintentar", "reintentar")}` : "";
    if (vista === "formulario") {
      const objetos = pagina?.objetos || [];
      const opciones = objetos.map((o, indice) => `<option value="${indice}" ${seleccion === o ? "selected" : ""}>${esc(tituloSeleccion(o.titulo, o.tipo))} · ${textoObjeto(o)}</option>`).join("");
      const bloqueada = enviando || !!intencion;
      const elegido = intencion?.orden?.objeto;
      contenido = `${boton("volver", "lista")}${recibo ? pintarRecibo(recibo) : `<form data-alegacion-formulario novalidate>
        ${elegido ? `<p>${esc(tituloSeleccion(intencion.titulo || "", elegido.tipo))} · ${textoObjeto(elegido)}</p>` : `<div class="campo"><label for="alegacion-objeto">${et("elegir_objeto")}</label><select id="alegacion-objeto" ${ocupado || bloqueada ? "disabled" : ""}><option value="">${et("elegir_objeto")}</option>${opciones}</select></div>`}
        ${estado === "vacio" ? `<p role="status">${et("objetos_vacios")}</p>` : ""}
        <div class="campo"><label for="alegacion-fundamento">${et("fundamento")}</label><textarea id="alegacion-fundamento" rows="6" maxlength="4000" required ${bloqueada || ocupado ? "disabled" : ""} ${errorFormulario ? 'aria-invalid="true" aria-describedby="alegacion-error"' : ""}>${esc(fundamento)}</textarea></div>
        ${errorFormulario ? `<p role="alert" id="alegacion-error">${et(errorFormulario)}</p>` : ""}
        <button type="submit" class="boton-primario" ${enviando || ocupado || !seleccion && !intencion ? "disabled" : ""}>${et(enviando ? "presentando" : intencion ? "reintentar" : "presentar")}</button>
        </form>`}`;
    } else if (vista === "ambitos") {
      contenido = `<div class="alegaciones-lista">${(pagina?.ambitos || []).map((a, i) => `<button type="button" class="tarjeta-modulo tarjeta-modulo-habilitada" data-alegacion-ambito="${i}"><strong>${esc(tituloSeleccion(a.titulo, a.tipo))}</strong><span>${et(a.tipo === "convocatoria" ? "convocatoria" : "bolsa")}</span></button>`).join("")}</div>`;
    } else {
      contenido = `${rrhh ? boton("volver", "ambitos") : boton("nueva", "formulario", "boton-primario")}
        ${ambito?.titulo ? `<h3>${esc(tituloSeleccion(ambito.titulo, ambito.tipo))}</h3>` : ""}
        <div class="alegaciones-lista">${(pagina?.alegaciones || []).map((a, i) => `<article class="alegaciones-tarjeta"><div><span class="estado-chip exito">${et("presentada")}</span><h3>${a.persona_resumen ? esc(a.persona_resumen) : et(rrhh ? "nombre_no_disponible" : a.objeto.tipo)}</h3><p>${textoObjeto(a.objeto)}</p><time datetime="${esc(a.presentada_en)}">${fecha(a.presentada_en)}</time></div><button type="button" class="boton-secundario" data-alegacion-abrir="${i}" ${ocupado ? "disabled" : ""}>${et("ver_alegacion")}</button></article>`).join("")}</div>${ficha ? pintarRecibo(ficha) : ""}`;
    }
    const totalClave = vista === "ambitos" ? "total_ambitos" : vista === "formulario" ? "total_objetos" : "total";
    const total = pagina && !recibo && !intencion ? `<p><a href="${esc(ruta({ alega_cursor: "" }))}" data-alegacion-accion="total">${esc(catalogo.plural(`alegaciones.${totalClave}`, pagina.total))}</a></p>` : "";
    const mas = pagina?.cursor_siguiente && !recibo && !intencion ? `<nav aria-label="${et("paginacion")}">${boton("siguiente", "siguiente")}</nav>` : "";
    const vacio = estado === "vacio" && vista !== "formulario" ? `<p role="status">${et("vacio")}</p>` : "";
    contenedor.innerHTML = `<section class="panel alegaciones-superficie" aria-labelledby="alegaciones-titulo"><header class="cabecera-panel"><h2 id="alegaciones-titulo" tabindex="-1">${et(rrhh ? "titulo" : "propias_titulo")}</h2><button type="button" class="boton-secundario" data-alegacion-accion="ayuda" aria-label="${et("ayuda_boton")}" aria-expanded="${ayuda}" aria-controls="alegaciones-ayuda">?</button></header><div class="cuerpo-panel panel-contenido alegaciones-cuerpo"><div id="alegaciones-ayuda" ${ayuda ? "" : "hidden"}>${et(rrhh ? "ayuda_rrhh" : "ayuda_propia")}</div>${aviso}${vacio}${total}${contenido}${mas}</div></section>`;
  }
  function errorLectura(e) {
    return e?.codigo === "sin_permiso" ? "denegada" : e?.codigo === "no_encontrado" ? "no_disponible" : "error";
  }
  async function cargar(cursor = "") {
    if (!vivo || enviando) return;
    const numero = ++secuencia; lectura?.abort(); lectura = new AbortController();
    estado = "cargando"; pagina = null; ficha = null; seleccion = null; pintar();
    try {
      const opciones = { cursor, signal: lectura.signal };
      const p = vista === "ambitos" ? await cliente.ambitos(opciones)
        : vista === "formulario" ? await cliente.objetos(opciones)
          : rrhh ? await cliente.rrhh({ ...opciones, ...ambito }) : await cliente.propias(opciones);
      if (!vivo || numero !== secuencia) return;
      pagina = p; const elementos = p.alegaciones || p.ambitos || p.objetos;
      estado = elementos.length ? "listo" : "vacio"; pintar();
    } catch (e) {
      if (!vivo || numero !== secuencia || e?.codigo === "cancelado") return;
      pagina = null; ficha = null; estado = errorLectura(e); pintar();
    }
  }
  async function abrir(indice) {
    const a = pagina?.alegaciones?.[indice]; if (!a || estado === "cargando") return;
    if (!rrhh) { ficha = a; pintar(); contenedor.querySelector("#alegaciones-ficha-titulo")?.focus(); return; }
    const numero = ++secuencia; lectura?.abort(); lectura = new AbortController(); estado = "cargando"; ficha = null; pintar();
    try {
      const d = await cliente.detalle({ alegacionRef: a.alegacion_ref, signal: lectura.signal });
      if (!vivo || numero !== secuencia) return;
      ficha = d; estado = "listo"; pintar(); contenedor.querySelector("#alegaciones-ficha-titulo")?.focus();
    } catch (e) {
      if (!vivo || numero !== secuencia || e?.codigo === "cancelado") return;
      pagina = null; ficha = null; estado = errorLectura(e); pintar();
    }
  }
  async function presentar(evento) {
    if (!evento.target.matches?.("[data-alegacion-formulario]")) return;
    evento.preventDefault(); if (rrhh || enviando || !vivo) return;
    if (!intencion) {
      if (!seleccion) return;
      if (!fundamento.trim()) { errorFormulario = "fundamento_falta"; pintar(); contenedor.querySelector("#alegacion-fundamento")?.focus(); return; }
      if ([...fundamento].length > 4000) { errorFormulario = "fundamento_largo"; pintar(); return; }
      const clave = globalThis.crypto?.randomUUID?.();
      if (!clave) { errorFormulario = "error_presentar"; pintar(); return; }
      intencion = { titulo: seleccion.titulo, orden: { objeto: objetoAlegacion(seleccion), fundamento, clave_idempotencia: clave } };
    }
    const orden = intencion.orden; enviando = true; errorFormulario = ""; pintar();
    try {
      const r = await cliente.presentar(orden);
      if (!vivo) return;
      recibo = r; intencion = null; fundamento = ""; errorFormulario = "";
    } catch (e) {
      if (!vivo) return;
      if ([400, 409, 422].includes(e?.estado)) {
        intencion = null; seleccion = null; pagina = null;
        errorFormulario = e.estado === 409 ? "objeto_cambiado" : "datos_invalidos";
        estado = "error_objetos";
      } else errorFormulario = e?.codigo === "sin_permiso" ? "denegada" : "respuesta_incierta";
    } finally {
      enviando = false;
      if (vivo) { pintar(); if (recibo) contenedor.querySelector("#alegaciones-ficha-titulo")?.focus(); }
    }
  }
  const click = (evento) => {
    const control = evento.target.closest?.("[data-alegacion-accion], [data-alegacion-ambito], [data-alegacion-abrir]");
    if (!control || !contenedor.contains(control) || control.disabled) return;
    evento.preventDefault();
    if (control.dataset.alegacionAbrir !== undefined) { void abrir(Number(control.dataset.alegacionAbrir)); return; }
    if (control.dataset.alegacionAmbito !== undefined) {
      ambito = pagina?.ambitos?.[Number(control.dataset.alegacionAmbito)]; if (!ambito) return;
      vista = "lista"; conservarRuta({ alega_tipo: ambito.tipo, alega_ref: ambito.referencia, alega_cursor: "" }); void cargar(); return;
    }
    const accion = control.dataset.alegacionAccion;
    if (accion === "ayuda") { ayuda = !ayuda; pintar(); contenedor.querySelector('[data-alegacion-accion="ayuda"]')?.focus(); }
    else if (accion === "cerrar-ficha") { ficha = null; recibo = null; if (vista === "formulario") { vista = "lista"; void cargar(); } else pintar(); }
    else if (accion === "siguiente") { const cursor = pagina?.cursor_siguiente; if (cursor) { conservarRuta({ alega_cursor: cursor }); void cargar(cursor); } }
    else if (accion === "reintentar" || accion === "total") { conservarRuta({ alega_cursor: "" }); void cargar(); }
    else if (["lista", "formulario", "ambitos"].includes(accion) && !enviando) {
      vista = accion; recibo = null; if (accion === "ambitos") ambito = null;
      conservarRuta({ alega_cursor: "", ...(accion === "ambitos" ? { alega_tipo: "", alega_ref: "" } : {}) });
      if (vista === "formulario" && intencion) { estado = "listo"; pintar(); } else void cargar();
    }
  };
  const entrada = (evento) => {
    if (evento.target.id === "alegacion-fundamento" && !intencion) fundamento = evento.target.value;
    if (evento.type === "change" && evento.target.id === "alegacion-objeto" && !intencion) {
      seleccion = evento.target.value === "" ? null : pagina?.objetos?.[Number(evento.target.value)];
      errorFormulario = ""; pintar(); contenedor.querySelector("#alegacion-objeto")?.focus();
    }
  };
  const pop = () => {
    if (enviando) return;
    const q = new URLSearchParams(localizacion.search);
    if (rrhh) {
      ambito = ["convocatoria", "bolsa"].includes(q.get("alega_tipo")) && REF.test(q.get("alega_ref") || "")
        ? { tipo: q.get("alega_tipo"), referencia: q.get("alega_ref"), titulo: "" } : null;
      vista = ambito ? "lista" : "ambitos";
    }
    const cursor = q.get("alega_cursor") || ""; void cargar(REF.test(cursor) ? cursor : "");
  };
  function desmontar() {
    if (!vivo) return; vivo = false; ++secuencia; lectura?.abort(); estilo.remove();
    contenedor.removeEventListener("click", click); contenedor.removeEventListener("input", entrada);
    contenedor.removeEventListener("change", entrada); contenedor.removeEventListener("submit", presentar);
    globalThis.removeEventListener?.("popstate", pop); signal?.removeEventListener?.("abort", desmontar);
    contenedor.replaceChildren();
  }
  const montaje = Object.freeze({ desmontar, pendiente: () => intencion });
  if (signal?.aborted) { desmontar(); return montaje; }
  signal?.addEventListener?.("abort", desmontar, { once: true });
  try { catalogo = await cargarCatalogo("bolsa-alegaciones"); }
  catch (e) { if (!vivo) return montaje; try { catalogo = await reintentarCatalogo("bolsa-alegaciones"); } catch (f) { desmontar(); throw f; } }
  if (!vivo) return montaje;
  contenedor.addEventListener("click", click); contenedor.addEventListener("input", entrada);
  contenedor.addEventListener("change", entrada); contenedor.addEventListener("submit", presentar);
  globalThis.addEventListener?.("popstate", pop);
  if (intencion) { estado = "listo"; pintar(); }
  else { const cursor = query.get("alega_cursor") || ""; void cargar(REF.test(cursor) ? cursor : ""); }
  return montaje;
}
