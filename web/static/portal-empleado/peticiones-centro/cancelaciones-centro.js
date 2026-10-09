/**
 * Cancelación del expediente por el centro solicitante (duda 12 de RRHH).
 *
 * Reutiliza la lectura de los expedientes de las peticiones del centro (la
 * bandeja de incorporaciones) para ligar cada petición con su expediente y,
 * en los que siguen en una fase que admite el catálogo, ofrece «Cancelar» con
 * un motivo del catálogo. Fases y motivos los decide el servidor; la vista
 * solo evita ofrecer la cancelación cuando no procede.
 */
import { bandejaCompartidaPagina, periodoVisible, prepararCausasFin } from "./incorporaciones-centro.js?v=20261009-retoques-textos-v1";
import { validarConsultaCancelacion, validarReciboCancelacion, validarSolicitudCancelacion } from "../modulos/contratacion-temporal/cliente-http-cancelacion.js?v=20260926-huecos-rrhh-v1";
import { instalarCopiaJustificantes, renderizarJustificante } from "../portal-justificante.js";

import { IDIOMA_POR_DEFECTO } from "../../comun/idioma.js";
import { IDIOMA_EFECTIVO_PETICIONES_CENTRO,
  MENSAJES_CANCELACIONES_CENTRO } from "./i18n-peticiones-centro.js?v=20261007-pc-recuperacion-v1";

export const RUTAS_CANCELACIONES_CENTRO = Object.freeze({
  consulta: "/api/vec/contratacion-temporal/peticiones-centro/cancelacion",
  cancelaciones: "/api/vec/contratacion-temporal/peticiones-centro/cancelaciones",
});
const MAXIMO_RESPUESTA = 32 * 1024;
const TIEMPO_MAXIMO_MS = 15_000;

export const MENSAJES_CANCELACIONES_CENTRO_ES = IDIOMA_EFECTIVO_PETICIONES_CENTRO === IDIOMA_POR_DEFECTO
  ? MENSAJES_CANCELACIONES_CENTRO : undefined;

export function crearTraductorCancelacionesCentro(mensajes = MENSAJES_CANCELACIONES_CENTRO) {
  return (clave, valores = {}) => {
    const plantilla = typeof mensajes?.[clave] === "string" ? mensajes[clave] : MENSAJES_CANCELACIONES_CENTRO[clave] ?? clave;
    return plantilla.replace(/\{([a-z_]+)\}/gu, (_, n) => (Object.hasOwn(valores, n) ? String(valores[n]) : `{${n}}`));
  };
}

