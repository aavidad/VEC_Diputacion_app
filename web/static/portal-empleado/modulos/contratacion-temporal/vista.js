import { LIMITES_ALTA_CONTRATACION, numeroExpedienteMOADValido } from "./contrato.js?v=20261002-ct-fin-moad-v1";
import { crearTraductorContratacionTemporal } from "./i18n.js?v=20261007-pantallas-textos-final-v1";
import { cabecera, escaparHTML, extraerBorrador, filaResumen, formulario, revision } from "./alta-renderer-puro.js?v=20261007-pantallas-textos-final-v1";
import { justificanteTraducido } from "../../portal-justificante.js";

function recibo(estado, t, locale, zonaHoraria) {
  const dato = estado.recibo;
  const fecha = new Intl.DateTimeFormat(locale, {
    dateStyle: "long",
    timeStyle: "medium",
    timeZone: zonaHoraria,
  }).format(new Date(dato.confirmada_en));
  return `<section class="ct-recibo" data-ct-recibo role="status" aria-live="polite"
    aria-atomic="true" tabindex="-1" aria-labelledby="ct-recibo-titulo">
    <p class="sobrelinea">${escaparHTML(t("recibo_sobrelinea"))}</p>
    <h3 id="ct-recibo-titulo">${escaparHTML(t("recibo_titulo"))}</h3>
    <p>${escaparHTML(t("recibo_descripcion"))}</p>
    <dl>
      ${filaResumen(t("recibo_numero_visible"), dato.numero_visible)}
      ${filaResumen(t("recibo_version"), dato.version)}
      <div><dt>${escaparHTML(t("recibo_ref"))}</dt><dd>${justificanteTraducido(dato.recibo_ref, escaparHTML, t)}</dd></div>
      ${filaResumen(t("recibo_fecha"), fecha)}
    </dl>
  </section>`;
}

export function renderizarAltaContratacionTemporal(estado, {
  mensajes = {},
  locale = "es-ES",
  zonaHoraria = "Europe/Madrid",
} = {}) {
  const t = crearTraductorContratacionTemporal(mensajes);
  const contenido = estado.fase === "edicion"
    ? formulario(estado, t)
    : (estado.fase === "recibo"
      ? recibo(estado, t, locale, zonaHoraria)
      : revision(estado, t, locale));
  return `<section class="ct-alta" data-modulo="contratacion-temporal"
    aria-labelledby="ct-alta-titulo">
    ${cabecera(estado, t)}
    ${contenido}
  </section>`;
}

// Adaptadores mínimos para el circuito previo: reutilizan el formulario y la
// revisión del alta sin crear un segundo formulario ni un recibo de expediente.
export function renderizarFormularioPeticionCentro(estado, opciones = {}) {
  return formulario(estado, crearTraductorContratacionTemporal(opciones.mensajes));
}

export function renderizarRevisionPeticionCentro(estado, opciones = {}) {
  return revision(estado, crearTraductorContratacionTemporal(opciones.mensajes), opciones.locale ?? "es-ES");
}

export function extraerBorradorPeticionCentro(formularioDOM) {
  return extraerBorrador(formularioDOM, false);
}

function enfocarVisible(elemento) {
  elemento?.focus?.();
  elemento?.scrollIntoView?.({ block: "nearest", inline: "nearest" });
}

