import { textoContactoPropio } from "./i18n-contacto-propio.js";

export const RUTA_CONTACTO_PROPIO = "/api/vec/usuarios/contacto-propio";
export const RUTA_RECIBO_CONTACTO_PROPIO = `${RUTA_CONTACTO_PROPIO}/recibo`;

export function capturarCorreoEnviado(entrada) {
  return String(entrada?.value ?? "").trim();
}

function esContextoAutorizado(contexto) {
  return contexto !== null && typeof contexto === "object"
    && contexto.capacidad === true
    && Number.isSafeInteger(contexto.version)
    && contexto.version >= 0
    && contexto.version < Number.MAX_SAFE_INTEGER;
}

function mensajeError(estado) {
  if (estado === 400) return textoContactoPropio("errorEntrada");
  if (estado === 403) return textoContactoPropio("errorPermiso");
  return textoContactoPropio("errorServicio");
}

function validarRespuesta(respuesta, versionEsperada) {
  if (!respuesta || typeof respuesta !== "object" || Array.isArray(respuesta)
    || typeof respuesta.recibo_ref !== "string" || respuesta.recibo_ref.length === 0
    || !Number.isSafeInteger(respuesta.version) || respuesta.version !== versionEsperada + 1) {
    throw new TypeError(textoContactoPropio("errorServicio"));
  }
  return Object.freeze({ reciboRef: respuesta.recibo_ref, version: respuesta.version });
}

function crearOperacion(operacion, version) {
  if (operacion && typeof operacion === "object" && Number.isSafeInteger(operacion.versionEsperada)) return { ...operacion };
  return { versionEsperada: version, intencion: null, versionIntentada: null, recibo: null, resultado: null };
}

export function crearControladorContactoPropio({ autorizacionServidor = null, fetchImpl = globalThis.fetch, presentacion = false, operacion = null, alActualizarOperacion = null } = {}) {
  let enviando = false;
  let consultando = false;
  let estadoOperacion = crearOperacion(operacion, autorizacionServidor?.version);
  const suscriptores = new Set();
  const notificarEstado = () => {
    for (const suscriptor of suscriptores) suscriptor();
  };
  const publicarOperacion = () => {
    const instantanea = Object.freeze({ ...estadoOperacion, intencion: estadoOperacion.intencion && { ...estadoOperacion.intencion }, recibo: estadoOperacion.recibo && { ...estadoOperacion.recibo }, resultado: estadoOperacion.resultado && { ...estadoOperacion.resultado } });
    alActualizarOperacion?.(instantanea);
    notificarEstado();
  };
  const limpiarIntento = () => { estadoOperacion = { ...estadoOperacion, intencion: null, versionIntentada: null }; publicarOperacion(); };

  async function guardar(correo) {
    if (presentacion === true || !esContextoAutorizado(autorizacionServidor)) {
      throw new Error(textoContactoPropio("sinAutorizacion"));
    }
    if (typeof fetchImpl !== "function") throw new Error(textoContactoPropio("errorServicio"));
    const correoNormalizado = String(correo ?? "").trim();
    if (!correoNormalizado || correoNormalizado.length > 254) {
      throw new Error(textoContactoPropio("errorEntrada"));
    }
    if (enviando || consultando) throw new Error(textoContactoPropio("preparando"));
    if (estadoOperacion.intencion && correoNormalizado !== estadoOperacion.intencion.correo) throw new Error(textoContactoPropio("reintentoExacto"));
    enviando = true;
    try {
      const intencion = estadoOperacion.intencion ?? Object.freeze({ correo: correoNormalizado, version_esperada: estadoOperacion.versionEsperada });
      estadoOperacion = { ...estadoOperacion, intencion, versionIntentada: intencion.version_esperada + 1 };
      publicarOperacion();
      const respuesta = await fetchImpl(RUTA_CONTACTO_PROPIO, {
        method: "POST",
        credentials: "same-origin",
        cache: "no-store",
        redirect: "error",
        referrerPolicy: "no-referrer",
        headers: { Accept: "application/json", "Content-Type": "application/json" },
        body: JSON.stringify(intencion),
      });
      if (respuesta?.status !== 201 && respuesta?.status !== 200) {
        if (respuesta?.status === 400 || respuesta?.status === 403) limpiarIntento();
        throw new Error(mensajeError(respuesta?.status));
      }
      const resultado = validarRespuesta(await respuesta.json(), intencion.version_esperada);
      estadoOperacion = { ...estadoOperacion, versionEsperada: resultado.version, intencion: null, versionIntentada: null, recibo: resultado, resultado: { ...resultado, correo: intencion.correo } };
      publicarOperacion();
      return resultado;
    } catch (error) {
      if (error instanceof Error && Object.values({
        entrada: textoContactoPropio("errorEntrada"),
        permiso: textoContactoPropio("errorPermiso"),
        servicio: textoContactoPropio("errorServicio"),
      }).includes(error.message)) throw error;
      throw new Error(textoContactoPropio("errorServicio"));
    } finally {
      enviando = false;
      notificarEstado();
    }
  }

  async function consultarRecibo() {
    if (presentacion === true || !esContextoAutorizado(autorizacionServidor)
      || autorizacionServidor.consultarRecibo !== true || estadoOperacion.versionIntentada === null) {
      throw new Error(textoContactoPropio("consultaNoDisponible"));
    }
    if (enviando || consultando) throw new Error(textoContactoPropio("consultando"));
    const version = estadoOperacion.versionIntentada;
    consultando = true;
    try {
      const respuesta = await fetchImpl(RUTA_RECIBO_CONTACTO_PROPIO, {
        method: "POST", credentials: "same-origin", cache: "no-store", redirect: "error",
        referrerPolicy: "no-referrer",
        headers: { Accept: "application/json", "Content-Type": "application/json" },
        body: JSON.stringify({ version }),
      });
      if (respuesta?.status !== 200) throw new Error();
      // La consulta acredita una versión histórica, no el correo del intento.
      // No modifica recibo, versión esperada ni el estado del guardado.
      return validarRespuesta(await respuesta.json(), version - 1);
    } catch {
      throw new Error(textoContactoPropio("consultaSinConfirmacion"));
    } finally {
      consultando = false;
      notificarEstado();
    }
  }

  return Object.freeze({
    autorizado: presentacion !== true && esContextoAutorizado(autorizacionServidor),
    guardar,
    consultarRecibo,
    suscribir(suscriptor) {
      if (typeof suscriptor !== "function") return () => {};
      suscriptores.add(suscriptor);
      return () => suscriptores.delete(suscriptor);
    },
    get puedeConsultarRecibo() {
      return presentacion !== true && esContextoAutorizado(autorizacionServidor)
        && autorizacionServidor.consultarRecibo === true && estadoOperacion.versionIntentada !== null;
    },
    get consultando() { return consultando; },
    get enviando() { return enviando; },
    get recibo() { return estadoOperacion.recibo; },
    get intencion() { return estadoOperacion.intencion && { ...estadoOperacion.intencion }; },
    get resultado() { return estadoOperacion.resultado && { ...estadoOperacion.resultado }; },
  });
}

