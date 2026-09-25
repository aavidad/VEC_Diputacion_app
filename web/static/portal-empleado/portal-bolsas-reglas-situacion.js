/**
 * Reglas del catálogo que afectan al cambio de situación de una participación:
 * destinos admitidos, causas de baja (art. 11) y propuesta de reposición
 * (art. 9). Las calcula el servidor; este módulo solo las pide y las presenta.
 * Sin catálogo la pantalla sigue como antes: tabla de transiciones actual,
 * fecha de disponibilidad a mano y motivo libre.
 */
import { etiquetaModalidadReposicion, traducirReglasSituacion as t } from "./portal-bolsas-reglas-situacion-i18n.js?v=20260926-integracion-bolsa-ct-v1";

export const RUTA_REGLAS_SITUACION = "/api/vec/bolsa/reglas-situacion";
const ESQUEMA = "vec.bolsa.rrhh.reglas_situacion.v1";
const CODIGO = /^[a-z][a-z0-9_]{0,63}$/;
const FECHA = /^\d{4}-\d{2}-\d{2}$/;

// Réplica de la tabla compilada del servidor. Solo se usa si la consulta de
// reglas no responde; el servidor es quien valida el cambio.
const TRANSICIONES_SIN_REGLAS = Object.freeze({
  disponible: ["no_disponible", "pendiente_incorporacion", "renuncia", "excluido"],
  no_disponible: ["disponible", "excluido"],
  pendiente_incorporacion: ["trabajando", "disponible", "renuncia", "excluido"],
  trabajando: ["disponible", "disponible_desde", "excluido"],
  disponible_desde: ["disponible", "excluido"],
  renuncia: ["disponible", "excluido"],
  excluido: [],
});

function procedenciaValida(p) {
  return p && typeof p === "object" && ["clave", "referencia", "articulo", "norma"].every((campo) => typeof p[campo] === "string")
    && typeof p.ejemplo === "boolean";
}

function reglasValidas(datos) {
  if (datos?.esquema !== ESQUEMA || typeof datos.configuradas !== "boolean" || !datos.transiciones || typeof datos.transiciones !== "object") return false;
  if (!Object.values(datos.transiciones).every((lista) => Array.isArray(lista) && lista.every((d) => typeof d === "string"))) return false;
  if (!Array.isArray(datos.causas_baja) || !datos.causas_baja.every((c) => CODIGO.test(c?.codigo || "") && typeof c.etiqueta === "string" && procedenciaValida(c.procedencia))) return false;
  if (datos.reposicion === null) return true;
  const { modalidades, propuesta } = datos.reposicion || {};
  if (!Array.isArray(modalidades) || !modalidades.every((m) => CODIGO.test(m?.codigo || "") && Number.isInteger(m.meses))) return false;
  return propuesta === null || (typeof propuesta?.fecha_disponible === "string" && !Number.isNaN(Date.parse(propuesta.fecha_disponible))
    && FECHA.test(propuesta.ultimo_dia_no_disponible || "") && Number.isInteger(propuesta.meses) && procedenciaValida(propuesta.procedencia));
}

/** Consulta las reglas; con finRelacion (AAAA-MM-DD) incluye la propuesta de reposición. */
export async function consultarReglasSituacion({ finRelacion = "", modalidad = "", fetchImpl = fetch, signal } = {}) {
  const parametros = new URLSearchParams();
  if (finRelacion) {
    if (!FECHA.test(finRelacion)) return { ok: false, codigo: "solicitud_invalida" };
    parametros.set("fin_relacion", finRelacion);
    if (modalidad) parametros.set("modalidad", modalidad);
  }
  const consulta = parametros.toString();
  try {
    const respuesta = await fetchImpl(consulta ? `${RUTA_REGLAS_SITUACION}?${consulta}` : RUTA_REGLAS_SITUACION, {
      method: "GET", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal, headers: { Accept: "application/json" },
    });
    if (!respuesta.ok) return { ok: false, status: respuesta.status, codigo: "no_disponible" };
    const cuerpo = await respuesta.json();
    return reglasValidas(cuerpo?.data) ? { ok: true, datos: cuerpo.data } : { ok: false, codigo: "respuesta_invalida" };
  } catch {
    return { ok: false, codigo: "error_red" };
  }
}

/** Destinos que el servidor admitirá desde origen. */
export function destinosSituacion(reglas, origen) {
  const lista = reglas?.transiciones?.[origen] ?? TRANSICIONES_SIN_REGLAS[origen] ?? [];
  return [...lista];
}

/** Causas del catálogo para la exclusión; vacío sin catálogo. */
export function causasBaja(reglas) {
  return reglas?.configuradas ? reglas.causas_baja : [];
}

/** Fecha de hoy (AAAA-MM-DD) en la hora del navegador. */
export function hoyCivil(ahora = new Date()) {
  const dos = (n) => String(n).padStart(2, "0");
  return `${ahora.getFullYear()}-${dos(ahora.getMonth() + 1)}-${dos(ahora.getDate())}`;
}

/** Instante ISO a valor de <input type="datetime-local"> en la hora del navegador. */
export function valorFechaLocal(iso) {
  const fecha = new Date(iso);
  if (Number.isNaN(fecha.getTime())) return "";
  const dos = (n) => String(n).padStart(2, "0");
  return `${hoyCivil(fecha)}T${dos(fecha.getHours())}:${dos(fecha.getMinutes())}`;
}

function textoArticulo(procedencia) {
  return procedencia.articulo
    ? t("procedencia_reglamento", { articulo: procedencia.articulo })
    : t("procedencia_ejemplo");
}

