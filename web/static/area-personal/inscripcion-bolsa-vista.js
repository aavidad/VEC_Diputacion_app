import { cargarTextos, reintentarTextos } from "../comun/textos.js";
import { crearClienteInscripcionBolsa } from "./inscripcion-bolsa-api.js";

const CLAVES_PENDIENTES = new Map(); // Sólo memoria de esta pestaña, para repetir el mismo acto incierto.
const ENVIOS_ACTIVOS = new Set();
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9:._-]{0,199}$/u;
const esc = (valor) => String(valor ?? "").replaceAll("&", "&amp;")
  .replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");

function leerId(url) {
  const valor = new URL(url).searchParams.get("id") ?? "";
  if (valor === "mis-solicitudes") return { tipo: "propias" };
  if (valor.startsWith("convocatoria:") && REFERENCIA.test(valor.slice(13))) return { tipo: "bolsa", ref: valor.slice(13) };
  if (valor.startsWith("solicitud:") && REFERENCIA.test(valor.slice(10))) return { tipo: "solicitud", ref: valor.slice(10) };
  return null;
}

function urlId(url, id) {
  const destino = new URL(url);
  if (id) destino.searchParams.set("id", id);
  else destino.searchParams.delete("id");
  return `${destino.pathname}${destino.search}`;
}

function descripcionError(error, textos) {
  if (error?.status === 401 || error?.status === 403) CLAVES_PENDIENTES.clear();
  if (error?.status === 401) return textos.traducir("vista.sinSesion");
  if (error?.status === 403) return textos.traducir("vista.denegada");
  if (error?.status === 404) return textos.traducir("vista.noEncontrada");
  if (error?.status === 400) return textos.traducir("vista.datosNoAceptados");
  if (error?.status === 409) return textos.traducir("vista.conflicto");
  if (error?.status === 422 && error?.codigo === "plazo_cerrado") return textos.traducir("vista.plazoCerrado");
  if (error?.status === 422) return textos.traducir("vista.requisitosCambiados");
  return textos.traducir("vista.error");
}

function cabecera(textos, ayuda) {
  const t = (clave) => esc(textos.traducir(`vista.${clave}`));
  return `<header class="cabecera-panel"><div><h2 id="titulo-inscripcion-bolsa" tabindex="-1">${t("titulo")}</h2></div>
    <button type="button" class="boton-secundario" data-inscripcion-accion="ayuda"
      aria-expanded="${ayuda}" aria-controls="ayuda-inscripcion-bolsa" aria-label="${t("ayudaEtiqueta")}">?</button></header>
    <div class="cuerpo-panel" id="ayuda-inscripcion-bolsa" ${ayuda ? "" : "hidden"}><p>${t("ayuda")}</p></div>`;
}

function fecha(textos, valor) {
  return esc(textos.fecha(valor, { dateStyle: "medium", timeZone: "Europe/Madrid" }));
}
function fechaHora(textos, valor) {
  return esc(textos.fecha(valor, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }));
}

function tarjetaAbierta(bolsa, textos) {
  const t = (clave) => esc(textos.traducir(`vista.${clave}`));
  const propia = bolsa.estado_solicitud_propia;
  const enlace = propia && bolsa.solicitud_ref
    ? `<button type="button" class="boton-secundario" data-inscripcion-accion="solicitud" data-ref="${esc(bolsa.solicitud_ref)}">${t("verSolicitud")}</button>`
    : `<button type="button" class="boton-primario" data-inscripcion-accion="bolsa" data-ref="${esc(bolsa.convocatoria_ref)}">${t("verBolsa")}</button>`;
  return `<article class="panel portal-mi-bolsa__bolsa"><div class="cabecera-panel"><h3>${esc(bolsa.titulo)}</h3>
    ${propia ? `<span class="estado-chip info">${t(`estado_${propia}`)}</span>` : ""}</div>
    <div class="cuerpo-panel"><dl class="lista-datos"><div><dt>${t("plazo")}</dt><dd>${fechaHora(textos, bolsa.plazo_inicio)} – ${fechaHora(textos, bolsa.plazo_fin)}</dd></div>
    <div><dt>${t("categoria")}</dt><dd>${esc(bolsa.categorias_resumen)}</dd></div>
    <div><dt>${t("requisitos")}</dt><dd>${esc(bolsa.requisitos_resumen)}</dd></div></dl>
    <div class="acciones-vista">${enlace}</div></div></article>`;
}

