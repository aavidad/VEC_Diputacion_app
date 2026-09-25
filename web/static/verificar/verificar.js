import { traducirVerificar } from "./i18n.js?v=20260924-cotejo-espera-i18n-v1";

const RUTA_COTEJO_PUBLICO = "/api/publico/documentos/cotejo";
const formulario = document.getElementById("formulario-cotejo");
const entrada = document.getElementById("referencia");
const resultado = document.getElementById("resultado-cotejo");
const TIEMPO_MAX_COTEJO_MS = 12000;
let intentoActual = 0;
let controladorActual;

function parametrosCerrados() {
  const parametros = new URLSearchParams(window.location.search);
  if ([...parametros.keys()].some((clave) => clave !== "ref")) return "";
  const referencias = parametros.getAll("ref");
  return referencias.length === 1 ? referencias[0] : "";
}

function referenciaValida(valor) {
  return typeof valor === "string" && /^[A-Za-z0-9][A-Za-z0-9._:-]{7,79}$/.test(valor);
}

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");
}

function pintar(respuesta) {
  const valido = respuesta?.valido === true;
  resultado.hidden = false;
  resultado.dataset.estado = valido ? "valido" : "error";
  resultado.innerHTML = valido
    ? `<strong>${escaparHTML(respuesta.titulo)}</strong><p>${escaparHTML(respuesta.mensaje)}</p><dl><dt>Referencia</dt><dd><code>${escaparHTML(respuesta.referencia)}</code></dd><dt>Estado</dt><dd>${escaparHTML(respuesta.estado)}</dd><dt>Alcance</dt><dd>${escaparHTML(respuesta.alcance)}</dd></dl>`
    : `<strong>No se ha podido acreditar el documento</strong><p>${escaparHTML(respuesta?.mensaje || "La referencia no consta como vigente o el servicio no está disponible.")}</p>`;
}

async function cotejar(referencia, signal) {
  if (!referenciaValida(referencia)) throw new Error("La referencia no respeta el formato admitido.");
  const respuesta = await fetch(RUTA_COTEJO_PUBLICO, {
    method: "POST",
    credentials: "omit",
    signal,
    headers: { Accept: "application/json", "Content-Type": "application/json" },
    body: JSON.stringify({ esquema: "vec.documentos.cotejo.publico.solicitud.v1", referencia }),
  });
  if (!respuesta.ok) throw new Error(respuesta.status === 404
    ? "La referencia no consta como vigente." : "El servicio de cotejo no está disponible.");
  const envelope = await respuesta.json();
  if (!envelope?.data || typeof envelope.data !== "object") throw new Error("El servicio devolvió una respuesta no válida.");
  return envelope.data;
}

async function comprobar(evento) {
  evento?.preventDefault();
  const referencia = entrada.value.trim();
  const boton = formulario.querySelector("button[type='submit']");
  const intento = ++intentoActual;
  controladorActual?.abort();
  const controlador = new AbortController();
  controladorActual = controlador;
  let temporizador;
  boton.disabled = true;
  resultado.hidden = false;
  resultado.removeAttribute("data-estado");
  resultado.textContent = "Comprobando la referencia…";
  try {
    const operacion = cotejar(referencia, controlador.signal);
    const respuesta = await Promise.race([
      operacion,
      new Promise((_, rechazar) => {
        temporizador = setTimeout(() => {
          rechazar(new Error(traducirVerificar("tiempo_espera")));
          controlador.abort();
        }, TIEMPO_MAX_COTEJO_MS);
      }),
    ]);
    if (intento === intentoActual) pintar(respuesta);
  } catch (error) {
    if (intento === intentoActual) {
      pintar({ valido: false, mensaje: error instanceof Error ? error.message : "No se pudo completar la comprobación." });
    }
  } finally {
    clearTimeout(temporizador);
    if (intento === intentoActual) {
      controladorActual = undefined;
      boton.disabled = false;
    }
  }
}

formulario.addEventListener("submit", comprobar);
const parametros = parametrosCerrados();
if (referenciaValida(parametros)) {
  entrada.value = parametros;
  void comprobar();
}
