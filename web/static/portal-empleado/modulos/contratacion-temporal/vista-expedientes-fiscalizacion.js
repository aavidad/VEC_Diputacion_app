/** Montaje aislado de la superficie de fiscalización de contratación temporal. */

import { escaparHTML } from "./componentes-expedientes.js?v=20261018-f136c-informe-v4";
import { montarFormularioFiscalizacion } from "./formulario-fiscalizacion.js?v=20261011-ct-bolsa-vinculo-v1";
import { montarFormularioLlamamiento } from "./formulario-llamamiento.js?v=20261018-f136c-informe-v4";
import { crearTraductorContratacionTemporal } from "./i18n.js?v=20261008-alta-rpt-circular-v6";
import { PATRON_REFERENCIA } from "./vista-expedientes-analisis.js?v=20261002-ct-fin-modalidad-v1";

export function montarModuloFiscalizacionContratacionTemporal({
  raiz,
  cliente,
  mensajes = {},
  anunciar = () => {},
  confirmarOperacion = () => false,
  locale = "es-ES",
  zonaHoraria = "Europe/Madrid",
  expedienteRef = "",
  expedienteVersion = null,
} = {}) {
  if (!raiz || typeof raiz.addEventListener !== "function"
    || typeof raiz.querySelector !== "function"
    || typeof cliente?.registrarResultadoFiscalizacion !== "function"
    || typeof anunciar !== "function" || typeof confirmarOperacion !== "function") {
    throw new TypeError("dependencias de fiscalización no válidas");
  }
  const t = crearTraductorContratacionTemporal(mensajes);
  // Un enlace de expediente y versión rellena el acceso; si no es íntegro, se
  // deja vacío para que Intervención lo escriba.
  const precarga = typeof expedienteRef === "string" && PATRON_REFERENCIA.test(expedienteRef)
    && Number.isSafeInteger(expedienteVersion) && expedienteVersion >= 1
    ? { referencia: ` value="${escaparHTML(expedienteRef)}"`, version: ` value="${expedienteVersion}"` }
    : { referencia: "", version: "" };
  let montado = true;
  let desmontarFormulario = null;
  let desmontarLlamamiento = null;
  let consultaLlamamiento = null;
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
          type="text" maxlength="160" autocomplete="off" required${precarga.referencia}>
      </div>
      <div class="ct-campo">
        <label for="ct-fiscalizacion-version">${escaparHTML(t("fiscalizacion_version_remitida"))}</label>
        <input id="ct-fiscalizacion-version" name="version_esperada"
          type="number" min="1" step="1" required${precarga.version}>
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
      alConfirmar: async (recibo) => {
        // Solo la fiscalización que deja el expediente en fiscalización abre
        // el llamamiento; la de una modificación vuelve al nombramiento.
        if (recibo.resultado === "desfavorable" || recibo.fase_resultante !== "fiscalizacion"
          || recibo.version_resultante < 6
          || desmontarLlamamiento !== null || consultaLlamamiento !== null || ![
            "seleccionarLlamamiento",
            "registrarComunicacionLlamamiento",
            "registrarRespuestaRecibida",
            "resolverLlamamiento",
            "continuarLlamamiento",
          ].every((metodo) => typeof cliente[metodo] === "function")) return;
        const controlador = new AbortController();
        consultaLlamamiento = controlador;
        let numeroPersonas = null;
        try {
          if (typeof cliente.consultarDetalleRRHH === "function") {
            const detalle = await cliente.consultarDetalleRRHH({ expediente_ref: recibo.expediente_ref,
              version_observada: recibo.version_resultante }, { signal: controlador.signal });
            if (detalle?.resumen?.expediente_ref === recibo.expediente_ref
              && detalle.resumen.version === recibo.version_resultante
              && Number.isSafeInteger(detalle.solicitud?.numero_personas)
              && detalle.solicitud.numero_personas >= 1 && detalle.solicitud.numero_personas <= 100) {
              numeroPersonas = detalle.solicitud.numero_personas;
            }
          }
        } catch { /* Sin cantidad acreditada, el formulario bloquea un envío nuevo. */ }
        if (!montado || controlador.signal.aborted) return;
        consultaLlamamiento = null;
        desmontarLlamamiento = montarFormularioLlamamiento({
          raiz: raiz.querySelector("[data-ct-exp-llamamiento]"), cliente,
          contexto: { expediente_ref: recibo.expediente_ref,
            version_esperada: recibo.version_resultante,
            ...(numeroPersonas !== null ? { numero_personas: numeroPersonas } : {}) },
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
      consultaLlamamiento?.abort();
      if (typeof desmontarFormulario === "function") desmontarFormulario();
      desmontarLlamamiento?.();
      raiz.removeEventListener("input", manejarEdicion);
      raiz.removeEventListener("submit", manejarEnvio);
    },
  });
}