function vistaAbiertas(estado, textos) {
  const t = (clave, variables) => esc(textos.traducir(`vista.${clave}`, variables));
  if (estado.carga) return `<div class="cuerpo-panel" role="status" aria-live="polite">${t("cargando")}</div>`;
  if (estado.error) return `<div class="cuerpo-panel" role="alert"><p>${esc(estado.error)}</p>
    <button type="button" class="boton-secundario" data-inscripcion-accion="reintentar">${t("reintentar")}</button></div>`;
  if (!estado.abiertas.length) return `<div class="cuerpo-panel vacio-controlado" role="status"><p>${t("vacio")}</p>
    <button type="button" class="boton-secundario" data-inscripcion-accion="propias">${t("misSolicitudes")}</button>
    <button type="button" class="boton-secundario" data-inscripcion-accion="reintentar">${t("actualizar")}</button></div>`;
  return `<div class="cuerpo-panel"><div class="acciones-vista"><button type="button" class="boton-secundario"
      data-inscripcion-accion="propias">${t("misSolicitudes")}</button></div>
    <div class="marco-participaciones">${estado.abiertas.map((bolsa) => tarjetaAbierta(bolsa, textos)).join("")}</div>
    ${estado.cursor ? `<div class="acciones-vista"><button type="button" class="boton-secundario" data-inscripcion-accion="mas">${t("mostrarMas")}</button></div>` : ""}</div>`;
}

