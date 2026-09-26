/**
 * Bandeja de solicitudes de participación (Selección) para RRHH: convocatoria,
 * lista paginada por cursor con filtro por turno, y ficha con datos, requisitos
 * declarados, méritos, autobaremo del servicio e historia. Usa los componentes
 * visuales de modulos/solicitudes (solicitudes.css). No decide nada: muestra
 * lo que devuelve el servidor.
 */
import { crearTraductorSeleccion, textoErrorSeleccion } from "./i18n.js?v=20260926-convoca-f1-v2";
import { referenciaCopiableTraducida } from "../../portal-justificante.js";

const TAMANO_PAGINA = 25;
const escapar = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
const formatoFecha = new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" });
const formatoDia = new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeZone: "UTC" });
const formatoNumero = new Intl.NumberFormat("es-ES", { maximumFractionDigits: 3 });
const fecha = (valor, t) => valor ? formatoFecha.format(new Date(valor)) : t("sin_dato");
const dia = (valor, t) => /^\d{4}-\d\d-\d\d$/u.test(valor || "") ? formatoDia.format(new Date(`${valor}T00:00:00Z`)) : t("sin_dato");
const puntos = (valor, t) => valor === null || valor === undefined ? t("sin_dato") : formatoNumero.format(Number(valor));
const textoOpcional = (clave, t) => { try { return t(clave); } catch { return ""; } };
const estadoTexto = (estado, t) => textoOpcional(`estado_${estado}`, t) || t("estado_otro");
const ayuda = (t) => `<details class="solicitudes-ayuda"><summary aria-label="${escapar(t("ayuda"))}">?</summary><p>${escapar(t("ayuda_texto"))}</p></details>`;

function cabecera(t) {
  return `<header class="solicitudes-cabecera"><p class="solicitudes-sobrelinea">${escapar(t("sobrelinea"))}</p><h2>${escapar(t("titulo"))}</h2><p>${escapar(t("descripcion"))}</p></header>`;
}

function errorEstado(t, codigo) {
  return `<div class="solicitudes-estado solicitudes-estado--error" role="alert"><strong>${escapar(textoErrorSeleccion(t, codigo))}</strong><button type="button" class="boton-secundario" data-seleccion-reintentar>${escapar(t("reintentar"))}</button></div>`;
}

function filtros(t, estado) {
  const turnos = [...new Set(estado.filas.map((fila) => fila.turno).filter(Boolean))];
  return `<form class="solicitudes-filtros" data-seleccion-filtros><label>${escapar(t("convocatoria"))}<select name="convocatoria"><option value="">${escapar(t("convocatoria_elegir"))}</option>${estado.convocatorias.map((item) => `<option value="${escapar(item.convocatoria_ref)}"${item.convocatoria_ref === estado.convocatoriaRef ? " selected" : ""}>${escapar(item.titulo)}</option>`).join("")}</select></label><label>${escapar(t("turno"))}<select name="turno"${turnos.length ? "" : " disabled"}><option value="">${escapar(t("todos_turnos"))}</option>${turnos.map((turno) => `<option value="${escapar(turno)}"${turno === estado.turno ? " selected" : ""}>${escapar(turno)}</option>`).join("")}</select></label></form>`;
}

