import assert from "node:assert/strict";
import test from "node:test";

import { validarPreparacionCoberturaVigente, validarPropuestaCobertura } from "./contrato-cobertura.js";
import { renderizarViasPreparacion, traduccionesPreparacionDisponibles } from "./vias-preparacion-presentacion.js";
import { renderizarAlta } from "./vista-expedientes-render.js";
import { montarFormularioCobertura } from "./formulario-cobertura.js";

const HUELLA = "a".repeat(64);
const MENSAJES = {
  cobertura_preparacion_titulo: "Preparación",
  cobertura_preparacion_bolsa: "Por bolsa de trabajo",
  cobertura_preparacion_sae: "Por oferta al SAE",
  cobertura_preparacion_documentos: "Documentos",
  cobertura_preparacion_datos: "Datos",
  cobertura_preparacion_ejemplo: "Muestra pendiente de validación por RRHH",
  cobertura_preparacion_pendiente: "Pendiente",
  cobertura_preparacion_no_disponible: "Datos no disponibles",
  cobertura_preparacion_antes_alta: "Registre la petición para consultar la relación",
  cobertura_preparacion_sae_solo_lista: "Consulta de preparación; la oferta SAE aún no se tramita aquí",
  cobertura_preparacion_sin_elementos: "No se piden elementos adicionales",
  "contratacion_temporal.cobertura.doc.ficha_preparacion_cobertura": "Ficha de preparación",
  "contratacion_temporal.cobertura.dato.categoria": "Categoría",
  "contratacion_temporal.cobertura.doc.nota_informativa_sae": "Nota informativa SAE",
  "contratacion_temporal.cobertura.dato.descripcion_puesto": "Descripción del puesto",
};
const t = (clave) => {
  if (!Object.hasOwn(MENSAJES, clave)) throw new Error("traducción ausente");
  return MENSAJES[clave];
};

function evaluacion(via_clave, prioridad) {
  return { via_clave, prioridad, estado: "viable", resultados_omitidos: [],
    ausencias_bloqueantes: [], ausencias_admitidas: [], no_habilitantes: [], conflictos: [] };
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
    catalogo: { referencia: "catalogo:ct:preparacion:v2", version: 2,
      huella_sha256: HUELLA, es_ejemplo: true, vias: [
        { clave: "bolsa_vigente", orden: 1,
          documentos: [{ clave: "ficha", orden: 1, clave_i18n: "contratacion_temporal.cobertura.doc.ficha_preparacion_cobertura" }],
          datos: [{ clave: "categoria", orden: 1, clave_i18n: "contratacion_temporal.cobertura.dato.categoria" }] },
        { clave: "oferta_sae", orden: 2,
          documentos: [{ clave: "nota_sae", orden: 1, clave_i18n: "contratacion_temporal.cobertura.doc.nota_informativa_sae" }],
          datos: [{ clave: "descripcion_puesto", orden: 1, clave_i18n: "contratacion_temporal.cobertura.dato.descripcion_puesto" }] },
      ] },
  };
}

test("V1 conserva su contrato y no recibe una relación inventada", () => {
  const v1 = propuestaV2();
  v1.esquema = "vec.contratacion-temporal.propuesta-cobertura.v1";
  delete v1.catalogo;
  const validada = validarPropuestaCobertura(v1);
  assert.equal(validada.catalogo, undefined);
});

test("V2 presenta solo documentos y datos de cada vía autorizada", () => {
  const validada = validarPropuestaCobertura(propuestaV2());
  assert.equal(traduccionesPreparacionDisponibles(validada, t), true);
  const html = renderizarViasPreparacion(validada, t);
  assert.match(html, /Por bolsa de trabajo/u);
  assert.match(html, /Por oferta al SAE/u);
  assert.match(html, /Muestra pendiente de validación/u);
  assert.match(html, /Ficha de preparación/u);
  assert.match(html, /Nota informativa SAE/u);
  assert.match(html, /<details[^>]*data-ct-preparacion-via="oferta_sae"[^>]*>\s*<summary>Por oferta al SAE<\/summary>/u);
  assert.doesNotMatch(html, /catalogo:ct:|sha256|descripcion_puesto/u);
  assert.doesNotMatch(html, /<form|type="submit"|data-ct-cobertura-form/u);
});

