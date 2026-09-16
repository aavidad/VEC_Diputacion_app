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

function nuevaIntencion(correo, version) {
  if (typeof globalThis.crypto?.randomUUID !== "function") {
    throw new Error(textoContactoPropio("intencionNoDisponible"));
  }
  const referencia = globalThis.crypto.randomUUID();
  if (typeof referencia !== "string" || !/^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/iu.test(referencia)) {
    throw new Error(textoContactoPropio("intencionNoDisponible"));
  }
  const cuerpo = JSON.stringify({ correo, version_esperada: version, intencion_ref: referencia });
  return Object.freeze({ correo, version, referencia, cuerpo });
}

export function crearControladorContactoPropio({ autorizacionServidor = null, fetchImpl = globalThis.fetch, presentacion = false } = {}) {
  let enviando = false;
  let recibo = null;
  let versionEsperada = autorizacionServidor?.version;
  let versionIntentada = null;
  let consultando = false;
  let intencionPendiente = null;

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
    const recuperable = autorizacionServidor.guardadoRecuperable === true;
    if (intencionPendiente && correoNormalizado !== intencionPendiente.correo) {
      throw new Error(textoContactoPropio("reintentarOriginal"));
    }
    if (recuperable && !intencionPendiente) intencionPendiente = nuevaIntencion(correoNormalizado, versionEsperada);
    const intencion = intencionPendiente;
    const versionParaEnvio = intencion?.version ?? versionEsperada;
    const cuerpo = intencion?.cuerpo ?? JSON.stringify({ correo: correoNormalizado, version_esperada: versionParaEnvio });
    enviando = true;
    try {
      versionIntentada = versionParaEnvio + 1;
      const respuesta = await fetchImpl(RUTA_CONTACTO_PROPIO, {
        method: "POST",
        credentials: "omit",
        cache: "no-store",
        redirect: "error",
        referrerPolicy: "no-referrer",
        headers: { Accept: "application/json", "Content-Type": "application/json" },
        body: cuerpo,
      });
      const estadosCorrectos = recuperable ? [200, 201] : [201];
      if (!estadosCorrectos.includes(respuesta?.status)) {
        intencionPendiente = null;
        versionIntentada = null;
        throw new Error(mensajeError(respuesta?.status));
      }
      const resultado = validarRespuesta(await respuesta.json(), versionParaEnvio);
      recibo = resultado;
      versionEsperada = resultado.version;
      versionIntentada = null;
      intencionPendiente = null;
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
    }
  }

  async function consultarRecibo() {
    if (presentacion === true || !esContextoAutorizado(autorizacionServidor)
      || autorizacionServidor.consultarRecibo !== true || versionIntentada === null) {
      throw new Error(textoContactoPropio("consultaNoDisponible"));
    }
    if (enviando || consultando) throw new Error(textoContactoPropio("consultando"));
    const version = versionIntentada;
    consultando = true;
    try {
      const respuesta = await fetchImpl(RUTA_RECIBO_CONTACTO_PROPIO, {
        method: "POST", credentials: "omit", cache: "no-store", redirect: "error",
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
    }
  }

  return Object.freeze({
    autorizado: presentacion !== true && esContextoAutorizado(autorizacionServidor),
    guardar,
    consultarRecibo,
    get puedeReintentarOriginal() { return intencionPendiente !== null; },
    get correoPendiente() { return intencionPendiente?.correo ?? ""; },
    get puedeConsultarRecibo() {
      return presentacion !== true && esContextoAutorizado(autorizacionServidor)
        && autorizacionServidor.consultarRecibo === true && versionIntentada !== null;
    },
    get consultando() { return consultando; },
    get enviando() { return enviando; },
    get recibo() { return recibo; },
  });
}

export function montarContactoPropio({
  contenedor, correo = "", autorizacionServidor = null, fetchImpl,
  presentacion = false, reciboAnterior = null, alGuardar = null,
} = {}) {
  if (!contenedor || typeof contenedor.replaceChildren !== "function") return null;
  const controlador = crearControladorContactoPropio({ autorizacionServidor, fetchImpl, presentacion });
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
  entrada.value = correo;
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
  boton.textContent = textoContactoPropio("guardar");
  const consultar = documento.createElement("button");
  consultar.type = "button";
  consultar.className = "boton-secundario";
  consultar.textContent = textoContactoPropio("consultarRecibo");
  consultar.hidden = true;
  const reintentar = documento.createElement("button");
  reintentar.type = "button";
  reintentar.className = "boton-secundario";
  reintentar.textContent = textoContactoPropio("botonReintentarOriginal");
  reintentar.hidden = true;
  formulario.append(campo, boton, reintentar, consultar, estado);
  const habilitado = controlador.autorizado && presentacion !== true;
  if (!habilitado) {
    entrada.disabled = true;
    boton.disabled = true;
    boton.setAttribute("aria-disabled", "true");
    estado.hidden = false;
    estado.className = "nota aviso";
    estado.textContent = textoContactoPropio("sinAutorizacion");
  } else if (reciboAnterior?.reciboRef) {
    estado.hidden = false;
    estado.textContent = textoContactoPropio("correctoAnterior", { recibo: reciboAnterior.reciboRef });
  }
  formulario.addEventListener("submit", async (evento) => {
    evento.preventDefault();
    if (!entrada.checkValidity()) {
      estado.hidden = false;
      estado.className = "nota error";
      estado.textContent = textoContactoPropio("errorEntrada");
      entrada.focus();
      return;
    }
    if (controlador.enviando || controlador.consultando) return;
    const correoEnviado = capturarCorreoEnviado(entrada);
    boton.disabled = true;
    entrada.disabled = true;
    consultar.disabled = true;
    reintentar.disabled = true;
    estado.hidden = false;
    estado.className = "nota";
    estado.textContent = textoContactoPropio("preparando");
    try {
      const resultado = await controlador.guardar(correoEnviado);
      estado.className = "nota";
      estado.textContent = textoContactoPropio("correcto", { recibo: resultado.reciboRef });
      alGuardar?.({ ...resultado, correo: correoEnviado });
    } catch (error) {
      estado.className = "nota error";
      estado.textContent = error instanceof Error ? error.message : textoContactoPropio("errorServicio");
    } finally {
      boton.disabled = !habilitado;
      entrada.disabled = !habilitado;
      consultar.hidden = !controlador.puedeConsultarRecibo;
      consultar.disabled = false;
      reintentar.hidden = !controlador.puedeReintentarOriginal;
      reintentar.disabled = false;
    }
  });
  reintentar.addEventListener("click", async () => {
    if (controlador.enviando || controlador.consultando || !controlador.puedeReintentarOriginal) return;
    entrada.value = controlador.correoPendiente;
    formulario.requestSubmit();
  });
  consultar.addEventListener("click", async () => {
    if (controlador.enviando || controlador.consultando) return;
    consultar.disabled = true;
    boton.disabled = true;
    entrada.disabled = true;
    estado.hidden = false;
    estado.className = "nota";
    estado.textContent = textoContactoPropio("consultando");
    try {
      const resultado = await controlador.consultarRecibo();
      estado.textContent = textoContactoPropio("reciboConsultado", {
        recibo: resultado.reciboRef, version: resultado.version,
      });
    } catch {
      estado.className = "nota aviso";
      estado.textContent = textoContactoPropio("consultaSinConfirmacion");
    } finally {
      consultar.disabled = false;
      boton.disabled = !habilitado;
      entrada.disabled = !habilitado;
    }
  });
  contenedor.replaceChildren(formulario);
  return controlador;
}
