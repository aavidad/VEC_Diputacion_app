import { crearClienteEquipoInformesCronosHTTP, ErrorClienteEquipoInformesCronos } from "./cliente-equipo-informes-c10.js";
import { crearTraductorEquipoInformesCronos } from "./i18n-equipo-informes-c10.js";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

const VISTAS = new Set(["presencia", "incidencias"]);
const FILTROS = new Set(["total", "indeterminado", "registrado", "incidencias"]);
const ESC = (v) => String(v ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
const fechaValida = (valor) => {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(valor)) return false;
  const [a,m,d] = valor.split("-").map(Number); const f = new Date(Date.UTC(a,m-1,d));
  return f.getUTCFullYear()===a && f.getUTCMonth()===m-1 && f.getUTCDate()===d;
};
const periodoAdmitido = (desde, hasta) => fechaValida(desde) && fechaValida(hasta)
  && desde <= hasta && Date.parse(`${hasta}T12:00:00Z`) - Date.parse(`${desde}T12:00:00Z`) <= 31 * 86400000;
const fechaVisible = (f, locale) => new Intl.DateTimeFormat(locale, { day: "2-digit", month: "2-digit", year: "numeric", timeZone: "UTC" }).format(new Date(`${f}T12:00:00Z`));
const esIndeterminado = (f) => ["indeterminado", "no_evaluado"].includes(f.estado);
const esRegistrado = (f) => ["entrada_registrada", "registrado"].includes(f.estado);
const esIncidencia = (f) => f.estado === "registro_incompleto";
function visibles(datos, filtro) {
  if (!datos) return [];
  if (filtro === "indeterminado") return datos.filas.filter(esIndeterminado);
  if (filtro === "registrado") return datos.filas.filter(esRegistrado);
  if (filtro === "incidencias") return datos.filas.filter(esIncidencia);
  return datos.filas;
}
function estadoTexto(f, t) {
  const clave = `estado_${f.estado}`;
  const texto = t(clave);
  return f.causa ? `${texto}. ${t(`causa_${f.causa}`)}` : texto;
}

