import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { crearSuperficieConvocatoriasS1 } from "./portal-vistas-convocatorias.js";
import { crearTraductorConvocatoriasS1, MENSAJES_CONVOCATORIAS_S1_ES } from "./portal-i18n-convocatorias.js";

test("la vista resuelve el catálogo S1 con la versión de caché F2", async () => {
  const codigo = readFileSync(new URL("./portal-vistas-convocatorias.js", import.meta.url), "utf8");
  assert.match(codigo, /from "\.\/portal-i18n-convocatorias\.js\?v=20260924-f2-web2"/);
  const catalogo = await import("./portal-i18n-convocatorias.js?v=20260924-f2-web2");
  assert.equal(catalogo.traducirConvocatoriasS1("titulo"), "Convocatorias, bases y calendario");
});

function contenedorFalso() {
  const oyentes = new Map();
  return {
    innerHTML: "",
    destinos: [],
    addEventListener(nombre, oyente) { oyentes.set(nombre, oyente); },
    removeEventListener(nombre) { oyentes.delete(nombre); },
    replaceChildren() { this.innerHTML = ""; },
    contains() { return true; },
    pulsar(referencia) {
      oyentes.get("click")?.({ target: { closest: (selector) => selector === "[data-s1-convocatoria]" ? { dataset: { s1Convocatoria: referencia } } : null } });
    },
    pulsarSeccion(seccion) {
      oyentes.get("click")?.({ target: { closest: (selector) => selector === "[data-s1-seccion]" ? { dataset: { s1Seccion: seccion } } : null } });
    },
    querySelector(selector) {
      this.destinos.push(selector);
      return { scrollIntoView: () => this.destinos.push(`scroll:${selector}`), focus: () => this.destinos.push(`focus:${selector}`) };
    },
    get oyentes() { return oyentes.size; },
  };
}

const lista = [{ referencia: "conv-1", titulo: "Auxiliar", categoria: "Administración", estado: "Publicada", version_actual: "v2", cierre_plazo: "2026-10-15" }];
const detalle = {
  referencia: "conv-1", titulo: "Auxiliar", resumen: "Bolsa de personal auxiliar", categoria: "Administración", estado: "Publicada",
  identificador_publico: "auxiliar-2026", version_actual: "v2",
  versiones: [{ codigo: "v1", estado: "Sustituida", publicada_en: "2026-08-01" }, { codigo: "v2", estado: "Publicada", publicada_en: "2026-09-01" }],
  bases: {
    codigo: "v2", resumen: "Bases de la versión publicada", requisitos: [{ referencia: "titulacion", titulo: "Titulación", descripcion: "Título exigido por las bases", obligatorio: true, hito_exigibilidad: "Fin del plazo" }],
    hitos: [{ titulo: "Fin de solicitudes", fecha: "2026-10-15T23:59:00+02:00" }],
    documentos: [{ titulo: "Bases publicadas", referencia: "BOP-123" }],
  },
};

test("sin conector muestra no configurado y no inventa convocatorias", async () => {
  const contenedor = contenedorFalso();
  const vista = crearSuperficieConvocatoriasS1({ contenedor });
  await vista.montar();
  assert.match(contenedor.innerHTML, /Consulta no configurada/);
  assert.doesNotMatch(contenedor.innerHTML, /Auxiliar|DEMO/);
  vista.desmontar();
  assert.equal(contenedor.oyentes, 0);
  assert.equal(contenedor.innerHTML, "");
});

test("consulta versiones, bases, requisitos, hitos y documentos sin habilitar efectos", async () => {
  const contenedor = contenedorFalso();
  const vista = crearSuperficieConvocatoriasS1({ contenedor, consultarLista: async () => lista, consultarDetalle: async () => detalle });
  await vista.montar();
  for (const texto of ["Auxiliar", "v1", "v2", "Bases de la versión publicada", "Titulación", "Fin del plazo", "Fin de solicitudes", "Bases publicadas", "BOP-123"]) {
    assert.ok(contenedor.innerHTML.includes(texto), texto);
  }
  assert.match(contenedor.innerHTML, /data-s1-convocatoria="conv-1"/);
  assert.match(contenedor.innerHTML, /aria-current="true"/);
  assert.match(contenedor.innerHTML, /<details class="s1-ayuda">/);
  for (const apartado of ["resumen", "bases", "versiones", "requisitos", "hitos", "documentos"]) {
    assert.match(contenedor.innerHTML, new RegExp(`data-s1-seccion="${apartado}"`));
    assert.match(contenedor.innerHTML, new RegExp(`id="s1-seccion-${apartado}"`));
  }
  assert.equal((contenedor.innerHTML.match(/disabled aria-disabled="true"/g) || []).length, 3);
  assert.doesNotMatch(contenedor.innerHTML, /DEMO|data-operacion=/);
  vista.desmontar();
});

