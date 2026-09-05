import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { createHash, webcrypto } from "node:crypto";
import { File } from "node:buffer";
import test from "node:test";
import { montarFormularioLlamamiento } from "./formulario-llamamiento.js";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import {
  montarModuloContratacionTemporal, renderizarModuloContratacionTemporal,
} from "./vista-expedientes.js";

const CLAVE = "123e4567-e89b-42d3-a456-426614174000";
const EXPEDIENTE = "expediente:ct:sintetico:001";
const seleccion = () => ({
  expediente_ref: EXPEDIENTE, version_esperada: "6", clave_idempotencia: CLAVE,
});
const recibo = {
  esquema: "vec.contratacion-temporal.recibo-seleccion-llamamiento.v1",
  estado: "confirmado", recibo_ref: "recibo:sintetico:001", confirmada_en: "2026-09-05T08:00:00Z",
  organizacion_ref: "organizacion:sintetica:001", llamamiento_ref: "llamamiento:sintetico:001",
  version_llamamiento: 1,
};
function raizPrueba() {
  const eventos = new Map();
  const borradores = {};
  const foco = [];
  const raiz = {
    innerHTML: "", eventos, borradores, foco,
    addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo),
    contains: () => true,
    replaceChildren() { this.innerHTML = ""; },
    querySelector(selector) {
      const tipo = selector.match(/^\[data-ct-llamamiento-form="(seleccion|comunicacion|respuesta|resolucion)"\]$/u)?.[1];
      if (tipo) return borradores[tipo] ?? null;
      if (selector === "[data-ct-llamamiento-comunicacion]") return { open: false };
      return { focus: () => foco.push(selector), scrollIntoView() {} };
    },
    preparar(tipo, valores) {
      const controles = Object.fromEntries(Object.entries(valores).map(
        ([nombre, value]) => [nombre, typeof value === "boolean"
          ? { value: "on", checked: value } : { value }],
      ));
      borradores[tipo] = {
        dataset: { ctLlamamientoForm: tipo },
        elements: { namedItem: (nombre) => controles[nombre] },
        closest: () => borradores[tipo],
      };
      return borradores[tipo];
    },
    enviar(tipo, valores) {
      const form = valores ? this.preparar(tipo, valores) : borradores[tipo];
      return eventos.get("submit")({ target: form, preventDefault() {} });
    },
    archivo(archivo) {
      const control = { files: archivo ? [archivo] : [], closest() { return this; } };
      return eventos.get("change")({ target: control });
    },
  };
  return raiz;
}
function montar(raiz, cliente = {}, extras = {}) {
  return montarFormularioLlamamiento({
    raiz, cliente: { seleccionarLlamamiento: async () => recibo,
      registrarComunicacionLlamamiento: async () => {},
      registrarRespuestaRecibida: async () => {}, resolverLlamamiento: async () => {}, ...cliente },
    confirmarOperacion: () => true, criptografia: webcrypto, ...extras,
  });
}

function estadoSeleccionado(expedienteRef = EXPEDIENTE) {
  return {
    vista: "expediente", carga: "listo", expediente_ref: expedienteRef,
    cuadro: { demostracion: false, expedientes: [
      { expediente_ref: expedienteRef, version: 6, fase_clave: "fiscalizacion" },
    ] },
    expediente: {
      demostracion: false, expediente_ref: expedienteRef, version: 6,
      numero_visible: "CT-SINTETICO-001", cabecera: [], fases: [], tareas: [],
    },
    tipo_mensaje: "informacion", mensaje_clave: "estado_expediente_listo",
    ocupado: false, actualizacion_pendiente: false, resultado_indeterminado: false,
  };
}

async function montarExpedienteSeleccionado(inicial, alta = null) {
  let estado = inicial, html = "", formulario;
  let peticiones = 0;
  const eventos = new Map();
  const raiz = {
    addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo),
    contains: () => true,
    get innerHTML() { return html; },
    set innerHTML(valor) { html = valor; formulario = raizPrueba(); },
    querySelector: (selector) => selector === "[data-ct-exp-llamamiento]"
      && html.includes("data-ct-exp-llamamiento") ? formulario : null,
  };
  const modulo = await montarModuloContratacionTemporal({
    raiz,
    alta,
    presentador: {
      obtenerEstado: () => estado,
      cargar: async () => estado,
      async seleccionarExpediente(referencia) { estado = estadoSeleccionado(referencia); },
    },
    llamamiento: { cliente: {
      seleccionarLlamamiento: async () => { peticiones += 1; return recibo; },
      registrarComunicacionLlamamiento: async () => { peticiones += 1; },
      registrarRespuestaRecibida: async () => { peticiones += 1; },
      resolverLlamamiento: async () => { peticiones += 1; },
    } },
  });
  return {
    formulario: () => formulario,
    peticiones: () => peticiones,
    desmontar: modulo.desmontar,
    abrir: (referencia) => eventos.get("click")({
      target: { closest: (selector) => selector === "[data-ct-exp-abrir]"
        ? { dataset: { ctExpAbrir: referencia } } : null },
      preventDefault() {},
    }),
  };
}

