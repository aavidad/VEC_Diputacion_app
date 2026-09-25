import test from "node:test";
import assert from "node:assert/strict";
import {
  crearControladorOperacionesSituacion,
  consultarOperacionesSituacion,
  registrarOperacionSituacion,
  referenciaContieneDocumentoIdentidad,
  renderizarOperacionesSituacion,
  rutaOperacionesSituacion,
  operacionesDisponibles,
} from "./portal-bolsas-operaciones.js";

test("P-WEB-14 rechaza DNI, NIE y etiquetas de identidad antes del POST B8", async () => {
  for (const referencia of ["12345678Z", "REG/X1234567L", "exp:12.34.56.78-Z", "dni:123", "nie-ref"] ) {
    assert.equal(referenciaContieneDocumentoIdentidad(referencia), true, referencia);
    const resultado = await registrarOperacionSituacion("bolsa:uno", "participacion:dos",
      { ...cuerpo, justificante: { ...cuerpo.justificante, referencia } }, "clave", {
        fetchImpl: () => { throw new Error("No debe enviarse"); },
      });
    assert.equal(resultado.status, 400);
    assert.equal(resultado.mensaje, "La referencia no puede contener un DNI o NIE; use el número de registro o de expediente");
  }
  assert.equal(referenciaContieneDocumentoIdentidad("EXP-2026-123"), false);
});

test("P-WEB-14 muestra el motivo en el paso de justificante y conserva el formulario", () => {
  const anterior = globalThis.FormData;
  globalThis.FormData = class { constructor(formulario) { this.valores = formulario.valores; } get(campo) { return this.valores[campo]; } };
  try {
    const flujo = { carga: "listo", items: [], paso: 2, operacion: "pausar", formulario: { motivo: "Solicitud" } };
    const estado = { modalFicha: { candidato: { estado_clave: "disponible" }, operacionesB8: flujo } };
    const controlador = crearControladorOperacionesSituacion({ estado, renderizar() {}, recargar() {} });
    controlador.manejarSubmit({ target: { closest: () => ({ valores: { tipo: "correo", referencia: "X1234567L", sha256: "a".repeat(64) } }) }, preventDefault() {} });
    assert.equal(flujo.paso, 2);
    assert.match(renderizarOperacionesSituacion({ candidato: estado.modalFicha.candidato, estado: flujo }), /La referencia no puede contener un DNI o NIE/);
  } finally { globalThis.FormData = anterior; }
});

const cuerpo = {
  operacion: "excluir",
  motivo: "Solicitud recibida",
  validador: "Persona validadora",
  justificante: { tipo: "solicitud_candidato", referencia: "EXP-1", sha256: "a".repeat(64) },
};
function response(status, body) {
  return { ok: status >= 200 && status < 300, status, json: async () => body };
}

test("P-WEB-13 compone la ruta canónica y registra 201/200 con JSON e Idempotency-Key", async () => {
  assert.equal(rutaOperacionesSituacion("bolsa:uno", "participacion:dos"), "/api/vec/bolsa/bolsas/bolsa:uno/candidatos/participacion:dos/operaciones");
  for (const status of [201, 200]) {
    let llamada;
    const fetchImpl = async (...args) => { llamada = args; return response(status, { data: { recibo_ref: "recibo:1", situacion: "excluido", desde: "2026-09-23T08:00:00Z", reutilizada: status === 200 } }); };
    const resultado = await registrarOperacionSituacion("bolsa:uno", "participacion:dos", cuerpo, "clave-estable", { fetchImpl });
    assert.equal(resultado.ok, true);
    assert.equal(resultado.datos.reutilizada, status === 200);
    assert.equal(llamada[0], rutaOperacionesSituacion("bolsa:uno", "participacion:dos"));
    assert.deepEqual([llamada[1].credentials, llamada[1].mode, llamada[1].cache, llamada[1].redirect], ["same-origin", "same-origin", "no-store", "error"]);
    assert.deepEqual(llamada[1].headers, { Accept: "application/json", "Content-Type": "application/json", "Idempotency-Key": "clave-estable" });
    assert.deepEqual(JSON.parse(llamada[1].body), cuerpo);
  }
});

