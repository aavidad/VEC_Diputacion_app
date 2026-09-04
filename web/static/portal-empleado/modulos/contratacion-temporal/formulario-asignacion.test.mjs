import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import { montarFormularioAsignacion } from "./formulario-asignacion.js";
import { renderizarModuloContratacionTemporal } from "./vista-expedientes.js";

const EXPEDIENTE = "expediente:ct:sintetico:asignacion-001";
const CLAVE = "123e4567-e89b-42d3-a456-426614174000";

function recibo() {
  return {
    esquema: "vec.contratacion-temporal.recibo-asignacion.v1",
    operacion: "asignar",
    expediente_ref: EXPEDIENTE,
    version_resultante: 4,
    recibo_ref: "recibo:ct:asignacion:sintetico-001",
    confirmada_en: "2026-09-04T12:50:00Z",
  };
}

function respuestaJSON(datos, estado = 201) {
  const texto = JSON.stringify(datos);
  return new Response(texto, {
    status: estado,
    headers: {
      "Content-Type": "application/json; charset=utf-8",
      "Content-Length": String(new TextEncoder().encode(texto).byteLength),
    },
  });
}

function raizFalsa() {
  const eventos = new Map();
  const focos = [];
  return {
    innerHTML: "",
    eventos,
    focos,
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
    },
    contains() { return true; },
    querySelector(selector) {
      return { focus() { focos.push(selector); }, scrollIntoView() {} };
    },
    replaceChildren() { this.innerHTML = ""; },
    enviar(valido = true) {
      const formulario = {
        closest(selector) {
          return selector === "[data-ct-asignacion-form]" ? this : null;
        },
        checkValidity() { return valido; },
        reportValidity() {},
      };
      return eventos.get("submit")({ target: formulario, preventDefault() {} });
    },
    recuperar() {
      const control = {
        dataset: { ctAsignacionAccion: "recuperar" },
        closest(selector) {
          return selector === "[data-ct-asignacion-accion]" ? this : null;
        },
      };
      return eventos.get("click")({ target: control, preventDefault() {} });
    },
  };
}

function montar(raiz, cliente, extras = {}) {
  return montarFormularioAsignacion({
    raiz,
    cliente,
    contexto: { expediente_ref: EXPEDIENTE, version_esperada: 3 },
    generarClaveIdempotencia: () => CLAVE,
    confirmarOperacion: () => true,
    ...extras,
  });
}

test("envía una sola asignación cerrada y muestra el recibo mínimo", async () => {
  const raiz = raizFalsa();
  const llamadas = [];
  let confirmaciones = 0;
  const desmontar = montar(raiz, {
    asignarUnidad(solicitud, opciones) {
      llamadas.push({ solicitud, opciones });
      return Promise.resolve(recibo());
    },
  }, { confirmarOperacion() { confirmaciones += 1; return true; } });

  assert.match(raiz.innerHTML, /unidad:desarrollo:rrhh/u);
  assert.match(raiz.innerHTML, /persona:responsable-sintetica-001/u);
  assert.match(raiz.innerHTML, /name="confirmacion" type="checkbox" required/u);
  await raiz.enviar(false);
  assert.equal(llamadas.length, 0);
  await Promise.all([raiz.enviar(), raiz.enviar()]);
  assert.equal(confirmaciones, 1);
  assert.equal(llamadas.length, 1);
  assert.deepEqual(llamadas[0].solicitud, {
    expediente_ref: EXPEDIENTE,
    version_esperada: 3,
    clave_idempotencia: CLAVE,
    unidad_ref: "unidad:desarrollo:rrhh",
    responsable_ref: "persona:responsable-sintetica-001",
  });
  assert.equal(Object.isFrozen(llamadas[0].solicitud), true);
  assert.deepEqual(Object.keys(llamadas[0].opciones), ["signal"]);
  assert.match(raiz.innerHTML, /data-ct-asignacion-recibo/u);
  assert.match(raiz.innerHTML, /recibo:ct:asignacion:sintetico-001/u);
  assert.doesNotMatch(raiz.innerHTML, /name="(?:actor|perfil|organizacion|autorizacion)"/u);
  desmontar();
  assert.equal(raiz.eventos.size, 0);
});

