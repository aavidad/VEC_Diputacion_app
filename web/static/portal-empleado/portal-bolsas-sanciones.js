// Histórico de sanciones de una participación de Bolsa (petición RRHH p. 2,
// duda 62). Bloque «Sanciones» de la ficha del candidato en la vista de RRHH.
// Las consecuencias, su efecto, los plazos y los estados del recurso vienen del
// catálogo que sirve la API: aquí no se fija ninguno.
import { referenciaContieneDocumentoIdentidad } from "./portal-bolsas-operaciones.js?v=20260926-referencias-legibles-v1";
import { ayudaHuellaArchivo, instalarHuellaArchivo, renderizarCampoHuellaArchivo } from "./portal-huella-archivo.js";
import { traducirPortal } from "./portal-i18n.js?v=20260926-huecos-analisis-v1";
import { justificanteTraducido } from "./portal-justificante.js";

const BASE = "/api/vec/bolsa/bolsas";
const ESQUEMA = "vec.bolsa.rrhh.sanciones.v1";
const HEX_SHA256 = /^[a-f0-9]{64}$/;
const FECHA = /^\d{4}-\d{2}-\d{2}$/;
const POR_PAGINA = 6;

export const MENSAJES_SANCIONES_ES = Object.freeze({
  titulo: "Sanciones",
  subtitulo: "Histórico de sanciones y recursos de reposición",
  ayuda_aria: "Ayuda sobre las sanciones",
  ayuda: "Cada sanción aplica una consecuencia del catálogo de reglas, con el efecto que ese catálogo declara. La baja y la suspensión cambian la situación de la participación; la suspensión con fecha de fin termina sola ese día. Pasar al final no cambia la situación, pero coloca a la persona tras las no sancionadas de su bolsa. El recurso de reposición vence según el mismo catálogo; si se anota un estado que revoca la sanción (por ejemplo, estimado), otra persona lo resuelve y la participación vuelve a la situación anterior y a su puesto. La resolución permanece en su custodia: solo se anota su referencia y su huella digital. {huella}",
  nueva: "Registrar sanción",
  cancelar: "Cancelar",
  cargando: "Cargando histórico de sanciones…",
  vacio: "No hay sanciones registradas.",
  reintentar: "Reintentar",
  sin_catalogo: "El catálogo de sanciones no está configurado: el histórico se puede consultar, pero no registrar sanciones nuevas.",
  tabla: "Histórico de sanciones",
  col_fecha: "Notificación",
  col_consecuencia: "Consecuencia",
  col_causa: "Causa",
  col_resolucion: "Resolución",
  col_suspension: "Efecto aplicado",
  col_recurso: "Recurso de reposición",
  col_acciones: "Acciones",
  sin_suspension: "Sin cambio de situación",
  efecto_vuelve: "Vuelve al turno el {fecha}",
  efecto_suspendida: "No disponible hasta que se reactive",
  efecto_suspension_hasta: "Suspensión hasta el {fecha}",
  efecto_excluida: "Baja",
  efecto_al_final: "Al final de la lista",
  readmitida: "Readmitida el {fecha}",
  efectos_anulados: "Efectos anulados el {fecha}",
  readmision_resuelta: "Resuelve {persona}",
  situacion_restaurada: "Vuelve a: {situacion}",
  recurso_vence: "Vence el {fecha}",
  recurso_sin_estado: "Sin recurso anotado",
  anotar_recurso: "Anotar recurso",
  efecto_excluir: "Baja",
  efecto_pausar: "Suspensión",
  efecto_ninguna: "Sin cambio de situación",
  regla_ejemplo: "regla de ejemplo",
  campo_consecuencia: "Consecuencia",
  elegir: "Seleccione una opción",
  campo_causa: "Causa",
  campo_fecha_notificacion: "Fecha de notificación de la resolución",
  campo_resuelta_por: "Persona que resuelve",
  campo_resolucion_ref: "Referencia de la resolución en su custodia",
  campo_resolucion_archivo: "Archivo de la resolución",
  confirmar_sancion: "Confirmar sanción",
  registrando: "Registrando…",
  aviso_baja: "La baja definitiva no se puede deshacer y requiere que la resuelva otra persona.",
  campo_estado: "Estado del recurso",
  campo_fecha: "Fecha",
  campo_documento_ref: "Referencia del escrito (opcional)",
  campo_documento_archivo: "Archivo del escrito (opcional)",
  campo_documento_ref_obligatorio: "Referencia de la resolución que lo estima",
  campo_documento_archivo_obligatorio: "Archivo de la resolución que lo estima",
  campo_resuelta_por_recurso: "Persona que resuelve el recurso",
  aviso_revierte: "Este estado revoca la sanción: la participación vuelve a su situación anterior y a su puesto. Lo resuelve otra persona.",
  error_revierte: "Para revocar la sanción indique quién resuelve, la referencia de la resolución y elija su archivo.",
  confirmar_recurso: "Anotar estado",
  exito_sancion: "Sanción registrada.",
  exito_recurso: "Estado del recurso anotado.",
  exito_readmision: "Recurso anotado y sanción revocada.",
  recuperada: " (respuesta recuperada)",
  paginacion: "Paginación del histórico de sanciones",
  mostrando: "Mostrando {inicio} a {fin} de {total}",
  anterior: "Anterior",
  siguiente: "Siguiente",
  error_formulario: "Revise los campos: todos son obligatorios, también el archivo de la resolución.",
  error_documento: "Indique la referencia del escrito y elija su archivo, o deje ambos en blanco.",
  referencia_identidad: "La referencia no puede contener un DNI o NIE; use el número de registro o de expediente.",
  error_400: "La solicitud no es válida. Revise los campos del formulario.",
  error_403: "La sesión no dispone de permiso para gestionar sanciones.",
  error_404: "La participación o la sanción no existe.",
  error_409_clave: "Esta clave ya se usó con otros datos. Revise el histórico antes de reintentar.",
  error_409: "La sanción no puede aplicarse a la situación vigente. Revise la ficha y el histórico.",
  error_503_catalogo: "El catálogo de sanciones no está configurado.",
  error_503: "El servicio de sanciones no está disponible. Puede reintentar esta misma operación.",
  error_http: "No se pudo completar la operación (HTTP {estado}).",
  error_red: "No se pudo comunicar con el servicio de sanciones. Puede reintentar esta misma operación.",
  error_contrato: "La respuesta del servicio de sanciones no respeta su contrato.",
});