test("P-WEB-13 traduce 400, 403, 409 y 503 sin exponer datos de error", async () => {
  const esperados = new Map([
    [400, "solicitud_invalida"], [403, "acceso_denegado"], [409, "transicion_no_valida"], [503, "servicio_no_disponible"],
  ]);
  for (const [status, codigo] of esperados) {
    const resultado = await registrarOperacionSituacion("bolsa:uno", "participacion:dos", cuerpo, "clave", {
      fetchImpl: async () => response(status, { error: { codigo, datos_personales: "no debe mostrarse" } }),
    });
    assert.equal(resultado.ok, false);
    assert.equal(resultado.codigo, codigo);
    assert.equal(resultado.mensaje.includes("no debe mostrarse"), false);
  }
  const reutilizada = await registrarOperacionSituacion("bolsa:uno", "participacion:dos", cuerpo, "clave", {
    fetchImpl: async () => response(409, { error: { codigo: "clave_reutilizada" } }),
  });
  assert.equal(reutilizada.codigo, "clave_reutilizada");
  assert.match(reutilizada.mensaje, /clave de idempotencia ya se usó/i);
});

test("P-WEB-13 señala 404 como operación aún no desplegada y valida el GET", async () => {
  const get404 = await consultarOperacionesSituacion("bolsa:uno", "participacion:dos", { fetchImpl: async (_url, opciones) => {
    assert.deepEqual(opciones.headers, { Accept: "application/json" });
    assert.deepEqual([opciones.credentials, opciones.mode, opciones.cache, opciones.redirect], ["same-origin", "same-origin", "no-store", "error"]);
    return response(404, {});
  } });
  assert.match(get404.mensaje, /operación no disponible todavía/i);
  const post404 = await registrarOperacionSituacion("bolsa:uno", "participacion:dos", cuerpo, "clave", { fetchImpl: async () => response(404, {}) });
  assert.match(post404.mensaje, /operación no disponible todavía/i);
  const sinRuta = renderizarOperacionesSituacion({ candidato: { estado_clave: "disponible" }, estado: { carga: "error", noDisponible: true, error: get404.mensaje } });
  assert.match(sinRuta, /operación no disponible todavía/i);
  assert.doesNotMatch(sinRuta, /data-b8-accion="seleccionar"/);
  const historial = await consultarOperacionesSituacion("bolsa:uno", "participacion:dos", { fetchImpl: async () => response(200, {
    data: { esquema: "vec.bolsa.rrhh.operaciones_situacion.v1", items: [{ desde: "hoy", operacion: "pausar", situacion: "no_disponible", motivo: "motivo", justificante: cuerpo.justificante, actor: "actor", validador: "validador", validada_en: "hoy" }] },
  }) });
  assert.equal(historial.ok, true);
  assert.equal(historial.datos.length, 1);
});

test("P-WEB-13 pinta solo operaciones admitidas y documenta custodia y validación de exclusión", () => {
  const disponible = renderizarOperacionesSituacion({ candidato: { estado_clave: "disponible" }, estado: { carga: "listo", items: [] } });
  assert.match(disponible, /data-operacion="pausar"/);
  assert.match(disponible, /data-operacion="excluir"/);
  assert.doesNotMatch(disponible, /data-operacion="reactivar"/);
  const trabajando = renderizarOperacionesSituacion({ candidato: { estado_clave: "trabajando" }, estado: { carga: "listo", items: [] } });
  assert.match(trabajando, /data-operacion="reactivar"/);
  assert.match(trabajando, /data-operacion="excluir"/);
  const excluir = renderizarOperacionesSituacion({ candidato: { estado_clave: "disponible" }, estado: { carga: "listo", items: [], operacion: "excluir", paso: 2, formulario: { motivo: "Solicitud" } } });
  assert.match(excluir, /El documento permanece en su custodia y no se sube a VEC/);
  const validarExclusion = renderizarOperacionesSituacion({ candidato: { estado_clave: "disponible" }, estado: { carga: "listo", items: [], operacion: "excluir", paso: 3, formulario: { validador: "RRHH" } } });
  assert.match(validarExclusion, /persona validadora distinta/i);
  assert.match(validarExclusion, /name="confirma_validador_distinto" required/);
  assert.match(validarExclusion, /Regla provisional, duda 6/);
  const enviando = renderizarOperacionesSituacion({ candidato: { estado_clave: "disponible" }, estado: { carga: "listo", items: [], operacion: "excluir", paso: 3, formulario: {}, enviando: true } });
  assert.match(enviando, /Registrando…/);
  assert.match(enviando, /data-b8-accion="cancelar" disabled/);
});

