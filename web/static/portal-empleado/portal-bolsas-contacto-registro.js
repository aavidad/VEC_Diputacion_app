/**
 * Registro o corrección del contacto por RRHH desde la ficha de participación
 * (duda 45). Envía una versión nueva del contacto (correo y hasta dos
 * teléfonos) con su origen: propio o de CONVOCA. La vigencia del contacto de
 * CONVOCA la fija el servidor con la regla del catálogo; aquí no se calcula.
 * El formulario nunca se rellena con el contacto vigente: el claro no se pide.
 */
import { traducirPortal } from "./portal-i18n.js?v=20260926-huecos-rrhh-v2";
import { justificanteTraducido } from "./portal-justificante.js";

const BASE = "/api/vec/bolsa/bolsas";
const ORIGENES = Object.freeze(["", "convoca"]);
const CORREO = /^[^\s@]{1,64}@[^\s@]{1,190}\.[^\s@]{2,}$/u;
const TELEFONO = /^(?:\+[0-9]{9,15}|[6789][0-9]{8})$/u;

export const MENSAJES_REGISTRO_CONTACTO_ES = Object.freeze({
  abrir: "Registrar o corregir contacto",
  titulo: "Contacto de la persona",
  correo: "Correo electrónico",
  telefono_1: "Teléfono",
  telefono_2: "Segundo teléfono",
  origen: "Origen",
  origen_propio: "Propio (comprobado por RRHH)",
  origen_convoca: "CONVOCA (pendiente de confirmar por la persona)",
  motivo: "Motivo del registro",
  cancelar: "Cancelar",
  registrar: "Registrar contacto",
  registrando: "Registrando…",
  exito: "Contacto registrado (versión {version}).",
  recuperada: " (respuesta recuperada)",
  error_datos: "Indique un correo o un teléfono válidos y el motivo.",
  error_400: "La petición no es válida.",
  error_403: "Sin permiso para registrar el contacto.",
  error_409: "Los datos de contacto no son válidos o chocan con una versión anterior.",
  error_422: "El origen CONVOCA no está disponible: falta su regla en el catálogo.",
  error_servidor: "No se pudo registrar el contacto. No se ha guardado nada.",
});

export function traducirRegistroContacto(clave, variables = {}) {
  const plantilla = MENSAJES_REGISTRO_CONTACTO_ES[clave] ?? clave;
  return plantilla.replace(/\{(\w+)\}/g, (_, nombre) => String(variables[nombre] ?? ""));
}
const t = traducirRegistroContacto;

