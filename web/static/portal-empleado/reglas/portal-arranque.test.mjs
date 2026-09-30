import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { EventEmitter, once } from "node:events";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { createServer } from "node:http";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";

const raiz = path.resolve(fileURLToPath(new URL("../..", import.meta.url)));
const tipos = { ".html": "text/html", ".js": "text/javascript", ".json": "application/json", ".css": "text/css", ".svg": "image/svg+xml" };

function lanzarEnGrupo(archivo, argumentos, { maxBuffer }, callback) {
  const chrome = spawn(archivo, argumentos, { detached: true, stdio: ["ignore", "pipe", "pipe"] });
  const trozos = [[], []];
  const tamanos = [0, 0];
  let errorProceso;
  chrome.once("error", (error) => { errorProceso = error; });
  for (const [indice, tuberia] of [chrome.stdout, chrome.stderr].entries()) {
    tuberia?.on("data", (trozo) => {
      if (errorProceso) return;
      tamanos[indice] += trozo.length;
      if (tamanos[indice] > maxBuffer) {
        errorProceso = Object.assign(new Error("Chrome superó maxBuffer"), { code: "ERR_CHILD_PROCESS_STDIO_MAXBUFFER" });
        chrome.kill("SIGKILL");
        if (Number.isInteger(chrome.pid) && chrome.pid > 0) {
          try { process.kill(-chrome.pid, "SIGKILL"); }
          catch (error) { if (error.code !== "ESRCH") errorProceso = error; }
        }
        return;
      }
      trozos[indice].push(trozo);
    });
  }
  chrome.once("close", (codigo, senal) => {
    const error = errorProceso || ((codigo !== 0 || senal)
      ? Object.assign(new Error(`Chrome salió con ${codigo ?? senal}`), { code: codigo, signal: senal }) : null);
    callback(error, Buffer.concat(trozos[0]).toString("utf8"), Buffer.concat(trozos[1]).toString("utf8"));
  });
  return chrome;
}

function volcarChrome(argumentos, { lanzar = lanzarEnGrupo, matarGrupo = process.kill, plazoMs = 15000, cierreMs = 500 } = {}) {
  return new Promise((resolver, rechazar) => {
    let terminado = false;
    let plazoAgotado = false;
    let errorAnterior;
    let salida;
    let cierreRecibido = false;
    let falloAntesDelPlazo;
    let plazo;
    let cierre;
    const finalizar = (error, resultado) => {
      if (terminado) return;
      terminado = true;
      clearTimeout(plazo);
      clearTimeout(cierre);
      if (error) rechazar(new Error(
        `Chrome falló (${estado()}, motivo=${error.code ?? error.signal ?? error.message})`,
        { cause: error },
      ));
      else resolver(resultado);
    };
    const estado = () => `exit=${salida ? `${salida.codigo ?? "null"}/${salida.senal ?? "null"}` : "pendiente"}, cierre=${cierreRecibido}`;
    const salidaAnomala = () => salida && (plazoAgotado
      ? ((salida.codigo != null && salida.codigo !== 0) || (salida.senal && salida.senal !== "SIGKILL"))
      : (salida.codigo !== 0 || salida.senal));
    const chrome = lanzar("/usr/bin/google-chrome", argumentos,
      { detached: true, maxBuffer: 8 * 1024 * 1024 }, (error, dom) => {
        cierreRecibido = true;
        const cierrePorPlazo = plazoAgotado && error && error.code == null && error.signal === "SIGKILL";
        if (falloAntesDelPlazo || errorAnterior || chrome.killed || salidaAnomala() || (error && !cierrePorPlazo)) {
          finalizar(errorAnterior ?? error ?? new Error("Chrome falló durante el cierre"));
        } else if (plazoAgotado) {
          finalizar(null, { plazoAgotado: true, estado: estado() });
        } else {
          finalizar(null, { plazoAgotado: false, dom });
        }
      });
    chrome.once("error", (error) => { errorAnterior = error; });
    chrome.once("exit", (codigo, senal) => { salida = { codigo, senal }; });
    plazo = setTimeout(() => {
      // El cierre espera también a stdout/stderr. Un renderer puede mantenerlos
      // abiertos después de que el proceso principal ya haya salido.
      falloAntesDelPlazo = errorAnterior || chrome.killed || salidaAnomala();
      if (!falloAntesDelPlazo) plazoAgotado = true;
      if (Number.isInteger(chrome.pid) && chrome.pid > 0) {
        try { matarGrupo(-chrome.pid, "SIGKILL"); }
        catch (error) { if (error.code !== "ESRCH") { finalizar(error); return; } }
      }
      cierre = setTimeout(() => {
        // La salida del proceso no garantiza que sus tuberías cierren a tiempo.
        // Tras matar nuestro grupo se dejan de esperar y se conserva el plazo.
        chrome.stdout?.destroy();
        chrome.stderr?.destroy();
        if (falloAntesDelPlazo || errorAnterior || chrome.killed || salidaAnomala()) {
          finalizar(errorAnterior ?? new Error(falloAntesDelPlazo
            ? "Chrome falló antes de agotar el plazo" : "Chrome falló durante el cierre"));
        } else {
          finalizar(null, { plazoAgotado: true, estado: estado() });
        }
      }, cierreMs);
    }, plazoMs);
  });
}