export function montarAltaContratacionTemporal({
  raiz,
  presentador,
  mensajes = {},
  anunciar = () => {},
  locale = "es-ES",
  zonaHoraria = "Europe/Madrid",
} = {}) {
  if (!raiz || typeof raiz.addEventListener !== "function"
    || typeof raiz.querySelector !== "function"
    || typeof presentador?.obtenerEstado !== "function"
    || typeof anunciar !== "function") {
    throw new TypeError("dependencias DOM del alta no válidas");
  }
  const t = crearTraductorContratacionTemporal(mensajes);
  let montada = true;

  function repintar(selectorFoco = "") {
    if (!montada) return;
    const estado = presentador.obtenerEstado();
    raiz.innerHTML = renderizarAltaContratacionTemporal(estado, {
      mensajes,
      locale,
      zonaHoraria,
    });
    if (selectorFoco) enfocarVisible(raiz.querySelector(selectorFoco));
    anunciar(t(estado.mensaje_clave), estado.tipo_mensaje);
  }

  function enfocarTrasValidacion() {
    const estado = presentador.obtenerEstado();
    if (Object.keys(estado.errores).length > 0) {
      repintar("[data-ct-error-general]");
      return;
    }
    repintar("#ct-revision-titulo");
  }

  async function alPulsar(evento) {
    const enfocar = evento.target?.closest?.("[data-ct-enfocar]");
    if (enfocar && raiz.contains(enfocar)) {
      evento.preventDefault();
      const campo = enfocar.dataset.ctEnfocar;
      if (Object.hasOwn(presentador.obtenerEstado().borrador, campo)) {
        enfocarVisible(raiz.querySelector(`#ct-${campo}`));
      }
      return;
    }
    const control = evento.target?.closest?.("[data-ct-accion]");
    if (!control || !raiz.contains(control)) return;
    evento.preventDefault();
    if (control.dataset.ctAccion === "volver") {
      presentador.volverAEdicion();
      repintar("#ct-centro_ref");
      return;
    }
    if (control.dataset.ctAccion === "cancelar") {
      presentador.cancelarEnvio();
      repintar("[data-ct-estado]");
      return;
    }
    if (control.dataset.ctAccion === "confirmar") {
      const tarea = presentador.enviar();
      repintar("[data-ct-accion='cancelar']");
      await tarea;
      const estado = presentador.obtenerEstado();
      repintar(estado.fase === "recibo"
        ? "[data-ct-recibo]"
        : (estado.fase === "pendiente"
          ? "[data-ct-operacion-pendiente]"
          : "[data-ct-accion='confirmar']"));
    }
  }

  function alEnviar(evento) {
    const formularioDOM = evento.target?.closest?.("[data-ct-form]");
    if (!formularioDOM || !raiz.contains(formularioDOM)) return;
    evento.preventDefault();
    presentador.prepararRevision(extraerBorrador(formularioDOM));
    enfocarTrasValidacion();
  }

  function alCambiar(evento) {
    const campo = evento.target?.name;
    if (!["centro_ref", "categoria_ref", "rc_existe", "motivo_clave", "fin"].includes(campo)) return;
    const formularioDOM = evento.target.closest?.("[data-ct-form]");
    if (!formularioDOM || !raiz.contains(formularioDOM)) return;
    const borrador = extraerBorrador(formularioDOM);
    if (campo === "motivo_clave" && presentador.obtenerEstado().catalogos.motivos.find(
      ({ clave }) => clave === borrador.motivo_clave)?.fecha_fin === "no_aplica") borrador.fin = "";
    presentador.actualizarBorrador(borrador);
    const selectorFoco = campo === "rc_existe"
      ? `[name="rc_existe"][value="${borrador.rc_existe ? "si" : "no"}"]`
      : `#ct-${campo}`;
    repintar(selectorFoco);
  }

  function alSalirCampo(evento) {
    const control = evento.target;
    if (control?.name !== "numero_expediente_moad") return;
    const invalido = !numeroExpedienteMOADValido(control.value);
    const idError = "ct-numero_expediente_moad-error";
    const error = raiz.querySelector(`#${idError}`);
    if (invalido) {
      control.setAttribute?.("aria-invalid", "true");
      control.setAttribute?.("aria-describedby", idError);
      if (!error) control.insertAdjacentHTML?.("afterend", `<span class="ct-error-campo" id="${idError}">${escaparHTML(t("error_numero_moad"))}</span>`);
    } else {
      control.removeAttribute?.("aria-invalid");
      control.removeAttribute?.("aria-describedby");
      error?.remove?.();
    }
  }

  function alIntroducir(evento) {
    const campo = evento.target?.name;
    if (!["detalle", "observaciones"].includes(campo)) return;
    const contador = raiz.querySelector(`[data-ct-contador="${campo}"]`);
    if (contador) {
      contador.textContent = t("contador_caracteres", {
        actual: [...evento.target.value].length,
        maximo: LIMITES_ALTA_CONTRATACION.texto,
      });
    }
  }

  raiz.addEventListener("click", alPulsar);
  raiz.addEventListener("submit", alEnviar);
  raiz.addEventListener("change", alCambiar);
  raiz.addEventListener("input", alIntroducir);
  raiz.addEventListener("focusout", alSalirCampo);
  repintar();

  return () => {
    montada = false;
    raiz.removeEventListener("click", alPulsar);
    raiz.removeEventListener("submit", alEnviar);
    raiz.removeEventListener("change", alCambiar);
    raiz.removeEventListener("input", alIntroducir);
    raiz.removeEventListener("focusout", alSalirCampo);
    presentador.desmontar?.();
  };
}