export function renderizarEquipoInformesCronos({ estado = "seleccion", datos = null, equipoNombre = "", vista = "presencia", desde = "", hasta = "", filtro = "total", pagina = 1,
  exportacionesPermitidas = [], informeSeleccionado = "", descargando = false, mensaje = "", errorPeriodo = "", recibo = "", mensajes, locale = LOCALIZACION_ACTUAL } = {}) {
  const t = crearTraductorEquipoInformesCronos(mensajes);
  if (!VISTAS.has(vista) || !FILTROS.has(filtro)) throw new TypeError("vista de equipo inválida");
  const etiquetas = { total: "total", indeterminado: "indeterminado", registrado: "registrado", incidencias: "incidencias_recuento" };
  const datosVisibles = estado === "listo" ? datos : null;
  const filas = visibles(datosVisibles, filtro);
  const informesPermitidos = [...new Set(exportacionesPermitidas)].filter((v) => /^(saldo|movimientos|permisos)\.(pdf|csv)$/u.test(v));
  const informePermitido = informesPermitidos.length > 0;
  const rol = datosVisibles ? `<span class="cronos-estado">${ESC(t(`rol_${datosVisibles.alcance}`))}</span>` : "";
  const cabecera = `<header class="cronos-encabezado"><div><h2 id="cronos-equipo-titulo">${ESC(t("titulo"))}</h2>${equipoNombre ? `<p>${ESC(equipoNombre)}</p>` : ""}</div>${rol}</header>`;
  const selector = `<div class="cronos-selector-paso" role="group" aria-label="${ESC(t("titulo"))}">
    ${["presencia", "incidencias"].map((v) => `<button type="button" class="boton-secundario" data-cronos-vista-equipo="${v}" aria-pressed="${v === vista}">${ESC(t(v))}</button>`).join("")}</div>`;
  const formulario = `<form data-cronos-periodo-equipo class="cronos-formulario"><div class="cronos-campos">
    <div class="campo"><label for="cronos-equipo-desde">${ESC(t("desde"))}</label><input id="cronos-equipo-desde" name="desde" type="date" required value="${ESC(desde)}"></div>
    <div class="campo"><label for="cronos-equipo-hasta">${ESC(t("hasta"))}</label><input id="cronos-equipo-hasta" name="hasta" type="date" required value="${ESC(hasta)}"></div>
    <button type="submit" class="boton-primario">${ESC(t("consultar"))}</button></div>${errorPeriodo ? `<p role="alert" id="cronos-equipo-error-periodo">${ESC(errorPeriodo)}</p>` : ""}</form>`;
  const kpis = datosVisibles ? `<section aria-label="${ESC(t("esta_pagina"))}"><h3>${ESC(t("esta_pagina"))}</h3><div class="rejilla-kpi">
    ${Object.entries(etiquetas).map(([clave, etiqueta]) => `<button type="button" class="tarjeta-kpi" data-cronos-filtro-equipo="${clave}" aria-pressed="${clave === filtro}"><span>${ESC(t(etiqueta))}</span><strong>${datosVisibles.agregado[clave]}</strong></button>`).join("")}</div></section>` : "";
  let contenido = "";
  if (estado === "cargando") contenido = `<p role="status">${ESC(t("cargando"))}</p>`;
  else if (estado === "denegado") contenido = `<p role="alert">${ESC(t("denegado"))}</p>`;
  else if (estado === "error") contenido = `<p role="alert">${ESC(t("error"))}</p><button type="button" class="boton-secundario" data-cronos-reintentar-equipo>${ESC(t("reintentar"))}</button>`;
  else if (estado === "listo") {
    const cabeceras = ["col_persona", "col_fecha", "col_estado", "col_cobertura", "col_hechos"];
    const cuerpo = filas.length ? filas.map((f) => `<tr><th scope="row">${ESC(f.empleado_etiqueta)}</th><td>${ESC(fechaVisible(f.fecha, locale))}</td>
      <td>${ESC(estadoTexto(f, t))}</td><td>${ESC(t(`cobertura_${f.cobertura}`))}</td><td>${f.hechos_registrados}</td></tr>`).join("")
      : `<tr><td colspan="5">${ESC(t("vacio"))}</td></tr>`;
    contenido = `${kpis}<div class="cronos-tabla-contenedor"><table class="cronos-tabla"><caption>${ESC(t(vista))}</caption><thead><tr>${cabeceras.map((k) => `<th scope="col">${ESC(t(k))}</th>`).join("")}</tr></thead><tbody>${cuerpo}</tbody></table></div>
      <p class="cronos-aviso">${ESC(t("no_ausencia"))}</p><nav class="cronos-paginacion" aria-label="${ESC(t("pagina", { numero: pagina }))}">
      <button type="button" class="boton-secundario" data-cronos-pagina-equipo="anterior"${pagina <= 1 ? " disabled" : ""}>${ESC(t("volver"))}</button>
      <span>${ESC(t("pagina", { numero: pagina }))}</span><button type="button" class="boton-secundario" data-cronos-pagina-equipo="siguiente"${!datosVisibles.siguiente_cursor ? " disabled" : ""}>${ESC(t("siguiente"))}</button></nav>`;
  } else contenido = `<p role="status">${ESC(t("seleccione_periodo"))}</p>`;
  const exportar = `<section class="panel cronos-panel" aria-labelledby="cronos-equipo-informes-titulo"><header class="cabecera-panel"><h3 id="cronos-equipo-informes-titulo">${ESC(t("informes"))}</h3></header>
    <div class="cuerpo-panel"><form data-cronos-exportar-equipo><div class="cronos-campos"><div class="campo"><label for="cronos-equipo-informe">${ESC(t("informe_y_formato"))}</label>
    <select id="cronos-equipo-informe" name="informe"${!informePermitido ? " disabled" : ""}>${informesPermitidos.map((v) => { const [tipo, formato] = v.split("."); return `<option value="${v}"${v === informeSeleccionado ? " selected" : ""}>${ESC(t(`informe_${tipo}`))} · ${ESC(t(`formato_${formato}`))}</option>`; }).join("")}</select></div>
    <button type="submit" class="boton-secundario"${!informePermitido || descargando || estado !== "listo" ? " disabled" : ""} aria-describedby="cronos-equipo-exportacion-estado">${ESC(t(descargando ? "descargando" : "descargar"))}</button></div></form>
    <p id="cronos-equipo-exportacion-estado" role="status">${ESC(!informePermitido ? t("exportacion_no_disponible") : mensaje)}</p>
    ${recibo ? `<details><summary>${ESC(t("recibo"))}</summary><p>${ESC(recibo)}</p></details>` : ""}</div></section>`;
  return `<section class="cronos-area" aria-labelledby="cronos-equipo-titulo" data-cronos-equipo-estado="${ESC(estado)}">${cabecera}${selector}
    <section class="panel cronos-panel" aria-labelledby="cronos-equipo-lista-titulo"><header class="cabecera-panel"><h3 id="cronos-equipo-lista-titulo" tabindex="-1">${ESC(t(vista))}</h3></header><div class="cuerpo-panel">${formulario}${contenido}</div></section>${exportar}</section>`;
}

