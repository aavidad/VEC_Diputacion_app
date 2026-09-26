/**
 * Plazo de respuesta del llamamiento. El contacto efectivo abre el plazo (el
 * aviso por correo no); el vencimiento, la regla aplicable y la propuesta de
 * expiración proceden del recibo del servidor. La situación «en plazo» o
 * «vencido» es solo lectura: el servidor decide con su propio reloj.
 */
import { escaparHTML as e } from "./componentes-expedientes.js";
import { CAMPOS_REVISION_RESOLUCION, respuestaFueraDePlazo, situacionPlazoRespuesta } from "./contrato-llamamiento.js";

/** Estado derivado del plazo, compartido por el formulario y la vista. */
export function lecturaPlazoLlamamiento(estado, ahora) {
  const plazo = estado.contacto?.recibo?.plazo ?? null;
  if (!plazo) return null;
  const respuesta = estado.respuesta?.recibo ?? null;
  const fuera = respuesta ? respuestaFueraDePlazo(plazo, respuesta.recibida_en) : false;
  const situacion = situacionPlazoRespuesta(plazo, ahora);
  return Object.freeze({
    plazo, situacion, fuera,
    // VEC propone al vencer sin respuesta; RRHH confirma la propuesta.
    propuestaExpiracion: situacion === "vencido" && !respuesta && !estado.resolucion?.recibo,
    exigeCausa: fuera && plazo.tratamiento_fuera_de_plazo === "exige_causa_justificada",
    noAdmitida: fuera && plazo.tratamiento_fuera_de_plazo === "no_admitir",
  });
}