export function montarContactoPropio({
  contenedor, correo = "", autorizacionServidor = null, fetchImpl,
  presentacion = false, reciboAnterior = null, alGuardar = null, controlador: controladorExterno = null,
} = {}) {
  if (!contenedor || typeof contenedor.replaceChildren !== "function") return null;
  const controlador = controladorExterno ?? crearControladorContactoPropio({ autorizacionServidor, fetchImpl, presentacion });
  const documento = contenedor.ownerDocument;
  const formulario = documento.createElement("form");
  formulario.noValidate = true;
  const campo = documento.createElement("div");
  campo.className = "campo";
  const etiqueta = documento.createElement("label");
  etiqueta.htmlFor = "correo-contacto-propio";
  etiqueta.textContent = textoContactoPropio("etiquetaCorreo");
  const entrada = documento.createElement("input");
  entrada.id = "correo-contacto-propio";
  entrada.name = "correo";
  entrada.type = "email";
  entrada.autocomplete = "email";
  entrada.required = true;
  entrada.maxLength = 254;
  entrada.value = controlador.intencion?.correo ?? correo;
  const ayuda = documento.createElement("small");
  ayuda.textContent = textoContactoPropio("ayuda");
  campo.append(etiqueta, entrada, ayuda);
  const estado = documento.createElement("p");
  estado.className = "nota";
  estado.hidden = true;
  estado.setAttribute("role", "status");
  const boton = documento.createElement("button");
  boton.type = "submit";
  boton.className = "boton-primario";
  boton.textContent = controlador.intencion ? textoContactoPropio("reintentoExacto") : textoContactoPropio("guardar");
  const consultar = documento.createElement("button");
  consultar.type = "button";
  consultar.className = "boton-secundario";
  consultar.textContent = textoContactoPropio("consultarRecibo");
  consultar.hidden = true;
  const historial = documento.createElement("p");
  historial.className = "nota";
  historial.hidden = true;
  formulario.append(campo, boton, consultar, estado, historial);
  const habilitado = controlador.autorizado && presentacion !== true;
  let activa = true;
  let resultadoMostrado = controlador.resultado?.reciboRef ?? null;
  let estadoActual = "inicial";
  const mostrarEstado = (clase, texto, tipo) => {
    estado.hidden = false;
    estado.className = clase;
    estado.textContent = texto;
    estadoActual = tipo;
  };
  const mostrarHistorico = (resultado) => {
    if (!resultado?.reciboRef) return;
    historial.hidden = false;
    historial.textContent = textoContactoPropio("correctoAnterior", { recibo: resultado.reciboRef });
  };
  const sincronizar = () => {
    if (!activa) return;
    const intencion = controlador.intencion;
    if (intencion) entrada.value = intencion.correo;
    entrada.disabled = !habilitado || Boolean(intencion) || controlador.enviando || controlador.consultando;
    boton.disabled = !habilitado || controlador.enviando || controlador.consultando;
    boton.textContent = intencion ? textoContactoPropio("reintentoExacto") : textoContactoPropio("guardar");
    consultar.hidden = !controlador.puedeConsultarRecibo;
    consultar.disabled = controlador.enviando || controlador.consultando;
    if (intencion && controlador.resultado) mostrarHistorico(controlador.resultado);
    if (controlador.resultado?.reciboRef && controlador.resultado.reciboRef !== resultadoMostrado) {
      resultadoMostrado = controlador.resultado.reciboRef;
      mostrarEstado("nota", textoContactoPropio("correcto", { recibo: resultadoMostrado }), "exito");
    } else if (intencion && estadoActual === "inicial") {
      mostrarEstado("nota aviso", textoContactoPropio("errorServicio"), "incertidumbre");
    }
  };
  mostrarHistorico(controlador.resultado ?? reciboAnterior);
  if (!habilitado) {
    entrada.disabled = true;
    boton.disabled = true;
    boton.setAttribute("aria-disabled", "true");
    mostrarEstado("nota aviso", textoContactoPropio("sinAutorizacion"), "denegado");
  } else if (controlador.intencion) {
    entrada.disabled = true;
    mostrarEstado("nota aviso", textoContactoPropio("errorServicio"), "incertidumbre");
  }
  sincronizar();
  const cancelarSuscripcion = controlador.suscribir?.(sincronizar);
  const guardar = async (evento) => {
    evento.preventDefault();
    if (!entrada.checkValidity()) {
      mostrarEstado("nota error", textoContactoPropio("errorEntrada"), "error");
      entrada.focus();
      return;
    }
    if (controlador.enviando || controlador.consultando) return;
    const correoEnviado = capturarCorreoEnviado(entrada);
    boton.disabled = true;
    entrada.disabled = true;
    consultar.disabled = true;
    mostrarEstado("nota", textoContactoPropio("preparando"), "preparando");
    try {
      const resultado = await controlador.guardar(correoEnviado);
      if (!activa) return;
      mostrarEstado("nota", textoContactoPropio("correcto", { recibo: resultado.reciboRef }), "exito");
      alGuardar?.({ ...resultado, correo: correoEnviado });
    } catch (error) {
      if (!activa) return;
      mostrarEstado("nota error", error instanceof Error ? error.message : textoContactoPropio("errorServicio"), "error");
    } finally {
      if (!activa) return;
      sincronizar();
    }
  };
  const consultarRecibo = async () => {
    if (controlador.enviando || controlador.consultando) return;
    consultar.disabled = true;
    boton.disabled = true;
    entrada.disabled = true;
    mostrarEstado("nota", textoContactoPropio("consultando"), "consulta");
    try {
      const resultado = await controlador.consultarRecibo();
      if (!activa) return;
      mostrarEstado("nota", textoContactoPropio("reciboConsultado", {
        recibo: resultado.reciboRef, version: resultado.version,
      }), "consulta");
    } catch {
      if (!activa) return;
      mostrarEstado("nota aviso", textoContactoPropio("consultaSinConfirmacion"), "consulta");
    } finally {
      if (!activa) return;
      sincronizar();
    }
  };
  formulario.addEventListener("submit", guardar);
  consultar.addEventListener("click", consultarRecibo);
  contenedor.replaceChildren(formulario);
  return Object.freeze({ controlador, destruir() { activa = false; cancelarSuscripcion?.(); formulario.removeEventListener("submit", guardar); consultar.removeEventListener("click", consultarRecibo); } });
}
