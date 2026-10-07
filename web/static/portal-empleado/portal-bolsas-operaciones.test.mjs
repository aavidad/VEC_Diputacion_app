import test from "node:test";
import { prepararTextosPortal } from "./portal-i18n.js?v=20261001-ct-a-i18n-v1";
import { prepararMensajesContratos } from "./portal-bolsas-contratos.js?v=20261002-a-recuperar-379-v1";
await prepararTextosPortal("bolsa");
await prepararMensajesContratos();
import assert from "node:assert/strict";
import {
  crearControladorOperacionesSituacion,
  consultarOperacionesSituacion,
  consultarSolicitudesDocumentalesRRHH,
  registrarOperacionSituacion,
  referenciaContieneDocumentoIdentidad,
  renderizarOperacionesSituacion,
  rutaOperacionesSituacion,
  operacionesDisponibles,
} from "./portal-bolsas-operaciones.js?v=20261001-f-reconciliacion-323-v1";

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
    const flujo = { carga: "listo", items: [], paso: 2, operacion: "excluir", formulario: { motivo: "Solicitud" } };
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
const transicionesRRHH18 = { renuncia: ["en_revision", "disponible", "excluido"], en_revision: ["disponible", "excluido"], excluido: ["disponible"] };
const desde = "2026-09-23T08:00:00Z";

test("RRHH17 enlaza una solicitud documental pendiente sin cambiar estado al consultarla", async () => {
  const solicitud = { solicitud_ref: `solicitud-documental:${"b".repeat(64)}`, version: 1,
    contenido_sha256: "c".repeat(64), documento_ref: "documento:fin-causa", documento_sha256: "a".repeat(64),
    fecha_fin_causa: "2026-09-22", estado: "pendiente_rrhh", recibo_ref: "recibo:solicitud", registrada_en: desde };
  const respuesta = (items) => async (_ruta, opciones) => {
    assert.equal(opciones.method, "GET");
    return response(200, { data: { esquema: "vec.bolsa.rrhh.solicitudes_documentales.v1", items } });
  };
  const bien = await consultarSolicitudesDocumentalesRRHH("bolsa:uno", "participacion:dos", { fetchImpl: respuesta([solicitud]) });
  assert.equal(bien.ok, true);
  const vista = renderizarOperacionesSituacion({ candidato: { estado_clave: "en_revision", estado_desde: desde },
    estado: { carga: "listo", items: [], transiciones: transicionesRRHH18, solicitudesDocumentales: bien.datos } });
  assert.match(vista, /data-b8-accion="seleccionar-solicitud"/);
  assert.doesNotMatch(vista, /data-b8-accion="seleccionar" data-operacion="regularizar"/);
	const sinFecha = await consultarSolicitudesDocumentalesRRHH("bolsa:uno", "participacion:dos", { fetchImpl: respuesta([{ ...solicitud, fecha_fin_causa: null }]) });
	assert.equal(sinFecha.ok, true);
	const vistaSinFecha = renderizarOperacionesSituacion({ candidato: { estado_clave: "en_revision", estado_desde: desde },
		estado: { carga: "listo", items: [], transiciones: transicionesRRHH18, solicitudesDocumentales: sinFecha.datos } });
	assert.match(vistaSinFecha, /data-b8-accion="seleccionar-solicitud"/);
	assert.doesNotMatch(vistaSinFecha, /data-b8-accion="seleccionar-solicitud"[^>]*disabled/);
  const mal = await consultarSolicitudesDocumentalesRRHH("bolsa:uno", "participacion:dos", { fetchImpl: respuesta([{ ...solicitud, documento_sha256: "mal" }]) });
  assert.equal(mal.codigo, "respuesta_invalida");
  const comando = { operacion: "regularizar", motivo: "Fin de causa validado", validador: "persona:rrhh",
    justificante: { tipo: "solicitud_candidato", referencia: solicitud.documento_ref, sha256: solicitud.documento_sha256 },
    situacion_esperada_desde: desde, causa_finalizada_en: solicitud.fecha_fin_causa,
    solicitud_ref: solicitud.solicitud_ref, solicitud_version_esperada: solicitud.version,
    solicitud_contenido_sha256: solicitud.contenido_sha256 };
  const post = await registrarOperacionSituacion("bolsa:uno", "participacion:dos", comando, "clave", { fetchImpl: async (_ruta, opciones) => {
    assert.equal(JSON.parse(opciones.body).solicitud_ref, solicitud.solicitud_ref);
    return response(201, { data: { recibo_ref: "recibo:regularizacion", recibo_resolucion_ref: "recibo:solicitud-documental-resolucion:uno", resuelta_en: desde,
      situacion: "disponible", desde, reutilizada: false } });
  } });
  assert.equal(post.ok, true);
  const incompleto = await registrarOperacionSituacion("bolsa:uno", "participacion:dos", comando, "clave", { fetchImpl: async () =>
    response(201, { data: { recibo_ref: "recibo:regularizacion", situacion: "disponible", desde, reutilizada: false } }) });
  assert.equal(incompleto.codigo, "respuesta_invalida");
});