function vistaBolsa(estado, textos) {
  const t = (clave) => esc(textos.traducir(`vista.${clave}`));
  const bolsa = estado.bolsa;
  if (!bolsa) return "";
  const pendiente = CLAVES_PENDIENTES.has(bolsa.convocatoria_ref);
  const categoria = bolsa.categorias.find((c) => c.categoria_ref === estado.categoriaRef);
  const requisitos = bolsa.requisitos.map((requisito) => `<li>${esc(requisito.descripcion)}${requisito.obligatorio
    ? ` <span class="estado-chip aviso">${t("obligatorio")}</span>` : ""}
    <span class="estado-chip ${requisito.estado === "cumple" ? "exito" : requisito.estado === "no_cumple" ? "peligro" : "aviso"}">${t(`requisito_${requisito.estado}`)}</span>
    <p>${esc(requisito.motivo_etiqueta)}</p>
    ${requisito.hito_etiqueta ? `<p>${t("cuandoExigido")}: ${esc(requisito.hito_etiqueta)}${requisito.hito_fecha ? ` · ${fecha(textos, requisito.hito_fecha)}` : ""}</p>` : ""}
    <label class="opcion-check"><input type="checkbox" data-inscripcion-requisito="${esc(requisito.codigo)}"
      ${estado.declaraciones.has(requisito.codigo) ? "checked" : ""} ${pendiente || requisito.estado === "no_cumple" ? "disabled" : ""}><span>${t("declarar")}: ${esc(requisito.descripcion)}</span></label></li>`).join("");
  const declarados = bolsa.requisitos.filter((requisito) => estado.declaraciones.has(requisito.codigo));
  const enviando = estado.enviando || ENVIOS_ACTIVOS.has(bolsa.convocatoria_ref);
  return `<div class="cuerpo-panel"><button type="button" class="boton-secundario" data-inscripcion-accion="volver">${t("volver")}</button>
    <h3 tabindex="-1">${esc(bolsa.titulo)}</h3><dl class="lista-datos"><div><dt>${t("plazo")}</dt>
      <dd>${fechaHora(textos, bolsa.plazo_inicio)} – ${fechaHora(textos, bolsa.plazo_fin)}</dd></div></dl>
    <div class="campo"><label for="categoria-inscripcion-bolsa">${t("categoria")}</label>
      <select id="categoria-inscripcion-bolsa" data-inscripcion-categoria ${pendiente || bolsa.categorias.length === 1 ? "disabled" : ""}>
      ${bolsa.categorias.length > 1 ? `<option value="">${t("elegirCategoria")}</option>` : ""}
      ${bolsa.categorias.map((c) => `<option value="${esc(c.categoria_ref)}" ${c.categoria_ref === estado.categoriaRef ? "selected" : ""}>${esc(c.categoria)}</option>`).join("")}
      </select></div>
    <section class="panel"><div class="cabecera-panel"><h4>${t("requisitos")}</h4></div><div class="cuerpo-panel">${requisitos ? `<ul>${requisitos}</ul>` : `<p>${t("sinRequisitos")}</p>`}</div></section>
    ${estado.revision ? `<section class="panel"><div class="cabecera-panel"><h4 tabindex="-1" data-inscripcion-revision>${t("revisarTitulo")}</h4></div><div class="cuerpo-panel">
      <dl class="lista-datos"><div><dt>${t("convocatoria")}</dt><dd>${esc(bolsa.titulo)}</dd></div>
      <div><dt>${t("categoria")}</dt><dd>${esc(categoria?.categoria)}</dd></div>
      <div><dt>${t("plazo")}</dt><dd>${fechaHora(textos, bolsa.plazo_inicio)} – ${fechaHora(textos, bolsa.plazo_fin)}</dd></div></dl>
      <p>${t(pendiente ? "revisarPendiente" : "revisarActo")}</p><p>${t("declaracionesRevisar")}</p>
      ${declarados.length ? `<ul>${declarados.map((requisito) => `<li>${esc(requisito.descripcion)} · ${t(`requisito_${requisito.estado}`)}</li>`).join("")}</ul>` : `<p>${t("sinDeclaraciones")}</p>`}
      <div class="acciones-vista"><button type="button" class="boton-secundario" data-inscripcion-accion="corregir" ${enviando ? "disabled" : ""}>${t("corregir")}</button>
      <button type="button" class="boton-primario" data-inscripcion-accion="confirmar" ${enviando || estado.bloqueoActo ? "disabled" : ""}>${t(enviando ? "enviando" : "confirmar")}</button></div></div></section>`
      : `<div class="acciones-vista">${bolsa.puede_iniciar ? "" : `<p role="status">${esc(bolsa.impedimento_etiqueta)}</p>`}
        <button type="button" class="boton-primario" data-inscripcion-accion="revisar" ${enviando || estado.comprobarEnvio || estado.bloqueoActo || !bolsa.puede_iniciar || !categoria ? "disabled" : ""}>${t("solicitar")}</button></div>`}
    ${estado.error ? `<p role="alert">${esc(estado.error)}</p>${estado.bloqueoActo
      ? `<button type="button" class="boton-secundario" data-inscripcion-accion="actualizar-ficha">${t("actualizarFicha")}</button>` : ""}
      <button type="button" class="boton-secundario" data-inscripcion-accion="propias">${t("misSolicitudes")}</button>` : ""}</div>`;
}

function vistaSolicitud(estado, textos) {
  const t = (clave) => esc(textos.traducir(`vista.${clave}`));
  const solicitud = estado.solicitud;
  if (!solicitud) return "";
  return `<div class="cuerpo-panel"><button type="button" class="boton-secundario" data-inscripcion-accion="volver">${t("volver")}</button>
    <h3 tabindex="-1">${t("solicitudTitulo")}: ${esc(solicitud.categoria)}</h3><dl class="lista-datos">
    <div><dt>${t("estado")}</dt><dd><span class="estado-chip ${solicitud.estado === "rechazada" ? "peligro" : solicitud.estado === "incorporada" ? "exito" : "aviso"}">${t(`estado_${solicitud.estado}`)}</span></dd></div>
    <div><dt>${t("fechaRegistro")}</dt><dd>${fecha(textos, solicitud.registrada_en)}</dd></div>
    <div><dt>${t("recibo")}</dt><dd>${esc(solicitud.recibo_ref)}</dd></div>
    ${solicitud.estado === "rechazada" ? `<div><dt>${t("motivo")}</dt><dd>${esc(solicitud.motivo_etiqueta)}</dd></div>` : ""}</dl>
    <p>${t(`paso_${solicitud.estado}`)}</p></div>`;
}

