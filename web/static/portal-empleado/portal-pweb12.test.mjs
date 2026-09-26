import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { manejarAccionAvisos } from "./portal-bolsas-avisos.js";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js";

const directorio = new URL("./", import.meta.url);
const portal = await readFile(new URL("portal.js", directorio), "utf8");

const avisos = {
  esquema: "vec.bolsa.rrhh.avisos.v1",
  provisionalidad: "Cómputo pendiente de RRHH.",
  conteos: { salto_orden: 1, tres_anos: 0 },
  items: [{
    tipo: "salto_orden", bolsa: "bolsa:pweb12", referencia: "aviso:pweb12",
    fecha: "2026-09-23T10:00:00Z",
    detalle: { participacion_ref: "participacion:pweb12", orden: 2, orden_primero_llamado: 4 },
  }],
  paginacion: { desde: 1, hasta: 1, total: 7, cursor_siguiente: "cursor:pweb12" },
};

test("P-WEB-12 monta Avisos inmediatamente después del cuadro B12 y declara el límite temporal", () => {
  const presentador = crearPresentadorPanelInterno({
    claseEstado: () => "neutro", encabezadoVista: () => "<header>Cuadro</header>",
    escaparHTML: (valor) => String(valor ?? ""), numero: (valor) => String(valor ?? 0),
    obtenerDatosPanel: () => null, tituloVista: () => "Cuadro",
    obtenerDatosBolsas: () => ({ carga: "listo", datos: { bolsas: [] }, error: "" }),
    obtenerDatosAvisos: () => ({ carga: "listo", datos: avisos, cursor: "", error: "" }),
  });
  const html = presentador.renderizarSoloBolsas("resumen");
  assert.ok(html.indexOf('id="titulo-cuadro-b12"') >= 0);
  assert.ok(html.indexOf('id="titulo-cuadro-b12"') < html.indexOf("<h2>Avisos</h2>"));
  assert.match(html, /data-accion="abrir-ficha-b5"/);
  assert.doesNotMatch(html, /Pendiente de RRHH|data-avisos-provisional/);
  assert.doesNotMatch(html, /Cuadro B12|Control interno|Provisional:/);
});

test("P-WEB-12 consulta Avisos una sola vez al montar y cancela al salir", () => {
  assert.match(portal, /if \(vista === "resumen" && estado\.datosAvisos === null\) void cargarAvisosBolsa\(\);/);
  assert.match(portal, /controladorBolsas\.cancelarPeticiones\(\); cancelarAvisosBolsa\(\);/);
  assert.match(portal, /estado\.datosAvisos = \{ carga: "cargando", datos: null, cursor, error: "" \}/);
});

test("P-WEB-12 delega Abrir ficha B5, Siguiente y Reintentar", () => {
  const llamadas = [];
  for (const [accion, esperado] of [["abrir-ficha-b5", "ficha"], ["siguiente-avisos", "siguiente"], ["reintentar-avisos", "reintentar"]]) {
    const control = { disabled: false, dataset: { accion, bolsaRef: "bolsa:pweb12", participacionRef: "participacion:pweb12" } };
    const consumida = manejarAccionAvisos({ target: { closest: () => control } }, {
      abrirFichaB5: () => llamadas.push("ficha"),
      siguiente: () => llamadas.push("siguiente"),
      reintentar: () => llamadas.push("reintentar"),
    });
    assert.equal(consumida, true);
    assert.equal(llamadas.at(-1), esperado);
  }
  assert.match(portal, /controladorBolsas\.abrirFicha\(participacionRef\)/);
});
