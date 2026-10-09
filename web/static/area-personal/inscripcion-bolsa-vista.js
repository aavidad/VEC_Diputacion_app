import { cargarTextos, reintentarTextos } from "../comun/textos.js";
import { crearClienteInscripcionBolsa } from "./inscripcion-bolsa-api.js?v=20261009-inscripcion-v1";

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

function descripcionError(error, textos, envio = false) {
  if (error?.status === 401 || error?.status === 403) CLAVES_PENDIENTES.clear();
  if (error?.status === 401) return textos.traducir("vista.sinSesion");
  if (error?.status === 403) return textos.traducir("vista.denegada");
  if (error?.status === 404) return textos.traducir("vista.noEncontrada");
  if (error?.status === 400) return textos.traducir("vista.datosNoAceptados");
  if (error?.status === 409) return textos.traducir("vista.conflicto");
  if (error?.status === 422 && error?.codigo === "plazo_cerrado") return textos.traducir("vista.plazoCerrado");
  if (error?.status === 422) return textos.traducir("vista.requisitosCambiados");
  if (error?.status === 503) return textos.traducir(envio ? "vista.envioTemporal" : "vista.temporal");
  return textos.traducir("vista.error");
}

// Un mismo estado se pinta con el mismo color en lista, ficha y Mis solicitudes.
const CLASE_ESTADO = Object.freeze({ pendiente: "aviso", admitida_a_convocatoria: "info", incorporada: "exito", rechazada: "peligro" });
const CLASE_REQUISITO = Object.freeze({ cumple: "exito", no_cumple: "peligro", pendiente: "aviso" });
const TITULO_PANTALLA = Object.freeze({ abiertas: "tituloAbiertas", bolsa: "tituloFicha", solicitud: "solicitudTitulo", propias: "misSolicitudes" });

function chipEstado(estado, t) {
  return `<span class="estado-chip ${CLASE_ESTADO[estado] ?? "aviso"}">${t(`estado_${estado}`)}</span>`;
}

function avisoError(texto, acciones = "") {
  return `<div class="nota error" role="alert"><p>${esc(texto)}</p>${acciones ? `<div class="fila-acciones">${acciones}</div>` : ""}</div>`;
}

function cabecera(estado, textos) {
  const t = (clave) => esc(textos.traducir(`vista.${clave}`));
  return `<header><div><h2 id="titulo-inscripcion-bolsa" tabindex="-1">${t(TITULO_PANTALLA[estado.tipo] ?? "tituloAbiertas")}</h2></div>
    <button type="button" class="boton-secundario" data-inscripcion-accion="ayuda"
      aria-expanded="${estado.ayuda}" aria-controls="ayuda-inscripcion-bolsa" aria-label="${t("ayudaEtiqueta")}">?</button></header>
    <div class="panel-contenido" id="ayuda-inscripcion-bolsa" ${estado.ayuda ? "" : "hidden"}><p>${t("ayuda")}</p></div>`;
}

function fecha(textos, valor) {
  return esc(textos.fecha(valor, { dateStyle: "medium", timeZone: "Europe/Madrid" }));
}
function fechaHora(textos, valor) {
  return esc(textos.fecha(valor, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }));
}

function textoBuscable(valor) {
  return String(valor ?? "").normalize("NFD").replace(/\p{M}/gu, "").toLocaleLowerCase();
}

