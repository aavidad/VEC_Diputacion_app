import assert from "node:assert/strict";
import test from "node:test";

import { renderizarAvisosViaCobertura } from "./avisos-via-cobertura.js";
import { validarAvisosViaCobertura } from "./contrato-avisos-via-cobertura.js";
import { validarPropuestaCobertura } from "./contrato-cobertura.js";
import { montarFormularioCobertura } from "./formulario-cobertura.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";

const HUELLA = "a".repeat(64);
const REGLA_AGOTAMIENTO = {
  clave: "b26.agotamiento", etiqueta: "Agotamiento de la bolsa",
  descripcion: "La bolsa se agota si no quedan integrantes disponibles.", origen: "reglamento",
  articulo: "arts. 3.3 y 7.c", norma: "Reglamento de bolsas", parte_ejemplo: "Circuito de ejemplo.",
  referencia: "vec.bolsa.reglas:1:b26.agotamiento", ejemplo: true,
};
const REGLA_SAE = { ...REGLA_AGOTAMIENTO, clave: "b20.sae_duracion_maxima", etiqueta: "Duración máxima vía SAE", articulo: "art. 3.3", parte_ejemplo: undefined, ejemplo: false };
delete REGLA_SAE.parte_ejemplo;
const REGLA_VIGENCIA = { ...REGLA_SAE, clave: "b25.vigencia_bolsa", etiqueta: "Vigencia de la bolsa", articulo: "arts. 7.b y 3.4" };

function avisos() {
  return {
    esquema: "vec.contratacion-temporal.avisos-via-cobertura.v1",
    estado: "evaluados",
    evaluada_en: "2026-09-25T08:00:00Z",
    avisos: [
      { clave: "bolsa_agotada_provisionalmente", disponibles: 0, integrantes: 3, umbral: 0, reglas: [REGLA_AGOTAMIENTO] },
      { clave: "propuesta_oferta_sae", duracion_maxima_meses: 9, fin_maximo: "2027-07-01", fin_previsto: "2027-12-31", excede_duracion: true, reglas: [REGLA_SAE, REGLA_AGOTAMIENTO] },
      { clave: "nueva_convocatoria", motivos: ["vigencia_superada", "bolsa_agotada"], constituida_en: "2021-02-01", vigencia_hasta: "2026-02-01", reglas: [REGLA_VIGENCIA, REGLA_AGOTAMIENTO] },
    ],
  };
}

function propuesta(conAvisos = true) {
  const base = {
    esquema: "vec.contratacion-temporal.propuesta-cobertura.v1",
    estado: "viable",
    via_recomendada: "bolsa_vigente",
    evaluaciones: [{
      via_clave: "bolsa_vigente", prioridad: 1, estado: "viable",
      resultados_omitidos: [], ausencias_bloqueantes: [],
      ausencias_admitidas: [], no_habilitantes: [], conflictos: [],
    }],
    identidad_semantica: {
      referencia: `propuesta-cobertura-semantica:sha256:${HUELLA}`,
      huella_sha256: HUELLA,
      canon: {
        dominio: "vec.dipgra.contratacion-temporal.propuesta-decision-cobertura-semantica",
        version_esquema: 1,
        algoritmo: "sha-256",
      },
    },
  };
  return conAvisos ? { ...base, avisos_via: avisos() } : base;
}

test("el contrato acepta los avisos y los congela", () => {
  const validados = validarAvisosViaCobertura(avisos());
  assert.equal(validados.avisos.length, 3);
  assert.ok(Object.isFrozen(validados.avisos[0].reglas[0]));
  const conPropuesta = validarPropuestaCobertura(propuesta());
  assert.equal(conPropuesta.avisos_via.avisos[1].excede_duracion, true);
  assert.equal(validarPropuestaCobertura(propuesta(false)).avisos_via, undefined);
});

test("el contrato rechaza avisos incoherentes o con campos de más", () => {
  const casos = [
    (a) => { a.esquema = "otro"; },
    (a) => { a.estado = "agotada"; },
    (a) => { a.estado = "no_disponible"; },
    (a) => { a.avisos[0].disponibles = 5; },
    (a) => { a.avisos[0].nombre = "Persona"; },
    (a) => { a.avisos[1].excede_duracion = "si"; },
    (a) => { a.avisos[2].motivos = ["otro"]; },
    (a) => { a.avisos[2].reglas = []; },
    (a) => { a.avisos[0].reglas[0] = { ...REGLA_AGOTAMIENTO, articulo: "" }; },
    (a) => { a.avisos[1].fin_maximo = "01/07/2027"; },
  ];
  for (const alterar of casos) {
    const entrada = avisos();
    alterar(entrada);
    assert.throws(() => validarAvisosViaCobertura(entrada), TypeError);
  }
});

