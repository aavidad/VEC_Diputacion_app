/**
 * Cancelación del expediente por el centro solicitante (duda 12 de RRHH).
 *
 * Reutiliza la lectura de los expedientes de las peticiones del centro (la
 * bandeja de incorporaciones) para ligar cada petición con su expediente y,
 * en los que siguen en una fase que admite el catálogo, ofrece «Cancelar» con
 * un motivo del catálogo. Fases y motivos los decide el servidor; la vista
 * solo evita ofrecer la cancelación cuando no procede.
 */
import { crearClienteIncorporacionesCentro } from "./incorporaciones-centro.js?v=20260926-pulido-portal-v1";
import { validarConsultaCancelacion, validarReciboCancelacion, validarSolicitudCancelacion } from "../modulos/contratacion-temporal/cliente-http-cancelacion.js?v=20260926-huecos-rrhh-v1";
import { instalarCopiaJustificantes, renderizarJustificante } from "../portal-justificante.js";

export const RUTAS_CANCELACIONES_CENTRO = Object.freeze({
  consulta: "/api/vec/contratacion-temporal/peticiones-centro/cancelacion",
  cancelaciones: "/api/vec/contratacion-temporal/peticiones-centro/cancelaciones",
});
const MAXIMO_RESPUESTA = 32 * 1024;
const TIEMPO_MAXIMO_MS = 15_000;

export const MENSAJES_CANCELACIONES_CENTRO_ES = Object.freeze({
  titulo: "Cancelar un expediente",
  cargando: "Consultando los expedientes que se pueden cancelar…",
  sin_expedientes: "No hay expedientes de este centro que se puedan cancelar.",
  error_lectura: "No se han podido consultar las cancelaciones. Inténtelo de nuevo.",
  reintentar: "Volver a consultar",
  expediente: "Expediente",
  periodo: "Periodo solicitado",
  situacion: "Situación",
  cancelacion: "Cancelación",
  cancelar: "Cancelar",
  cancelado: "Cancelado",
  motivo: "Motivo de la cancelación",
  observaciones: "Observaciones (opcional)",
  confirmacion_expresa: "Confirmo que el expediente quedará cancelado y no admitirá más actuaciones.",
  enviar: "Cancelar el expediente",
  volver: "Volver sin cancelar",
  enviando: "Cancelando; espere el recibo.",
  exito: "Expediente cancelado.",
  error_datos: "Elija un motivo, revise las observaciones y confirme la cancelación.",
  error_fase_no_admitida: "El expediente ya no está en una fase en la que se pueda cancelar. Se ha actualizado la lista.",
  error_tras_fiscalizacion: "El expediente ya pasó por fiscalización y no se puede cancelar.",
  error_cancelacion_existente: "El expediente ya estaba cancelado. Se ha actualizado la lista.",
  error_version_en_conflicto: "El expediente ha cambiado. Se ha actualizado la lista; revíselo antes de continuar.",
  error_clave_reutilizada: "Esta cancelación ya se registró con otros datos.",
  error_acceso_denegado: "No tiene permiso para cancelar este expediente.",
  error_pendiente: "No se ha podido saber si quedó registrada. Pulse de nuevo: se usará la misma operación y no se duplicará.",
  error_general: "No se ha podido cancelar el expediente. Inténtelo de nuevo más tarde.",
  justificante_registrado: "Justificante de la cancelación",
  justificante_copiar: "Copiar la referencia del recibo",
  justificante_copiado: "Referencia copiada",
  fase_solicitud: "Solicitud en RRHH",
  fase_asignacion_unidad: "Asignación de unidad",
  fase_informe_juridico: "Informe jurídico",
  fase_otra: "En tramitación en RRHH",
  ayuda_titulo: "¿Cuándo puede el centro cancelar un expediente?",
  ayuda: "Mientras RRHH no haya fiscalizado el expediente, el centro puede cancelarlo si la necesidad ha desaparecido o por otro motivo del catálogo. Solo aparecen los expedientes de las peticiones de su centro que siguen en una fase en la que el catálogo admite la cancelación. Elija el motivo y, si quiere, añada una observación. El expediente queda cancelado, conserva toda su historia y RRHH ve quién lo canceló y por qué.",
});

export function crearTraductorCancelacionesCentro(mensajes = MENSAJES_CANCELACIONES_CENTRO_ES) {
  return (clave, valores = {}) => {
    const plantilla = typeof mensajes?.[clave] === "string" ? mensajes[clave] : MENSAJES_CANCELACIONES_CENTRO_ES[clave] ?? clave;
    return plantilla.replace(/\{([a-z_]+)\}/gu, (_, n) => (Object.hasOwn(valores, n) ? String(valores[n]) : `{${n}}`));
  };
}

const escapar = (v) => String(v ?? "").replace(/[&<>"']/gu, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);

function fechaVisible(valor) {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(valor)) return "—";
  const f = new Date(`${valor}T00:00:00Z`);
  return Number.isFinite(f.getTime()) ? new Intl.DateTimeFormat("es-ES", { dateStyle: "long", timeZone: "UTC" }).format(f) : valor;
}

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
export function montarCancelacionesCentro({ contenedor, bandeja = crearClienteIncorporacionesCentro(), cliente = crearClienteCancelacionesCentro(), mensajes,
  generarClave = () => globalThis.crypto?.randomUUID?.() } = {}) {
  if (!contenedor || typeof contenedor.addEventListener !== "function") throw new TypeError("contenedor no válido");
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
    const periodo = e.periodo ? `${fechaVisible(e.periodo.inicio)} — ${fechaVisible(e.periodo.fin)}` : "—";
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
  async function cargar() {
    datos = null; pintar();
    try {
      const filas = (await bandeja.bandeja()).expedientes;
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
          abierto = null; const guardado = aviso; await cargar(); aviso = guardado;
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
  cargar();
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
