import assert from "node:assert/strict";
import { execFile } from "node:child_process";
import { cp, mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { pathToFileURL } from "node:url";
import { promisify } from "node:util";
import test from "node:test";

const ejecutar = promisify(execFile);

async function probarIdioma(idioma) {
  const temporal = await mkdtemp(path.join(tmpdir(), "vec-portal-grupos-"));
  try {
    const raiz = new URL("../", import.meta.url);
    const fuentes = ["portal-empleado/portal-i18n.js"];
    for (const codigo of ["es", "en"]) {
      for (const modulo of ["portal", "portal-ayuda", "preferencias", "bolsa"]) {
        fuentes.push(`textos/${codigo}/${modulo}.json`);
      }
    }
    for (const fuente of fuentes) {
      const destino = path.join(temporal, fuente);
      await mkdir(path.dirname(destino), { recursive: true });
      await cp(new URL(fuente, raiz), destino);
    }
    await mkdir(path.join(temporal, "comun"), { recursive: true });
    await writeFile(path.join(temporal, "package.json"), JSON.stringify({ type: "module" }));
    await writeFile(path.join(temporal, "comun/idioma.js"), `
      export const IDIOMA_POR_DEFECTO = "es";
      export const localizacionDe = (idioma) => idioma === "en" ? "en-GB" : "es-ES";
    `);
    await writeFile(path.join(temporal, "comun/textos.js"), `
      import { readFile } from "node:fs/promises";
      export async function cargarTextos(modulo, { idioma = globalThis.__idioma } = {}) {
        globalThis.__peticiones.push({ modulo, idioma });
        if (globalThis.__fallo === modulo) { globalThis.__fallo = null; throw new Error("503 sintético"); }
        const efectivo = globalThis.__respaldo === modulo ? "es" : idioma;
        globalThis.__respaldo = null;
        const dato = JSON.parse(await readFile(new URL(\`../textos/\${efectivo}/\${modulo}.json\`, import.meta.url), "utf8"));
        return { idioma: efectivo, seccion: (nombre) => dato[nombre] };
      }
      export function reintentarTextos(...args) {
        globalThis.__reintentos += 1;
        return cargarTextos(...args);
      }
    `);
    const entrada = pathToFileURL(path.join(temporal, "portal-empleado/portal-i18n.js")).href;
    const script = `
      import assert from "node:assert/strict";
      globalThis.__idioma = process.argv[2];
      globalThis.__peticiones = [];
      globalThis.__fallo = null;
      globalThis.__respaldo = null;
      globalThis.__reintentos = 0;
      const portal = await import(process.argv[1]);
      assert.deepEqual(globalThis.__peticiones, [{ modulo: "portal", idioma: globalThis.__idioma }]);
      for (const grupo of ["ayuda", "preferencias", "bolsa"]) assert.equal(portal.textosGrupoPortalPreparados(grupo), false);
      assert.throws(() => portal.traducirPortal("ayuda_pasos"), /desconocida/);
      for (const [grupo, modulo] of [["ayuda", "portal-ayuda"], ["preferencias", "preferencias"], ["bolsa", "bolsa"]]) {
        globalThis.__fallo = modulo;
        await assert.rejects(portal.prepararTextosPortal(grupo), /503 sintético/);
        assert.equal(portal.textosGrupoPortalPreparados(grupo), false);
        const antes = globalThis.__peticiones.length;
        await Promise.all([portal.prepararTextosPortal(grupo), portal.prepararTextosPortal(grupo)]);
        assert.equal(globalThis.__peticiones.length, antes + 1, grupo);
        assert.equal(portal.textosGrupoPortalPreparados(grupo), true);
      }
      assert.equal(globalThis.__reintentos, 3);
      assert.deepEqual(globalThis.__peticiones.map(({ modulo }) => modulo),
        ["portal", "portal-ayuda", "portal-ayuda", "preferencias", "preferencias", "bolsa", "bolsa"]);
      assert.ok(globalThis.__peticiones.every(({ idioma }) => idioma === globalThis.__idioma));
      assert.ok(portal.traducirPortal("ayuda_pasos"));
      assert.ok(portal.traducirPortal("preferencias_titulo"));
      assert.ok(portal.traducirPortal("rrhh_plazos_error_carga"));
    `;
    await ejecutar(process.execPath, ["--input-type=module", "-e", script, entrada, idioma]);
  } finally {
    await rm(temporal, { recursive: true, force: true });
  }
}

test("la base solo lee portal y Ayuda, Preferencias y Bolsa se preparan al abrir en ES", () => probarIdioma("es"));
test("la carga diferida usa solo EN y reintenta el grupo fallido sin mezclar idiomas", () => probarIdioma("en"));