function tabla(t, estado) {
  if (!estado.convocatoriaRef) return `<p class="solicitudes-vacio">${escapar(t("elige_convocatoria"))}</p>`;
  if (estado.fase === "cargando") return `<p class="solicitudes-vacio" role="status">${escapar(t("cargando"))}</p>`;
  if (estado.fase === "error") return errorEstado(t, estado.error);
  if (!estado.filas.length) return `<p class="solicitudes-vacio">${escapar(t("sin_solicitudes"))}</p>`;
  const visibles = estado.filas.filter((fila) => !estado.turno || fila.turno === estado.turno);
  const cabeceras = ["col_justificante", "col_persona", "col_documento", "col_turno", "col_presentada", "col_puntuacion", "col_accion"];
  const cuerpo = visibles.length ? visibles.map((fila) => `<tr class="solicitudes-fila"><th scope="row">${escapar(fila.numero_justificante || t("sin_dato"))}</th><td>${escapar(fila.nombre_visible || t("sin_dato"))}</td><td>${escapar(fila.documento_parcial || t("sin_dato"))}</td><td>${escapar(fila.turno || t("sin_dato"))}</td><td>${escapar(fecha(fila.presentada_en, t))}</td><td>${escapar(puntos(fila.puntuacion_autobaremo, t))}</td><td><button type="button" data-seleccion-ficha="${escapar(fila.solicitud_ref)}" aria-label="${escapar(t("ver_ficha_aria", { numero: fila.numero_justificante }))}">${escapar(t("ver_ficha"))}</button></td></tr>`).join("")
    : `<tr><td colspan="${cabeceras.length}" class="solicitudes-vacio">${escapar(t("sin_coincidencias"))}</td></tr>`;
  const paginacion = estado.indicePagina > 0 || estado.cursorSiguiente
    ? `<nav class="paginacion-bolsa" aria-label="${escapar(t("paginacion"))}"><button type="button" class="boton-secundario" data-seleccion-pagina="-1"${estado.indicePagina > 0 ? "" : " disabled"}>${escapar(t("pagina_anterior"))}</button><span>${escapar(t("pagina", { pagina: estado.indicePagina + 1 }))}</span><button type="button" class="boton-secundario" data-seleccion-pagina="1"${estado.cursorSiguiente ? "" : " disabled"}>${escapar(t("pagina_siguiente"))}</button></nav>` : "";
  return `<div class="solicitudes-tabla-wrap" role="region" tabindex="0" aria-label="${escapar(t("region_tabla"))}"><table class="solicitudes-tabla"><caption>${escapar(t("tabla"))}</caption><thead><tr>${cabeceras.map((clave) => `<th scope="col">${escapar(t(clave))}</th>`).join("")}</tr></thead><tbody>${cuerpo}</tbody></table></div>${paginacion}`;
}

function datosPersonales(t, datos) {
  const direccion = datos.direccion && typeof datos.direccion === "object"
    ? ["via", "codigo_postal", "municipio", "provincia"].map((campo) => datos.direccion[campo]).filter((valor) => typeof valor === "string" && valor).join(", ") : "";
  const campos = [["nombre", datos.nombre], ["apellidos", datos.apellidos], ["documento_identidad", datos.documento_identidad],
    ["fecha_nacimiento", dia(datos.fecha_nacimiento, t)], ["nacionalidad", datos.nacionalidad], ["correo", datos.correo],
    ["telefono", datos.telefono], ["direccion", direccion]];
  return `<dl>${campos.map(([clave, valor]) => `<div><dt>${escapar(t(clave))}</dt><dd>${escapar(typeof valor === "string" && valor ? valor : t("sin_dato"))}</dd></div>`).join("")}</dl>`;
}

