import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { validarPropuestaCobertura } from "./contrato-cobertura.js";
import { validarCatalogosAlta } from "./contrato.js";
import {
  preparacionPresentable, renderizarViasPreparacion, selectorPestanaPreparacion, viaPreparacionDeEvento,
} from "./vias-preparacion-cobertura.js";
import { renderizarAlta } from "./vista-expedientes-render.js?v=20261001-ct-a-i18n-v1";
import { montarFormularioCobertura } from "./formulario-cobertura.js?v=20261001-ct-a-i18n-v1";

const HUELLA = "a".repeat(64);
const t = (clave) => clave;

function evaluacion(via_clave, prioridad) {
  return { via_clave, prioridad, estado: "viable", resultados_omitidos: [],
    ausencias_bloqueantes: [], ausencias_admitidas: [], no_habilitantes: [], conflictos: [] };
}

function catalogoV2() {
  return { referencia: "catalogo:ct:preparacion:v2", version: 2,
    huella_sha256: HUELLA, es_ejemplo: true, vias: [
      { clave: "bolsa_vigente", orden: 1,
        documentos: [{ clave: "ficha", orden: 1, clave_i18n: "contratacion_temporal.cobertura.doc.ficha_preparacion_cobertura" }],
        datos: [{ clave: "categoria", orden: 1, clave_i18n: "contratacion_temporal.cobertura.dato.categoria" }] },
      { clave: "oferta_sae", orden: 2,
        documentos: [{ clave: "nota_sae", orden: 1, clave_i18n: "contratacion_temporal.cobertura.doc.nota_informativa_sae" }],
        datos: [{ clave: "descripcion_puesto", orden: 1, clave_i18n: "contratacion_temporal.cobertura.dato.descripcion_puesto" }] },
    ] };
}

function propuestaV2() {
  return {
    esquema: "vec.contratacion-temporal.propuesta-cobertura.v2",
    estado: "viable", via_recomendada: "bolsa_vigente",
    evaluaciones: [evaluacion("bolsa_vigente", 1), evaluacion("oferta_sae", 2)],
    identidad_semantica: { referencia: `propuesta-cobertura-semantica:sha256:${HUELLA}`,
      huella_sha256: HUELLA, canon: {
        dominio: "vec.dipgra.contratacion-temporal.propuesta-decision-cobertura-semantica",
        version_esquema: 1, algoritmo: "sha-256" } },
    catalogo: catalogoV2(),
  };
}

function catalogosAlta(extra = {}) {
  return { esquema: "vec.contratacion_temporal.catalogos_alta.v1",
    centros: [{ referencia: "centro:rpt:001", etiqueta: "Centro", contactos: [{ referencia: "contacto:rpt:001", etiqueta: "Contacto" }] }],
    categorias: [{ referencia: "categoria:rpt:c2", etiqueta: "Auxiliar", grupos_subgrupos: [{ clave: "C2", etiqueta: "Grupo C2" }] }],
    motivos: [{ clave: "sustitucion", etiqueta: "Sustitución" }], documentos: [], ...extra };
}

test("V1 conserva su contrato y no recibe una relación inventada", () => {
  const v1 = propuestaV2();
  v1.esquema = "vec.contratacion-temporal.propuesta-cobertura.v1";
  delete v1.catalogo;
  assert.equal(validarPropuestaCobertura(v1).catalogo, undefined);
});

test("dos pestañas accesibles: bolsa abierta, SAE cerrada y solo informativa", () => {
  const catalogo = validarPropuestaCobertura(propuestaV2()).catalogo;
  assert.equal(preparacionPresentable(catalogo), true);
  const html = renderizarViasPreparacion(catalogo, { prefijo: "ct-prueba" });
  assert.match(html, /role="tablist"/u);
  assert.match(html, /<button type="button" role="tab" id="ct-prueba-pestana-bolsa_vigente"[^>]*aria-selected="true" tabindex="0"[^>]*>Por bolsa de trabajo<\/button>/u);
  assert.match(html, /id="ct-prueba-pestana-oferta_sae"[^>]*aria-selected="false" tabindex="-1"[^>]*>Por oferta al SAE<\/button>/u);
  assert.match(html, /id="ct-prueba-panel-oferta_sae"[^>]*data-ct-preparacion-via="oferta_sae" hidden>/u);
  assert.match(html, /Lista de ejemplo, pendiente de que RRHH la confirme/u);
  assert.match(html, /Ficha de preparación de la cobertura/u);
  assert.match(html, /Nota informativa sobre la oferta al SAE/u);
  assert.match(html, /La oferta al SAE no se prepara ni se envía desde aquí/u);
  assert.doesNotMatch(html, /catalogo:ct:|sha256|descripcion_puesto|contratacion_temporal\./u);
  assert.doesNotMatch(html, /<form|type="submit"|data-ct-cobertura-form/u);
  const sae = renderizarViasPreparacion(catalogo, { prefijo: "ct-prueba", seleccionada: "oferta_sae" });
  assert.match(sae, /id="ct-prueba-pestana-oferta_sae"[^>]*aria-selected="true"/u);
  assert.match(sae, /id="ct-prueba-panel-bolsa_vigente"[^>]*hidden>/u);
});

