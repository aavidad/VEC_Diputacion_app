import assert from "node:assert/strict";
import { execFile } from "node:child_process";
import { cp, mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { pathToFileURL } from "node:url";
import { promisify } from "node:util";
import test from "node:test";

import { cargarMensajesPortal } from "./portal-i18n.js";
import { cargarTextos } from "../comun/textos.js";

const ejecutar = promisify(execFile);

test("las fases del shell proceden del portal solicitado, también al pedir el otro idioma", async () => {
  for (const idioma of ["es", "en"]) {
    const [mensajes, textos] = await Promise.all([
      cargarMensajesPortal(idioma), cargarTextos("portal", { idioma }),
    ]);
    for (const [clave, texto] of Object.entries(textos.seccion("fases_rrhh"))) {
      assert.equal(mensajes[`tramite_${clave}`], texto);
    }
  }
});

test("si falla un catálogo inglés, el shell muestra un conjunto español coherente sin importar CT", async () => {
  const temporal = await mkdtemp(path.join(tmpdir(), "vec-portal-p5-"));
  try {
    const raiz = new URL("../", import.meta.url);
    const archivos = ["comun/idioma.js", "comun/textos.js", "textos/idiomas.json", "portal-empleado/portal-i18n.js"];
    for (const idioma of ["es", "en"]) {
      for (const modulo of ["portal", "portal-ayuda", "preferencias", "bolsa"]) {
        if (idioma === "en" && modulo === "portal-ayuda") continue;
        archivos.push(`textos/${idioma}/${modulo}.json`);
      }
    }
    for (const archivo of archivos) {
      const destino = path.join(temporal, archivo);
      await mkdir(path.dirname(destino), { recursive: true });
      await cp(new URL(archivo, raiz), destino);
    }
    await writeFile(path.join(temporal, "package.json"), JSON.stringify({ type: "module" }));
    const entrada = pathToFileURL(path.join(temporal, "portal-empleado/portal-i18n.js")).href;
    const script = `
      import assert from 'node:assert/strict';
      globalThis.location = { href: 'file:///portal?lang=en' };
      const { cargarMensajesPortal, MENSAJES_PORTAL, MENSAJES_BOLSA_INTERNA, LOCALIZACION_PORTAL } = await import(process.argv[1]);
      const textos = await import(new URL('../comun/textos.js', process.argv[1]));
      const espanol = (await textos.cargarTextos('portal', { idioma: 'es' })).seccion('fases_rrhh');
      const resultado = await cargarMensajesPortal('en');
      assert.equal(MENSAJES_PORTAL.tramite_fase_solicitud, espanol.fase_solicitud);
      assert.equal(resultado.tramite_fase_solicitud, espanol.fase_solicitud);
      assert.equal(resultado.txt_modulos, (await textos.cargarTextos('portal', { idioma: 'es' })).seccion('textos').txt_modulos);
      assert.equal(MENSAJES_BOLSA_INTERNA.fecha_sin_valor,
        (await textos.cargarTextos('portal', { idioma: 'es' })).seccion('bolsa_interna').fecha_sin_valor);
      assert.equal(LOCALIZACION_PORTAL, 'es-ES');
    `;
    const salida = await ejecutar(process.execPath, ["--input-type=module", "-e", script, entrada]);
    assert.equal(salida.stderr.includes("ERR_MODULE_NOT_FOUND"), false);
  } finally {
    await rm(temporal, { recursive: true, force: true });
  }
});