function ficha(t, estado) {
  const volver = `<button type="button" class="solicitudes-volver" data-seleccion-volver>${escapar(t("volver"))}</button>`;
  if (estado.fichaFase === "cargando") return `${volver}<p class="solicitudes-vacio" role="status">${escapar(t("cargando_ficha"))}</p>`;
  if (estado.fichaFase === "error") return `${volver}${errorEstado(t, estado.error)}`;
  const dato = estado.ficha;
  const requisitos = dato.requisitos.length ? `<div class="solicitudes-tabla-wrap"><table class="solicitudes-tabla"><caption>${escapar(t("ficha_requisitos"))}</caption><thead><tr>${["requisito", "declaracion", "procedencia", "fecha_referencia"].map((clave) => `<th scope="col">${escapar(t(clave))}</th>`).join("")}</tr></thead><tbody>${dato.requisitos.map((item) => `<tr><th scope="row">${escapar(item.titulo)}</th><td>${escapar(textoOpcional(`requisito_${item.estado}`, t) || t("sin_dato"))}</td><td>${escapar(textoOpcional(`procedencia_${item.procedencia}`, t) || t("sin_dato"))}</td><td>${escapar(dia(item.fecha_referencia, t))}</td></tr>`).join("")}</tbody></table></div>` : `<p class="solicitudes-vacio">${escapar(t("ficha_sin_requisitos"))}</p>`;
  const meritos = dato.meritos.length ? `<div class="solicitudes-tabla-wrap"><table class="solicitudes-tabla"><caption>${escapar(t("ficha_meritos"))}</caption><thead><tr>${["merito", "cantidad", "puntos"].map((clave) => `<th scope="col">${escapar(t(clave))}</th>`).join("")}</tr></thead><tbody>${dato.meritos.map((item) => `<tr><th scope="row">${escapar(item.titulo || item.descripcion || t("sin_dato"))}${item.titulo && item.descripcion ? `<small>${escapar(item.descripcion)}</small>` : ""}</th><td>${escapar(item.cantidad === null ? t("sin_dato") : `${formatoNumero.format(Number(item.cantidad))} ${item.unidad}`)}</td><td>${escapar(puntos(item.puntos, t))}</td></tr>`).join("")}</tbody></table></div>` : `<p class="solicitudes-vacio">${escapar(t("ficha_sin_meritos"))}</p>`;
  const historia = dato.historia.length ? `<ol>${dato.historia.map((hito) => `<li><time${hito.en ? ` datetime="${escapar(hito.en)}"` : ""}>${escapar(fecha(hito.en, t))}</time><strong>${escapar(textoOpcional(`historia_${hito.tipo}`, t) ? t(`historia_${hito.tipo}`, { version: hito.version ?? "" }) : t("historia_otro", { version: hito.version ?? "" }))}</strong></li>`).join("")}</ol>` : `<p>${escapar(t("ficha_sin_historia"))}</p>`;
  const referencia = referenciaCopiableTraducida(dato.solicitud_ref, escapar, t, t("referencia_aria", { numero: dato.numero_justificante }));
  return `${volver}<h3 id="seleccion-ficha-titulo" tabindex="-1">${escapar(t("ficha_titulo", { numero: dato.numero_justificante }))}</h3>
    <div class="solicitudes-ficha"><dl>
      <div><dt>${escapar(t("convocatoria"))}</dt><dd>${escapar(dato.convocatoria_titulo || t("sin_dato"))}</dd></div>
      <div><dt>${escapar(t("estado"))}</dt><dd><span class="solicitudes-chip">${escapar(estadoTexto(dato.estado, t))}</span></dd></div>
      <div><dt>${escapar(t("presentada_en"))}</dt><dd>${escapar(fecha(dato.presentada_en, t))}</dd></div>
      <div><dt>${escapar(t("turno"))}</dt><dd>${escapar(dato.turno || t("sin_dato"))}</dd></div>
      <div><dt>${escapar(t("puntuacion_total"))}</dt><dd>${escapar(puntos(dato.puntuacion_autobaremo, t))}</dd></div>
      <div><dt>${escapar(t("justificante_registrado"))}</dt><dd>${referencia}</dd></div>
    </dl><div class="solicitudes-ficha-descripcion"><h4>${escapar(t("ficha_datos"))}</h4>${datosPersonales(t, dato.datos)}</div></div>
    <section class="solicitudes-historial" aria-label="${escapar(t("ficha_requisitos"))}"><h4>${escapar(t("ficha_requisitos"))}</h4>${requisitos}</section>
    <section class="solicitudes-historial" aria-label="${escapar(t("ficha_meritos"))}"><h4>${escapar(t("ficha_meritos"))}</h4>${meritos}</section>
    <section class="solicitudes-historial" aria-label="${escapar(t("ficha_historia"))}"><h4>${escapar(t("ficha_historia"))}</h4>${historia}</section>`;
}

/** Presentación pura de la vista a partir de su estado. */
export function renderizarSolicitudesSeleccion(estado, t = crearTraductorSeleccion()) {
  if (estado.faseConvocatorias === "cargando") return `<section class="solicitudes-modulo">${cabecera(t)}<p class="solicitudes-vacio" role="status">${escapar(t("cargando"))}</p></section>`;
  if (estado.faseConvocatorias === "error") return `<section class="solicitudes-modulo">${cabecera(t)}${errorEstado(t, estado.error)}</section>`;
  if (!estado.convocatorias.length) return `<section class="solicitudes-modulo">${cabecera(t)}<p class="solicitudes-vacio">${escapar(t("sin_convocatorias"))}</p></section>`;
  const cuerpo = estado.ficha || estado.fichaFase ? ficha(t, estado) : `${filtros(t, estado)}${tabla(t, estado)}`;
  return `<section class="solicitudes-modulo">${cabecera(t)}<section class="solicitudes-panel panel"><header class="cabecera-panel solicitudes-panel-cabecera"><div><h3>${escapar(t("titulo"))}</h3></div>${ayuda(t)}</header><div class="cuerpo-panel solicitudes-panel-cuerpo">${cuerpo}</div></section></section>`;
}

