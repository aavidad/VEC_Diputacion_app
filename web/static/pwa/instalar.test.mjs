import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { iniciarPWA } from "./instalar.js";

function preparar(portal, instalada, segura = true) {
  const registros = [];
  const movimientos = [];
  const botones = ["volver", "inicio", "recargar"].map((accion) => ({
    dataset: { pwaAccion: accion },
    setAttribute(nombre, valor) { this[nombre] = valor; },
    addEventListener(_tipo, accionClick) { this.click = accionClick; },
  }));
  const navegacion = {
    dataset: { pwaScope: portal }, hidden: true,
    setAttribute(nombre, valor) { this[nombre] = valor; },
    querySelectorAll: () => botones,
  };
  const manifiesto = { dataset: { pwaManifest: portal === "/area-personal/" ? "pwa-area-personal" : "pwa-portal-empleado" } };
  const documento = { querySelector: (selector) => selector.startsWith("link") ? manifiesto : navegacion };
  const ventana = {
    location: { origin: "https://vec.example", pathname: portal, assign: (ruta) => movimientos.push(ruta), reload: () => movimientos.push("reload") },
    history: { length: 1, back: () => movimientos.push("back") },
    matchMedia: () => ({ matches: instalada, addEventListener() {} }),
    isSecureContext: segura,
  };
  const navegador = { serviceWorker: { register: async (...args) => registros.push(args) } };
  return { documento, ventana, navegador, registros, movimientos, botones, navegacion, manifiesto };
}

test("RRHH y área personal muestran acciones sólo instalados y registran su propio scope", async () => {
  for (const [portal, idioma] of [["/portal-empleado/", "es"], ["/area-personal/", "en"]]) {
    const caso = preparar(portal, true);
    await iniciarPWA({ ...caso, idioma });
    assert.equal(caso.navegacion.hidden, false);
    assert.match(caso.manifiesto.href, new RegExp(`^/textos/${idioma}/pwa-[a-z-]+\\.json\\?v=20261002-pwa-v1$`));
    assert.deepEqual(caso.registros, [[`${portal}sw.js?v=20261003-pwa-ci-v5`, { scope: portal, updateViaCache: "none" }]]);
    assert.ok(caso.botones.every((boton) => boton.textContent && boton["aria-label"] === boton.textContent));
    caso.botones[0].click();
    caso.botones[1].click();
    caso.botones[2].click();
    assert.deepEqual(caso.movimientos, [`${portal}?lang=${idioma}`, `${portal}?lang=${idioma}`, "reload"]);
  }
});

test("fuera del modo instalado no muestra acciones; sin contexto seguro no registra worker", async () => {
  const caso = preparar("/area-personal/", false, false);
  await iniciarPWA(caso);
  assert.equal(caso.navegacion.hidden, true);
  assert.deepEqual(caso.registros, []);
});

test("el idioma de la URL se resuelve después de cargar el índice antes de pedir textos PWA", async () => {
  const caso = preparar("/portal-empleado/", true);
  caso.ventana.location.href = "https://vec.example/portal-empleado/?lang=en";
  await iniciarPWA(caso);
  assert.match(caso.manifiesto.href, /^\/textos\/en\/pwa-portal-empleado\.json\?/u);
  assert.equal(caso.botones[1].textContent, "Home");
  caso.botones[1].click();
  assert.deepEqual(caso.movimientos, ["/portal-empleado/?lang=en"]);
});

test("ambos portales enlazan PWA; el área espera a elegir idioma", async () => {
  const portal = await readFile(new URL("../portal-empleado/index.html", import.meta.url), "utf8");
  assert.match(portal, /data-pwa-manifest="pwa-portal-empleado"/u);
  assert.match(portal, /\/pwa\/instalar\.js\?v=20261008-pwa-idioma-v1/u);
  assert.match(portal, /\/pwa\/navegacion\.css\?v=20261002-pwa-v2/u);
  assert.match(portal, /data-pwa-scope="\/portal-empleado\/"/u);
  assert.doesNotMatch(portal, /href="\/textos\/es\/pwa-portal-empleado\.json/u);

  const [area, arranque] = await Promise.all([
    readFile(new URL("../area-personal/index.html", import.meta.url), "utf8"),
    readFile(new URL("../area-personal/arranque.js", import.meta.url), "utf8"),
  ]);
  assert.match(area, /<link rel="manifest" data-pwa-manifest="pwa-area-personal">/u);
  assert.doesNotMatch(area, /<script[^>]+\/pwa\/instalar\.js/u);
  assert.match(area, /\/pwa\/navegacion\.css\?v=20261002-pwa-v2/u);
  assert.match(area, /data-pwa-scope="\/area-personal\/"/u);
  assert.match(arranque, /const idioma = await iniciarI18nAreaPersonal\([\s\S]*if \(idioma === idiomaAreaPersonal[\s\S]*import\("\.\.\/pwa\/instalar\.js\?v=20261008-pwa-idioma-v1"\)/u);
});
