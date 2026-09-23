import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js";

test("P-WEB-14 cuenta las personas en renuncia del contrato B12", () => {
  const presentador = crearPresentadorPanelInterno({
    claseEstado: () => "neutro", encabezadoVista: () => "<header>Cuadro</header>",
    escaparHTML: (valor) => String(valor ?? ""), numero: (valor) => String(valor ?? 0),
    obtenerDatosPanel: () => null, tituloVista: () => "Cuadro",
    obtenerDatosBolsas: () => ({ carga: "listo", datos: { bolsas: [
      { bolsa_ref: "bolsa:uno", categoria: "Uno", categoria_clave: "uno", tipo_lista: "rotatoria", vigente_desde: "2026-09-18", total: 200, por_estado: { disponible: 100, renuncia: 15 }, llamamientos_en_curso: 0 },
      { bolsa_ref: "bolsa:dos", categoria: "Dos", categoria_clave: "dos", tipo_lista: "rotatoria", vigente_desde: "2026-09-18", total: 190, por_estado: { disponible: 152, renuncia: 9 }, llamamientos_en_curso: 0 },
    ] }, error: "" }),
  });
  const html = presentador.renderizarSoloBolsas("resumen");
  assert.match(html, /<span>Personas en renuncia<\/span><strong>24<\/strong>/);
  assert.doesNotMatch(html, /Renuncias pendientes/);
  assert.match(html, /<div class="cuadro-bolsa-solo">[\s\S]*<h2>Avisos<\/h2>/);
});

test("P-WEB-14 encaja B12 y Avisos en dos marcos internos del lienzo R10", async () => {
  const css = await readFile(new URL("./portal.css", import.meta.url), "utf8");
  assert.match(css, /\.cuadro-bolsa-solo\s*\{[^}]*height:\s*100%;[^}]*grid-template-rows:\s*auto minmax\(0, 1\.25fr\) minmax\(0, 1fr\)/);
  assert.match(css, /\.cuadro-bolsa-solo > \.panel > \.tabla-contenedor\s*\{[^}]*min-height:\s*0;[^}]*overflow:\s*auto/);
});