function tarjetaAbierta(bolsa, textos) {
  const t = (clave) => esc(textos.traducir(`vista.${clave}`));
  const propia = bolsa.estado_solicitud_propia;
  const enlace = propia && bolsa.solicitud_ref
    ? `<button type="button" class="boton-secundario" data-inscripcion-accion="solicitud" data-ref="${esc(bolsa.solicitud_ref)}">${t("verSolicitud")}</button>`
    : `<button type="button" class="boton-primario" data-inscripcion-accion="bolsa" data-ref="${esc(bolsa.convocatoria_ref)}">${t("verBolsa")}</button>`;
  return `<article class="panel" tabindex="-1"><header><h3>${esc(bolsa.titulo)}</h3>
    ${propia ? chipEstado(propia, t) : ""}</header>
    <div class="panel-contenido"><dl class="dato-lista"><dt>${t("plazo")}</dt><dd>${fechaHora(textos, bolsa.plazo_inicio)} – ${fechaHora(textos, bolsa.plazo_fin)}</dd>
    <dt>${t("categoria")}</dt><dd>${esc(textos.plural("vista.categorias", bolsa.numero_categorias))}</dd>
    <dt>${t("requisitos")}</dt><dd>${esc(bolsa.requisitos_resumen)}</dd></dl>
    <div class="fila-acciones">${enlace}</div></div></article>`;
}

function vistaAbiertas(estado, textos) {
  const t = (clave, variables) => esc(textos.traducir(`vista.${clave}`, variables));
  if (estado.carga) return `<div class="panel-contenido" role="status" aria-live="polite">${t("cargando")}</div>`;
  if (estado.error) return `<div class="panel-contenido">${avisoError(estado.error, [401, 403, 404].includes(estado.errorStatus)
    ? "" : `<button type="button" class="boton-secundario" data-inscripcion-accion="reintentar">${t("reintentar")}</button>`)}</div>`;
  if (!estado.abiertas.length) return `<div class="panel-contenido estado-vacio" role="status"><p>${t("vacio")}</p>
    <div class="fila-acciones"><button type="button" class="boton-secundario" data-inscripcion-accion="propias">${t("misSolicitudes")}</button>
    <button type="button" class="boton-secundario" data-inscripcion-accion="reintentar">${t("actualizar")}</button></div></div>`;
  return `<div class="panel-contenido"><div class="fila-acciones"><button type="button" class="boton-secundario"
      data-inscripcion-accion="propias">${t("misSolicitudes")}</button></div>
    <div class="marco-participaciones">${estado.abiertas.map((bolsa) => tarjetaAbierta(bolsa, textos)).join("")}</div>
    ${estado.cursor ? `<div class="fila-acciones"><button type="button" class="boton-secundario" data-inscripcion-accion="mas">${t("mostrarMas")}</button></div>` : ""}</div>`;
}

function requisitoFicha(requisito, indice, estado, pendiente, t) {
  const id = `requisito-inscripcion-${indice}`;
  const casilla = requisito.estado === "cumple" ? "" : `<label class="opcion-check"><input type="checkbox" data-inscripcion-requisito="${esc(requisito.codigo)}"
      aria-describedby="${id}" ${estado.declaraciones.has(requisito.codigo) ? "checked" : ""} ${pendiente || requisito.estado === "no_cumple" ? "disabled" : ""}><span>${t("declarar")}</span></label>`;
  return `<li><span id="${id}">${esc(requisito.descripcion)}</span>
    <span class="estado-chip ${requisito.obligatorio ? "aviso" : "info"}">${t(requisito.obligatorio ? "obligatorio" : "opcional")}</span>
    <span class="estado-chip ${CLASE_REQUISITO[requisito.estado]}">${t(`requisito_${requisito.estado}`)}</span>
    <p>${esc(requisito.motivo_etiqueta)}</p>
    ${requisito.hito_etiqueta ? `<p>${t("cuandoExigido")}: ${esc(requisito.hito_etiqueta)}${requisito.hito_fecha ? ` · ${fecha(t.textos, requisito.hito_fecha)}` : ""}</p>` : ""}
    ${casilla}</li>`;
}

