import { crearTraductorPruebas } from "./i18n.js";

const ESTADOS = new Set(["no_configurado", "cargando", "vacio", "denegado", "error", "disponible"]);
const ESTADOS_FILA = new Set(["pendiente", "programada", "realizada", "suspendida", "borrador", "en_revision", "aprobada", "publicada", "registrado", "validado"]);
const ESCAPAR = (valor) => String(valor ?? "").replace(/[&<>"']/gu, (caracter) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[caracter]);
const texto = (valor, t, etiquetaVacia = "sin_dato") => valor === null || valor === undefined || String(valor).trim() === "" ? t(etiquetaVacia) : String(valor);
const lista = (valor) => Array.isArray(valor) ? valor : [];

function fecha(valor, t) {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}(?:T.*)?$/u.test(valor)) return t("sin_dato");
  const instante = new Date(valor);
  return Number.isNaN(instante.valueOf()) ? t("sin_dato") : new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeZone: "UTC" }).format(instante);
}

function estadoFila(valor, t) {
  const clave = ESTADOS_FILA.has(valor) ? valor : "sin_estado";
  const tono = ["realizada", "aprobada", "publicada", "validado"].includes(clave) ? "exito" : ["en_revision", "registrado"].includes(clave) ? "violeta" : ["suspendida"].includes(clave) ? "peligro" : "neutro";
  return `<span class="estado-chip ${tono}">${ESCAPAR(t(clave))}</span>`;
}

function tabla(t, titulo, cabeceras, filas, vacio) {
  if (!filas.length) return `<p class="pruebas-vacio">${ESCAPAR(t(vacio))}</p>`;
  return `<div class="tabla-contenedor pruebas-tabla" role="region" tabindex="0" aria-label="${ESCAPAR(t(titulo))}"><table class="tabla-datos"><caption>${ESCAPAR(t(titulo))}</caption><thead><tr>${cabeceras.map((clave) => `<th scope="col">${ESCAPAR(t(clave))}</th>`).join("")}</tr></thead><tbody>${filas.join("")}</tbody></table></div>`;
}

function panel(t, titulo, subtitulo, contenido, clase = "") {
  return `<section class="panel pruebas-panel ${clase}" aria-labelledby="pruebas-${titulo}"><div class="cabecera-panel"><div><h3 id="pruebas-${titulo}">${ESCAPAR(t(titulo))}</h3><p>${ESCAPAR(t(subtitulo))}</p></div></div><div class="cuerpo-panel">${contenido}</div></section>`;
}

function detalleDisponible(datos, t) {
  const pruebas = lista(datos.pruebas).map((item) => `<tr><th scope="row">${ESCAPAR(texto(item?.nombre, t))}</th><td>${ESCAPAR(fecha(item?.fecha, t))}</td><td>${estadoFila(item?.estado, t)}</td></tr>`);
  const resultados = lista(datos.resultados).map((item) => `<tr><th scope="row">${ESCAPAR(texto(item?.prueba, t))}</th><td>${ESCAPAR(texto(item?.aspirante, t))}</td><td>${ESCAPAR(texto(item?.resultado, t, "sin_resultado"))}</td><td>${estadoFila(item?.estado, t)}</td></tr>`);
  const actas = lista(datos.actas).map((item) => `<tr><th scope="row">${ESCAPAR(texto(item?.prueba, t))}</th><td>${ESCAPAR(texto(item?.referencia, t))}</td><td>${ESCAPAR(fecha(item?.fecha, t))}</td><td>${estadoFila(item?.estado, t)}</td></tr>`);
  return `<div class="pruebas-rejilla">${panel(t, "pruebas", "pruebas_subtitulo", tabla(t, "pruebas", ["prueba", "fecha", "estado"], pruebas, "sin_pruebas"))}${panel(t, "actas", "actas_subtitulo", tabla(t, "actas", ["prueba", "acta", "fecha", "estado"], actas, "sin_actas"))}${panel(t, "resultados", "resultados_subtitulo", tabla(t, "resultados", ["prueba", "aspirante", "resultado", "estado"], resultados, "sin_resultados"), "pruebas-panel--ancho")}</div>`;
}

