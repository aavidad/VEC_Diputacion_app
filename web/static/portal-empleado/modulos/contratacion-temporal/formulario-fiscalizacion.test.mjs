import assert from "node:assert/strict";
import test from "node:test";

import { montarFormularioFiscalizacion } from "./formulario-fiscalizacion.js";
import { renderizarModuloContratacionTemporal,
  montarModuloFiscalizacionContratacionTemporal } from "./vista-expedientes.js";

const EXPEDIENTE = "expediente:ct:fiscalizacion:formulario-001";
const CLAVE = "123e4567-e89b-42d3-a456-426614174000";

function recibo(resultado = "desfavorable") {
  return {
    esquema: "vec.contratacion-temporal.recibo-fiscalizacion.v1",
    operacion: "registrar_resultado",
    expediente_ref: EXPEDIENTE,
    version_resultante: 6,
    resultado,
    fase_resultante: resultado === "desfavorable" ? "subsanacion_unidad" : "fiscalizacion",
    estado_resultante: resultado === "desfavorable" ? "incidencia" : "en_curso",
    recibo_ref: "recibo:fiscalizacion:formulario:001",
    auditoria_ref: "auditoria:fiscalizacion:formulario:001",
    evento_ref: "evento:fiscalizacion:formulario:001",
    actor_ref: "actor:intervencion:formulario:001",
    registrada_en: "2026-09-04T19:00:00Z",
    ...(resultado === "desfavorable" ? {
      unidad_retorno_ref: "unidad:retorno:formulario:001",
      responsable_retorno_ref: "responsable:retorno:formulario:001",
    } : {}),
  };
}

function raizFalsa() {
  const eventos = new Map();
  const observaciones = {
    value: "", required: false, disabled: false,
    setAttribute() {}, removeAttribute() {},
  };
  return {
    innerHTML: "",
    eventos,
    observaciones,
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
    },
    contains() { return true; },
    querySelector(selector) {
      if (selector === "[name=\"observaciones\"]") return observaciones;
      return { focus() {}, scrollIntoView() {} };
    },
    replaceChildren() { this.innerHTML = ""; },
    enviar(resultado, observaciones) {
      const controles = {
        resultado: { value: resultado },
        observaciones: { value: observaciones },
      };
      const formulario = {
        elements: { namedItem: (nombre) => controles[nombre] },
        closest(selector) {
          return selector === "[data-ct-fiscalizacion-form]" ? this : null;
        },
        checkValidity() { return true; },
        reportValidity() {},
      };
      return eventos.get("submit")({ target: formulario, preventDefault() {} });
    },
    cambiar(nombre, value) {
      if (nombre === "observaciones") observaciones.value = value;
      return eventos.get("change")({ target: { name: nombre, value } });
    },
    recuperar() {
      const control = {
        dataset: { ctFiscalizacionAccion: "recuperar" },
        closest(selector) {
          return selector === "[data-ct-fiscalizacion-accion]" ? this : null;
        },
      };
      return eventos.get("click")({ target: control, preventDefault() {} });
    },
  };
}

function montar(raiz, cliente, extras = {}) {
  return montarFormularioFiscalizacion({
    raiz,
    cliente,
    contexto: {
      expediente_ref: EXPEDIENTE,
      version_esperada: 5,
      fase_clave: "informe_juridico",
      informe_ref: "informe:juridico:formulario:001",
    },
    generarClaveIdempotencia: () => CLAVE,
    confirmarOperacion: () => true,
    ...extras,
  });
}

test("entrega el recibo validado a la continuación sin ocultarlo si falla el montaje", async () => {
  const raiz = raizFalsa();
  let siguiente;
  montar(raiz, { registrarResultadoFiscalizacion: async () => recibo("favorable") }, {
    alConfirmar: (resultado) => { siguiente = resultado; throw new Error("montaje"); },
  });
  await raiz.enviar("favorable", "");
  assert.equal(siguiente.version_resultante, 6);
  assert.equal(siguiente.resultado, "favorable");
  assert.match(raiz.innerHTML, /data-ct-fiscalizacion-recibo/u);
});

test("Intervención enlaza el llamamiento al recibo favorable dentro del módulo", async () => {
  const raiz = raizFalsa();
  const fiscalizacion = raizFalsa();
  const llamamiento = raizFalsa();
  raiz.querySelector = (selector) => selector === "[data-ct-exp-fiscalizacion]"
    ? fiscalizacion : selector === "[data-ct-exp-llamamiento]" ? llamamiento : null;
  const modulo = montarModuloFiscalizacionContratacionTemporal({
    raiz, cliente: {
      registrarResultadoFiscalizacion: async () => recibo("favorable"),
      seleccionarLlamamiento: async () => {}, registrarComunicacionLlamamiento: async () => {},
      registrarRespuestaRecibida: async () => {},
    },
    confirmarOperacion: () => true,
  });
  const formulario = {
    elements: { namedItem: (nombre) => ({ value: nombre === "expediente_ref" ? EXPEDIENTE : "5" }) },
    closest() { return this; }, checkValidity() { return true; },
  };
  raiz.eventos.get("submit")({ target: formulario, preventDefault() {} });
  await fiscalizacion.enviar("favorable", "");
  assert.match(llamamiento.innerHTML, /data-ct-llamamiento/u);
  assert.match(llamamiento.innerHTML, /Datos del expediente fiscalizado/u);
  assert.match(llamamiento.innerHTML, /value="6"/u);
  modulo.desmontar();
  assert.equal(llamamiento.eventos.size, 0);
});