test("Nueva petición conserva recuperación manual tras remontar incluso si falla el cuadro", async () => {
  const alta = {
    catalogos: {},
    ejecutor: () => { throw new Error("no debe registrar otra petición"); },
  };
  for (const carga of ["listo", "error"]) {
    for (let reinicio = 0; reinicio < 2; reinicio += 1) {
      const montaje = await montarExpedienteSeleccionado({
        ...estadoSeleccionado(), vista: "alta", carga, cuadro: null,
        expediente: null, expediente_ref: "",
        mensaje_clave: carga === "error" ? "estado_error_carga" : "estado_inicial",
      }, alta);
      const html = montaje.formulario().innerHTML;
      assert.match(html, /Llamamiento y comunicación/u);
      assert.match(html, /id="ct-llamamiento-seleccion-expediente_ref"[^>]*value=""/u);
      assert.match(html, /id="ct-llamamiento-seleccion-clave_idempotencia"[^>]*value=""/u);
      assert.equal(montaje.peticiones(), 0);
      montaje.desmontar();
    }
  }
});

const CORREO = "Subject: Respuesta sintetica\r\n\r\nAceptacion declarada por RRHH.\r\n";
const HUELLA = createHash("sha256").update(CORREO).digest("hex");
const archivoCorreo = () => new File([CORREO], "respuesta-sintetica.eml");
const comunicacionRegistrada = {
  esquema: "vec.contratacion-temporal.registro-comunicacion-llamamiento.v1",
  estado_local: "registrada_localmente", comunicacion_ref: "comunicacion:sintetica:001",
  recibo_ref: "recibo:comunicacion:001", auditoria_ref: "auditoria:sintetica:001",
  version_resultante: 2, registrada_en: "2026-09-05T08:05:00Z",
  intencion_envio_ref: "intencion:sintetica:001",
};
const declaracion = () => ({
  clave_idempotencia: "123e4567-e89b-42d3-a456-426614174002",
  respuesta: "aceptacion", correo_ref: "correo:sintetico:001", recibida_en: "2026-09-05T08:30",
});
const justificante = (solicitud) => ({
  ...solicitud, esquema: "vec.contratacion-temporal.respuesta-recibida-llamamiento.v1",
  justificante_ref: "justificante:sintetico:001", recibo_ref: "recibo:respuesta:001",
  auditoria_ref: "auditoria:respuesta:001", registrada_en: "2026-09-05T09:00:00.123456Z",
  estado: "registrada_por_rrhh",
});
async function abrirRespuesta(raiz, cliente = {}, extras = {}) {
  const cerrar = montar(raiz, {
    registrarComunicacionLlamamiento: async () => comunicacionRegistrada,
    registrarRespuestaRecibida: async (s) => justificante(s), ...cliente,
  }, extras);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="respuesta"/u);
  await raiz.enviar("seleccion", seleccion());
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="respuesta"/u);
  await raiz.enviar("comunicacion", { clave_idempotencia: CLAVE });
  raiz.preparar("respuesta", declaracion());
  return cerrar;
}
const CLAVE_RESOLUCION = "123e4567-e89b-42d3-a456-426614174003";
const revisionManual = { revision_respuesta_rrhh: true, revision_plazo_rrhh: true };
const resolucionConfirmada = {
  esquema: "vec.contratacion-temporal.resolucion-comunicacion-llamamiento.v1",
  respuesta: "aceptacion", estado_plazo: "vigente", estado_local: "confirmado",
  resolucion_ref: "resolucion:sintetica:001", recibo_local_ref: "recibo:resolucion:001",
  auditoria_ref: "auditoria:resolucion:001", version_resultante: 3,
  resuelta_en: "2026-09-05T09:05:00.123450Z",
};
async function abrirResolucion(raiz, cliente = {}, extras = {}) {
  const cerrar = await abrirRespuesta(raiz, cliente, extras);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="resolucion"/u);
  await raiz.archivo(archivoCorreo());
  await raiz.enviar("respuesta", declaracion());
  raiz.preparar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION,
    revision_respuesta_rrhh: false, revision_plazo_rrhh: false });
  return cerrar;
}

test("cuarta operación exige justificante confirmado de aceptación y no se solicita automáticamente", async () => {
  for (const opcion of ["aceptacion", "renuncia", "recibo_invalido"]) {
    const raiz = raizPrueba(); let llamadas = 0;
    const cerrar = await abrirRespuesta(raiz, {
      registrarRespuestaRecibida: async (s) => opcion === "recibo_invalido"
        ? { ...justificante(s), estado: "aceptada" } : justificante(s),
      resolverLlamamiento: async () => { llamadas += 1; },
    });
    await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION });
    assert.equal(llamadas, 0);
    delete raiz.borradores.resolucion; // El envío simulado no crea un formulario real.
    await raiz.archivo(archivoCorreo());
    await raiz.enviar("respuesta", { ...declaracion(), respuesta: opcion === "renuncia" ? opcion : "aceptacion" });
    assert.equal(llamadas, 0);
    if (opcion === "aceptacion") {
      const formulario = raiz.innerHTML.match(/<form data-ct-llamamiento-form="resolucion"[\s\S]*?<\/form>/u)[0];
      assert.match(formulario, /name="clave_idempotencia" value=""/u);
      assert.doesNotMatch(formulario, /type="file"|name="(?:actor_ref|estado_plazo|evaluacion_plazo_ref|politica_ref)"/u);
      assert.match(formulario, /name="respuesta" required disabled/u);
      for (const nombre of Object.keys(revisionManual)) {
        const control = formulario.match(new RegExp(`<input[^>]*name="${nombre}"[^>]*>`, "u"))[0];
        assert.match(control, /type="checkbox"/u);
        assert.doesNotMatch(control, /\schecked|\sdisabled/u);
        assert.match(formulario, new RegExp(`label for="ct-llamamiento-resolucion-${nombre}"`, "u"));
      }
      assert.match(formulario, /name="criterio_validacion_ref" value="politica:ct:revision-manual-sintetica:20260906"[^>]*readonly/u);
      assert.match(formulario, /Validación manual de desarrollo: no acredita entrega de correo ni plazo legal real/u);
      assert.match(formulario, /He comprobado la respuesta y su justificante/u);
      assert.match(formulario, /Para este ejercicio sintético, he comprobado que la respuesta llegó dentro del plazo del ejercicio/u);
      for (const campo of ["organizacion_ref", "expediente_ref", "llamamiento_ref",
        "comunicacion_ref", "version_esperada", "prueba_respuesta_ref"]) {
        assert.match(formulario, new RegExp(`name="${campo}"[^>]*readonly`, "u"));
      }
    } else {
      assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="resolucion"/u);
      await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION });
      assert.equal(llamadas, 0);
    }
    cerrar();
  }
});

