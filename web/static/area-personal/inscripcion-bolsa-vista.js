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
  if (valor.startsWith("bolsa:") && REFERENCIA.test(valor.slice(6))) return { tipo: "bolsa", ref: valor.slice(6) };
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

function tarjetaAbierta(bolsa, textos) {
  const t = (clave) => esc(textos.traducir(`vista.${clave}`));
  const propia = bolsa.estado_solicitud_propia;
  const enlace = propia && bolsa.solicitud_ref
    ? `<button type="button" class="boton-secundario" data-inscripcion-accion="solicitud" data-ref="${esc(bolsa.solicitud_ref)}">${t("verSolicitud")}</button>`
    : `<button type="button" class="boton-primario" data-inscripcion-accion="bolsa" data-ref="${esc(bolsa.bolsa_ref)}">${t("verBolsa")}</button>`;
  return `<article class="panel portal-mi-bolsa__bolsa"><div class="cabecera-panel"><h3>${esc(bolsa.categoria)}</h3>
    ${propia ? `<span class="estado-chip info">${t(`estado_${propia}`)}</span>` : ""}</div>
    <div class="cuerpo-panel"><dl class="lista-datos"><div><dt>${t("plazo")}</dt><dd>${fecha(textos, bolsa.plazo_inicio)} – ${fecha(textos, bolsa.plazo_fin)}</dd></div>
    <div><dt>${t("requisitos")}</dt><dd>${esc(bolsa.requisitos_resumen)}</dd></div></dl>
    <div class="acciones-vista">${enlace}</div></div></article>`;
}