function revisionFicha(estado, bolsa, categoria, pendiente, enviando, textos, t) {
  const declarados = bolsa.requisitos.filter((requisito) => estado.declaraciones.has(requisito.codigo));
  return `<section class="panel"><header><h4 tabindex="-1" data-inscripcion-revision>${t("revisarTitulo")}</h4></header><div class="panel-contenido">
      <dl class="dato-lista"><dt>${t("convocatoria")}</dt><dd>${esc(bolsa.titulo)}</dd>
      <dt>${t("categoria")}</dt><dd>${esc(categoria?.categoria)}</dd>
      <dt>${t("plazo")}</dt><dd>${fechaHora(textos, bolsa.plazo_inicio)} – ${fechaHora(textos, bolsa.plazo_fin)}</dd></dl>
      <p>${t(pendiente ? "revisarPendiente" : "revisarActo")}</p><p>${t("declaracionesRevisar")}</p>
      ${declarados.length ? `<ul>${declarados.map((requisito) => `<li>${esc(requisito.descripcion)} · ${t(`requisito_${requisito.estado}`)}</li>`).join("")}</ul>` : `<p>${t("sinDeclaraciones")}</p>`}
      <div class="fila-acciones"><button type="button" class="boton-secundario" data-inscripcion-accion="corregir" ${enviando ? "disabled" : ""}>${t("corregir")}</button>
      <button type="button" class="boton-primario" data-inscripcion-accion="confirmar" ${enviando || estado.bloqueoActo || !categoria ? "disabled" : ""}>${t(enviando ? "enviando" : "confirmar")}</button></div></div></section>`;
}

function vistaBolsa(estado, textos) {
  const t = Object.assign((clave) => esc(textos.traducir(`vista.${clave}`)), { textos });
  const bolsa = estado.bolsa;
  if (!bolsa) return "";
  const pendiente = CLAVES_PENDIENTES.has(bolsa.convocatoria_ref);
  const categoria = bolsa.categorias.find((c) => c.categoria_ref === estado.categoriaRef);
  const enviando = estado.enviando || ENVIOS_ACTIVOS.has(bolsa.convocatoria_ref);
  const error = estado.error ? avisoError(estado.error, `${estado.bloqueoActo
    ? `<button type="button" class="boton-secundario" data-inscripcion-accion="actualizar-ficha">${t("actualizarFicha")}</button>` : ""}
      <button type="button" class="boton-secundario" data-inscripcion-accion="propias">${t("misSolicitudes")}</button>`) : "";
  const inicio = `<div class="panel-contenido"><button type="button" class="boton-secundario" data-inscripcion-accion="volver">${t("volver")}</button>
    <h3 tabindex="-1">${esc(bolsa.titulo)}</h3><dl class="dato-lista"><dt>${t("plazo")}</dt>
      <dd>${fechaHora(textos, bolsa.plazo_inicio)} – ${fechaHora(textos, bolsa.plazo_fin)}</dd></dl>`;
  // En la revisión sólo queda el resumen: lo elegido se cambia con «Volver a la ficha».
  if (estado.revision) return `${inicio}${revisionFicha(estado, bolsa, categoria, pendiente, enviando, textos, t)}${error}</div>`;
  const filtroCategoria = textoBuscable(estado.filtroCategoria.trim());
  const categoriasVisibles = filtroCategoria
    ? bolsa.categorias.filter((c) => textoBuscable(c.categoria).includes(filtroCategoria)) : bolsa.categorias;
  const requisitos = bolsa.requisitos.map((requisito, indice) => requisitoFicha(requisito, indice, estado, pendiente, t)).join("");
  const faltaCategoria = estado.faltaCategoria && !categoria;
  return `${inicio}
    ${bolsa.categorias.length > 10 ? `<div class="campo"><label for="buscar-categoria-inscripcion-bolsa">${t("buscarCategoria")}</label>
      <input id="buscar-categoria-inscripcion-bolsa" type="search" data-inscripcion-buscar-categoria
        value="${esc(estado.filtroCategoria)}" autocomplete="off" ${pendiente ? "disabled" : ""}></div>` : ""}
    <div class="campo"><label for="categoria-inscripcion-bolsa">${t(bolsa.categorias.length > 1 ? "categoriaObligatoria" : "categoria")}</label>
      <select id="categoria-inscripcion-bolsa" data-inscripcion-categoria ${faltaCategoria ? 'aria-invalid="true" aria-describedby="falta-categoria-inscripcion"' : ""}
        ${pendiente || bolsa.categorias.length === 1 || !categoriasVisibles.length ? "disabled" : ""}>
      ${bolsa.categorias.length > 1 ? `<option value="">${t("elegirCategoria")}</option>` : ""}
      ${categoriasVisibles.map((c) => `<option value="${esc(c.categoria_ref)}" ${c.categoria_ref === estado.categoriaRef ? "selected" : ""}>${esc(c.categoria)}</option>`).join("")}
      </select>${faltaCategoria ? `<small id="falta-categoria-inscripcion" class="error-campo">${t("faltaCategoria")}</small>` : ""}
      ${categoria && bolsa.categorias.length > 1 ? `<small>${t("categoriaElegida")}: ${esc(categoria.categoria)}</small>` : ""}
      ${categoriasVisibles.length ? "" : `<p role="status">${t("sinCategorias")}</p>`}</div>
    <section class="panel"><header><h4>${t("requisitos")}</h4></header><div class="panel-contenido">${requisitos ? `<ul class="lista-requisitos-inscripcion">${requisitos}</ul>` : `<p>${t("sinRequisitos")}</p>`}</div></section>
    ${bolsa.puede_iniciar ? "" : `<p class="nota aviso" role="status">${esc(bolsa.impedimento_etiqueta)}</p>`}
    <div class="fila-acciones"><button type="button" class="boton-primario" data-inscripcion-accion="revisar" ${enviando || estado.comprobarEnvio || estado.bloqueoActo || !bolsa.puede_iniciar ? "disabled" : ""}>${t("solicitar")}</button></div>
    ${error}</div>`;
}