test("403 documental queda aislado, sin reintento ni acción de regularizar", async () => {
  const resultado = await consultarSolicitudesDocumentalesRRHH("bolsa:uno", "participacion:dos", {
    fetchImpl: async (_ruta, opciones) => {
      assert.equal(opciones.method, "GET");
      return response(403, { error: { detalle: "no exponer" } });
    },
  });
  assert.equal(resultado.status, 403);
  assert.equal(resultado.mensaje, "Acceso denegado");
  const vista = renderizarOperacionesSituacion({ candidato: { estado_clave: "en_revision", estado_desde: desde },
    estado: { carga: "listo", items: [], transiciones: transicionesRRHH18,
      solicitudesDocumentales: [], solicitudesError: resultado.mensaje, solicitudesStatus: 403 } });
  assert.match(vista, /Acceso denegado/u);
  assert.doesNotMatch(vista, /data-b8-accion="reintentar-solicitudes"|data-operacion="regularizar"/u);
});

test("403 de operaciones no borra identidad o situación principal ni ofrece escritura", async () => {
  const anterior = globalThis.fetch;
  globalThis.fetch = async () => response(403, { error: { codigo: "acceso_denegado", detalle: "interno" } });
  try {
    const modal = { candidato: { participacion_ref: "participacion:dos", nombre_visible: "Persona autorizada",
      estado_clave: "en_revision", estado_desde: desde } };
    const estado = { bolsaSeleccionada: "bolsa:uno", modalFicha: modal };
    const controlador = crearControladorOperacionesSituacion({ estado, renderizar() {}, recargar() {},
      consultarReglas: async () => ({ ok: true, datos: { transiciones: transicionesRRHH18 } }),
      consultarDocumentales: async () => ({ ok: false, status: 403, mensaje: "Acceso denegado" }),
    });
    await controlador.cargar(modal);
    assert.equal(modal.candidato.nombre_visible, "Persona autorizada");
    assert.equal(modal.candidato.estado_clave, "en_revision");
    assert.equal(modal.candidato.estado_desde, desde);
    assert.equal(modal.operacionesB8.carga, "error");
    assert.equal(modal.operacionesB8.noDisponible, true);
    const vista = renderizarOperacionesSituacion({ candidato: modal.candidato, estado: modal.operacionesB8 });
    assert.match(vista, /Acceso denegado/u);
    assert.doesNotMatch(vista, /data-b8-accion="reintentar"|data-b8-accion="reintentar-solicitudes"|data-operacion=/u);
  } finally { globalThis.fetch = anterior; }
});

test("reintentar cada sección consulta solo su fuente y conserva al candidato autorizado", async () => {
  const anterior = globalThis.fetch;
  const rutas = [];
  let operaciones = 0, documentales = 0, reglas = 0;
  globalThis.fetch = async (ruta, opciones) => {
    assert.equal(opciones.method, "GET");
    rutas.push(String(ruta));
    if (String(ruta).endsWith("/operaciones")) {
      operaciones++;
      return response(200, { data: { esquema: "vec.bolsa.rrhh.operaciones_situacion.v1", items: [],
        situacion_vigente: { situacion: "en_revision", desde } } });
    }
    if (String(ruta).endsWith("/contratos")) return response(404, {});
    if (String(ruta).endsWith("/reincorporaciones-titular")) return response(401, {});
    assert.fail(`ruta no esperada: ${ruta}`);
  };
  try {
    const modal = { candidato: { participacion_ref: "participacion:dos", nombre_visible: "Persona autorizada",
      estado_clave: "en_revision", estado_desde: desde } };
    const estado = { bolsaSeleccionada: "bolsa:uno", modalFicha: modal };
    const controlador = crearControladorOperacionesSituacion({ estado, renderizar() {}, recargar() {},
      consultarReglas: async () => { reglas++; return { ok: true, datos: { transiciones: transicionesRRHH18 } }; },
      consultarDocumentales: async () => {
        documentales++;
        return documentales === 1
          ? { ok: false, status: 503, mensaje: "No se pudieron consultar las solicitudes." }
          : { ok: true, datos: [] };
      },
    });
    await controlador.cargar(modal);
    await new Promise((resolver) => setImmediate(resolver));
    assert.equal(operaciones, 1);
    assert.equal(reglas, 1);
    assert.equal(documentales, 1);
    assert.equal(rutas.filter((ruta) => ruta.endsWith("/contratos")).length, 1);
    assert.equal(rutas.filter((ruta) => ruta.endsWith("/reincorporaciones-titular")).length, 1);
    const pulsar = (accion) => controlador.manejarClick({ target: { closest: () => ({ dataset: { b8Accion: accion } }) }, preventDefault() {} });
    pulsar("reintentar-solicitudes");
    await new Promise((resolver) => setImmediate(resolver));
    assert.equal(documentales, 2);
    assert.equal(operaciones, 1);
    assert.equal(reglas, 1);
    pulsar("reintentar");
    await new Promise((resolver) => setImmediate(resolver));
    assert.equal(operaciones, 2);
    assert.equal(reglas, 2);
    assert.equal(documentales, 3);
    assert.equal(rutas.filter((ruta) => ruta.endsWith("/contratos")).length, 1);
    assert.equal(rutas.filter((ruta) => ruta.endsWith("/reincorporaciones-titular")).length, 1);
    assert.equal(modal.candidato.nombre_visible, "Persona autorizada");
  } finally { globalThis.fetch = anterior; }
});

