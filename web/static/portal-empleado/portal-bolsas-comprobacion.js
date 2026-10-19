/**
 * Comprobación del ofrecimiento en la ficha de la persona (F1-57).
 *
 * Se abre desde el aviso «pendiente de comprobación» de la selección de
 * Contratación temporal, con el contexto que publicó el servidor. RRHH indica
 * los once hechos del artículo 9 y, para cada uno, el documento que lo
 * acredita: el contrato que Bolsa ya conserva (su evento y huella) o un
 * documento que RRHH custodia, declarado con una referencia legible y la huella
 * SHA-256 que calcula el navegador. El archivo no sale del equipo.
 * El resultado lo calcula el servidor; la pantalla solo muestra su recibo.
 */
import { cargarTextos } from "../comun/textos.js";
import { LOCALIZACION_PORTAL, traducirPortal, ZONA_HORARIA_PORTAL } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";
import { calcularHuellaArchivo, LIMITE_HUELLA_ARCHIVO_BYTES } from "./portal-huella-archivo.js";
import { justificanteTraducido } from "./portal-justificante.js";
import { propuestaComprobacionValida } from "./portal-bolsas-comprobacion-ruta.js?v=20261019-f157-v1";

const BASE = "/api/vec/bolsa/bolsas";
const HUELLA = /^[0-9a-f]{64}$/u;
const CLAVE = /^[a-z][a-z0-9._-]{1,79}$/u;
const FECHA = /^\d{4}-\d{2}-\d{2}$/u;
const EVENTO_CONTRATO = /^evento:ct:contrato-bolsa:[0-9a-f]{64}$/u;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:Z|[+-]\d{2}:\d{2})$/u;
const COMPROBACION_REF = /^comprobacion:[0-9a-f]{64}$/u;
// Espejo de b77_documento_ref_opaco_v1 y de ReferenciaDocumentoCustodiaValida.
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,255}$/u;
const ETIQUETA_IDENTIDAD = /(^|[._:/#-])(dni|nie|nif|pasaporte|passport)([._:/#-]|$)/iu;
const NUMERO_IDENTIDAD = /((?:[0-9][._:/#-]?){8}|[XYZ][._:/#-]?(?:[0-9][._:/#-]?){7})[A-Z]/iu;
const PROPIA_SISTEMA = /^[a-z_]+(?::[a-z_]+)*:[0-9a-f]{64}$/u;
const MAXIMO_DOCUMENTOS = 6;
const ESPERA_MS = 30_000;
const MOTIVOS = Object.freeze({
  pendiente: ["antecedentes_incompletos", "criterios_invalidos", "guarda_contratacion_pendiente", "descuento_pendiente_criterio_rrhh"],
  no_aplicable: ["sin_supuesto_articulo_9"],
});
const TIPOS_RELACION = Object.freeze(["acumulacion", "sustitucion", "programa", "otra"]);
const MODALIDADES = Object.freeze(["acumulacion_tareas", "sustitucion", "programa", "vacante"]);
const TIPO_POR_MODALIDAD = Object.freeze({ acumulacion_tareas: "acumulacion", sustitucion: "sustitucion", programa: "programa" });
// Orden de la pantalla, agrupado como se comprueba: relación actual, puesto y bolsa, cierre.
export const HECHOS_COMPROBACION = Object.freeze([
  ["relacion", "trabaja_en_diputacion", "si_no"], ["relacion", "tipo_relacion", "opcion"], ["relacion", "modalidad", "modalidad"],
  ["relacion", "causa_relacion", "clave"], ["relacion", "inicio", "fecha"], ["relacion", "fin_previsto", "fecha"],
  ["relacion", "fin_inicial_programa", "fecha"], ["relacion", "fecha_inicial_baja", "fecha"],
  ["puesto", "es_vacante", "si_no"], ["puesto", "prelacion", "fijo"], ["cierre", "antecedentes_completos", "si_no"],
].map(([grupo, clave, tipo]) => Object.freeze({ grupo, clave, tipo })));

let TEXTOS = null;
let preparacion = null;
/** Carga el catálogo de la pantalla una vez, en el idioma activo. */
const MODULO_TEXTOS = "bolsa-comprobacion-ofrecimiento";
function leerTextos(leerCatalogoB57 = cargarTextos) { return leerCatalogoB57(MODULO_TEXTOS); }
export function prepararTextosComprobacion(lector) {
  if (TEXTOS) return Promise.resolve(TEXTOS);
  if (!preparacion) {
    preparacion = leerTextos(lector).then((textos) => { TEXTOS = textos; return textos; })
      .catch((error) => { preparacion = null; throw error; });
  }
  return preparacion;
}
const t = (clave, variables) => TEXTOS.traducir(clave, variables);

export function referenciaDocumentoValida(v) {
  return typeof v === "string" && REFERENCIA.test(v) && !ETIQUETA_IDENTIDAD.test(v)
    && (PROPIA_SISTEMA.test(v) || !NUMERO_IDENTIDAD.test(v));
}

function segmento(valor) {
  return encodeURIComponent(String(valor)).replace(/%3A/gi, ":");
}
export function rutaComprobacionOfrecimiento(bolsa, participacion) {
  return `${BASE}/${segmento(bolsa)}/candidatos/${segmento(participacion)}/comprobaciones-ofrecimiento`;
}

function fechaCivil(instante) {
  if (typeof instante !== "string" || !INSTANTE.test(instante) || !Number.isFinite(Date.parse(instante))) return "";
  const partes = Object.fromEntries(new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { timeZone: ZONA_HORARIA_PORTAL, year: "numeric", month: "2-digit", day: "2-digit" })
    .formatToParts(new Date(instante)).map(({ type, value }) => [type, value]));
  return `${partes.year}-${partes.month}-${partes.day}`;
}
function fechaVisible(civil) {
  return new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { timeZone: "UTC", dateStyle: "short" }).format(new Date(`${civil}T00:00:00Z`));
}
function fechaValida(v) {
  return FECHA.test(v) && new Date(`${v}T00:00:00Z`).toISOString().slice(0, 10) === v;
}

/** Contrato más reciente que Bolsa conserva, con su evento y huella; null si no hay. */
export function contratoEvidencia(contratos) {
  const item = contratos?.carga === "listo" ? contratos.items?.[0] : null;
  return item && EVENTO_CONTRATO.test(item.evento_ref) && HUELLA.test(item.huella_sha256 ?? "") ? item : null;
}

/** Valores que se deducen del contrato registrado; el resto lo indica RRHH. */
export function prerrellenoContrato(contrato, hoy = fechaCivil(new Date().toISOString())) {
  if (!contrato) return {};
  const inicio = fechaCivil(contrato.inicio);
  const fin = fechaCivil(contrato.fin_previsto);
  const valores = {};
  if (inicio && inicio <= hoy && (!fin || fin >= hoy)) valores.trabaja_en_diputacion = "si";
  if (TIPO_POR_MODALIDAD[contrato.modalidad_clave]) valores.tipo_relacion = TIPO_POR_MODALIDAD[contrato.modalidad_clave];
  if (CLAVE.test(contrato.modalidad_clave ?? "")) valores.modalidad = contrato.modalidad_clave;
  if (CLAVE.test(contrato.causa_clave ?? "")) valores.causa_relacion = contrato.causa_clave;
  if (inicio) valores.inicio = inicio;
  if (fin) valores.fin_previsto = fin;
  return valores;
}

/** Abre el bloque en la ficha si la propuesta de la URL es de esta persona. */
export function iniciarComprobacion(modal, propuesta, bolsaRef) {
  const valida = propuestaComprobacionValida(propuesta);
  if (!modal || !valida || valida.contexto.bolsa_ref !== bolsaRef
    || valida.contexto.participacion_ref !== modal.candidato?.participacion_ref) return false;
  modal.comprobacionB57 = { propuesta: valida, valores: {}, docs: {}, documentos: [{ id: "d1", referencia: "", huella: "" }],
    siguiente: 2, paso: "edicion", errores: [], clave: "", cuerpo: "", recibo: null, mensaje: "" };
  return true;
}

function estadoHechos(st, modal) {
  const contrato = contratoEvidencia(modal.contratosB13);
  const previos = prerrellenoContrato(contrato);
  previos.prelacion = String(st.propuesta.contexto.orden_acta);
  const valor = (clave) => st.valores[clave] ?? previos[clave] ?? "";
  const doc = (clave) => st.docs[clave] ?? (contrato && Object.hasOwn(previos, clave) && clave !== "prelacion" ? "contrato" : "");
  return { contrato, valor, doc };
}

function textoHecho(clave) { return t(`ficha.hecho_${clave}`); }

/** Valida y compone `hechos`; devuelve { hechos, errores }. */
export function componerHechos(st, modal) {
  const { contrato, valor, doc } = estadoHechos(st, modal);
  const errores = [];
  const hechos = {};
  const documentos = new Map(st.documentos.map((d, i) => [d.id, { ...d, numero: i + 1 }]));
  const usados = new Set();
  for (const { clave, tipo } of HECHOS_COMPROBACION) {
    const v = valor(clave);
    const d = doc(clave);
    const conValor = tipo === "fijo" ? d !== "" : v !== "";
    if (!conValor && !d) continue;
    if (!conValor) { errores.push({ campo: `b57-hecho-${clave}`, mensaje: t("ficha.error_hecho_sin_valor", { hecho: textoHecho(clave) }) }); continue; }
    if (!d) { errores.push({ campo: `b57-doc-${clave}`, mensaje: t("ficha.error_hecho_sin_documento", { hecho: textoHecho(clave) }) }); continue; }
    if (tipo === "fecha" && !fechaValida(v)) { errores.push({ campo: `b57-hecho-${clave}`, mensaje: t("ficha.error_hecho_fecha", { hecho: textoHecho(clave) }) }); continue; }
    if ((tipo === "clave" || tipo === "modalidad") && !CLAVE.test(v)) { errores.push({ campo: `b57-hecho-${clave}`, mensaje: t("ficha.error_hecho_clave", { hecho: textoHecho(clave) }) }); continue; }
    let evidencia;
    if (d === "contrato" && contrato) evidencia = { documento_ref: contrato.evento_ref, huella: contrato.huella_sha256 };
    else if (documentos.has(d)) { usados.add(d); const x = documentos.get(d); evidencia = { documento_ref: x.referencia, huella: x.huella }; }
    else { errores.push({ campo: `b57-doc-${clave}`, mensaje: t("ficha.error_hecho_sin_documento", { hecho: textoHecho(clave) }) }); continue; }
    const json = tipo === "si_no" ? v === "si" : tipo === "fijo" ? st.propuesta.contexto.orden_acta : v;
    hechos[clave] = { valor: json, ...evidencia };
  }
  for (const id of usados) {
    const x = documentos.get(id);
    if (!referenciaDocumentoValida(x.referencia)) errores.push({ campo: `b57-docref-${id}`, mensaje: t("ficha.error_documento_referencia", { numero: x.numero }) });
    if (x.calculando) errores.push({ campo: `b57-docarchivo-${id}`, mensaje: t("ficha.error_documento_calculando", { numero: x.numero }) });
    else if (!HUELLA.test(x.huella)) errores.push({ campo: `b57-docarchivo-${id}`, mensaje: t("ficha.error_documento_archivo", { numero: x.numero }) });
  }
  if (!errores.length && !Object.keys(hechos).length) errores.push({ campo: "b57-hecho-trabaja_en_diputacion", mensaje: t("ficha.error_sin_hechos") });
  return { hechos, errores };
}

/** Recibo mínimo del servidor; cualquier otra forma no se presenta como registrado. */
export function reciboValido(r) {
  return r && typeof r === "object" && COMPROBACION_REF.test(r.comprobacion_ref ?? "") && r.recibo_ref === `recibo:${r.comprobacion_ref}`
    && Number.isSafeInteger(r.version) && r.version > 0 && r.version === r.version_esperada + 1
    && Object.hasOwn(MOTIVOS, r.resultado) && MOTIVOS[r.resultado].includes(r.motivo_clave)
    && typeof r.registrada_en === "string" && INSTANTE.test(r.registrada_en) && typeof r.reutilizada === "boolean";
}

/** POST con Idempotency-Key. 4xx: rechazo sin efecto. 5xx, red o respuesta ilegible: incierto. */
export async function registrarComprobacion(bolsa, participacion, cuerpo, clave, { fetchImpl = fetch, espera = ESPERA_MS } = {}) {
  const controlador = new AbortController();
  const temporizador = setTimeout(() => controlador.abort(), espera);
  try {
    const respuesta = await fetchImpl(rutaComprobacionOfrecimiento(bolsa, participacion), {
      method: "POST", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
      signal: controlador.signal, body: cuerpo,
      headers: { Accept: "application/json", "Content-Type": "application/json", "Idempotency-Key": clave },
    });
    const datos = await respuesta.json().catch(() => null);
    if ((respuesta.status === 200 || respuesta.status === 201) && reciboValido(datos?.data)) return { ok: true, recibo: datos.data };
    const codigo = typeof datos?.error?.codigo === "string" ? datos.error.codigo : "";
    if (respuesta.status >= 400 && respuesta.status < 500) return { ok: false, rechazo: true, codigo };
    return { ok: false, incierto: true, codigo };
  } catch {
    return { ok: false, incierto: true, codigo: "" };
  } finally {
    clearTimeout(temporizador);
  }
}

function mensajeError(codigo) {
  return TEXTOS.traducir(`errores.${["comprobacion_en_conflicto", "solicitud_invalida", "acceso_denegado", "comprobacion_ofrecimiento_no_habilitada"].includes(codigo) ? codigo : "servicio_no_disponible"}`);
}

function textoContrato(contrato) {
  const inicio = fechaCivil(contrato?.inicio);
  return inicio ? t("ficha.contrato_registrado", { inicio: fechaVisible(inicio) }) : t("ficha.contrato_registrado_sin_fecha");
}

function opcionesDocumento(st, contrato, actual, e) {
  const opciones = [["", t("ficha.sin_documento")], ...(contrato ? [["contrato", textoContrato(contrato)]] : []),
    ...st.documentos.map((d, i) => [d.id, d.referencia ? `${t("ficha.documento_n", { numero: i + 1 })} · ${d.referencia}` : t("ficha.documento_n", { numero: i + 1 })])];
  return opciones.map(([v, texto]) => `<option value="${e(v)}"${v === actual ? " selected" : ""}>${e(texto)}</option>`).join("");
}

function textoValor(clave, tipo, v) {
  if (v === "") return t("ficha.sin_indicar");
  if (tipo === "si_no") return t(`ficha.${v}`);
  if (tipo === "opcion") return t(`ficha.tipo_${v}`);
  if (tipo === "modalidad" && MODALIDADES.includes(v)) return t(`ficha.modalidad_${v}`);
  if (tipo === "fecha" && fechaValida(v)) return fechaVisible(v);
  return v;
}

function campoValor({ clave, tipo }, v, error, e) {
  const id = `b57-hecho-${clave}`;
  const etiqueta = textoHecho(clave);
  const invalido = error ? ` aria-invalid="true" aria-describedby="${id}-error"` : "";
  const msg = error ? `<small id="${id}-error" class="mensaje-error">${e(error)}</small>` : "";
  if (tipo === "si_no") {
    return `<fieldset class="grupo-campo" id="${id}"${invalido}><legend>${e(etiqueta)}</legend>${["si", "no"].map((x) => `<label><input type="radio" name="${id}" value="${x}" data-b57-hecho="${clave}"${v === x ? " checked" : ""}> ${e(t(`ficha.${x}`))}</label>`).join(" ")}${msg}</fieldset>`;
  }
  if (tipo === "fijo") return `<div class="campo"><span>${e(etiqueta)}</span><output id="${id}">${e(new Intl.NumberFormat(LOCALIZACION_PORTAL).format(Number(v)))}</output></div>`;
  let control;
  if (tipo === "opcion" || tipo === "modalidad") {
    const valores = tipo === "opcion" ? TIPOS_RELACION : [...MODALIDADES, ...(v && !MODALIDADES.includes(v) ? [v] : [])];
    control = `<select id="${id}" data-b57-hecho="${clave}"${invalido}><option value="">${e(t("ficha.sin_indicar"))}</option>${valores.map((x) => `<option value="${e(x)}"${x === v ? " selected" : ""}>${e(textoValor(clave, tipo, x))}</option>`).join("")}</select>`;
  } else if (tipo === "fecha") {
    control = `<input id="${id}" type="date" data-b57-hecho="${clave}" value="${e(v)}"${invalido}>`;
  } else {
    control = `<input id="${id}" data-b57-hecho="${clave}" value="${e(v)}" maxlength="80" autocomplete="off"${invalido}><small>${e(t("ficha.hecho_causa_relacion_pista"))}</small>`;
  }
  return `<label class="campo" for="${id}"><span>${e(etiqueta)}</span>${control}${msg}</label>`;
}

function bloqueDocumento(d, i, errores, bloqueado, e) {
  const n = i + 1;
  const ref = `b57-docref-${d.id}`;
  const archivo = `b57-docarchivo-${d.id}`;
  const err = (campo) => errores.find((x) => x.campo === campo)?.mensaje;
  const estadoArchivo = err(archivo) || (d.calculando ? t("huella.calculando") : HUELLA.test(d.huella) ? t("huella.comprobado_nombre", { nombre: d.nombre ?? "" }) : d.errorArchivo || "");
  return `<fieldset class="grupo-campo" data-b57-documento="${e(d.id)}"><legend>${e(t("ficha.documento_n", { numero: n }))}</legend>
    <label class="campo" for="${ref}"><span>${e(t("ficha.documento_referencia"))}</span><input id="${ref}" data-b57-doc-ref="${e(d.id)}" value="${e(d.referencia)}" maxlength="256" autocomplete="off" aria-describedby="${ref}-pista"${err(ref) ? ' aria-invalid="true"' : ""}${bloqueado ? " disabled" : ""}><small id="${ref}-pista"${err(ref) ? ' class="mensaje-error"' : ""}>${e(err(ref) || t("ficha.documento_referencia_pista"))}</small></label>
    <label class="campo" for="${archivo}"><span>${e(t("ficha.documento_archivo", { numero: n }))}</span><input id="${archivo}" type="file" data-b57-doc-archivo="${e(d.id)}" aria-describedby="${archivo}-estado"${err(archivo) ? ' aria-invalid="true"' : ""}${bloqueado ? " disabled" : ""}><small id="${archivo}-estado" role="status">${e(estadoArchivo)}</small></label>
    ${i > 0 && !bloqueado ? `<button type="button" class="boton-terciario" data-b57-accion="quitar-documento" data-b57-id="${e(d.id)}">${e(t("ficha.quitar_documento", { numero: n }))}</button>` : ""}</fieldset>`;
}

function renderizarRecibo(r, e) {
  return `<div class="mensaje-exito" role="status" tabindex="-1" data-b57-recibo><p><strong>${e(t("ficha.recibo_titulo"))}</strong>${r.reutilizada ? ` ${e(t("ficha.recibo_recuperada"))}` : ""}</p>
    <dl class="resumen-expediente"><div class="fila-resumen"><dt>${e(t("ficha.recibo_resultado"))}</dt><dd>${e(t(`ficha.resultado_${r.resultado}`))}<br><small>${e(t(`ficha.motivo_${r.motivo_clave}`))}</small></dd></div>
    <div class="fila-resumen"><dt>${e(t("ficha.recibo_fecha"))}</dt><dd><time datetime="${e(r.registrada_en)}">${e(new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { timeZone: ZONA_HORARIA_PORTAL, dateStyle: "short", timeStyle: "short" }).format(new Date(r.registrada_en)))}</time></dd></div>
    </dl><p>${justificanteTraducido(r.recibo_ref, e, (clave) => traducirPortal(`panel_${clave}`))}</p>
    <p>${e(t("ficha.siguiente_paso"))}</p><div class="acciones-vista"><button type="button" class="boton-secundario" data-vista="contratacion-temporal">${e(t("ficha.ir_peticiones"))}</button></div></div>`;
}

/** Bloque de la ficha; vacío si la persona no llega desde una comprobación pendiente. */
export function renderizarComprobacionOfrecimiento({ modal, escaparHTML: e }) {
  const st = modal?.comprobacionB57;
  if (!st) return "";
  if (!TEXTOS) {
    return `<section class="panel panel-separado" data-b57-raiz="true"><div class="cuerpo-panel">${st.errorTextos
      ? `<p class="mensaje-error" role="alert">${e(traducirPortal("txt_error_de_carga"))}</p><button type="button" class="boton-secundario" data-b57-accion="reintentar-textos">${e(traducirPortal("txt_reintentar"))}</button>`
      : '<p class="vacio-controlado" role="status" aria-busy="true"></p>'}</div></section>`;
  }
  const { contrato, valor, doc } = estadoHechos(st, modal);
  const bloqueado = st.paso !== "edicion";
  const errorDe = (campo) => st.errores.find((x) => x.campo === campo)?.mensaje;
  const c = st.propuesta.contexto;
  const resumenErrores = st.errores.length ? `<div class="mensaje-error" role="alert" tabindex="-1" data-b57-errores><p>${e(t("ficha.errores_resumen", { cuenta: st.errores.length }))}</p><ul>${st.errores.map((x) => `<li><a href="#${e(x.campo)}" data-b57-accion="ir-campo" data-b57-campo="${e(x.campo)}">${e(x.mensaje)}</a></li>`).join("")}</ul></div>` : "";
  const grupos = ["relacion", "puesto", "cierre"].map((grupo) => `<fieldset class="grupo-campo"><legend>${e(t(`ficha.grupo_${grupo}`))}</legend><div class="rejilla-formulario rejilla-llamamiento">${HECHOS_COMPROBACION.filter((h) => h.grupo === grupo).map((h) => {
    const idDoc = `b57-doc-${h.clave}`;
    return `${campoValor(h, valor(h.clave), errorDe(`b57-hecho-${h.clave}`), e)}<label class="campo" for="${idDoc}"><span>${e(t("ficha.documento_que_acredita"))}</span><select id="${idDoc}" data-b57-doc-hecho="${h.clave}" aria-label="${e(t("ficha.documento_que_acredita_aria", { hecho: textoHecho(h.clave) }))}"${errorDe(idDoc) ? ` aria-invalid="true" aria-describedby="${idDoc}-error"` : ""}>${opcionesDocumento(st, contrato, doc(h.clave), e)}</select>${errorDe(idDoc) ? `<small id="${idDoc}-error" class="mensaje-error">${e(errorDe(idDoc))}</small>` : ""}</label>`;
  }).join("")}</div></fieldset>`).join("");
  const documentos = `<fieldset class="grupo-campo"><legend>${e(t("ficha.documentos_titulo"))}</legend>${st.documentos.map((d, i) => bloqueDocumento(d, i, st.errores, bloqueado, e)).join("")}${!bloqueado && st.documentos.length < MAXIMO_DOCUMENTOS ? `<button type="button" class="boton-secundario" data-b57-accion="anadir-documento">${e(t("ficha.anadir_documento"))}</button>` : ""}</fieldset>`;
  const edicion = `<form data-b57-form="comprobacion" novalidate${bloqueado ? " hidden" : ""}>${resumenErrores}<p>${e(t("ficha.solo_acreditados"))}</p>${documentos}${grupos}<div class="acciones-vista"><button type="submit" class="boton-primario">${e(t("ficha.revisar"))}</button></div></form>`;
  let revision = "";
  if (bloqueado && st.paso !== "recibo") {
    const filas = HECHOS_COMPROBACION.map((h) => {
      if (!st.hechos?.[h.clave]) return `<div class="fila-resumen"><dt>${e(textoHecho(h.clave))}</dt><dd>${e(t("ficha.sin_indicar"))}</dd></div>`;
      const d = doc(h.clave);
      const n = st.documentos.findIndex((x) => x.id === d) + 1;
      const fuente = d === "contrato" ? textoContrato(contrato) : `${t("ficha.documento_n", { numero: n })} · ${st.documentos[n - 1]?.referencia ?? ""}`;
      return `<div class="fila-resumen"><dt>${e(textoHecho(h.clave))}</dt><dd>${e(textoValor(h.clave, h.tipo, valor(h.clave)))}<br><small>${e(fuente)}</small></dd></div>`;
    }).join("");
    revision = `<section class="panel" data-b57-revision tabindex="-1" aria-labelledby="b57-revision-titulo"><div class="cabecera-panel"><h5 id="b57-revision-titulo">${e(t("ficha.revision_titulo"))}</h5></div><div class="cuerpo-panel"><dl class="resumen-expediente">${filas}</dl>
      ${st.paso === "enviando" ? `<p role="status">${e(t("ficha.registrando"))}</p>` : ""}${st.paso === "incierto" ? `<p class="nota-pendiente" role="alert">${e(t("ficha.incierto"))}</p>` : ""}${st.mensaje ? `<p class="mensaje-error" role="alert">${e(st.mensaje)}</p>` : ""}
      <div class="acciones-vista"><button type="button" class="boton-secundario" data-b57-accion="cambiar"${st.paso === "revision" ? "" : " disabled"}>${e(t("ficha.cambiar"))}</button>
      <button type="button" class="boton-primario" data-b57-accion="registrar"${st.paso === "enviando" ? " disabled" : ""}>${e(t(st.paso === "incierto" ? "ficha.reintentar" : "ficha.registrar"))}</button></div></div></section>`;
  }
  const aviso = `<p class="nota-pendiente">${e(t("ficha.detenida", { posicion: new Intl.NumberFormat(LOCALIZACION_PORTAL).format(c.orden_acta) }))}${st.propuesta.version_esperada > 0 ? ` ${e(t("ficha.rectificacion", { version: st.propuesta.version_esperada }))}` : ""}</p>`;
  const cuerpo = st.paso === "recibo" ? renderizarRecibo(st.recibo, e) : `${aviso}${st.paso === "edicion" && st.mensaje ? `<p class="mensaje-error" role="alert" tabindex="-1" data-b57-mensaje>${e(st.mensaje)}</p>` : ""}${edicion}${revision}`;
  const megas = new Intl.NumberFormat(LOCALIZACION_PORTAL, { maximumFractionDigits: 0 }).format(LIMITE_HUELLA_ARCHIVO_BYTES / (1024 * 1024));
  return `<section class="panel panel-separado" id="b57-comprobacion" data-b57-raiz="true" tabindex="-1" aria-labelledby="b57-titulo"><div class="cabecera-panel"><div><h4 id="b57-titulo">${e(t("ficha.titulo"))}</h4></div><details><summary aria-label="${e(t("ficha.ayuda_aria"))}">?</summary><p>${e(t("ficha.ayuda"))}</p><p>${e(t("ficha.ayuda_huella", { limite: megas }))}</p></details></div><div class="cuerpo-panel">${cuerpo}</div></section>`;
}

function enfocar(documento, selector) {
  globalThis.queueMicrotask?.(() => documento.querySelector?.(selector)?.focus?.());
}

/** Estado, eventos y envío del bloque; mismo patrón que los demás de la ficha. */
export function crearControladorComprobacion({ estado, renderizar, fetchImpl, cargarCatalogo, generarClave = () => `comprobacion:${globalThis.crypto.randomUUID()}` }) {
  async function cargar(modal) {
    if (!iniciarComprobacion(modal, estado.comprobacionOfrecimiento, estado.bolsaSeleccionada)) return false;
    renderizar();
    try {
      await prepararTextosComprobacion(cargarCatalogo);
    } catch {
      if (estado.modalFicha === modal) { modal.comprobacionB57.errorTextos = true; renderizar(); }
      return true;
    }
    if (estado.modalFicha !== modal) return true;
    renderizar();
    const raiz = globalThis.document?.querySelector?.("#b57-comprobacion");
    raiz?.scrollIntoView?.({ block: "start" });
    raiz?.focus?.({ preventScroll: true });
    return true;
  }

  async function enviar(modal, st, documento) {
    st.paso = "enviando"; st.mensaje = "";
    renderizar();
    const c = st.propuesta.contexto;
    const r = await registrarComprobacion(c.bolsa_ref, c.participacion_ref, st.cuerpo, st.clave, fetchImpl ? { fetchImpl } : {});
    if (estado.modalFicha !== modal || modal.comprobacionB57 !== st) return;
    if (r.ok) { st.paso = "recibo"; st.recibo = r.recibo; estado.comprobacionOfrecimiento = null; renderizar(); enfocar(documento, "[data-b57-recibo]"); return; }
    st.mensaje = mensajeError(r.codigo);
    // Un rechazo no tuvo efecto: el siguiente intento, corregido, lleva clave nueva.
    if (r.rechazo) { st.paso = "edicion"; st.clave = ""; st.cuerpo = ""; } else st.paso = "incierto";
    renderizar();
    enfocar(documento, r.rechazo ? "[data-b57-mensaje]" : '[data-b57-accion="registrar"]');
  }

  function manejarClick(evento, documento) {
    const control = evento.target?.closest?.("[data-b57-accion]");
    const modal = estado.modalFicha;
    const st = modal?.comprobacionB57;
    if (!control || !st) return false;
    evento.preventDefault();
    const accion = control.dataset.b57Accion;
    if (accion === "reintentar-textos") { void cargar(modal); return true; }
    // El portal decide la pantalla por el hash: el enlace solo lleva el foco al campo.
    if (accion === "ir-campo") {
      const campo = documento.getElementById?.(control.dataset.b57Campo);
      (campo?.matches?.("fieldset") ? campo.querySelector("input") : campo)?.focus?.();
      return true;
    }
    if (accion === "anadir-documento" && st.paso === "edicion" && st.documentos.length < MAXIMO_DOCUMENTOS) {
      const id = `d${st.siguiente++}`;
      st.documentos.push({ id, referencia: "", huella: "" });
      renderizar(); enfocar(documento, `#b57-docref-${id}`);
    } else if (accion === "quitar-documento" && st.paso === "edicion") {
      const id = control.dataset.b57Id;
      st.documentos = st.documentos.filter((d) => d.id !== id);
      for (const [clave, d] of Object.entries(st.docs)) if (d === id) st.docs[clave] = "";
      renderizar();
    } else if (accion === "cambiar" && st.paso === "revision") {
      st.paso = "edicion"; renderizar(); enfocar(documento, "[data-b57-form] button[type=submit]");
    } else if (accion === "registrar" && (st.paso === "revision" || st.paso === "incierto")) {
      if (!st.clave) { st.clave = generarClave(); st.cuerpo = JSON.stringify({ contexto: st.propuesta.contexto, hechos: st.hechos, version_esperada: st.propuesta.version_esperada }); }
      void enviar(modal, st, documento);
    }
    return true;
  }

  function manejarSubmit(evento, documento) {
    const formulario = evento.target?.closest?.("[data-b57-form]");
    const modal = estado.modalFicha;
    const st = modal?.comprobacionB57;
    if (!formulario || !st || st.paso !== "edicion") return false;
    evento.preventDefault();
    const { hechos, errores } = componerHechos(st, modal);
    st.errores = errores; st.mensaje = "";
    if (errores.length) { renderizar(); enfocar(documento, "[data-b57-errores]"); return true; }
    st.hechos = hechos; st.paso = "revision";
    renderizar(); enfocar(documento, "[data-b57-revision]");
    return true;
  }

  async function manejarCambio(evento) {
    const objetivo = evento.target;
    const st = estado.modalFicha?.comprobacionB57;
    if (!st || st.paso !== "edicion" || !objetivo?.closest?.("[data-b57-raiz]")) return;
    const d = objetivo.dataset ?? {};
    if (d.b57Hecho) st.valores[d.b57Hecho] = objetivo.value;
    else if (d.b57DocHecho) st.docs[d.b57DocHecho] = objetivo.value;
    else if (d.b57DocRef) { const x = st.documentos.find((y) => y.id === d.b57DocRef); if (x) x.referencia = objetivo.value.trim(); }
    else if (d.b57DocArchivo && evento.type === "change") {
      const x = st.documentos.find((y) => y.id === d.b57DocArchivo);
      if (!x) return;
      const lectura = (x.lectura ?? 0) + 1;
      Object.assign(x, { lectura, huella: "", nombre: "", calculando: true, errorArchivo: "" });
      const estadoTexto = objetivo.parentElement?.querySelector?.("[role=status]");
      if (estadoTexto) estadoTexto.textContent = t("huella.calculando");
      const archivo = objetivo.files?.length === 1 ? objetivo.files[0] : null;
      const r = archivo ? await calcularHuellaArchivo(archivo) : { ok: false, motivo: "sin_archivo" };
      if (x.lectura !== lectura) return;
      x.calculando = false;
      if (r.ok) { x.huella = r.huella; x.nombre = archivo.name; }
      else x.errorArchivo = t(`huella.${r.motivo}`, { limite: new Intl.NumberFormat(LOCALIZACION_PORTAL).format(LIMITE_HUELLA_ARCHIVO_BYTES / (1024 * 1024)) });
      if (estadoTexto?.isConnected) estadoTexto.textContent = r.ok ? t("huella.comprobado_nombre", { nombre: x.nombre }) : x.errorArchivo;
    }
  }

  function instalar(documento = globalThis.document) {
    documento.addEventListener("click", (evento) => { manejarClick(evento, documento); });
    documento.addEventListener("submit", (evento) => { manejarSubmit(evento, documento); });
    documento.addEventListener("input", manejarCambio);
    documento.addEventListener("change", manejarCambio);
  }

  return Object.freeze({ cargar, instalar, manejarClick, manejarSubmit, manejarCambio });
}