const escapar = (v) => String(v ?? "").replace(/[&<>"']/gu, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);

/** Cliente de las dos rutas del centro: mismo origen, sin caché, redirecciones ni referente. */
export function crearClienteCancelacionesCentro(fetchImpl = globalThis.fetch) {
  async function pedir(ruta, cuerpo, efecto) {
    const control = new AbortController();
    const temporizador = setTimeout(() => control.abort(), TIEMPO_MAXIMO_MS);
    try {
      let respuesta;
      try {
        respuesta = await fetchImpl(ruta, {
          method: "POST", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
          headers: { Accept: "application/json", "Content-Type": "application/json; charset=utf-8" }, body: JSON.stringify(cuerpo), signal: control.signal,
        });
      } catch {
        throw Object.assign(new Error("indeterminado"), { indeterminado: efecto });
      }
      const texto = await respuesta.text();
      if (texto.length > MAXIMO_RESPUESTA) throw Object.assign(new Error("respuesta"), { indeterminado: efecto });
      let json = null;
      try { json = texto ? JSON.parse(texto) : null; } catch { json = null; }
      if (!respuesta.ok) {
        throw Object.assign(new Error("rechazo"), { estado: respuesta.status, codigo: typeof json?.error?.codigo === "string" ? json.error.codigo : "",
          indeterminado: efecto && respuesta.status >= 500 });
      }
      if (!json || typeof json !== "object") throw Object.assign(new Error("respuesta"), { indeterminado: efecto });
      return json.data;
    } finally {
      clearTimeout(temporizador);
    }
  }
  return Object.freeze({
    consultar: async (expedienteRef) => validarConsultaCancelacion(await pedir(RUTAS_CANCELACIONES_CENTRO.consulta, { expediente_ref: expedienteRef }, false), expedienteRef),
    cancelar: async (solicitud) => {
      const cuerpo = validarSolicitudCancelacion(solicitud);
      try {
        return validarReciboCancelacion(await pedir(RUTAS_CANCELACIONES_CENTRO.cancelaciones, cuerpo, true), cuerpo);
      } catch (error) {
        if (error instanceof TypeError) throw Object.assign(new Error("recibo"), { indeterminado: true });
        throw error;
      }
    },
  });
}

/** Monta la sección en `contenedor`. Si el servidor no la compone o el perfil no cancela, no se muestra. */
export function montarCancelacionesCentro({ contenedor, bandeja = bandejaCompartidaPagina(), cliente = crearClienteCancelacionesCentro(), mensajes,
  generarClave = () => globalThis.crypto?.randomUUID?.() } = {}) {
  if (!contenedor || typeof contenedor.addEventListener !== "function") throw new TypeError("contenedor no válido");
  contenedor.setAttribute?.("lang", IDIOMA_EFECTIVO_PETICIONES_CENTRO);
  const t = crearTraductorCancelacionesCentro(mensajes);
  let datos = null;
  let aviso = null;
  let abierto = null;
  let ocupado = false;
  const claves = new Map();
  const claveDe = (exp) => { if (!claves.has(exp)) claves.set(exp, generarClave()); return claves.get(exp); };
  const fase = (f) => { const c = `fase_${f}`; const v = t(c); return v === c ? t("fase_otra") : v; };

  function formulario(e) {
    const id = `cc-form-${escapar(e.expediente_ref)}`;
    const motivos = datos.opciones.motivos.map((m) => `<option value="${escapar(m.clave)}">${escapar(m.etiqueta)}</option>`).join("");
    return `<form class="pc-detalle" data-cc-form="${escapar(e.expediente_ref)}" aria-labelledby="${id}" novalidate>
      <h3 id="${id}">${escapar(t("cancelar"))} · ${escapar(e.numero_visible)}</h3>
      <label>${escapar(t("motivo"))}<select name="motivo_clave" required ${ocupado ? "disabled" : ""}><option value=""></option>${motivos}</select></label>
      <label>${escapar(t("observaciones"))}<textarea name="observaciones" maxlength="2000" rows="3" ${ocupado ? "disabled" : ""}></textarea></label>
      <label class="pc-confirmacion"><input type="checkbox" name="confirmacion" required ${ocupado ? "disabled" : ""}> ${escapar(t("confirmacion_expresa"))}</label>
      <div class="pc-acciones"><button type="submit" class="boton-primario" ${ocupado ? "disabled" : ""}>${escapar(t("enviar"))}</button>
      <button type="button" class="boton-secundario" data-cc-volver ${ocupado ? "disabled" : ""}>${escapar(t("volver"))}</button></div></form>`;
  }

  function fila(e) {
    const accion = e.estado === "cancelado"
      ? `<span class="pc-estado">${escapar(t("cancelado"))}</span>`
      : `<button type="button" class="boton-secundario" data-cc-abrir="${escapar(e.expediente_ref)}" aria-expanded="${abierto === e.expediente_ref}">${escapar(t("cancelar"))}</button>`;
    const periodo = periodoVisible(e.periodo, t);
    return `<tr><td>${escapar(e.numero_visible)}</td><td>${escapar(periodo)}</td><td>${escapar(e.estado === "cancelado" ? t("cancelado") : fase(e.fase))}</td><td>${accion}</td></tr>`;
  }

  function pintar() {
    if (datos?.ausente) { contenedor.innerHTML = ""; contenedor.hidden = true; return; }
    contenedor.hidden = false;
    const cabecera = `<h2 id="cc-titulo">${escapar(t("titulo"))}</h2>`;
    if (!datos) { contenedor.innerHTML = `<section class="pc-panel" aria-labelledby="cc-titulo" aria-busy="true">${cabecera}<p class="pc-cargando" role="status">${escapar(t("cargando"))}</p></section>`; return; }
    if (datos.error) {
      contenedor.innerHTML = `<section class="pc-panel" aria-labelledby="cc-titulo">${cabecera}<p class="pc-error" role="alert">${escapar(t("error_lectura"))}</p>
        <div class="pc-acciones"><button type="button" class="boton-secundario" data-cc-recargar>${escapar(t("reintentar"))}</button></div></section>`;
      return;
    }
    const lista = datos.expedientes.length === 0 ? `<p>${escapar(t("sin_expedientes"))}</p>`
      : `<div class="pc-tabla-wrap"><table class="pc-tabla"><thead><tr><th scope="col">${escapar(t("expediente"))}</th><th scope="col">${escapar(t("periodo"))}</th>
        <th scope="col">${escapar(t("situacion"))}</th><th scope="col">${escapar(t("cancelacion"))}</th></tr></thead><tbody>${datos.expedientes.map(fila).join("")}</tbody></table></div>`;
    const seleccionado = datos.expedientes.find((e) => e.expediente_ref === abierto && e.estado === "en_curso");
    const avisoHTML = aviso ? `<div class="${aviso.tono === "exito" ? "pc-recibo" : aviso.tono === "aviso" ? "pc-pendiente" : "pc-error"}" role="${aviso.tono === "peligro" ? "alert" : "status"}" tabindex="-1" data-cc-aviso>
      <p>${escapar(aviso.texto)}</p>${aviso.recibo ? renderizarJustificante(aviso.recibo, { escapar, etiqueta: t("justificante_registrado"), copiar: t("justificante_copiar"), copiado: t("justificante_copiado") }) : ""}</div>` : "";
    contenedor.innerHTML = `<section class="pc-panel" aria-labelledby="cc-titulo" ${ocupado ? 'aria-busy="true"' : ""}>${cabecera}${avisoHTML}${lista}${seleccionado ? formulario(seleccionado) : ""}</section>`;
  }

  // Las fases y los motivos son los mismos para todos los expedientes del
  // canal: se consultan con el primero en curso. Un 404 (sin componer) o un
  // 403 (perfil que no cancela) ocultan la sección.
  // La primera carga comparte la lectura de incorporaciones; las demás piden datos nuevos.
  async function cargar(recargar = true) {
    datos = null; pintar();
    try {
      const filas = (await bandeja.bandeja({ recargar })).expedientes;
      await prepararCausasFin(filas);
      const enCurso = filas.filter((e) => e.estado === "en_curso");
      if (enCurso.length === 0) {
        datos = { expedientes: filas.filter((e) => e.estado === "cancelado"), opciones: { motivos: [], fases_admitidas: [] } };
      } else {
        const opciones = await cliente.consultar(enCurso[0].expediente_ref);
        datos = { opciones, expedientes: filas.filter((e) => e.estado === "cancelado" || (e.estado === "en_curso" && opciones.fases_admitidas.includes(e.fase))) };
      }
    } catch (error) {
      datos = error?.estado === 404 || error?.estado === 403 ? { ausente: true } : { error: true };
    }
    pintar();
  }

  async function enviar(form) {
    if (ocupado) return;
    const e = datos?.expedientes?.find((x) => x.expediente_ref === form.dataset.ccForm);
    if (!e) return;
    const campos = Object.fromEntries(new FormData(form).entries());
    let solicitud;
    try {
      if (campos.confirmacion !== "on") throw new TypeError("sin confirmación");
      solicitud = validarSolicitudCancelacion({ expediente_ref: e.expediente_ref, version_esperada: e.version, clave_idempotencia: claveDe(e.expediente_ref),
        motivo_clave: String(campos.motivo_clave ?? ""), observaciones: String(campos.observaciones ?? "").trim() });
    } catch {
      aviso = { tono: "peligro", texto: t("error_datos") }; pintar(); contenedor.querySelector("[data-cc-aviso]")?.focus?.(); return;
    }
    ocupado = true; aviso = { tono: "aviso", texto: t("enviando") }; pintar();
    try {
      const recibo = await cliente.cancelar(solicitud);
      claves.delete(e.expediente_ref);
      ocupado = false; abierto = null;
      await cargar();
      bandeja.avisarCambio?.();
      aviso = { tono: "exito", texto: t("exito"), recibo: recibo.recibo_ref };
      pintar();
    } catch (error) {
      ocupado = false;
      if (error?.indeterminado) aviso = { tono: "aviso", texto: t("error_pendiente") };
      else {
        claves.delete(e.expediente_ref);
        const conocido = ["fase_no_admitida", "tras_fiscalizacion", "cancelacion_existente", "version_en_conflicto", "clave_reutilizada", "acceso_denegado"].includes(error?.codigo);
        aviso = { tono: "peligro", texto: t(conocido ? `error_${error.codigo}` : error?.codigo === "contenido_no_valido" ? "error_datos" : "error_general") };
        if (["fase_no_admitida", "cancelacion_existente", "version_en_conflicto", "tras_fiscalizacion"].includes(error?.codigo)) {
          abierto = null; const guardado = aviso; await cargar(); bandeja.avisarCambio?.(); aviso = guardado;
        }
      }
      pintar();
    }
    contenedor.querySelector("[data-cc-aviso]")?.focus?.();
  }

  const alPulsar = (evento) => {
    const boton = evento.target?.closest?.("button");
    if (!boton || ocupado) return;
    if (boton.matches("[data-cc-recargar]")) { cargar(); return; }
    if (boton.matches("[data-cc-volver]")) { abierto = null; pintar(); return; }
    if (boton.dataset.ccAbrir) { abierto = abierto === boton.dataset.ccAbrir ? null : boton.dataset.ccAbrir; aviso = null; pintar(); }
  };
  const alEnviar = (evento) => {
    const form = evento.target?.closest?.("[data-cc-form]");
    if (!form) return;
    evento.preventDefault();
    enviar(form);
  };
  instalarCopiaJustificantes(contenedor.ownerDocument ?? globalThis.document);
  contenedor.addEventListener("click", alPulsar);
  contenedor.addEventListener("submit", alEnviar);
  cargar(false);
  return () => {
    contenedor.removeEventListener("click", alPulsar);
    contenedor.removeEventListener("submit", alEnviar);
  };
}

/** Rellena la ayuda «?» de la página con los textos de esta sección. */
export function instalarAyudaCancelacionesCentro(doc, t = crearTraductorCancelacionesCentro()) {
  for (const elemento of doc?.querySelectorAll?.("[data-i18n-ayuda-cancelacion]") ?? []) {
    elemento.textContent = t(elemento.dataset.i18nAyudaCancelacion);
  }
}

if (typeof document !== "undefined" && document.querySelector("#cancelaciones-centro")) {
  instalarAyudaCancelacionesCentro(document);
  montarCancelacionesCentro({ contenedor: document.querySelector("#cancelaciones-centro") });
}
