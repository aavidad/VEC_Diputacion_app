/**
 * Confirmación de la incorporación por el centro (dudas 11 y 12 de RRHH).
 *
 * Lista los expedientes de las peticiones del centro y, en los que están en
 * nombramiento, permite confirmar la fecha de incorporación con el documento
 * que la acredita (el tipo lo fija el catálogo del servidor: toma de posesión
 * o contrato firmado). Del documento solo viajan su referencia y la huella
 * calculada en este equipo; el contenido no sale del navegador.
 */
import { instalarHuellaArchivo, renderizarCampoHuellaArchivo } from "../portal-huella-archivo.js";
import { instalarCopiaJustificantes, renderizarJustificante } from "../portal-justificante.js";

export const RUTAS_INCORPORACIONES_CENTRO = Object.freeze({
  bandeja: "/api/vec/contratacion-temporal/peticiones-centro/incorporaciones",
  confirmaciones: "/api/vec/contratacion-temporal/peticiones-centro/incorporaciones/confirmaciones",
});
const MAXIMO_RESPUESTA = 256 * 1024;
const TIEMPO_MAXIMO_MS = 15_000;
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u;
const REF = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const HUELLA = /^[0-9a-f]{64}$/u;
const FECHA = /^\d{4}-\d{2}-\d{2}$/u;

export const MENSAJES_INCORPORACIONES_CENTRO_ES = Object.freeze({
  titulo: "Incorporaciones del centro",
  cargando: "Consultando las incorporaciones del centro…",
  sin_expedientes: "No hay expedientes de las peticiones de este centro.",
  error_lectura: "No se han podido consultar las incorporaciones. Inténtelo de nuevo.",
  reintentar: "Volver a consultar",
  expediente: "Expediente",
  periodo: "Periodo solicitado",
  situacion: "Situación",
  incorporacion: "Incorporación",
  pendiente: "Pendiente de confirmar",
  no_procede: "Aún no procede",
  confirmada: "Confirmada el {fecha} con {documento}",
  confirmar: "Confirmar la incorporación",
  fecha_incorporacion: "Fecha de incorporación",
  documento_referencia: "Referencia del documento",
  documento_archivo: "Archivo del documento",
  documento_exigido: "Documento que acredita la incorporación: {documento}",
  confirmacion_expresa: "Confirmo que la persona se ha incorporado en esa fecha y que el documento la acredita.",
  enviar: "Registrar la confirmación",
  enviando: "Registrando la confirmación; espere el recibo.",
  exito: "Incorporación confirmada el {fecha}.",
  error_datos: "Revise la fecha (no puede ser futura), la referencia, el archivo y la confirmación.",
  error_no_admitida: "Este expediente ya no admite la confirmación. Se ha actualizado la lista.",
  error_clave_reutilizada: "Esta confirmación ya se registró con otros datos.",
  error_denegado: "No tiene permiso para confirmar incorporaciones de este centro.",
  error_pendiente: "No se ha podido saber si quedó registrada. Pulse de nuevo: se usará la misma operación y no se duplicará.",
  error_general: "No se ha podido registrar la confirmación. Inténtelo de nuevo más tarde.",
  justificante_registrado: "Recibo registrado.",
  justificante_copiar: "Copiar la referencia del recibo",
  justificante_copiado: "Referencia copiada",
  documento_toma_posesion: "la toma de posesión",
  documento_contrato_firmado: "el contrato firmado",
  fase_nombramiento: "Nombramiento o contrato en curso",
  fase_otra: "En tramitación en RRHH",
  estado_completado: "Expediente cerrado",
  ayuda_titulo: "¿Cómo se confirma la incorporación?",
  ayuda: "Cuando RRHH ha nombrado o contratado a la persona, el centro confirma el día en que se incorporó con el documento que lo acredita: la toma de posesión en los nombramientos y el contrato firmado en los contratos laborales, según fija el catálogo. Indique la referencia del documento (la de su registro en Documentos o la del registro del centro) y elija el archivo: se comprueba en este equipo para calcular su huella digital, que es lo único que se registra; el documento no se envía ni se guarda en VEC. Si RRHH aún no ha nombrado, el expediente aparece como «Aún no procede».",
});