/** Modelo de entrada ya autorizado: { estado, convocatoria?, pruebas?, resultados?, actas? }. Sin red ni efectos. */
export function renderizarVistaPruebas(datos = {}, mensajes) {
  const t = crearTraductorPruebas(mensajes);
  const estado = ESTADOS.has(datos?.estado) ? datos.estado : "no_configurado";
  const disponibles = estado === "disponible";
  const convocatoria = disponibles ? datos.convocatoria : undefined;
  const identificada = convocatoria && (convocatoria.nombre || convocatoria.referencia);
  const mensajeEstado = estado === "disponible" && ![...lista(datos.pruebas), ...lista(datos.resultados), ...lista(datos.actas)].length ? "vacio" : estado;
  const nombre = identificada ? texto(convocatoria.nombre, t, "convocatoria_pendiente") : t("convocatoria_pendiente");
  const referencia = identificada ? texto(convocatoria.referencia, t) : t("sin_dato");
  const estadoConvocatoria = disponibles ? estadoFila(convocatoria?.estado, t) : `<span class="estado-chip neutro">${ESCAPAR(t(estado === "no_configurado" ? "no_configurado" : "sin_estado"))}</span>`;
  const tono = mensajeEstado === "disponible" ? "exito" : ["error", "denegado"].includes(mensajeEstado) ? "peligro" : "neutro";
  const sinDatos = (clave) => `<p class="pruebas-vacio">${ESCAPAR(t(mensajeEstado === "vacio" ? clave : estado === "no_configurado" ? "datos_sin_fuente" : `${mensajeEstado}_detalle`))}</p>`;
  const contenido = disponibles && mensajeEstado === "disponible" ? detalleDisponible(datos, t) : `<div class="pruebas-rejilla">
    ${panel(t, "pruebas", "pruebas_subtitulo", sinDatos("sin_pruebas"))}
    ${panel(t, "actas", "actas_subtitulo", sinDatos("sin_actas"))}
  </div>`;
  return `<section class="pruebas-vista" data-seleccion-pruebas data-estado="${mensajeEstado}">
    <header class="pruebas-cabecera"><div>
      <p class="pruebas-sobrelinea">${ESCAPAR(t("sobrelinea"))}</p>
      <h2>${ESCAPAR(t("titulo"))}</h2><p>${ESCAPAR(t("descripcion"))}</p>
    </div><details class="pruebas-ayuda"><summary aria-label="${ESCAPAR(t("ayuda"))}">${ESCAPAR(t("ayuda_simbolo"))}</summary><p>${ESCAPAR(t("ayuda_texto"))}</p></details></header>
    <div class="pruebas-franja" role="status"><span class="estado-chip ${tono}">${ESCAPAR(t(mensajeEstado))}</span><span>${ESCAPAR(t(`${mensajeEstado}_detalle`))}</span></div>
    <div class="pruebas-contexto"><span>${ESCAPAR(t("convocatoria"))}</span><strong>${ESCAPAR(nombre)}</strong><span>${ESCAPAR(t("referencia"))}: ${ESCAPAR(referencia)}</span><span>${ESCAPAR(t("estado_convocatoria"))}: ${estadoConvocatoria}</span></div>
    ${contenido}
    <div class="pruebas-acciones"><p>${ESCAPAR(t("acciones_pendientes"))}</p>
      <button type="button" class="boton-secundario" disabled title="${ESCAPAR(t("acciones_pendientes"))}">${ESCAPAR(t("registrar_resultados"))}</button>
      <button type="button" class="boton-secundario" disabled title="${ESCAPAR(t("acciones_pendientes"))}">${ESCAPAR(t("adjuntar_acta"))}</button>
    </div>
  </section>`;
}

/** Firma para el integrador: montarVistaPruebas({ raiz, datos?, mensajes?, registrarDesmontar? }) → { actualizar, desmontar }. */
export function montarVistaPruebas({ raiz, datos = {}, mensajes, registrarDesmontar } = {}) {
  if (!raiz?.replaceChildren || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("raíz de pruebas no disponible");
  let activa = true;
  const pintar = (siguiente) => { if (activa) raiz.innerHTML = renderizarVistaPruebas(siguiente, mensajes); };
  pintar(datos);
  const actualizar = (siguiente) => pintar(siguiente);
  const desmontar = () => { if (!activa) return; activa = false; raiz.replaceChildren(); };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ actualizar, desmontar });
}