function vistaSolicitud(estado, textos) {
  const t = (clave) => esc(textos.traducir(`vista.${clave}`));
  const solicitud = estado.solicitud;
  if (!solicitud) return "";
  const volver = estado.origenSolicitud === "propias"
    ? `<button type="button" class="boton-secundario" data-inscripcion-accion="propias">${t("volverSolicitudes")}</button>`
    : `<button type="button" class="boton-secundario" data-inscripcion-accion="volver">${t("volver")}</button>`;
  return `<div class="panel-contenido">${volver}
    ${estado.recienRegistrada ? `<p class="nota exito" tabindex="-1" data-inscripcion-aviso>${t("registrada")}</p>` : ""}
    <dl class="dato-lista">${solicitud.tituloBolsa ? `<dt>${t("convocatoria")}</dt><dd>${esc(solicitud.tituloBolsa)}</dd>` : ""}
    <dt>${t("categoria")}</dt><dd>${esc(solicitud.categoria)}</dd>
    <dt>${t("estado")}</dt><dd>${chipEstado(solicitud.estado, t)}</dd>
    <dt>${t("fechaRegistro")}</dt><dd>${fecha(textos, solicitud.registrada_en)}</dd>
    <dt>${t("recibo")}</dt><dd>${esc(solicitud.recibo_ref)}</dd>
    ${solicitud.estado === "rechazada" ? `<dt>${t("motivo")}</dt><dd>${esc(solicitud.motivo_etiqueta)}</dd>` : ""}</dl>
    <p>${t(`paso_${solicitud.estado}`)}</p></div>`;
}

function vistaPropias(estado, textos) {
  const t = (clave) => esc(textos.traducir(`vista.${clave}`));
  const volver = `<button type="button" class="boton-secundario" data-inscripcion-accion="volver">${t("volver")}</button>`;
  if (estado.error) return `<div class="panel-contenido">${volver}</div>`;
  if (!estado.propias.length) return `<div class="panel-contenido">${volver}<p role="status">${t("sinSolicitudes")}</p></div>`;
  return `<div class="panel-contenido"><div class="fila-acciones">${volver}<button type="button" class="boton-secundario" data-inscripcion-accion="actualizar-propias">${t("actualizarSolicitudes")}</button></div>
    <div class="marco-participaciones">${estado.propias.map((solicitud) =>
    `<article class="panel" tabindex="-1"><header><h3>${esc(solicitud.categoria)}</h3>${chipEstado(solicitud.estado, t)}</header><div class="panel-contenido">
      <dl class="dato-lista"><dt>${t("fechaRegistro")}</dt><dd>${fecha(textos, solicitud.registrada_en)}</dd></dl>
      <div class="fila-acciones"><button type="button" class="boton-secundario" data-inscripcion-accion="solicitud" data-ref="${esc(solicitud.solicitud_ref)}">${t("verSolicitud")}</button></div>
      </div></article>`).join("")}</div>${estado.cursorPropias ? `<div class="fila-acciones"><button type="button" class="boton-secundario" data-inscripcion-accion="mas-propias">${t("mostrarMas")}</button></div>` : ""}</div>`;
}