test("las flechas, Inicio y Fin mueven entre pestañas; otros eventos no", () => {
  const evento = (type, via, key) => ({ type, key,
    target: { closest: (selector) => (selector === "[data-ct-preparacion-pestana]"
      ? { dataset: { ctPreparacionPestana: via } } : null) } });
  assert.equal(viaPreparacionDeEvento(evento("click", "oferta_sae")), "oferta_sae");
  assert.equal(viaPreparacionDeEvento(evento("keydown", "bolsa_vigente", "ArrowRight")), "oferta_sae");
  assert.equal(viaPreparacionDeEvento(evento("keydown", "bolsa_vigente", "ArrowLeft")), "oferta_sae");
  assert.equal(viaPreparacionDeEvento(evento("keydown", "oferta_sae", "Home")), "bolsa_vigente");
  assert.equal(viaPreparacionDeEvento(evento("keydown", "bolsa_vigente", "End")), "oferta_sae");
  assert.equal(viaPreparacionDeEvento(evento("keydown", "bolsa_vigente", "Enter")), null);
  assert.equal(viaPreparacionDeEvento(evento("click", "via_ajena")), null);
  assert.equal(viaPreparacionDeEvento({ type: "click", target: { closest: () => null } }), null);
  assert.equal(selectorPestanaPreparacion("oferta_sae"), '[data-ct-preparacion-pestana="oferta_sae"]');
  assert.equal(selectorPestanaPreparacion("otra"), "");
});

test("una clave sin texto publicado o una vía ausente no se muestran como código", () => {
  const sinTexto = catalogoV2();
  Object.assign(sinTexto.vias[1].datos[0], { clave_i18n: "contratacion_temporal.cobertura.dato.no_publicado" });
  const ajena = catalogoV2();
  Object.assign(ajena.vias[1].datos[0], { clave_i18n: "preparacion.titulo" });
  const sinSae = catalogoV2();
  sinSae.vias.pop();
  for (const catalogo of [sinTexto, ajena, sinSae]) {
    assert.equal(preparacionPresentable(catalogo), false);
    const html = renderizarViasPreparacion(catalogo);
    assert.match(html, /data-ct-preparacion-no-disponible/u);
    assert.doesNotMatch(html, /no_publicado|preparacion\.titulo|role="tab"/u);
  }
});

test("V2 rechaza proyección parcial y vía desconocida", () => {
  const sinDatos = propuestaV2();
  delete sinDatos.catalogo.vias[1].datos;
  assert.throws(() => validarPropuestaCobertura(sinDatos), /contrato cerrado/u);
  const sinVia = propuestaV2();
  sinVia.catalogo.vias.pop();
  assert.throws(() => validarPropuestaCobertura(sinVia), /catálogo de preparación/u);
  const viaAjena = propuestaV2();
  Object.assign(viaAjena.catalogo.vias[1], { clave: "via_ajena" });
  assert.throws(() => validarPropuestaCobertura(viaAjena), /vía de preparación/u);
});

test("los catálogos del alta admiten la relación por vía y la validan cerrada", () => {
  assert.equal(Object.hasOwn(validarCatalogosAlta(catalogosAlta()), "preparacion_vias"), false);
  const validados = validarCatalogosAlta(catalogosAlta({ preparacion_vias: catalogoV2() }));
  assert.equal(validados.preparacion_vias.es_ejemplo, true);
  assert.equal(Object.isFrozen(validados.preparacion_vias.vias[0]), true);
  // Revalidar lo ya validado (como hace el coordinador) no cambia la forma.
  assert.deepEqual(validarCatalogosAlta(validados), validados);
  const parcial = catalogoV2();
  parcial.vias.pop();
  assert.throws(() => validarCatalogosAlta(catalogosAlta({ preparacion_vias: parcial })), /catálogo de preparación/u);
  const abierta = catalogoV2();
  abierta.vias[0].comprobaciones = ["existe_bolsa_vigente"];
  assert.throws(() => validarCatalogosAlta(catalogosAlta({ preparacion_vias: abierta })), /contrato cerrado/u);
  assert.throws(() => validarCatalogosAlta(catalogosAlta({ extra: true })), /contrato cerrado/u);
});

test("la nueva petición reserva la zona de la relación antes del formulario", () => {
  const alta = renderizarAlta(t, true, false, false, false, false, false);
  assert.ok(alta.indexOf("data-ct-exp-preparacion") < alta.indexOf("data-ct-exp-alta"));
});

