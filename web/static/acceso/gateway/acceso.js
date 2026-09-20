const RUTAS = Object.freeze({
  certificado: "/api/acceso/certificado",
  sesion: "/api/acceso/sesion",
  cerrar: "/api/acceso/cerrar",
});
const TIEMPO_LIMITE_MS = 15_000;
const CATALOGO = "/acceso/gateway/locales/es.json";

export class ErrorGateway extends Error {
  constructor(codigo) {
    super(codigo);
    this.codigo = codigo;
  }
}

function esObjetoPlano(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor);
}

function clavesExactas(objeto, esperadas) {
  const claves = Object.keys(objeto).sort();
  return claves.length === esperadas.length && claves.every((clave, indice) => clave === esperadas[indice]);
}

export function validarEstadoSesion(envelope) {
  if (!esObjetoPlano(envelope) || !clavesExactas(envelope, ["data"]) || !esObjetoPlano(envelope.data)) {
    throw new ErrorGateway("respuesta_invalida");
  }
  const estado = envelope.data;
  if (!clavesExactas(estado, ["autenticada", "expira_en"]) || typeof estado.autenticada !== "boolean") {
    throw new ErrorGateway("respuesta_invalida");
  }
  if (!estado.autenticada && estado.expira_en !== null) throw new ErrorGateway("respuesta_invalida");
  if (estado.autenticada && (typeof estado.expira_en !== "string" || !Number.isFinite(Date.parse(estado.expira_en)))) {
    throw new ErrorGateway("respuesta_invalida");
  }
  return Object.freeze({ autenticada: estado.autenticada, expira_en: estado.expira_en });
}

async function leerJSON(respuesta) {
  const tipo = respuesta.headers?.get?.("Content-Type") || "";
  if (!/^application\/json(?:\s*;|$)/iu.test(tipo)) throw new ErrorGateway("respuesta_invalida");
  const texto = await respuesta.text();
  if (texto.length > 8 * 1024) throw new ErrorGateway("respuesta_invalida");
  try { return JSON.parse(texto); } catch { throw new ErrorGateway("respuesta_invalida"); }
}

async function pedir(fetchImpl, ruta, method, { signal } = {}) {
  const controlador = new AbortController();
  const temporizador = setTimeout(() => controlador.abort("tiempo_agotado"), TIEMPO_LIMITE_MS);
  const abortar = () => controlador.abort(signal?.reason || "cancelado");
  signal?.addEventListener("abort", abortar, { once: true });
  try {
    return await fetchImpl(ruta, {
      method,
      credentials: "same-origin",
      mode: "same-origin",
      cache: "no-store",
      redirect: "error",
      referrerPolicy: "no-referrer",
      headers: { Accept: "application/json" },
      signal: controlador.signal,
    });
  } catch (error) {
    if (controlador.signal.aborted) throw new ErrorGateway("indeterminado");
    throw new ErrorGateway("red");
  } finally {
    clearTimeout(temporizador);
    signal?.removeEventListener("abort", abortar);
  }
}

export function crearClienteGateway({ fetchImpl = fetch } = {}) {
  async function consultarSesion({ signal } = {}) {
    const respuesta = await pedir(fetchImpl, RUTAS.sesion, "GET", { signal });
    if (respuesta.status === 401 || respuesta.status === 403) return Object.freeze({ autenticada: false, expira_en: null });
    if (respuesta.status !== 200) throw new ErrorGateway("servicio");
    return validarEstadoSesion(await leerJSON(respuesta));
  }

  async function iniciarCertificado({ signal } = {}) {
    const respuesta = await pedir(fetchImpl, RUTAS.certificado, "POST", { signal });
    if (respuesta.status === 401 || respuesta.status === 403) throw new ErrorGateway("rechazado");
    if (respuesta.status !== 200) throw new ErrorGateway("servicio");
    return validarEstadoSesion(await leerJSON(respuesta));
  }

  async function cerrarSesion({ signal } = {}) {
    const respuesta = await pedir(fetchImpl, RUTAS.cerrar, "POST", { signal });
    if (respuesta.status !== 204) throw new ErrorGateway("servicio");
  }
  return Object.freeze({ consultarSesion, iniciarCertificado, cerrarSesion });
}