function vistaPropias(estado, textos) {
  const t = (clave) => esc(textos.traducir(`vista.${clave}`));
  if (estado.error) return `<div class="cuerpo-panel"><button type="button" class="boton-secundario" data-inscripcion-accion="volver">${t("volver")}</button></div>`;
  if (!estado.propias.length) return `<div class="cuerpo-panel"><button type="button" class="boton-secundario" data-inscripcion-accion="volver">${t("volver")}</button>
    <p role="status">${t("sinSolicitudes")}</p></div>`;
  return `<div class="cuerpo-panel"><button type="button" class="boton-secundario" data-inscripcion-accion="volver">${t("volver")}</button>
    <h3>${t("misSolicitudes")}</h3><button type="button" class="boton-secundario" data-inscripcion-accion="actualizar-propias">${t("actualizarSolicitudes")}</button>
    <div class="marco-participaciones">${estado.propias.map((solicitud) =>
    `<article class="panel portal-mi-bolsa__bolsa"><div class="cabecera-panel"><h4>${esc(solicitud.categoria)}</h4>
      <span class="estado-chip ${solicitud.estado === "rechazada" ? "peligro" : solicitud.estado === "incorporada" ? "exito" : "aviso"}">${t(`estado_${solicitud.estado}`)}</span></div><div class="cuerpo-panel">
      <p>${fecha(textos, solicitud.registrada_en)}</p><button type="button" class="boton-secundario" data-inscripcion-accion="solicitud" data-ref="${esc(solicitud.solicitud_ref)}">${t("verSolicitud")}</button>
      </div></article>`).join("")}</div>${estado.cursorPropias ? `<button type="button" class="boton-secundario" data-inscripcion-accion="mas-propias">${t("mostrarMas")}</button>` : ""}</div>`;
}