export function crearTraductorSanciones(catalogo = MENSAJES_SANCIONES_ES) {
  return (clave, variables = {}) => {
    if (typeof catalogo[clave] !== "string") throw new Error(`clave i18n de sanciones desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_c, v) => String(variables[v] ?? ""));
  };
}
const t = crearTraductorSanciones();

function segmento(valor) {
  return encodeURIComponent(String(valor ?? "").trim()).replace(/%3A/gi, ":");
}

export function rutaSanciones(bolsa, participacion) {
  return `${BASE}/${segmento(bolsa)}/candidatos/${segmento(participacion)}/sanciones`;
}

export function rutaRecursoSancion(bolsa, participacion, sancion) {
  return `${rutaSanciones(bolsa, participacion)}/${segmento(sancion)}/recursos`;
}

function errorHttp(estado, codigo = "") {
  const mensajes = {
    400: t("error_400"), 403: t("error_403"), 404: t("error_404"),
    409: codigo === "clave_reutilizada" ? t("error_409_clave") : t("error_409"),
    503: codigo === "sanciones_no_configuradas" ? t("error_503_catalogo") : t("error_503"),
  };
  return { ok: false, status: estado, codigo: codigo || "error_servidor", mensaje: mensajes[estado] || t("error_http", { estado }) };
}

const cadena = (valor) => typeof valor === "string";

function sancionValida(item) {
  return item && cadena(item.sancion_ref) && cadena(item.consecuencia) && cadena(item.consecuencia_etiqueta)
    && ["excluir", "pausar", "ninguna"].includes(item.efecto) && cadena(item.causa) && FECHA.test(item.fecha_notificacion || "")
    && item.resolucion && cadena(item.resolucion.referencia) && HEX_SHA256.test(item.resolucion.sha256 || "")
    && (item.suspension_hasta === null || FECHA.test(item.suspension_hasta))
    && item.recurso && FECHA.test(item.recurso.vence || "") && Array.isArray(item.recurso.eventos)
    && (item.recurso.estado === null || cadena(item.recurso.estado))
    && efectoAplicadoValido(item.efecto_aplicado) && reversionValida(item.reversion);
}

const INSTANTE = /^\d{4}-\d{2}-\d{2}T[\d:.]+Z$/;

function efectoAplicadoValido(efecto) {
  return efecto === undefined || (efecto && (efecto.situacion === null || cadena(efecto.situacion))
    && (efecto.vuelve_al_turno === null || INSTANTE.test(efecto.vuelve_al_turno || "")) && typeof efecto.orden_final === "boolean");
}

function reversionValida(reversion) {
  return reversion === undefined || reversion === null || (cadena(reversion.estado_recurso) && cadena(reversion.resuelta_por)
    && cadena(reversion.recibo_ref) && INSTANTE.test(reversion.registrada_en || "")
    && (reversion.situacion_restaurada === null || cadena(reversion.situacion_restaurada)));
}

async function pedir(ruta, opciones, fetchImpl) {
  const respuesta = await fetchImpl(ruta, { credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", ...opciones });
  const cuerpo = await respuesta.json().catch(() => ({}));
  return { respuesta, cuerpo };
}

export async function consultarSanciones(bolsa, participacion, { fetchImpl = fetch, signal } = {}) {
  try {
    const { respuesta, cuerpo } = await pedir(rutaSanciones(bolsa, participacion), { method: "GET", signal, headers: { Accept: "application/json" } }, fetchImpl);
    if (!respuesta.ok) return errorHttp(respuesta.status, cuerpo?.error?.codigo);
    const datos = cuerpo?.data;
    if (datos?.esquema !== ESQUEMA || typeof datos.catalogo_disponible !== "boolean" || !Array.isArray(datos.items)
      || !Array.isArray(datos.consecuencias) || !Array.isArray(datos.estados_recurso) || !datos.items.every(sancionValida)) {
      return { ok: false, status: 0, codigo: "respuesta_invalida", mensaje: t("error_contrato") };
    }
    return { ok: true, datos };
  } catch {
    return { ok: false, status: 0, codigo: "error_red", mensaje: t("error_red") };
  }
}

export function validarComandoSancion(comando) {
  if (referenciaContieneDocumentoIdentidad(comando?.resolucion?.referencia)) return t("referencia_identidad");
  const valido = comando && cadena(comando.consecuencia) && comando.consecuencia
    && cadena(comando.causa) && comando.causa.trim() && FECHA.test(comando.fecha_notificacion || "")
    && cadena(comando.resuelta_por) && comando.resuelta_por.trim()
    && cadena(comando.resolucion?.referencia) && comando.resolucion.referencia.trim() && HEX_SHA256.test(comando.resolucion?.sha256 || "");
  return valido ? "" : t("error_formulario");
}

export function validarComandoRecurso(comando, revocatorios = []) {
  if (!comando || !cadena(comando.estado) || !comando.estado || !FECHA.test(comando.fecha || "")) return t("error_formulario");
  const revierte = revocatorios.includes(comando.estado);
  if (revierte && (!cadena(comando.resuelta_por) || !comando.resuelta_por.trim() || !comando.documento)) return t("error_revierte");
  if (!revierte && comando.resuelta_por !== undefined) return t("error_formulario");
  if (comando.documento) {
    if (referenciaContieneDocumentoIdentidad(comando.documento.referencia)) return t("referencia_identidad");
    if (!cadena(comando.documento.referencia) || !comando.documento.referencia.trim() || !HEX_SHA256.test(comando.documento.sha256 || "")) return t("error_documento");
  }
  return "";
}

async function enviar(ruta, comando, clave, fetchImpl, valido) {
  try {
    const { respuesta, cuerpo } = await pedir(ruta, {
      method: "POST", body: JSON.stringify(comando),
      headers: { Accept: "application/json", "Content-Type": "application/json", "Idempotency-Key": clave },
    }, fetchImpl);
    if ((respuesta.status === 200 || respuesta.status === 201) && valido(cuerpo?.data)) return { ok: true, datos: cuerpo.data };
    if (respuesta.ok) return { ok: false, status: 0, codigo: "respuesta_invalida", mensaje: t("error_contrato") };
    return errorHttp(respuesta.status, cuerpo?.error?.codigo);
  } catch {
    return { ok: false, status: 0, codigo: "error_red", mensaje: t("error_red") };
  }
}

export async function registrarSancion(bolsa, participacion, comando, clave, { fetchImpl = fetch } = {}) {
  const error = validarComandoSancion(comando);
  if (error || !bolsa || !participacion || !clave) return { ok: false, status: 400, codigo: "solicitud_invalida", mensaje: error || t("error_400") };
  return enviar(rutaSanciones(bolsa, participacion), comando, clave, fetchImpl,
    (d) => d && cadena(d.sancion_ref) && cadena(d.recibo_ref) && typeof d.reutilizada === "boolean");
}

export async function registrarRecursoSancion(bolsa, participacion, sancion, comando, clave, { fetchImpl = fetch, revocatorios = [] } = {}) {
  const error = validarComandoRecurso(comando, revocatorios);
  if (error || !bolsa || !participacion || !sancion || !clave) return { ok: false, status: 400, codigo: "solicitud_invalida", mensaje: error || t("error_400") };
  return enviar(rutaRecursoSancion(bolsa, participacion, sancion), comando, clave, fetchImpl,
    (d) => d && d.sancion_ref === sancion && cadena(d.estado) && typeof d.reutilizada === "boolean"
      && (d.revertida === undefined || typeof d.revertida === "boolean"));
}

function html(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function fechaVisible(valor) {
  if (!FECHA.test(valor || "")) return "";
  const [anio, mes, dia] = valor.split("-").map(Number);
  return new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeZone: "UTC" }).format(new Date(Date.UTC(anio, mes - 1, dia)));
}

const CLASE_EFECTO = Object.freeze({ excluir: "peligro", pausar: "advertencia", ninguna: "info" });
const CLASE_RECURSO = Object.freeze({ interpuesto: "info", estimado: "exito", desestimado: "peligro", inadmitido: "peligro" });

function fechaInstante(valor) {
  if (!INSTANTE.test(valor || "")) return "";
  return new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeZone: "Europe/Madrid" }).format(new Date(valor));
}

// Lo que la sanción dejó hecho y, si se revocó, la readmisión.
function celdaEfecto(item, e) {
  const aplicado = item.efecto_aplicado || { situacion: null, vuelve_al_turno: null, orden_final: false };
  const partes = [];
  if (item.efecto === "excluir") partes.push(t("efecto_excluida"));
  else if (item.efecto === "pausar") {
    partes.push(aplicado.vuelve_al_turno ? t("efecto_vuelve", { fecha: fechaInstante(aplicado.vuelve_al_turno) })
      : item.suspension_hasta ? t("efecto_suspension_hasta", { fecha: fechaVisible(item.suspension_hasta) }) : t("efecto_suspendida"));
  }
  if (aplicado.orden_final) partes.push(t("efecto_al_final"));
  if (!partes.length) partes.push(t("sin_suspension"));
  let celda = partes.map((p) => e(p)).join("<br>");
  const r = item.reversion;
  if (r) {
    const fecha = fechaInstante(r.situacion_desde || r.registrada_en);
    const titulo = r.situacion_restaurada ? t("readmitida", { fecha }) : t("efectos_anulados", { fecha });
    celda += `<br><span class="estado-chip exito">${e(titulo)}</span>`;
    if (r.situacion_restaurada) celda += `<br><small>${e(t("situacion_restaurada", { situacion: etiquetaEstado(r.situacion_restaurada) }))}</small>`;
    celda += `<br><small>${e(t("readmision_resuelta", { persona: r.resuelta_por }))}</small>`;
  }
  return celda;
}

function etiquetaEstado(estado) {
  return String(estado || "").replaceAll("_", " ").replace(/^./, (c) => c.toUpperCase());
}

function filaSancion(item, e, recursoAbierto, catalogo) {
  const efecto = `<span class="estado-chip ${CLASE_EFECTO[item.efecto]}">${e(t(`efecto_${item.efecto}`))}</span>`;
  const estadoRecurso = item.recurso.estado
    ? `<span class="estado-chip ${CLASE_RECURSO[item.recurso.estado] || "neutro"}">${e(etiquetaEstado(item.recurso.estado))}</span>`
    : `<span class="estado-chip neutro">${e(t("recurso_sin_estado"))}</span>`;
  const accion = catalogo && recursoAbierto !== item.sancion_ref && !item.reversion
    ? `<button type="button" class="boton-secundario" data-b24-accion="abrir-recurso" data-sancion-ref="${e(item.sancion_ref)}">${e(t("anotar_recurso"))}</button>` : "";
  return `<tr><td>${e(fechaVisible(item.fecha_notificacion))}</td><td>${e(item.consecuencia_etiqueta)}<br>${efecto}</td><td>${e(item.causa)}</td><td>${e(item.resolucion.referencia)}<br><small>${e(item.resuelta_por)}</small></td><td>${celdaEfecto(item, e)}</td><td>${estadoRecurso}<br><small>${e(t("recurso_vence", { fecha: fechaVisible(item.recurso.vence) }))}</small></td><td>${accion}</td></tr>`;
}

function campo(etiqueta, control, e) {
  return `<label class="campo"><span>${e(etiqueta)}</span>${control}</label>`;
}

function formularioSancion(estado, e) {
  const f = estado.formulario || {};
  const opciones = (estado.datos?.consecuencias || []).map((c) => {
    const detalle = c.articulo || "";
    return `<option value="${e(c.clave)}" ${f.consecuencia === c.clave ? "selected" : ""}>${e(c.etiqueta)}${detalle ? ` (${e(detalle)})` : ""}</option>`;
  }).join("");
  const elegida = (estado.datos?.consecuencias || []).find((c) => c.clave === f.consecuencia);
  const aviso = elegida?.efecto === "excluir" ? `<p class="nota-seguridad" role="note">${e(t("aviso_baja"))}</p>` : "";
  const acciones = `<div class="acciones-formulario"><button type="button" class="boton-secundario" data-b24-accion="cancelar" ${estado.enviando ? "disabled" : ""}>${e(t("cancelar"))}</button><button type="submit" class="boton-primario" ${estado.enviando ? "disabled" : ""}>${e(estado.enviando ? t("registrando") : t("confirmar_sancion"))}</button></div>`;
  const error = estado.errorFormulario ? `<p class="mensaje-error" role="alert">${e(estado.errorFormulario)}</p>` : "";
  return `<form data-b24-form="sancion" class="formulario-gobernado"><fieldset><legend>${e(t("nueva"))}</legend><div class="rejilla-formulario">${campo(t("campo_consecuencia"), `<select name="consecuencia" required data-b24-campo="consecuencia"><option value="">${e(t("elegir"))}</option>${opciones}</select>`, e)}${campo(t("campo_fecha_notificacion"), `<input type="date" name="fecha_notificacion" required value="${e(f.fecha_notificacion || "")}">`, e)}${campo(t("campo_resuelta_por"), `<input name="resuelta_por" required maxlength="200" value="${e(f.resuelta_por || "")}">`, e)}${campo(t("campo_resolucion_ref"), `<input name="referencia" required maxlength="240" value="${e(f.referencia || "")}">`, e)}</div>${aviso}${campo(t("campo_causa"), `<textarea name="causa" required maxlength="1000">${e(f.causa || "")}</textarea>`, e)}${renderizarCampoHuellaArchivo({ id: "b24-resolucion-archivo", nombre: "sha256", huella: f.sha256, etiqueta: t("campo_resolucion_archivo"), escapar: e })}</fieldset>${error}${acciones}</form>`;
}

function formularioRecurso(estado, e) {
  const f = estado.formularioRecurso || {};
  const opciones = (estado.datos?.estados_recurso || []).map((valor) => `<option value="${e(valor)}" ${f.estado === valor ? "selected" : ""}>${e(etiquetaEstado(valor))}</option>`).join("");
  const revierte = (estado.datos?.estados_revocatorios || []).includes(f.estado);
  const requerido = revierte ? " required" : "";
  const resuelta = revierte ? campo(t("campo_resuelta_por_recurso"), `<input name="resuelta_por" required maxlength="200" value="${e(f.resuelta_por || "")}">`, e) : "";
  const aviso = revierte ? `<p class="nota-seguridad" role="note">${e(t("aviso_revierte"))}</p>` : "";
  const acciones = `<div class="acciones-formulario"><button type="button" class="boton-secundario" data-b24-accion="cancelar" ${estado.enviando ? "disabled" : ""}>${e(t("cancelar"))}</button><button type="submit" class="boton-primario" ${estado.enviando ? "disabled" : ""}>${e(estado.enviando ? t("registrando") : t("confirmar_recurso"))}</button></div>`;
  const error = estado.errorFormulario ? `<p class="mensaje-error" role="alert">${e(estado.errorFormulario)}</p>` : "";
  return `<form data-b24-form="recurso" class="formulario-gobernado"><fieldset><legend>${e(t("anotar_recurso"))}</legend><div class="rejilla-formulario">${campo(t("campo_estado"), `<select name="estado" required data-b24-campo="estado"><option value="">${e(t("elegir"))}</option>${opciones}</select>`, e)}${campo(t("campo_fecha"), `<input type="date" name="fecha" required value="${e(f.fecha || "")}">`, e)}${resuelta}${campo(t(revierte ? "campo_documento_ref_obligatorio" : "campo_documento_ref"), `<input name="referencia" maxlength="240"${requerido} value="${e(f.referencia || "")}">`, e)}${renderizarCampoHuellaArchivo({ id: "b24-recurso-archivo", nombre: "sha256", huella: f.sha256, obligatorio: revierte, etiqueta: t(revierte ? "campo_documento_archivo_obligatorio" : "campo_documento_archivo"), escapar: e })}</div>${aviso}</fieldset>${error}${acciones}</form>`;
}

export function renderizarSanciones({ estado = {}, escaparHTML = html }) {
  const e = escaparHTML;
  const carga = estado.carga || "cargando";
  const catalogo = carga === "listo" && estado.datos?.catalogo_disponible === true;
  let contenido;
  if (carga === "cargando") contenido = `<p class="vacio-controlado" role="status" aria-busy="true">${e(t("cargando"))}</p>`;
  else if (carga === "error") contenido = `<p class="mensaje-error" role="alert">${e(estado.error)}</p><button type="button" class="boton-secundario" data-b24-accion="reintentar">${e(t("reintentar"))}</button>`;
  else if (!estado.datos.items.length) contenido = `<p class="vacio-controlado" role="status">${e(t("vacio"))}</p>`;
  else {
    const items = estado.datos.items;
    const paginas = Math.max(1, Math.ceil(items.length / POR_PAGINA));
    const pagina = Math.min(Math.max(0, Number(estado.pagina) || 0), paginas - 1);
    const visibles = items.slice(pagina * POR_PAGINA, (pagina + 1) * POR_PAGINA);
    const cabecera = ["col_fecha", "col_consecuencia", "col_causa", "col_resolucion", "col_suspension", "col_recurso", "col_acciones"].map((c) => `<th scope="col">${e(t(c))}</th>`).join("");
    const pie = `<nav class="paginacion-bolsa" aria-label="${e(t("paginacion"))}"><span>${e(t("mostrando", { inicio: pagina * POR_PAGINA + 1, fin: Math.min((pagina + 1) * POR_PAGINA, items.length), total: items.length }))}</span>${paginas > 1 ? `<button type="button" class="boton-secundario" data-b24-accion="pagina" data-pagina="${pagina - 1}" ${pagina === 0 ? "disabled" : ""}>${e(t("anterior"))}</button><button type="button" class="boton-secundario" data-b24-accion="pagina" data-pagina="${pagina + 1}" ${pagina + 1 >= paginas ? "disabled" : ""}>${e(t("siguiente"))}</button>` : ""}</nav>`;
    contenido = `<div class="tabla-contenedor"><table class="tabla-datos"><caption>${e(t("tabla"))}</caption><thead><tr>${cabecera}</tr></thead><tbody>${visibles.map((item) => filaSancion(item, e, estado.recursoAbierto, catalogo && !estado.formularioAbierto)).join("")}</tbody></table></div>${pie}`;
  }
  const sinCatalogo = carga === "listo" && !catalogo ? `<p class="nota-seguridad" role="note">${e(t("sin_catalogo"))}</p>` : "";
  const boton = catalogo && !estado.formularioAbierto && !estado.recursoAbierto
    ? `<button type="button" class="boton-primario" data-b24-accion="nueva">${e(t("nueva"))}</button>` : "";
  const formulario = estado.formularioAbierto ? formularioSancion(estado, e) : estado.recursoAbierto ? formularioRecurso(estado, e) : "";
  const recibo = estado.exito && estado.recibo ? ` ${justificanteTraducido(estado.recibo, e, (clave) => traducirPortal(`panel_${clave}`))}` : "";
  const exito = estado.exito ? `<p class="mensaje-exito" role="status">${e(estado.exito)}${recibo}</p>` : "";
  const errorOperacion = estado.errorOperacion ? `<p class="mensaje-error" role="alert">${e(estado.errorOperacion)}</p>` : "";
  return `<section class="panel panel-separado" data-b24-raiz="true" aria-labelledby="b24-titulo"><div class="cabecera-panel"><div><h4 id="b24-titulo">${e(t("titulo"))}</h4><p>${e(t("subtitulo"))}</p></div><details><summary aria-label="${e(t("ayuda_aria"))}">?</summary><p>${e(t("ayuda", { huella: ayudaHuellaArchivo() }))}</p></details></div><div class="cuerpo-panel">${sinCatalogo}<div class="acciones-vista">${boton}</div>${exito}${errorOperacion}${formulario}${contenido}</div></section>`;
}

function claveIdempotente(flujo, comando) {
  const huella = JSON.stringify(comando);
  if (flujo.huella !== huella) {
    flujo.huella = huella;
    flujo.clave = globalThis.crypto?.randomUUID?.() || `b24-${Date.now()}-${Math.random().toString(16).slice(2)}`;
  }
  return flujo.clave;
}

function cerrarFormularios(flujo) {
  delete flujo.formularioAbierto; delete flujo.recursoAbierto; delete flujo.formulario; delete flujo.formularioRecurso;
  delete flujo.clave; delete flujo.huella; flujo.errorFormulario = "";
}

export function crearControladorSanciones({ estado, renderizar, recargar = async () => {}, fetchImpl }) {
  const opciones = fetchImpl ? { fetchImpl } : {};

  async function cargar(modal) {
    const controlador = new AbortController();
    modal.controladorSanciones?.abort();
    modal.controladorSanciones = controlador;
    // Sin pintar aquí: quien abre la ficha pinta el primer fotograma.
    modal.sancionesB24 = { ...modal.sancionesB24, carga: "cargando" };
    const res = await consultarSanciones(estado.bolsaSeleccionada, modal.candidato.participacion_ref, { ...opciones, signal: controlador.signal });
    if (controlador.signal.aborted || estado.modalFicha !== modal) return;
    modal.sancionesB24 = res.ok
      ? { ...modal.sancionesB24, carga: "listo", datos: res.datos }
      : { ...modal.sancionesB24, carga: "error", error: res.mensaje, datos: null };
    renderizar();
  }

  function manejarClick(evento) {
    const control = evento.target?.closest?.("[data-b24-accion]");
    if (!control || !estado.modalFicha) return false;
    evento.preventDefault?.();
    const modal = estado.modalFicha;
    const flujo = modal.sancionesB24 || (modal.sancionesB24 = { carga: "cargando" });
    if (flujo.enviando) return true;
    const accion = control.dataset.b24Accion;
    if (accion === "reintentar") { void cargar(modal); renderizar(); return true; }
    if (accion === "nueva") { cerrarFormularios(flujo); flujo.formularioAbierto = true; flujo.formulario = {}; flujo.exito = ""; flujo.errorOperacion = ""; }
    else if (accion === "abrir-recurso") { cerrarFormularios(flujo); flujo.recursoAbierto = control.dataset.sancionRef; flujo.formularioRecurso = {}; flujo.exito = ""; flujo.errorOperacion = ""; }
    else if (accion === "cancelar") cerrarFormularios(flujo);
    else if (accion === "pagina") flujo.pagina = Math.max(0, Number(control.dataset.pagina) || 0);
    renderizar();
    return true;
  }

  function manejarCambio(evento) {
    const flujo = estado.modalFicha?.sancionesB24;
    const campoRecurso = evento.target?.closest?.('[data-b24-campo="estado"]');
    if (campoRecurso && flujo?.recursoAbierto) {
      flujo.formularioRecurso = leerFormularioRecurso(new FormData(campoRecurso.form));
      renderizar();
      return true;
    }
    const campo = evento.target?.closest?.('[data-b24-campo="consecuencia"]');
    if (!campo || !flujo?.formularioAbierto) return false;
    const datos = new FormData(campo.form);
    flujo.formulario = leerFormularioSancion(datos);
    renderizar();
    return true;
  }

  async function finalizar(modal, flujo, respuesta, mensajeExito) {
    if (estado.modalFicha !== modal) return;
    flujo.enviando = false;
    if (!respuesta.ok) {
      flujo.errorOperacion = respuesta.mensaje;
      if (respuesta.status === 409) cerrarFormularios(flujo);
      renderizar();
      return;
    }
    cerrarFormularios(flujo);
    flujo.errorOperacion = "";
    flujo.exito = mensajeExito + (respuesta.datos.reutilizada ? t("recuperada") : "");
    flujo.recibo = typeof respuesta.datos.recibo_ref === "string" ? respuesta.datos.recibo_ref : "";
    if (respuesta.datos.situacion) modal.candidato = { ...modal.candidato, estado_clave: respuesta.datos.situacion, estado_desde: respuesta.datos.desde };
    await recargar(modal.candidato.participacion_ref);
    if (estado.modalFicha === modal) await cargar(modal);
  }

  function manejarSubmit(evento) {
    const formulario = evento.target?.closest?.("[data-b24-form]");
    if (!formulario || !estado.modalFicha) return false;
    evento.preventDefault?.();
    const modal = estado.modalFicha;
    const flujo = modal.sancionesB24;
    if (!flujo || flujo.enviando) return true;
    const datos = new FormData(formulario);
    if (formulario.dataset.b24Form === "sancion") {
      flujo.formulario = leerFormularioSancion(datos);
      const comando = { consecuencia: flujo.formulario.consecuencia, causa: flujo.formulario.causa, fecha_notificacion: flujo.formulario.fecha_notificacion,
        resuelta_por: flujo.formulario.resuelta_por, resolucion: { referencia: flujo.formulario.referencia, sha256: flujo.formulario.sha256 } };
      flujo.errorFormulario = validarComandoSancion(comando);
      if (flujo.errorFormulario) { renderizar(); return true; }
      const clave = claveIdempotente(flujo, comando);
      flujo.enviando = true; renderizar();
      void registrarSancion(estado.bolsaSeleccionada, modal.candidato.participacion_ref, comando, clave, opciones)
        .then((r) => finalizar(modal, flujo, r, r.ok ? t("exito_sancion") : ""));
      return true;
    }
    const f = leerFormularioRecurso(datos);
    flujo.formularioRecurso = f;
    const revocatorios = flujo.datos?.estados_revocatorios || [];
    const comando = { estado: f.estado, fecha: f.fecha };
    if (f.referencia || f.sha256) comando.documento = { referencia: f.referencia, sha256: f.sha256 };
    if (revocatorios.includes(f.estado)) comando.resuelta_por = f.resuelta_por;
    flujo.errorFormulario = validarComandoRecurso(comando, revocatorios);
    if (flujo.errorFormulario) { renderizar(); return true; }
    const sancion = flujo.recursoAbierto;
    const clave = claveIdempotente(flujo, { sancion, ...comando });
    flujo.enviando = true; renderizar();
    void registrarRecursoSancion(estado.bolsaSeleccionada, modal.candidato.participacion_ref, sancion, comando, clave, { ...opciones, revocatorios })
      .then((r) => finalizar(modal, flujo, r, r.ok && r.datos.revertida ? t("exito_readmision") : t("exito_recurso")));
    return true;
  }

  function instalar(documento = globalThis.document) {
    instalarHuellaArchivo(documento);
    documento.addEventListener("click", (evento) => { manejarClick(evento); });
    documento.addEventListener("submit", (evento) => { manejarSubmit(evento); });
    documento.addEventListener("change", (evento) => { manejarCambio(evento); });
  }

  return Object.freeze({ cargar, instalar, manejarClick, manejarSubmit, manejarCambio });
}

function leerFormularioRecurso(datos) {
  return {
    estado: String(datos.get("estado") || ""), fecha: String(datos.get("fecha") || ""),
    referencia: String(datos.get("referencia") || "").trim(), sha256: String(datos.get("sha256") || "").trim().toLowerCase(),
    resuelta_por: String(datos.get("resuelta_por") || "").trim(),
  };
}

function leerFormularioSancion(datos) {
  return {
    consecuencia: String(datos.get("consecuencia") || ""), causa: String(datos.get("causa") || "").trim(),
    fecha_notificacion: String(datos.get("fecha_notificacion") || ""), resuelta_por: String(datos.get("resuelta_por") || "").trim(),
    referencia: String(datos.get("referencia") || "").trim(), sha256: String(datos.get("sha256") || "").trim().toLowerCase(),
  };
}
