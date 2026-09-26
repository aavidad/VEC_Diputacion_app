// Control de intentos de contacto del llamamiento en la ficha de Bolsa
// (Reglamento de bolsas, art. 8.2.a). Las reglas y su estado los calcula el
// servidor con el catálogo; aquí solo se muestran y se registran intentos.
// La baja se propone con la operación de exclusión existente (B8).
import { traducirIntentos as t } from "./portal-i18n-intentos.js?v=20260926-integracion-bolsa-ct-v1";
import { LOCALIZACION_PORTAL, ZONA_HORARIA_PORTAL, traducirPortal } from "./portal-i18n.js?v=20260926-reparos-firma-v1";
import { justificanteTraducido } from "./portal-justificante.js";

const BASE = "/api/vec/bolsa/bolsas";
const RESULTADOS_INTENTO = Object.freeze(["contactado", "no_contesta", "numero_erroneo"]);
const AVISOS = Object.freeze(["antes_de_separacion", "fuera_de_franja", "dia_no_habil"]);
const ERRORES = Object.freeze(["intento_antes_de_separacion", "intento_fuera_de_franja", "intentos_agotados", "acceso_denegado", "contacto_en_conflicto", "solicitud_invalida"]);

function segmento(valor) {
  return encodeURIComponent(String(valor ?? "").trim()).replace(/%3A/gi, ":");
}

export function rutaContactosCandidato(bolsa, participacion) {
  return `${BASE}/${segmento(bolsa)}/candidatos/${segmento(participacion)}/contactos`;
}

function html(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function instante(valor) {
  if (!valor) return t("sin_valor");
  const fecha = new Date(valor);
  if (!Number.isFinite(fecha.getTime())) return String(valor);
  return new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { dateStyle: "short", timeStyle: "short", timeZone: ZONA_HORARIA_PORTAL }).format(fecha);
}

function intentosValidos(i) {
  if (!i || typeof i !== "object" || typeof i.configurado !== "boolean") return false;
  if (!i.configurado) return true;
  return ["sin_contacto", "maximo", "proceso", "intento", "intentos_por_proceso", "procesos"].every((c) => Number.isInteger(i[c]))
    && typeof i.baja_propuesta === "boolean" && typeof i.contactado === "boolean" && typeof i.completo === "boolean"
    && Array.isArray(i.avisos) && Array.isArray(i.reglas)
    && i.reglas.every((r) => r && typeof r.clave === "string" && typeof r.etiqueta === "string" && typeof r.referencia === "string" && typeof r.ejemplo === "boolean");
}

export async function consultarIntentosContacto(bolsa, participacion, llamamiento, { fetchImpl = fetch, signal } = {}) {
  try {
    const respuesta = await fetchImpl(`${rutaContactosCandidato(bolsa, participacion)}?llamamiento_ref=${encodeURIComponent(llamamiento)}`, {
      method: "GET", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal, headers: { Accept: "application/json" },
    });
    if (!respuesta.ok) return { ok: false, status: respuesta.status, mensaje: t("error_carga") };
    const cuerpo = await respuesta.json();
    const intentos = cuerpo?.data?.intentos;
    if (cuerpo?.data?.esquema !== "vec.bolsa.rrhh.contactos.v1" || !intentosValidos(intentos)) {
      return { ok: false, status: 0, mensaje: t("error_carga") };
    }
    return { ok: true, datos: intentos };
  } catch {
    return { ok: false, status: 0, mensaje: t("error_carga") };
  }
}

export async function registrarContactoIntento(bolsa, participacion, comando, clave, { fetchImpl = fetch } = {}) {
  try {
    const respuesta = await fetchImpl(rutaContactosCandidato(bolsa, participacion), {
      method: "POST", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
      headers: { Accept: "application/json", "Content-Type": "application/json", "Idempotency-Key": clave },
      body: JSON.stringify(comando),
    });
    const cuerpo = await respuesta.json().catch(() => ({}));
    if (respuesta.ok && typeof cuerpo?.data?.recibo_ref === "string") return { ok: true, datos: cuerpo.data };
    const codigo = cuerpo?.error?.codigo;
    return { ok: false, status: respuesta.status, mensaje: ERRORES.includes(codigo) ? t(`error_${codigo}`) : t("error_servicio") };
  } catch {
    return { ok: false, status: 0, mensaje: t("error_servicio") };
  }
}

function pastillaEstado(i) {
  if (i.contactado) return `<span class="estado-chip exito">${t("estado_contactado")}</span>`;
  if (i.baja_propuesta) return `<span class="estado-chip peligro">${t("estado_baja_propuesta")}</span>`;
  return `<span class="estado-chip info">${t("estado_proceso", { proceso: i.proceso, procesos: i.procesos, intento: i.intento, intentos: i.intentos_por_proceso })}</span>`;
}

