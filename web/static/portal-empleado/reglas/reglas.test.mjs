import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { versionDe } from "../versiones-cache.test-helper.mjs";
import {
  API_REGLAS, ErrorReglas, crearCliente, filtrar, mensajeError, origenRegla, renderizarCatalogo, renderizarResumen,
  validarReglas, valorRegla,
} from "./reglas.js";
import { MENSAJES_REGLAS_ES, crearTraductorReglas } from "./i18n.js";

const leer = (nombre) => readFileSync(new URL(nombre, import.meta.url), "utf8");

function regla(extra = {}) {
  return { clave: "b05.plazo_respuesta", etiqueta: "Plazo de respuesta", descripcion: "Un día hábil.", unidad: "dias_habiles",
    cantidad: 1, computo: "administrativo", inicio: "contacto_efectivo", origen: "ejemplo", norma: "Supuesto de trabajo.",
    duda: "Duda 1 de RRHH.", version: 1, referencia: "vec.bolsa.reglas:1:b05.plazo_respuesta", paquete_ejemplo: true, ...extra };
}

function respuesta() {
  return { data: { esquema: "vec.reglas.vigentes.v1", catalogos: [
    { modulo: "bolsa", catalogo_id: "vec.bolsa.reglas", estado: "disponible", version: 1, huella_sha256: "a".repeat(64), paquete_ejemplo: true,
      reglas: [regla(), regla({ clave: "b04.franja_llamadas", etiqueta: "Franja <b>", unidad: "franja_horaria", cantidad: undefined, computo: undefined,
        valor: "09:00-14:00", origen: "reglamento", articulo: "art. 8.2.a", duda: "Sin duda abierta." })] },
    { modulo: "contratacion_temporal", catalogo_id: "vec.contratacion_temporal.reglas", estado: "sin_catalogo", paquete_ejemplo: false, reglas: [] },
  ] } };
}

test("el catálogo i18n está completo y toda clave de la página existe", () => {
  const t = crearTraductorReglas();
  assert.throws(() => crearTraductorReglas({ titulo: "x" }), /incompleto/u);
  assert.throws(() => t("desconocida"), /desconocida/u);
  const html = leer("./index.html");
  for (const [, clave] of html.matchAll(/data-i18n(?:-label)?="([^"]+)"/gu)) assert.ok(Object.hasOwn(MENSAJES_REGLAS_ES, clave), clave);
  const js = leer("./reglas.js");
  for (const [, id] of js.matchAll(/\$\("([a-z-]+)"\)/gu)) assert.match(html, new RegExp(`id="${id}"`, "u"), id);
  assert.ok(!/style=|<script>/u.test(html), "sin estilos ni guiones en línea");
  assert.match(html, /id="rg-ayuda-abrir"[^>]*>\?</u, "la ayuda solo se abre con «?»");
});

test("las versiones en caché se renuevan juntas y la pantalla está en el manifiesto interno", () => {
  const html = leer("./index.html");
  const js = leer("./reglas.js");
  assert.equal(versionDe(html, "reglas.js"), versionDe(html, "reglas.css"));
  assert.equal(versionDe(js, "i18n.js"), versionDe(html, "i18n.js"));
  assert.equal(versionDe(js, "i18n.js"), versionDe(html, "reglas.js"));
  for (const nombre of ["interno.manifest", "produccion.manifest"]) {
    const manifiesto = readFileSync(new URL(`../../../${nombre}`, import.meta.url), "utf8");
    for (const f of ["index.html", "reglas.js", "reglas.css", "i18n.js"]) assert.match(manifiesto, new RegExp(`static/portal-empleado/reglas/${f.replace(".", "\\.")}`, "u"), `${nombre}: ${f}`);
  }
});

test("valida el contrato y rechaza lo que no encaja", () => {
  const d = validarReglas(respuesta());
  assert.equal(d.catalogos.length, 2);
  assert.throws(() => validarReglas({ data: { esquema: "otro", catalogos: [] } }), ErrorReglas);
  const sinArticulo = respuesta();
  sinArticulo.data.catalogos[0].reglas[1].articulo = undefined;
  assert.throws(() => validarReglas(sinArticulo), ErrorReglas);
  const ejemploConArticulo = respuesta();
  ejemploConArticulo.data.catalogos[0].reglas[0].articulo = "art. 1";
  assert.throws(() => validarReglas(ejemploConArticulo), ErrorReglas);
  const sinCatalogoConReglas = respuesta();
  sinCatalogoConReglas.data.catalogos[1].reglas = [regla()];
  assert.throws(() => validarReglas(sinCatalogoConReglas), ErrorReglas);
});

test("pinta valor, unidad, origen, duda y versión, escapando el contenido", () => {
  const d = validarReglas(respuesta());
  const html = renderizarCatalogo(d.catalogos[0]);
  assert.match(html, /Reglamento, art\. 8\.2\.a/u);
  assert.match(html, /09:00–14:00/u);
  assert.match(html, /Días hábiles/u);
  assert.match(html, /Cómputo administrativo/u);
  assert.match(html, /Duda 1 de RRHH\./u);
  assert.match(html, /Paquete de ejemplo/u);
  assert.match(html, /Franja &lt;b&gt;/u);
  assert.ok(!html.includes("<b>"));
  assert.match(renderizarCatalogo(d.catalogos[1]), /Sin catálogo de reglas cargado/u);
  assert.equal(valorRegla({ unidad: "ninguna" }), "No aplica");
  assert.equal(valorRegla({ unidad: "lista", valor: "a,b" }), "a, b");
  assert.equal(origenRegla({ origen: "ejemplo" }), "Ejemplo");
  const resumen = renderizarResumen(d.catalogos);
  assert.match(resumen, /Reglas vigentes<\/span><strong>2</u);
  assert.match(resumen, /Del Reglamento<\/span><strong>1</u);
});

test("filtra por módulo, origen y texto", () => {
  const d = validarReglas(respuesta());
  assert.equal(filtrar(d.catalogos, { modulo: "bolsa" }).length, 1);
  assert.equal(filtrar(d.catalogos, { origen: "reglamento" })[0].reglas.length, 1);
  assert.equal(filtrar(d.catalogos, { texto: "DUDA 1" })[0].reglas.length, 1);
});

test("el cliente pide solo lectura, sin identidad propia, y traduce los errores", async () => {
  let pedido;
  const ok = crearCliente(async (url, opciones) => {
    pedido = { url, opciones };
    return new Response(JSON.stringify(respuesta()), { status: 200 });
  });
  await ok.reglas();
  assert.equal(pedido.url, API_REGLAS);
  assert.equal(pedido.opciones.method, "GET");
  assert.deepEqual(Object.keys(pedido.opciones.headers), ["Accept"]);
  for (const [estado, codigo] of [[401, "error_autenticacion_requerida"], [403, "error_acceso_denegado"], [503, "error_servicio_no_disponible"]]) {
    const cliente = crearCliente(async () => new Response("{}", { status: estado }));
    await assert.rejects(cliente.reglas(), (e) => e.codigo === codigo && mensajeError(e).length > 0);
  }
  const roto = crearCliente(async () => new Response("no json", { status: 200 }));
  await assert.rejects(roto.reglas(), (e) => e.codigo === "error_respuesta");
  const caido = crearCliente(async () => { throw new TypeError("red"); });
  await assert.rejects(caido.reglas(), (e) => e.codigo === "error_servicio_no_disponible");
});
