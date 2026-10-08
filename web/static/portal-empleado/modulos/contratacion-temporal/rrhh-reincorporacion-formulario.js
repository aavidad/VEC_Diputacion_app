import { validarReciboReincorporacionRRHH, validarSolicitudReincorporacionRRHH } from "./rrhh-reincorporacion-contrato.js";
import { crearTraductorReincorporacionRRHH } from "./rrhh-reincorporacion-i18n.js?v=20261007-pantallas-textos-final-v1";

const REF = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const traducirPorDefecto = crearTraductorReincorporacionRRHH();
const esc = (valor) => String(valor ?? "").replaceAll("&", "&amp;")
  .replaceAll("<", "&lt;").replaceAll(">", "&gt;")
  .replaceAll('"', "&quot;").replaceAll("'", "&#39;");
const k = (nombre) => `rrhh_reincorporacion_${nombre}`;

function contextoValido(expediente) {
  return expediente && typeof expediente === "object" && !Array.isArray(expediente)
    && typeof expediente.expediente_ref === "string" && REF.test(expediente.expediente_ref)
    && Number.isSafeInteger(expediente.version_esperada) && expediente.version_esperada > 0;
}

function campo(t, nombre, etiqueta, valor, atributos = "") {
  return `<label class="ct-rrhh-reincorporacion-campo" for="ct-reincorp-${nombre}"><span>${esc(t(k(etiqueta)))}</span><input id="ct-reincorp-${nombre}" name="${nombre}" value="${esc(valor)}" ${atributos} required></label>`;
}

function fila(etiqueta, valor) {
  return `<div><dt>${esc(etiqueta)}</dt><dd>${esc(valor)}</dd></div>`;
}

export function renderizarReincorporacionRRHH({ expediente, datos, estado, recibo, ayudaAbierta, ocupado, puedeRegistrar, t = traducirPorDefecto, fecha = (valor) => valor }) {
  if (!contextoValido(expediente) || typeof t !== "function") throw new TypeError("vista de reincorporación RRHH no válida");
  const aviso = t(k(estado));
  const formulario = !recibo && puedeRegistrar ? `<form data-ct-rrhh-reincorporacion-form aria-busy="${ocupado}"><fieldset ${ocupado || estado === "incierta" || estado === "conflicto" ? "disabled" : ""}>
    <div class="ct-rrhh-reincorporacion-campos">
      ${campo(t, "relacion_ref", "relacion", datos.relacion_ref, 'autocomplete="off" maxlength="160"')}
      ${campo(t, "fecha_efectiva", "fecha", datos.fecha_efectiva, 'type="date"')}
      ${campo(t, "documento_ref", "documento", datos.documento_ref, 'autocomplete="off" maxlength="160"')}
      ${campo(t, "documento_sha256", "huella", datos.documento_sha256, 'autocomplete="off" inputmode="text" pattern="[0-9a-fA-F]{64}" maxlength="64"')}
    </div></fieldset>
    <div class="ct-rrhh-reincorporacion-acciones"><button class="boton-primario" type="submit" ${ocupado || estado === "conflicto" ? "disabled" : ""}>${esc(t(k(estado === "incierta" ? "reintentar" : "registrar")))}</button></div>
  </form>` : "";
  const resultado = recibo ? `<section class="ct-rrhh-reincorporacion-resultado" aria-labelledby="ct-reincorp-recibo-titulo">
    <h4 id="ct-reincorp-recibo-titulo">${esc(t(k("recibo")))}</h4>
    <dl>${fila(t(k("recibo_ref")), recibo.recibo_ref)}${fila(t(k("cese_recibo_ref")), recibo.cese_recibo_ref)}${fila(t(k("evento_ref")), recibo.evento_ref)}${fila(t(k("registrada_en")), fecha(recibo.registrada_en))}${fila(t(k("fecha")), recibo.fecha_efectiva)}</dl>
    <div class="ct-rrhh-reincorporacion-bolsa" role="status"><span>${esc(t(k("bolsa")))}</span><strong>${esc(t(k("bolsa_pendiente")))}</strong></div>
  </section>` : "";
  const clave = estado === "incierta" && datos.clave_idempotencia
    ? `<p class="ct-rrhh-reincorporacion-clave"><span>${esc(t(k("incierta_clave")))}</span> <code>${esc(datos.clave_idempotencia)}</code></p>` : "";
  return `<section class="ct-rrhh-reincorporacion panel" data-ct-rrhh-reincorporacion>
    <header class="cabecera-panel"><div><h3>${esc(t(k("titulo")))}</h3><p>${esc(t(k("subtitulo")))}</p></div>
      <button class="ct-rrhh-reincorporacion-ayuda-boton boton-secundario" type="button" data-ct-rrhh-reincorporacion-ayuda aria-label="${esc(t(k("ayuda_boton")))}" aria-expanded="${ayudaAbierta}" aria-controls="ct-reincorp-ayuda">?</button></header>
    <div class="cuerpo-panel">
      <div id="ct-reincorp-ayuda" class="ct-rrhh-reincorporacion-ayuda" ${ayudaAbierta ? "" : "hidden"}>${esc(t(k("ayuda")))}</div>
      <ol class="ct-rrhh-reincorporacion-pasos" aria-label="${esc(t(k("pasos")))}"><li><span aria-hidden="true">1</span>${esc(t(k("paso_cese")))}</li><li aria-current="step"><span aria-hidden="true">2</span>${esc(t(k("paso_reincorporacion")))}</li><li><span aria-hidden="true">3</span>${esc(t(k("paso_cierre")))}</li></ol>
      <dl class="ct-rrhh-reincorporacion-contexto">${fila(t(k("expediente")), expediente.expediente_ref)}${fila(t(k("version")), expediente.version_esperada)}</dl>
      ${formulario}${resultado}${clave}
      <p class="ct-rrhh-reincorporacion-estado" data-estado="${esc(estado)}" role="status" aria-live="polite">${esc(aviso)}</p>
    </div>
  </section>`;
}