export function renderizarPlazoLlamamiento(estado, t, tiempoVisible, ahora) {
  if (!estado.plazoDisponible || estado.comunicacion?.recibo?.version_resultante !== 2) return "";
  const lectura = lecturaPlazoLlamamiento(estado, ahora);
  function estadoPaso(operacion) {
    const paso = estado[operacion];
    return `<div class="ct-estado ct-estado-${e(paso.tono)}" data-ct-llamamiento-estado="${operacion}"
      role="${paso.tono === "error" ? "alert" : "status"}" aria-live="polite" aria-atomic="true"
      tabindex="-1">${e(t(paso.mensaje))}</div>`;
  }
  function entrada(operacion, nombre, tipo, etiqueta, bloqueado) {
    const id = `ct-llamamiento-${operacion}-${nombre}`;
    const valor = estado[operacion].valores[nombre] ?? "";
    const visible = tipo === "datetime-local" ? String(valor).replace(/Z$/u, "") : valor;
    const atributos = tipo === "datetime-local" ? 'type="datetime-local" step="0.000001"' : 'type="text" maxlength="160"';
    return `<div class="ct-campo"><label for="${id}">${e(t(etiqueta))} *</label>
      <input id="${id}" name="${nombre}" ${atributos} value="${e(visible)}" required autocomplete="off"
        spellcheck="false"${bloqueado ? " readonly" : ""}></div>`;
  }
  function acciones(operacion, envio) {
    const paso = estado[operacion];
    return `<div class="ct-acciones">
      ${paso.solicitud === null && !paso.claveConservada ? `<button class="boton-secundario" type="button"
        data-ct-llamamiento-clave="${operacion}">${e(t("llamamiento_crear_clave"))}</button>` : ""}
      ${!paso.recibo && !paso.bloqueado ? `<button class="boton-primario" type="submit"${paso.ocupado ? " disabled" : ""}>${e(t(
        paso.solicitud !== null ? "llamamiento_recuperar" : envio))}</button>` : ""}
    </div>`;
  }
  function formularioEvento(operacion, etiquetas) {
    const paso = estado[operacion];
    const bloqueado = paso.solicitud !== null;
    return `<form data-ct-llamamiento-form="${operacion}" novalidate aria-busy="${paso.ocupado}">
      <fieldset${paso.ocupado ? " disabled" : ""}><legend>${e(t("llamamiento_" + operacion))}</legend>
        <div class="ct-campos">
          ${entrada(operacion, "clave_idempotencia", "text", "llamamiento_clave_idempotencia", bloqueado || paso.claveConservada)}
          ${entrada(operacion, "instante_en", "datetime-local", etiquetas.instante, bloqueado)}
          ${entrada(operacion, "prueba_ref", "text", etiquetas.prueba, bloqueado)}
        </div></fieldset>
      ${acciones(operacion, etiquetas.envio)}
    </form>${estadoPaso(operacion)}`;
  }
  function formularioExpiracion() {
    const paso = estado.expiracion;
    const bloqueado = paso.solicitud !== null;
    return `<form data-ct-llamamiento-form="expiracion" novalidate aria-busy="${paso.ocupado}">
      <fieldset${paso.ocupado ? " disabled" : ""}><legend>${e(t("llamamiento_expiracion"))}</legend>
        <div class="ct-campos">
          ${entrada("expiracion", "clave_idempotencia", "text", "llamamiento_clave_idempotencia", bloqueado || paso.claveConservada)}
          ${CAMPOS_REVISION_RESOLUCION.map((nombre) => `<div class="ct-campo"><label for="ct-llamamiento-expiracion-${nombre}">
            <input id="ct-llamamiento-expiracion-${nombre}" name="${nombre}" type="checkbox" autocomplete="off"${
  paso.valores[nombre] === true ? " checked" : ""}${bloqueado ? " disabled" : ""}> ${e(t("llamamiento_" + nombre + "_expiracion"))}</label></div>`).join("")}
        </div></fieldset>
      ${acciones("expiracion", "llamamiento_confirmar_expiracion")}
    </form>${estadoPaso("expiracion")}`;
  }
  function recibo(operacion, filas) {
    const datos = estado[operacion].recibo;
    if (!datos) return "";
    return `<section class="ct-recibo" data-ct-llamamiento-recibo="${operacion}" tabindex="-1"
      aria-labelledby="ct-llamamiento-recibo-${operacion}">
      <h4 id="ct-llamamiento-recibo-${operacion}">${e(t("llamamiento_" + operacion + "_recibo"))}</h4>
      <dl>${filas.map(([etiqueta, valor]) => `<div><dt>${e(t(etiqueta))}</dt><dd>${valor}</dd></div>`).join("")}</dl>
    </section>`;
  }
  const partes = [];
  if (!estado.contacto.recibo) {
    partes.push(formularioEvento("contacto", {
      instante: "llamamiento_instante_en", prueba: "llamamiento_prueba_ref", envio: "llamamiento_registrar_contacto",
    }));
  } else {
    const { plazo, situacion } = lectura;
    partes.push(`<dl class="ct-llamamiento-plazo-resumen" data-ct-llamamiento-plazo-situacion="${situacion}">
      <div><dt>${e(t("llamamiento_plazo_vence"))}</dt><dd>${tiempoVisible(plazo.respuesta_hasta) || e(plazo.respuesta_hasta)}</dd></div>
      <div><dt>${e(t("llamamiento_plazo_ultimo_dia"))}</dt><dd>${e(plazo.ultimo_dia)}</dd></div>
      <div><dt>${e(t("llamamiento_plazo_situacion"))}</dt><dd><span class="ct-insignia ct-insignia-plazo-${situacion}">${e(t("llamamiento_plazo_" + situacion))}</span></dd></div>
      <div><dt>${e(t("llamamiento_plazo_tratamiento"))}</dt><dd>${e(t("llamamiento_plazo_tratamiento_" + plazo.tratamiento_fuera_de_plazo))}</dd></div>
    </dl>`);
    partes.push(recibo("contacto", [
      ["llamamiento_instante_en", tiempoVisible(estado.contacto.recibo.instante_en) || e(estado.contacto.recibo.instante_en)],
      ["llamamiento_prueba_ref", e(estado.contacto.recibo.prueba_ref)],
      ["llamamiento_estado_local", e(t("llamamiento_" + estado.contacto.recibo.estado))],
    ]));
    if (lectura.fuera) {
      partes.push(`<p class="ct-exp-mensaje ct-tono-aviso" role="status" data-ct-llamamiento-fuera-de-plazo>${e(t(
        lectura.noAdmitida ? "llamamiento_plazo_no_admitida" : "llamamiento_plazo_fuera_de_plazo"))}</p>`);
    }
    if (lectura.exigeCausa || estado.causa.recibo) {
      partes.push(estado.causa.recibo ? recibo("causa", [
        ["llamamiento_prueba_ref_causa", e(estado.causa.recibo.prueba_ref)],
        ["llamamiento_estado_local", e(t("llamamiento_" + estado.causa.recibo.estado))],
      ]) : formularioEvento("causa", {
        instante: "llamamiento_instante_en_causa", prueba: "llamamiento_prueba_ref_causa", envio: "llamamiento_acreditar_causa",
      }));
    }
    if (lectura.propuestaExpiracion || estado.expiracion.solicitud !== null || estado.expiracion.recibo) {
      partes.push(`<section class="ct-llamamiento-propuesta-expiracion" aria-labelledby="ct-llamamiento-propuesta-expiracion-titulo"
        data-ct-llamamiento-propuesta-expiracion>
        <h4 id="ct-llamamiento-propuesta-expiracion-titulo">${e(t("llamamiento_propuesta_expiracion_titulo"))}</h4>
        <p>${e(t("llamamiento_propuesta_expiracion"))}</p>
        ${estado.expiracion.recibo ? recibo("expiracion", [
          ["llamamiento_estado_plazo", e(t("llamamiento_plazo_" + estado.expiracion.recibo.estado_plazo))],
          ["llamamiento_resolucion_ref", e(estado.expiracion.recibo.resolucion_ref)],
          ["llamamiento_resuelta_en", tiempoVisible(estado.expiracion.recibo.resuelta_en) || e(estado.expiracion.recibo.resuelta_en)],
          ["llamamiento_intencion_siguiente_estado_local", e(t("llamamiento_intencion_siguiente_" + estado.expiracion.recibo.intencion_siguiente.estado_local))],
        ]) : formularioExpiracion()}
      </section>`);
    }
  }
  return `<section class="ct-llamamiento-paso ct-llamamiento-plazo" aria-labelledby="ct-llamamiento-plazo-titulo"
    data-ct-llamamiento-plazo>
    <h3 id="ct-llamamiento-plazo-titulo">${e(t("llamamiento_plazo_titulo"))}</h3>
    ${partes.join("")}
  </section>`;
}
