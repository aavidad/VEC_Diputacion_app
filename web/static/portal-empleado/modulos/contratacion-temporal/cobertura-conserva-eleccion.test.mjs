import assert from "node:assert/strict";
import test from "node:test";

import { montarFormularioCobertura } from "./formulario-cobertura.js";

const EXPEDIENTE = "expediente:ct:prueba:cobertura:eleccion";
const CLAVE = "11111111-1111-4111-8111-111111111111";
const MOTIVO = `motivo.${"a".repeat(32)}`;
const MOTIVO_OTRA_VIA = `motivo.${"b".repeat(32)}`;

function propuesta() {
  const evaluacion = {
    via_clave: "bolsa_vigente", prioridad: 1, estado: "viable",
    resultados_omitidos: [], ausencias_bloqueantes: [], ausencias_admitidas: [],
    no_habilitantes: [], conflictos: [],
  };
  const huella = "a".repeat(64);
  return {
    esquema: "vec.contratacion-temporal.propuesta-cobertura.v1",
    estado: "viable",
    via_recomendada: "bolsa_vigente",
    evaluaciones: [evaluacion,
      { ...evaluacion, via_clave: "oferta_sae", prioridad: 2 },
      { ...evaluacion, via_clave: "nueva_convocatoria_bolsa", prioridad: 3 }],
    motivos_alternativa: [
      { clave: MOTIVO, via_clave: "oferta_sae", etiqueta_i18n: "motivo_sae" },
      { clave: MOTIVO_OTRA_VIA, via_clave: "nueva_convocatoria_bolsa", etiqueta_i18n: "motivo_nueva" },
    ],
    identidad_semantica: {
      referencia: `propuesta-cobertura-semantica:sha256:${huella}`,
      huella_sha256: huella,
      canon: {
        dominio: "vec.dipgra.contratacion-temporal.propuesta-decision-cobertura-semantica",
        version_esquema: 1, algoritmo: "sha-256",
      },
    },
  };
}

function recibo() {
  return {
    esquema: "vec.contratacion-temporal.recibo-cobertura.v1",
    recibo_ref: "recibo:ct:cobertura:eleccion",
    estado: "aplicada",
    decision_cobertura_ref: `decision-cobertura:sha256:${"b".repeat(64)}`,
    version_resultante: 3,
    confirmada_en: "2026-09-04T10:49:22.962178Z",
  };
}

// El setter reconstruye controles desde el HTML como innerHTML en un navegador:
// una selección sin checked/selected desaparece en cada repintado.
function raizDOM() {
  const eventos = new Map();
  let html = "";
  let radios = [];
  let opciones = [];
  const formulario = {
    closest(selector) { return selector === "[data-ct-cobertura-form]" ? this : null; },
    querySelector(selector) {
      if (selector === "[name=via_elegida]:checked") {
        const radio = radios.find((item) => item.checked);
        return radio ? { value: radio.value } : null;
      }
      if (selector === "[name=motivo_clave]") {
        return { value: opciones.find((item) => item.selected)?.value ?? "" };
      }
      return null;
    },
  };
  return {
    get innerHTML() { return html; },
    set innerHTML(valor) {
      html = valor;
      radios = [...html.matchAll(/<input\b[^>]*name="via_elegida"[^>]*>/gu)]
        .map(([tag]) => ({
          value: /value="([^"]+)"/u.exec(tag)?.[1],
          checked: /\schecked(?:\s|>)/u.test(tag),
          disabled: /\sdisabled(?:\s|>)/u.test(tag),
          closest(selector) { return selector === "[name=via_elegida]" ? this : null; },
          focus() {}, scrollIntoView() {},
        }));
      opciones = [...html.matchAll(/<option\b[^>]*>/gu)]
        .map(([tag]) => ({
          value: /value="([^"]*)"/u.exec(tag)?.[1],
          selected: /\sselected(?:\s|>)/u.test(tag),
        }));
      if (opciones.length && !opciones.some((item) => item.selected)) opciones[0].selected = true;
    },
    addEventListener(tipo, manejar) { eventos.set(tipo, manejar); },
    removeEventListener(tipo, manejar) {
      if (eventos.get(tipo) === manejar) eventos.delete(tipo);
    },
    querySelector(selector) {
      if (selector === "[name=motivo_clave]") {
        return { value: opciones.find((item) => item.selected)?.value ?? "" };
      }
      if (selector === "[name=via_elegida]:checked") {
        return radios.find((item) => item.checked) ?? null;
      }
      return null;
    },
    contains(elemento) {
      return elemento === formulario
        || radios.includes(elemento)
        || html.includes(`data-ct-cobertura-accion="${elemento?.dataset?.ctCoberturaAccion}"`);
    },
    replaceChildren() { this.innerHTML = ""; },
    elegirVia(via) {
      assert.ok(radios.some((item) => item.value === via && !item.disabled));
      radios.forEach((item) => { item.checked = item.value === via; });
      eventos.get("change")?.({ target: radios.find((item) => item.checked) });
    },
    elegirMotivo(motivo) {
      assert.ok(opciones.some((item) => item.value === motivo));
      opciones.forEach((item) => { item.selected = item.value === motivo; });
    },
    viaMarcada() { return radios.find((item) => item.checked)?.value ?? ""; },
    motivoMarcado() { return opciones.find((item) => item.selected)?.value ?? ""; },
    enviar() {
      assert.match(html, /data-ct-cobertura-form/u);
      return eventos.get("submit")({ target: formulario, preventDefault() {} });
    },
    pulsar(accion) {
      const boton = {
        dataset: { ctCoberturaAccion: accion },
        closest(selector) { return selector === "[data-ct-cobertura-accion]" ? this : null; },
      };
      return eventos.get("click")({ target: boton, preventDefault() {} });
    },
  };
}