test("la navegación local desplaza y enfoca apartados sin cambiar la URL", async () => {
  const contenedor = contenedorFalso();
  const vista = crearSuperficieConvocatoriasS1({ contenedor, consultarLista: async () => lista, consultarDetalle: async () => detalle });
  await vista.montar();
  contenedor.pulsarSeccion("requisitos");
  assert.ok(contenedor.destinos.includes("scroll:#s1-seccion-requisitos"));
  assert.ok(contenedor.destinos.includes("focus:#s1-seccion-requisitos"));
  assert.doesNotMatch(contenedor.innerHTML, /href="#s1-/);
  vista.desmontar();
});

test("vacío, denegación, error y contrato inválido quedan explícitos", async () => {
  const casos = [
    { consultarLista: async () => [], espera: "Sin convocatorias" },
    { consultarLista: async () => { throw { status: 403 }; }, espera: "Acceso denegado" },
    { consultarLista: async () => { throw new Error("secreto interno"); }, espera: "Consulta no disponible" },
    { consultarLista: async () => [{ referencia: "x", titulo: "X" }, { referencia: "x", titulo: "Y" }], espera: "Consulta no disponible" },
  ];
  for (const caso of casos) {
    const contenedor = contenedorFalso();
    await crearSuperficieConvocatoriasS1({ contenedor, ...caso }).montar();
    assert.ok(contenedor.innerHTML.includes(caso.espera));
    assert.doesNotMatch(contenedor.innerHTML, /secreto interno/);
  }
});

test("detalle inválido o denegado no se mezcla con datos anteriores", async () => {
  const contenedor = contenedorFalso();
  const vista = crearSuperficieConvocatoriasS1({
    contenedor,
    consultarLista: async () => [...lista, { ...lista[0], referencia: "conv-2", titulo: "Otra" }],
    consultarDetalle: async (referencia) => {
      if (referencia === "conv-2") throw { status: 403 };
      return detalle;
    },
  });
  await vista.montar();
  contenedor.pulsar("conv-2");
  await new Promise((resolve) => setImmediate(resolve));
  assert.match(contenedor.innerHTML, /Acceso denegado/);
  assert.doesNotMatch(contenedor.innerHTML, /Bases publicadas/);
  vista.desmontar();
});

test("un detalle tardío no sustituye la selección nueva ni revive tras desmontar", async () => {
  const contenedor = contenedorFalso();
  let resolverPrimero;
  const primero = new Promise((resolve) => { resolverPrimero = resolve; });
  const vista = crearSuperficieConvocatoriasS1({
    contenedor,
    consultarLista: async () => [...lista, { ...lista[0], referencia: "conv-2", titulo: "Otra" }],
    consultarDetalle: async (referencia) => referencia === "conv-1" ? primero : {
      ...detalle, referencia: "conv-2", titulo: "Otra convocatoria", resumen: "Segunda convocatoria", bases: { ...detalle.bases, codigo: "v3" },
    },
  });
  const montaje = vista.montar();
  await new Promise((resolve) => setImmediate(resolve));
  contenedor.pulsar("conv-2");
  await new Promise((resolve) => setImmediate(resolve));
  resolverPrimero(detalle);
  await montaje;
  assert.match(contenedor.innerHTML, /Otra convocatoria/);
  assert.doesNotMatch(contenedor.innerHTML, /Bolsa de personal auxiliar/);
  vista.desmontar();
  assert.equal(contenedor.innerHTML, "");
});

test("escapa datos de la fuente y conserva i18n y CSS R10", async () => {
  const contenedor = contenedorFalso();
  const vista = crearSuperficieConvocatoriasS1({
    contenedor,
    consultarLista: async () => [{ ...lista[0], titulo: '<img src=x onerror="alert(1)">' }],
    consultarDetalle: async () => ({ ...detalle, titulo: '<script>alert(1)</script>', bases: { ...detalle.bases, documentos: [{ titulo: '<img src=x>', referencia: "doc" }] } }),
  });
  await vista.montar();
  assert.doesNotMatch(contenedor.innerHTML, /<script>|<img src=x/);
  assert.match(contenedor.innerHTML, /&lt;script&gt;/);
  assert.match(contenedor.innerHTML, /&lt;img src=x/);
  assert.equal(typeof crearTraductorConvocatoriasS1(MENSAJES_CONVOCATORIAS_S1_ES)("titulo"), "string");
  const css = readFileSync(new URL("./portal-convocatorias.css", import.meta.url), "utf8");
  assert.match(css, /@media \(min-width: 1024px\)[\s\S]*max-height:[^;]+; overflow: auto/);
  assert.match(css, /@media \(max-width: 480px\)/);
  assert.match(css, /:focus-visible/);
  assert.doesNotMatch(css, /#[0-9a-fA-F]{3,8}\b/);
  vista.desmontar();
});
