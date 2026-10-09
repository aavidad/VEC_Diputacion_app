import { spawn } from "node:child_process";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";

const CHROME = "/usr/bin/google-chrome";
// Límites de fallo, no esperas: cada paso termina en cuanto Chrome responde o
// la pantalla alcanza su estado final. Solo se agotan si algo se cuelga.
// El primer arranque en un runner frío (caché de fuentes, binario fuera de la
// caché de disco) puede tardar decenas de segundos y no debe cortarse.
const LIMITE_ARRANQUE_MS = 60000;
const LIMITE_ORDEN_MS = 30000;
const LIMITE_PANTALLA_MS = 20000;
const LIMITE_CIERRE_MS = 5000;
const OPCIONES_CHROME = [
  "--headless=new", "--no-sandbox", "--disable-gpu", "--disable-dev-shm-usage",
  "--disable-background-networking", "--disable-extensions", "--no-first-run",
  // Las pestañas abiertas por CDP no tienen foco: sin estas opciones Chrome
  // baja la prioridad de su renderer y frena sus temporizadores, y con la CPU
  // saturada la pantalla deja de avanzar (las mismas que usa Puppeteer).
  "--disable-renderer-backgrounding", "--disable-background-timer-throttling",
  "--disable-backgrounding-occluded-windows",
];

// Chrome se lanza en un grupo de procesos propio para poder terminar también
// sus renderers. Con --remote-debugging-pipe lee órdenes del protocolo de
// depuración (CDP) por el descriptor 3 y responde por el 4.
export function lanzarEnGrupo(archivo, argumentos) {
  return spawn(archivo, argumentos, { detached: true, stdio: ["ignore", "ignore", "pipe", "pipe", "pipe"] });
}

export function matarGrupo(proceso) {
  if (!Number.isInteger(proceso.pid) || proceso.pid <= 0) return;
  try { process.kill(-proceso.pid, "SIGKILL"); }
  catch (error) { if (error.code !== "ESRCH") throw error; }
}

function conLimite(promesa, ms, descripcion) {
  let plazo;
  const vencido = new Promise((_, rechazar) => {
    plazo = setTimeout(() => rechazar(new Error(`Chrome no respondió a ${descripcion} en ${ms} ms`)), ms);
  });
  return Promise.race([promesa, vencido]).finally(() => clearTimeout(plazo));
}

// Cliente CDP mínimo sobre la tubería: mensajes JSON terminados en NUL.
function conectarCDP(proceso) {
  const pendientes = new Map();
  let siguiente = 0;
  let resto = Buffer.alloc(0);
  let caida;
  const caer = (error) => {
    caida ??= Object.assign(error, { chromeCaido: true });
    for (const { rechazar } of pendientes.values()) rechazar(caida);
    pendientes.clear();
  };
  proceso.stdio[4].on("data", (trozo) => {
    resto = Buffer.concat([resto, trozo]);
    for (let fin = resto.indexOf(0); fin !== -1; fin = resto.indexOf(0)) {
      const mensaje = JSON.parse(resto.subarray(0, fin).toString("utf8"));
      resto = resto.subarray(fin + 1);
      const pendiente = pendientes.get(mensaje.id);
      if (!pendiente) continue; // Eventos: esta prueba no los necesita.
      pendientes.delete(mensaje.id);
      if (mensaje.error) pendiente.rechazar(new Error(`${pendiente.metodo}: ${mensaje.error.message}`));
      else pendiente.resolver(mensaje.result);
    }
  });
  proceso.once("exit", (codigo, senal) => caer(new Error(`Chrome salió con ${codigo ?? senal}`)));
  proceso.once("error", caer);
  proceso.stdio[3].on("error", caer);
  return (metodo, parametros = {}, sesion, limiteMs = LIMITE_ORDEN_MS) => {
    if (caida) return Promise.reject(caida);
    const id = ++siguiente;
    const respuesta = new Promise((resolver, rechazar) => { pendientes.set(id, { resolver, rechazar, metodo }); });
    proceso.stdio[3].write(`${JSON.stringify({ id, method: metodo, params: parametros, ...(sesion ? { sessionId: sesion } : {}) })}\0`);
    return conLimite(respuesta, limiteMs, metodo);
  };
}

// Un solo Chrome por prueba con perfil temporal propio. Cada pantalla se abre
// en un contexto de navegación nuevo (caché y almacenamiento aislados), así
// que no comparte estado con las anteriores ni con otras pruebas.
export async function abrirNavegador() {
  const perfil = await mkdtemp(path.join(tmpdir(), "vec-reglas-chrome-"));
  const proceso = lanzarEnGrupo(CHROME, [...OPCIONES_CHROME, `--user-data-dir=${perfil}`, "--remote-debugging-pipe", "about:blank"]);
  const cerrado = new Promise((resolver) => { proceso.once("close", resolver); });
  let errores = "";
  proceso.stderr.on("data", (trozo) => { errores = (errores + trozo).slice(-2048); });
  const cdp = conectarCDP(proceso);
  const cerrar = async () => {
    await cdp("Browser.close").catch(() => {});
    await conLimite(cerrado, LIMITE_CIERRE_MS, "cierre").catch(() => {});
    matarGrupo(proceso);
    await conLimite(cerrado, LIMITE_CIERRE_MS, "cierre forzado").catch(() => {});
    await rm(perfil, { recursive: true, force: true });
  };
  try {
    await cdp("Browser.getVersion", {}, undefined, LIMITE_ARRANQUE_MS);
  } catch (error) {
    await cerrar();
    throw new Error(`Chrome no arrancó: ${error.message}; stderr: ${errores}`, { cause: error });
  }
  return { cdp, cerrar };
}

// Navega y espera a que la pantalla alcance su estado final (expresión `listo`
// verdadera) en lugar de un tiempo virtual fijo. Devuelve el DOM resultante;
// si el estado no llega, devuelve el DOM tal cual para que las aserciones
// digan qué falta.
export async function cargarPantalla({ cdp }, url, listo) {
  const { browserContextId } = await cdp("Target.createBrowserContext");
  try {
    const { targetId } = await cdp("Target.createTarget", { url: "about:blank", browserContextId });
    const { sessionId } = await cdp("Target.attachToTarget", { targetId, flatten: true });
    const { errorText } = await cdp("Page.navigate", { url }, sessionId);
    if (errorText) throw new Error(`Chrome no pudo abrir ${url}: ${errorText}`);
    const evaluar = async (expresion) => (await cdp("Runtime.evaluate", { expression: expresion, returnByValue: true }, sessionId)).result?.value;
    const limite = Date.now() + LIMITE_PANTALLA_MS;
    for (;;) {
      // Durante la navegación el contexto puede no existir aún: se reintenta.
      const preparado = await evaluar(`Boolean(${listo})`).catch((error) => { if (error.chromeCaido) throw error; return false; });
      if (preparado || Date.now() >= limite) break;
      await new Promise((resolver) => { setTimeout(resolver, 50); });
    }
    return await evaluar("document.documentElement.outerHTML");
  } finally {
    await cdp("Target.disposeBrowserContext", { browserContextId }).catch(() => {});
  }
}