test("al elegir favorable elimina y bloquea observaciones previas", () => {
  const raiz = raizFalsa();
  montar(raiz, { registrarResultadoFiscalizacion() {} });
  raiz.cambiar("observaciones", "Texto anterior.");
  raiz.cambiar("resultado", "favorable");
  assert.equal(raiz.observaciones.value, "");
  assert.equal(raiz.observaciones.required, false);
  assert.equal(raiz.observaciones.disabled, true);
});

test("muestra contexto, registra un resultado y publica el recibo auditable", async () => {
  const raiz = raizFalsa();
  const solicitudes = [];
  const confirmaciones = [];
  montar(raiz, {
    registrarResultadoFiscalizacion(solicitud) {
      solicitudes.push(solicitud);
      return Promise.resolve(recibo());
    },
  }, { confirmarOperacion(datos) { confirmaciones.push(datos); return true; } });

  assert.match(raiz.innerHTML, /data-ct-fiscalizacion-contexto/u);
  assert.match(raiz.innerHTML, /informe:juridico:formulario:001/u);
  assert.match(raiz.innerHTML, /name="resultado" type="radio"/u);
  assert.match(raiz.innerHTML, /role="status"/u);
  await raiz.enviar("desfavorable", "Reparo sintético para la unidad.");

  assert.deepEqual(solicitudes, [{
    expediente_ref: EXPEDIENTE,
    version_esperada: 5,
    clave_idempotencia: CLAVE,
    resultado: "desfavorable",
    observaciones: "Reparo sintético para la unidad.",
  }]);
  assert.match(confirmaciones[0].advertencia, /devolverá el expediente/u);
  assert.match(raiz.innerHTML, /data-ct-fiscalizacion-recibo/u);
  for (const referencia of [
    "recibo:fiscalizacion:formulario:001",
    "auditoria:fiscalizacion:formulario:001",
    "evento:fiscalizacion:formulario:001",
    "actor:intervencion:formulario:001",
    "unidad:retorno:formulario:001",
    "responsable:retorno:formulario:001",
  ]) assert.match(raiz.innerHTML, new RegExp(referencia, "u"));
});

test("recupera con el mismo cuerpo y la misma clave", async () => {
  const raiz = raizFalsa();
  const solicitudes = [];
  let intento = 0;
  montar(raiz, {
    registrarResultadoFiscalizacion(solicitud) {
      solicitudes.push(solicitud);
      intento += 1;
      if (intento === 1) {
        const error = new Error("respuesta interrumpida");
        error.resultadoIndeterminado = true;
        return Promise.reject(error);
      }
      return Promise.resolve(recibo("favorable"));
    },
  });

  await raiz.enviar("favorable", "");
  assert.match(raiz.innerHTML, /data-ct-fiscalizacion-accion="recuperar"/u);
  await raiz.recuperar();
  assert.equal(solicitudes.length, 2);
  assert.strictEqual(solicitudes[0], solicitudes[1]);
  assert.match(raiz.innerHTML, /data-ct-fiscalizacion-recibo/u);
});

test("monta la continuación al reabrir fase informe_juridico sin fijar versión", () => {
  const expediente = {
    expediente_ref: EXPEDIENTE,
    numero_visible: "2026/CT-001",
    version: 9,
    flujo_ref: "flujo:ct:sintetico",
    flujo_version: 1,
    flujo_huella: "b".repeat(64),
    cabecera: [],
    fases: [],
    tareas: [],
  };
  const html = renderizarModuloContratacionTemporal({
    vista: "expediente",
    carga: "listo",
    cuadro: { expedientes: [{
      expediente_ref: EXPEDIENTE,
      version: 9,
      fase_clave: "informe_juridico",
    }] },
    expediente,
    tarea_ref: "",
    mensaje_clave: "estado_expediente_listo",
    tipo_mensaje: "informacion",
  }, { fiscalizacionDisponible: true });

  assert.match(html, /data-ct-exp-fiscalizacion/u);
  assert.doesNotMatch(html, /data-ct-exp-informe-juridico/u);
});

test("reserva la continuación cuando el informe se prepara desde un expediente abierto", () => {
  const expediente = {
    expediente_ref: EXPEDIENTE,
    numero_visible: "2026/CT-001",
    version: 4,
    flujo_ref: "flujo:ct:sintetico",
    flujo_version: 1,
    flujo_huella: "b".repeat(64),
    cabecera: [],
    fases: [],
    tareas: [],
  };
  const html = renderizarModuloContratacionTemporal({
    vista: "expediente",
    carga: "listo",
    cuadro: { expedientes: [{
      expediente_ref: EXPEDIENTE,
      version: 4,
      fase_clave: "asignacion_unidad",
    }] },
    expediente,
    tarea_ref: "",
    mensaje_clave: "estado_expediente_listo",
    tipo_mensaje: "informacion",
  }, { informeJuridicoDisponible: true, fiscalizacionDisponible: true });

  assert.match(html, /data-ct-exp-informe-juridico/u);
  assert.match(html, /data-ct-exp-fiscalizacion/u);
});
