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

test("los ejemplos conservan sus referencias y permanecen fuera de la fuente de consulta", async () => {
  const { DATOS_SOLICITUDES_PRESENTACION: ejemplos } = await import("./datos-presentacion.js");
  assert.deepEqual(ejemplos.tramites.map(({ id, fecha }) => [id, fecha]), [
    ["SOL-2026-00184", "17/09/2026"], ["SOL-2026-00167", "12/09/2026"],
    ["SOL-2026-00121", "03/09/2026"], ["SOL-2026-00098", "28/08/2026"],
  ]);
  assert.deepEqual(ejemplos.catalogo.map(({ id }) => id), ["servicios", "permiso", "accion-social", "compatibilidad"]);
  assert.deepEqual(ejemplos.certificados.map(({ referencia }) => referencia), ["CERT-2026-041", "CERT-2026-039", "CERT-2026-031"]);
  for (const valor of [ejemplos, ejemplos.tramites, ejemplos.catalogo, ejemplos.certificados, ...ejemplos.tramites, ...ejemplos.catalogo, ...ejemplos.certificados]) assert.ok(Object.isFrozen(valor));
  for (const { codigo } of INDICE_IDIOMAS.idiomas) {
    const textos = await cargarTextos("solicitudes-ejemplos", { idioma: codigo });
    assert.deepEqual(textos.faltantes, []);
    assert.ok(textos.traducir("presentacion.tramites.item_0.tipo"));
  }
  assert.doesNotMatch(renderizarSolicitudes(), /SOL-2026|CERT-2026|Antonio López/);
});