function reemplazar(texto, valores = {}) {
  return String(texto).replace(/\{([a-z_]+)\}/gu, (_, clave) => valores[clave] ?? "");
}

function formatoExpiracion(valor) {
  return new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeStyle: "short" }).format(new Date(valor));
}

async function cargarCatalogo(fetchImpl) {
  try {
    const respuesta = await fetchImpl(CATALOGO, { credentials: "omit", cache: "no-store" });
    if (!respuesta.ok) return {};
    const catalogo = await respuesta.json();
    return esObjetoPlano(catalogo) ? catalogo : {};
  } catch { return {}; }
}

function aplicarCatalogo(documento, catalogo) {
  documento.querySelectorAll("[data-i18n]").forEach((elemento) => {
    const texto = catalogo[elemento.getAttribute("data-i18n")];
    if (typeof texto === "string") elemento.textContent = texto;
  });
  documento.querySelectorAll("[data-i18n-atributo]").forEach((elemento) => {
    const [atributo, clave, extra] = String(elemento.getAttribute("data-i18n-atributo")).split(":");
    if (!extra && (atributo === "content" || atributo === "aria-label") && typeof catalogo[clave] === "string") elemento.setAttribute(atributo, catalogo[clave]);
  });
}

export async function iniciarAccesoGateway(documento = document, fetchImpl = fetch) {
  const catalogo = await cargarCatalogo(fetchImpl);
  aplicarCatalogo(documento, catalogo);
  const cliente = crearClienteGateway({ fetchImpl });
  const estado = documento.getElementById("estado-acceso");
  const titulo = documento.getElementById("estado-titulo");
  const descripcion = documento.getElementById("estado-descripcion");
  const certificado = documento.getElementById("boton-certificado");
  const cerrar = documento.getElementById("boton-cerrar");
  let enCurso = null;

  function texto(clave, valores) { return reemplazar(catalogo[clave] || clave, valores); }
  function presentar(nombre, opciones = {}) {
    estado.dataset.estado = nombre;
    titulo.textContent = texto(`gateway.estado.${nombre}.titulo`);
    descripcion.textContent = texto(`gateway.estado.${nombre}.descripcion`, opciones);
    certificado.disabled = opciones.ocupado === true || nombre === "iniciada";
    cerrar.hidden = nombre !== "iniciada";
  }
  function focoEstado() { estado.focus({ preventScroll: true }); }
  async function ejecutar(operacion) {
    enCurso?.abort("sustituida");
    enCurso = new AbortController();
    try { return await operacion(enCurso.signal); } finally { enCurso = null; }
  }
  async function refrescar() {
    presentar("comprobando", { ocupado: true });
    try {
      const sesion = await ejecutar((signal) => cliente.consultarSesion({ signal }));
      if (sesion.autenticada) presentar("iniciada", { expiracion: formatoExpiracion(sesion.expira_en) });
      else presentar("necesaria");
    } catch { presentar("error"); }
  }
  certificado.addEventListener("click", async () => {
    presentar("iniciando", { ocupado: true });
    try {
      const sesion = await ejecutar((signal) => cliente.iniciarCertificado({ signal }));
      if (!sesion.autenticada) throw new ErrorGateway("rechazado");
      presentar("iniciada", { expiracion: formatoExpiracion(sesion.expira_en) });
    } catch (error) { presentar(error?.codigo === "rechazado" ? "rechazado" : "error"); }
    focoEstado();
  });
  cerrar.addEventListener("click", async () => {
    presentar("cerrando", { ocupado: true });
    try { await ejecutar((signal) => cliente.cerrarSesion({ signal })); presentar("cerrada"); }
    catch { presentar("error"); }
    focoEstado();
  });
  await refrescar();
  return Object.freeze({ refrescar });
}

if (typeof document !== "undefined" && typeof fetch === "function") void iniciarAccesoGateway();
