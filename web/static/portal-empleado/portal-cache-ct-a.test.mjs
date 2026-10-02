import { readFile } from "node:fs/promises";
import test from "node:test";
import { exigirVersiones, posterior } from "./versiones-cache.test-helper.mjs";

const versionEntradaAnterior = "20261002-a-recuperar-379-v1";
const versionCoordinador = "20261002-ct-fin-modalidad-v1";
const raiz = new URL("./", import.meta.url);

test("la extracción CT renueva cada padre hasta la entrada del portal", async () => {
  const [html, portal, expediente] = await Promise.all([
    readFile(new URL("index.html", raiz), "utf8"),
    readFile(new URL("portal.js", raiz), "utf8"),
    readFile(new URL("modulos/contratacion-temporal/vista-expedientes.js", raiz), "utf8"),
  ]);
  const versionEntrada = exigirVersiones(html, "/portal-empleado/portal.js", posterior(versionEntradaAnterior));
  exigirVersiones(html, "/portal-empleado/portal-modulos-coordinador.js", versionCoordinador);
  exigirVersiones(portal, "./portal-modulos-coordinador.js", versionCoordinador);
  exigirVersiones(expediente, "./seguimiento-cese.js", "20261001-ct-a-i18n-v1");
});