test("V2 rechaza proyección parcial, vía desconocida y traducción no publicada", () => {
  const sinDatos = propuestaV2();
  delete sinDatos.catalogo.vias[1].datos;
  assert.throws(() => validarPropuestaCobertura(sinDatos), /contrato cerrado/u);
  const sinVia = propuestaV2();
  sinVia.catalogo.vias.pop();
  assert.throws(() => validarPropuestaCobertura(sinVia), /catálogo de preparación/u);
  const viaAjena = propuestaV2();
  viaAjena.catalogo.vias[1].clave = "via_ajena";
  assert.throws(() => validarPropuestaCobertura(viaAjena), /vía de preparación/u);
  const sinTraduccion = propuestaV2();
  sinTraduccion.catalogo.vias[1].datos[0].clave_i18n = "contratacion_temporal.cobertura.dato.no_publicado";
  assert.equal(traduccionesPreparacionDisponibles(validarPropuestaCobertura(sinTraduccion), t), false);
});

test("antes del GET el alta no atribuye datos al servidor", () => {
  const alta = renderizarAlta(t, true, false, false, false, false, false);
  assert.match(alta, /data-ct-exp-alta/u);
  assert.match(alta, /data-ct-exp-preparacion/u);
  assert.doesNotMatch(alta, /Por oferta al SAE|Nota informativa SAE/u);
});

test("GET vigente valida sobre exacto y reutiliza catálogo cerrado", () => {
  const respuesta = { esquema: "vec.contratacion-temporal.preparacion-cobertura.v1",
    catalogo: propuestaV2().catalogo };
  const validada = validarPreparacionCoberturaVigente(respuesta);
  assert.equal(validada.catalogo.es_ejemplo, true);
  assert.match(renderizarViasPreparacion(validada, t), /Nota informativa SAE/u);
  assert.equal(Object.isFrozen(validada.catalogo.vias[0]), true);
  assert.throws(() => validarPreparacionCoberturaVigente({ ...respuesta, extra: true }), /contrato cerrado/u);
  const parcial = structuredClone(respuesta);
  parcial.catalogo.vias.pop();
  assert.throws(() => validarPreparacionCoberturaVigente(parcial), /catálogo de preparación/u);
});

test("el formulario consume V2 y oculta una clave nominal sin traducción", async () => {
  const raiz = () => ({ innerHTML: "", eventos: new Map(),
    addEventListener(tipo, manejador) { this.eventos.set(tipo, manejador); },
    removeEventListener(tipo) { this.eventos.delete(tipo); },
    querySelector() { return null; }, contains() { return true; },
    replaceChildren() { this.innerHTML = ""; } });
  for (const [respuesta, visible] of [
    [propuestaV2(), true],
    [(() => { const dato = propuestaV2();
      dato.catalogo.vias[1].datos[0].clave_i18n = "contratacion_temporal.cobertura.dato.no_publicado";
      return dato; })(), false],
  ]) {
    const elemento = raiz();
    const desmontar = montarFormularioCobertura({
      raiz: elemento,
      cliente: { async proponerCobertura() { return respuesta; },
        async decidirCobertura() { throw new Error("no se decide"); },
        async consultarResultadoCobertura() { throw new Error("no se consulta"); } },
      contexto: { expediente_ref: "expediente:ct:prueba:001", version_esperada: 2 },
      mensajes: MENSAJES, etiquetasVias: async () => new Map(),
    });
    await new Promise((resolver) => setImmediate(resolver));
    assert.equal(elemento.innerHTML.includes("data-ct-preparacion-vias"), visible);
    assert.equal(elemento.innerHTML.includes("Nota informativa SAE"), visible);
    if (!visible) assert.match(elemento.innerHTML, /data-ct-cobertura-error/u);
    desmontar();
  }
});
