// El registro técnico acepta únicamente valores cerrados. Los detalles del
// Error, la URL y la pila nunca salen del navegador.
const RUTA = "/api/vec/observabilidad/errores-cliente";
const CODIGOS = new Set(["CLIENTE_FALLO_NO_CLASIFICADO", "MODULO_WEB_NO_CARGADO"]);
const PANTALLAS = new Set(["portal_empleado"]);
const MAXIMO_POR_MINUTO = 5;
const instantes = [];
let instalacion = null;

function correlacionAleatoria() {
  if (typeof globalThis.crypto?.getRandomValues !== "function") return "";
  const bytes = globalThis.crypto.getRandomValues(new Uint8Array(16));
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("");
}

export function registrarErrorCliente({ pantalla = "portal_empleado", codigo = "CLIENTE_FALLO_NO_CLASIFICADO" } = {}) {
  if (!PANTALLAS.has(pantalla) || !CODIGOS.has(codigo)) return false;
  if (globalThis.location?.pathname?.startsWith("/portal-empleado/") !== true) return false;
  const ahora = Date.now();
  while (instantes.length && ahora - instantes[0] >= 60_000) instantes.shift();
  if (instantes.length >= MAXIMO_POR_MINUTO) return false;
  const correlacion = correlacionAleatoria();
  if (!correlacion || typeof globalThis.fetch !== "function") return false;
  instantes.push(ahora);
  // fetch permite fijar explícitamente la frontera de canal; sendBeacon no
  // permite declarar mode, credentials ni referrerPolicy.
  try {
    void globalThis.fetch(RUTA, {
      method: "POST", mode: "same-origin", credentials: "same-origin",
      cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
      keepalive: true,
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ pantalla, codigo, correlacion }),
    }).catch(() => {});
    return true;
  } catch {
    return false;
  }
}

export function iniciarRegistroErrores(ventana = globalThis.window) {
  if (!ventana?.addEventListener || !ventana?.removeEventListener) return () => {};
  if (instalacion && instalacion.ventana !== ventana) return () => {};
  if (!instalacion) {
    const error = () => { registrarErrorCliente(); };
    const rechazo = () => { registrarErrorCliente(); };
    ventana.addEventListener("error", error);
    ventana.addEventListener("unhandledrejection", rechazo);
    instalacion = { ventana, error, rechazo, usos: 0 };
  }
  const actual = instalacion;
  actual.usos++;
  let cerrado = false;
  return () => {
    if (cerrado) return;
    cerrado = true;
    actual.usos--;
    if (actual.usos === 0 && instalacion === actual) {
      actual.ventana.removeEventListener("error", actual.error);
      actual.ventana.removeEventListener("unhandledrejection", actual.rechazo);
      instalacion = null;
    }
  };
}