test("resolución exige clave propia y confirmación; el DOM no concede antecedentes, identidad ni plazo", async () => {
  const raiz = raizPrueba(), solicitudes = [], confirmaciones = []; let confirmar = false;
  const cerrar = await abrirResolucion(raiz, { resolverLlamamiento: async (s) => {
    solicitudes.push(s); return resolucionConfirmada;
  } }, { confirmarOperacion: (datos) => {
    if (!datos.datos.prueba_respuesta_ref) return true;
    confirmaciones.push(datos); return confirmar;
  } });
  for (const [revision_respuesta_rrhh, revision_plazo_rrhh] of [[false, false], [true, false], [false, true], ["true", "true"]]) {
    await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION, revision_respuesta_rrhh, revision_plazo_rrhh });
    assert.equal(solicitudes.length, 0);
    assert.equal(confirmaciones.length, 0);
    assert.match(raiz.innerHTML, /Marque ambas comprobaciones expresas del ejercicio sintético/u);
  }
  for (const clave_idempotencia of [CLAVE, declaracion().clave_idempotencia]) {
    await raiz.enviar("resolucion", { clave_idempotencia, ...revisionManual });
    assert.match(raiz.innerHTML, /utilice una clave propia de resolución/u);
  }
  assert.equal(confirmaciones.length, 0);
  await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION, ...revisionManual });
  assert.equal(solicitudes.length, 0);
  confirmar = true;
  await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION, ...revisionManual,
    organizacion_ref: "org:inventada", expediente_ref: "exp:inventado", llamamiento_ref: "llam:inventado",
    comunicacion_ref: "com:inventada", version_esperada: "99", respuesta: "renuncia",
    prueba_respuesta_ref: "prueba:inventada", estado_plazo: "vigente", actor_ref: "actor:inventado",
    criterio_validacion_ref: "politica:inventada" });
  assert.deepEqual(solicitudes, [{ clave_idempotencia: CLAVE_RESOLUCION,
    organizacion_ref: recibo.organizacion_ref, expediente_ref: EXPEDIENTE,
    llamamiento_ref: recibo.llamamiento_ref, comunicacion_ref: comunicacionRegistrada.comunicacion_ref,
    version_esperada: 2, respuesta: "aceptacion", prueba_respuesta_ref: justificante({}).justificante_ref,
    ...revisionManual, criterio_validacion_ref: "politica:ct:revision-manual-sintetica:20260906" }]);
  assert.ok(Object.isFrozen(solicitudes[0]));
  assert.match(confirmaciones.at(-1).advertencia, /no acredita entrega de correo ni plazo legal real/u);
  assert.match(confirmaciones.at(-1).advertencia, /politica:ct:revision-manual-sintetica:20260906/u);
  assert.match(confirmaciones.at(-1).advertencia, /justificante:sintetico:001/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="resolucion"/u);
  assert.match(raiz.innerHTML, /Aceptación registrada · ejercicio sintético/u);
  assert.match(raiz.innerHTML, /El servidor confirma el registro en Contratación temporal y Bolsa/u);
  assert.match(raiz.innerHTML, /2026-09-05T09:05:00.123450Z/u);
  assert.equal(raiz.foco.at(-1), '[data-ct-llamamiento-recibo="resolucion"]');
  await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION });
  assert.equal(solicitudes.length, 1);
  cerrar();
});