test("reintentar el historial no repite una lectura documental ya denegada", async () => {
  const anterior = globalThis.fetch;
  let operaciones = 0, documentales = 0;
  globalThis.fetch = async (ruta, opciones) => {
    assert.equal(opciones.method, "GET");
    if (String(ruta).endsWith("/operaciones")) {
      operaciones++;
      return operaciones === 1 ? response(503, {})
        : response(200, { data: { esquema: "vec.bolsa.rrhh.operaciones_situacion.v1", items: [],
          situacion_vigente: { situacion: "en_revision", desde } } });
    }
    return response(404, {});
  };
  try {
    const modal = { candidato: { participacion_ref: "participacion:dos", nombre_visible: "Persona autorizada",
      estado_clave: "en_revision", estado_desde: desde } };
    const estado = { bolsaSeleccionada: "bolsa:uno", modalFicha: modal };
    const controlador = crearControladorOperacionesSituacion({ estado, renderizar() {}, recargar() {},
      consultarReglas: async () => ({ ok: true, datos: { transiciones: transicionesRRHH18 } }),
      consultarDocumentales: async () => { documentales++; return { ok: false, status: 403, mensaje: "Acceso denegado" }; },
    });
    await controlador.cargar(modal);
    const vista = renderizarOperacionesSituacion({ candidato: modal.candidato, estado: modal.operacionesB8 });
    assert.match(vista, /data-b8-accion="reintentar"/u);
    controlador.manejarClick({ target: { closest: () => ({ dataset: { b8Accion: "reintentar" } }) }, preventDefault() {} });
    await new Promise((resolver) => setImmediate(resolver));
    assert.equal(operaciones, 2);
    assert.equal(documentales, 1);
    assert.equal(modal.candidato.nombre_visible, "Persona autorizada");
    assert.equal(modal.operacionesB8.solicitudesStatus, 403);
  } finally { globalThis.fetch = anterior; }
});