export function montarVistaSolicitudesSeleccion({ raiz, cliente, anunciar = () => {}, t = crearTraductorSeleccion() } = {}) {
  if (!raiz?.addEventListener || typeof cliente?.consultar !== "function") throw new TypeError("vista de Selección no disponible");
  const controlador = new AbortController();
  const signal = controlador.signal;
  let activa = true;
  const estado = { faseConvocatorias: "cargando", convocatorias: [], convocatoriaRef: "", turno: "", fase: "", error: "",
    filas: [], paginas: [""], indicePagina: 0, cursorSiguiente: "", ficha: null, fichaFase: "", fichaOrigen: "" };
  const pintar = (enfocar = "") => {
    if (!activa) return;
    raiz.innerHTML = renderizarSolicitudesSeleccion(estado, t);
    if (enfocar) raiz.querySelector(enfocar)?.focus?.();
  };
  async function cargarConvocatorias() {
    estado.faseConvocatorias = "cargando"; pintar();
    try {
      estado.convocatorias = await cliente.convocatorias({ signal });
      estado.faseConvocatorias = "listo";
      if (estado.convocatorias.length === 1) { estado.convocatoriaRef = estado.convocatorias[0].convocatoria_ref; await cargarPagina(0); return; }
    } catch (error) { if (signal.aborted) return; estado.faseConvocatorias = "error"; estado.error = error?.codigo || "respuesta_incompatible"; }
    pintar();
  }
  async function cargarPagina(indice) {
    estado.fase = "cargando"; estado.indicePagina = indice; pintar();
    try {
      const pagina = await cliente.consultar({ convocatoriaRef: estado.convocatoriaRef, cursor: estado.paginas[indice], limite: TAMANO_PAGINA }, { signal });
      if (!activa) return;
      estado.filas = [...pagina.solicitudes]; estado.cursorSiguiente = pagina.cursorSiguiente; estado.fase = "listo";
      if (pagina.cursorSiguiente) estado.paginas[indice + 1] = pagina.cursorSiguiente;
      pintar();
      anunciar(t("anuncio_lista", { total: estado.filas.length }), "informacion");
    } catch (error) {
      if (signal.aborted) return;
      estado.fase = "error"; estado.error = error?.codigo || "respuesta_incompatible"; pintar();
    }
  }
  async function abrirFicha(ref) {
    estado.fichaFase = "cargando"; estado.fichaOrigen = ref; estado.ficha = null; pintar("[data-seleccion-volver]");
    try {
      estado.ficha = await cliente.detalle(ref, { signal });
      estado.fichaFase = "";
      pintar("#seleccion-ficha-titulo");
      anunciar(t("anuncio_ficha", { numero: estado.ficha.numero_justificante }), "informacion");
    } catch (error) {
      if (signal.aborted) return;
      estado.fichaFase = "error"; estado.error = error?.codigo || "respuesta_incompatible"; pintar("[data-seleccion-volver]");
    }
  }
  const alCambiar = (evento) => {
    const formulario = evento.target?.closest?.("[data-seleccion-filtros]");
    if (!formulario) return;
    if (evento.target.name === "convocatoria") {
      estado.convocatoriaRef = evento.target.value; estado.turno = ""; estado.paginas = [""]; estado.filas = []; estado.cursorSiguiente = "";
      if (estado.convocatoriaRef) void cargarPagina(0); else pintar();
    } else if (evento.target.name === "turno") {
      estado.turno = evento.target.value; pintar("[data-seleccion-filtros] select[name=turno]");
    }
  };
  const alPulsar = (evento) => {
    const objetivo = evento.target;
    const botonFicha = objetivo?.closest?.("[data-seleccion-ficha]");
    if (botonFicha) { void abrirFicha(botonFicha.dataset.seleccionFicha); return; }
    if (objetivo?.closest?.("[data-seleccion-volver]")) {
      const origen = estado.fichaOrigen;
      estado.ficha = null; estado.fichaFase = ""; pintar();
      [...raiz.querySelectorAll("[data-seleccion-ficha]")].find((boton) => boton.dataset.seleccionFicha === origen)?.focus?.();
      return;
    }
    const pagina = objetivo?.closest?.("[data-seleccion-pagina]");
    if (pagina) { void cargarPagina(Math.max(0, estado.indicePagina + Number(pagina.dataset.seleccionPagina))); return; }
    if (objetivo?.closest?.("[data-seleccion-reintentar]")) {
      if (estado.faseConvocatorias === "error") void cargarConvocatorias();
      else if (estado.fichaFase === "error") void abrirFicha(estado.fichaOrigen);
      else void cargarPagina(estado.indicePagina);
    }
  };
  const alEnviar = (evento) => { if (evento.target?.closest?.("[data-seleccion-filtros]")) evento.preventDefault(); };
  raiz.addEventListener("change", alCambiar);
  raiz.addEventListener("click", alPulsar);
  raiz.addEventListener("submit", alEnviar);
  void cargarConvocatorias();
  return Object.freeze({
    desmontar() {
      if (!activa) return;
      activa = false; controlador.abort();
      raiz.removeEventListener("change", alCambiar); raiz.removeEventListener("click", alPulsar); raiz.removeEventListener("submit", alEnviar);
      raiz.replaceChildren?.();
    },
  });
}