test("HTTP 409 pendiente sin efectos libera las casillas pero conserva la clave y exige otra confirmación", async () => {
  const raiz = raizPrueba(), solicitudes = []; let claves = 0, confirmaciones = 0;
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta, opciones) => {
    assert.ok(ruta.endsWith("/resoluciones"));
    solicitudes.push(opciones.body);
    return new Response(JSON.stringify({ error: {
      codigo: "validacion_respuesta_pendiente",
      clave_i18n: "api.contratacion_temporal.comunicacion_llamamiento.error.validacion_respuesta_pendiente",
      correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
    } }), { status: 409, headers: { "content-type": "application/json; charset=utf-8" } });
  } });
  const cerrar = await abrirResolucion(raiz, { resolverLlamamiento: cliente.resolverLlamamiento }, {
    confirmarOperacion: () => { confirmaciones += 1; return true; },
    generarClaveIdempotencia: () => { claves += 1; return CLAVE; },
  });
  assert.equal(solicitudes.length, 0);
  for (let intento = 0; intento < 2; intento += 1) {
    await raiz.enviar("resolucion", { clave_idempotencia: intento === 0 ? CLAVE_RESOLUCION : CLAVE, ...revisionManual });
    assert.equal(solicitudes.length, intento + 1);
    assert.match(raiz.innerHTML, /Pendiente de validar respuesta y plazo por RRHH\. No se ha confirmado la aceptación\./u);
    assert.match(raiz.innerHTML, /Revisar y solicitar resolución de aceptación/u);
    assert.match(raiz.innerHTML, /name="clave_idempotencia" value="123e4567-e89b-42d3-a456-426614174003"[^>]*readonly/u);
    const control = raiz.innerHTML.match(/<input[^>]*name="revision_plazo_rrhh"[^>]*>/u)[0];
    assert.match(control, /\schecked/u); assert.doesNotMatch(control, /\sdisabled/u);
    assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-recibo="resolucion"|No se ha podido confirmar el resultado/u);
    assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-clave="resolucion"/u); // gitleaks:allow — selector HTML.
    raiz.eventos.get("click")({ target: { closest: () => ({ dataset: { ctLlamamientoClave: "resolucion" } }) },
      preventDefault() {} });
    assert.equal(claves, 0);
    await raiz.enviar("resolucion", { clave_idempotencia: CLAVE,
      ...revisionManual, revision_plazo_rrhh: false });
    assert.equal(solicitudes.length, intento + 1); // Corregir no autoriza a enviar sin ambas casillas.
    assert.doesNotMatch(raiz.innerHTML.match(/<input[^>]*name="revision_plazo_rrhh"[^>]*>/u)[0], /\schecked/u);
  }
  assert.equal(solicitudes[0], solicitudes[1]);
  assert.equal(JSON.parse(solicitudes[1]).clave_idempotencia, CLAVE_RESOLUCION);
  assert.equal(confirmaciones, 5); // Tres antecedentes más dos solicitudes expresas.
  cerrar();
});

test("resolución ambigua o recibo inválido conserva el intento incluso si el replay pierde permiso", async () => {
  for (const invalido of [false, true, "replay_pendiente"]) {
    const raiz = raizPrueba(), solicitudes = [];
    const cerrar = await abrirResolucion(raiz, { resolverLlamamiento: async (s) => {
      solicitudes.push(s);
      if (solicitudes.length === 1) {
        if (invalido === true) return { ...resolucionConfirmada, estado_plazo: "vencido" };
        throw new Error("transporte interrumpido");
      }
      if (solicitudes.length === 2) throw Object.assign(new Error(), {
        codigo: invalido === "replay_pendiente" ? "validacion_respuesta_pendiente" : "acceso_denegado",
        estado: invalido === "replay_pendiente" ? 409 : 403, envelopeValido: true, resultadoIndeterminado: false,
      });
      return { ...resolucionConfirmada, estado_local: "replay_confirmado" };
    } });
    for (let intento = 0; intento < 3; intento += 1) {
      await raiz.enviar("resolucion", { clave_idempotencia: intento === 0 ? CLAVE_RESOLUCION : CLAVE,
        revision_respuesta_rrhh: intento === 0, revision_plazo_rrhh: intento === 0 });
      assert.deepEqual(solicitudes[intento], solicitudes[0]);
      if (intento < 2) {
        assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-recibo="resolucion"/u);
        assert.match(raiz.innerHTML.match(/<input[^>]*name="revision_plazo_rrhh"[^>]*>/u)[0], /checked disabled/u);
        assert.match(raiz.innerHTML, /No se ha podido confirmar el resultado/u);
      }
      assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-clave="resolucion"/u); // gitleaks:allow — selector HTML.
    }
    assert.match(raiz.innerHTML, /Recuperado sin repetir el efecto/u);
    cerrar();
  }
});

test("resolución no duplica al pulsar dos veces; desmontar aborta y descarta el recibo tardío", async () => {
  const raiz = raizPrueba(); let resolver, signal, llamadas = 0;
  const cerrar = await abrirResolucion(raiz, { resolverLlamamiento: (_, opciones) => {
    llamadas += 1; signal = opciones.signal;
    return new Promise((resolve) => { resolver = resolve; });
  } });
  raiz.preparar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION, ...revisionManual });
  const pendiente = raiz.enviar("resolucion");
  await raiz.enviar("resolucion");
  assert.equal(llamadas, 1);
  cerrar();
  assert.equal(signal.aborted, true);
  resolver(resolucionConfirmada);
  await pendiente;
  assert.equal(raiz.innerHTML, "");
  assert.equal(raiz.eventos.size, 0);
});