export function crearTraductorIncorporacionesCentro(mensajes = MENSAJES_INCORPORACIONES_CENTRO_ES) {
  return (clave, valores = {}) => {
    const plantilla = typeof mensajes?.[clave] === "string" ? mensajes[clave] : MENSAJES_INCORPORACIONES_CENTRO_ES[clave] ?? clave;
    return plantilla.replace(/\{([a-z_]+)\}/gu, (_, n) => (Object.hasOwn(valores, n) ? String(valores[n]) : `{${n}}`));
  };
}

const escapar = (v) => String(v ?? "").replace(/[&<>"']/gu, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);

function fechaVisible(valor) {
  if (typeof valor !== "string" || !FECHA.test(valor)) return "—";
  const f = new Date(`${valor}T00:00:00Z`);
  return Number.isFinite(f.getTime()) ? new Intl.DateTimeFormat("es-ES", { dateStyle: "long", timeZone: "UTC" }).format(f) : valor;
}

export function fechaCivilValida(valor) {
  if (typeof valor !== "string" || !FECHA.test(valor)) return false;
  const f = new Date(`${valor}T00:00:00Z`);
  return Number.isFinite(f.getTime()) && f.toISOString().slice(0, 10) === valor;
}

/** Valida la bandeja: lo desconocido se rechaza entero. */
export function validarBandejaIncorporaciones(data) {
  if (!data || typeof data !== "object" || data.esquema !== "vec.contratacion-temporal.incorporaciones-centro.v1"
    || !Array.isArray(data.expedientes) || data.expedientes.length > 50 || typeof data.puede_confirmar !== "boolean") {
    throw new TypeError("bandeja de incorporaciones no válida");
  }
  for (const e of data.expedientes) {
    if (!e || typeof e !== "object" || !REF.test(e.peticion_ref) || !REF.test(e.expediente_ref) || typeof e.numero_visible !== "string"
      || !Number.isSafeInteger(e.version) || typeof e.fase !== "string" || typeof e.estado !== "string"
      || typeof e.documento_exigido !== "string"
      || (e.confirmacion !== null && (typeof e.confirmacion !== "object" || !fechaCivilValida(e.confirmacion.fecha_incorporacion)
        || typeof e.confirmacion.documento_tipo !== "string" || !REF.test(e.confirmacion.recibo_ref)))) {
      throw new TypeError("expediente de la bandeja no válido");
    }
  }
  return data;
}

/** Solicitud exacta de la confirmación; el servidor revalida todo. */
export function validarSolicitudConfirmacionCentro(s, hoy) {
  const campos = ["clave_idempotencia", "peticion_ref", "expediente_ref", "fecha_incorporacion", "documento_ref", "documento_sha256"];
  if (!s || typeof s !== "object" || Object.keys(s).length !== campos.length || !campos.every((c) => typeof s[c] === "string")
    || !UUID.test(s.clave_idempotencia) || !REF.test(s.peticion_ref) || !REF.test(s.expediente_ref)
    || !fechaCivilValida(s.fecha_incorporacion) || (hoy && s.fecha_incorporacion > hoy)
    || !REF.test(s.documento_ref) || !HUELLA.test(s.documento_sha256) || s.documento_sha256 === "0".repeat(64)) {
    throw new TypeError("solicitud de confirmación no válida");
  }
  return Object.freeze({ ...s });
}

async function leerLimitado(respuesta) {
  const texto = await respuesta.text();
  if (texto.length > MAXIMO_RESPUESTA) throw new Error("respuesta demasiado grande");
  return texto;
}

/** Cliente de las dos rutas: mismo origen, sin caché ni redirecciones ni referente. */
export function crearClienteIncorporacionesCentro(fetchImpl = globalThis.fetch) {
  async function pedir(ruta, { method = "GET", cuerpo } = {}) {
    const control = new AbortController();
    const temporizador = setTimeout(() => control.abort(), TIEMPO_MAXIMO_MS);
    try {
      let respuesta;
      try {
        respuesta = await fetchImpl(ruta, {
          method, credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
          headers: cuerpo === undefined ? { Accept: "application/json" } : { Accept: "application/json", "Content-Type": "application/json; charset=utf-8" },
          body: cuerpo === undefined ? undefined : JSON.stringify(cuerpo), signal: control.signal,
        });
      } catch {
        throw Object.assign(new Error("indeterminado"), { indeterminado: true });
      }
      const texto = await leerLimitado(respuesta);
      let json = null;
      try { json = texto ? JSON.parse(texto) : null; } catch { json = null; }
      if (!respuesta.ok) {
        throw Object.assign(new Error("rechazo"), { estado: respuesta.status, codigo: typeof json?.error?.codigo === "string" ? json.error.codigo : "",
          indeterminado: respuesta.status >= 500 && method !== "GET" });
      }
      if (!json || typeof json !== "object") throw Object.assign(new Error("respuesta"), { indeterminado: method !== "GET" });
      return json.data;
    } finally {
      clearTimeout(temporizador);
    }
  }
  return Object.freeze({
    bandeja: async () => validarBandejaIncorporaciones(await pedir(RUTAS_INCORPORACIONES_CENTRO.bandeja)),
    confirmar: async (solicitud, hoy) => {
      const cuerpo = validarSolicitudConfirmacionCentro(solicitud, hoy);
      const recibo = await pedir(RUTAS_INCORPORACIONES_CENTRO.confirmaciones, { method: "POST", cuerpo });
      if (!recibo || recibo.expediente_ref !== cuerpo.expediente_ref || recibo.fecha_incorporacion !== cuerpo.fecha_incorporacion
        || !REF.test(recibo.recibo_ref) || !["registrado", "replay_confirmado"].includes(recibo.estado_local)) {
        throw Object.assign(new Error("recibo"), { indeterminado: true });
      }
      return recibo;
    },
  });
}

function hoyMadrid(ahora = new Date()) {
  const partes = new Intl.DateTimeFormat("en-CA", { timeZone: "Europe/Madrid", year: "numeric", month: "2-digit", day: "2-digit" }).formatToParts(ahora);
  const v = Object.fromEntries(partes.map((p) => [p.type, p.value]));
  return `${v.year}-${v.month}-${v.day}`;
}

/** Monta la sección en `contenedor`. Si el servidor no la compone, no se muestra. */
export function montarIncorporacionesCentro({ contenedor, cliente = crearClienteIncorporacionesCentro(), mensajes, generarClave = () => globalThis.crypto?.randomUUID?.(), ahora = () => new Date() } = {}) {
  if (!contenedor || typeof contenedor.addEventListener !== "function") throw new TypeError("contenedor no válido");
  const t = crearTraductorIncorporacionesCentro(mensajes);
  const documento = (tipo) => { const c = `documento_${tipo}`; const v = t(c); return v === c ? tipo : v; };
  let datos = null;
  let aviso = null;
  let abierto = null;
  let ocupado = false;
  const claves = new Map();
  const claveDe = (exp) => { if (!claves.has(exp)) claves.set(exp, generarClave()); return claves.get(exp); };

  function situacion(e) {
    if (e.estado === "completado") return t("estado_completado");
    return e.fase === "nombramiento" ? t("fase_nombramiento") : t("fase_otra");
  }

  function formulario(e) {
    const hoy = hoyMadrid(ahora());
    return `<form class="pc-detalle" data-ic-form="${escapar(e.expediente_ref)}" aria-labelledby="ic-form-${escapar(e.expediente_ref)}" novalidate>
      <h3 id="ic-form-${escapar(e.expediente_ref)}">${escapar(t("confirmar"))} · ${escapar(e.numero_visible)}</h3>
      <p>${escapar(t("documento_exigido", { documento: documento(e.documento_exigido) }))}</p>
      <label>${escapar(t("fecha_incorporacion"))}<input type="date" name="fecha_incorporacion" max="${escapar(hoy)}" required ${ocupado ? "disabled" : ""}></label>
      <label>${escapar(t("documento_referencia"))}<input type="text" name="documento_ref" maxlength="160" autocomplete="off" required ${ocupado ? "disabled" : ""}></label>
      ${renderizarCampoHuellaArchivo({ id: `ic-archivo-${e.version}-${e.numero_visible.replace(/[^A-Za-z0-9-]/gu, "-")}`, nombre: "documento_sha256", etiqueta: t("documento_archivo"), clase: "pc-campo-archivo", deshabilitado: ocupado, escapar })}
      <label class="pc-confirmacion"><input type="checkbox" name="confirmacion" required ${ocupado ? "disabled" : ""}> ${escapar(t("confirmacion_expresa"))}</label>
      <div class="pc-acciones"><button type="submit" class="boton-primario" ${ocupado ? "disabled" : ""}>${escapar(t("enviar"))}</button></div></form>`;
  }

  function fila(e) {
    const estadoIncorporacion = e.confirmacion
      ? `<span class="pc-estado pc-estado-confirmada">${escapar(t("confirmada", { fecha: fechaVisible(e.confirmacion.fecha_incorporacion), documento: documento(e.confirmacion.documento_tipo) }))}</span>`
      : e.documento_exigido && e.fase === "nombramiento" && e.estado === "en_curso"
        ? (datos.puede_confirmar
          ? `<button type="button" class="boton-secundario" data-ic-abrir="${escapar(e.expediente_ref)}" aria-expanded="${abierto === e.expediente_ref}">${escapar(t("confirmar"))}</button>`
          : `<span class="pc-estado pc-estado-pendiente">${escapar(t("pendiente"))}</span>`)
        : `<span class="pc-estado">${escapar(t("no_procede"))}</span>`;
    const periodo = e.periodo ? `${fechaVisible(e.periodo.inicio)} — ${fechaVisible(e.periodo.fin)}` : "—";
    return `<tr><td>${escapar(e.numero_visible)}</td><td>${escapar(periodo)}</td><td>${escapar(situacion(e))}</td><td>${estadoIncorporacion}</td></tr>`;
  }

  function pintar() {
    if (datos?.ausente) { contenedor.innerHTML = ""; contenedor.hidden = true; return; }
    contenedor.hidden = false;
    const cabecera = `<h2 id="ic-titulo">${escapar(t("titulo"))}</h2>`;
    if (!datos) { contenedor.innerHTML = `<section class="pc-panel" aria-labelledby="ic-titulo" aria-busy="true">${cabecera}<p class="pc-cargando" role="status">${escapar(t("cargando"))}</p></section>`; return; }
    if (datos.error) {
      contenedor.innerHTML = `<section class="pc-panel" aria-labelledby="ic-titulo">${cabecera}<p class="pc-error" role="alert">${escapar(t("error_lectura"))}</p>
        <div class="pc-acciones"><button type="button" class="boton-secundario" data-ic-recargar>${escapar(t("reintentar"))}</button></div></section>`;
      return;
    }
    const lista = datos.expedientes.length === 0 ? `<p>${escapar(t("sin_expedientes"))}</p>`
      : `<div class="pc-tabla-wrap"><table class="pc-tabla"><thead><tr><th scope="col">${escapar(t("expediente"))}</th><th scope="col">${escapar(t("periodo"))}</th>
        <th scope="col">${escapar(t("situacion"))}</th><th scope="col">${escapar(t("incorporacion"))}</th></tr></thead><tbody>${datos.expedientes.map(fila).join("")}</tbody></table></div>`;
    const seleccionado = datos.expedientes.find((e) => e.expediente_ref === abierto && !e.confirmacion);
    const avisoHTML = aviso ? `<div class="${aviso.tono === "exito" ? "pc-recibo" : aviso.tono === "aviso" ? "pc-pendiente" : "pc-error"}" role="${aviso.tono === "peligro" ? "alert" : "status"}" tabindex="-1" data-ic-aviso>
      <p>${escapar(aviso.texto)}</p>${aviso.recibo ? renderizarJustificante(aviso.recibo, { escapar, etiqueta: t("justificante_registrado"), copiar: t("justificante_copiar"), copiado: t("justificante_copiado") }) : ""}</div>` : "";
    contenedor.innerHTML = `<section class="pc-panel" aria-labelledby="ic-titulo" ${ocupado ? 'aria-busy="true"' : ""}>${cabecera}${avisoHTML}${lista}${seleccionado ? formulario(seleccionado) : ""}</section>`;
  }

  async function cargar() {
    datos = null; pintar();
    try { datos = await cliente.bandeja(); } catch (error) { datos = error?.estado === 404 ? { ausente: true } : { error: true }; }
    pintar();
  }

  async function enviar(form) {
    if (ocupado) return;
    const e = datos?.expedientes?.find((x) => x.expediente_ref === form.dataset.icForm);
    if (!e) return;
    const campos = Object.fromEntries(new FormData(form).entries());
    let solicitud;
    try {
      if (campos.confirmacion !== "on") throw new TypeError("sin confirmación");
      solicitud = validarSolicitudConfirmacionCentro({ clave_idempotencia: claveDe(e.expediente_ref), peticion_ref: e.peticion_ref, expediente_ref: e.expediente_ref,
        fecha_incorporacion: String(campos.fecha_incorporacion ?? "").trim(), documento_ref: String(campos.documento_ref ?? "").trim(),
        documento_sha256: String(campos.documento_sha256 ?? "").toLowerCase() }, hoyMadrid(ahora()));
    } catch {
      aviso = { tono: "peligro", texto: t("error_datos") }; pintar(); contenedor.querySelector("[data-ic-aviso]")?.focus?.(); return;
    }
    ocupado = true; aviso = { tono: "aviso", texto: t("enviando") }; pintar();
    try {
      const recibo = await cliente.confirmar(solicitud, hoyMadrid(ahora()));
      claves.delete(e.expediente_ref);
      ocupado = false; abierto = null;
      aviso = { tono: "exito", texto: t("exito", { fecha: fechaVisible(recibo.fecha_incorporacion) }), recibo: recibo.recibo_ref };
      await cargar();
    } catch (error) {
      ocupado = false;
      if (error?.indeterminado) aviso = { tono: "aviso", texto: t("error_pendiente") };
      else {
        claves.delete(e.expediente_ref);
        const codigo = { no_admitida: "error_no_admitida", clave_reutilizada: "error_clave_reutilizada", operacion_denegada: "error_denegado", solicitud_invalida: "error_datos" }[error?.codigo];
        aviso = { tono: "peligro", texto: t(codigo ?? "error_general") };
        if (error?.codigo === "no_admitida") { abierto = null; await cargar(); }
      }
      pintar();
    }
    contenedor.querySelector("[data-ic-aviso]")?.focus?.();
  }

  const alPulsar = (evento) => {
    const boton = evento.target?.closest?.("button");
    if (!boton || ocupado) return;
    if (boton.matches("[data-ic-recargar]")) { cargar(); return; }
    if (boton.dataset.icAbrir) { abierto = abierto === boton.dataset.icAbrir ? null : boton.dataset.icAbrir; aviso = null; pintar(); }
  };
  const alEnviar = (evento) => {
    const form = evento.target?.closest?.("[data-ic-form]");
    if (!form) return;
    evento.preventDefault();
    enviar(form);
  };
  const retirarHuella = instalarHuellaArchivo(contenedor);
  instalarCopiaJustificantes(contenedor.ownerDocument ?? globalThis.document);
  contenedor.addEventListener("click", alPulsar);
  contenedor.addEventListener("submit", alEnviar);
  cargar();
  return () => {
    retirarHuella();
    contenedor.removeEventListener("click", alPulsar);
    contenedor.removeEventListener("submit", alEnviar);
  };
}

/** Rellena la ayuda «?» de la página con los textos de esta sección. */
export function instalarAyudaIncorporacionesCentro(doc, t = crearTraductorIncorporacionesCentro()) {
  for (const elemento of doc?.querySelectorAll?.("[data-i18n-ayuda-incorporacion]") ?? []) {
    elemento.textContent = t(elemento.dataset.i18nAyudaIncorporacion);
  }
}

if (typeof document !== "undefined" && document.querySelector("#incorporaciones-centro")) {
  instalarAyudaIncorporacionesCentro(document);
  montarIncorporacionesCentro({ contenedor: document.querySelector("#incorporaciones-centro") });
}