test("P-WEB-13 conserva la clave en un reintento 503 y refresca después del recibo", async () => {
  const fetchAnterior = globalThis.fetch;
  const formDataAnterior = globalThis.FormData;
  const claves = [];
  let recargas = 0;
  const estado = { bolsaSeleccionada: "bolsa:uno", modalFicha: { candidato: { estado_clave: "disponible", participacion_ref: "participacion:dos" } } };
  globalThis.FormData = class { constructor(formulario) { this.valores = formulario.valores; } get(clave) { return this.valores[clave]; } has(clave) { return Boolean(this.valores[clave]); } };
  globalThis.fetch = async (_ruta, opciones) => {
    if (opciones.method === "GET") return response(200, { data: { esquema: "vec.bolsa.rrhh.operaciones_situacion.v1", items: [] } });
    claves.push(opciones.headers["Idempotency-Key"]);
    return claves.length === 1 ? response(503, { error: { codigo: "servicio_no_disponible" } })
      : response(201, { data: { recibo_ref: "recibo:1", situacion: "no_disponible", desde: "2026-09-23T08:00:00Z", reutilizada: false } });
  };
  try {
    const controlador = crearControladorOperacionesSituacion({ estado, renderizar() {}, recargar: async () => { recargas += 1; } });
    const click = (accion, operacion) => controlador.manejarClick({ target: { closest: () => ({ dataset: { b8Accion: accion, operacion } }) }, preventDefault() {} });
    const submit = (valores) => controlador.manejarSubmit({ target: { closest: () => ({ valores }) }, preventDefault() {} });
    click("seleccionar", "pausar");
    submit({ motivo: "Solicitud" });
    submit({ tipo: "solicitud_candidato", referencia: "EXP-1", sha256: "A".repeat(64) });
    submit({ validador: "per_validadora" });
    await new Promise((resolver) => setTimeout(resolver, 0));
    assert.match(estado.modalFicha.operacionesB8.errorOperacion, /no está disponible/);
    submit({ validador: "per_validadora" });
    await new Promise((resolver) => setTimeout(resolver, 0));
    assert.deepEqual(claves, [claves[0], claves[0]]);
    assert.equal(estado.modalFicha.operacionesB8.recibo, "recibo:1");
    assert.equal(estado.modalFicha.candidato.estado_clave, "no_disponible");
    assert.equal(recargas, 1);
  } finally {
    globalThis.fetch = fetchAnterior;
    globalThis.FormData = formDataAnterior;
  }
});

test("B8 ofrece pausar una renuncia solo si el servidor admite pasar a no disponible", () => {
  const botones = (html) => [...html.matchAll(/data-operacion="([a-z]+)"/g)].map((m) => m[1]);
  const renuncia = { estado_clave: "renuncia" };
  // Sin reglas del servidor: lo de siempre.
  assert.deepEqual(operacionesDisponibles("renuncia", null), ["excluir"]);
  assert.deepEqual(operacionesDisponibles("disponible", undefined), ["pausar", "excluir"]);
  // Política de 000012: la renuncia vuelve a disponible, pero B8 no reactiva renuncias.
  assert.deepEqual(operacionesDisponibles("renuncia", { renuncia: ["disponible", "excluido"] }), ["excluir"]);
  // Política del Reglamento publicada: renuncia justificada a no disponible.
  const publicada = { renuncia: ["no_disponible", "excluido"], disponible: ["no_disponible", "pendiente_incorporacion", "renuncia", "excluido"], excluido: [] };
  assert.deepEqual(botones(renderizarOperacionesSituacion({ candidato: renuncia, estado: { carga: "listo", items: [], transiciones: publicada } })), ["pausar", "excluir"]);
  assert.deepEqual(operacionesDisponibles("disponible", publicada), ["pausar", "excluir"]);
  assert.deepEqual(operacionesDisponibles("excluido", publicada), []);
  // El servidor cierra una transición: el botón desaparece.
  assert.deepEqual(operacionesDisponibles("trabajando", { trabajando: ["excluido"] }), ["excluir"]);
});