test("RRHH18 solo ofrece revisión y regularización con catálogo nuevo y situación vigente", () => {
  assert.deepEqual(operacionesDisponibles("renuncia", transicionesRRHH18), ["revisar", "regularizar", "excluir"]);
  assert.deepEqual(operacionesDisponibles("en_revision", transicionesRRHH18), ["regularizar", "excluir"]);
  assert.deepEqual(operacionesDisponibles("excluido", transicionesRRHH18), ["regularizar"]);
  assert.deepEqual(operacionesDisponibles("excluido", { ...transicionesRRHH18, excluido: [] }), []);
  for (const clave of ["disponible", "trabajando", "no_disponible", "suspendido", "desconocido"]) {
    const vista = renderizarOperacionesSituacion({ candidato: { estado_clave: clave, estado_desde: desde }, estado: { carga: "listo", transiciones: { ...transicionesRRHH18, [clave]: ["disponible", "en_revision", "no_disponible", "excluido"] } } });
    assert.doesNotMatch(vista, /data-operacion="(?:pausar|reactivar|revisar|regularizar)"/);
  }
  for (const candidato of [{ estado_clave: "renuncia" }, { estado_clave: "renuncia", estado_desde: "ayer" }]) {
    const vista = renderizarOperacionesSituacion({ candidato, estado: { carga: "listo", transiciones: transicionesRRHH18 } });
    assert.doesNotMatch(vista, /data-operacion="(?:revisar|regularizar)"/);
    assert.match(vista, /Recargue la ficha o consulte con RRHH/);
  }
  assert.deepEqual(operacionesDisponibles("renuncia", { renuncia: ["disponible", "excluido"] }), ["excluir"]);
  const vista = renderizarOperacionesSituacion({ candidato: { estado_clave: "renuncia", estado_desde: desde }, estado: { carga: "listo", transiciones: transicionesRRHH18 } });
  assert.match(vista, /data-operacion="revisar"/);
  assert.match(vista, /data-operacion="regularizar"/);
	const revisionTrasRenuncia = renderizarOperacionesSituacion({
		candidato: { estado_clave: "no_disponible", estado_desde: desde },
		estado: { carga: "listo", transiciones: { ...transicionesRRHH18, no_disponible: ["en_revision", "excluido"] },
			items: [{ desde, recibo_ref: "recibo:renuncia", operacion: "pausar", situacion: "no_disponible", motivo: "Renuncia justificada", justificante: { tipo: "otro", referencia: "documento:1", sha256: "a".repeat(64) }, actor: "persona:rrhh", validador: "persona:rrhh", validada_en: desde }],
			cambios: [{ campo: "situacion", recibo_ref: "recibo:renuncia", valor_anterior: "renuncia", valor_nuevo: "no_disponible", instante: desde, actor: "persona:rrhh" }] },
	});
	assert.match(revisionTrasRenuncia, /data-operacion="revisar"/);
});

test("RRHH18 rechaza códigos históricos y exige CAS y fin de causa antes del POST", async () => {
  for (const comando of [
    { ...cuerpo, operacion: "pausar" }, { ...cuerpo, operacion: "reactivar" },
    { ...cuerpo, operacion: "revisar" },
    { ...cuerpo, operacion: "regularizar", situacion_esperada_desde: desde },
    ...["2026-02-30", "2099-01-01", "2026-09-23T00:00:00Z"].map((fecha) => ({ ...cuerpo, operacion: "regularizar", situacion_esperada_desde: desde, causa_finalizada_en: fecha })),
  ]) {
    const res = await registrarOperacionSituacion("bolsa:uno", "participacion:dos", comando, "clave", { fetchImpl: () => assert.fail("no debe enviar un comando inválido") });
    assert.equal(res.ok, false);
    assert.equal(res.status, 400);
  }
  for (const operacion of ["revisar", "regularizar"]) {
    const comando = { ...cuerpo, operacion, situacion_esperada_desde: desde, ...(operacion === "regularizar" ? { causa_finalizada_en: "2026-09-22" } : {}) };
    const situacion = operacion === "revisar" ? "en_revision" : "disponible";
    const res = await registrarOperacionSituacion("bolsa:uno", "participacion:dos", comando, "clave", { fetchImpl: async (_url, opciones) => {
      assert.deepEqual(JSON.parse(opciones.body), comando);
      return response(201, { data: { recibo_ref: "recibo:rrhh18", situacion, desde, reutilizada: false } });
    } });
    assert.equal(res.ok, true);
    const inconsistente = await registrarOperacionSituacion("bolsa:uno", "participacion:dos", comando, "clave", { fetchImpl: async () => response(201, { data: { recibo_ref: "recibo:rrhh18", situacion: "no_disponible", desde, reutilizada: false } }) });
    assert.equal(inconsistente.codigo, "respuesta_invalida");
  }
});

test("RRHH18 carga el CAS del estado vigente autorizado, nunca del historial", async () => {
  const fetchAnterior = globalThis.fetch;
  try {
    for (const vigente of [{ situacion: "en_revision", desde }, undefined]) {
      globalThis.fetch = async () => response(200, { data: { esquema: "vec.bolsa.rrhh.operaciones_situacion.v1", items: [], situacion_vigente: vigente } });
      const modal = { candidato: { estado_clave: "renuncia", estado_desde: "2026-01-01T00:00:00Z", participacion_ref: "participacion:dos" } };
      const estado = { bolsaSeleccionada: "bolsa:uno", modalFicha: modal };
      const controlador = crearControladorOperacionesSituacion({ estado, renderizar() {}, recargar() {}, consultarReglas: async () => ({ ok: true, datos: { transiciones: transicionesRRHH18 } }) });
      await controlador.cargar(modal);
      assert.equal(modal.candidato.estado_desde, vigente?.desde);
      if (vigente) assert.equal(modal.candidato.estado_clave, vigente.situacion);
    }
  } finally { globalThis.fetch = fetchAnterior; }
});