function vistaAbiertas(estado, textos) {
  const t = (clave, variables) => esc(textos.traducir(`vista.${clave}`, variables));
  if (estado.carga) return `<div class="cuerpo-panel" role="status" aria-live="polite">${t("cargando")}</div>`;
  if (estado.error) return `<div class="cuerpo-panel" role="alert"><p>${esc(estado.error)}</p>
    <button type="button" class="boton-secundario" data-inscripcion-accion="reintentar">${t("reintentar")}</button></div>`;
  if (!estado.abiertas.length) return `<div class="cuerpo-panel vacio-controlado" role="status"><p>${t("vacio")}</p>
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
  const pendiente = CLAVES_PENDIENTES.has(bolsa.bolsa_ref);
  const requisitos = bolsa.requisitos.map((requisito) => `<li>${esc(requisito.descripcion)}${requisito.obligatorio
    ? ` <span class="estado-chip aviso">${t("obligatorio")}</span>` : ""}
    <label class="opcion-check"><input type="checkbox" data-inscripcion-requisito="${esc(requisito.codigo)}"
      ${estado.declaraciones.has(requisito.codigo) ? "checked" : ""} ${pendiente ? "disabled" : ""}><span>${t("declarar")}</span></label></li>`).join("");
  const declarados = bolsa.requisitos.filter((requisito) => estado.declaraciones.has(requisito.codigo));
  const enviando = estado.enviando || ENVIOS_ACTIVOS.has(bolsa.bolsa_ref);
  return `<div class="cuerpo-panel"><button type="button" class="boton-secundario" data-inscripcion-accion="volver">${t("volver")}</button>
    <h3 tabindex="-1">${esc(bolsa.categoria)}</h3><dl class="lista-datos"><div><dt>${t("plazo")}</dt>
      <dd>${fecha(textos, bolsa.plazo_inicio)} – ${fecha(textos, bolsa.plazo_fin)}</dd></div></dl>
    <section class="panel"><div class="cabecera-panel"><h4>${t("requisitos")}</h4></div><div class="cuerpo-panel">${requisitos ? `<ul>${requisitos}</ul>` : `<p>${t("sinRequisitos")}</p>`}</div></section>
    ${estado.revision ? `<section class="panel"><div class="cabecera-panel"><h4>${t("revisarTitulo")}</h4></div><div class="cuerpo-panel">
      <p>${t(pendiente ? "revisarPendiente" : "revisarActo")}</p><p>${t("declaracionesRevisar")}</p>
      ${declarados.length ? `<ul>${declarados.map((requisito) => `<li>${esc(requisito.descripcion)}</li>`).join("")}</ul>` : `<p>${t("sinDeclaraciones")}</p>`}
      <div class="acciones-vista"><button type="button" class="boton-secundario" data-inscripcion-accion="corregir" ${enviando ? "disabled" : ""}>${t("corregir")}</button>
      <button type="button" class="boton-primario" data-inscripcion-accion="confirmar" ${enviando ? "disabled" : ""}>${t(enviando ? "enviando" : "confirmar")}</button></div></div></section>`
      : `<div class="acciones-vista"><button type="button" class="boton-primario" data-inscripcion-accion="revisar" ${enviando || estado.comprobarEnvio ? "disabled" : ""}>${t("solicitar")}</button></div>`}
    ${estado.error ? `<p role="alert">${esc(estado.error)}</p><button type="button" class="boton-secundario" data-inscripcion-accion="propias">${t("misSolicitudes")}</button>` : ""}</div>`;
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
  if (!estado.propias.length) return `<div class="cuerpo-panel"><button type="button" class="boton-secundario" data-inscripcion-accion="volver">${t("volver")}</button>
    <p role="status">${t("sinSolicitudes")}</p></div>`;
  return `<div class="cuerpo-panel"><button type="button" class="boton-secundario" data-inscripcion-accion="volver">${t("volver")}</button>
    <h3>${t("misSolicitudes")}</h3><button type="button" class="boton-secundario" data-inscripcion-accion="actualizar-propias">${t("actualizarSolicitudes")}</button>
    <div class="marco-participaciones">${estado.propias.map((solicitud) =>
    `<article class="panel portal-mi-bolsa__bolsa"><div class="cabecera-panel"><h4>${esc(solicitud.categoria)}</h4>
      <span class="estado-chip info">${t(`estado_${solicitud.estado}`)}</span></div><div class="cuerpo-panel">
      <p>${fecha(textos, solicitud.registrada_en)}</p><button type="button" class="boton-secundario" data-inscripcion-accion="solicitud" data-ref="${esc(solicitud.solicitud_ref)}">${t("verSolicitud")}</button>
      </div></article>`).join("")}</div>${estado.cursorPropias ? `<button type="button" class="boton-secundario" data-inscripcion-accion="mas-propias">${t("mostrarMas")}</button>` : ""}</div>`;
}

export function montarInscripcionBolsa({ contenedor, cliente = crearClienteInscripcionBolsa(),
  idioma, ventana = globalThis.window, anunciar = () => {}, textoBase = () => "" } = {}) {
  if (!contenedor?.addEventListener || !contenedor?.removeEventListener || !ventana?.location
    || !ventana?.history || !cliente?.abiertas || !cliente?.bolsa || !cliente?.propias
    || !cliente?.detallePropio || !cliente?.inscribir) throw new TypeError("Montaje de inscripción inválido");
  const estado = { tipo: "abiertas", carga: true, abiertas: [], propias: [], total: 0,
    cursor: null, cursorPropias: null, bolsa: null, solicitud: null, error: "", revision: false,
    enviando: false, comprobarEnvio: false, ayuda: false, declaraciones: new Set() };
  let textos = null;
  let montado = true;
  let consulta = null;
  let secuencia = 0;
  const t = (clave) => textos.traducir(`vista.${clave}`);

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
      const datos = await cliente.abiertas({ cursor: mas ? estado.cursor : "", signal });
      if (!montado || signal.aborted || version !== secuencia) return;
      estado.abiertas = mas ? [...estado.abiertas, ...datos.bolsas] : datos.bolsas;
      estado.total = datos.total; estado.cursor = datos.cursor_siguiente;
    } catch (error) {
      if (!montado || signal.aborted || version !== secuencia) return;
      if (error?.status === 401 || error?.status === 403) {
        estado.abiertas = []; estado.total = 0; estado.cursor = null;
      }
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
    estado.revision = false; estado.comprobarEnvio = false; estado.declaraciones = new Set(); pintar();
    try {
      const datos = await cliente.bolsa(ref, { signal });
      if (!montado || signal.aborted || version !== secuencia) return;
      estado.bolsa = datos.bolsa;
      const pendiente = CLAVES_PENDIENTES.get(ref);
      if (pendiente) estado.declaraciones = new Set(pendiente.declaraciones.map((d) => d.requisito_codigo));
    } catch (error) {
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
      const datos = await cliente.detallePropio(ref, { signal });
      if (!montado || signal.aborted || version !== secuencia) return;
      estado.solicitud = datos.solicitud;
    } catch (error) {
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
      const datos = await cliente.propias({ cursor: mas ? estado.cursorPropias : "", signal });
      if (!montado || signal.aborted || version !== secuencia) return;
      estado.propias = mas ? [...estado.propias, ...datos.solicitudes] : datos.solicitudes;
      estado.cursorPropias = datos.cursor_siguiente;
    } catch (error) {
      if (!montado || signal.aborted || version !== secuencia) return;
      if (error?.status === 401 || error?.status === 403) {
        estado.propias = []; estado.cursorPropias = null;
      }
      estado.error = descripcionError(error, textos); anunciar(estado.error);
    } finally {
      if (montado && version === secuencia) { estado.carga = false; pintar(); }
    }
  }

  async function confirmar() {
    const bolsa = estado.bolsa;
    if (!bolsa || !estado.revision || estado.enviando || ENVIOS_ACTIVOS.has(bolsa.bolsa_ref)
      || estado.comprobarEnvio) return;
    const version = secuencia;
    let peticion = CLAVES_PENDIENTES.get(bolsa.bolsa_ref);
    if (!peticion) {
      const clave = globalThis.crypto?.randomUUID?.();
      if (!clave) { estado.error = t("error"); pintar(); return; }
      peticion = Object.freeze({ clave, catalogoVersion: bolsa.catalogo_version,
        declaraciones: Object.freeze(bolsa.requisitos.filter((r) => estado.declaraciones.has(r.codigo))
          .map((r) => Object.freeze({ requisito_codigo: r.codigo }))) });
      CLAVES_PENDIENTES.set(bolsa.bolsa_ref, peticion);
    }
    ENVIOS_ACTIVOS.add(bolsa.bolsa_ref);
    estado.enviando = true; estado.error = ""; pintar();
    try {
      const recibo = await cliente.inscribir({ bolsaRef: bolsa.bolsa_ref,
        catalogoVersion: peticion.catalogoVersion, claveIdempotencia: peticion.clave,
        declaraciones: peticion.declaraciones });
      CLAVES_PENDIENTES.delete(bolsa.bolsa_ref);
      if (!montado) return;
      if (version !== secuencia) {
        if (estado.tipo === "bolsa" && estado.bolsa?.bolsa_ref === bolsa.bolsa_ref) {
          estado.comprobarEnvio = true; estado.error = t("envioComprobar");
        }
        return;
      }
      if (estado.tipo !== "bolsa" || estado.bolsa?.bolsa_ref !== bolsa.bolsa_ref) return;
      estado.solicitud = { ...recibo, categoria: bolsa.categoria }; estado.tipo = "solicitud"; estado.revision = false;
      cambiarURL(`solicitud:${recibo.solicitud_ref}`); anunciar(t("registrada"));
    } catch (error) {
      if (!montado || version !== secuencia || estado.tipo !== "bolsa" || estado.bolsa?.bolsa_ref !== bolsa.bolsa_ref) return;
      estado.error = descripcionError(error, textos); anunciar(estado.error);
    } finally {
      ENVIOS_ACTIVOS.delete(bolsa.bolsa_ref);
      if (montado && (version === secuencia || (estado.tipo === "bolsa" && estado.bolsa?.bolsa_ref === bolsa.bolsa_ref))) {
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
    } else if (accion === "bolsa" && REFERENCIA.test(ref ?? "")) { cambiarURL(`bolsa:${ref}`); void cargarBolsa(ref); }
    else if (accion === "solicitud" && REFERENCIA.test(ref ?? "")) { cambiarURL(`solicitud:${ref}`); void cargarSolicitud(ref); }
    else if (accion === "propias") { cambiarURL("mis-solicitudes"); void propias(); }
    else if (accion === "actualizar-propias") void propias();
    else if (accion === "mas" && estado.cursor) void abiertas({ mas: true });
    else if (accion === "mas-propias" && estado.cursorPropias) void propias({ mas: true });
    else if (accion === "revisar" && estado.bolsa) { estado.revision = true; pintar(); }
    else if (accion === "corregir") { estado.revision = false; pintar(); }
    else if (accion === "confirmar") void confirmar();
  }

  function cambiar(evento) {
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
