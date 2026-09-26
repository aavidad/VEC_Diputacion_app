import { LOCALIZACION_PORTAL, ZONA_HORARIA_PORTAL } from "./portal-i18n.js?v=20260926-i18n-v1";
/**
 * B13 · Histórico de contratos de la participación (Petición RRHH p. 2).
 * Solo lectura: los contratos proceden de Contratación temporal por evento
 * y Bolsa los guarda en su propio histórico. Textos por clave i18n.
 */
const BASE = "/api/vec/bolsa/bolsas";
export const ESQUEMA_CONTRATOS = "vec.bolsa.rrhh.contratos_participacion.v1";
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z$/;
const CLAVE = /^[a-z][a-z0-9._-]{1,79}$/;
const POR_PAGINA = 6;

export const MENSAJES_CONTRATOS_ES = Object.freeze({
  titulo: "Histórico de contratos",
  descripcion: "Contratos registrados en Contratación temporal a partir de un llamamiento de esta bolsa.",
  tabla: "Contratos de la participación",
  col_tipo: "Hecho",
  col_periodo: "Periodo",
  col_modalidad: "Modalidad",
  col_categoria: "Categoría",
  col_causa: "Causa",
  col_registrado: "Registrado",
  tipo_incorporacion: "Incorporación",
  hasta_previsto: "{inicio} – {fin} (prevista)",
  sin_fin: "{inicio} – sin fecha de fin",
  sin_dato: "—",
  cargando: "Cargando histórico de contratos…",
  vacio: "No hay contratos registrados para esta participación.",
  reintentar: "Reintentar histórico",
  paginacion: "Paginación del histórico de contratos",
  mostrando: "Mostrando {desde} a {hasta} de {total}",
  anterior: "Anterior",
  siguiente: "Siguiente",
  error_red: "No se pudo comunicar con el histórico de contratos.",
  error_contrato: "La respuesta del histórico de contratos no respeta su contrato.",
  error_403: "La sesión no dispone de permiso para consultar el histórico de contratos.",
  error_404: "El histórico de contratos no está disponible todavía en este entorno.",
  error_503: "El histórico de contratos no está disponible ahora. Puede reintentar.",
  error_http: "No se pudo consultar el histórico de contratos (HTTP {estado}).",
});

/** Traductor estricto: una clave inexistente es un error de programación. */
export function traducirContratos(clave, variables = {}, catalogo = MENSAJES_CONTRATOS_ES) {
  const plantilla = catalogo[clave];
  if (typeof plantilla !== "string") throw new Error(`Clave i18n de contratos inexistente: ${clave}`);
  return plantilla.replace(/\{(\w+)\}/g, (_, nombre) => String(variables[nombre] ?? ""));
}

function segmento(valor) {
  return encodeURIComponent(String(valor ?? "").trim()).replace(/%3A/gi, ":");
}

export function rutaContratosParticipacion(bolsa, participacion) {
  return `${BASE}/${segmento(bolsa)}/candidatos/${segmento(participacion)}/contratos`;
}

function instanteOpcional(valor) {
  return valor === null || (typeof valor === "string" && INSTANTE.test(valor));
}

function itemValido(item) {
  return item && typeof item === "object"
    && typeof item.evento_ref === "string" && item.evento_ref !== ""
    && typeof item.tipo === "string" && /^[a-z][a-z0-9_]{1,39}$/.test(item.tipo)
    && instanteOpcional(item.inicio) && instanteOpcional(item.fin_previsto)
    && typeof item.ocurrido_en === "string" && INSTANTE.test(item.ocurrido_en)
    && ["modalidad_clave", "causa_clave"].every((campo) => item[campo] === "" || (typeof item[campo] === "string" && CLAVE.test(item[campo])))
    && typeof item.categoria_ref === "string" && item.categoria_ref.length <= 512;
}

function errorHttp(status) {
  const clave = { 403: "error_403", 404: "error_404", 503: "error_503" }[status];
  return { ok: false, status, mensaje: clave ? traducirContratos(clave) : traducirContratos("error_http", { estado: status }) };
}

export async function consultarContratosParticipacion(bolsa, participacion, { fetchImpl = fetch, signal } = {}) {
  try {
    const respuesta = await fetchImpl(rutaContratosParticipacion(bolsa, participacion), {
      method: "GET", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal, headers: { Accept: "application/json" },
    });
    if (!respuesta.ok) return errorHttp(respuesta.status);
    const cuerpo = await respuesta.json();
    const items = cuerpo?.data?.items;
    if (cuerpo?.data?.esquema !== ESQUEMA_CONTRATOS || !Array.isArray(items) || !items.every(itemValido)) {
      return { ok: false, status: 0, mensaje: traducirContratos("error_contrato") };
    }
    return { ok: true, datos: items };
  } catch (error) {
    if (error?.name === "AbortError") return { ok: false, status: 0, abortada: true, mensaje: "" };
    return { ok: false, status: 0, mensaje: traducirContratos("error_red") };
  }
}

/**
 * Carga el histórico de la ficha abierta. Solo escribe si la ficha sigue
 * siendo la misma; una ficha nueva cancela la lectura anterior. Quien ya
 * va a pintar la ficha a continuación pasa renderizarAlIniciar=false.
 */
export async function cargarContratosFicha(modalFicha, { estado, renderizar, consultar = consultarContratosParticipacion, renderizarAlIniciar = true }) {
  if (!modalFicha?.candidato?.participacion_ref) return;
  modalFicha.controladorContratos?.abort();
  const controlador = new AbortController();
  modalFicha.controladorContratos = controlador;
  modalFicha.contratosB13 = { carga: "cargando", items: [], pagina: 0 };
  if (renderizarAlIniciar) renderizar();
  const res = await consultar(estado.bolsaSeleccionada, modalFicha.candidato.participacion_ref, { signal: controlador.signal });
  if (controlador.signal.aborted || estado.modalFicha !== modalFicha) return;
  modalFicha.contratosB13 = res.ok
    ? { carga: "listo", items: res.datos, pagina: 0 }
    : { carga: "error", error: res.mensaje, items: [], pagina: 0 };
  renderizar();
}