test("RRHH18 el formulario exige fecha, documento de fin de causa y confirmación de RRHH", () => {
  const candidato = { estado_clave: "en_revision", estado_desde: desde };
  const flujo = { carga: "listo", items: [], transiciones: transicionesRRHH18, operacion: "regularizar", paso: 2, formulario: {} };
  assert.match(renderizarOperacionesSituacion({ candidato, estado: flujo }), /name="causa_finalizada_en" required/);
  assert.match(renderizarOperacionesSituacion({ candidato, estado: flujo }), /El documento debe acreditar que la causa ha terminado/);
  flujo.paso = 3;
  assert.match(renderizarOperacionesSituacion({ candidato, estado: flujo }), /name="confirma_fin_causa" required/);
  const formAnterior = globalThis.FormData;
  globalThis.FormData = class { constructor(f) { this.valores = f.valores; } get(k) { return this.valores[k] ?? null; } has(k) { return Boolean(this.valores[k]); } };
  try {
    const estado = { modalFicha: { candidato, operacionesB8: flujo } };
    const ctl = crearControladorOperacionesSituacion({ estado, renderizar() {}, recargar() {} });
    const submit = (valores) => ctl.manejarSubmit({ target: { closest: () => ({ valores }) }, preventDefault() {} });
    submit({ validador: "rrhh:sintetico" });
    assert.match(flujo.errorFormulario, /Valide el documento de fin de causa/);
    assert.equal(flujo.enviando, undefined);
    flujo.paso = 2;
    submit({ tipo: "resolucion", referencia: "EXP-1", sha256: "a".repeat(64), causa_finalizada_en: "2026-02-30" });
    assert.equal(flujo.paso, 2);
    assert.match(flujo.errorFormulario, /Indique una fecha/);
    submit({ tipo: "resolucion", referencia: "EXP-1", sha256: "a".repeat(64), causa_finalizada_en: "2026-09-22" });
    assert.equal(flujo.paso, 3);
    ctl.manejarClick({ target: { closest: () => ({ dataset: { b8Accion: "seleccionar", operacion: "reactivar" } }) }, preventDefault() {} });
    assert.equal(flujo.operacion, "regularizar");
    assert.match(flujo.errorOperacion, /Recargue la ficha/);
  } finally { globalThis.FormData = formAnterior; }
});
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
  assert.doesNotMatch(disponible, /data-operacion="pausar"/);
  assert.match(disponible, /data-operacion="excluir"/);
  assert.doesNotMatch(disponible, /data-operacion="reactivar"/);
  const trabajando = renderizarOperacionesSituacion({ candidato: { estado_clave: "trabajando" }, estado: { carga: "listo", items: [] } });
  assert.doesNotMatch(trabajando, /data-operacion="reactivar"/);
  assert.match(trabajando, /data-operacion="excluir"/);
  const excluir = renderizarOperacionesSituacion({ candidato: { estado_clave: "disponible" }, estado: { carga: "listo", items: [], operacion: "excluir", paso: 2, formulario: { motivo: "Solicitud" } } });
  // La huella se calcula del archivo en el equipo: no se teclea ni se muestra.
  assert.match(excluir, /<input id="b8-justificante-archivo" type="file" data-huella-archivo aria-describedby="b8-justificante-archivo-estado" aria-required="true"/);
  assert.match(excluir, /<input type="hidden" name="sha256" value="" data-huella-valor>/);
  assert.doesNotMatch(excluir, /pattern="\[a-fA-F0-9\]\{64\}"/);
  assert.match(excluir, /<summary aria-label="Ayuda sobre el justificante">\?<\/summary><p>[^<]*no se envía ni se guarda en VEC y sigue en su custodia/);
  const validarExclusion = renderizarOperacionesSituacion({ candidato: { estado_clave: "disponible" }, estado: { carga: "listo", items: [], operacion: "excluir", paso: 3, formulario: { validador: "RRHH" } } });
  assert.match(validarExclusion, /Confirmo que el validador es otra persona/);
  assert.match(validarExclusion, /name="confirma_validador_distinto" required/);
  assert.doesNotMatch(validarExclusion, /Regla provisional|duda 6/);
  const enviando = renderizarOperacionesSituacion({ candidato: { estado_clave: "disponible" }, estado: { carga: "listo", items: [], operacion: "excluir", paso: 3, formulario: {}, enviando: true } });
  assert.match(enviando, /Registrando…/);
  assert.match(enviando, /data-b8-accion="cancelar" disabled/);
});