test("respuesta RRHH se deriva del recibo v2; confirma datos y envía solo declaración y huella", async () => {
  const raiz = raizPrueba(), confirmaciones = [], solicitudes = [];
  const cerrar = await abrirRespuesta(raiz, { registrarRespuestaRecibida: async (s) => {
    solicitudes.push(s); return justificante(s);
  } }, { confirmarOperacion: (datos) => { confirmaciones.push(datos); return true; } });
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="respuesta"/u);
  for (const campo of ["organizacion_ref", "expediente_ref", "llamamiento_ref",
    "comunicacion_ref", "version_comunicacion_esperada", "correo_sha256"]) {
    assert.match(raiz.innerHTML, new RegExp(`name="${campo}"[^>]*readonly`, "u"));
  }
  await raiz.archivo(archivoCorreo());
  assert.equal(solicitudes.length, 0, "calcular la huella no registra nada");
  assert.match(raiz.innerHTML, /Huella calculada: se conserva la huella mostrada/u);
  assert.match(raiz.innerHTML, /Fecha de recepción declarada \(UTC\)/u);
  await raiz.enviar("respuesta", { ...declaracion(), organizacion_ref: "org:inventada",
    expediente_ref: "exp:inventado", llamamiento_ref: "llam:inventado",
    comunicacion_ref: "com:inventada", version_comunicacion_esperada: "99",
    correo_sha256: "f".repeat(64), contenido: CORREO, actor_ref: "actor:inventado" });
  assert.deepEqual(solicitudes, [{
    clave_idempotencia: declaracion().clave_idempotencia,
    organizacion_ref: recibo.organizacion_ref, expediente_ref: EXPEDIENTE,
    llamamiento_ref: recibo.llamamiento_ref, comunicacion_ref: comunicacionRegistrada.comunicacion_ref,
    version_comunicacion_esperada: 2, respuesta: "aceptacion", correo_ref: declaracion().correo_ref,
    correo_sha256: HUELLA, recibida_en: "2026-09-05T08:30:00Z",
  }]);
  const confirmacion = confirmaciones.at(-1);
  assert.equal(confirmacion.referencia, EXPEDIENTE);
  assert.match(confirmacion.advertencia, new RegExp(HUELLA, "u"));
  assert.match(confirmacion.advertencia, /no cambia la candidatura/iu);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="respuesta"/u);
  assert.match(raiz.innerHTML, /no resuelve aceptación o renuncia/u);
  assert.match(raiz.innerHTML, /2026-09-05T09:00:00.123456Z/u);
  assert.doesNotMatch(raiz.innerHTML, /Subject:|respuesta-sintetica.eml|name="actor_ref"/u);
  assert.equal(raiz.foco.at(-1), '[data-ct-llamamiento-recibo="respuesta"]');
  await raiz.enviar("respuesta", declaracion());
  assert.equal(solicitudes.length, 1);
  cerrar();
});

test("respuesta exige comunicación confirmada, archivo, datos y confirmación explícita", async () => {
  const raiz = raizPrueba(); let llamadas = 0;
  const cerrar = montar(raiz, { registrarRespuestaRecibida: async () => { llamadas += 1; } });
  await raiz.enviar("respuesta", declaracion());
  assert.equal(llamadas, 0);
  cerrar();
  const otra = raizPrueba();
  await abrirRespuesta(otra, { registrarRespuestaRecibida: async () => { llamadas += 1; } }, {
    confirmarOperacion: (datos) => !datos.datos.respuesta,
  });
  await otra.enviar("respuesta", declaracion());
  assert.match(otra.innerHTML, /Calcule la huella desde un .eml/u);
  await otra.archivo(archivoCorreo());
  await otra.enviar("respuesta", { ...declaracion(), respuesta: "expiracion_gobernada" });
  await otra.enviar("respuesta", declaracion());
  assert.equal(llamadas, 0);
  assert.doesNotMatch(otra.innerHTML, /data-ct-llamamiento-recibo="respuesta"/u);
});

test("correo limita tamaño antes de leer, vacía huella anterior y falla cerrado sin WebCrypto", async () => {
  for (const archivo of [null, { name: "otro.pdf", size: 10 },
    { name: "vacio.eml", size: 0 }, { name: "grande.eml", size: 2 * 1024 * 1024 + 1 }]) {
    const raiz = raizPrueba(); let llamadas = 0;
    await abrirRespuesta(raiz, { registrarRespuestaRecibida: async () => { llamadas += 1; } });
    await raiz.archivo(archivoCorreo());
    if (archivo) archivo.arrayBuffer = () => assert.fail("no debe leer");
    await raiz.archivo(archivo);
    await raiz.enviar("respuesta", declaracion());
    assert.equal(llamadas, 0);
    assert.match(raiz.innerHTML, /name="correo_sha256" value=""/u);
  }
  const raiz = raizPrueba();
  await abrirRespuesta(raiz, {}, { criptografia: null });
  await raiz.archivo(archivoCorreo());
  assert.match(raiz.innerHTML, /No se pudo calcular la huella/u);
  assert.match(raiz.innerHTML, /name="correo_sha256" value=""/u);
});

test("huella admite exactamente 2 MiB y descarta bytes locales al terminar", async () => {
  const raiz = raizPrueba();
  await abrirRespuesta(raiz);
  const bytes = new Uint8Array(2 * 1024 * 1024).fill(65);
  const esperada = createHash("sha256").update(bytes).digest("hex");
  await raiz.archivo({ name: "limite.eml", size: bytes.length, arrayBuffer: async () => bytes.buffer });
  assert.match(raiz.innerHTML, new RegExp(`name="correo_sha256" value="${esperada}"`, "u"));
  assert.ok(bytes.every((b) => b === 0));
});

