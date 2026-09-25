/**
 * Correo personalizado del nuevo llamamiento (B7). La sustitución de los datos
 * de cada persona ocurre en el servidor; aquí solo se obtiene la plantilla del
 * catálogo, se insertan marcadores en el texto y se pide la vista previa de un
 * destinatario. Sin catálogo, el asistente conserva el correo literal anterior.
 */
import { traducirCorreoLlamamiento as tc } from "./portal-i18n-correo-llamamiento.js?v=20260925-correo-personalizado-v1";

export const RUTA_PLANTILLA_CORREO = "/api/vec/bolsa/llamamientos/emisiones/plantilla";
export const RUTA_VISTA_PREVIA_CORREO = "/api/vec/bolsa/llamamientos/emisiones/vista-previa";
const VERSION = /^bolsa-llamamiento-v[1-9][0-9]{0,3}$/;
const CLAVE_MARCADOR = /^[a-z][a-z0-9_]{0,39}$/;
const OPCIONES = Object.freeze({ credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error" });

function escapar(valor) {
  return String(valor ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);
}

const entero = (v, min, max) => Number.isSafeInteger(v) && v >= min && v <= max;

export function validarPlantillaCorreo(d) {
  if (!d || typeof d !== "object" || !VERSION.test(d.plantilla_version) || typeof d.personalizada !== "boolean"
    || !entero(d.limite, 1, 20000) || !entero(d.limite_asunto, 1, 998) || typeof d.asunto !== "string" || typeof d.cuerpo !== "string"
    || !d.asunto.trim() || !d.cuerpo.trim() || d.cuerpo.length > 4000 || !Array.isArray(d.marcadores)
    || d.marcadores.length > 32 || !d.marcadores.every((m) => m && CLAVE_MARCADOR.test(m.clave))) return null;
  return Object.freeze({ plantilla_version: d.plantilla_version, personalizada: d.personalizada, limite: d.limite, limite_asunto: d.limite_asunto,
    asunto: d.asunto, cuerpo: d.cuerpo, marcadores: Object.freeze(d.marcadores.map((m) => m.clave)) });
}

export async function consultarPlantillaCorreo({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") return { ok: false, status: 0 };
  try {
    const r = await fetchImpl(RUTA_PLANTILLA_CORREO, { ...OPCIONES, method: "GET", headers: { Accept: "application/json" } });
    const cuerpo = await r.json().catch(() => ({}));
    const datos = r.status === 200 ? validarPlantillaCorreo(cuerpo?.data) : null;
    return datos ? { ok: true, datos } : { ok: false, status: r.status };
  } catch {
    return { ok: false, status: 0 };
  }
}

const ERRORES_VISTA_PREVIA = Object.freeze({ plantilla_invalida: "error_plantilla_invalida", datos_incompletos: "error_datos_incompletos", correo_excede_limite: "error_correo_excede_limite", acceso_denegado: "error_acceso_denegado" });

export async function consultarVistaPreviaCorreo(peticion, { fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function" || !peticion?.bolsa_ref || !peticion?.participacion_ref || !peticion?.configuracion) return { ok: false, mensaje: tc("error_servicio") };
  try {
    const r = await fetchImpl(RUTA_VISTA_PREVIA_CORREO, { ...OPCIONES, method: "POST", headers: { Accept: "application/json", "Content-Type": "application/json" },
      body: JSON.stringify({ bolsa_ref: peticion.bolsa_ref, participacion_ref: peticion.participacion_ref, configuracion: peticion.configuracion }) });
    const cuerpo = await r.json().catch(() => ({}));
    const d = cuerpo?.data;
    if (r.status === 200 && d && typeof d.asunto === "string" && typeof d.cuerpo === "string" && entero(d.caracteres, 0, 100000) && entero(d.limite, 1, 20000)) {
      return { ok: true, datos: { asunto: d.asunto, cuerpo: d.cuerpo, caracteres: d.caracteres, limite: d.limite } };
    }
    const clave = ERRORES_VISTA_PREVIA[cuerpo?.error?.codigo] || (r.status === 403 ? "error_acceso_denegado" : "error_servicio");
    return { ok: false, mensaje: tc(clave) };
  } catch {
    return { ok: false, mensaje: tc("error_servicio") };
  }
}

/** Aplica la plantilla del catálogo sin pisar lo que RRHH ya haya escrito. */
export function aplicarPlantillaAlFlujo(flujo, plantilla) {
  if (!flujo || !plantilla) return;
  flujo.correo = plantilla;
  flujo.plantilla_version = plantilla.plantilla_version;
  if (!flujo.cuerpoBorrador) flujo.cuerpoBorrador = plantilla.cuerpo;
  if (!flujo.configuracion?.asunto) flujo.configuracion = { ...(flujo.configuracion || {}), asunto: plantilla.asunto };
}

/** Botones que insertan un marcador en el asunto o en el texto del correo. */
export function renderizarMarcadoresCorreo(flujo) {
  const correo = flujo?.correo;
  if (!correo?.personalizada || !correo.marcadores.length) return "";
  const botones = correo.marcadores.map((clave) => {
    const marcador = `{${clave}}`;
    return `<button type="button" class="boton-secundario" data-b7-correo="insertar" data-marcador="${escapar(clave)}" aria-label="${escapar(tc("marcador_insertar", { marcador }))}">${escapar(marcador)}</button>`;
  }).join(" ");
  return `<div class="campo-ancho acciones-fila" role="group" aria-label="${escapar(tc("marcadores_grupo"))}">${botones}</div>`;
}

function resultadoVistaPrevia(vista) {
  if (!vista) return "";
  if (vista.cargando) return `<p>${escapar(tc("vista_previa_cargando"))}</p>`;
  if (vista.error) return `<p class="mensaje-error">${escapar(vista.error)}</p>`;
  return `<dl class="resumen-expediente"><div class="fila-resumen"><dt>${escapar(tc("vista_previa_asunto"))}</dt><dd>${escapar(vista.asunto)}</dd></div>`
    + `<div class="fila-resumen"><dt>${escapar(tc("vista_previa_cuerpo"))}</dt><dd>${escapar(vista.cuerpo).replace(/\r?\n/g, "<br>")}</dd></div></dl>`
    + `<p class="estado-chip info">${escapar(tc("vista_previa_longitud", { caracteres: vista.caracteres, limite: vista.limite }))}</p>`;
}

/** Apartado del paso 4: elegir una persona seleccionada y ver su correo exacto. */
export function renderizarVistaPreviaCorreo(flujo, candidatos = []) {
  if (!flujo?.correo?.personalizada || !flujo.participaciones?.length) return "";
  const nombres = new Map((candidatos || []).map((c) => [c.participacion_ref, c.nombre_visible]));
  const elegida = flujo.vistaPrevia?.participacion || flujo.participaciones[0];
  const opciones = flujo.participaciones.map((ref, i) => `<option value="${escapar(ref)}" ${ref === elegida ? "selected" : ""}>${escapar(nombres.get(ref) || tc("vista_previa_persona", { numero: i + 1 }))}</option>`).join("");
  return `<section class="panel" aria-labelledby="b7-vista-previa-titulo"><div class="cabecera-panel"><h4 id="b7-vista-previa-titulo">${escapar(tc("vista_previa_titulo"))}</h4></div>`
    + `<div class="cuerpo-panel"><label>${escapar(tc("vista_previa_destinatario"))} <select data-b7-correo="destinatario">${opciones}</select></label> `
    + `<button type="button" class="boton-secundario" data-b7-correo="vista-previa">${escapar(tc("vista_previa_ver"))}</button>`
    + `<div data-b7-vista-previa-resultado aria-live="polite">${resultadoVistaPrevia(flujo.vistaPrevia)}</div></div></section>`;
}

export function crearControladorCorreoLlamamiento({ estado, renderizar, fetchImpl = globalThis.fetch?.bind(globalThis) } = {}) {
  let plantilla = null;
  let cargando = null;
  let ultimoCampo = "cuerpo";
  const flujoActual = () => estado?.filtrosBolsa?.nuevo_llamamiento;

  function prepararFlujo(flujo) {
    if (!flujo) return;
    if (plantilla) { aplicarPlantillaAlFlujo(flujo, plantilla); return; }
    cargando ||= consultarPlantillaCorreo({ fetchImpl }).then((res) => {
      cargando = null;
      if (!res.ok) return;
      plantilla = res.datos;
      const vigente = flujoActual();
      if (!vigente || vigente.enviando || vigente.recibo) return;
      aplicarPlantillaAlFlujo(vigente, plantilla);
      if ((Number(vigente.paso) || 1) < 3) renderizar?.();
    });
  }

  function insertar(documento, clave) {
    const formulario = documento.querySelector?.('[data-bolsa-form="b7-paso3"]');
    const campo = formulario?.querySelector?.(ultimoCampo === "asunto" ? 'input[name="asunto"]' : 'textarea[name="cuerpo"]');
    if (!campo || !CLAVE_MARCADOR.test(clave)) return;
    const texto = `{${clave}}`;
    const inicio = campo.selectionStart ?? campo.value.length;
    const fin = campo.selectionEnd ?? inicio;
    campo.value = campo.value.slice(0, inicio) + texto + campo.value.slice(fin);
    campo.focus?.();
    campo.setSelectionRange?.(inicio + texto.length, inicio + texto.length);
  }

  async function vistaPrevia(documento) {
    const flujo = flujoActual();
    if (!flujo?.configuracion || flujo.vistaPrevia?.cargando) return;
    const participacion = documento.querySelector?.('[data-b7-correo="destinatario"]')?.value || flujo.participaciones?.[0];
    if (!participacion || !flujo.participaciones?.includes(participacion)) return;
    const region = () => documento.querySelector?.("[data-b7-vista-previa-resultado]");
    const pintar = () => { const r = region(); if (r) r.innerHTML = resultadoVistaPrevia(flujo.vistaPrevia); };
    flujo.vistaPrevia = { participacion, cargando: true };
    pintar();
    const res = await consultarVistaPreviaCorreo({ bolsa_ref: estado.bolsaSeleccionada, participacion_ref: participacion, configuracion: flujo.configuracion }, { fetchImpl });
    if (flujoActual() !== flujo) return;
    flujo.vistaPrevia = res.ok ? { participacion, ...res.datos } : { participacion, error: res.mensaje };
    pintar();
  }

  function instalar(documento) {
    documento?.addEventListener?.("focusin", (evento) => {
      const nombre = evento.target?.closest?.('[data-bolsa-form="b7-paso3"]') ? evento.target.name : "";
      if (nombre === "asunto" || nombre === "cuerpo") ultimoCampo = nombre;
    });
    documento?.addEventListener?.("click", (evento) => {
      const boton = evento.target?.closest?.("[data-b7-correo]");
      const accion = boton?.dataset?.b7Correo;
      if (accion === "insertar") { evento.preventDefault?.(); insertar(documento, boton.dataset.marcador || ""); }
      else if (accion === "vista-previa") { evento.preventDefault?.(); void vistaPrevia(documento); }
    });
  }

  return Object.freeze({ instalar, prepararFlujo, vistaPrevia, insertar });
}
