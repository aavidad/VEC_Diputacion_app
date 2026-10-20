import { renderizarResultadoBolsa } from "./resultado-bolsa.js?v=VEC_WEB_VERSION";
import { renderizarLineaFases } from "./vista-expedientes-ficha.js?v=VEC_WEB_VERSION";
import { consultarPersonasLlamamientos, textoPersonasLlamamiento } from "./personas-llamamiento-bolsa.js?v=VEC_WEB_VERSION";

// Controles del apartado cuyo foco se recupera al repintar.
const CONTROLES_FOCO = ["#ct-bolsa-emision", "form[data-ct-bolsa-vincular] button[type=submit]",
  "[data-ct-bolsa-mas]", "[data-ct-bolsa-anterior]", "[data-ct-bolsa-vinculo-reintentar]"];

/** Conserva el comando de un vínculo incierto en memoria; una navegación no lo sustituye. */
export function crearGestorResultadoBolsa({ raiz, presentador, cliente, t, locale, zonaHoraria,
  confirmarOperacion, repintar, generarClave = () => globalThis.crypto?.randomUUID?.(),
  consultarPersonas = consultarPersonasLlamamientos } = {}) {
  const pendientes = new Map();
  let montado = true;
  let ocupado = false;
  let paginas = null;
  let lectura = null;
  // Emisiones del último pintado: el valor del desplegable es su índice.
  let pintadas = [];
  // Nombres de las personas llamadas, por llamamiento: se piden solo los que
  // faltan, se conservan al paginar y uno que falló se vuelve a pedir después.
  const nombres = new Map();
  const pedidos = new Set();
  const lecturasNombres = new Set();
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
    cargarPersonas(paginas.resultado);
    // Repintar no debe perder lo que la persona ya eligió ni dónde estaba el foco.
    const previa = zona.querySelector("#ct-bolsa-emision");
    const elegido = previa?.value ? pintadas[Number(previa.value)]?.llamamiento_ref : null;
    const activo = raiz.ownerDocument?.activeElement;
    const foco = activo && zona.contains?.(activo) ? CONTROLES_FOCO.find((s) => activo.matches?.(s)) : null;
    pintadas = expediente.resultado_bolsa.emisiones_vinculables ?? [];
    zona.innerHTML = renderizarResultadoBolsa(expediente, t, locale, zonaHoraria,
      { personas: nombres, ocupado: ocupado || Boolean(lectura) || typeof cliente?.vincularLlamamientoBolsa !== "function" || Boolean(pendiente?.incierto),
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
    const indice = elegido ? pintadas.findIndex((e) => e.llamamiento_ref === elegido) : -1;
    const nueva = zona.querySelector("#ct-bolsa-emision");
    if (indice >= 0 && nueva) nueva.value = String(indice);
    if (foco) zona.querySelector(foco)?.focus?.();
  }
  function cargarPersonas(resultado) {
    if (typeof consultarPersonas !== "function") return;
    const faltan = [...(resultado.emisiones_vinculables ?? []), ...(resultado.vinculos ?? [])]
      .filter((l) => !nombres.has(l.llamamiento_ref) && !pedidos.has(l.llamamiento_ref));
    if (!faltan.length) return;
    const control = new AbortController();
    lecturasNombres.add(control);
    for (const l of faltan) pedidos.add(l.llamamiento_ref);
    const terminar = (mapa) => {
      lecturasNombres.delete(control);
      // Sin la lectura de Bolsa el selector sigue con referencia y fecha.
      for (const l of faltan) if (!mapa.has(l.llamamiento_ref)) pedidos.delete(l.llamamiento_ref);
      if (!montado || control.signal.aborted || !mapa.size) return;
      for (const [ref, lista] of mapa) nombres.set(ref, lista);
      actualizar();
    };
    consultarPersonas(faltan, { signal: control.signal }).then(terminar, () => terminar(new Map()));
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
    const persona = textoPersonasLlamamiento(nombres.get(emision.llamamiento_ref), t);
    const propia = emision.referencia_visible === expediente.expediente_ref;
    const referencia = propia ? t("resultado_bolsa_emision_este_expediente", { fecha }) : emision.referencia_visible;
    if (!await confirmarOperacion({ titulo: t("resultado_bolsa_vincular"),
      advertencia: propia ? t("resultado_bolsa_confirmar_este_expediente", { fecha })
        : persona ? t("resultado_bolsa_confirmar_persona", { referencia, fecha, persona })
          : t("resultado_bolsa_confirmar", { referencia, fecha }),
      referencia })) return;
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
    for (const control of lecturasNombres) control.abort();
    raiz.removeEventListener("submit", enviarFormulario);
    raiz.removeEventListener("click", reintentar);
    raiz.removeEventListener("click", paginar);
  } });
}
