import { renderizarResultadoBolsa } from "./resultado-bolsa.js?v=20261010-ct-bolsa-cohorte-v10";
import { renderizarLineaFases } from "./vista-expedientes-ficha.js?v=20261010-ct-bolsa-cohorte-v10";

/** Conserva el comando de un vínculo incierto en memoria; una navegación no lo sustituye. */
export function crearGestorResultadoBolsa({ raiz, presentador, cliente, t, locale, zonaHoraria,
  confirmarOperacion, repintar, generarClave = () => globalThis.crypto?.randomUUID?.() } = {}) {
  const pendientes = new Map();
  let montado = true;
  let ocupado = false;
  let paginas = null;
  let lectura = null;
  function expedienteActual() {
    const estado = presentador.obtenerEstado();
    return estado.vista === "expediente" && estado.carga === "listo" && !estado.ocupado
      && !estado.actualizacion_pendiente && !estado.resultado_indeterminado
      && estado.expediente?.expediente_ref === estado.expediente_ref ? estado.expediente : null;
  }
  function actualizar() {
    const expediente = expedienteActual();
    if (!expediente || (paginas && (paginas.expediente_ref !== expediente.expediente_ref
      || paginas.base !== expediente.resultado_bolsa))) {
      lectura?.abort(); lectura = null; paginas = null;
    }
    const zona = raiz.querySelector("[data-ct-resultado-bolsa-zona]");
    if (!zona || !expediente?.resultado_bolsa) return;
    if (!paginas) paginas = { expediente_ref: expediente.expediente_ref, base: expediente.resultado_bolsa,
      resultado: expediente.resultado_bolsa, cursor: null, anteriores: [] };
    const pendiente = pendientes.get(expediente.expediente_ref);
    zona.innerHTML = renderizarResultadoBolsa(expediente, t, locale, zonaHoraria,
      { ocupado: ocupado || Boolean(lectura) || typeof cliente?.vincularLlamamientoBolsa !== "function" || Boolean(pendiente?.incierto),
        mensaje: pendiente?.mensaje ?? paginas.mensaje,
        vinculos: paginas.resultado.vinculos, siguiente_cursor: paginas.resultado.siguiente_cursor,
        anterior: paginas.anteriores.length > 0 });
    if (pendiente?.incierto && !ocupado) {
      const boton = raiz.ownerDocument.createElement("button");
      boton.type = "button"; boton.className = "boton-secundario";
      boton.setAttribute("data-ct-bolsa-vinculo-reintentar", "");
      boton.textContent = t("resultado_bolsa_reintentar");
      zona.querySelector("[data-ct-bolsa-estado]")?.after(boton);
    }
  }
  async function enviar(expediente, pendiente) {
    if (ocupado || !montado || typeof cliente?.vincularLlamamientoBolsa !== "function") return;
    ocupado = true; pendiente.mensaje = "resultado_bolsa_guardando"; actualizar();
    try {
      await cliente.vincularLlamamientoBolsa(pendiente.solicitud);
      pendiente.incierto = false;
      pendiente.mensaje = "resultado_bolsa_actualizacion";
      if (montado && expedienteActual()?.expediente_ref === expediente.expediente_ref) {
        await presentador.seleccionarExpediente(expediente.expediente_ref, "expediente");
        if (montado && expedienteActual()?.expediente_ref === expediente.expediente_ref) {
          pendientes.delete(expediente.expediente_ref);
          repintar("#ct-resultado-bolsa-titulo");
        }
      }
    } catch (error) {
      const determinado = error?.resultadoIndeterminado === false || error instanceof TypeError;
      pendiente.incierto = !determinado;
      pendiente.mensaje = error?.estado === 401 || error?.estado === 403 ? "resultado_bolsa_denegado"
        : error?.estado === 409 && determinado ? "resultado_bolsa_conflicto" : "resultado_bolsa_error";
    } finally {
      ocupado = false;
      if (montado) actualizar();
    }
  }
  async function enviarFormulario(evento) {
    const formulario = evento.target?.closest?.("[data-ct-bolsa-vincular]");
    if (!formulario || !raiz.contains(formulario)) return;
    evento.preventDefault();
    const expediente = expedienteActual();
    if (!expediente || ocupado || pendientes.get(expediente.expediente_ref)?.incierto) return;
    const valor = formulario.querySelector('[name="emision"]')?.value;
    if (!/^(0|[1-9][0-9]*)$/u.test(valor ?? "")) return;
    const emision = expediente.resultado_bolsa?.emisiones_vinculables?.[Number(valor)];
    if (!emision) return;
    const fecha = new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short", timeZone: zonaHoraria }).format(new Date(emision.emitido_en));
    if (!await confirmarOperacion({ titulo: t("resultado_bolsa_vincular"),
      advertencia: t("resultado_bolsa_confirmar", { referencia: emision.referencia_visible, fecha }),
      referencia: emision.referencia_visible })) return;
    // La confirmación puede ser asíncrona: no enviar sobre otra ficha o versión.
    const actual = expedienteActual();
    if (!montado || actual?.expediente_ref !== expediente.expediente_ref || actual.version !== expediente.version || ocupado) return;
    const pendiente = { incierto: false, solicitud: Object.freeze({ expediente_ref: expediente.expediente_ref,
      version_esperada: expediente.version, bolsa_ref: emision.bolsa_ref,
      llamamiento_ref: emision.llamamiento_ref, recibo_emision_ref: emision.recibo_emision_ref,
      clave_idempotencia: generarClave() }) };
    pendientes.set(expediente.expediente_ref, pendiente);
    await enviar(expediente, pendiente);
  }
  function reintentar(evento) {
    const boton = evento.target?.closest?.("[data-ct-bolsa-vinculo-reintentar]");
    if (!boton || !raiz.contains(boton)) return;
    const expediente = expedienteActual();
    const pendiente = expediente && pendientes.get(expediente.expediente_ref);
    if (pendiente?.incierto) { evento.preventDefault(); void enviar(expediente, pendiente); }
  }
  async function paginar(evento) {
    const boton = evento.target?.closest?.("[data-ct-bolsa-mas], [data-ct-bolsa-anterior]");
    if (!boton || !raiz.contains(boton)) return;
    evento.preventDefault();
    const expediente = expedienteActual();
    if (!expediente || !paginas || ocupado || lectura || typeof cliente?.consultarDetalleRRHH !== "function") return;
    const anterior = boton.hasAttribute("data-ct-bolsa-anterior");
    const pagina = paginas;
    const cursor = anterior ? pagina.anteriores.at(-1) : pagina.resultado.siguiente_cursor;
    if ((anterior && !pagina.anteriores.length) || (!anterior && !cursor)) return;
    const control = new AbortController(); lectura = control; actualizar();
    try {
      const detalle = await cliente.consultarDetalleRRHH({ expediente_ref: expediente.expediente_ref,
        version_observada: expediente.version, ...(cursor ? { resultado_bolsa_cursor: cursor } : {}) },
      { signal: control.signal });
      if (control.signal.aborted || !montado || paginas !== pagina
        || expedienteActual()?.expediente_ref !== expediente.expediente_ref) return;
      if (detalle.resumen?.expediente_ref !== expediente.expediente_ref
        || detalle.resumen.version !== expediente.version || !detalle.resultado_bolsa) throw new TypeError("pagina_bolsa_ajena");
      if (anterior) pagina.anteriores.pop(); else pagina.anteriores.push(pagina.cursor);
      pagina.cursor = cursor; pagina.resultado = detalle.resultado_bolsa; pagina.mensaje = "";
      const rail = raiz.querySelector(".ct-exp-fases");
      if (rail) rail.outerHTML = renderizarLineaFases({ ...expediente, resultado_bolsa: detalle.resultado_bolsa }, t);
    } catch (error) {
      if (!control.signal.aborted && montado && paginas === pagina) {
        pagina.mensaje = error?.estado === 401 || error?.estado === 403 ? "resultado_bolsa_denegado" : "resultado_bolsa_error_lectura";
      }
    } finally {
      if (lectura === control) lectura = null;
      if (montado) actualizar();
    }
  }
  raiz.addEventListener("submit", enviarFormulario);
  raiz.addEventListener("click", reintentar);
  raiz.addEventListener("click", paginar);
  return Object.freeze({ actualizar, desmontar() {
    montado = false;
    lectura?.abort(); lectura = null;
    raiz.removeEventListener("submit", enviarFormulario);
    raiz.removeEventListener("click", reintentar);
    raiz.removeEventListener("click", paginar);
  } });
}