test("el catálogo de ejemplo de reglas tiene texto en castellano e inglés para cada elemento", async () => {
  const reglas = JSON.parse(await readFile(new URL("../../../../../data/demo/reglas/ct_reglas.ejemplo.demo.json", import.meta.url), "utf8"));
  const claves = new Set();
  const recorrer = (valor) => {
    if (Array.isArray(valor)) valor.forEach(recorrer);
    else if (valor && typeof valor === "object") {
      for (const atributo of ["documentos", "datos"]) {
        if (typeof valor.atributos?.[atributo] === "string" && String(valor.clave).startsWith("c17.via_cobertura.")) {
          for (const par of valor.atributos[atributo].split(",")) claves.add(par.split(":")[1]);
        }
      }
      Object.values(valor).forEach(recorrer);
    }
  };
  recorrer(reglas);
  assert.ok(claves.size >= 10);
  for (const idioma of ["es", "en"]) {
    const catalogo = JSON.parse(await readFile(new URL(`../../../textos/${idioma}/contratacion-temporal-cobertura.json`, import.meta.url), "utf8"));
    for (const clave of claves) {
      const texto = clave.split(".").reduce((nodo, parte) => nodo?.[parte], catalogo);
      assert.equal(typeof texto, "string", `${idioma}: falta ${clave}`);
    }
  }
});

test("el formulario de la ficha pinta las pestañas, cambia de vía y conserva la propuesta si falta un texto", async () => {
  const raiz = () => ({ innerHTML: "", eventos: new Map(), enfocado: null,
    addEventListener(tipo, manejador) { this.eventos.set(tipo, manejador); },
    removeEventListener(tipo) { this.eventos.delete(tipo); },
    querySelector(selector) { return { focus: () => { this.enfocado = selector; } }; },
    contains() { return true; },
    replaceChildren() { this.innerHTML = ""; } });
  for (const [respuesta, pestanas] of [
    [propuestaV2(), true],
    [(() => { const dato = propuestaV2();
      Object.assign(dato.catalogo.vias[1].datos[0], { clave_i18n: "contratacion_temporal.cobertura.dato.no_publicado" });
      return dato; })(), false],
  ]) {
    const elemento = raiz();
    const desmontar = montarFormularioCobertura({
      raiz: elemento,
      cliente: { async proponerCobertura() { return respuesta; },
        async decidirCobertura() { throw new Error("no se decide"); },
        async consultarResultadoCobertura() { throw new Error("no se consulta"); } },
      contexto: { expediente_ref: "expediente:ct:prueba:001", version_esperada: 2 },
      etiquetasVias: async () => new Map(),
    });
    await new Promise((resolver) => setImmediate(resolver));
    assert.equal(elemento.innerHTML.includes("data-ct-preparacion-vias"), pestanas);
    assert.equal(elemento.innerHTML.includes("data-ct-preparacion-no-disponible"), !pestanas);
    assert.match(elemento.innerHTML, /data-ct-cobertura-form/u);
    assert.doesNotMatch(elemento.innerHTML, /no_publicado/u);
    if (pestanas) {
      elemento.eventos.get("click")({ type: "click", preventDefault() {},
        target: { closest: (selector) => (selector === "[data-ct-preparacion-pestana]"
          ? { dataset: { ctPreparacionPestana: "oferta_sae" } } : null) } });
      assert.match(elemento.innerHTML, /id="ct-preparacion-ficha-pestana-oferta_sae"[^>]*aria-selected="true"/u);
      assert.equal(elemento.enfocado, '[data-ct-preparacion-pestana="oferta_sae"]');
    }
    desmontar();
  }
});

test("cambiar de pestaña en la ficha no borra el motivo ya elegido", async () => {
  const MOTIVO = "eleccion_procedimiento_rrhh";
  const propuesta = propuestaV2();
  propuesta.motivos_alternativa = [{ clave: MOTIVO, via_clave: "oferta_sae",
    etiqueta_i18n: "contratacion_temporal.cobertura.motivo.eleccion_procedimiento_rrhh" }];
  let motivoEnPantalla = "";
  const elemento = { innerHTML: "", eventos: new Map(),
    addEventListener(tipo, manejador) { this.eventos.set(tipo, manejador); },
    removeEventListener(tipo) { this.eventos.delete(tipo); },
    querySelector(selector) {
      return selector === "[name=motivo_clave]" ? { value: motivoEnPantalla } : { focus() {} };
    },
    contains() { return true; },
    replaceChildren() { this.innerHTML = ""; } };
  const desmontar = montarFormularioCobertura({
    raiz: elemento,
    cliente: { async proponerCobertura() { return propuesta; },
      async decidirCobertura() { throw new Error("no se decide"); },
      async consultarResultadoCobertura() { throw new Error("no se consulta"); } },
    contexto: { expediente_ref: "expediente:ct:prueba:001", version_esperada: 2 },
    etiquetasVias: async () => new Map(),
  });
  await new Promise((resolver) => setImmediate(resolver));
  const radio = { value: "oferta_sae" };
  elemento.eventos.get("change")({ target: { closest: (selector) => (selector === "[name=via_elegida]" ? radio : null) } });
  motivoEnPantalla = MOTIVO;
  elemento.eventos.get("click")({ type: "click", preventDefault() {},
    target: { closest: (selector) => (selector === "[data-ct-preparacion-pestana]"
      ? { dataset: { ctPreparacionPestana: "oferta_sae" } } : null) } });
  assert.match(elemento.innerHTML, new RegExp(`value="${MOTIVO}" selected`, "u"));
  desmontar();
});
