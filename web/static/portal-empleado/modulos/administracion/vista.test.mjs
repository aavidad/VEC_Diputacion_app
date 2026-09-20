import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { DATOS_ADMINISTRACION_PRESENTACION, crearEstadoAdministracion, PESTANAS_ADMINISTRACION } from "./datos-presentacion.js";
import { CLAVES_I18N_ADMINISTRACION, crearTraductorAdministracion } from "./i18n.js";
import { filtrarFilasAdministracion } from "./vista.js";

test("Administración declara datos sintéticos, sus secciones y conexión pendiente", () => {
  assert.equal(DATOS_ADMINISTRACION_PRESENTACION.aviso, "Datos ficticios de presentación");
  const estado = crearEstadoAdministracion(crearTraductorAdministracion());
  assert.equal(estado.estado, "visual_pendiente_backend");
  assert.deepEqual(PESTANAS_ADMINISTRACION, ["resumen", "roles", "catalogos", "calendarios", "reglas", "conectores", "modulos", "privacidad", "ia"]);
  assert.match(estado.pendientes.join(" "), /cifrado.*secretos/iu);
});

test("el catálogo español es cerrado y contiene todas las claves que consume la vista", async () => {
  const t = crearTraductorAdministracion();
  assert.equal(t("tab_ia"), "IA local y RAG");
  assert.throws(() => t("clave_inexistente"), /clave i18n/u);
  const fuente = await readFile(new URL("./vista.js", import.meta.url), "utf8");
  const claves = [...fuente.matchAll(/t\("([a-z_]+)"/gu)].map((coincidencia) => coincidencia[1]);
  claves.forEach((clave) => assert.ok(CLAVES_I18N_ADMINISTRACION.includes(clave), `clave ausente: ${clave}`));
});

test("la vista administrativa no incorpora red, almacenamiento ni literales UI clave", async () => {
  const fuente = await readFile(new URL("./vista.js", import.meta.url), "utf8");
  assert.match(fuente, /export function montarVistaAdministracion/u);
  assert.doesNotMatch(fuente, /(?:Probar conexión|Activar IA local|Guardar catálogo|Rotar secreto|Filtrar elementos visibles)/u);
  assert.doesNotMatch(fuente, /(?:fetch\(|localStorage|sessionStorage|document\.cookie|indexedDB)/u);
});

test("el filtro conserva el índice original usado por el detalle", () => {
  const filas = [
    ["Gestión de RRHH", "rol:rrhh", "Activo"],
    ["Administración de seguridad", "rol:seguridad", "Activo"],
  ];
  assert.deepEqual(filtrarFilasAdministracion(filas, "seguridad"), [
    { fila: filas[1], indice: 1 },
  ]);
});
