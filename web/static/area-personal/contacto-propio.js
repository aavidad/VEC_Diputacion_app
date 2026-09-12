import { textoContactoPropio } from "./i18n-contacto-propio.js";

export const RUTA_CONTACTO_PROPIO = "/api/vec/usuarios/contacto-propio";

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

export function crearControladorContactoPropio({ autorizacionServidor = null, fetchImpl = globalThis.fetch, presentacion = false } = {}) {
  let enviando = false;
  let recibo = null;
  let versionEsperada = autorizacionServidor?.version;

  async function guardar(correo) {
    if (presentacion === true || !esContextoAutorizado(autorizacionServidor)) {
      throw new Error(textoContactoPropio("sinAutorizacion"));
    }
    if (typeof fetchImpl !== "function") throw new Error(textoContactoPropio("errorServicio"));
    const correoNormalizado = String(correo ?? "").trim();
    if (!correoNormalizado || correoNormalizado.length > 254) {
      throw new Error(textoContactoPropio("errorEntrada"));
    }
    if (enviando) throw new Error(textoContactoPropio("preparando"));
    enviando = true;
    try {
      const respuesta = await fetchImpl(RUTA_CONTACTO_PROPIO, {
        method: "POST",
        credentials: "omit",
        cache: "no-store",
        redirect: "error",
        referrerPolicy: "no-referrer",
        headers: { Accept: "application/json", "Content-Type": "application/json" },
        body: JSON.stringify({ correo: correoNormalizado, version_esperada: versionEsperada }),
      });
      if (respuesta?.status !== 201) throw new Error(mensajeError(respuesta?.status));
      const resultado = validarRespuesta(await respuesta.json(), versionEsperada);
      recibo = resultado;
      versionEsperada = resultado.version;
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

  return Object.freeze({
    autorizado: presentacion !== true && esContextoAutorizado(autorizacionServidor),
    guardar,
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
  formulario.append(campo, boton, estado);
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
    if (controlador.enviando) return;
    const correoEnviado = capturarCorreoEnviado(entrada);
    boton.disabled = true;
    entrada.disabled = true;
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
    }
  });
  contenedor.replaceChildren(formulario);
  return controlador;
}