export function montarEquipoInformesCronos({ raiz, equipo, equipoNombre = "", cliente = crearClienteEquipoInformesCronosHTTP(), mensajes,
  locale = LOCALIZACION_ACTUAL, exportacionesPermitidas = [], anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof equipo !== "string" || !equipo || typeof cliente?.consultar !== "function"
    || typeof cliente?.exportar !== "function" || typeof anunciar !== "function" || !Array.isArray(exportacionesPermitidas)) throw new TypeError("montaje de equipo Cronos no disponible");
  const t = crearTraductorEquipoInformesCronos(mensajes);
  const nodo = raiz.ownerDocument.createElement("section"); nodo.dataset.cronosEquipoInformes = ""; raiz.append(nodo);
  let activa = true; let estado = "seleccion"; let vista = "presencia"; let desde = ""; let hasta = ""; let filtro = "total";
  let informeSeleccionado = exportacionesPermitidas[0] || "";
  let datos = null; let mensaje = ""; let errorPeriodo = ""; let recibo = ""; let descargando = false; let consulta = null; let descarga = null; let turno = 0;
  let operacionPendiente = null;
  let cursores = [""]; const urlsDescarga = new Set();
  const dibujar = () => { if (activa) nodo.innerHTML = renderizarEquipoInformesCronos({ estado, datos, equipoNombre, vista, desde, hasta, filtro, pagina: cursores.length,
    exportacionesPermitidas, informeSeleccionado, descargando, mensaje, errorPeriodo, recibo, mensajes, locale }); };
  const cargar = async ({ foco } = {}) => {
    if (!activa || !desde || !hasta) return;
    consulta?.abort(); consulta = new AbortController(); const actual = ++turno; estado = "cargando"; datos = null; filtro = "total"; dibujar();
    try { const resultado = await cliente.consultar({ equipo, vista, desde, hasta, cursor: cursores.at(-1) }, { signal: consulta.signal });
      if (!activa || actual !== turno || consulta.signal.aborted) return; datos = resultado; estado = "listo"; dibujar(); anunciar(t("esta_pagina")); if (foco) nodo.querySelector?.(foco)?.focus?.(); }
    catch (e) { if (!activa || actual !== turno || consulta.signal.aborted) return;
      estado = e instanceof ErrorClienteEquipoInformesCronos && ["acceso_denegado", "autenticacion_requerida"].includes(e.codigo) ? "denegado" : "error";
      datos = null; dibujar(); anunciar(t(estado)); if (foco) nodo.querySelector?.(foco)?.focus?.(); }
  };
  const descargar = async (tipo, formato) => {
    if (!activa || descargando || estado !== "listo" || !exportacionesPermitidas.includes(`${tipo}.${formato}`)) return;
    const seleccion = `${equipo}|${tipo}|${formato}|${desde}|${hasta}`;
    if (!operacionPendiente || operacionPendiente.seleccion !== seleccion) operacionPendiente = { seleccion, clave: globalThis.crypto.randomUUID() };
    descarga?.abort(); const envio = new AbortController(); descarga = envio; descargando = true; mensaje = ""; recibo = ""; dibujar();
    try { const r = await cliente.exportar({ equipo, tipo, formato, desde, hasta, clave_operacion: operacionPendiente.clave }, { signal: envio.signal });
      if (!activa || envio.signal.aborted) return;
      const blob = new Blob([r.bytes], { type: r.tipo }); r.bytes.fill(0);
      const url = URL.createObjectURL(blob); urlsDescarga.add(url); setTimeout(() => { URL.revokeObjectURL(url); urlsDescarga.delete(url); }, 1000);
      const enlace = raiz.ownerDocument.createElement("a"); enlace.href = url; enlace.download = `cronos-informe.${formato}`;
      nodo.append(enlace); enlace.click(); enlace.remove();
      recibo = r.recibo; operacionPendiente = null; mensaje = t("recibo"); anunciar(mensaje);
    } catch (e) { if (!activa || envio.signal.aborted) return;
      mensaje = t(e instanceof ErrorClienteEquipoInformesCronos && e.codigo === "acceso_denegado" ? "exportacion_denegada" : "error_exportar"); anunciar(mensaje);
    } finally { descargando = false; dibujar(); nodo.querySelector?.('[data-cronos-exportar-equipo] button[type="submit"]')?.focus?.(); }
  };
  const alEnviar = (e) => {
    const form = e.target;
    if (form?.matches?.("[data-cronos-periodo-equipo]")) { e.preventDefault(); const d = form.elements.namedItem("desde")?.value; const h = form.elements.namedItem("hasta")?.value;
      if (!periodoAdmitido(d,h)) { desde = d || ""; hasta = h || ""; consulta?.abort(); descarga?.abort(); ++turno; estado = "seleccion"; datos = null;
        errorPeriodo = t("periodo_largo"); dibujar(); nodo.querySelector?.("#cronos-equipo-hasta")?.focus?.(); return; }
      desde = d; hasta = h; errorPeriodo = ""; cursores = [""]; recibo = ""; operacionPendiente = null; descarga?.abort(); void cargar({ foco: "#cronos-equipo-lista-titulo" }); }
    if (form?.matches?.("[data-cronos-exportar-equipo]")) { e.preventDefault(); informeSeleccionado = form.elements.namedItem("informe")?.value || "";
      const [tipo, formato] = informeSeleccionado.split("."); void descargar(tipo, formato); }
  };
  const alPulsar = (e) => {
    const botonVista = e.target?.closest?.("[data-cronos-vista-equipo]"); if (botonVista && VISTAS.has(botonVista.dataset.cronosVistaEquipo)) { vista = botonVista.dataset.cronosVistaEquipo; cursores = [""]; recibo = "";
      if (desde && hasta) void cargar({ foco: `[data-cronos-vista-equipo="${vista}"]` }); else { dibujar(); nodo.querySelector?.(`[data-cronos-vista-equipo="${vista}"]`)?.focus?.(); } return; }
    const botonFiltro = e.target?.closest?.("[data-cronos-filtro-equipo]"); if (botonFiltro && estado === "listo" && FILTROS.has(botonFiltro.dataset.cronosFiltroEquipo)) { filtro = botonFiltro.dataset.cronosFiltroEquipo; dibujar(); nodo.querySelector?.(`[data-cronos-filtro-equipo="${filtro}"]`)?.focus?.(); return; }
    const pagina = e.target?.closest?.("[data-cronos-pagina-equipo]"); if (pagina && estado === "listo") {
      if (pagina.dataset.cronosPaginaEquipo === "siguiente" && datos?.siguiente_cursor) { cursores.push(datos.siguiente_cursor); void cargar({ foco: '[data-cronos-pagina-equipo="anterior"]' }); }
      else if (pagina.dataset.cronosPaginaEquipo === "anterior" && cursores.length > 1) { cursores.pop(); void cargar({ foco: '[data-cronos-pagina-equipo="siguiente"]' }); } return;
    }
    if (e.target?.closest?.("[data-cronos-reintentar-equipo]") && estado === "error") void cargar();
  };
  const alCambiar = (e) => {
    const campo = e.target;
    if (campo?.matches?.("#cronos-equipo-informe")) { informeSeleccionado = campo.value; return; }
    if (!campo?.matches?.('[data-cronos-periodo-equipo] input[type="date"]')) return;
    const form = campo.form; const d = form?.elements.namedItem("desde")?.value || ""; const h = form?.elements.namedItem("hasta")?.value || "";
    if (d === desde && h === hasta) return;
    desde = d; hasta = h; consulta?.abort(); descarga?.abort(); ++turno; estado = "seleccion"; datos = null; recibo = "";
    errorPeriodo = ""; operacionPendiente = null; cursores = [""]; dibujar(); nodo.querySelector?.(`#${campo.id}`)?.focus?.();
  };
  nodo.addEventListener("submit", alEnviar); nodo.addEventListener("click", alPulsar); nodo.addEventListener("change", alCambiar); dibujar();
  const desmontar = () => { if (!activa) return; activa = false; ++turno; consulta?.abort(); descarga?.abort(); for (const url of urlsDescarga) URL.revokeObjectURL(url); urlsDescarga.clear();
    nodo.removeEventListener("submit", alEnviar); nodo.removeEventListener("click", alPulsar); nodo.removeEventListener("change", alCambiar); nodo.remove?.(); };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, consultar: (periodo) => {
    desde = periodo?.desde || ""; hasta = periodo?.hasta || ""; cursores = [""];
    if (!periodoAdmitido(desde,hasta)) { errorPeriodo = t("periodo_largo"); estado = "seleccion"; datos = null; dibujar(); return Promise.resolve(); }
    errorPeriodo = ""; return cargar();
  } });
}