test("el panel muestra propuestas y deja la procedencia en la ayuda «?»", () => {
  const t = crearTraductorContratacionTemporal();
  const formateadorFechas = new Intl.DateTimeFormat("es-ES", { dateStyle: "long", timeZone: "UTC" });
  const cerrado = renderizarAvisosViaCobertura(validarAvisosViaCobertura(avisos()), t, { formateadorFechas });
  assert.match(cerrado, /Bolsa agotada provisionalmente/);
  assert.match(cerrado, /0 personas disponibles de 3/);
  assert.match(cerrado, /Duración máxima de 9 meses: el nombramiento no podría pasar del 1 de julio de 2027/);
  assert.match(cerrado, /supera esa duración máxima/);
  assert.match(cerrado, /superó su vigencia el 1 de febrero de 2026/);
  assert.match(cerrado, /aria-expanded="false"/);
  assert.match(cerrado, /id="ct-avisos-via-ayuda" hidden/);
  assert.match(cerrado, /<span aria-hidden="true">\?<\/span>/);
  // La procedencia solo está dentro de la ayuda.
  const [fuera] = cerrado.split('<div class="ct-avisos-via-ayuda"');
  assert.doesNotMatch(fuera, /Reglamento de bolsas, art/);
  const abierto = renderizarAvisosViaCobertura(validarAvisosViaCobertura(avisos()), t, { formateadorFechas, ayudaAbierta: true });
  assert.match(abierto, /aria-expanded="true"/);
  assert.match(abierto, /Reglamento de bolsas, arts\. 3\.3 y 7\.c/);
  assert.match(abierto, /Reglamento de bolsas, arts\. 7\.b y 3\.4/);
  assert.match(abierto, /Reglamento de bolsas, art\. 3\.3/);
  assert.match(abierto, /Parte de ejemplo: Circuito de ejemplo\./);
  assert.equal((abierto.match(/data-ct-aviso-via-regla="b26.agotamiento"/g) ?? []).length, 1);
  assert.equal(renderizarAvisosViaCobertura(undefined, t, { formateadorFechas }), "");
  const noDisponible = renderizarAvisosViaCobertura(validarAvisosViaCobertura({ ...avisos(), estado: "no_disponible", avisos: [] }), t, { formateadorFechas });
  assert.match(noDisponible, /no están disponibles ahora/);
  const ejemplo = renderizarAvisosViaCobertura(validarAvisosViaCobertura({ ...avisos(), avisos: [{ ...avisos().avisos[0], reglas: [{ ...REGLA_AGOTAMIENTO, origen: "ejemplo", articulo: "" }] }] }), t, { formateadorFechas, ayudaAbierta: true });
  assert.match(ejemplo, /Regla de ejemplo, sin aprobación de RRHH/);
});

function raizFalsa() {
  const eventos = new Map();
  return {
    innerHTML: "",
    eventos,
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) { if (eventos.get(tipo) === manejador) eventos.delete(tipo); },
    contains() { return true; },
    querySelector() { return { focus() {}, scrollIntoView() {} }; },
    replaceChildren() { this.innerHTML = ""; },
    pulsar(accion) {
      const control = { dataset: { ctCoberturaAccion: accion }, closest: (s) => (s === "[data-ct-cobertura-accion]" ? control : null) };
      return eventos.get("click")({ target: control, preventDefault() {} });
    },
  };
}

test("el formulario de cobertura abre y cierra la ayuda sin tocar la decisión", async () => {
  const raiz = raizFalsa();
  const cliente = {
    async proponerCobertura() { return validarPropuestaCobertura(propuesta()); },
    async decidirCobertura() { throw new Error("decisión inesperada"); },
    async consultarResultadoCobertura() { throw new Error("consulta inesperada"); },
  };
  const desmontar = montarFormularioCobertura({
    raiz, cliente, contexto: { expediente_ref: "expediente:ct:prueba:avisos:001", version_esperada: 2 },
  });
  for (let i = 0; i < 4; i += 1) await Promise.resolve();
  assert.match(raiz.innerHTML, /data-ct-avisos-via/);
  assert.match(raiz.innerHTML, /data-ct-cobertura-evaluacion="bolsa_vigente"/);
  assert.match(raiz.innerHTML, /id="ct-avisos-via-ayuda" hidden/);
  raiz.pulsar("ayuda-avisos-via");
  assert.match(raiz.innerHTML, /aria-expanded="true"/);
  raiz.eventos.get("keydown")({ key: "Escape", preventDefault() {} });
  assert.match(raiz.innerHTML, /aria-expanded="false"/);
  desmontar();
  assert.equal(raiz.eventos.size, 0);
});

test("sin avisos en la propuesta la pantalla queda como hoy", async () => {
  const raiz = raizFalsa();
  const desmontar = montarFormularioCobertura({
    raiz,
    cliente: {
      async proponerCobertura() { return validarPropuestaCobertura(propuesta(false)); },
      async decidirCobertura() {}, async consultarResultadoCobertura() {},
    },
    contexto: { expediente_ref: "expediente:ct:prueba:avisos:002", version_esperada: 2 },
  });
  for (let i = 0; i < 4; i += 1) await Promise.resolve();
  assert.doesNotMatch(raiz.innerHTML, /data-ct-avisos-via/);
  assert.match(raiz.innerHTML, /data-ct-cobertura-evaluacion="bolsa_vigente"/);
  desmontar();
});
