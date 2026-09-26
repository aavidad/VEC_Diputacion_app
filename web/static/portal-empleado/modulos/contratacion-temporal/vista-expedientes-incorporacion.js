/** Montaje y refresco de resolución de formalización e incorporación al ejercicio. */

import { CODIGO_CIERRE_SIN_CESE_NO_CONTEMPLADO } from "./cliente-http-transporte.js";
import { escaparHTML } from "./componentes-expedientes.js";
import { montarFichaGINPIX } from "./ficha-ginpix.js";
import { montarFormularioAnotacionAdministrativa } from "./formulario-anotacion-administrativa.js";
import { montarFormularioCierreAdministrativo } from "./formulario-cierre-administrativo.js";
import { montarFormularioIncorporacionEjercicio } from "./formulario-incorporacion-ejercicio.js";
import { montarFormularioResolucionFormalizacion } from "./formulario-resolucion-formalizacion.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";
import { montarSeguimientoIncorporacion } from "./seguimiento-incorporacion.js";

export function crearGestorIncorporacion({
  raiz,
  presentador,
  clienteLlamamiento,
  resolucionFormalizacionDisponible,
  incorporacionEjercicioDisponible,
  confirmarOperacion,
  mensajes = {},
  locale = "es-ES",
  zonaHoraria = "Europe/Madrid",
  anunciar = () => {},
  repintar = () => {},
  entornoDescarga = globalThis,
  esMontada = () => true,
} = {}) {
  let desmontarIncorporacionEjercicio = null;
  let consultaIncorporacionEjercicio = null;
  let desmontarResolucionFormalizacion = null;
  let consultaResolucionFormalizacion = null;

  async function montarResolucionFormalizacion() {
    const estado = presentador.obtenerEstado();
    if (!esMontada() || !resolucionFormalizacionDisponible || desmontarResolucionFormalizacion || consultaResolucionFormalizacion
      || estado.carga !== "listo" || estado.expediente?.demostracion !== false
      || estado?.vista !== "expediente" || ![7, 8].includes(estado?.expediente?.version)) return;
    const resumen = estado.cuadro?.expedientes?.find(
      ({ expediente_ref: referencia }) => referencia === estado.expediente.expediente_ref,
    );
    if (resumen?.fase_clave !== "nombramiento"
      || resumen.version !== estado.expediente.version) return;
    const contenedor = raiz.querySelector("[data-ct-exp-resolucion-formalizacion]");
    if (!contenedor) return;
    const expedienteRef = estado.expediente.expediente_ref;
    const version = estado.expediente.version;
    const controlador = new AbortController();
    consultaResolucionFormalizacion = controlador;
    const vigente = () => {
      const actual = presentador.obtenerEstado();
      return esMontada() && consultaResolucionFormalizacion === controlador
        && !controlador.signal.aborted && raiz.contains?.(contenedor)
        && raiz.querySelector("[data-ct-exp-resolucion-formalizacion]") === contenedor
        && actual?.vista === "expediente" && actual.carga === "listo"
        && actual.expediente?.expediente_ref === expedienteRef
        && actual.expediente?.version === version;
    };
    const t = crearTraductorExpedientesContratacion(mensajes);
    contenedor.innerHTML = `<p class="ct-ayuda" role="status">${escaparHTML(t("resolucion_preparacion_cargando"))}</p>`;
    try {
      const preparacion = await clienteLlamamiento.prepararResolucionFormalizacion(
        expedienteRef, { signal: controlador.signal },
      );
      if (!vigente()) return;
      const promocionAutorizada = version === 7 && preparacion.version_actual === 8
        && preparacion.recibo !== null;
      if (preparacion.expediente_ref !== expedienteRef
        || (preparacion.version_actual !== version && !promocionAutorizada)) throw new TypeError("preparación no ligada");
      desmontarResolucionFormalizacion = montarFormularioResolucionFormalizacion({
        raiz: contenedor, cliente: clienteLlamamiento, confirmarOperacion, mensajes, locale, zonaHoraria,
        preparacion,
      });
    } catch (error) {
      if (!vigente()) return;
      const denegada = error?.envelopeValido === true && [401, 403].includes(error.estado);
      contenedor.innerHTML = `<p class="ct-estado ct-estado-aviso" role="status">${escaparHTML(t(denegada
        ? "resolucion_preparacion_denegada" : "resolucion_preparacion_no_disponible"))}</p>
        ${denegada ? "" : `<button class="boton-secundario" type="button" data-ct-exp-accion="reintentar-resolucion">${escaparHTML(t("resolucion_preparacion_reintentar"))}</button>`}`;
      desmontarResolucionFormalizacion = null;
    } finally {
      if (consultaResolucionFormalizacion === controlador) consultaResolucionFormalizacion = null;
    }
  }

  // Expediente cuya incorporación ya se ha pedido: los repintados posteriores
  // del mismo expediente (también tras confirmarla) la vuelven a consultar.
  let incorporacionSolicitada = "";
  const claveIncorporacion = (expediente) => expediente?.expediente_ref || "";

  function incorporacionProcede(estado) {
    return esMontada() && incorporacionEjercicioDisponible
      && estado.carga === "listo" && estado.expediente?.demostracion === false
      && estado.vista === "expediente" && Number.isSafeInteger(estado.expediente?.version)
      && estado.expediente.version >= 8;
  }

  /**
   * Al abrir el expediente la incorporación NO se consulta: la mayoría de los
   * expedientes en nombramiento no tienen preparación sellada y el servidor
   * respondería 409 («preparacion_pendiente»). Se ofrece un botón y se consulta
   * al pedirla; una vez pedida para este detalle, los repintados la consultan.
   */
  async function ofrecerIncorporacionEjercicio() {
    const estado = presentador.obtenerEstado();
    if (!incorporacionProcede(estado) || desmontarIncorporacionEjercicio || consultaIncorporacionEjercicio) return;
    if (incorporacionSolicitada === claveIncorporacion(estado.expediente)) {
      await montarIncorporacionEjercicio();
      return;
    }
    const contenedor = raiz.querySelector("[data-ct-exp-incorporacion-ejercicio]");
    if (!contenedor) return;
    const t = crearTraductorExpedientesContratacion(mensajes);
    contenedor.innerHTML = `<button class="boton-secundario" type="button" data-ct-exp-accion="consultar-incorporacion">${escaparHTML(t("incorporacion_consultar"))}</button>`;
  }

  async function montarIncorporacionEjercicio() {
    const estado = presentador.obtenerEstado();
    if (!incorporacionProcede(estado) || desmontarIncorporacionEjercicio || consultaIncorporacionEjercicio) return;
    const contenedor = raiz.querySelector("[data-ct-exp-incorporacion-ejercicio]");
    if (!contenedor) return;
    const expedienteRef = estado.expediente.expediente_ref;
    const version = estado.expediente.version;
    incorporacionSolicitada = claveIncorporacion(estado.expediente);
    const controlador = new AbortController();
    consultaIncorporacionEjercicio = controlador;
    const vigente = () => {
      const actual = presentador.obtenerEstado();
      return esMontada() && consultaIncorporacionEjercicio === controlador && !controlador.signal.aborted
        && raiz.contains?.(contenedor) && raiz.querySelector("[data-ct-exp-incorporacion-ejercicio]") === contenedor
        && actual?.vista === "expediente" && actual.carga === "listo"
        && actual.expediente?.expediente_ref === expedienteRef && actual.expediente?.version === version;
    };
    const t = crearTraductorExpedientesContratacion(mensajes);
    const tCT = crearTraductorContratacionTemporal(mensajes);
    contenedor.innerHTML = `<p class="ct-ayuda" role="status">${escaparHTML(t("incorporacion_preparacion_cargando"))}</p>`;
    try {
      const preparacion = await clienteLlamamiento.prepararIncorporacionEjercicio(expedienteRef, { signal: controlador.signal });
      if (!vigente()) return;
      if (preparacion.expediente_ref !== expedienteRef) {
        throw new TypeError("preparación de incorporación no ligada al detalle actual");
      }
      if (preparacion.version_actual_expediente !== version) {
        if (preparacion.recibo !== null && preparacion.recibo.expediente_ref === expedienteRef
          && preparacion.version_actual_expediente > version) {
          // El recibo acredita una versión posterior; no se presenta el detalle
          // v8 como vigente ni se monta una acción sobre esa proyección.
          await presentador.cargar();
          if (!esMontada() || controlador.signal.aborted) return;
          const resumen = presentador.obtenerEstado().cuadro?.expedientes?.find(
            ({ expediente_ref: referencia }) => referencia === expedienteRef,
          );
          if (!resumen || resumen.version < preparacion.version_actual_expediente) {
            contenedor.innerHTML = `<p class="ct-estado ct-estado-aviso" role="status">${escaparHTML(t("incorporacion_preparacion_detalle_obsoleto", {
              version_anterior: version, version_actual: preparacion.version_actual_expediente,
            }))}</p>`;
            return;
          }
          await presentador.seleccionarExpediente(expedienteRef, "expediente");
          if (!esMontada() || controlador.signal.aborted) return;
          const actualizado = presentador.obtenerEstado().expediente;
          if (actualizado?.version < preparacion.version_actual_expediente) {
            contenedor.innerHTML = `<p class="ct-estado ct-estado-aviso" role="status">${escaparHTML(t("incorporacion_preparacion_detalle_obsoleto", {
              version_anterior: version, version_actual: preparacion.version_actual_expediente,
            }))}</p>`;
            return;
          }
          repintar("[data-ct-exp-mensaje]");
          return;
        }
        throw new TypeError("preparación de incorporación no ligada al detalle actual");
      }
      desmontarIncorporacionEjercicio = montarFormularioIncorporacionEjercicio({
        raiz: contenedor, cliente: clienteLlamamiento, preparacion,
        confirmarOperacion, mensajes, locale, zonaHoraria, anunciar,
      });
      // Sólo una incorporación recuperada y validada habilita la ficha. La
      // descarga vuelve a consultar al servidor; el recibo del DOM no autoriza.
      if (preparacion.recibo !== null && typeof clienteLlamamiento.descargarFichaGINPIX === "function") {
        const documento = contenedor.ownerDocument ?? entornoDescarga.document;
        const bloque = documento.createElement("div");
        contenedor.append(bloque);
        const desmontarFormulario = desmontarIncorporacionEjercicio;
        let urlFicha = null;
        const liberarFicha = () => {
          if (urlFicha !== null) entornoDescarga.URL?.revokeObjectURL?.(urlFicha);
          urlFicha = null;
        };
        const resumenFicha = presentador.obtenerEstado().cuadro?.expedientes?.find(
          (fila) => fila.expediente_ref === expedienteRef && fila.version === version,
        );
        const desmontarFicha = montarFichaGINPIX({
          raiz: bloque, cliente: clienteLlamamiento, recibo: preparacion.recibo, mensajes,
          locale,
          resumen: resumenFicha ? { centro: resumenFicha.centro, categoria: resumenFicha.categoria } : undefined,
          descargarArchivo: (archivo, nombre) => {
            if (!esMontada() || !raiz.contains(contenedor)) return;
            const BlobImpl = entornoDescarga.Blob ?? globalThis.Blob;
            const enlace = documento.createElement("a");
            liberarFicha();
            urlFicha = entornoDescarga.URL?.createObjectURL?.(new BlobImpl([archivo.contenido], { type: "application/json" }));
            try {
              enlace.href = urlFicha; enlace.download = nombre; enlace.hidden = true;
              documento.body.append(enlace); enlace.click();
            } finally {
              enlace.remove(); setTimeout(liberarFicha, 0);
            }
          },
        });
        desmontarIncorporacionEjercicio = () => {
          desmontarFicha(); liberarFicha(); desmontarFormulario();
        };
      }
      if (preparacion.recibo !== null && preparacion.recibo.expediente_ref === expedienteRef
        && typeof clienteLlamamiento.anotacionAdministrativa?.registrar === "function"
        && typeof clienteLlamamiento.anotacionAdministrativa?.recuperar === "function") {
        const documento = contenedor.ownerDocument ?? entornoDescarga.document;
        const bloque = documento.createElement("div");
        const bloqueCierre = documento.createElement("div");
        contenedor.append(bloque);
        contenedor.append(bloqueCierre);
        let panelActivo = true;
        let versionObsoleta = null;
        let desmontarCierre = null;
        let lecturaCierreEnCurso = false;
        const panelSigueMontado = () => panelActivo && esMontada() && !controlador.signal.aborted
          && raiz.contains?.(contenedor);
        const vigenteMontaje = () => {
          const actual = presentador.obtenerEstado();
          return versionObsoleta === null && panelSigueMontado()
            && actual?.vista === "expediente" && actual.carga === "listo"
            && actual.expediente?.expediente_ref === expedienteRef
            && actual.expediente?.version === version;
        };
        const mostrarCierreNoDisponible = (mensaje, recuperable = false) => {
          if (!panelSigueMontado()) return;
          desmontarCierre?.();
          desmontarCierre = null;
          bloqueCierre.innerHTML = `<section class="ct-alta" data-ct-cierre-administrativo>
            <h3>${escaparHTML(tCT("cierre_titulo"))}</h3>
            <p class="ct-ayuda" role="status">${escaparHTML(mensaje)}</p>
            ${recuperable ? `<button type="button" class="boton-secundario" data-ct-exp-reintentar-cierre>${escaparHTML(tCT("cierre_reintentar_lectura"))}</button>` : ""}
          </section>`;
          if (recuperable) {
            const boton = bloqueCierre.querySelector?.("[data-ct-exp-reintentar-cierre]");
            boton?.addEventListener?.("click", () => refrescarCierre());
          }
        };
        async function refrescarCierre(trasConfirmacion = false) {
          if (!vigenteMontaje() || typeof clienteLlamamiento.consultarPreparacionCierreSinCese !== "function"
            || typeof clienteLlamamiento.cerrar !== "function" || lecturaCierreEnCurso) return;
          lecturaCierreEnCurso = true;
          try {
            const cierre = await clienteLlamamiento.consultarPreparacionCierreSinCese({
              expediente_ref: expedienteRef,
              seguimiento_ref: preparacion.recibo.seguimiento_ref,
            }, { signal: controlador.signal });
            if (!vigenteMontaje()) return;
            if (trasConfirmacion) return cierre;
            desmontarCierre?.();
            desmontarCierre = null;
            bloqueCierre.replaceChildren();
            desmontarCierre = montarFormularioCierreAdministrativo({
              raiz: bloqueCierre,
              cliente: clienteLlamamiento,
              preparacion: cierre.preparacion,
              estadoActual: cierre.estado_actual,
              contextoRecuperacion: {
                expediente_ref: expedienteRef,
                seguimiento_ref: preparacion.recibo.seguimiento_ref,
              },
              confirmarOperacion,
              alConfirmar: () => refrescarCierre(true),
              t: tCT,
            });
            return true;
          } catch (error) {
            // La regla de cierre (c10) no contempla el cierre sin cese: no se ofrece.
            if (error?.codigo === CODIGO_CIERRE_SIN_CESE_NO_CONTEMPLADO && error?.envelopeValido === true) {
              desmontarCierre?.();
              desmontarCierre = null;
              bloqueCierre.replaceChildren();
              return false;
            }
            if (vigenteMontaje() && !trasConfirmacion) {
              mostrarCierreNoDisponible(
                tCT("cierre_lectura_error"),
                true,
              );
            }
            return false;
          } finally {
            lecturaCierreEnCurso = false;
          }
        }
        async function refrescarDetalleTrasAnotacion(reciboAnotacion) {
          if (!panelSigueMontado() || reciboAnotacion.expediente_ref !== expedienteRef
            || reciboAnotacion.version_resultante <= version) return;
          versionObsoleta = reciboAnotacion.version_resultante;
          mostrarCierreNoDisponible(
            tCT("cierre_detalle_obsoleto", {
              version: reciboAnotacion.version_resultante,
              version_anterior: version,
            }),
          );
          try {
            await presentador.cargar();
            if (!panelSigueMontado()) return;
            const cuadro = presentador.obtenerEstado().cuadro;
            const resumen = cuadro?.expedientes?.find(({ expediente_ref: referencia }) => referencia === expedienteRef);
            if (!resumen || resumen.version < reciboAnotacion.version_resultante) return;
            await presentador.seleccionarExpediente(expedienteRef, "expediente");
            if (!panelSigueMontado()) return;
            const actualizado = presentador.obtenerEstado().expediente;
            if (actualizado?.expediente_ref !== expedienteRef
              || actualizado.version < reciboAnotacion.version_resultante) return;
            repintar("[data-ct-exp-mensaje]");
            const mensaje = raiz.querySelector("[data-ct-exp-mensaje]");
            const confirmacion = tCT("anotacion_detalle_actualizado", {
              recibo: reciboAnotacion.recibo_ref,
              version: actualizado.version,
            });
            if (mensaje) {
              mensaje.textContent = confirmacion;
              mensaje.setAttribute("role", "status");
            }
            anunciar(confirmacion, "exito");
          } catch {
            // El aviso de obsolescencia conserva el recibo; una lectura posterior puede recuperarlo.
          }
        }
        const desmontarAnotacion = montarFormularioAnotacionAdministrativa({
          raiz: bloque,
          cliente: clienteLlamamiento.anotacionAdministrativa,
          expediente: { expediente_ref: expedienteRef, version },
          confirmarOperacion,
          locale,
          zonaHoraria,
          alConfirmar: refrescarDetalleTrasAnotacion,
          t: tCT,
        });
        const desmontarAntesAnotacion = desmontarIncorporacionEjercicio;
        desmontarIncorporacionEjercicio = () => {
          panelActivo = false;
          desmontarCierre?.();
          desmontarAnotacion();
          bloqueCierre.replaceChildren();
          desmontarAntesAnotacion();
        };
        await refrescarCierre();
        if (!vigente()) return;
      }
      if (preparacion.recibo !== null && typeof clienteLlamamiento.seguimientoIncorporacion?.consultar === "function") {
        const documento = contenedor.ownerDocument ?? entornoDescarga.document;
        const bloque = documento.createElement("div");
        contenedor.append(bloque);
        const desmontarAnterior = desmontarIncorporacionEjercicio;
        const desmontarSeguimiento = montarSeguimientoIncorporacion({
          raiz: bloque, cliente: clienteLlamamiento.seguimientoIncorporacion,
          recibo: preparacion.recibo, mensajes,
        });
        desmontarIncorporacionEjercicio = () => {
          desmontarSeguimiento(); desmontarAnterior();
        };
      }
    } catch (error) {
      if (!vigente()) return;
      const denegada = error?.envelopeValido === true && [401, 403].includes(error.estado);
      const pendiente = error?.envelopeValido === true && error.estado === 409
        && error.codigo === "preparacion_pendiente";
      contenedor.innerHTML = `<p class="ct-estado ct-estado-aviso" role="status">${escaparHTML(t(denegada
        ? "incorporacion_preparacion_denegada" : pendiente
          ? "incorporacion_preparacion_pendiente" : "incorporacion_preparacion_no_disponible"))}</p>
        ${denegada ? "" : `<button class="boton-secundario" type="button" data-ct-exp-accion="reintentar-incorporacion">${escaparHTML(t(pendiente
          ? "incorporacion_preparacion_actualizar" : "incorporacion_preparacion_reintentar"))}</button>`}`;
    } finally {
      if (consultaIncorporacionEjercicio === controlador) consultaIncorporacionEjercicio = null;
    }
  }

  return Object.freeze({
    montarResolucionFormalizacion,
    montarIncorporacionEjercicio,
    ofrecerIncorporacionEjercicio,
    abortar() {
      consultaResolucionFormalizacion?.abort();
      consultaResolucionFormalizacion = null;
      consultaIncorporacionEjercicio?.abort();
      consultaIncorporacionEjercicio = null;
    },
    retirar() {
      consultaResolucionFormalizacion?.abort();
      consultaResolucionFormalizacion = null;
      consultaIncorporacionEjercicio?.abort();
      consultaIncorporacionEjercicio = null;
      if (typeof desmontarResolucionFormalizacion === "function") desmontarResolucionFormalizacion();
      desmontarResolucionFormalizacion = null;
      if (typeof desmontarIncorporacionEjercicio === "function") desmontarIncorporacionEjercicio();
      desmontarIncorporacionEjercicio = null;
    },
  });
}
