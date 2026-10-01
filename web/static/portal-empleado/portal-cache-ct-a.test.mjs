import { readFile } from "node:fs/promises";
import test from "node:test";
import { exigirVersiones } from "./versiones-cache.test-helper.mjs";

const version = "20261001-f-ct-validacion-v1";
const raiz = new URL("./", import.meta.url);

test("la extracción CT renueva cada padre hasta la entrada del portal", async () => {
  const [html, portal, expediente] = await Promise.all([
    readFile(new URL("index.html", raiz), "utf8"),
    readFile(new URL("portal.js", raiz), "utf8"),
    readFile(new URL("modulos/contratacion-temporal/vista-expedientes.js", raiz), "utf8"),
  ]);
  exigirVersiones(html, "/portal-empleado/portal.js", version);
  exigirVersiones(html, "/portal-empleado/portal-modulos-coordinador.js", version);
  exigirVersiones(portal, "./portal-modulos-coordinador.js", version);
  exigirVersiones(expediente, "./seguimiento-cese.js", "20261001-ct-a-i18n-v1");
});