test("P-WEB-13 conserva la clave en un reintento 503 y refresca después del recibo", async () => {
  const fetchAnterior = globalThis.fetch;
  const formDataAnterior = globalThis.FormData;
  const claves = [];
  let recargas = 0;
  const estado = { bolsaSeleccionada: "bolsa:uno", modalFicha: { candidato: { estado_clave: "renuncia", estado_desde: "2026-09-23T08:00:00Z", participacion_ref: "participacion:dos" } } };
  globalThis.FormData = class { constructor(formulario) { this.valores = formulario.valores; } get(clave) { return this.valores[clave]; } has(clave) { return Boolean(this.valores[clave]); } };
  globalThis.fetch = async (_ruta, opciones) => {
    if (opciones.method === "GET") return response(200, { data: { esquema: "vec.bolsa.rrhh.operaciones_situacion.v1", items: [] } });
    claves.push(opciones.headers["Idempotency-Key"]);
    return claves.length === 1 ? response(503, { error: { codigo: "servicio_no_disponible" } })
      : response(201, { data: { recibo_ref: "recibo:1", situacion: "en_revision", desde: "2026-09-24T08:00:00Z", reutilizada: false } });
  };
  try {
    const controlador = crearControladorOperacionesSituacion({ estado, renderizar() {}, recargar: async () => { recargas += 1; } });
    const click = (accion, operacion) => controlador.manejarClick({ target: { closest: () => ({ dataset: { b8Accion: accion, operacion } }) }, preventDefault() {} });
    const submit = (valores) => controlador.manejarSubmit({ target: { closest: () => ({ valores }) }, preventDefault() {} });
    estado.modalFicha.operacionesB8 = { carga: "listo", items: [], transiciones: { renuncia: ["en_revision"], en_revision: ["disponible"] } };
    click("seleccionar", "revisar");
    submit({ motivo: "Solicitud" });
    submit({ tipo: "solicitud_candidato", referencia: "EXP-1", sha256: "A".repeat(64) });
    submit({ validador: "per_validadora" });
    await new Promise((resolver) => setTimeout(resolver, 0));
    assert.match(estado.modalFicha.operacionesB8.errorOperacion, /no está disponible/);
    submit({ validador: "per_validadora" });
    await new Promise((resolver) => setTimeout(resolver, 0));
    assert.deepEqual(claves, [claves[0], claves[0]]);
    assert.equal(estado.modalFicha.operacionesB8.recibo, "recibo:1");
    assert.equal(estado.modalFicha.candidato.estado_clave, "en_revision");
    assert.equal(recargas, 1);
  } finally {
    globalThis.fetch = fetchAnterior;
    globalThis.FormData = formDataAnterior;
  }
});

test("RRHH18 conserva restricciones de catálogos anteriores sin acciones de suspensión", () => {
  const antigua = { renuncia: ["disponible", "excluido"], disponible: ["no_disponible", "excluido"], excluido: ["disponible"] };
  assert.deepEqual(operacionesDisponibles("renuncia", antigua), ["excluir"]);
  assert.deepEqual(operacionesDisponibles("disponible", antigua), ["excluir"]);
  assert.deepEqual(operacionesDisponibles("excluido", antigua), []);
  assert.deepEqual(operacionesDisponibles("excluido", null), []);
  assert.deepEqual(operacionesDisponibles("trabajando", { trabajando: ["excluido"] }), ["excluir"]);
  assert.deepEqual(operacionesDisponibles("renuncia", { ...transicionesRRHH18, renuncia: ["excluido"] }), ["excluir"]);
});

test("RRHH18 reincorpora una exclusión solo por la política vigente con documento fin de causa", () => {
  const candidato = { estado_clave: "excluido", estado_desde: desde };
  const estado = { carga: "listo", items: [], transiciones: transicionesRRHH18 };
  const ficha = renderizarOperacionesSituacion({ candidato, estado });
  assert.match(ficha, /data-operacion="regularizar">Reincorporar con justificante validado<\/button>/);
  assert.match(ficha, /una sanción vigente la impide/);
  const revision = renderizarOperacionesSituacion({ candidato, estado: { ...estado, paso: 3, operacion: "regularizar", formulario: { motivo: "Justificante validado", tipo: "solicitud_candidato", referencia: "REG-2026/15", causa_finalizada_en: "2026-09-22" } } });
  assert.match(revision, /<dt>Operación seleccionada:<\/dt><dd>Reincorporar con justificante validado<\/dd>/);
  assert.match(revision, /Personal de RRHH que valida el justificante/);
  assert.match(revision, /Confirmar vuelta a Disponible/);
  assert.match(revision, /Fecha de fin de la causa/);
  assert.match(revision, /name="confirma_fin_causa" required/);
});

