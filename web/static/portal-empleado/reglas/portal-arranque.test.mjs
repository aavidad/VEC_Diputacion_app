import assert from "node:assert/strict";
import { once } from "node:events";
import { readFile } from "node:fs/promises";
import { createServer } from "node:http";
import path from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";
import { abrirNavegador, cargarPantalla, lanzarEnGrupo, matarGrupo } from "../../../../scripts/tests/chrome_cdp.mjs";

const raiz = path.resolve(fileURLToPath(new URL("../..", import.meta.url)));
const tipos = { ".html": "text/html", ".js": "text/javascript", ".json": "application/json", ".css": "text/css", ".svg": "image/svg+xml" };

const PORTAL_INICIADO = /id="espacio-trabajo"[^>]*>[\s\S]*?(?:data-inicio-pendiente|data-inicio-sin-modulos|portal-rrhh-inicio|portal-inicio-empleado)/u;
test("el lanzador crea y termina un grupo de procesos propio", async () => {
  const proceso = lanzarEnGrupo("/usr/bin/sleep", ["10"]);
  const cerrado = once(proceso, "close");
  let cierreConfirmado = false;
  try {
    const estadisticas = await readFile(`/proc/${proceso.pid}/stat`, "utf8");
    const campos = estadisticas.slice(estadisticas.lastIndexOf(")") + 1).trim().split(/\s+/u);
    assert.equal(Number(campos[2]), proceso.pid, "el proceso debe liderar su grupo antes de enviar la señal");
    matarGrupo(proceso);
    const [, senal] = await cerrado;
    cierreConfirmado = true;
    assert.equal(senal, "SIGKILL");
  } finally {
    if (!cierreConfirmado) proceso.kill("SIGKILL");
  }
});

test("el portal inicia en Chrome con el catálogo de reglas ausente, malformado o parcial", { timeout: 240000 }, async (t) => {
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
  let navegador;
  try {
    await new Promise((resolver) => servidor.listen(0, "127.0.0.1", resolver));
    const puerto = servidor.address().port;
    const inicio = performance.now();
    navegador = await abrirNavegador();
    t.diagnostic(`arranque de Chrome: ${Math.round(performance.now() - inicio)} ms`);
    const abrir = (ruta, listo) => cargarPantalla(navegador, `http://127.0.0.1:${puerto}${ruta}`, listo);
    for (const escenario of ["ausente", "malformado", "general_vacio", "general_parcial"]) {
      fallo = escenario;
      const pantalla = await abrir("/portal-empleado/reglas/?lang=en",
        // Estado final de la pantalla sin catálogo: aviso con «Reintentar».
        "document.querySelector('#rg-estado .rg-secundario')");
      assert.match(pantalla, /<html lang="en"/u);
      assert.match(pantalla, /id="rg-estado"[^>]*>The service is not available right now/u);
      assert.match(pantalla, /<button type="button" class="rg-secundario">Try again<\/button>/u);
      assert.match(pantalla, /<h1\b[^>]*data-i18n="titulo"[^>]*>Current rules<\/h1>/u);
      const portal = await abrir("/portal-empleado/?lang=en", `${PORTAL_INICIADO}.test(document.documentElement.outerHTML)`);
      assert.match(portal, PORTAL_INICIADO);
    }
    assert.ok(reglasPedidas >= 4, "Chrome recibió los cuatro fallos del catálogo de reglas");
  } finally {
    await navegador?.cerrar();
    await new Promise((resolver) => servidor.close(resolver));
  }
});