/** Texto de procedencia de la propuesta; solo se pinta en la pantalla de RRHH. */
export function textoProcedenciaReposicion(propuesta, ahora = new Date()) {
  if (!propuesta) return "";
  const partes = [t("propuesta", { meses: propuesta.meses, procedencia: textoArticulo(propuesta.procedencia) })];
  if (propuesta.procedencia.ejemplo) partes.push(t("regla_ejemplo"));
  if (Date.parse(propuesta.fecha_disponible) <= ahora.getTime()) partes.push(t("propuesta_pasada"));
  return partes.join(" ");
}

/**
 * Campos de la reposición para el formulario de cambio de situación. Solo
 * existen si hay catálogo y la participación está trabajando, único origen
 * desde el que se llega a «disponible desde».
 */
export function renderizarCamposReposicion({ reglas, candidato, estadoReposicion = {}, escaparHTML }) {
  if (!reglas?.configuradas || !reglas.reposicion || candidato?.estado_clave !== "trabajando") return "";
  const fin = estadoReposicion.finRelacion || hoyCivil();
  const modalidad = estadoReposicion.modalidad || "general";
  const opciones = [`<option value="general"${modalidad === "general" ? " selected" : ""}>${escaparHTML(t("modalidad_general"))}</option>`]
    .concat(reglas.reposicion.modalidades.map((m) => `<option value="${escaparHTML(m.codigo)}"${modalidad === m.codigo ? " selected" : ""}>${escaparHTML(t("modalidad_meses", { modalidad: etiquetaModalidadReposicion(m.codigo), meses: m.meses }))}</option>`));
  const propuesta = estadoReposicion.propuesta || reglas.reposicion.propuesta;
  return `<fieldset data-bolsa-reposicion><legend>${escaparHTML(t("reposicion"))}</legend>`
    + `<label>${escaparHTML(t("fin_relacion"))} <input type="date" name="fin_relacion" value="${escaparHTML(fin)}"></label>`
    + `<label>${escaparHTML(t("modalidad"))} <select name="modalidad_relacion">${opciones.join("")}</select></label>`
    + `<p class="nota-procedencia" role="status" aria-live="polite" data-bolsa-procedencia-reposicion>${escaparHTML(textoProcedenciaReposicion(propuesta))}</p></fieldset>`;
}

/** Valor inicial del campo de fecha: la propuesta del catálogo, si la hay. */
export function fechaDisponiblePropuesta(reglas, estadoReposicion = {}) {
  const propuesta = estadoReposicion.propuesta || reglas?.reposicion?.propuesta;
  return propuesta ? valorFechaLocal(propuesta.fecha_disponible) : "";
}

/** Selector de causa de baja para el paso de motivo de la exclusión. */
export function renderizarCausasBaja({ causas, seleccion = "", detalle = "", escaparHTML }) {
  const opciones = causas.map((c) => `<option value="${escaparHTML(c.codigo)}"${seleccion === c.codigo ? " selected" : ""}>${escaparHTML(c.etiqueta)} · ${escaparHTML(textoArticulo(c.procedencia))}</option>`).join("");
  return `<label>${escaparHTML(t("causa_baja"))} <select name="causa" required><option value="">${escaparHTML(t("causa_elegir"))}</option>${opciones}<option value="otro"${seleccion === "otro" ? " selected" : ""}>${escaparHTML(t("causa_otro"))}</option></select></label>`
    + `<label>${escaparHTML(t("causa_detalle"))} <textarea name="motivo" maxlength="1000">${escaparHTML(detalle)}</textarea></label>`;
}

/**
 * Motivo que se registra: la causa con su referencia y el detalle opcional.
 * «Otro motivo» exige el texto libre. Devuelve "" si falta algo.
 */
export function motivoConCausa(causas, codigo, detalle) {
  const texto = String(detalle || "").trim();
  if (codigo === "otro") return texto.length >= 2 ? texto : "";
  const causa = causas.find((c) => c.codigo === codigo);
  if (!causa) return "";
  const base = t("motivo_causa", { causa: causa.etiqueta, procedencia: textoArticulo(causa.procedencia) });
  const motivo = texto ? `${base}. ${texto}` : base;
  return motivo.length <= 1000 ? motivo : "";
}

/**
 * Recalcula la propuesta cuando RRHH cambia la fecha de fin o la modalidad.
 * Actualiza el campo y la procedencia sin volver a pintar el formulario, para
 * no perder lo que ya se ha escrito. obtenerModal devuelve la ficha abierta.
 */
export function instalarPropuestaReposicion(documento, obtenerModal, { consultar = consultarReglasSituacion } = {}) {
  documento.addEventListener("change", async (evento) => {
    const control = evento.target;
    const formulario = control?.closest?.('[data-bolsa-form="cambio-situacion"]');
    if (!formulario || !["fin_relacion", "modalidad_relacion"].includes(control.name)) return;
    const modal = obtenerModal();
    if (!modal) return;
    const datos = new FormData(formulario);
    const estadoReposicion = { finRelacion: String(datos.get("fin_relacion") || ""), modalidad: String(datos.get("modalidad_relacion") || "general") };
    modal.reposicion = estadoReposicion;
    const procedencia = formulario.querySelector("[data-bolsa-procedencia-reposicion]");
    const res = estadoReposicion.finRelacion ? await consultar(estadoReposicion) : { ok: false };
    if (obtenerModal() !== modal || modal.reposicion !== estadoReposicion) return;
    const propuesta = res.ok ? res.datos.reposicion?.propuesta : null;
    estadoReposicion.propuesta = propuesta || null;
    const campo = formulario.querySelector('[name="fecha_disponible"]');
    if (campo && propuesta) campo.value = valorFechaLocal(propuesta.fecha_disponible);
    if (procedencia) procedencia.textContent = propuesta ? textoProcedenciaReposicion(propuesta) : t("propuesta_no_disponible");
  });
}