export function montarInscripcionBolsa({ contenedor, fetchImpl = globalThis.fetch,
  cliente = null, idioma,
  ventana = globalThis.window, anunciar = () => {}, textoBase = () => "" } = {}) {
  if (!contenedor?.addEventListener || !contenedor?.removeEventListener || !ventana?.location
    || !ventana?.history || (cliente !== null && (!cliente?.abiertas || !cliente?.convocatoria
      || !cliente?.propias || !cliente?.detallePropio || !cliente?.inscribir)))
    throw new TypeError("Montaje de inscripción inválido");
  const estado = { tipo: "abiertas", carga: true, abiertas: [], propias: [], total: 0,
    cursor: null, cursorPropias: null, bolsa: null, solicitud: null, error: "", revision: false,
    enviando: false, comprobarEnvio: false, bloqueoActo: false,
    ayuda: false, declaraciones: new Set(), categoriaRef: "" };
  let textos = null;
  let clienteEfectivo = cliente;
  let montado = true;
  let consulta = null;
  let secuencia = 0;
  const t = (clave) => textos.traducir(`vista.${clave}`);

  function limpiarDatosPrivados() {
    estado.abiertas = []; estado.propias = []; estado.bolsa = null; estado.solicitud = null;
    estado.cursor = null; estado.cursorPropias = null; estado.declaraciones.clear();
    estado.categoriaRef = ""; estado.revision = false; estado.total = 0;
  }

  function cerrarPorDenegacion(error) {
    if (!montado || (error?.status !== 401 && error?.status !== 403)) return false;
    ++secuencia;
    consulta?.abort();
    limpiarDatosPrivados();
    estado.carga = false;
    estado.enviando = false;
    estado.error = descripcionError(error, textos);
    pintar(); anunciar(estado.error);
    return true;
  }

  function pintar() {
    if (!montado) return;
    if (!textos) {
      contenedor.innerHTML = `<section class="panel" role="status" aria-live="polite"><div class="cuerpo-panel">${esc(textoBase("cargando"))}</div></section>`;
      return;
    }
    const contenido = estado.tipo === "abiertas" ? vistaAbiertas(estado, textos)
      : estado.carga ? `<div class="cuerpo-panel" role="status">${esc(t("cargando"))}</div>`
        : estado.tipo === "bolsa" ? vistaBolsa(estado, textos)
          : estado.tipo === "solicitud" ? vistaSolicitud(estado, textos) : vistaPropias(estado, textos);
    const error = estado.tipo !== "abiertas" && estado.error && (estado.tipo !== "bolsa" || !estado.bolsa)
      ? `<div class="cuerpo-panel" role="alert"><p>${esc(estado.error)}</p><button type="button" class="boton-secundario" data-inscripcion-accion="reintentar">${esc(t("reintentar"))}</button></div>` : "";
    contenedor.innerHTML = `<section class="panel" aria-labelledby="titulo-inscripcion-bolsa" aria-busy="${estado.carga}">
      ${cabecera(textos, estado.ayuda)}${contenido}${error}</section>`;
  }

  function iniciarConsulta() {
    consulta?.abort();
    consulta = new AbortController();
    estado.enviando = false;
    return { version: ++secuencia, signal: consulta.signal };
  }

  function cambiarURL(id) {
    ventana.history.pushState({ vista: "inscripcion" }, "", urlId(ventana.location.href, id));
  }

  async function abiertas({ mas = false } = {}) {
    const { version, signal } = iniciarConsulta();
    estado.tipo = "abiertas"; estado.carga = true; estado.error = ""; pintar();
    try {
      const datos = await clienteEfectivo.abiertas({ cursor: mas ? estado.cursor : "", signal });
      if (!montado || signal.aborted || version !== secuencia) return;
      estado.abiertas = mas ? [...estado.abiertas, ...datos.convocatorias] : datos.convocatorias;
      estado.total = datos.total; estado.cursor = datos.cursor_siguiente;
    } catch (error) {
      if (cerrarPorDenegacion(error)) return;
      if (!montado || signal.aborted || version !== secuencia) return;
      estado.error = descripcionError(error, textos);
      anunciar(estado.error);
    } finally {
      if (montado && version === secuencia) {
        estado.carga = false; pintar();
        if (!mas && !estado.error) contenedor.querySelector?.("#titulo-inscripcion-bolsa")?.focus?.({ preventScroll: true });
      }
    }
  }

  async function cargarBolsa(ref) {
    const { version, signal } = iniciarConsulta();
    estado.tipo = "bolsa"; estado.carga = true; estado.error = ""; estado.bolsa = null;
    estado.revision = false; estado.comprobarEnvio = false; estado.bloqueoActo = false;
    estado.declaraciones = new Set(); pintar();
    try {
      const datos = await clienteEfectivo.convocatoria(ref, { signal });
      if (!montado || signal.aborted || version !== secuencia) return;
      estado.bolsa = datos.convocatoria;
      const pendiente = CLAVES_PENDIENTES.get(ref);
      if (pendiente) estado.declaraciones = new Set(pendiente.declaraciones.map((d) => d.requisito_codigo));
      estado.categoriaRef = pendiente?.categoriaRef ?? (datos.convocatoria.categorias.length === 1
        ? datos.convocatoria.categorias[0].categoria_ref : "");
    } catch (error) {
      if (cerrarPorDenegacion(error)) return;
      if (!montado || signal.aborted || version !== secuencia) return;
      estado.error = descripcionError(error, textos); anunciar(estado.error);
    } finally {
      if (montado && version === secuencia) {
        estado.carga = false; pintar();
        if (!estado.error) contenedor.querySelector?.("h3[tabindex]")?.focus?.({ preventScroll: true });
      }
    }
  }

  async function cargarSolicitud(ref) {
    const { version, signal } = iniciarConsulta();
    estado.tipo = "solicitud"; estado.carga = true; estado.error = ""; estado.solicitud = null; pintar();
    try {
      const datos = await clienteEfectivo.detallePropio(ref, { signal });
      if (!montado || signal.aborted || version !== secuencia) return;
      estado.solicitud = datos.solicitud;
    } catch (error) {
      if (cerrarPorDenegacion(error)) return;
      if (!montado || signal.aborted || version !== secuencia) return;
      estado.error = descripcionError(error, textos); anunciar(estado.error);
    } finally {
      if (montado && version === secuencia) {
        estado.carga = false; pintar();
        if (!estado.error) contenedor.querySelector?.("h3[tabindex]")?.focus?.({ preventScroll: true });
      }
    }
  }

  async function propias({ mas = false } = {}) {
    const { version, signal } = iniciarConsulta();
    estado.tipo = "propias"; estado.carga = true; estado.error = "";
    estado.bolsa = null; estado.solicitud = null; pintar();
    try {
      const datos = await clienteEfectivo.propias({ cursor: mas ? estado.cursorPropias : "", signal });
      if (!montado || signal.aborted || version !== secuencia) return;
      estado.propias = mas ? [...estado.propias, ...datos.solicitudes] : datos.solicitudes;
      estado.cursorPropias = datos.cursor_siguiente;
    } catch (error) {
      if (cerrarPorDenegacion(error)) return;
      if (!montado || signal.aborted || version !== secuencia) return;
      estado.error = descripcionError(error, textos); anunciar(estado.error);
    } finally {
      if (montado && version === secuencia) { estado.carga = false; pintar(); }
    }
  }

  async function confirmar() {
    const bolsa = estado.bolsa;
    const categoria = bolsa?.categorias.find((c) => c.categoria_ref === estado.categoriaRef);
    if (!bolsa || !categoria || !bolsa.puede_iniciar || !estado.revision || estado.enviando || ENVIOS_ACTIVOS.has(bolsa.convocatoria_ref)
      || estado.comprobarEnvio || estado.bloqueoActo) return;
    const version = secuencia;
    let peticion = CLAVES_PENDIENTES.get(bolsa.convocatoria_ref);
    if (!peticion) {
      const clave = globalThis.crypto?.randomUUID?.();
      if (!clave) { estado.error = t("error"); pintar(); return; }
      peticion = Object.freeze({ clave, catalogoVersion: bolsa.catalogo_version, categoriaRef: categoria.categoria_ref,
        declaraciones: Object.freeze(bolsa.requisitos.filter((r) => estado.declaraciones.has(r.codigo))
          .map((r) => Object.freeze({ requisito_codigo: r.codigo }))) });
      CLAVES_PENDIENTES.set(bolsa.convocatoria_ref, peticion);
    }
    ENVIOS_ACTIVOS.add(bolsa.convocatoria_ref);
    estado.enviando = true; estado.error = ""; pintar();
    try {
      const recibo = await clienteEfectivo.inscribir({ convocatoriaRef: bolsa.convocatoria_ref,
        categoriaRef: peticion.categoriaRef,
        catalogoVersion: peticion.catalogoVersion, claveIdempotencia: peticion.clave,
        declaraciones: peticion.declaraciones });
      CLAVES_PENDIENTES.delete(bolsa.convocatoria_ref);
      if (!montado) return;
      if (version !== secuencia) {
        if (estado.tipo === "bolsa" && estado.bolsa?.convocatoria_ref === bolsa.convocatoria_ref) {
          estado.comprobarEnvio = true; estado.error = t("envioComprobar");
        }
        return;
      }
      if (estado.tipo !== "bolsa" || estado.bolsa?.convocatoria_ref !== bolsa.convocatoria_ref) return;
      estado.solicitud = { ...recibo, categoria: categoria.categoria }; estado.tipo = "solicitud"; estado.revision = false;
      cambiarURL(`solicitud:${recibo.solicitud_ref}`); anunciar(t("registrada"));
    } catch (error) {
      if (cerrarPorDenegacion(error)) return;
      if (error?.status === 400 || error?.status === 409 || error?.status === 422) CLAVES_PENDIENTES.delete(bolsa.convocatoria_ref);
      if (!montado || version !== secuencia || estado.tipo !== "bolsa" || estado.bolsa?.convocatoria_ref !== bolsa.convocatoria_ref) return;
      if (error?.status === 400 || error?.status === 409 || error?.status === 422) estado.bloqueoActo = true;
      estado.error = descripcionError(error, textos); anunciar(estado.error);
    } finally {
      ENVIOS_ACTIVOS.delete(bolsa.convocatoria_ref);
      if (montado && (version === secuencia || (estado.tipo === "bolsa" && estado.bolsa?.convocatoria_ref === bolsa.convocatoria_ref))) {
        estado.enviando = false; pintar();
      }
    }
  }

  function pulsar(evento) {
    const boton = evento.target?.closest?.("[data-inscripcion-accion]");
    if (!boton || !contenedor.contains(boton)) return;
    const accion = boton.dataset.inscripcionAccion;
    const ref = boton.dataset.ref;
    if (accion === "ayuda") { estado.ayuda = !estado.ayuda; pintar(); contenedor.querySelector('[data-inscripcion-accion="ayuda"]')?.focus(); }
    else if (accion === "volver") { cambiarURL(""); void abiertas(); }
    else if (accion === "reintentar") {
      const id = leerId(ventana.location.href);
      if (id?.tipo === "bolsa") void cargarBolsa(id.ref);
      else if (id?.tipo === "solicitud") void cargarSolicitud(id.ref);
      else if (id?.tipo === "propias") void propias();
      else void abiertas();
    } else if (accion === "bolsa" && REFERENCIA.test(ref ?? "")) { cambiarURL(`convocatoria:${ref}`); void cargarBolsa(ref); }
    else if (accion === "solicitud" && REFERENCIA.test(ref ?? "")) { cambiarURL(`solicitud:${ref}`); void cargarSolicitud(ref); }
    else if (accion === "propias") { cambiarURL("mis-solicitudes"); void propias(); }
    else if (accion === "actualizar-propias") void propias();
    else if (accion === "actualizar-ficha" && estado.bolsa?.convocatoria_ref) void cargarBolsa(estado.bolsa.convocatoria_ref);
    else if (accion === "mas" && estado.cursor) void abiertas({ mas: true });
    else if (accion === "mas-propias" && estado.cursorPropias) void propias({ mas: true });
    else if (accion === "revisar" && estado.bolsa?.puede_iniciar && estado.categoriaRef) {
      estado.revision = true; pintar();
      contenedor.querySelector?.("[data-inscripcion-revision]")?.focus?.({ preventScroll: true });
    }
    else if (accion === "corregir") {
      estado.revision = false; pintar(); contenedor.querySelector?.("h3[tabindex]")?.focus?.({ preventScroll: true });
    }
    else if (accion === "confirmar") void confirmar();
  }

  function cambiar(evento) {
    if (evento.target?.matches?.("[data-inscripcion-categoria]") && estado.tipo === "bolsa") {
      if (CLAVES_PENDIENTES.has(estado.bolsa?.convocatoria_ref)) return;
      const ref = evento.target.value;
      estado.categoriaRef = estado.bolsa?.categorias.some((c) => c.categoria_ref === ref) ? ref : "";
      pintar(); contenedor.querySelector?.("[data-inscripcion-categoria]")?.focus?.({ preventScroll: true });
      return;
    }
    const casilla = evento.target?.closest?.("[data-inscripcion-requisito]");
    if (!casilla || !contenedor.contains(casilla) || estado.tipo !== "bolsa") return;
    const codigo = casilla.dataset.inscripcionRequisito;
    if (!estado.bolsa?.requisitos.some((r) => r.codigo === codigo)) return;
    if (casilla.checked) estado.declaraciones.add(codigo);
    else estado.declaraciones.delete(codigo);
  }

  contenedor.addEventListener("click", pulsar);
  contenedor.addEventListener("change", cambiar);
  pintar();
  (async () => {
    try {
      try { textos = await cargarTextos("bolsa-inscripcion-aspirante", { idioma }); }
      catch { textos = await reintentarTextos("bolsa-inscripcion-aspirante", { idioma }); }
      if (!montado) return;
      clienteEfectivo ??= crearClienteInscripcionBolsa({ fetchImpl, idioma: textos.idioma });
      const id = leerId(ventana.location.href);
      if (id?.tipo === "bolsa") await cargarBolsa(id.ref);
      else if (id?.tipo === "solicitud") await cargarSolicitud(id.ref);
      else if (id?.tipo === "propias") await propias();
      else await abiertas();
    } catch {
      if (!montado) return;
      contenedor.innerHTML = `<section class="panel" role="alert"><div class="cuerpo-panel">${esc(textoBase("error"))}</div></section>`;
      anunciar(textoBase("error"));
    }
  })();
  return Object.freeze({ destruir() { montado = false; ++secuencia; consulta?.abort();
    contenedor.removeEventListener("click", pulsar); contenedor.removeEventListener("change", cambiar);
    contenedor.replaceChildren(); } });
}