test("huella en curso bloquea envío y desmontar descarta su resolución tardía", async () => {
  const raiz = raizPrueba(); let resolver, llamadas = 0;
  const cerrar = await abrirRespuesta(raiz, { registrarRespuestaRecibida: () => { llamadas += 1; } });
  const bytes = new Uint8Array([65]);
  const pendiente = raiz.archivo({ name: "pendiente.eml", size: 1,
    arrayBuffer: () => new Promise((resolve) => { resolver = resolve; }) });
  assert.match(raiz.innerHTML, /Calculando la huella local/u);
  await raiz.enviar("respuesta", declaracion());
  assert.equal(llamadas, 0);
  cerrar();
  resolver(bytes.buffer);
  await pendiente;
  assert.equal(raiz.innerHTML, "");
  assert.equal(raiz.eventos.size, 0);
  assert.equal(bytes[0], 0);
});

test("respuesta perdida congela clave, fecha, declaración y huella; replay usa el mismo intento", async () => {
  const raiz = raizPrueba(), solicitudes = [];
  await abrirRespuesta(raiz, { registrarRespuestaRecibida: async (s) => {
    solicitudes.push(s);
    if (solicitudes.length === 1) throw new Error("transporte interrumpido");
    if (solicitudes.length === 2) throw Object.assign(new Error("permiso de replay denegado"), {
      codigo: "acceso_denegado", envelopeValido: true, resultadoIndeterminado: false,
    });
    return { ...justificante(s), estado: "replay_registrada_por_rrhh" };
  } });
  await raiz.archivo(archivoCorreo());
  await raiz.enviar("respuesta", declaracion());
  assert.match(raiz.innerHTML, /Recuperar con los mismos datos/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-clave="respuesta"/u); // gitleaks:allow — selector HTML, no credencial.
  await raiz.archivo(new File(["otro"], "otro.eml"));
  await raiz.enviar("respuesta", { ...declaracion(), respuesta: "renuncia",
    clave_idempotencia: CLAVE, correo_ref: "correo:otro", recibida_en: "2026-09-06T10:00" });
  assert.deepEqual(solicitudes[0], solicitudes[1]);
  assert.equal(solicitudes[0].correo_sha256, HUELLA);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-clave="respuesta"/u); // gitleaks:allow — selector HTML, no credencial.
  await raiz.enviar("respuesta", { ...declaracion(), respuesta: "renuncia" });
  assert.deepEqual(solicitudes[0], solicitudes[2]);
  assert.match(raiz.innerHTML, /Misma declaración recuperada, sin nuevo registro/u);
});

test("doble envío de respuesta no duplica y conflicto impide reintentar", async () => {
  const raiz = raizPrueba(); let resolver; const solicitudes = [];
  await abrirRespuesta(raiz, { registrarRespuestaRecibida: (s) => {
    solicitudes.push(s); return new Promise((resolve) => { resolver = resolve; });
  } });
  await raiz.archivo(archivoCorreo());
  const pendiente = raiz.enviar("respuesta", declaracion());
  await raiz.enviar("respuesta", declaracion());
  assert.equal(solicitudes.length, 1);
  resolver(justificante(solicitudes[0]));
  await pendiente;
  for (const codigo of ["version_en_conflicto", "clave_idempotencia_reutilizada"]) {
    const otra = raizPrueba(); let llamadas = 0;
    await abrirRespuesta(otra, { registrarRespuestaRecibida: async () => {
      llamadas += 1; throw Object.assign(new Error(), { codigo, envelopeValido: true });
    } });
    await otra.archivo(archivoCorreo());
    await otra.enviar("respuesta", declaracion());
    await otra.enviar("respuesta", declaracion());
    assert.equal(llamadas, 1);
    assert.match(otra.innerHTML, /requiere revisión del servidor/u);
    assert.doesNotMatch(otra.innerHTML, /data-ct-llamamiento-recibo="respuesta"/u);
  }
});

test("el expediente fiscalizado seleccionado rellena el formulario existente sin POST ni clave automática", async () => {
  const montaje = await montarExpedienteSeleccionado(estadoSeleccionado());
  const html = montaje.formulario().innerHTML;
  assert.match(html, /Datos del expediente fiscalizado/u);
  assert.match(html, /id="ct-llamamiento-seleccion-expediente_ref"[^>]*value="expediente:ct:sintetico:001"/u);
  assert.match(html, /id="ct-llamamiento-seleccion-version_esperada"[^>]*value="6"/u);
  assert.match(html, /id="ct-llamamiento-seleccion-clave_idempotencia"[^>]*value=""/u);
  assert.equal(montaje.peticiones(), 0);
  montaje.desmontar();
});

test("no monta llamamiento para un detalle sin cargar, desfasado, ajeno o no fiscalizado", async () => {
  const casos = [
    (e) => { e.carga = "cargando"; },
    (e) => { e.carga = "error"; },
    (e) => { e.expediente = null; },
    (e) => { e.expediente_ref = "expediente:otro"; },
    (e) => { e.cuadro.expedientes[0].expediente_ref = "expediente:otro"; },
    (e) => { e.cuadro.expedientes[0].version = 5; },
    (e) => { e.cuadro.expedientes[0].fase_clave = "subsanacion_unidad"; },
    (e) => { e.actualizacion_pendiente = true; },
    (e) => { e.expediente.demostracion = true; },
  ];
  for (const modificar of casos) {
    const estado = estadoSeleccionado();
    modificar(estado);
    const montaje = await montarExpedienteSeleccionado(estado);
    assert.equal(montaje.formulario().innerHTML, "");
    assert.equal(montaje.formulario().eventos.size, 0);
    assert.equal(montaje.peticiones(), 0);
    montaje.desmontar();
  }
});