const FORMATO_FECHA = new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { timeZone: ZONA_HORARIA_PORTAL, day: "2-digit", month: "2-digit", year: "numeric" });

function fecha(valor) {
  return valor ? FORMATO_FECHA.format(new Date(valor)) : traducirContratos("sin_dato");
}

/** Rótulo legible de una clave de catálogo ajena cuando no hay traducción. */
function rotuloClave(clave) {
  if (!clave) return traducirContratos("sin_dato");
  const texto = clave.replace(/[._-]+/g, " ");
  return texto.charAt(0).toUpperCase() + texto.slice(1);
}

function rotuloTipo(tipo) {
  return Object.hasOwn(MENSAJES_CONTRATOS_ES, `tipo_${tipo}`) ? traducirContratos(`tipo_${tipo}`) : rotuloClave(tipo);
}

function periodo(item) {
  if (!item.inicio) return traducirContratos("sin_dato");
  return item.fin_previsto
    ? traducirContratos("hasta_previsto", { inicio: fecha(item.inicio), fin: fecha(item.fin_previsto) })
    : traducirContratos("sin_fin", { inicio: fecha(item.inicio) });
}

// La categoría del contrato llega como referencia opaca de Contratación: se
// nombra con la categoría de la bolsa en la que se hizo el llamamiento.
export function renderizarContratosParticipacion({ estado = {}, escaparHTML, categoria = "" }) {
  const t = (clave, variables) => escaparHTML(traducirContratos(clave, variables));
  const carga = estado.carga || "cargando";
  let contenido;
  if (carga === "cargando") contenido = `<p class="vacio-controlado" role="status" aria-busy="true">${t("cargando")}</p>`;
  else if (carga === "error") contenido = `<p class="mensaje-error" role="alert">${escaparHTML(estado.error || traducirContratos("error_red"))}</p><button type="button" class="boton-secundario" data-b13-accion="reintentar">${t("reintentar")}</button>`;
  else if (!estado.items?.length) contenido = `<p class="vacio-controlado" role="status">${t("vacio")}</p>`;
  else {
    const total = estado.items.length;
    const paginas = Math.max(1, Math.ceil(total / POR_PAGINA));
    const pagina = Math.min(Math.max(0, Number(estado.pagina) || 0), paginas - 1);
    const visibles = estado.items.slice(pagina * POR_PAGINA, (pagina + 1) * POR_PAGINA);
    const filas = visibles.map((item) => `<tr><td>${escaparHTML(rotuloTipo(item.tipo))}</td><td>${escaparHTML(periodo(item))}</td><td>${escaparHTML(rotuloClave(item.modalidad_clave))}</td><td>${item.categoria_ref && categoria ? escaparHTML(categoria) : t("sin_dato")}</td><td>${escaparHTML(rotuloClave(item.causa_clave))}</td><td>${escaparHTML(fecha(item.ocurrido_en))}</td></tr>`).join("");
    const resumen = t("mostrando", { desde: pagina * POR_PAGINA + 1, hasta: Math.min((pagina + 1) * POR_PAGINA, total), total });
    const navegacion = paginas > 1
      ? `<nav class="paginacion-bolsa" aria-label="${t("paginacion")}"><span>${resumen}</span><button type="button" class="boton-secundario" data-b13-accion="pagina" data-pagina="${pagina - 1}" ${pagina === 0 ? "disabled" : ""}>${t("anterior")}</button><button type="button" class="boton-secundario" data-b13-accion="pagina" data-pagina="${pagina + 1}" ${pagina + 1 >= paginas ? "disabled" : ""}>${t("siguiente")}</button></nav>`
      : `<p>${resumen}</p>`;
    contenido = `<div class="tabla-contenedor"><table class="tabla-datos"><caption>${t("tabla")}</caption><thead><tr><th scope="col">${t("col_tipo")}</th><th scope="col">${t("col_periodo")}</th><th scope="col">${t("col_modalidad")}</th><th scope="col">${t("col_categoria")}</th><th scope="col">${t("col_causa")}</th><th scope="col">${t("col_registrado")}</th></tr></thead><tbody>${filas}</tbody></table></div>${navegacion}`;
  }
  return `<section class="panel panel-separado" data-b13-raiz="true" aria-labelledby="b13-titulo"><div class="cabecera-panel"><div><h4 id="b13-titulo">${t("titulo")}</h4><p>${t("descripcion")}</p></div></div><div class="cuerpo-panel">${contenido}</div></section>`;
}

/** Atiende reintento y paginación de la sección B13 de la ficha abierta. */
export function manejarClickContratos(evento, { estado, renderizar, consultar } = {}) {
  const control = evento.target?.closest?.("[data-b13-accion]");
  if (!control || !estado?.modalFicha) return false;
  evento.preventDefault();
  const modal = estado.modalFicha;
  if (control.dataset.b13Accion === "reintentar") {
    void cargarContratosFicha(modal, { estado, renderizar, ...(consultar ? { consultar } : {}) });
    return true;
  }
  if (control.dataset.b13Accion === "pagina" && modal.contratosB13) {
    modal.contratosB13 = { ...modal.contratosB13, pagina: Math.max(0, Number(control.dataset.pagina) || 0) };
    renderizar();
  }
  return true;
}