function localAhora() {
  const d = new Date();
  const p = (n) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
}

function formularios(e, i, candidato, flujo) {
  const deshabilitado = flujo.enviando ? " disabled" : "";
  const etiquetaBoton = (clave) => (flujo.enviando ? t("boton_enviando") : t(clave));
  const campos = (tipo) => `<label>${t("campo_instante")} <input type="datetime-local" name="instante" required value="${localAhora()}"></label><label>${t("campo_anotacion")} <textarea name="anotacion" required maxlength="1000" data-intentos-campo="${tipo}"></textarea></label>`;
  let intento = "";
  if (i.baja_propuesta && !i.contactado) {
    const excluible = candidato.estado_clave !== "excluido";
    intento = `<p role="status">${e(t("baja_propuesta_texto"))}</p>${excluible ? `<button type="button" class="boton-primario" data-b8-accion="seleccionar" data-operacion="excluir">${t("boton_proponer_baja")}</button>` : ""}`;
  } else {
    intento = `<form data-intentos-form="intento"><h5>${t("formulario_intento")}</h5><label>${t("campo_resultado")} <select name="resultado" required>${RESULTADOS_INTENTO.map((r) => `<option value="${r}">${t(`resultado_${r}`)}</option>`).join("")}</select></label>${campos("intento")}<button type="submit" class="boton-primario"${deshabilitado}>${etiquetaBoton("boton_intento")}</button></form>`;
  }
  const rebote = `<form data-intentos-form="rebote"><h5>${t("formulario_rebote")}</h5>${campos("rebote")}<button type="submit" class="boton-secundario"${deshabilitado}>${etiquetaBoton("boton_rebote")}</button></form>`;
  return `${intento}${rebote}`;
}

export function renderizarIntentosContacto({ candidato, estado = {}, escaparHTML = html }) {
  const e = escaparHTML;
  const cabecera = (extra = "") => `<div class="cabecera-panel"><h4 id="titulo-intentos-contacto">${t("titulo")}</h4>${extra}</div>`;
  const envolver = (extra, cuerpo) => `<section class="panel panel-separado" data-intentos-raiz="true" aria-labelledby="titulo-intentos-contacto">${cabecera(extra)}<div class="cuerpo-panel">${cuerpo}</div></section>`;
  if (!candidato?.ultimo_llamamiento?.llamamiento_ref) return envolver("", `<p class="vacio-controlado" role="status">${t("sin_llamamiento")}</p>`);
  const actual = estado.carga || "cargando";
  if (actual === "cargando") return envolver("", `<p class="vacio-controlado" role="status" aria-busy="true">${t("cargando")}</p>`);
  if (actual === "error") return envolver("", `<p class="mensaje-error" role="alert">${e(estado.error || t("error_carga"))}</p><button type="button" class="boton-secundario" data-intentos-accion="reintentar">${t("reintentar")}</button>`);
  const i = estado.datos;
  const mensajes = `${estado.recibo ? `<p class="mensaje-exito" role="status">${e(t("registrado"))} ${justificanteTraducido(estado.recibo, e, (clave) => traducirPortal(`panel_${clave}`))}</p>` : ""}<p class="mensaje-error" role="alert">${e(estado.errorOperacion || "")}</p>`;
  if (!i.configurado) return envolver("", `<p class="vacio-controlado" role="status">${t("sin_catalogo")}</p>${mensajes}${formularios(e, { baja_propuesta: false }, candidato, estado)}`);
  const avisos = i.avisos.filter((a) => AVISOS.includes(a)).map((a) => `<span class="estado-chip aviso">${t(`aviso_${a}`)}</span>`).join(" ");
  const franja = i.franja ? `<div class="fila-resumen"><dt>${t("dato_franja")}</dt><dd>${e(i.franja.solo_dias_habiles ? t("franja_habiles", { valor: i.franja.valor }) : i.franja.valor)}</dd></div>` : "";
  const siguiente = !i.contactado && !i.baja_propuesta ? `<div class="fila-resumen"><dt>${t("dato_siguiente")}</dt><dd>${e(i.siguiente_permitido_desde ? instante(i.siguiente_permitido_desde) : t("sin_valor"))}</dd></div>` : "";
  // El origen de cada regla (Reglamento o ejemplo) se consulta en «Reglas vigentes».
  const reglas = i.reglas.map((r) => `<li>${e(r.etiqueta)}</li>`).join("");
  const cuerpo = `${avisos ? `<p>${avisos}</p>` : ""}${i.completo ? "" : `<p class="mensaje-error" role="alert">${t("historico_incompleto")}</p>`}
    <dl class="resumen-expediente">
      <div class="fila-resumen"><dt>${t("dato_sin_contacto")}</dt><dd>${e(t("dato_valor_sin_contacto", { sin: i.sin_contacto, maximo: i.maximo }))}</dd></div>
      <div class="fila-resumen"><dt>${t("dato_ultimo")}</dt><dd>${e(instante(i.ultimo_intento))}</dd></div>
      ${siguiente}${franja}
    </dl>${mensajes}${formularios(e, i, candidato, estado)}
    <h5>${t("reglas")}</h5><ul class="lista-reglas-intentos">${reglas}</ul>`;
  return envolver(pastillaEstado(i), cuerpo);
}