function html(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function segmento(valor) {
  return encodeURIComponent(String(valor ?? "").trim()).replace(/%3A/gi, ":");
}

export function rutaRegistroContacto(bolsa, participacion) {
  return `${BASE}/${segmento(bolsa)}/candidatos/${segmento(participacion)}/datos-contacto`;
}

/** Devuelve el mensaje de error o "" si el comando es enviable. */
export function validarComandoContacto(c) {
  if (!c || !ORIGENES.includes(c.origen)) return t("error_datos");
  if (!c.motivo || c.motivo.length > 1000) return t("error_datos");
  if (!c.correo && !c.telefono_1) return t("error_datos");
  if (c.correo && !CORREO.test(c.correo)) return t("error_datos");
  if (c.telefono_1 && !TELEFONO.test(c.telefono_1)) return t("error_datos");
  if (c.telefono_2 && (!c.telefono_1 || !TELEFONO.test(c.telefono_2) || c.telefono_2 === c.telefono_1)) return t("error_datos");
  return "";
}

export function leerComandoContacto(datos) {
  const campo = (nombre) => String(datos.get(nombre) ?? "").trim();
  return { correo: campo("correo"), telefono_1: campo("telefono_1"), telefono_2: campo("telefono_2"), motivo: campo("motivo"), origen: campo("origen") };
}

export async function registrarContacto(bolsa, participacion, comando, clave, { fetchImpl = fetch } = {}) {
  if (!bolsa || !participacion || !clave || validarComandoContacto(comando)) return { ok: false, status: 400, mensaje: t("error_datos") };
  try {
    const respuesta = await fetchImpl(rutaRegistroContacto(bolsa, participacion), {
      method: "POST", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
      headers: { Accept: "application/json", "Content-Type": "application/json", "Idempotency-Key": clave },
      body: JSON.stringify(comando),
    });
    const cuerpo = await respuesta.json().catch(() => null);
    const d = cuerpo?.data;
    if ((respuesta.status === 200 || respuesta.status === 201) && d && typeof d.recibo_ref === "string" && Number.isSafeInteger(d.version)) {
      return { ok: true, datos: d };
    }
    const mensaje = { 400: t("error_400"), 403: t("error_403"), 409: t("error_409"), 422: t("error_422") }[respuesta.status] || t("error_servidor");
    return { ok: false, status: respuesta.status, mensaje };
  } catch {
    return { ok: false, status: 0, mensaje: t("error_servidor") };
  }
}

function campo(etiqueta, control, e) {
  return `<label class="campo"><span>${e(etiqueta)}</span>${control}</label>`;
}

/** Botón y formulario del bloque, bajo los datos de la ficha. */
export function renderizarRegistroContacto({ estado = {}, escaparHTML = html } = {}) {
  const e = escaparHTML;
  const recibo = estado.recibo ? ` ${justificanteTraducido(estado.recibo, e, (clave) => traducirPortal(`panel_${clave}`))}` : "";
  const exito = estado.exito ? `<p class="mensaje-exito" role="status">${e(estado.exito)}${recibo}</p>` : "";
  if (!estado.abierto) {
    return `<div class="acciones-vista" data-contacto-rrhh="true">${exito}<button type="button" class="boton-secundario" data-contacto-rrhh-accion="abrir">${e(t("abrir"))}</button></div>`;
  }
  const f = estado.formulario || {};
  const origen = (valor, etiqueta) => `<option value="${e(valor)}" ${(f.origen ?? "") === valor ? "selected" : ""}>${e(t(etiqueta))}</option>`;
  const error = estado.error ? `<p class="mensaje-error" role="alert">${e(estado.error)}</p>` : "";
  return `<form data-contacto-rrhh-form="registro" class="formulario-gobernado" data-contacto-rrhh="true"><fieldset><legend>${e(t("titulo"))}</legend><div class="rejilla-formulario">
    ${campo(t("correo"), `<input type="email" name="correo" maxlength="254" autocomplete="off" value="${e(f.correo || "")}">`, e)}
    ${campo(t("telefono_1"), `<input type="tel" name="telefono_1" maxlength="20" autocomplete="off" value="${e(f.telefono_1 || "")}">`, e)}
    ${campo(t("telefono_2"), `<input type="tel" name="telefono_2" maxlength="20" autocomplete="off" value="${e(f.telefono_2 || "")}">`, e)}
    ${campo(t("origen"), `<select name="origen" required>${origen("", "origen_propio")}${origen("convoca", "origen_convoca")}</select>`, e)}
    </div>${campo(t("motivo"), `<textarea name="motivo" required maxlength="1000">${e(f.motivo || "")}</textarea>`, e)}</fieldset>${error}
    <div class="acciones-formulario"><button type="button" class="boton-secundario" data-contacto-rrhh-accion="cancelar" ${estado.enviando ? "disabled" : ""}>${e(t("cancelar"))}</button><button type="submit" class="boton-primario" ${estado.enviando ? "disabled" : ""}>${e(estado.enviando ? t("registrando") : t("registrar"))}</button></div></form>`;
}

/** Abre, envía y cierra el formulario; tras registrar recarga el origen. */
export function crearControladorRegistroContacto({ estado, renderizar, alRegistrar = async () => {}, fetchImpl }) {
  const opciones = fetchImpl ? { fetchImpl } : {};

  function manejarClick(evento) {
    const control = evento.target?.closest?.("[data-contacto-rrhh-accion]");
    if (!control || !estado.modalFicha) return false;
    evento.preventDefault?.();
    const modal = estado.modalFicha;
    const flujo = modal.registroContacto || (modal.registroContacto = {});
    if (flujo.enviando) return true;
    if (control.dataset.contactoRrhhAccion === "abrir") Object.assign(flujo, { abierto: true, formulario: {}, error: "", exito: "", recibo: "", clave: "" });
    else flujo.abierto = false;
    renderizar();
    return true;
  }

  function manejarSubmit(evento) {
    const formulario = evento.target?.closest?.('[data-contacto-rrhh-form="registro"]');
    if (!formulario || !estado.modalFicha) return false;
    evento.preventDefault?.();
    const modal = estado.modalFicha;
    const flujo = modal.registroContacto || (modal.registroContacto = { abierto: true });
    if (flujo.enviando) return true;
    const comando = leerComandoContacto(new FormData(formulario));
    flujo.formulario = comando;
    flujo.error = validarComandoContacto(comando);
    if (flujo.error) { renderizar(); return true; }
    // La misma clave se conserva hasta registrar: un reintento tras un corte
    // devuelve el recibo original; cambiar los datos exige otra clave.
    const huella = JSON.stringify(comando);
    if (!flujo.clave || flujo.huella !== huella) { flujo.clave = `contacto-${globalThis.crypto.randomUUID()}`; flujo.huella = huella; }
    flujo.enviando = true;
    renderizar();
    void registrarContacto(estado.bolsaSeleccionada, modal.candidato.participacion_ref, comando, flujo.clave, opciones).then(async (r) => {
      if (estado.modalFicha !== modal) return;
      flujo.enviando = false;
      if (!r.ok) { flujo.error = r.mensaje; renderizar(); return; }
      Object.assign(flujo, { abierto: false, formulario: {}, error: "", clave: "", huella: "",
        exito: t("exito", { version: r.datos.version }) + (r.datos.reutilizada ? t("recuperada") : ""), recibo: r.datos.recibo_ref });
      renderizar();
      await alRegistrar(modal);
    });
    return true;
  }

  function instalar(documento = globalThis.document) {
    documento.addEventListener("click", (evento) => { manejarClick(evento); });
    documento.addEventListener("submit", (evento) => { manejarSubmit(evento); });
  }

  return Object.freeze({ instalar, manejarClick, manejarSubmit });
}