test("abrir otro expediente no reutiliza referencias ni clave del formulario anterior", async () => {
  const montaje = await montarExpedienteSeleccionado(estadoSeleccionado());
  const anterior = montaje.formulario();
  anterior.preparar("seleccion", seleccion());
  await montaje.abrir("expediente:ct:sintetico:002");
  const actual = montaje.formulario();
  assert.notEqual(actual, anterior);
  assert.equal(anterior.eventos.size, 0);
  assert.match(actual.innerHTML, /id="ct-llamamiento-seleccion-expediente_ref"[^>]*value="expediente:ct:sintetico:002"/u);
  assert.match(actual.innerHTML, /id="ct-llamamiento-seleccion-clave_idempotencia"[^>]*value=""/u);
  assert.doesNotMatch(actual.innerHTML, /expediente:ct:sintetico:001/u);
  assert.equal(montaje.peticiones(), 0);
  montaje.desmontar();
});

test("manifiestos publican todos los recursos del llamamiento sin duplicados", async () => {
  const recursos = [
    "cliente-http-llamamiento.js", "contrato-llamamiento.js",
    "formulario-llamamiento.js", "i18n-llamamiento.js", "renderizado-llamamiento.js",
  ];
  for (const nombre of ["interno.manifest", "produccion.manifest"]) {
    const contenido = await readFile(new URL(`../../../../${nombre}`, import.meta.url), "utf8");
    const rutas = contenido.trim().split(/\r?\n/u);
    for (const recurso of recursos) {
      const ruta = `static/portal-empleado/modulos/contratacion-temporal/${recurso}`;
      assert.doesNotMatch(ruta, /presentacion|demo/iu, "No debe activar la exclusión de material de presentación");
      assert.equal(rutas.filter((entrada) => entrada === ruta).length, 1, `${nombre}: ${recurso}`);
      assert.ok((await readFile(new URL(`./${recurso}`, import.meta.url), "utf8")).length > 0);
    }
  }
});

test("enlaza fiscalización y exige confirmación sin inventar candidato ni autoridad", async () => {
  const raiz = raizPrueba();
  let llamadas = 0;
  const desmontar = montar(raiz, { seleccionarLlamamiento: () => { llamadas += 1; } },
    { confirmarOperacion: () => false });
  assert.equal(desmontar.actualizarContexto({ expediente_ref: EXPEDIENTE, version_esperada: 6 }), true);
  assert.match(raiz.innerHTML, /Datos del expediente fiscalizado/u);
  assert.match(raiz.innerHTML, /value="expediente:ct:sintetico:001"/u);
  await raiz.enviar("seleccion", seleccion());
  assert.equal(llamadas, 0);
  assert.doesNotMatch(raiz.innerHTML, /candidatura_ref|name="actor_ref"/u);
  desmontar();
  assert.equal(raiz.eventos.size, 0);
});

test("doble envío no duplica operación y el recibo minimizado abre comunicación", async () => {
  const raiz = raizPrueba();
  let resolver;
  const solicitudes = [];
  montar(raiz, {
    seleccionarLlamamiento: (solicitud) => {
      solicitudes.push(solicitud);
      return new Promise((resolve) => { resolver = resolve; });
    },
  });
  const primera = raiz.enviar("seleccion", seleccion());
  await raiz.enviar("seleccion", { ...seleccion(), clave_idempotencia: "otra" });
  assert.equal(solicitudes.length, 1);
  resolver(recibo);
  await primera;
  assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="seleccion"/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-comunicacion open/u);
  assert.match(raiz.innerHTML, /No expone identidad/u);
  assert.equal(solicitudes[0].version_esperada, 6);
});

test("respuesta perdida: recuperación usa petición congelada aunque cambien controles", async () => {
  const raiz = raizPrueba();
  const solicitudes = [];
  montar(raiz, { seleccionarLlamamiento: async (solicitud) => {
    solicitudes.push(solicitud);
    if (solicitudes.length === 1) throw new Error("red");
    return recibo;
  } });
  await raiz.enviar("seleccion", seleccion());
  assert.match(raiz.innerHTML, /Recuperar con los mismos datos/u);
  await raiz.enviar("seleccion", { ...seleccion(), expediente_ref: "expediente:distinto" });
  assert.deepEqual(solicitudes[0], solicitudes[1]);
});

test("tras desmontar permite recuperar mediante los inputs visibles, sin memoria web", async () => {
  const solicitudes = [];
  const cliente = { seleccionarLlamamiento: async (solicitud) => {
    solicitudes.push(solicitud);
    return recibo;
  } };
  const primera = raizPrueba();
  const cerrar = montar(primera, cliente);
  await primera.enviar("seleccion", seleccion());
  cerrar();
  const segunda = raizPrueba();
  montar(segunda, cliente);
  await segunda.enviar("seleccion", seleccion());
  assert.deepEqual(solicitudes[0], solicitudes[1]);
  assert.match(segunda.innerHTML, /Recibo verificado/u);
});

test("conflicto no reintentable mantiene datos y no permite otra escritura", async () => {
  const raiz = raizPrueba();
  let llamadas = 0;
  montar(raiz, { seleccionarLlamamiento: async () => {
    llamadas += 1;
    throw Object.assign(new Error("conflicto"), { codigo: "conflicto_no_reintentable" });
  } });
  await raiz.enviar("seleccion", seleccion());
  await raiz.enviar("seleccion", seleccion());
  assert.equal(llamadas, 1);
  assert.match(raiz.innerHTML, /requiere revisión del servidor/u);
});