export function crearControladorIntentosContacto({ estado, renderizar, fetchImpl }) {
  const opciones = fetchImpl ? { fetchImpl } : {};
  async function cargar(modal) {
    const llamamiento = modal?.candidato?.ultimo_llamamiento?.llamamiento_ref;
    if (!modal || !llamamiento) return;
    modal.controladorIntentos?.abort();
    const controlador = new AbortController();
    modal.controladorIntentos = controlador;
    modal.intentosContacto = { ...modal.intentosContacto, carga: "cargando" };
    renderizar();
    const res = await consultarIntentosContacto(estado.bolsaSeleccionada, modal.candidato.participacion_ref, llamamiento, { ...opciones, signal: controlador.signal });
    if (controlador.signal.aborted || estado.modalFicha !== modal) return;
    modal.intentosContacto = res.ok
      ? { ...modal.intentosContacto, carga: "listo", datos: res.datos }
      : { ...modal.intentosContacto, carga: "error", error: res.mensaje };
    renderizar();
  }

  function manejarClick(evento) {
    const control = evento.target?.closest?.("[data-intentos-accion]");
    if (!control || !estado.modalFicha) return false;
    evento.preventDefault();
    if (control.dataset.intentosAccion === "reintentar") void cargar(estado.modalFicha);
    return true;
  }

  async function enviar(modal, flujo, comando) {
    const huella = JSON.stringify(comando);
    if (flujo.huella !== huella) {
      flujo.huella = huella;
      flujo.clave = globalThis.crypto?.randomUUID?.() || `intento-${Date.now()}-${Math.random().toString(16).slice(2)}`;
    }
    flujo.enviando = true; flujo.errorOperacion = ""; flujo.recibo = "";
    renderizar();
    const res = await registrarContactoIntento(estado.bolsaSeleccionada, modal.candidato.participacion_ref, comando, flujo.clave, opciones);
    if (estado.modalFicha !== modal) return;
    flujo.enviando = false;
    if (!res.ok) {
      flujo.errorOperacion = res.mensaje;
      renderizar();
      return;
    }
    flujo.recibo = res.datos.recibo_ref;
    delete flujo.huella; delete flujo.clave;
    await cargar(modal);
  }

  function manejarSubmit(evento) {
    const formulario = evento.target?.closest?.("[data-intentos-form]");
    if (!formulario || !estado.modalFicha) return false;
    evento.preventDefault();
    const modal = estado.modalFicha;
    const flujo = modal.intentosContacto || (modal.intentosContacto = {});
    if (flujo.enviando) return true;
    const datos = new FormData(formulario);
    const fecha = new Date(String(datos.get("instante") || ""));
    const anotacion = String(datos.get("anotacion") || "").trim();
    if (!Number.isFinite(fecha.getTime()) || !anotacion) {
      flujo.errorOperacion = t("error_solicitud_invalida");
      renderizar();
      return true;
    }
    const rebote = formulario.dataset.intentosForm === "rebote";
    const resultado = rebote ? "no_entregado" : String(datos.get("resultado") || "");
    if (!rebote && !RESULTADOS_INTENTO.includes(resultado)) {
      flujo.errorOperacion = t("error_solicitud_invalida");
      renderizar();
      return true;
    }
    void enviar(modal, flujo, {
      canal: rebote ? "correo" : "telefono", resultado, anotacion, instante: fecha.toISOString(),
      llamamiento_ref: modal.candidato.ultimo_llamamiento.llamamiento_ref,
    });
    return true;
  }

  function instalar(documento = globalThis.document) {
    documento.addEventListener("click", (evento) => { manejarClick(evento); });
    documento.addEventListener("submit", (evento) => { manejarSubmit(evento); });
  }

  return Object.freeze({ cargar, instalar, manejarClick, manejarSubmit });
}