test("el lanzador crea y termina un grupo de procesos propio", async () => {
  let errorSalida;
  const proceso = lanzarEnGrupo("/usr/bin/sleep", ["10"], { maxBuffer: 1024 }, (error) => { errorSalida = error; });
  const cerrado = once(proceso, "close");
  let grupoPropio = false;
  let cierreConfirmado = false;
  try {
    const estadisticas = await readFile(`/proc/${proceso.pid}/stat`, "utf8");
    const campos = estadisticas.slice(estadisticas.lastIndexOf(")") + 1).trim().split(/\s+/u);
    grupoPropio = Number(campos[2]) === proceso.pid;
    assert.ok(grupoPropio, "el proceso debe liderar su grupo antes de enviar la señal");
    process.kill(-proceso.pid, "SIGKILL");
    await cerrado;
    cierreConfirmado = true;
    assert.equal(errorSalida.signal, "SIGKILL");
  } finally {
    if (!cierreConfirmado && grupoPropio) {
      try { process.kill(-proceso.pid, "SIGKILL"); }
      catch (error) { if (error.code !== "ESRCH") throw error; }
    } else if (!cierreConfirmado) proceso.kill("SIGKILL");
  }
});

test("el cierre tardío de tuberías conserva el vencimiento real y termina la espera", async () => {
  const chrome = new EventEmitter();
  const tuberias = [];
  chrome.pid = 424242;
  chrome.killed = false;
  chrome.stdout = { destroy: () => tuberias.push("stdout") };
  chrome.stderr = { destroy: () => tuberias.push("stderr") };
  let devolver;
  const grupos = [];
  const resultado = volcarChrome([], {
    lanzar: (_ruta, _argumentos, _opciones, callback) => {
      devolver = callback;
      queueMicrotask(() => chrome.emit("exit", 0, null));
      return chrome;
    },
    matarGrupo: (pid, senal) => grupos.push([pid, senal]),
    plazoMs: 10,
    cierreMs: 10,
  });
  assert.deepEqual(await resultado, { plazoAgotado: true, estado: "exit=0/null, cierre=false" });
  assert.deepEqual(grupos, [[-424242, "SIGKILL"]]);
  assert.deepEqual(tuberias, ["stdout", "stderr"]);
  devolver(null, "DOM tardío");
});