export function montarInscripcionBolsa({ contenedor, fetchImpl = globalThis.fetch,
  cliente = null, idioma,
  ventana = globalThis.window, anunciar = () => {}, textoBase = () => "" } = {}) {
  if (!contenedor?.addEventListener || !contenedor?.removeEventListener || !ventana?.location
    || !ventana?.history || (cliente !== null && (!cliente?.abiertas || !cliente?.convocatoria
      || !cliente?.propias || !cliente?.detallePropio || !cliente?.inscribir)))
    throw new TypeError("Montaje de inscripción inválido");
  const estado = { tipo: "abiertas", carga: true, abiertas: [], propias: [], total: 0,
    cursor: null, cursorPropias: null, bolsa: null, solicitud: null, error: "", errorStatus: 0,
    ultimaAbiertasMas: false, ultimaPropiasMas: false, revision: false,
    enviando: false, comprobarEnvio: false, bloqueoActo: false,
    ayuda: false, declaraciones: new Set(), categoriaRef: "", filtroCategoria: "",
    faltaCategoria: false, origenSolicitud: "abiertas", recienRegistrada: false };
  let textos = null;
  let clienteEfectivo = cliente;
  let montado = true;
  let consulta = null;
  let secuencia = 0;
  const t = (clave) => textos.traducir(`vista.${clave}`);

  function limpiarDatosPrivados() {
    estado.abiertas = []; estado.propias = []; estado.bolsa = null; estado.solicitud = null;
    estado.cursor = null; estado.cursorPropias = null; estado.declaraciones.clear();
    estado.categoriaRef = ""; estado.filtroCategoria = ""; estado.revision = false; estado.total = 0;
  }

  function cerrarPorDenegacion(error) {
    if (!montado || (error?.status !== 401 && error?.status !== 403)) return false;
    ++secuencia;
    consulta?.abort();
    limpiarDatosPrivados();
    estado.carga = false;
    estado.enviando = false;
    estado.errorStatus = error.status;
    estado.error = descripcionError(error, textos);
    pintar(); anunciar(estado.error);
    return true;
  }

  function pintar() {
    if (!montado) return;
    if (!textos) {
      contenedor.innerHTML = `<section class="panel" role="status" aria-live="polite"><div class="panel-contenido">${esc(textoBase("cargando"))}</div></section>`;
      return;
    }
    const contenido = estado.tipo === "abiertas" ? vistaAbiertas(estado, textos)
      : estado.carga ? `<div class="panel-contenido" role="status">${esc(t("cargando"))}</div>`
        : estado.tipo === "bolsa" ? vistaBolsa(estado, textos)
          : estado.tipo === "solicitud" ? vistaSolicitud(estado, textos) : vistaPropias(estado, textos);
    const error = estado.tipo !== "abiertas" && estado.error && (estado.tipo !== "bolsa" || !estado.bolsa)
      ? `<div class="panel-contenido">${avisoError(estado.error, [401, 403].includes(estado.errorStatus)
        ? "" : `<button type="button" class="boton-secundario" data-inscripcion-accion="reintentar">${esc(t("reintentar"))}</button>`)}</div>` : "";
    contenedor.innerHTML = `<section class="panel inscripcion-bolsa" aria-labelledby="titulo-inscripcion-bolsa" aria-busy="${estado.carga}">
      ${cabecera(estado, textos)}${contenido}${error}</section>`;
  }

  // Al cambiar de pantalla el foco va a su título; en la primera carga no se mueve.
  function enfocar(selector = "#titulo-inscripcion-bolsa") {
    contenedor.querySelector?.(selector)?.focus?.({ preventScroll: true });
  }

  function iniciarConsulta() {
    consulta?.abort();
    consulta = new AbortController();
    estado.enviando = false; estado.recienRegistrada = false; estado.faltaCategoria = false;
    return { version: ++secuencia, signal: consulta.signal };
  }

  function cambiarURL(id) {
    ventana.history.pushState({ vista: "inscripcion" }, "", urlId(ventana.location.href, id));
  }

  async function abiertas({ mas = false, inicial = false } = {}) {
    const { version, signal } = iniciarConsulta();
    const previas = mas ? estado.abiertas.length : 0;
    estado.tipo = "abiertas"; estado.carga = true; estado.error = ""; estado.errorStatus = 0;
    estado.ultimaAbiertasMas = mas; pintar();
    try {
      const datos = await clienteEfectivo.abiertas({ cursor: mas ? estado.cursor : "", signal });
      if (!montado || signal.aborted || version !== secuencia) return;
      estado.abiertas = mas ? [...estado.abiertas, ...datos.convocatorias] : datos.convocatorias;
      estado.total = datos.total; estado.cursor = datos.cursor_siguiente;
    } catch (error) {
      if (cerrarPorDenegacion(error)) return;
      if (!montado || signal.aborted || version !== secuencia) return;
      estado.errorStatus = error?.status ?? 0;
      // Sin la ruta publicada (inscripción apagada en el servidor) no hay nada que reintentar.
      estado.error = estado.errorStatus === 404 ? t("noDisponible") : descripcionError(error, textos);
      anunciar(estado.error);
    } finally {
      if (montado && version === secuencia) {
        estado.carga = false; pintar();
        if (mas && !estado.error) contenedor.querySelectorAll?.(".marco-participaciones > article")?.[previas]?.focus?.({ preventScroll: true });
        else if (!inicial) enfocar();
      }
    }
  }

  async function cargarBolsa(ref, { inicial = false } = {}) {
    const { version, signal } = iniciarConsulta();
    estado.tipo = "bolsa"; estado.carga = true; estado.error = ""; estado.errorStatus = 0; estado.bolsa = null;
    estado.revision = false; estado.comprobarEnvio = false; estado.bloqueoActo = false;
    estado.filtroCategoria = "";
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
      estado.errorStatus = error?.status ?? 0;
      estado.error = descripcionError(error, textos); anunciar(estado.error);
    } finally {
      if (montado && version === secuencia) {
        estado.carga = false; pintar();
        if (!inicial) enfocar();
      }
    }
  }

  async function cargarSolicitud(ref, { inicial = false } = {}) {
    const { version, signal } = iniciarConsulta();
    estado.tipo = "solicitud"; estado.carga = true; estado.error = ""; estado.errorStatus = 0;
    estado.solicitud = null; pintar();
    try {
      const datos = await clienteEfectivo.detallePropio(ref, { signal });
      if (!montado || signal.aborted || version !== secuencia) return;
      estado.solicitud = datos.solicitud;
    } catch (error) {
      if (cerrarPorDenegacion(error)) return;
      if (!montado || signal.aborted || version !== secuencia) return;
      estado.errorStatus = error?.status ?? 0;
      estado.error = descripcionError(error, textos); anunciar(estado.error);
    } finally {
      if (montado && version === secuencia) {
        estado.carga = false; pintar();
        if (!inicial) enfocar();
      }
    }
  }

  async function propias({ mas = false, inicial = false } = {}) {
    const { version, signal } = iniciarConsulta();
    const previas = mas ? estado.propias.length : 0;
    estado.tipo = "propias"; estado.carga = true; estado.error = ""; estado.errorStatus = 0;
    estado.ultimaPropiasMas = mas;
    estado.bolsa = null; estado.solicitud = null; pintar();
    try {
      const datos = await clienteEfectivo.propias({ cursor: mas ? estado.cursorPropias : "", signal });
      if (!montado || signal.aborted || version !== secuencia) return;
      estado.propias = mas ? [...estado.propias, ...datos.solicitudes] : datos.solicitudes;
      estado.cursorPropias = datos.cursor_siguiente;
    } catch (error) {
      if (cerrarPorDenegacion(error)) return;
      if (!montado || signal.aborted || version !== secuencia) return;
      estado.errorStatus = error?.status ?? 0;
      estado.error = descripcionError(error, textos); anunciar(estado.error);
    } finally {
      if (montado && version === secuencia) {
        estado.carga = false; pintar();
        if (mas && !estado.error) contenedor.querySelectorAll?.(".marco-participaciones > article")?.[previas]?.focus?.({ preventScroll: true });
        else if (!inicial) enfocar();
      }
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
    estado.enviando = true; estado.error = ""; estado.errorStatus = 0; pintar();
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
      estado.solicitud = { ...recibo, categoria: categoria.categoria, tituloBolsa: bolsa.titulo }; estado.tipo = "solicitud"; estado.revision = false;
      estado.origenSolicitud = "abiertas"; estado.recienRegistrada = true;
      cambiarURL(`solicitud:${recibo.solicitud_ref}`); anunciar(t("registrada"));
    } catch (error) {
      if (cerrarPorDenegacion(error)) return;
      if (error?.status === 400 || error?.status === 409 || error?.status === 422) CLAVES_PENDIENTES.delete(bolsa.convocatoria_ref);
      if (!montado || version !== secuencia || estado.tipo !== "bolsa" || estado.bolsa?.convocatoria_ref !== bolsa.convocatoria_ref) return;
      if (error?.status === 400 || error?.status === 409 || error?.status === 422) estado.bloqueoActo = true;
      estado.errorStatus = error?.status ?? 0;
      estado.error = descripcionError(error, textos, true); anunciar(estado.error);
    } finally {
      ENVIOS_ACTIVOS.delete(bolsa.convocatoria_ref);
      if (montado && (version === secuencia || (estado.tipo === "bolsa" && estado.bolsa?.convocatoria_ref === bolsa.convocatoria_ref))) {
        estado.enviando = false; pintar();
        if (estado.recienRegistrada) enfocar("[data-inscripcion-aviso]");
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
      if ([401, 403].includes(estado.errorStatus)) return;
      const id = leerId(ventana.location.href);
      if (id?.tipo === "bolsa") void cargarBolsa(id.ref);
      else if (id?.tipo === "solicitud") void cargarSolicitud(id.ref);
      else if (id?.tipo === "propias") void propias({ mas: estado.ultimaPropiasMas && Boolean(estado.cursorPropias) });
      else void abiertas({ mas: estado.ultimaAbiertasMas && Boolean(estado.cursor) });
    } else if (accion === "bolsa" && REFERENCIA.test(ref ?? "")) { cambiarURL(`convocatoria:${ref}`); void cargarBolsa(ref); }
    else if (accion === "solicitud" && REFERENCIA.test(ref ?? "")) {
      estado.origenSolicitud = estado.tipo === "propias" ? "propias" : "abiertas";
      cambiarURL(`solicitud:${ref}`); void cargarSolicitud(ref);
    }
    else if (accion === "propias") { cambiarURL("mis-solicitudes"); void propias(); }
    else if (accion === "actualizar-propias") void propias();
    else if (accion === "actualizar-ficha" && estado.bolsa?.convocatoria_ref) void cargarBolsa(estado.bolsa.convocatoria_ref);
    else if (accion === "mas" && estado.cursor) void abiertas({ mas: true });
    else if (accion === "mas-propias" && estado.cursorPropias) void propias({ mas: true });
    else if (accion === "revisar" && estado.bolsa?.puede_iniciar && !estado.categoriaRef) {
      estado.faltaCategoria = true; pintar(); enfocar("[data-inscripcion-categoria]");
    }
    else if (accion === "revisar" && estado.bolsa?.puede_iniciar && estado.categoriaRef) {
      estado.revision = true; pintar();
      contenedor.querySelector?.("[data-inscripcion-revision]")?.focus?.({ preventScroll: true });
    }
    else if (accion === "corregir") {
      estado.revision = false; pintar(); enfocar("h3[tabindex]");
    }
    else if (accion === "confirmar") void confirmar();
  }

  function cambiar(evento) {
    if (evento.target?.matches?.("[data-inscripcion-categoria]") && estado.tipo === "bolsa") {
      if (estado.revision || CLAVES_PENDIENTES.has(estado.bolsa?.convocatoria_ref)) return;
      const ref = evento.target.value;
      estado.categoriaRef = estado.bolsa?.categorias.some((c) => c.categoria_ref === ref) ? ref : "";
      if (estado.categoriaRef) estado.faltaCategoria = false;
      pintar(); contenedor.querySelector?.("[data-inscripcion-categoria]")?.focus?.({ preventScroll: true });
      return;
    }
    const casilla = evento.target?.closest?.("[data-inscripcion-requisito]");
    if (!casilla || !contenedor.contains(casilla) || estado.tipo !== "bolsa" || estado.revision) return;
    const codigo = casilla.dataset.inscripcionRequisito;
    if (!estado.bolsa?.requisitos.some((r) => r.codigo === codigo)) return;
    if (casilla.checked) estado.declaraciones.add(codigo);
    else estado.declaraciones.delete(codigo);
  }

  function buscarCategoria(evento) {
    if (!evento.target?.matches?.("[data-inscripcion-buscar-categoria]") || estado.tipo !== "bolsa"
      || estado.revision || CLAVES_PENDIENTES.has(estado.bolsa?.convocatoria_ref)) return;
    estado.filtroCategoria = String(evento.target.value ?? "").slice(0, 120);
    if (estado.categoriaRef && !estado.bolsa.categorias.some((c) => c.categoria_ref === estado.categoriaRef
      && textoBuscable(c.categoria).includes(textoBuscable(estado.filtroCategoria.trim())))) estado.categoriaRef = "";
    const posicion = evento.target.selectionStart;
    pintar();
    const nuevo = contenedor.querySelector?.("[data-inscripcion-buscar-categoria]");
    nuevo?.focus?.({ preventScroll: true });
    if (Number.isInteger(posicion)) nuevo?.setSelectionRange?.(posicion, posicion);
  }

  contenedor.addEventListener("click", pulsar);
  contenedor.addEventListener("change", cambiar);
  contenedor.addEventListener("input", buscarCategoria);
  pintar();
  (async () => {
    try {
      try { textos = await cargarTextos("bolsa-inscripcion-aspirante", { idioma }); }
      catch { textos = await reintentarTextos("bolsa-inscripcion-aspirante", { idioma }); }
      if (!montado) return;
      clienteEfectivo ??= crearClienteInscripcionBolsa({ fetchImpl, idioma: textos.idioma });
      const id = leerId(ventana.location.href);
      if (id?.tipo === "bolsa") await cargarBolsa(id.ref, { inicial: true });
      else if (id?.tipo === "solicitud") await cargarSolicitud(id.ref, { inicial: true });
      else if (id?.tipo === "propias") await propias({ inicial: true });
      else await abiertas({ inicial: true });
    } catch {
      if (!montado) return;
      contenedor.innerHTML = `<section class="panel" role="alert"><div class="panel-contenido">${esc(textoBase("error"))}</div></section>`;
      anunciar(textoBase("error"));
    }
  })();
  return Object.freeze({ destruir() { montado = false; ++secuencia; consulta?.abort();
    contenedor.removeEventListener("click", pulsar); contenedor.removeEventListener("change", cambiar);
    contenedor.removeEventListener("input", buscarCategoria);
    contenedor.replaceChildren(); } });
}