test("una referencia con huella del sistema no se toma por un documento de identidad", () => {
  assert.equal(referenciaContieneDocumentoIdentidad("llamamiento:823ae25fabcdefabcdefabcdefabcdefabcdefabcdefabcda31969244d148227"), false);
  assert.equal(referenciaContieneDocumentoIdentidad("justificante:12345678Z"), true);
  assert.equal(referenciaContieneDocumentoIdentidad("dni:" + "a".repeat(64)), true);
});

test("el historial de operaciones presenta fecha local, situación legible y papel en vez de referencias", () => {
  const html = renderizarOperacionesSituacion({ candidato: { estado_clave: "no_disponible" }, estado: { carga: "listo", items: [{
    desde: "2026-09-23T08:00:00.123456Z", operacion: "pausar", situacion: "no_disponible", motivo: "Solicitud",
    justificante: { tipo: "correo", referencia: "REG-2026/15", sha256: "a".repeat(64) }, actor: "per_rrhh", validador: "Ana Ruiz", validada_en: "2026-09-23T09:30:00Z",
  }] } });
  const visible = html.replace(/data-[a-z-]+="[^"]*"/gu, "");
  assert.doesNotMatch(visible, /per_rrhh|2026-09-23T|no_disponible|a{64}/u);
  assert.match(html, /23\/9\/26, 10:00/u);
  assert.match(html, /<td>No disponible<\/td>/u);
  assert.match(html, /Personal de RRHH <button[^>]*data-copiar-justificante="per_rrhh"/u);
  assert.match(html, /\/ Ana Ruiz<br>23\/9\/26, 11:30/u);
  assert.match(html, /REG-2026\/15/u);
});

test("B8 permite revisar la operación y el justificante antes de confirmar, con datos escapados", () => {
  const estado = { carga: "listo", items: [], operacion: "excluir", paso: 3, formulario: {
    motivo: '<img src=x onerror="alert(1)"> & solicitud',
    tipo: "correo", referencia: 'REG-1"><svg onload="alert(2)">',
    sha256: "a".repeat(64), validador: 'Ana "Ruiz"', confirma_validador_distinto: true,
  } };
  const html = renderizarOperacionesSituacion({ candidato: { estado_clave: "disponible" }, estado });
  assert.match(html, /<legend>Revisión<\/legend><dl class="resumen-expediente">/);
  assert.match(html, /<dt>Operación seleccionada:<\/dt><dd>Excluir<\/dd>/);
  assert.match(html, /<dt>Motivo<\/dt><dd>&lt;img src=x onerror=&quot;alert\(1\)&quot;&gt; &amp; solicitud<\/dd>/);
  assert.match(html, /<dt>Tipo de justificante<\/dt><dd>Correo<\/dd>/);
  assert.match(html, /<dt>Referencia del documento en su custodia<\/dt><dd>REG-1&quot;&gt;&lt;svg onload=&quot;alert\(2\)&quot;&gt;<\/dd>/);
  assert.doesNotMatch(html, /<img|<svg|a{64}/);
  assert.match(html, /name="validador"[^>]*value="Ana &quot;Ruiz&quot;"/);
  assert.match(html, /name="confirma_validador_distinto" required checked/);
  assert.match(html, /Confirmar Excluir/);
  assert.doesNotMatch(renderizarOperacionesSituacion({ candidato: { estado_clave: "disponible" }, estado: { ...estado, paso: 2 } }), /<legend>Revisión<\/legend>/);
});