test("recupera tras interrupción con el mismo cuerpo y la misma clave", async () => {
  const raiz = raizFalsa();
  const cuerpos = [];
  let claves = 0;
  let intento = 0;
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl(ruta, opciones) {
      assert.equal(ruta, "/api/vec/contratacion-temporal/asignaciones");
      cuerpos.push(opciones.body);
      intento += 1;
      return Promise.resolve(intento === 1
        ? respuestaJSON({ error: {
          codigo: "resultado_no_confiable",
          clave_i18n: "api.contratacion_temporal.asignacion.error.resultado_no_confiable",
          correlacion_ref: "corr_00000000000000000000000000000000",
        } }, 502)
        : respuestaJSON({ data: recibo() }));
    },
  });
  montar(raiz, cliente, { generarClaveIdempotencia() { claves += 1; return CLAVE; } });

  await raiz.enviar();
  assert.match(raiz.innerHTML, /data-ct-asignacion-accion="recuperar"/u);
  await raiz.recuperar();
  assert.equal(claves, 1);
  assert.equal(cuerpos.length, 2);
  assert.strictEqual(cuerpos[0], cuerpos[1]);
  assert.equal(JSON.parse(cuerpos[0]).clave_idempotencia, CLAVE);
  assert.match(raiz.innerHTML, /data-ct-asignacion-recibo/u);
});

test("los fallos posteriores al envío de asignación quedan indeterminados", async () => {
  const solicitud = {
    expediente_ref: EXPEDIENTE,
    version_esperada: 3,
    clave_idempotencia: CLAVE,
    unidad_ref: "unidad:desarrollo:rrhh",
    responsable_ref: "persona:responsable-sintetica-001",
  };
  for (const [estado, codigo] of [
    [502, "resultado_no_confiable"],
    [503, "servicio_no_disponible"],
    [504, "plazo_agotado"],
  ]) {
    const cliente = crearClienteHTTPContratacionTemporal({
      fetchImpl: async () => respuestaJSON({ error: {
        codigo,
        clave_i18n: `api.contratacion_temporal.asignacion.error.${codigo}`,
        correlacion_ref: "corr_00000000000000000000000000000000",
      } }, estado),
    });
    await assert.rejects(
      cliente.asignarUnidad(solicitud),
      (error) => error?.codigo === codigo && error.resultadoIndeterminado === true,
    );
  }
  const clienteSinRespuesta = crearClienteHTTPContratacionTemporal({
    fetchImpl: async () => { throw new Error("transporte interrumpido"); },
  });
  await assert.rejects(
    clienteSinRespuesta.asignarUnidad(solicitud),
    (error) => error?.codigo === "servicio_no_disponible"
      && error.resultadoIndeterminado === true,
  );
});

test("el cliente usa la ruta real sin fabricar autoridad ni almacenamiento", async () => {
  const llamadas = [];
  const cliente = crearClienteHTTPContratacionTemporal({
    fetchImpl: async (ruta, opciones) => {
      llamadas.push({ ruta, opciones });
      return respuestaJSON({ data: recibo() });
    },
  });
  const resultado = await cliente.asignarUnidad({
    expediente_ref: EXPEDIENTE,
    version_esperada: 3,
    clave_idempotencia: CLAVE,
    unidad_ref: "unidad:desarrollo:rrhh",
    responsable_ref: "persona:responsable-sintetica-001",
  });

  assert.deepEqual(resultado, recibo());
  assert.equal(llamadas[0].ruta, "/api/vec/contratacion-temporal/asignaciones");
  assert.equal(llamadas[0].opciones.method, "POST");
  assert.equal(llamadas[0].opciones.credentials, "same-origin");
  assert.equal(llamadas[0].opciones.cache, "no-store");
  assert.deepEqual([...llamadas[0].opciones.headers.keys()], ["accept", "content-type"]);
  assert.deepEqual(JSON.parse(llamadas[0].opciones.body), {
    expediente_ref: EXPEDIENTE,
    version_esperada: 3,
    clave_idempotencia: CLAVE,
    unidad_ref: "unidad:desarrollo:rrhh",
    responsable_ref: "persona:responsable-sintetica-001",
  });
});

test("montaje, textos y manifiestos publican una única continuación real", async () => {
  const html = renderizarModuloContratacionTemporal({
    vista: "alta", carga: "listo", cuadro: null, expediente: null,
    mensaje_clave: "estado_listo", tipo_mensaje: "informacion",
  }, {
    altaDisponible: true,
    analisisDisponible: true,
    coberturaDisponible: true,
    asignacionDisponible: true,
  });
  assert.match(html, /data-ct-exp-asignacion/u);

  for (const nombre of ["interno.manifest", "produccion.manifest"]) {
    const manifiesto = await readFile(new URL(`../../../../${nombre}`, import.meta.url), "utf8");
    for (const recurso of ["contrato-asignacion.js", "formulario-asignacion.js"]) {
      assert.equal(manifiesto.split(recurso).length - 1, 1, `${nombre}: ${recurso}`);
    }
  }
  const fuentes = await Promise.all([
    readFile(new URL("./formulario-asignacion.js", import.meta.url), "utf8"),
    readFile(new URL("./cliente-http.js", import.meta.url), "utf8"),
  ]);
  assert.doesNotMatch(fuentes.join("\n"), /localStorage|sessionStorage|document\.cookie/u);
});