test("un error anterior al plazo no se convierte en reintento", async () => {
  const chrome = new EventEmitter();
  chrome.pid = 424243;
  chrome.killed = true;
  chrome.stdout = { destroy: () => {} };
  chrome.stderr = { destroy: () => {} };
  const resultado = volcarChrome([], {
    lanzar: () => {
      queueMicrotask(() => chrome.emit("exit", null, "SIGTERM"));
      return chrome;
    },
    matarGrupo: () => {},
    plazoMs: 10,
    cierreMs: 10,
  });
  await assert.rejects(resultado, /Chrome falló \(exit=null\/SIGTERM, cierre=false, motivo=Chrome falló antes de agotar el plazo\)/u);
});

test("un error recibido después del plazo no se convierte en reintento", async () => {
  for (const error of [
    Object.assign(new Error("maxBuffer"), { code: "ERR_CHILD_PROCESS_STDIO_MAXBUFFER" }),
    Object.assign(new Error("señal ajena"), { signal: "SIGTERM" }),
  ]) {
    const chrome = new EventEmitter();
    chrome.pid = 424244;
    chrome.killed = false;
    chrome.stdout = { destroy: () => {} };
    chrome.stderr = { destroy: () => {} };
    let devolver;
    let notificarMuerte;
    const muerte = new Promise((resolver) => { notificarMuerte = resolver; });
    const resultado = volcarChrome([], {
      lanzar: (_ruta, _argumentos, _opciones, callback) => {
        devolver = callback;
        return chrome;
      },
      matarGrupo: () => notificarMuerte(),
      plazoMs: 10,
      cierreMs: 100,
    });
    await muerte;
    devolver(error, "");
    await assert.rejects(resultado, new RegExp(`motivo=${error.code ?? error.signal}`, "u"));
  }
});

test("el cierre causado por el plazo descarta cualquier DOM tardío", async () => {
  for (const error of [null, Object.assign(new Error("cierre propio"), { signal: "SIGKILL" })]) {
    const chrome = new EventEmitter();
    chrome.pid = 424245;
    chrome.killed = false;
    chrome.stdout = { destroy: () => {} };
    chrome.stderr = { destroy: () => {} };
    let devolver;
    let notificarMuerte;
    const muerte = new Promise((resolver) => { notificarMuerte = resolver; });
    const resultado = volcarChrome([], {
      lanzar: (_ruta, _argumentos, _opciones, callback) => {
        devolver = callback;
        return chrome;
      },
      matarGrupo: () => {
        chrome.emit("exit", null, "SIGKILL");
        notificarMuerte();
      },
      plazoMs: 10,
      cierreMs: 100,
    });
    await muerte;
    devolver(error, "DOM tardío");
    assert.deepEqual(await resultado, { plazoAgotado: true, estado: "exit=null/SIGKILL, cierre=true" });
  }
});

test("el cierre limitado rechaza errores y salidas anómalas tardías sin callback", async () => {
  for (const tipo of ["error", "salida"]) {
    const chrome = new EventEmitter();
    chrome.pid = 424246;
    chrome.killed = false;
    chrome.stdout = { destroy: () => {} };
    chrome.stderr = { destroy: () => {} };
    let notificarMuerte;
    const muerte = new Promise((resolver) => { notificarMuerte = resolver; });
    const resultado = volcarChrome([], {
      lanzar: () => chrome,
      matarGrupo: () => notificarMuerte(),
      plazoMs: 10,
      cierreMs: 10,
    });
    await muerte;
    if (tipo === "error") chrome.emit("error", Object.assign(new Error("EIO"), { code: "EIO" }));
    else chrome.emit("exit", null, "SIGTERM");
    await assert.rejects(resultado, tipo === "error" ? /motivo=EIO/u : /exit=null\/SIGTERM/u);
  }
});

