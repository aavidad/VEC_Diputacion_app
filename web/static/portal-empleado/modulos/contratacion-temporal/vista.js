import { ESQUEMA_CATALOGOS_NECESIDADES, LIMITES_ALTA_CONTRATACION, numeroExpedienteMOADValido } from "./contrato.js?v=20261008-alta-analisis-bolsa-v4";
import { cargarMensajesNecesidadesAlta, crearTraductorContratacionTemporal } from "./i18n.js?v=20261008-alta-rpt-circular-v6";
import { cabecera, escaparHTML, extraerBorrador, filaResumen, formulario, revision } from "./alta-renderer-puro.js?v=20261008-alta-analisis-bolsa-v4";
import { justificanteTraducido } from "../../portal-justificante.js";
import { crearClienteHTTPRPTPublica } from "../personal/cliente-http-rpt-publica.js?v=20261008-alta-rpt-circular-v4";

export function seleccionarPuestoPublicadoRPT(pagina, codigo) {
  const puesto = pagina?.total === 1 && Array.isArray(pagina.items) ? pagina.items[0] : null;
  if (!puesto || puesto.codigo !== codigo || typeof puesto.denominacion !== "string"
    || puesto.denominacion.trim() === "") return null;
  if (typeof pagina.fuente?.importacion !== "string"
    || !/^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u.test(pagina.fuente.importacion)
    || !/^[a-f0-9]{64}$/u.test(pagina.fuente.huella_sha256)) {
    throw new TypeError("publicación RPT incompatible");
  }
  return Object.freeze({ codigo: puesto.codigo, denominacion: puesto.denominacion,
    rpt_catalogo_ref: pagina.fuente.importacion,
    rpt_catalogo_huella_sha256: pagina.fuente.huella_sha256 });
}

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
  puestoRPT = null,
  busquedaPuesto = "",
} = {}) {
  if (estado.catalogos.esquema === ESQUEMA_CATALOGOS_NECESIDADES
    && !Object.hasOwn(mensajes, "necesidad_leyenda")) {
    throw new TypeError("textos del alta de necesidades no preparados");
  }
  const t = crearTraductorContratacionTemporal(mensajes);
  const contenido = estado.fase === "edicion"
    ? formulario({ ...estado, puestoRPT, busquedaPuesto }, t)
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
  clienteRPT = crearClienteHTTPRPTPublica(),
  cargarTextosNecesidades = cargarMensajesNecesidadesAlta,
  refrescarCatalogosAlta = null,
} = {}) {
  if (!raiz || typeof raiz.addEventListener !== "function"
    || typeof raiz.querySelector !== "function"
    || typeof presentador?.obtenerEstado !== "function"
    || typeof anunciar !== "function" || typeof cargarTextosNecesidades !== "function"
    || refrescarCatalogosAlta !== null && typeof refrescarCatalogosAlta !== "function") {
    throw new TypeError("dependencias DOM del alta no válidas");
  }
  let mensajesMontaje = mensajes;
  let t = crearTraductorContratacionTemporal(mensajesMontaje);
  let datosListos = presentador.obtenerEstado().catalogos.esquema !== ESQUEMA_CATALOGOS_NECESIDADES;
  let montada = true;
  let consultaPuesto = null;
  let busquedaPuesto = "";
  let puestoRPT = null;

  function repintar(selectorFoco = "") {
    if (!montada || !datosListos) return;
    const estado = presentador.obtenerEstado();
    raiz.innerHTML = renderizarAltaContratacionTemporal(estado, {
      mensajes: mensajesMontaje,
      locale,
      zonaHoraria,
      puestoRPT,
      busquedaPuesto,
    });
    if (selectorFoco) enfocarVisible(raiz.querySelector(selectorFoco));
    anunciar(t(estado.mensaje_clave), estado.tipo_mensaje);
  }

  function prepararTextosNecesidades() {
    datosListos = false;
    raiz.setAttribute?.("aria-busy", "true");
    raiz.innerHTML = "";
    void Promise.resolve().then(() => cargarTextosNecesidades()).then((cargados) => {
      if (!montada) return;
      mensajesMontaje = { ...cargados, ...mensajes };
      t = crearTraductorContratacionTemporal(mensajesMontaje);
      datosListos = true;
      raiz.removeAttribute?.("aria-busy");
      repintar();
    }).catch(() => {
      if (!montada) return;
      datosListos = false;
      raiz.removeAttribute?.("aria-busy");
      raiz.innerHTML = `<section class="ct-estado ct-estado-error" role="alert"><p>${escaparHTML(t("estado_no_disponible"))}</p>
        <button type="button" class="boton-secundario" data-ct-accion="reintentar-textos">${escaparHTML(t("pc_reintentar_consulta"))}</button></section>`;
      anunciar(t("estado_no_disponible"), "error");
    });
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
        enfocarVisible(raiz.querySelector(`#ct-${["puesto_codigo", "rpt_catalogo_ref",
          "rpt_catalogo_huella_sha256"].includes(campo) ? "puesto_busqueda" : campo}`));
      }
      return;
    }
    const control = evento.target?.closest?.("[data-ct-accion]");
    if (!control || !raiz.contains(control)) return;
    evento.preventDefault();
    if (control.dataset.ctAccion === "reintentar-textos") {
      prepararTextosNecesidades();
      return;
    }
    if (!datosListos) return;
    if (control.dataset.ctAccion === "ayuda") {
      const ayuda = raiz.querySelector("[data-ct-ayuda]");
      if (ayuda) {
        ayuda.hidden = !ayuda.hidden;
        control.setAttribute("aria-expanded", String(!ayuda.hidden));
      }
      return;
    }
    if (control.dataset.ctAccion === "buscar-puesto") {
      consultaPuesto?.abort();
      const formularioAntes = raiz.querySelector("[data-ct-form]");
      const codigo = raiz.querySelector("#ct-puesto_busqueda")?.value?.trim() ?? "";
      if (formularioAntes) presentador.actualizarBorrador(extraerBorrador(formularioAntes));
      busquedaPuesto = codigo;
      puestoRPT = { mensaje: "puesto_buscando" };
      repintar();
      if (!/^[A-Z0-9][A-Z0-9-]{0,63}$/u.test(codigo)) {
        puestoRPT = { mensaje: "puesto_no_encontrado" };
        repintar("#ct-puesto_busqueda");
        return;
      }
      const controlador = new AbortController();
      consultaPuesto = controlador;
      try {
        const pagina = await clienteRPT.listar({ vista: "puestos", q: "", codigo_puesto: codigo,
          limit: 1, offset: 0 }, { signal: controlador.signal });
        if (!montada || consultaPuesto !== controlador || controlador.signal.aborted) return;
        const puesto = seleccionarPuestoPublicadoRPT(pagina, codigo);
        if (!puesto) {
          puestoRPT = { mensaje: "puesto_no_encontrado" };
        } else {
          // El par de publicación pertenece a Personal; el esquema RPT no es
          // una versión de la publicación.
          const borrador = extraerBorrador(raiz.querySelector("[data-ct-form]"));
          presentador.actualizarBorrador({ ...borrador, puesto_codigo: puesto.codigo,
            rpt_catalogo_ref: puesto.rpt_catalogo_ref,
            rpt_catalogo_huella_sha256: puesto.rpt_catalogo_huella_sha256 });
          puestoRPT = { codigo: puesto.codigo, denominacion: puesto.denominacion,
            mensaje: "puesto_seleccionado" };
        }
        repintar("#ct-puesto_busqueda");
      } catch (error) {
        if (!montada || controlador.signal.aborted) return;
        puestoRPT = { mensaje: error instanceof TypeError
          ? "puesto_publicacion_pendiente" : "puesto_consulta_error" };
        repintar("#ct-puesto_busqueda");
      } finally {
        if (consultaPuesto === controlador) consultaPuesto = null;
      }
      return;
    }
    if (control.dataset.ctAccion === "volver") {
      const primerCampoInvalido = Object.keys(presentador.obtenerEstado().errores)[0];
      presentador.volverAEdicion();
      repintar(primerCampoInvalido ? `#ct-${primerCampoInvalido}` : "#ct-centro_ref");
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
      if (estado.errores.numero_expediente_moad
        && !Object.hasOwn(mensajesMontaje, "estado_numero_moad_no_valido")) {
        prepararTextosNecesidades();
        return;
      }
      repintar(estado.fase === "recibo"
        ? "[data-ct-recibo]"
        : (estado.fase === "pendiente"
          ? "[data-ct-operacion-pendiente]"
          : "[data-ct-accion='confirmar']"));
    }
  }

  async function alEnviar(evento) {
    const formularioDOM = evento.target?.closest?.("[data-ct-form]");
    if (!formularioDOM || !raiz.contains(formularioDOM)) return;
    evento.preventDefault();
    if (presentador.obtenerEstado().ocupado) return;
    let borrador = extraerBorrador(formularioDOM);
    if (presentador.tieneRechazoNumero?.()) {
      presentador.actualizarBorrador(borrador);
      if (presentador.necesitaRefrescoCatalogos()) {
        const tarea = presentador.refrescarCatalogos(refrescarCatalogosAlta);
        repintar();
        if (!await tarea || !montada) {
          if (montada) enfocarTrasValidacion();
          return;
        }
      }
      borrador = presentador.obtenerEstado().borrador;
    }
    presentador.prepararRevision(borrador);
    enfocarTrasValidacion();
  }

  function alCambiar(evento) {
    const campo = evento.target?.name;
    if (!["centro_ref", "categoria_ref", "rc_existe", "motivo_clave", "fin"].includes(campo)) return;
    const formularioDOM = evento.target.closest?.("[data-ct-form]");
    if (!formularioDOM || !raiz.contains(formularioDOM)) return;
    const borrador = extraerBorrador(formularioDOM);
    if (campo === "motivo_clave") {
      consultaPuesto?.abort();
      puestoRPT = null;
      busquedaPuesto = "";
    }
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
    if (presentador.tieneRechazoNumero?.()) {
      const formularioDOM = control.closest?.("[data-ct-form]");
      if (formularioDOM && raiz.contains(formularioDOM)) {
        presentador.actualizarBorrador(extraerBorrador(formularioDOM));
        repintar();
        return;
      }
      const estado = presentador.obtenerEstado();
      if (control.value === estado.borrador.numero_expediente_moad
        && estado.errores.numero_expediente_moad) {
        control.setAttribute?.("aria-invalid", "true");
        control.setAttribute?.("aria-describedby", "ct-numero_expediente_moad-error");
        return;
      }
    }
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
    if (campo === "puesto_busqueda") {
      busquedaPuesto = evento.target.value;
      if (puestoRPT?.codigo && busquedaPuesto !== puestoRPT.codigo) {
        consultaPuesto?.abort();
        const formularioDOM = evento.target.closest?.("[data-ct-form]");
        if (formularioDOM && raiz.contains(formularioDOM)) {
          const borrador = extraerBorrador(formularioDOM);
          presentador.actualizarBorrador({ ...borrador, puesto_codigo: "",
            rpt_catalogo_ref: "", rpt_catalogo_huella_sha256: "" });
          for (const nombre of ["puesto_codigo", "rpt_catalogo_ref", "rpt_catalogo_huella_sha256"]) {
            const oculto = formularioDOM.querySelector(`[name="${nombre}"]`);
            if (oculto) oculto.value = "";
          }
        }
        puestoRPT = null;
      }
      return;
    }
    if (!["detalle", "observaciones"].includes(campo)) return;
    const contador = raiz.querySelector(`[data-ct-contador="${campo}"]`);
    if (contador) {
      contador.textContent = t("contador_caracteres", {
        actual: [...evento.target.value].length,
        maximo: LIMITES_ALTA_CONTRATACION.texto,
      });
    }
  }

  function alTeclado(evento) {
    if (evento.key !== "Escape") return;
    const ayuda = raiz.querySelector("[data-ct-ayuda]");
    if (!ayuda || ayuda.hidden) return;
    ayuda.hidden = true;
    const boton = raiz.querySelector('[data-ct-accion="ayuda"]');
    boton?.setAttribute("aria-expanded", "false");
    boton?.focus?.();
  }

  raiz.addEventListener("click", alPulsar);
  raiz.addEventListener("submit", alEnviar);
  raiz.addEventListener("change", alCambiar);
  raiz.addEventListener("input", alIntroducir);
  raiz.addEventListener("focusout", alSalirCampo);
  if (presentador.obtenerEstado().catalogos.esquema === ESQUEMA_CATALOGOS_NECESIDADES) {
    raiz.addEventListener("keydown", alTeclado);
  }
  if (datosListos) repintar();
  else prepararTextosNecesidades();

  return () => {
    montada = false;
    consultaPuesto?.abort();
    raiz.removeEventListener("click", alPulsar);
    raiz.removeEventListener("submit", alEnviar);
    raiz.removeEventListener("change", alCambiar);
    raiz.removeEventListener("input", alIntroducir);
    raiz.removeEventListener("focusout", alSalirCampo);
    if (presentador.obtenerEstado().catalogos.esquema === ESQUEMA_CATALOGOS_NECESIDADES) {
      raiz.removeEventListener("keydown", alTeclado);
    }
    presentador.desmontar?.();
  };
}
