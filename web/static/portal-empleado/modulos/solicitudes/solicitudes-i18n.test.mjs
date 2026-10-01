import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { cargarTextos } from "../../../comun/textos.js";
import { INDICE_IDIOMAS, localizacionDe } from "../../../comun/idioma.js";
import { crearTraductorSolicitudes, formatearFechaSolicitudes } from "./i18n.js";
import { renderizarSolicitudes } from "./vista.js";

const datos = {
  tramites: [{ referencia: "SOL-001", titulo: "<Request>", fecha: "2026-09-30T22:30:00Z", estado: "en_revision", historial: [{ fecha: "2026-02-31", titulo: "<Milestone>" }] }],
  catalogo: [{ titulo: "<Type>" }], certificados: [{ tipo: "<Certificate>" }],
};

for (const { codigo, localizacion } of INDICE_IDIOMAS.idiomas) {
  test(`catálogo común completo y fechas Intl en ${codigo}`, async () => {
    const textos = await cargarTextos("solicitudes", { idioma: codigo });
    assert.deepEqual(textos.faltantes, []);
    const mensajes = textos.seccion("general");
    const t = crearTraductorSolicitudes(mensajes, localizacion);
    assert.equal(t.numero(12345), new Intl.NumberFormat(localizacion).format(12345));
    const fecha = new Intl.DateTimeFormat(localizacion, { day: "2-digit", month: "2-digit", year: "numeric", timeZone: "Europe/Madrid" }).format(new Date(datos.tramites[0].fecha));
    const html = renderizarSolicitudes({ situacion: "disponible", datos }, mensajes, localizacion);
    assert.ok(html.includes(mensajes.titulo));
    assert.ok(html.includes(mensajes.estado_en_revision));
    assert.ok(html.includes(fecha));
    assert.match(html, /&lt;Request&gt;/);
    const ficha = renderizarSolicitudes({ situacion: "disponible", datos, pestana: "seguimiento", seleccionada: "SOL-001" }, mensajes, localizacion);
    assert.match(ficha, /&lt;Milestone&gt;/);
    assert.ok(ficha.includes(`<time>${mensajes.sin_dato}</time>`));
    for (const pestana of ["nueva", "seguimiento", "certificados"]) {
      const contenido = renderizarSolicitudes({ situacion: "disponible", datos, pestana, seleccionada: "SOL-001" }, mensajes, localizacion);
      assert.match(contenido, /disabled aria-disabled="true"/);
      assert.ok(contenido.includes(mensajes[pestana === "nueva" ? "iniciar_motivo" : pestana === "seguimiento" ? "aportar_motivo" : "emitir_motivo"]));
    }
  });
}

test("fechas imposibles o ausentes no se presentan como válidas", () => {
  const localizacion = localizacionDe(INDICE_IDIOMAS.porDefecto);
  for (const valor of [undefined, "", "30/09/2026", "2026-02-31", "2026-13-01", "2026-09-20Tinválido"]) {
    assert.equal(formatearFechaSolicitudes(valor, localizacion, "missing"), "missing");
  }
  assert.throws(() => crearTraductorSolicitudes()("clave_desconocida"), /desconocida/);
});

test("el idioma elegido en URL carga textos y anuncios sin inyección de catálogo", async () => {
  for (const { codigo } of INDICE_IDIOMAS.idiomas) {
    const textos = await cargarTextos("solicitudes", { idioma: codigo });
    const salida = JSON.parse(execFileSync(process.execPath, ["--input-type=module", "-e", `
      globalThis.location = { href: ${JSON.stringify(`http://localhost/?lang=${codigo}`)} };
      const { renderizarSolicitudes } = await import(${JSON.stringify(new URL("vista.js", import.meta.url).href)});
      const { crearTraductorSolicitudes } = await import(${JSON.stringify(new URL("i18n.js", import.meta.url).href)});
      console.log(JSON.stringify({ html: renderizarSolicitudes(), anuncio: crearTraductorSolicitudes()("anuncio_detalle", { referencia: "SOL-001" }) }));
    `], { encoding: "utf8" }));
    assert.ok(salida.html.includes(textos.seccion("general").titulo));
    assert.equal(salida.anuncio, textos.traducir("general.anuncio_detalle", { referencia: "SOL-001" }));
  }
});

test("las hojas de Solicitudes usan catálogos comunes sin diccionarios ni idioma fijo", async () => {
  for (const archivo of ["i18n.js", "vista.js"]) {
    const codigo = await readFile(new URL(archivo, import.meta.url), "utf8");
    assert.doesNotMatch(codigo, /["'](?:es(?:-ES)?|en(?:-GB)?)["']/);
    assert.doesNotMatch(codigo, /titulo:\s*["']|localStorage|sessionStorage|document\.cookie|indexedDB/);
  }
});