test("comunicación exige sus referencias y muestra registro, no envío de correo", async () => {
  const raiz = raizPrueba();
  let solicitud;
  montar(raiz, { registrarComunicacionLlamamiento: async (entrada) => {
    solicitud = entrada;
    return {
      esquema: "vec.contratacion-temporal.registro-comunicacion-llamamiento.v1",
      estado_local: "confirmado", comunicacion_ref: "comunicacion:sintetica:001",
      recibo_ref: "recibo:comunicacion:001", auditoria_ref: "auditoria:sintetica:001",
      version_resultante: 2, respuesta_hasta: "2026-09-06T08:00:00Z",
    };
  } });
  await raiz.enviar("seleccion", seleccion());
  await raiz.enviar("comunicacion", { clave_idempotencia: CLAVE });
  assert.equal(solicitud.version_esperada, 1);
  assert.match(raiz.innerHTML, /Comunicación registrada/u);
  assert.match(raiz.innerHTML, /no acredita envío de correo real/u);
});

test("desmontar cancela la espera y una respuesta tardía no repinta", async () => {
  const raiz = raizPrueba();
  let resolver;
  let signal;
  const cerrar = montar(raiz, { seleccionarLlamamiento: (_, opciones) => {
    signal = opciones.signal;
    return new Promise((resolve) => { resolver = resolve; });
  } });
  const vuelo = raiz.enviar("seleccion", seleccion());
  cerrar();
  assert.equal(signal.aborted, true);
  resolver(recibo);
  await vuelo;
  assert.equal(raiz.innerHTML, "");
});

test("la bandeja con error conserva navegación y reintento sin formulario de llamamiento", () => {
  const html = renderizarModuloContratacionTemporal({
    vista: "cuadro", carga: "error", cuadro: null, expediente: null,
    tipo_mensaje: "error", mensaje_clave: "estado_error_carga",
  }, { llamamientoDisponible: true });
  assert.match(html, /ct-exp-navegacion/u);
  assert.match(html, /role="alert"/u);
  assert.match(html, /data-ct-exp-accion="reintentar"/u);
  assert.doesNotMatch(html, /data-ct-exp-llamamiento/u);
});

test("encadenado selección y replay autorrellenan comunicación local sin transcribir referencias", async () => {
  const claveComunicacion = "123e4567-e89b-42d3-a456-426614174001";
  const llamadas = [];
  for (const estadoHTTP of [201, 200]) {
    const raiz = raizPrueba();
    const cliente = crearClienteHTTPContratacionTemporal({
      fetchImpl: async (ruta, opciones) => {
        const entrada = JSON.parse(opciones.body);
        llamadas.push({ ruta, entrada });
        const data = ruta.endsWith("/seleccion") ? recibo : {
          esquema: "vec.contratacion-temporal.registro-comunicacion-llamamiento.v1",
          estado_local: estadoHTTP === 201 ? "registrada_localmente" : "replay_registrada_localmente",
          comunicacion_ref: "comunicacion:sintetica:001", recibo_ref: "recibo:comunicacion:001",
          auditoria_ref: "auditoria:sintetica:001", version_resultante: 2,
          registrada_en: "2026-09-05T08:05:00Z", intencion_envio_ref: "intencion:sintetica:001",
        };
        return new Response(JSON.stringify({ data }), {
          status: estadoHTTP, headers: { "Content-Type": "application/json; charset=utf-8" },
        });
      },
    });
    const cerrar = montar(raiz, cliente);
    assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="comunicacion"/u);
    await raiz.enviar("seleccion", seleccion());
    for (const [campo, valor] of Object.entries({
      organizacion_ref: recibo.organizacion_ref, expediente_ref: EXPEDIENTE,
      llamamiento_ref: recibo.llamamiento_ref, version_esperada: "1",
      prueba_entrega_ref: recibo.recibo_ref,
    })) {
      assert.match(raiz.innerHTML, new RegExp(
        'id="ct-llamamiento-comunicacion-' + campo + '"[^>]*value="' + valor + '"[^>]*readonly', "u",
      ));
    }
    // Ni siquiera al manipular controles se sustituyen los antecedentes del recibo.
    await raiz.enviar("comunicacion", {
      clave_idempotencia: claveComunicacion, organizacion_ref: "organizacion:inventada",
      expediente_ref: "expediente:inventado", llamamiento_ref: "llamamiento:inventado",
      version_esperada: "99", prueba_entrega_ref: "prueba:inventada",
    });
    assert.deepEqual(llamadas.at(-1).entrada, {
      clave_idempotencia: claveComunicacion,
      organizacion_ref: recibo.organizacion_ref, expediente_ref: EXPEDIENTE,
      llamamiento_ref: recibo.llamamiento_ref, version_esperada: 1,
      prueba_entrega_ref: recibo.recibo_ref,
    });
    assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="comunicacion"/u);
    assert.match(raiz.innerHTML, /Sin entrega acreditada/u);
    assert.doesNotMatch(raiz.innerHTML, /Plazo de respuesta registrado/u);
    cerrar();
  }
  assert.deepEqual(llamadas.slice(0, 2), llamadas.slice(2));
});
