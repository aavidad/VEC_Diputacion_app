import { validarSolicitudResolucionFormalizacion, validarReciboResolucionFormalizacion,
  validarPreparacionResolucionFormalizacion } from "./contrato-resolucion-formalizacion.js";
import { escaparHTML as e } from "./componentes-expedientes.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";

const textosRecibo = Object.freeze({
  resolucion_formalizacion_estado: "Estado del registro",
  resolucion_formalizacion_tipo_validacion: "Naturaleza de la validación",
  resolucion_formalizacion_firma_oficial: "Firma oficial",
  resolucion_formalizacion_eficacia_administrativa: "Eficacia administrativa",
  resolucion_formalizacion_esquema: "Esquema del recibo",
  resolucion_formalizacion_detalles_trazabilidad: "Detalles de trazabilidad",
});

export function montarFormularioResolucionFormalizacion({
  raiz, cliente, preparacion, confirmarOperacion = () => false,
  generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(),
  mensajes = {}, locale = "es-ES", zonaHoraria = "Europe/Madrid",
} = {}) {
  if (!raiz?.addEventListener || !raiz?.replaceChildren
    || typeof cliente?.registrarResolucionFormalizacion !== "function") {
    throw new TypeError("dependencias de resolución de formalización no válidas");
  }
  const contexto = validarPreparacionResolucionFormalizacion(preparacion, preparacion?.expediente_ref);
  const t = crearTraductorContratacionTemporal({ ...textosRecibo, ...mensajes });
  const fecha = new Intl.DateTimeFormat(locale, {
    dateStyle: "medium", timeStyle: "medium", timeZone: zonaHoraria,
  });
  let montado = true;
  let recibo = contexto.recibo;
  let solicitudPendiente = null;
  let reintentoInmutable = false;
  let ocupado = false;
  let controlador = null;
  let valores = {
    numero_resolucion: "", fecha_resolucion: "", motivo: "",
    confirma_revision_propuesta: false, confirma_ejercicio_manual: false,
    clave_idempotencia: "",
  };
  const texto = (clave) => e(t("resolucion_formalizacion_" + clave));
  const fila = (clave, valor) => `<div><dt>${texto(clave)}</dt><dd>${e(String(valor))}</dd></div>`;
  const siNo = (valor) => valor ? "Sí" : "No";
  const estadoLegible = (estado) => estado === "registrada" ? "Registrada" : "Recibo recuperado";
  const validacionLegible = () => "Validación manual de ejercicio sintético";

  function pintar(mensaje = "", focoCorreccion = false) {
    if (!montado) return;
    const v = valores;
    const camposRecibo = ["esquema", "estado", "tipo_validacion", "expediente_ref", "propuesta_ref", "resolucion_formalizacion_ref", "documento_resolucion_ref",
      "documento_resolucion_version", "documento_resolucion_sha256", "version_resultante",
      "actuacion_ref", "auditoria_ref", "outbox_ref"];
    raiz.innerHTML = `<section class="ct-alta" data-ct-resolucion-formalizacion>
      <h3>${texto("titulo")}</h3>
      <p id="ct-rf-ayuda" class="ct-ayuda">${texto("ayuda")}</p>
      <dl class="ct-resumen" aria-label="${texto("contexto")}">
        ${fila("expediente_ref", contexto.expediente_ref)}
        ${fila("propuesta_ref", contexto.propuesta_ref)}
        ${fila("version_esperada", contexto.version_esperada)}
        ${fila("version_actual", recibo?.version_resultante ?? contexto.version_actual)}
      </dl>
      ${recibo ? `<section class="ct-recibo" role="status">
        <h4>${texto("recibo_titulo")}</h4><p>${texto("limites")}</p>
        <dl>${fila("recibo_ref", recibo.recibo_ref)}${fila("registrada_en", `${fecha.format(new Date(recibo.registrada_en))} · ${recibo.registrada_en}`)}
          ${fila("estado", estadoLegible(recibo.estado))}${fila("tipo_validacion", validacionLegible())}
          ${fila("firma_oficial", siNo(recibo.firma_oficial))}${fila("eficacia_administrativa", siNo(recibo.eficacia_administrativa))}</dl>
        <details><summary>${texto("detalles_trazabilidad")}</summary><dl>${camposRecibo.map((campo) => fila(campo, recibo[campo])).join("")}</dl></details>
        <button type="button" class="boton-secundario" data-ct-exp-accion="volver-cuadro-actualizado">${texto("volver_cuadro")}</button>
      </section>` : `<form data-ct-resolucion-formalizacion-form aria-busy="${ocupado}">
        <fieldset${ocupado || reintentoInmutable ? " disabled" : ""} aria-describedby="ct-rf-ayuda">
          <legend>${texto("contexto")}</legend>
          <div class="ct-campo"><label for="ct-rf-numero">${texto("numero")} *</label>
            <input id="ct-rf-numero" required name="numero_resolucion" maxlength="80" autocomplete="off" value="${e(v.numero_resolucion)}"></div>
          <div class="ct-campo"><label for="ct-rf-fecha">${texto("fecha")} *</label>
            <input id="ct-rf-fecha" required type="date" name="fecha_resolucion" value="${e(v.fecha_resolucion)}"></div>
          <div class="ct-campo"><label for="ct-rf-motivo">${texto("motivo")} *</label>
            <textarea id="ct-rf-motivo" required name="motivo" maxlength="2000">${e(v.motivo)}</textarea></div>
          <div class="ct-campo"><label><input type="checkbox" name="confirma_revision_propuesta"${v.confirma_revision_propuesta ? " checked" : ""}> ${texto("revision")}</label></div>
          <div class="ct-campo"><label><input type="checkbox" name="confirma_ejercicio_manual"${v.confirma_ejercicio_manual ? " checked" : ""}> ${texto("ejercicio")}</label></div>
        </fieldset>
        <div class="ct-campo"><label for="ct-rf-clave">${texto("clave")}</label>
          <input id="ct-rf-clave" name="clave_idempotencia" readonly value="${e(v.clave_idempotencia)}"></div>
        <div class="ct-acciones"><button class="boton-primario" type="submit"${ocupado ? " disabled" : ""}>
          ${texto(solicitudPendiente ? "reintentar" : "registrar")}</button></div>
      </form>`}
      <p role="status" aria-live="polite">${texto(mensaje || (recibo ? "historico" : "pendiente"))}</p>
    </section>`;
    const documento = raiz.ownerDocument ?? globalThis.document;
    if (focoCorreccion && documento && documento.activeElement === documento.body) {
      raiz.querySelector?.("[name=numero_resolucion]")?.focus?.();
    }
  }

  function capturar(formulario) {
    const controles = formulario.elements;
    valores = {
      numero_resolucion: controles.numero_resolucion.value,
      fecha_resolucion: controles.fecha_resolucion.value,
      motivo: controles.motivo.value,
      confirma_revision_propuesta: controles.confirma_revision_propuesta.checked,
      confirma_ejercicio_manual: controles.confirma_ejercicio_manual.checked,
      // La clave retenida prevalece sobre cualquier control DOM.
      clave_idempotencia: valores.clave_idempotencia || generarClaveIdempotencia() || "",
    };
  }

  async function recuperarConflicto(signal) {
    // Lectura de historia, no confirmación del material posiblemente divergente.
    // Ni un GET v7 vacío ni un fallo de lectura liberan esta clave para editar.
    pintar("conflicto_consultando");
    try {
      if (!montado || signal.aborted) return "incierta";
      const resultado = await cliente.prepararResolucionFormalizacion(contexto.expediente_ref, { signal });
      if (!montado || signal.aborted) return "incierta";
      const historico = validarPreparacionResolucionFormalizacion(resultado, contexto.expediente_ref);
      if (historico.propuesta_ref !== contexto.propuesta_ref) return "conflicto_lectura_fallida";
      if (historico.version_actual === 8 && historico.recibo !== null) {
        recibo = historico.recibo;
        return "conflicto_historico";
      }
      return "conflicto_sin_recibo";
    } catch {
      return "conflicto_lectura_fallida";
    }
  }

  async function enviar(evento) {
    evento.preventDefault();
    if (!montado || ocupado || recibo) return;
    const recuperando = reintentoInmutable;
    if (!recuperando) {
      try {
        capturar(evento.target);
        solicitudPendiente = validarSolicitudResolucionFormalizacion({
          expediente_ref: contexto.expediente_ref, version_esperada: contexto.version_esperada,
          propuesta_ref: contexto.propuesta_ref, ...valores,
        });
      } catch {
        solicitudPendiente = null;
        pintar("validacion");
        return;
      }
    }
    let confirmado = false;
    try {
      confirmado = confirmarOperacion({
        titulo: t("resolucion_formalizacion_titulo"),
        advertencia: t("resolucion_formalizacion_ayuda"),
        referencia: solicitudPendiente.expediente_ref, datos: solicitudPendiente,
      }) === true;
    } catch { /* Una confirmación fallida no inicia el POST. */ }
    if (!confirmado) {
      if (!recuperando) solicitudPendiente = null;
      pintar(recuperando ? "recuperacion_cancelada" : "cancelada");
      return;
    }
    if (!montado) return;
    let mensaje = "";
    let focoCorreccion = false;
    let respuestaRecibida = false;
    ocupado = true;
    controlador = new AbortController();
    pintar("enviando");
    try {
      const respuesta = await cliente.registrarResolucionFormalizacion(solicitudPendiente, {
        signal: controlador.signal,
      });
      respuestaRecibida = true;
      recibo = validarReciboResolucionFormalizacion(respuesta, solicitudPendiente);
      mensaje = "confirmada";
    } catch (error) {
      // Una denegación de replay no borra la incertidumbre del intento anterior.
      const determinado = !recuperando && !respuestaRecibida
        && error?.estado !== 409
        && error?.resultadoIndeterminado === false && error?.envelopeValido === true;
      if (!respuestaRecibida && error?.estado === 409) {
        reintentoInmutable = true;
        mensaje = await recuperarConflicto(controlador.signal);
      } else if (determinado) {
        solicitudPendiente = null;
        reintentoInmutable = false;
        mensaje = "rechazada";
        focoCorreccion = true;
      } else {
        reintentoInmutable = true;
        mensaje = "incierta";
      }
    } finally {
      ocupado = false;
      controlador = null;
      pintar(mensaje, focoCorreccion);
    }
  }

  raiz.addEventListener("submit", enviar);
  pintar();
  return () => {
    montado = false;
    controlador?.abort();
    raiz.removeEventListener("submit", enviar);
    raiz.replaceChildren();
  };
}
