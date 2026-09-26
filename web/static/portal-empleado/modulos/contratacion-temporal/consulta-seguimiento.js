import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import { RUTA_SEGUIMIENTO_INCORPORACION } from "./cliente-http-seguimiento-incorporacion.js";
import { validarReferenciaExpedienteSeguimiento } from "./contrato-seguimiento-incorporacion.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";
import { renderizarConsultaSeguimientoIncorporacion } from "./seguimiento-incorporacion.js";

// Este listener usa certificado TLS personal. El navegador presenta el
// certificado aunque fetch omita las credenciales web de ambiente.
export function crearClienteConsultaSeguimientoInterno(fetchImpl = globalThis.fetch) {
  if (typeof fetchImpl !== "function") throw new TypeError("transporte de consulta no disponible");
  const fetchSinCredencialesWeb = (ruta, opciones) => {
    if (typeof ruta !== "string" || !ruta.startsWith(`${RUTA_SEGUIMIENTO_INCORPORACION}?expediente_ref=`)
      || ruta.includes("&") || opciones?.method !== "GET") {
      throw new TypeError("ruta de consulta interna no permitida");
    }
    const cabeceras = new Headers(opciones.headers);
    if ([...cabeceras.keys()].some((nombre) => nombre !== "accept")) {
      throw new TypeError("cabeceras de consulta interna no permitidas");
    }
    return fetchImpl(ruta, { ...opciones, credentials: "omit" });
  };
  return crearClienteHTTPContratacionTemporal({ fetchImpl: fetchSinCredencialesWeb }).seguimientoIncorporacion;
}

export function montarConsultaSeguimientoInterno({ documento, cliente, mensajes = {} } = {}) {
  const formulario = documento?.querySelector?.("[data-ct-consulta-form]");
  const entrada = documento?.querySelector?.("#ct-consulta-expediente");
  const estado = documento?.querySelector?.("[data-ct-consulta-estado]");
  const panel = documento?.querySelector?.("[data-ct-consulta-resultado-panel]");
  const resultado = documento?.querySelector?.("[data-ct-consulta-resultado]");
  if (!formulario || !entrada || !estado || !panel || !resultado
    || typeof cliente?.consultar !== "function") throw new TypeError("consulta interna no disponible");
  const t = crearTraductorContratacionTemporal(mensajes);
  for (const nodo of documento.querySelectorAll("[data-ct-copia]")) {
    nodo.textContent = t(nodo.getAttribute("data-ct-copia"));
  }
  for (const nodo of documento.querySelectorAll("[data-ct-copia-alt]")) {
    nodo.setAttribute("alt", t(nodo.getAttribute("data-ct-copia-alt")));
  }
  documento.title = t("consulta_seguimiento_pagina_titulo");
  let controlador = null, secuencia = 0, montado = true;
  const limpiar = (clave) => {
    resultado.replaceChildren();
    panel.hidden = true;
    estado.textContent = t(clave);
  };
  const cancelar = () => {
    ++secuencia;
    controlador?.abort();
    controlador = null;
  };
  const alEntrada = () => {
    cancelar();
    limpiar("consulta_seguimiento_sin_seleccion");
  };
  async function alEnviar(evento) {
    evento.preventDefault();
    cancelar();
    let expedienteRef;
    try {
      expedienteRef = validarReferenciaExpedienteSeguimiento(entrada.value.trim());
    } catch {
      limpiar("consulta_seguimiento_referencia_invalida");
      entrada.focus();
      return;
    }
    const actual = new AbortController();
    controlador = actual;
    const turno = secuencia;
    limpiar("consulta_seguimiento_cargando");
    try {
      const vista = await cliente.consultar(expedienteRef, { signal: actual.signal });
      if (!montado || actual.signal.aborted || secuencia !== turno) return;
      const contenido = renderizarConsultaSeguimientoIncorporacion(vista, expedienteRef, { mensajes });
      resultado.innerHTML = contenido;
      panel.hidden = false;
      estado.textContent = "";
    } catch (error) {
      if (!montado || actual.signal.aborted || secuencia !== turno) return;
      limpiar(error?.estado === 503
        ? "consulta_seguimiento_no_disponible"
        : "consulta_seguimiento_sin_datos");
    } finally {
      if (controlador === actual) controlador = null;
    }
  }
  formulario.addEventListener("submit", alEnviar);
  entrada.addEventListener("input", alEntrada);
  limpiar("consulta_seguimiento_sin_seleccion");
  return () => {
    montado = false;
    cancelar();
    formulario.removeEventListener("submit", alEnviar);
    entrada.removeEventListener("input", alEntrada);
    limpiar("consulta_seguimiento_sin_seleccion");
  };
}

if (globalThis.document) {
  montarConsultaSeguimientoInterno({
    documento: globalThis.document,
    cliente: crearClienteConsultaSeguimientoInterno(),
  });
}