test("el portal inicia en Chrome con el catálogo de reglas ausente, malformado o parcial", { timeout: 45000 }, async () => {
  let reglasPedidas = 0;
  let fallo = "ausente";
  const servidor = createServer(async (peticion, respuesta) => {
    const ruta = new URL(peticion.url, "http://localhost").pathname;
    if (ruta.startsWith("/textos/") && ruta.endsWith("/reglas.json")) {
      reglasPedidas += 1;
      if (fallo === "ausente") respuesta.writeHead(404, { "Cache-Control": "no-store" }).end();
      else respuesta.writeHead(200, { "Content-Type": "application/json", "Cache-Control": "no-store" })
        .end(fallo === "malformado" ? "{" : fallo === "general_vacio" ? '{"general":{}}'
          : '{"general":{"titulo":"Current rules"}}');
      return;
    }
    if (ruta.startsWith("/api/")) {
      respuesta.writeHead(503, { "Content-Type": "application/json" }).end("{}");
      return;
    }
    const relativa = ruta.endsWith("/") ? `${ruta}index.html` : ruta;
    const fichero = path.resolve(raiz, `.${relativa}`);
    if (!fichero.startsWith(`${raiz}${path.sep}`)) {
      respuesta.writeHead(404).end();
      return;
    }
    try {
      const contenido = await readFile(fichero);
      respuesta.writeHead(200, { "Content-Type": tipos[path.extname(fichero)] ?? "application/octet-stream" }).end(contenido);
    } catch { respuesta.writeHead(404).end(); }
  });
  try {
    await new Promise((resolver) => servidor.listen(0, "127.0.0.1", resolver));
    const puerto = servidor.address().port;
    const abrir = async (ruta, escenario) => {
      // Cada dump necesita su propio perfil: un Chrome anterior puede dejar
      // vivo su proceso de perfil durante unos instantes en el runner de CI.
      // Solo se repite un plazo agotado: Chrome puede salir 0 y sin DOM tras
      // recibir la señal de cierre. Un DOM vacío sin plazo sigue fallando.
      for (let intento = 0; intento < 2; intento += 1) {
        const perfil = await mkdtemp(path.join(tmpdir(), "vec-reglas-chrome-"));
        try {
          let resultado;
          try {
            resultado = await volcarChrome([
              "--headless=new", "--no-sandbox", "--disable-gpu", "--disable-dev-shm-usage",
              "--disable-background-networking", "--disable-extensions", "--no-first-run",
              `--user-data-dir=${perfil}`, "--virtual-time-budget=6000", "--dump-dom",
              `http://127.0.0.1:${puerto}${ruta}`,
            ]);
          } catch (error) {
            throw new Error(`Chrome falló en ${escenario} (${ruta}, intento ${intento + 1}): ${error.message}`, { cause: error });
          }
          if (!resultado.plazoAgotado) return resultado.dom;
          if (intento === 1) {
            throw new Error(`Chrome agotó dos veces el plazo de ${escenario} (${ruta}, intento ${intento + 1}, ${resultado.estado})`);
          }
        } finally {
          await rm(perfil, { recursive: true, force: true });
        }
      }
      throw new Error(`Chrome no produjo DOM para ${escenario} (${ruta})`);
    };
    for (const escenario of ["ausente", "malformado", "general_vacio", "general_parcial"]) {
      fallo = escenario;
      const pantalla = await abrir("/portal-empleado/reglas/?lang=en", escenario);
      assert.match(pantalla, /<html lang="en"/u);
      assert.match(pantalla, /id="rg-estado"[^>]*>The service is not available right now/u);
      assert.match(pantalla, /<button type="button" class="rg-secundario">Try again<\/button>/u);
      assert.match(pantalla, /<h1 data-i18n="titulo">Current rules<\/h1>/u);
      const portal = await abrir("/portal-empleado/?lang=en", `portal:${escenario}`);
      assert.match(portal, /id="espacio-trabajo"[^>]*>[\s\S]*?(?:data-inicio-pendiente|data-inicio-sin-modulos|portal-rrhh-inicio|portal-inicio-empleado)/u);
    }
    assert.ok(reglasPedidas >= 4, "Chrome recibió los cuatro fallos del catálogo de reglas");
  } finally {
    await new Promise((resolver) => servidor.close(resolver));
  }
});
