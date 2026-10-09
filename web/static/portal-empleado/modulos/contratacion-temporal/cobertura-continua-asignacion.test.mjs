import assert from "node:assert/strict";
import test from "node:test";

import { crearGestorTramitacion } from "./vista-expedientes-tramitacion.js?v=20261009-asignacion-cobertura-v1";

// Caso del recorrido HZ13: la ficha v2 (fase «solicitud») solo pinta el panel
// de cobertura. Tras el 201 de la decisión, la asignación debe abrirse junto al
// recibo sin esperar a recargar la ficha, y el recibo debe nombrar la vía.

const EXPEDIENTE = "expediente:ct:prueba:cobertura-asignacion:001";
const HUELLA = "c".repeat(64);

function propuesta() {
  return {
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
}

function recibo() {
  return {
    esquema: "vec.contratacion-temporal.recibo-cobertura.v1",
    recibo_ref: "recibo:ct:cobertura:prueba:002",
    estado: "aplicada",
    decision_cobertura_ref: `decision-cobertura:sha256:${"d".repeat(64)}`,
    version_resultante: 3,
    confirmada_en: "2026-10-09T04:12:36.000000Z",
  };
}

function estadoFichaV2() {
  return {
    vista: "expediente",
    carga: "listo",
    expediente_ref: EXPEDIENTE,
    expediente: {
      expediente_ref: EXPEDIENTE,
      version: 2,
      demostracion: false,
      cabecera: [{ clave: "resultado_rc", valor: "Crédito comprobado", etiqueta: "Crédito" }],
    },
    cuadro: {
      demostracion: false,
      expedientes: [{
        expediente_ref: EXPEDIENTE, version: 2, fase_clave: "solicitud", estado_clave: "en_curso",
      }],
    },
  };
}

function panel() {
  const eventos = new Map();
  return {
    innerHTML: "",
    eventos,
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
    },
    contains() { return true; },
    querySelector() { return { focus() {}, scrollIntoView() {} }; },
    replaceChildren() { this.innerHTML = ""; },
  };
}

function fichaSinPanelAsignacion() {
  let asignacion = null;
  const documento = {
    createElement() {
      const nuevo = panel();
      nuevo.atributos = new Map();
      nuevo.setAttribute = (nombre, valor) => nuevo.atributos.set(nombre, valor);
      return nuevo;
    },
  };
  const cobertura = panel();
  cobertura.ownerDocument = documento;
  cobertura.after = (nodo) => {
    assert.equal(asignacion, null, "solo se abre un panel de asignación");
    asignacion = nodo;
  };
  cobertura.enviar = () => cobertura.eventos.get("submit")({
    preventDefault() {},
    target: {
      closest(selector) { return selector === "[data-ct-cobertura-form]" ? this : null; },
      querySelector(selector) {
        if (selector === "[name=via_elegida]:checked") return { value: "bolsa_vigente" };
        if (selector === "[name=motivo_clave]") return { value: "" };
        return null;
      },
    },
  });
  const raiz = {
    querySelector(selector) {
      if (selector === "[data-ct-exp-cobertura]") return cobertura;
      if (selector === "[data-ct-exp-asignacion]") return asignacion;
      return null;
    },
  };
  return { raiz, cobertura, obtenerAsignacion: () => asignacion };
}

async function estabilizar() {
  for (let i = 0; i < 6; i += 1) await Promise.resolve();
}

test("tras confirmar la cobertura en la ficha se abre la asignación con la versión del recibo", async () => {
  const ficha = fichaSinPanelAsignacion();
  const decisiones = [];
  const gestor = crearGestorTramitacion({
    raiz: ficha.raiz,
    presentador: { obtenerEstado: estadoFichaV2 },
    composicionAnalisis: {
      cliente: {
        async proponerCobertura() { return propuesta(); },
        async decidirCobertura(solicitud) { decisiones.push(solicitud); return recibo(); },
        async consultarResultadoCobertura() { throw new Error("consulta inesperada"); },
        async asignarUnidad() { throw new Error("asignación inesperada"); },
      },
    },
    coberturaDisponible: true,
    asignacionDisponible: true,
    confirmarOperacion: () => true,
  });

  assert.equal(gestor.montarCoberturaDesdeEstado(), true);
  await estabilizar();
  assert.match(ficha.cobertura.innerHTML, /data-ct-cobertura-evaluacion="bolsa_vigente"/u);

  await ficha.cobertura.enviar();
  await estabilizar();

  assert.equal(decisiones.length, 1);
  assert.equal(decisiones[0].version_esperada, 2);
  const asignacion = ficha.obtenerAsignacion();
  assert.ok(asignacion, "la asignación se abre sin recargar la ficha");
  assert.equal(asignacion.atributos.get("data-ct-exp-asignacion"), "");
  assert.notEqual(asignacion.innerHTML, "", "el formulario de asignación queda pintado");
  assert.doesNotMatch(ficha.cobertura.innerHTML, /la asignación no está disponible/u);
  assert.match(ficha.cobertura.innerHTML, /La decisión de cobertura ha quedado confirmada/u);
  assert.match(ficha.cobertura.innerHTML,
    /Vía elegida por RRHH<\/dt><dd data-ct-cobertura-via-aplicada>Bolsa vigente<\/dd>/u);

  assert.equal(gestor.montarAsignacionDesdeEstado(), false,
    "el cuadro en memoria sigue en solicitud: no se monta un segundo panel");
  gestor.retirarComponentes();
});