test("B8 vuelve para corregir sin perder datos escritos ni enviar antes de la confirmación", async () => {
  const fetchAnterior = globalThis.fetch;
  const formDataAnterior = globalThis.FormData;
  const peticiones = [];
  globalThis.FormData = class {
    constructor(formulario) { this.valores = formulario.valores; }
    get(campo) { return this.valores[campo] ?? null; }
    has(campo) { return Boolean(this.valores[campo]); }
  };
  globalThis.fetch = async (ruta, opciones) => {
    peticiones.push({ ruta, opciones });
    return response(503, { error: { codigo: "servicio_no_disponible" } });
  };
  try {
    const estado = { bolsaSeleccionada: "bolsa:uno", modalFicha: { candidato: { estado_clave: "renuncia", estado_desde: "2026-09-23T08:00:00Z", participacion_ref: "participacion:dos" } } };
    const controlador = crearControladorOperacionesSituacion({ estado, renderizar() {}, recargar() {} });
    const click = (accion, valores = {}) => controlador.manejarClick({
      target: { closest: () => ({ dataset: { b8Accion: accion, operacion: "excluir" }, closest: () => ({ valores }) }) }, preventDefault() {},
    });
    const submit = (valores) => controlador.manejarSubmit({ target: { closest: () => ({ valores }) }, preventDefault() {} });
    click("seleccionar");
    submit({ motivo: "Solicitud original" });
    submit({ tipo: "correo", referencia: "REG-1", sha256: "a".repeat(64) });
    const flujo = estado.modalFicha.operacionesB8;
    assert.equal(flujo.paso, 3);
    click("anterior", { validador: "Ana Ruiz", confirma_validador_distinto: true });
    assert.equal(flujo.paso, 2);
    assert.equal(flujo.formulario.validador, "Ana Ruiz");
    assert.equal(flujo.formulario.confirma_validador_distinto, true);
    click("anterior", { tipo: "resolucion", referencia: "REG-2", sha256: "b".repeat(64) });
    assert.equal(flujo.paso, 1);
    assert.deepEqual(flujo.formulario, { motivo: "Solicitud original", tipo: "resolucion", referencia: "REG-2", sha256: "b".repeat(64), validador: "Ana Ruiz", confirma_validador_distinto: true, causa_finalizada_en: "", confirma_fin_causa: false });
    submit({ motivo: "Solicitud corregida" });
    const justificante = renderizarOperacionesSituacion({ candidato: estado.modalFicha.candidato, estado: flujo });
    assert.match(justificante, /value="REG-2"/);
    assert.match(justificante, /name="sha256" value="b{64}"/);
    submit({ tipo: flujo.formulario.tipo, referencia: flujo.formulario.referencia, sha256: flujo.formulario.sha256 });
    const revision = renderizarOperacionesSituacion({ candidato: estado.modalFicha.candidato, estado: flujo });
    assert.match(revision, /<dd>Solicitud corregida<\/dd>/);
    assert.match(revision, /<dd>REG-2<\/dd>/);
    assert.match(revision, /value="Ana Ruiz"/);
    assert.match(revision, /name="confirma_validador_distinto" required checked/);
    assert.equal(peticiones.length, 0);
    submit({ validador: "Ana Ruiz", confirma_validador_distinto: true });
    await new Promise((resolver) => setTimeout(resolver, 0));
    assert.equal(peticiones.length, 1);
    assert.equal(peticiones[0].opciones.method, "POST");
    assert.equal(peticiones[0].ruta, rutaOperacionesSituacion("bolsa:uno", "participacion:dos"));
    assert.deepEqual(JSON.parse(peticiones[0].opciones.body), { operacion: "excluir", motivo: "Solicitud corregida", validador: "Ana Ruiz", justificante: { tipo: "resolucion", referencia: "REG-2", sha256: "b".repeat(64) } });
    assert.ok(peticiones[0].opciones.headers["Idempotency-Key"]);
  } finally {
    globalThis.fetch = fetchAnterior;
    globalThis.FormData = formDataAnterior;
  }
});

test("B8 permite volver aunque haya datos incompletos o una referencia pendiente de corregir", () => {
  const formDataAnterior = globalThis.FormData;
  globalThis.FormData = class {
    constructor(formulario) { this.valores = formulario.valores; }
    get(campo) { return this.valores[campo] ?? null; }
    has(campo) { return Boolean(this.valores[campo]); }
  };
  try {
    const flujo = { carga: "listo", items: [], operacion: "excluir", paso: 3, formulario: { validador: "Ana", confirma_validador_distinto: true } };
    const estado = { modalFicha: { candidato: { estado_clave: "disponible" }, operacionesB8: flujo } };
    const controlador = crearControladorOperacionesSituacion({ estado, renderizar() {}, recargar() {} });
    const volver = (valores) => controlador.manejarClick({ target: { closest: () => ({ dataset: { b8Accion: "anterior" }, closest: () => ({ valores }) }) }, preventDefault() {} });
    volver({ validador: "" });
    assert.equal(flujo.paso, 2);
    assert.equal(flujo.formulario.validador, "");
    assert.equal(flujo.formulario.confirma_validador_distinto, false);
    volver({ tipo: "", referencia: "X1234567L", sha256: "" });
    assert.equal(flujo.paso, 1);
    assert.equal(flujo.formulario.referencia, "X1234567L");
    assert.equal(flujo.formulario.sha256, "");
  } finally { globalThis.FormData = formDataAnterior; }
});
