/** Montaje aislado de la superficie de fiscalización de contratación temporal. */

import { escaparHTML } from "./componentes-expedientes.js";
import { montarFormularioFiscalizacion } from "./formulario-fiscalizacion.js";
import { montarFormularioLlamamiento } from "./formulario-llamamiento.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";
import { PATRON_REFERENCIA } from "./vista-expedientes-analisis.js";

export function montarModuloFiscalizacionContratacionTemporal({
  raiz,
  cliente,
  mensajes = {},
  anunciar = () => {},
  confirmarOperacion = () => false,
  locale = "es-ES",
  zonaHoraria = "Europe/Madrid",
} = {}) {
  if (!raiz || typeof raiz.addEventListener !== "function"
    || typeof raiz.querySelector !== "function"
    || typeof cliente?.registrarResultadoFiscalizacion !== "function"
    || typeof anunciar !== "function" || typeof confirmarOperacion !== "function") {
    throw new TypeError("dependencias de fiscalización no válidas");
  }
  const t = crearTraductorContratacionTemporal(mensajes);
  let montado = true;
  let desmontarFormulario = null;
  let desmontarLlamamiento = null;
  raiz.innerHTML = `<section class="ct-expedientes" data-modulo="contratacion-temporal"
    aria-labelledby="ct-fiscalizacion-acceso-titulo">
    <header class="ct-exp-cabecera">
      <p class="sobrelinea">${escaparHTML(t("fiscalizacion_acceso_area"))}</p>
      <h2 id="ct-fiscalizacion-acceso-titulo">${escaparHTML(t("fiscalizacion_acceso_titulo"))}</h2>
      <p>${escaparHTML(t("fiscalizacion_acceso_descripcion"))}</p>
    </header>
    <form class="ct-exp-filtros" data-ct-fiscalizacion-acceso>
      <div class="ct-campo">
        <label for="ct-fiscalizacion-expediente">${escaparHTML(t("fiscalizacion_contexto_expediente"))}</label>
        <input id="ct-fiscalizacion-expediente" name="expediente_ref"
          type="text" maxlength="160" autocomplete="off" required>
      </div>
      <div class="ct-campo">
        <label for="ct-fiscalizacion-version">${escaparHTML(t("fiscalizacion_version_remitida"))}</label>
        <input id="ct-fiscalizacion-version" name="version_esperada"
          type="number" min="1" step="1" required>
      </div>
      <div class="ct-acciones">
        <button class="boton-primario" type="submit">${escaparHTML(t("fiscalizacion_acceso_abrir"))}</button>
      </div>
    </form>
  </section>`;

  function manejarEnvio(evento) {
    const formulario = evento.target?.closest?.("[data-ct-fiscalizacion-acceso]");
    if (!formulario || !raiz.contains(formulario) || !montado) return;
    evento.preventDefault();
    const campoReferencia = formulario.elements?.namedItem?.("expediente_ref");
    const referencia = String(campoReferencia?.value ?? "").trim();
    if (PATRON_REFERENCIA.test(referencia)) campoReferencia?.setCustomValidity?.("");
    if (typeof formulario.checkValidity === "function" && !formulario.checkValidity()) {
      formulario.reportValidity?.();
      return;
    }
    const version = Number(
      formulario.elements?.namedItem?.("version_esperada")?.value ?? 0,
    );
    if (!PATRON_REFERENCIA.test(referencia) || !Number.isSafeInteger(version) || version < 1) {
      campoReferencia?.setCustomValidity?.(
        t("fiscalizacion_acceso_referencia_invalida"),
      );
      formulario.reportValidity?.();
      return;
    }
    raiz.innerHTML = `<section class="ct-expedientes" data-modulo="contratacion-temporal">
      <div data-ct-exp-fiscalizacion></div>
      <div data-ct-exp-llamamiento></div>
    </section>`;
    const contenedor = raiz.querySelector("[data-ct-exp-fiscalizacion]");
    desmontarFormulario = montarFormularioFiscalizacion({
      raiz: contenedor,
      cliente,
      contexto: Object.freeze({
        expediente_ref: referencia,
        version_esperada: version,
        fase_clave: "",
        informe_ref: "",
      }),
      confirmarOperacion,
      mensajes,
      locale,
      zonaHoraria,
      anunciar,
      alConfirmar: (recibo) => {
        if (recibo.resultado === "desfavorable" || recibo.version_resultante < 6
          || desmontarLlamamiento !== null || ![
            "seleccionarLlamamiento",
            "registrarComunicacionLlamamiento",
            "registrarRespuestaRecibida",
            "resolverLlamamiento",
            "continuarLlamamiento",
          ].every((metodo) => typeof cliente[metodo] === "function")) return;
        desmontarLlamamiento = montarFormularioLlamamiento({
          raiz: raiz.querySelector("[data-ct-exp-llamamiento]"), cliente,
          contexto: { expediente_ref: recibo.expediente_ref,
            version_esperada: recibo.version_resultante },
          confirmarOperacion, mensajes, locale, zonaHoraria, anunciar,
        });
      },
    });
  }

  function manejarEdicion(evento) {
    const campo = evento.target;
    if (!montado || campo?.name !== "expediente_ref" || !raiz.contains(campo)) return;
    campo.setCustomValidity?.("");
  }

  raiz.addEventListener("input", manejarEdicion);
  raiz.addEventListener("submit", manejarEnvio);
  return Object.freeze({
    desmontar() {
      if (!montado) return;
      montado = false;
      if (typeof desmontarFormulario === "function") desmontarFormulario();
      desmontarLlamamiento?.();
      raiz.removeEventListener("input", manejarEdicion);
      raiz.removeEventListener("submit", manejarEnvio);
    },
  });
}
