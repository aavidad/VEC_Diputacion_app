import { crearTraductorPruebas } from "./i18n.js";

const ESTADOS = new Set(["no_configurado", "cargando", "vacio", "denegado", "error", "disponible"]);
const ESTADOS_FILA = new Set(["pendiente", "programada", "realizada", "suspendida", "borrador", "en_revision", "aprobada", "publicada", "registrado", "validado"]);
const ESCAPAR = (valor) => String(valor ?? "").replace(/[&<>"']/gu, (caracter) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[caracter]);
const texto = (valor, t, etiquetaVacia = "sin_dato") => valor === null || valor === undefined || String(valor).trim() === "" ? t(etiquetaVacia) : String(valor);
const lista = (valor) => Array.isArray(valor) ? valor : [];
const idPrueba = (item) => item?.id === null || item?.id === undefined ? "" : String(item.id).trim();
const idVinculo = (item) => item?.prueba_id === null || item?.prueba_id === undefined ? "" : String(item.prueba_id).trim();

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

function detalleDisponible(datos, t, seleccionId) {
  const seleccion = lista(datos.pruebas).find((item) => idPrueba(item) && idPrueba(item) === seleccionId);
  const actasVinculadas = seleccion ? lista(datos.actas).filter((item) => idVinculo(item) === seleccionId) : lista(datos.actas);
  const resultadosVinculados = seleccion ? lista(datos.resultados).filter((item) => idVinculo(item) === seleccionId) : lista(datos.resultados);
  const pruebas = lista(datos.pruebas).map((item) => {
    const id = idPrueba(item);
    const nombre = texto(item?.nombre, t);
    const activo = Boolean(seleccion && id === seleccionId);
    const enlace = id ? `<button type="button" class="pruebas-enlace" data-prueba-detalle="${ESCAPAR(id)}" aria-label="${ESCAPAR(t("ver_detalle"))} ${ESCAPAR(nombre)}" aria-pressed="${activo}">${ESCAPAR(nombre)}</button>` : ESCAPAR(nombre);
    return `<tr${activo ? ' data-seleccionada="true"' : ""}><th scope="row">${enlace}</th><td>${ESCAPAR(fecha(item?.fecha, t))}</td><td>${estadoFila(item?.estado, t)}</td></tr>`;
  });
  const resultados = resultadosVinculados.map((item) => `<tr><th scope="row">${ESCAPAR(texto(item?.prueba, t))}</th><td>${ESCAPAR(texto(item?.aspirante, t))}</td><td>${ESCAPAR(texto(item?.resultado, t, "sin_resultado"))}</td><td>${estadoFila(item?.estado, t)}</td></tr>`);
  const actas = actasVinculadas.map((item) => `<tr><th scope="row">${ESCAPAR(texto(item?.prueba, t))}</th><td>${ESCAPAR(texto(item?.referencia, t))}</td><td>${ESCAPAR(fecha(item?.fecha, t))}</td><td>${estadoFila(item?.estado, t)}</td></tr>`);
  const numero = (valor) => new Intl.NumberFormat("es-ES").format(valor);
  const ficha = seleccion ? `<aside id="pruebas-ficha" class="pruebas-ficha" tabindex="-1" aria-labelledby="pruebas-ficha-titulo">
    <div class="pruebas-ficha-cabecera"><div><p>${ESCAPAR(t("detalle_prueba"))}</p><h4 id="pruebas-ficha-titulo">${ESCAPAR(texto(seleccion.nombre, t))}</h4></div><button type="button" class="boton-secundario" data-pruebas-todas>${ESCAPAR(t("volver_todas"))}</button></div>
    <p>${ESCAPAR(texto(seleccion.descripcion, t, "sin_descripcion_prueba"))}</p>
    <div class="pruebas-ficha-recuento"><span><strong>${numero(resultados.length)}</strong> ${ESCAPAR(t(resultados.length === 1 ? "resultado_vinculado" : "resultados_vinculados"))}</span><span><strong>${numero(actas.length)}</strong> ${ESCAPAR(t(actas.length === 1 ? "acta_vinculada" : "actas_vinculadas"))}</span></div>
  </aside>` : "";
  return `<div class="pruebas-rejilla">
    ${panel(t, "pruebas", "pruebas_subtitulo", tabla(t, "pruebas", ["prueba", "fecha", "estado"], pruebas, "sin_pruebas") + ficha)}
    ${panel(t, "actas", "actas_subtitulo", tabla(t, "actas", ["prueba", "acta", "fecha", "estado"], actas, seleccion ? "sin_actas_vinculadas" : "sin_actas"))}
    ${panel(t, "resultados", "resultados_subtitulo", tabla(t, "resultados", ["prueba", "aspirante", "resultado", "estado"], resultados, seleccion ? "sin_resultados_vinculados" : "sin_resultados"), "pruebas-panel--ancho")}
  </div>`;
}

/** Modelo ya autorizado: { estado, convocatoria?, pruebas?, resultados?, actas? }.
 * El detalle requiere pruebas[].id y vincula resultados/actas solo por prueba_id. Sin red ni efectos.
 */
export function renderizarVistaPruebas(datos = {}, mensajes, seleccionId = "") {
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
  const contenido = disponibles && mensajeEstado === "disponible" ? detalleDisponible(datos, t, seleccionId) : `<div class="pruebas-rejilla">
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

/** Firma F1: montarVistaPruebas({ raiz, datos?, mensajes?, registrarDesmontar? }) → { actualizar, desmontar }. */
export function montarVistaPruebas({ raiz, datos = {}, mensajes, registrarDesmontar } = {}) {
  if (!raiz?.replaceChildren || !raiz?.addEventListener || !raiz?.removeEventListener || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("raíz de pruebas no disponible");
  let activa = true;
  let actuales = datos;
  let seleccionId = "";
  const pintar = () => { if (activa) raiz.innerHTML = renderizarVistaPruebas(actuales, mensajes, seleccionId); };
  const alClick = (evento) => {
    const detalle = evento.target?.closest?.("[data-prueba-detalle]");
    if (detalle) {
      seleccionId = detalle.dataset.pruebaDetalle;
      pintar();
      raiz.querySelector?.("#pruebas-ficha")?.focus();
      return;
    }
    if (evento.target?.closest?.("[data-pruebas-todas]")) {
      seleccionId = "";
      pintar();
      raiz.querySelector?.("[data-prueba-detalle]")?.focus();
    }
  };
  raiz.addEventListener("click", alClick);
  pintar();
  const actualizar = (siguiente) => {
    actuales = siguiente;
    if (siguiente?.estado !== "disponible" || !lista(siguiente?.pruebas).some((item) => idPrueba(item) === seleccionId)) seleccionId = "";
    pintar();
  };
  const desmontar = () => { if (!activa) return; activa = false; raiz.removeEventListener("click", alClick); raiz.replaceChildren(); };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ actualizar, desmontar });
}