export function montarFormularioReincorporacionRRHH({ raiz, cliente, expediente, puedeRegistrar = false,
  datosIniciales = {},
  t = traducirPorDefecto, confirmarOperacion = () => false,
  generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(),
  alConfirmar = () => {}, alDenegacion = () => {}, locale = "es-ES", zonaHoraria = "Europe/Madrid" } = {}) {
  if (!raiz?.addEventListener || !raiz?.removeEventListener || !raiz?.replaceChildren
    || !contextoValido(expediente) || typeof puedeRegistrar !== "boolean"
    || !datosIniciales || typeof datosIniciales !== "object" || Array.isArray(datosIniciales)
    || typeof cliente?.registrarReincorporacion !== "function" || typeof t !== "function"
    || typeof confirmarOperacion !== "function" || typeof generarClaveIdempotencia !== "function"
    || typeof alConfirmar !== "function" || typeof alDenegacion !== "function") {
    throw new TypeError("dependencias de reincorporación RRHH no válidas");
  }
  const contexto = Object.freeze({ expediente_ref: expediente.expediente_ref, version_esperada: expediente.version_esperada });
  const formateador = new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short", timeZone: zonaHoraria });
  let montado = true;
  let ocupado = false;
  let autorizado = puedeRegistrar;
  let ayudaAbierta = false;
  let estado = autorizado ? "lista" : "denegada";
  let controlador = null;
  let solicitud = null;
  let recibo = null;
  const datos = { relacion_ref: "", fecha_efectiva: "", documento_ref: "", documento_sha256: "", clave_idempotencia: "" };
  for (const nombre of ["relacion_ref", "fecha_efectiva", "documento_ref", "documento_sha256"]) {
    if (typeof datosIniciales[nombre] === "string") datos[nombre] = datosIniciales[nombre];
  }

  function pintar() {
    if (!montado) return;
    raiz.innerHTML = renderizarReincorporacionRRHH({ expediente: contexto, datos, estado, recibo, ayudaAbierta,
      ocupado, puedeRegistrar: autorizado, t, fecha: (valor) => formateador.format(new Date(valor)) });
  }

  function leerFormulario(formulario) {
    for (const nombre of ["relacion_ref", "fecha_efectiva", "documento_ref", "documento_sha256"]) {
      datos[nombre] = String(formulario.elements?.namedItem?.(nombre)?.value ?? "").trim();
    }
    datos.documento_sha256 = datos.documento_sha256.toLowerCase();
  }

  async function enviar(evento) {
    if (evento.target?.dataset?.ctRrhhReincorporacionForm === undefined) return;
    evento.preventDefault();
    if (!montado || ocupado || !autorizado || recibo || estado === "conflicto") return;
    if (estado !== "incierta") {
      leerFormulario(evento.target);
      try {
        solicitud = validarSolicitudReincorporacionRRHH({ ...contexto,
          relacion_ref: datos.relacion_ref, fecha_efectiva: datos.fecha_efectiva,
          documento_ref: datos.documento_ref, documento_sha256: datos.documento_sha256,
          clave_idempotencia: generarClaveIdempotencia() });
      } catch {
        solicitud = null; estado = "validacion"; pintar(); return;
      }
      let aceptada = false;
      try { aceptada = confirmarOperacion({ titulo: t(k("titulo")), advertencia: t(k("confirmar")), referencia: contexto.expediente_ref, datos: solicitud }) === true; } catch {}
      if (!aceptada) { solicitud = null; estado = "cancelada"; pintar(); return; }
      datos.clave_idempotencia = solicitud.clave_idempotencia;
    }
    ocupado = true; estado = "enviando"; controlador = new AbortController(); pintar();
    const { signal } = controlador;
    try {
      const respuesta = await cliente.registrarReincorporacion(solicitud, { signal });
      if (!montado || signal.aborted) return;
      recibo = validarReciboReincorporacionRRHH(respuesta, solicitud);
      estado = "confirmada";
      try { alConfirmar(recibo); } catch {}
    } catch (error) {
      if (!montado || signal.aborted) return;
      if (error?.estado === 401 || error?.estado === 403) {
        autorizado = false; solicitud = null; recibo = null;
        for (const nombre of Object.keys(datos)) datos[nombre] = "";
        estado = "denegada";
        try { alDenegacion(); } catch {}
      } else if (error?.estado === 409 && error?.resultadoIndeterminado === false) {
        solicitud = null; estado = "conflicto";
      } else if (error?.resultadoIndeterminado === false && error?.envelopeValido === true) {
        solicitud = null; estado = "rechazada";
      } else {
        estado = "incierta";
      }
    } finally {
      if (montado) { ocupado = false; controlador = null; pintar(); }
    }
  }

  function pulsar(evento) {
    if (evento.target?.closest?.("[data-ct-rrhh-reincorporacion-ayuda]")) {
      ayudaAbierta = !ayudaAbierta; pintar();
    }
  }

  raiz.addEventListener("submit", enviar);
  raiz.addEventListener("click", pulsar);
  pintar();
  return () => {
    montado = false; controlador?.abort();
    raiz.removeEventListener("submit", enviar);
    raiz.removeEventListener("click", pulsar);
    raiz.replaceChildren();
  };
}
