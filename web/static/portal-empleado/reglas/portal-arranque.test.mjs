import assert from "node:assert/strict";
import { execFile } from "node:child_process";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { createServer } from "node:http";
import { tmpdir } from "node:os";
import path from "node:path";
import { promisify } from "node:util";
import { fileURLToPath } from "node:url";
import test from "node:test";

const ejecutar = promisify(execFile);
const raiz = path.resolve(fileURLToPath(new URL("../..", import.meta.url)));
const tipos = { ".html": "text/html", ".js": "text/javascript", ".json": "application/json", ".css": "text/css", ".svg": "image/svg+xml" };

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
      // Un timeout de arranque se repite una vez con otro perfil; nunca se
      // repiten respuestas HTML que no superen las aserciones.
      for (let intento = 0; intento < 2; intento += 1) {
        const perfil = await mkdtemp(path.join(tmpdir(), "vec-reglas-chrome-"));
        try {
          const proceso = ejecutar("/usr/bin/google-chrome", [
            "--headless=new", "--no-sandbox", "--disable-gpu", "--disable-dev-shm-usage",
            "--disable-background-networking", "--disable-extensions", "--no-first-run",
            `--user-data-dir=${perfil}`, "--virtual-time-budget=6000", "--dump-dom",
            `http://127.0.0.1:${puerto}${ruta}`,
          ], { detached: true, timeout: 15000, maxBuffer: 8 * 1024 * 1024 });
          try {
            return (await proceso).stdout;
          } catch (error) {
            const agotado = error.killed === true && error.signal === "SIGTERM" && error.code == null;
            if (agotado && proceso.child?.pid) {
              try { process.kill(-proceso.child.pid, "SIGKILL"); }
              catch (matarError) { if (matarError.code !== "ESRCH") throw matarError; }
            }
            if (!agotado || intento === 1) {
              throw new Error(`Chrome no terminó ${escenario} (${ruta}) en el intento ${intento + 1}`, { cause: error });
            }
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