async function montar(raiz, decidirCobertura, confirmarOperacion = () => true) {
  const desmontar = montarFormularioCobertura({
    raiz,
    cliente: {
      async proponerCobertura() { return propuesta(); },
      decidirCobertura,
      async consultarResultadoCobertura() {
        return {
          esquema: "vec.contratacion-temporal.resultado-consulta-cobertura.v1",
          estado: "confirmado",
          recibo: recibo(),
        };
      },
    },
    contexto: { expediente_ref: EXPEDIENTE, version_esperada: 2 },
    generarClaveIdempotencia: () => CLAVE,
    confirmarOperacion,
    mensajes: { motivo_sae: "Elección de SAE", motivo_nueva: "Nueva convocatoria" },
  });
  await Promise.resolve();
  await Promise.resolve();
  return desmontar;
}

test("vía alternativa sobrevive al error y el reintento envía una decisión exacta", async () => {
  const raiz = raizDOM();
  const decisiones = [];
  let confirmaciones = 0;
  const desmontar = await montar(raiz, async (solicitud) => {
    decisiones.push(solicitud);
    return recibo();
  }, () => { confirmaciones += 1; return true; });

  raiz.elegirVia("oferta_sae");
  await raiz.enviar();
  assert.equal(raiz.viaMarcada(), "oferta_sae");
  assert.equal(raiz.motivoMarcado(), "");
  assert.equal(confirmaciones, 0);
  assert.equal(decisiones.length, 0);

  raiz.elegirMotivo(MOTIVO);
  await raiz.enviar();
  assert.equal(confirmaciones, 1);
  assert.equal(decisiones.length, 1);
  assert.deepEqual(decisiones[0], {
    expediente_ref: EXPEDIENTE,
    version_esperada: 2,
    clave_idempotencia: CLAVE,
    identidad_semantica: propuesta().identidad_semantica,
    via_elegida: "oferta_sae",
    motivo_clave: MOTIVO,
  });
  assert.match(raiz.innerHTML, /data-ct-cobertura-recibo/u);
  desmontar();
});

test("al cambiar de vía descarta el motivo cruzado y conserva el de la misma vía", async () => {
  const raiz = raizDOM();
  const decisiones = [];
  let confirmaciones = 0;
  const desmontar = await montar(raiz, async (solicitud) => {
    decisiones.push(solicitud);
    return recibo();
  }, () => { confirmaciones += 1; return true; });

  raiz.elegirVia("oferta_sae");
  raiz.elegirMotivo(MOTIVO);
  raiz.elegirVia("oferta_sae");
  assert.equal(raiz.motivoMarcado(), MOTIVO, "la misma vía conserva su motivo");

  raiz.elegirVia("nueva_convocatoria_bolsa");
  assert.equal(raiz.viaMarcada(), "nueva_convocatoria_bolsa");
  assert.equal(raiz.motivoMarcado(), "");
  assert.doesNotMatch(raiz.innerHTML, new RegExp(`value="${MOTIVO}"`, "u"));
  assert.match(raiz.innerHTML, new RegExp(`value="${MOTIVO_OTRA_VIA}"`, "u"));
  await raiz.enviar();
  assert.equal(raiz.viaMarcada(), "nueva_convocatoria_bolsa");
  assert.equal(raiz.motivoMarcado(), "");
  assert.doesNotMatch(raiz.innerHTML, new RegExp(`value="${MOTIVO}"`, "u"));
  assert.equal(confirmaciones, 0);
  assert.equal(decisiones.length, 0);

  raiz.elegirVia("oferta_sae");
  assert.equal(raiz.motivoMarcado(), "", "volver a la vía anterior exige elegir de nuevo");
  raiz.elegirMotivo(MOTIVO);
  await raiz.enviar();
  assert.deepEqual(decisiones.map(({ via_elegida, motivo_clave }) => ({
    via_elegida, motivo_clave,
  })), [{ via_elegida: "oferta_sae", motivo_clave: MOTIVO }]);
  desmontar();
});

test("cancelar una alternativa conservada no produce efecto", async () => {
  const raiz = raizDOM();
  let decisiones = 0;
  const desmontar = await montar(raiz, async () => {
    decisiones += 1;
    return recibo();
  }, () => false);
  raiz.elegirVia("oferta_sae");
  await raiz.enviar();
  raiz.elegirMotivo(MOTIVO);
  await raiz.enviar();
  assert.equal(decisiones, 0);
  assert.equal(raiz.viaMarcada(), "oferta_sae");
  assert.equal(raiz.motivoMarcado(), MOTIVO);
  desmontar();
});

test("resultado indeterminado bloquea una segunda decisión y solo consulta el recibo", async () => {
  const raiz = raizDOM();
  const decisiones = [];
  const error = new Error("indeterminado");
  error.resultadoIndeterminado = true;
  const desmontar = await montar(raiz, async (solicitud) => {
    decisiones.push(solicitud);
    throw error;
  });
  raiz.elegirVia("oferta_sae");
  raiz.elegirMotivo(MOTIVO);
  await raiz.enviar();
  assert.equal(decisiones.length, 1);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-cobertura-form/u);
  assert.match(raiz.innerHTML, /data-ct-cobertura-indeterminado/u);
  await raiz.pulsar("consultar-resultado");
  assert.equal(decisiones.length, 1);
  assert.match(raiz.innerHTML, /data-ct-cobertura-recibo/u);
  desmontar();
});
