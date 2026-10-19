import { crearTraductorAuditoria } from "./i18n.js?v=VEC_WEB_VERSION";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";
import { ZONA_HORARIA_PORTAL } from "../../portal-i18n.js?v=VEC_WEB_VERSION";
// Fases y estados de las peticiones: un solo catálogo para todas las pantallas.
import { nombreEstado, nombreFaseRRHH } from "../contratacion-temporal/i18n-fases-rrhh.js?v=VEC_WEB_VERSION";

const escapar = (v) => String(v ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
  .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
const texto = (v, n = 512) => typeof v === "string" && v.length <= n && !/[\x00-\x1f\x7f]/u.test(v);
const numeroHumano = (v) => typeof v === "string" && v.length > 0 && v.length <= 80
  && !/[\x00-\x1f\x7f]/u.test(v);
const fecha = (v) => typeof v === "string" && /^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d+)?Z$/u.test(v) && Number.isFinite(Date.parse(v));
const digest = (v) => v == null || v === "" || typeof v === "string" && /^[a-f0-9]{64}$/iu.test(v);
const formatoFecha = new Intl.DateTimeFormat(LOCALIZACION_ACTUAL, { dateStyle: "medium", timeStyle: "medium", timeZone: ZONA_HORARIA_PORTAL });
const formatoNumero = new Intl.NumberFormat(LOCALIZACION_ACTUAL);
const formatoPartesMadrid = new Intl.DateTimeFormat(/* localización técnica */ "en-GB", { timeZone: ZONA_HORARIA_PORTAL,
  year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", hourCycle: "h23" });
const DIA_MS = 86400000;

/** Etiquetas exclusivas de la muestra ficticia; nunca se infieren nombres reales de referencias. */
const PERSONAS_EJEMPLO = Object.freeze({ per_1: "Carmen Molina" });
const EXPEDIENTES_EJEMPLO = Object.freeze({ exp_1: "EXP-2026-001" });
// Código de acción de cada fuente (origen de la versión en peticiones,
// operación en Bolsa) → clave del catálogo de Auditoría.
const ACCIONES = Object.freeze({ "vec.auditoria.consultar": "accion_auditoria_consultar", "relacion.actualizada": "accion_relacion_actualizada",
  "bolsa.participacion.cambiar": "accion_participacion_cambiada",
  alta_o2: "accion_ct_alta", analisis_o3: "accion_ct_analisis", cobertura_o4: "accion_ct_cobertura",
  asignacion_o5: "accion_ct_asignacion", informe_juridico_o5: "accion_ct_informe_juridico",
  fiscalizacion_o5: "accion_ct_fiscalizacion", subsanacion_reparos_v1: "accion_ct_subsanacion",
  propuesta_formalizacion_o6: "accion_ct_propuesta", resolucion_formalizacion_o6: "accion_ct_resolucion",
  anotacion_administrativa_ct86: "accion_ct_anotacion",
  pausar: "accion_bolsa_pausar", reactivar: "accion_bolsa_reactivar", excluir: "accion_bolsa_excluir" });
const RESULTADOS = Object.freeze({ confirmado: "resultado_confirmado", denegado: "resultado_denegado", ok: "resultado_confirmado" });
// Motivos que la fuente publica tal cual (Bolsa B56); el resto queda en el detalle técnico.
const MOTIVOS = Object.freeze({ "Constitución de bolsa": "motivo_constitucion_bolsa",
  "Motivo reservado en Bolsa": "motivo_reservado" });
const CAMPOS = Object.freeze({ fase: "campo_fase", estado: "campo_estado", situacion: "campo_situacion",
  fecha_disponible: "campo_fecha_disponible", datos_contacto: "campo_datos_contacto", correo: "campo_correo",
  telefono_1: "campo_telefono", telefono_2: "campo_telefono" });
const SITUACIONES = Object.freeze(["disponible", "no_disponible", "trabajando", "pendiente_incorporacion",
  "renuncia", "excluido", "disponible_desde"]);
const PROTEGIDOS = new Set(["datos_contacto", "correo", "telefono_1", "telefono_2"]);

function claveAccion(accion) {
  if (Object.hasOwn(ACCIONES, accion)) return ACCIONES[accion];
  if (accion.startsWith("situacion:")) return "accion_bolsa_situacion";
  if (accion.startsWith("valor:")) return "accion_bolsa_dato";
  return "accion_otra";
}

/** Valor legible de un campo conocido; «» si no hay forma segura de mostrarlo. */
function valorCampo(campo, valor, t) {
  if (campo === "fase") return nombreFaseRRHH(valor);
  if (campo === "estado") return nombreEstado(valor);
  if (campo === "situacion") return SITUACIONES.includes(valor) ? t(`situacion_${valor}`) : "";
  if (campo === "fecha_disponible") return fecha(valor) ? formatoFecha.format(new Date(valor)) : "";
  return "";
}

/** «Fase: Solicitud → Análisis RRHH» por cada dato que cambia; sin códigos. */
function resumenCambio(r, t) {
  if (!r.datos_disponibles) return t("sin_valores");
  const partes = [];
  for (const campo of new Set([...Object.keys(r.antes), ...Object.keys(r.despues)])) {
    const antes = r.antes[campo], despues = r.despues[campo];
    if (antes === despues) continue;
    if (!Object.hasOwn(CAMPOS, campo)) { partes.push(t("cambio_otro_dato")); continue; }
    if (PROTEGIDOS.has(campo)) { partes.push(t("cambio_protegido", { campo: t(CAMPOS[campo]) })); continue; }
    const anterior = antes === undefined ? "" : valorCampo(campo, antes, t) || t("sin_dato");
    const nuevo = despues === undefined ? t("sin_valor") : valorCampo(campo, despues, t) || t("sin_dato");
    // Dos fases del servidor pueden tener el mismo nombre para RRHH.
    if (anterior === nuevo) continue;
    partes.push(anterior ? t("cambio_de_a", { campo: t(CAMPOS[campo]), antes: anterior, despues: nuevo })
      : t("cambio_valor", { campo: t(CAMPOS[campo]), valor: nuevo }));
  }
  return partes.length ? [...new Set(partes)].join("; ") : t("sin_cambios_visibles");
}

function presentarRegistroAuditoria(registro, t, ejemplo, contexto = {}) {
  const actor = ejemplo && Object.hasOwn(PERSONAS_EJEMPLO, registro.actor_ref)
    ? PERSONAS_EJEMPLO[registro.actor_ref]
    : registro.actor_ref.startsWith("sistema:") ? t("persona_sistema") : t("persona_no_disponible");
  let expediente = t("numero_no_disponible");
  if (ejemplo && Object.hasOwn(EXPEDIENTES_EJEMPLO, registro.expediente_ref)) {
    expediente = EXPEDIENTES_EJEMPLO[registro.expediente_ref];
  } else if (contexto.expedienteRef && registro.expediente_ref === contexto.expedienteRef) {
    expediente = numeroHumano(contexto.numeroVisible) ? contexto.numeroVisible
      : t(contexto.fuenteContexto === "bolsa" ? "relacion_participacion_actual" : "relacion_expediente_actual");
  }
  const relacionado = registro.recibo_ref ? t("relacion_con_justificante", { relacion: expediente }) : expediente;
  const resultado = Object.hasOwn(RESULTADOS, registro.resultado) ? RESULTADOS[registro.resultado] : null;
  const motivo = registro.accion === "vec.auditoria.consultar" && registro.modulo_id === "auditoria" ? registro.motivo || t("sin_dato")
    : !registro.motivo ? t("sin_dato")
    : Object.hasOwn(MOTIVOS, registro.motivo) ? t(MOTIVOS[registro.motivo]) : t("motivo_en_detalle");
  return Object.freeze({ actor, expediente: relacionado, accion: t(claveAccion(registro.accion)),
    resultado: t(resultado || "resultado_otro"), motivo, cambio: resumenCambio(registro, t) });
}

function presentarExpedienteAuditoria(ref, t, ejemplo, numeroVisible) {
  return ejemplo && Object.hasOwn(EXPEDIENTES_EJEMPLO, ref)
    ? EXPEDIENTES_EJEMPLO[ref]
    : numeroHumano(numeroVisible) ? numeroVisible : t("numero_no_disponible");
}

/** Convierte una hora local de Madrid; prueba ambos lados del cambio horario y rechaza huecos. */
function instanteMadrid(valor) {
  if (!/^\d{4}-\d\d-\d\dT\d\d:\d\d$/u.test(valor)) return NaN;
  const local = Date.parse(`${valor}Z`);
  if (!Number.isFinite(local)) return NaN;
  const candidatos = [];
  for (const prueba of [local - DIA_MS, local, local + DIA_MS]) {
    const partes = Object.fromEntries(formatoPartesMadrid.formatToParts(new Date(prueba)).map(({ type, value }) => [type, value]));
    const desplazamiento = Date.parse(`${partes.year}-${partes.month}-${partes.day}T${partes.hour}:${partes.minute}:00Z`) - prueba;
    const instante = local - desplazamiento;
    const real = Object.fromEntries(formatoPartesMadrid.formatToParts(new Date(instante)).map(({ type, value }) => [type, value]));
    if (`${real.year}-${real.month}-${real.day}T${real.hour}:${real.minute}` === valor) candidatos.push(instante);
  }
  return candidatos.length ? Math.min(...candidatos) : NaN;
}

function proyeccion(v) {
  if (!v || typeof v !== "object" || Array.isArray(v) || Object.keys(v).length > 20
    || Object.entries(v).some(([k, s]) => !/^[a-z][a-z0-9_.-]{0,79}$/u.test(k) || !texto(s, 500))) throw new TypeError("proyección inválida");
  return Object.freeze({ ...v });
}

/** Valida toda la página antes de representar cualquier registro. */
export function validarRespuestaAuditoria(v) {
  if (!v || !Array.isArray(v.registros) || v.registros.length > 100 || typeof v.siguiente_cursor !== "string"
    || v.siguiente_cursor.length > 512 || /[\s\\/?%*]/u.test(v.siguiente_cursor)) throw new TypeError("respuesta de Auditoría inválida");
  const ids = new Set();
  const registros = v.registros.map((r) => {
    if (!r || !texto(r.id, 128) || !r.id || ids.has(r.id) || !texto(r.modulo_id, 128)
      || !texto(r.accion, 128) || !texto(r.actor_ref) || !fecha(r.ocurrido_en)
      || !texto(r.resultado, 80) || !texto(r.expediente_ref) || !texto(r.recibo_ref)
      || !digest(r.antes_sha256) || !digest(r.despues_sha256) || !texto(r.motivo, 500)
      || !texto(r.fuente, 128) || typeof r.datos_disponibles !== "boolean") throw new TypeError("registro de Auditoría inválido");
    for (const clave of ["perfil_ref", "finalidad_ref", "canal"]) {
      if (r[clave] !== undefined && !texto(r[clave], 512)) throw new TypeError("dato de Auditoría inválido");
    }
    const antes = proyeccion(r.antes), despues = proyeccion(r.despues);
    if (!r.datos_disponibles && (Object.keys(antes).length || Object.keys(despues).length)) throw new TypeError("proyección contradictoria");
    ids.add(r.id);
    return Object.freeze({ ...r, antes, despues });
  });
  if (v.auditoria_ref !== undefined && (!texto(v.auditoria_ref, 128) || !v.auditoria_ref || /[\x00-\x20\x7f]/u.test(v.auditoria_ref))) throw new TypeError("recibo de consulta inválido");
  return Object.freeze({ registros: Object.freeze(registros), siguienteCursor: v.siguiente_cursor || "",
    ...(v.auditoria_ref ? { auditoriaRef: v.auditoria_ref } : {}) });
}

function bloqueValores(t, clave, valores, huella) {
  const filas = Object.entries(valores).map(([k, v]) => `<div><dt>${escapar(k)}</dt><dd>${escapar(v)}</dd></div>`).join("");
  return `<section class="auditoria-proyeccion"><h5>${escapar(t(clave))}</h5>
    ${filas ? `<dl>${filas}</dl>` : `<p>${escapar(t("sin_valores"))}</p>`}
    ${huella ? `<p class="auditoria-huella">${escapar(t("huella"))} <code>${escapar(huella)}</code></p>` : ""}</section>`;
}

function fila(t, r, ejemplo, contexto) {
  const visible = presentarRegistroAuditoria(r, t, ejemplo, contexto);
  return `<tr><td><time datetime="${escapar(r.ocurrido_en)}">${escapar(formatoFecha.format(new Date(r.ocurrido_en)))}</time></td>
    <td>${escapar(visible.actor)}</td><td>${escapar(visible.accion)}</td><td>${escapar(visible.cambio)}</td>
    <td>${escapar(visible.motivo)}</td><td>${escapar(visible.expediente)}</td>
    <td><details><summary>${escapar(t("ver_cambio"))}</summary><div class="auditoria-detalle">
      <h4>${escapar(t("detalle_tecnico"))}</h4>
      <p><span class="auditoria-resultado">${escapar(t("resultado"))}: ${escapar(visible.resultado)}</span></p>
      <dl class="auditoria-metadatos">
        ${[["campo_registro_ref", r.id], ["campo_actor_ref", r.actor_ref], ["campo_accion_ref", r.accion],
          ["campo_resultado_ref", r.resultado], ["campo_expediente_ref", r.expediente_ref],
          ["campo_modulo", r.modulo_id], ["campo_recibo", r.recibo_ref],
          ["campo_fuente", r.fuente], ["campo_motivo", r.motivo]].map(([k, v]) =>
            `<div><dt>${escapar(t(k))}</dt><dd>${escapar(v || t("sin_dato"))}</dd></div>`).join("")}
        ${[["campo_perfil", r.perfil_ref], ["campo_finalidad", r.finalidad_ref], ["campo_canal", r.canal]].filter(([,v]) => v).map(([k,v]) => `<div><dt>${escapar(t(k))}</dt><dd>${escapar(v)}</dd></div>`).join("")}
      </dl><div class="auditoria-comparacion">${bloqueValores(t, "antes", r.antes, r.antes_sha256)}
        ${bloqueValores(t, "despues", r.despues, r.despues_sha256)}</div>
      ${r.datos_disponibles ? "" : `<p class="auditoria-dato-ausente">${escapar(t("valores_no_disponibles"))}</p>`}
    </div></details></td></tr>`;
}

/** Marcado puro para verificar estados y escape sin consultar datos. */
export function renderizarVistaAuditoria({ estado = "no_configurado", ayudaAbierta = false, habilitada = false,
  expedienteRef = "", numeroVisible = "", fuenteContexto = "", ejemplo = false, filtros = {}, registros = [], pagina = 1, siguienteCursor = "", puedeAnterior = false, auditoriaRef = "",
  seleccionObjetivo = false, objetivos = [], despuesObjetivos = "", buscando = false } = {}) {
  const t = crearTraductorAuditoria();
  const mensaje = t(`estado_${["no_configurado", "cargando_opciones", "esperando", "cargando", "disponible", "vacio", "denegado", "no_disponible", "error", "invalido", "vacio_objetivos"].includes(estado) ? estado : "error"}`);
  const bloqueada = !habilitada;
  return `<section class="modulo-auditoria-rrhh" data-auditoria-vista data-estado="${escapar(estado)}">
    <header class="auditoria-cabecera"><h2>${escapar(t("titulo"))}</h2>
      <button type="button" class="boton-secundario auditoria-ayuda-boton" data-auditoria-ayuda aria-controls="auditoria-ayuda"
        aria-expanded="${ayudaAbierta}" aria-label="${escapar(t("ayuda_aria"))}">?</button></header>
    <section id="auditoria-ayuda" class="panel auditoria-ayuda" ${ayudaAbierta ? "" : "hidden"}>
      <div class="cabecera-panel"><h3>${escapar(t("ayuda_titulo"))}</h3></div>
      <div class="cuerpo-panel"><p>${escapar(t("ayuda_alcance"))}</p><p>${escapar(t("ayuda_lectura"))}</p></div></section>
    <section class="panel auditoria-filtro-panel" aria-labelledby="auditoria-filtros-titulo">
      <div class="cabecera-panel"><h3 id="auditoria-filtros-titulo">${escapar(t("filtros_titulo"))}</h3></div>
      <div class="cuerpo-panel">
        ${seleccionObjetivo ? `<form data-auditoria-busqueda class="auditoria-filtros">
          <label>${escapar(t("tipo_objetivo"))}<select name="fuente_objetivo" ${bloqueada ? "disabled" : ""}>
            ${["ct", "bolsa", "persona"].map((f) => `<option value="${f}" ${f === fuenteContexto ? "selected" : ""}>${escapar(t(`objetivo_${f}`))}</option>`).join("")}</select></label>
          <label>${escapar(t("motivo_consulta"))}<input name="motivo_consulta" required minlength="3" maxlength="500" value="${escapar(filtros.motivo_consulta || "")}" ${bloqueada ? "disabled" : ""}></label>
          <label>${escapar(t("buscar_objetivo"))}<input name="busqueda" maxlength="80" value="${escapar(filtros.busqueda || "")}" ${bloqueada ? "disabled" : ""}></label>
          <button type="submit" class="boton-secundario" ${bloqueada || buscando ? "disabled" : ""}>${escapar(t(buscando ? "buscando_objetivos" : "buscar"))}</button>
        </form>
        ${objetivos.length ? `<label class="auditoria-elegir">${escapar(t("elegir_objetivo"))}<select data-auditoria-objetivo>
          <option value="">${escapar(t("elegir_objetivo"))}</option>${objetivos.map((o) => `<option value="${escapar(o.referencia)}" ${o.referencia === expedienteRef ? "selected" : ""}>${escapar(o.etiqueta)}</option>`).join("")}</select></label>` : ""}
        ${despuesObjetivos ? `<button type="button" class="boton-secundario" data-auditoria-mas-objetivos ${buscando ? "disabled" : ""}>${escapar(t("mas_objetivos"))}</button>` : ""}` : ""}
        <p class="auditoria-expediente"><strong>${escapar(t(seleccionObjetivo ? `objetivo_${fuenteContexto}` : fuenteContexto === "bolsa" ? "participacion" : "expediente"))}:</strong> ${escapar(expedienteRef ? presentarExpedienteAuditoria(expedienteRef, t, ejemplo, numeroVisible) : t("sin_expediente"))}</p>
        <form data-auditoria-filtros class="auditoria-filtros">
        ${[["desde", "desde"], ["hasta", "hasta"]].map(([name, label]) =>
          `<label>${escapar(t(label))}<input name="${name}" type="datetime-local" value="${escapar(filtros[name] || "")}" ${bloqueada ? "disabled" : ""}></label>`).join("")}
        <button type="submit" class="boton-primario" ${bloqueada || estado === "cargando" || (seleccionObjetivo && (!expedienteRef || !filtros.motivo_consulta)) ? "disabled" : ""}>${escapar(t("consultar"))}</button>
      </form></div></section>
    <section class="panel auditoria-resultados" aria-labelledby="auditoria-resultados-titulo">
      <div class="cabecera-panel"><h3 id="auditoria-resultados-titulo">${escapar(t("resultados_titulo"))}</h3>
        <span class="estado-chip ${["denegado", "error"].includes(estado) ? "peligro" : estado === "disponible" ? "exito" : "aviso"}">${escapar(estado === "no_disponible" ? t("estado_no_disponible_titulo") : mensaje)}</span></div>
      <div class="cuerpo-panel"><p class="auditoria-estado-texto" role="status" aria-live="polite">${escapar(mensaje)}</p>
      ${auditoriaRef ? `<details data-auditoria-recibo><summary>${escapar(t("consulta_registrada"))}</summary><p>${escapar(t("referencia_consulta"))} <code>${escapar(auditoriaRef)}</code></p></details>` : ""}
      ${["error", "no_disponible"].includes(estado) && (expedienteRef || seleccionObjetivo) ? `<button type="button" class="boton-secundario auditoria-reintentar" data-auditoria-reintentar>${escapar(t("reintentar"))}</button>` : ""}
      ${estado === "disponible" ? `<div class="auditoria-tabla" role="region" tabindex="0" aria-label="${escapar(t("tabla_aria"))}">
        <table><thead><tr>${["fecha", "actor", "accion", "cambio", "motivo", "relacionado", "detalle"].map((k) => `<th scope="col">${escapar(t(k))}</th>`).join("")}</tr></thead>
        <tbody>${registros.map((r) => fila(t, r, ejemplo, { expedienteRef, numeroVisible, fuenteContexto })).join("")}</tbody></table></div>` : ""}
      ${["disponible", "vacio"].includes(estado) ? `<nav class="auditoria-paginacion" aria-label="${escapar(t("paginacion"))}">
        <button type="button" class="boton-secundario" data-auditoria-anterior ${puedeAnterior ? "" : "disabled"}>${escapar(t("anterior"))}</button>
        <span>${escapar(t("pagina", { numero: formatoNumero.format(pagina) }))}</span>
        <button type="button" class="boton-secundario" data-auditoria-siguiente ${siguienteCursor ? "" : "disabled"}>${escapar(t("siguiente"))}</button>
      </nav>` : ""}</div></section></section>`;
}

/** Expediente procede de navegación ya autorizada; opciones del GET autenticado. */
export function montarVistaAuditoria({ raiz, fuente, expedienteRef = "", numeroVisible = "", fuenteContexto = "", anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.replaceChildren || typeof anunciar !== "function" ||
    (fuente !== undefined && (typeof fuente.consultar !== "function" || typeof fuente.obtenerOpciones !== "function"))
    || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("vista de Auditoría no disponible");
  const t = crearTraductorAuditoria();
  const puedeSeleccionar = typeof fuente?.buscarObjetivos === "function";
  let seleccionObjetivo = false, objetivos = [], despuesObjetivos = "", buscando = false;
  let expedienteValido = typeof expedienteRef === "string" && expedienteRef.length > 0 && expedienteRef.length <= 512
    && !/[\x00-\x20\x7f*?%\\/]/u.test(expedienteRef) && !expedienteRef.includes("..");
  let fuenteValida = ["ct", "bolsa", "persona"].includes(fuenteContexto);
  if (puedeSeleccionar && !fuenteValida) { fuenteContexto = "ct"; fuenteValida = true; }
  let habilitada = false, opciones = null;
  let activa = true, ayudaAbierta = false, estado = fuente && (puedeSeleccionar || expedienteValido && fuenteValida) ? "cargando_opciones" : "no_configurado";
  const ahoraApertura = Date.now();
  let auditoriaRef = "";
  let filtros = {}, registros = [], siguienteCursor = "", cursores = [""], controlador, secuencia = 0;
  if (puedeSeleccionar && globalThis.location?.href) {
    const url = new URL(globalThis.location.href);
    const tipo = url.searchParams.get("auditoria_fuente"), ref = url.searchParams.get("auditoria_objetivo");
    if (tipo === "persona") {
      fuenteContexto = "persona"; url.searchParams.delete("auditoria_objetivo");
      globalThis.history?.replaceState?.(null, "", url);
    }
    if (["ct", "bolsa"].includes(tipo) && ref && ref.length <= 512 && !/[\x00-\x20\x7f*?%\\/]/u.test(ref) && !ref.includes("..")) {
      fuenteContexto = tipo; expedienteRef = ref; expedienteValido = true; fuenteValida = true;
    }
    for (const campo of ["desde", "hasta"]) {
      const valor = url.searchParams.get(`auditoria_${campo}`);
      if (valor && Number.isFinite(instanteMadrid(valor))) filtros[campo] = valor;
    }
  }
  const pintar = () => { if (activa) raiz.innerHTML = renderizarVistaAuditoria({
    estado, ayudaAbierta, habilitada, expedienteRef: expedienteValido && fuenteValida ? expedienteRef : "",
    numeroVisible: expedienteValido && fuenteValida && numeroHumano(numeroVisible) ? numeroVisible : "",
    fuenteContexto: fuenteValida ? fuenteContexto : "",
    ejemplo: opciones?.es_ejemplo === true, filtros, registros, pagina: cursores.length,
    siguienteCursor, puedeAnterior: cursores.length > 1, auditoriaRef, seleccionObjetivo, objetivos, despuesObjetivos, buscando,
  }); };
  const cancelar = () => { auditoriaRef = ""; ++secuencia; controlador?.abort(); controlador = undefined; };
  async function cargarOpciones() {
    cancelar(); controlador = new AbortController(); const signal = controlador.signal, actual = secuencia;
    try {
      const recibidas = await fuente.obtenerOpciones({ signal });
      if (!activa || signal.aborted || secuencia !== actual) return;
      if (!recibidas || typeof recibidas.finalidad_ref !== "string" || !recibidas.finalidad_ref
        || typeof recibidas.motivo_ref !== "string" || !recibidas.motivo_ref
        || typeof recibidas.es_ejemplo !== "boolean" || !Array.isArray(recibidas.fuentes)) throw new TypeError("opciones incompatibles");
      if (!recibidas.fuentes.includes(fuenteContexto)) {
        opciones = null; habilitada = false; registros = []; estado = "denegado"; pintar(); return;
      }
      opciones = recibidas; habilitada = true; seleccionObjetivo = puedeSeleccionar && recibidas.seleccion_objetivo === true;
      if (seleccionObjetivo) { estado = "esperando"; pintar(); } else { void consultar(); }
    } catch (error) {
      if (!activa || signal.aborted || secuencia !== actual) return;
      opciones = null; habilitada = false; registros = [];
      estado = error?.codigo === "denegado" ? "denegado" : error?.codigo === "no_disponible" ? "no_disponible" : "error"; pintar();
    }
  }
  async function consultar() {
    if (!habilitada || (seleccionObjetivo && (!expedienteValido || !filtros.motivo_consulta))) return;
    cancelar(); controlador = new AbortController(); const signal = controlador.signal, actual = secuencia;
    estado = "cargando"; registros = []; siguienteCursor = ""; pintar();
    try {
      const inicio = filtros.desde ? instanteMadrid(filtros.desde) : ahoraApertura - 30 * DIA_MS;
      const fin = filtros.hasta ? instanteMadrid(filtros.hasta) : ahoraApertura;
      if (!Number.isFinite(inicio) || !Number.isFinite(fin) || fin <= inicio || fin - inicio > 31 * DIA_MS) {
        estado = "invalido"; pintar(); anunciar(t("estado_invalido"), "error"); return;
      }
      const respuesta = validarRespuestaAuditoria(await fuente.consultar({
        fuente: fuenteContexto, expediente_ref: expedienteRef, actor_ref: "",
        desde: new Date(inicio).toISOString(), hasta: new Date(fin).toISOString(),
        finalidad_ref: opciones.finalidad_ref, motivo_ref: opciones.motivo_ref, cursor: cursores.at(-1),
        ...(seleccionObjetivo ? { motivo_consulta: filtros.motivo_consulta } : {}),
      }, { signal }));
      if (!activa || signal.aborted || secuencia !== actual) return;
      registros = respuesta.registros; siguienteCursor = respuesta.siguienteCursor; auditoriaRef = respuesta.auditoriaRef || "";
      estado = registros.length ? "disponible" : "vacio"; pintar(); anunciar(t(`estado_${estado}`), "info");
    } catch (error) {
      if (!activa || signal.aborted || secuencia !== actual) return;
      registros = []; siguienteCursor = "";
      estado = error?.codigo === "denegado" ? "denegado" : error?.codigo === "no_disponible" ? "no_disponible" : "error";
      pintar(); anunciar(t(`estado_${estado}`), "error");
    }
  }
  async function buscarObjetivos(despuesRef = "") {
    if (!habilitada || !seleccionObjetivo || !filtros.motivo_consulta) return;
    cancelar(); controlador = new AbortController(); const signal = controlador.signal, actual = secuencia;
    buscando = true; estado = "cargando"; registros = []; siguienteCursor = ""; pintar();
    try {
      const respuesta = await fuente.buscarObjetivos({ fuente: fuenteContexto, busqueda: filtros.busqueda || "",
        despues_ref: despuesRef, finalidad_ref: opciones.finalidad_ref, motivo_ref: opciones.motivo_ref,
        motivo_consulta: filtros.motivo_consulta }, { signal });
      if (!activa || signal.aborted || secuencia !== actual) return;
      objetivos = respuesta.objetivos; despuesObjetivos = respuesta.despues_ref;
      estado = objetivos.length ? "esperando" : "vacio_objetivos";
      buscando = false; pintar();
    } catch (error) {
      if (!activa || signal.aborted || secuencia !== actual) return;
      objetivos = []; despuesObjetivos = ""; buscando = false;
      estado = error?.codigo === "denegado" ? "denegado" : error?.codigo === "no_disponible" ? "no_disponible" : "error"; pintar();
    }
  }
  function conservarFiltroURL() {
    if (!seleccionObjetivo || !globalThis.location || !globalThis.history?.replaceState) return;
    const url = new URL(globalThis.location.href);
    for (const [clave, valor] of Object.entries({ auditoria_fuente: fuenteContexto, auditoria_objetivo: fuenteContexto === "persona" ? "" : expedienteRef,
      auditoria_desde: filtros.desde || "", auditoria_hasta: filtros.hasta || "" })) {
      if (valor) url.searchParams.set(clave, valor); else url.searchParams.delete(clave);
    }
    globalThis.history.replaceState(null, "", url);
  }
  const pulsar = (evento) => {
    if (evento.target.closest?.("[data-auditoria-ayuda]")) {
      ayudaAbierta = !ayudaAbierta; pintar(); raiz.querySelector("[data-auditoria-ayuda]")?.focus();
    } else if (evento.target.closest?.("[data-auditoria-siguiente]") && siguienteCursor) {
      cursores.push(siguienteCursor); void consultar();
    } else if (evento.target.closest?.("[data-auditoria-anterior]") && cursores.length > 1) {
      cursores.pop(); void consultar();
    } else if (evento.target.closest?.("[data-auditoria-mas-objetivos]") && despuesObjetivos) {
      void buscarObjetivos(despuesObjetivos);
    } else if (evento.target.closest?.("[data-auditoria-reintentar]") && fuente && (seleccionObjetivo || expedienteValido && fuenteValida)) {
      cursores = [""]; estado = "cargando_opciones"; pintar(); void cargarOpciones();
    }
  };
  const enviar = (evento) => {
    if (seleccionObjetivo && evento.target.matches?.("[data-auditoria-busqueda]")) {
      evento.preventDefault(); if (!habilitada) return;
      const datos = new FormData(evento.target);
      fuenteContexto = String(datos.get("fuente_objetivo"));
      filtros = { ...filtros, busqueda: String(datos.get("busqueda") || "").trim(), motivo_consulta: String(datos.get("motivo_consulta") || "").trim() };
      expedienteRef = ""; expedienteValido = false; numeroVisible = ""; cursores = [""];
      conservarFiltroURL(); void buscarObjetivos(); return;
    }
    if (!evento.target.matches?.("[data-auditoria-filtros]")) return;
    evento.preventDefault(); if (!habilitada) return;
    const datos = new FormData(evento.target);
    filtros = { ...filtros, ...Object.fromEntries(["desde", "hasta"].map((k) => [k, String(datos.get(k) || "").trim()])) };
    cursores = [""]; conservarFiltroURL();
    void consultar();
  };
  const elegir = (evento) => {
    if (seleccionObjetivo && evento.target.matches?.('[data-auditoria-busqueda] [name="fuente_objetivo"]')) {
      cancelar(); fuenteContexto = evento.target.value; expedienteRef = ""; numeroVisible = ""; expedienteValido = false;
      objetivos = []; despuesObjetivos = ""; registros = []; siguienteCursor = ""; cursores = [""]; buscando = false;
      estado = "esperando"; conservarFiltroURL(); pintar(); return;
    }
    if (!evento.target.matches?.("[data-auditoria-objetivo]") || !seleccionObjetivo) return;
    cancelar(); const elegido = objetivos.find((o) => o.referencia === evento.target.value);
    expedienteRef = elegido?.referencia || ""; numeroVisible = elegido?.etiqueta || ""; expedienteValido = !!elegido;
    registros = []; siguienteCursor = ""; cursores = [""]; estado = "esperando"; conservarFiltroURL(); pintar();
  };
  const cambiar = (evento) => {
    if (seleccionObjetivo && evento.target.matches?.("[data-auditoria-busqueda] input")) {
      const campo = evento.target.name;
      if (!["busqueda", "motivo_consulta"].includes(campo)) return;
      cancelar(); filtros = { ...filtros, [campo]: evento.target.value };
      registros = []; siguienteCursor = ""; cursores = [""]; buscando = false; estado = "esperando";
      if (campo === "busqueda") { objetivos = []; despuesObjetivos = ""; expedienteRef = ""; expedienteValido = false; }
      const inicio = evento.target.selectionStart, fin = evento.target.selectionEnd;
      pintar(); const nuevo = raiz.querySelector?.(`[data-auditoria-busqueda] [name="${campo}"]`);
      nuevo?.focus?.(); if (inicio !== null && fin !== null) nuevo?.setSelectionRange?.(inicio, fin);
      return;
    }
    if (!evento.target.matches?.("[data-auditoria-filtros] input") || !habilitada) return;
    cancelar(); registros = []; siguienteCursor = ""; cursores = [""]; estado = "esperando";
    const campo = evento.target.name;
    if (["desde", "hasta"].includes(campo)) filtros = { ...filtros, [campo]: evento.target.value };
    const inicio = evento.target.selectionStart, fin = evento.target.selectionEnd;
    pintar();
    const nuevo = raiz.querySelector?.(`[data-auditoria-filtros] [name="${campo}"]`);
    nuevo?.focus?.();
    try { if (inicio !== null && fin !== null) nuevo?.setSelectionRange?.(inicio, fin); } catch {}
  };
  raiz.addEventListener("click", pulsar); raiz.addEventListener("submit", enviar);
  raiz.addEventListener("input", cambiar); raiz.addEventListener("change", elegir); pintar();
  if (fuente && (puedeSeleccionar || expedienteValido && fuenteValida)) void cargarOpciones();
  const desmontar = () => { if (!activa) return; activa = false; cancelar(); raiz.removeEventListener("click", pulsar);
    raiz.removeEventListener("submit", enviar); raiz.removeEventListener("input", cambiar); raiz.removeEventListener("change", elegir); raiz.replaceChildren(); };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
