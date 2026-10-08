import test from "node:test";
import assert from "node:assert/strict";
import { crearFuenteTramitesPropios } from "./fuente-tramites-propios.js";

function cronos() {
  return {
    anio: 2026,
    permisos: [{ permiso_ref: "permiso:cronos:asuntos-propios", version_ref: "catalogo:cronos:asuntos-propios:v1", nombre: "Asuntos propios",
      unidad: "dia", computo: "laborables", circuito: "J-A", minimo: 1, maximo_solicitud: null, maximo_mensual: null, maximo_anual: 6,
      justificante_exigido: false, solicitable: true, sintetico: true, solicitado: 1, concedido: 0, pendiente_justificar: 0, resta: 5 }],
    solicitudes: [{ solicitud_ref: "permiso:cronos:solicitud:clave000001", catalogo_version_ref: "catalogo:cronos:asuntos-propios:v1",
      permiso_ref: "permiso:cronos:asuntos-propios", desde: "2026-10-05", hasta: "2026-10-06", cantidad: 2, unidad: "dia", estado: "solicitado",
      version: 1, pendiente_justificar: false, solicitada_en: "2026-10-01T10:00:00.123456Z", circuito: "J-A", pendiente_asignacion: true }],
  };
}
function dietas() {
  return { items: [{ comision: { referencia: `dco_${"a".repeat(22)}`, version: 3, numero_documento: "VEC-D-2026-000001",
    estado: "pendiente_autorizacion", fecha_inicio: "2026-10-05", fecha_fin: "2026-10-06", fecha_apertura: "2026-10-01T10:00:00.123456Z",
    motivo: "Dato reservado", codigos_ruta: ["ruta_reservada"], relacion_ref: `rel_${"b".repeat(22)}`, centro_ref: "centro_reservado",
    unidad_ref: "unidad_reservada", rutas: [{ origen: "origen reservado" }], documento: { total_orientativo_centimos: 12345 } },
    recibo: { referencia: `rcd_${"c".repeat(22)}`, version: 3, registrado_en: "2026-10-01T10:00:00.123456Z", repeticion: false,
      actor: "persona reservada" }, motivo: "campo ajeno" }], siguiente_cursor: "cursor:pagina:2", empleado_ref: "empleado reservado" };
}
const codigo = (esperado) => (error) => error.codigo === esperado && error.message === "";

test("expone solo las lecturas inyectadas y disponibilidad inmutable", async () => {
  const fuente = crearFuenteTramitesPropios();
  assert.deepEqual(Object.keys(fuente), ["disponibles", "consultarCronos", "consultarDietas"]);
  assert.deepEqual(fuente.disponibles, { cronos: false, dietas: false });
  assert.ok(Object.isFrozen(fuente) && Object.isFrozen(fuente.disponibles));
  await assert.rejects(fuente.consultarCronos({ anio: 2026 }), codigo("fuente_no_configurada"));
  await assert.rejects(fuente.consultarDietas(), codigo("fuente_no_configurada"));
  assert.throws(() => crearFuenteTramitesPropios({ consultarPermisos: {} }), codigo("fuente_no_configurada"));
});

test("Cronos consulta solo el año y proyecta el nombre del catálogo sin inventar recibo", async () => {
  const original = cronos(); const llamadas = []; const controlador = new AbortController();
  const fuente = crearFuenteTramitesPropios({ consultarPermisos: async (...args) => { llamadas.push(args); return original; } });
  const resultado = await fuente.consultarCronos({ anio: 2026, signal: controlador.signal });
  assert.deepEqual(llamadas, [[{ anio: 2026 }, { signal: controlador.signal }]]);
  assert.deepEqual(resultado, { anio: 2026, solicitudes: [{ solicitud_ref: original.solicitudes[0].solicitud_ref,
    nombre: "Asuntos propios", estado: "solicitado", version: 1, solicitada_en: original.solicitudes[0].solicitada_en,
    desde: "2026-10-05", hasta: "2026-10-06", pendiente_justificar: false, circuito: "J-A", pendiente_asignacion: true }] });
  assert.ok(Object.isFrozen(resultado) && Object.isFrozen(resultado.solicitudes) && Object.isFrozen(resultado.solicitudes[0]));
  original.solicitudes[0].estado = "concedido"; original.permisos[0].nombre = "Alterado";
  assert.equal(resultado.solicitudes[0].estado, "solicitado"); assert.equal(resultado.solicitudes[0].nombre, "Asuntos propios");
});

test("un catálogo sin coincidencia no crea un nombre ni copia la referencia de permiso", async () => {
  const original = cronos(); original.permisos = [];
  const resultado = await crearFuenteTramitesPropios({ consultarPermisos: async () => original }).consultarCronos({ anio: 2026 });
  assert.equal(Object.hasOwn(resultado.solicitudes[0], "nombre"), false);
  assert.equal(Object.hasOwn(resultado.solicitudes[0], "permiso_ref"), false);
  assert.equal(Object.hasOwn(resultado.solicitudes[0], "recibo"), false);
});

test("Dietas consulta una sola página sin relación y conserva solo comisión mínima y recibo real", async () => {
  const original = dietas(); const llamadas = []; const controlador = new AbortController();
  const fuente = crearFuenteTramitesPropios({ listarComisiones: async (...args) => { llamadas.push(args); return original; } });
  const resultado = await fuente.consultarDietas({ cursor: "cursor:pagina:1", signal: controlador.signal });
  assert.deepEqual(llamadas, [[{ limit: 20, cursor: "cursor:pagina:1" }, { signal: controlador.signal }]]);
  assert.deepEqual(Object.keys(resultado), ["items", "siguiente_cursor"]);
  assert.deepEqual(Object.keys(resultado.items[0].comision), ["referencia", "version", "estado", "fecha_inicio", "fecha_fin", "numero_documento", "fecha_apertura"]);
  assert.deepEqual(Object.keys(resultado.items[0].recibo), ["referencia", "version", "registrado_en", "repeticion"]);
  assert.equal(resultado.siguiente_cursor, "cursor:pagina:2");
  assert.doesNotMatch(JSON.stringify(resultado), /reservad|relacion_ref|codigos_ruta|importe|motivo|total_orientativo/u);
  for (const objeto of [resultado, resultado.items, resultado.items[0], resultado.items[0].comision, resultado.items[0].recibo]) assert.ok(Object.isFrozen(objeto));
  original.items[0].comision.estado = "fiscalizada"; original.items[0].recibo.repeticion = true;
  assert.equal(resultado.items[0].comision.estado, "pendiente_autorizacion"); assert.equal(resultado.items[0].recibo.repeticion, false);
});

test("Dietas omite metadatos ausentes y no fabrica un recibo", async () => {
  const original = dietas(); delete original.items[0].recibo; delete original.siguiente_cursor;
  delete original.items[0].comision.fecha_apertura; delete original.items[0].comision.numero_documento; delete original.items[0].comision.version;
  const resultado = await crearFuenteTramitesPropios({ listarComisiones: async () => original }).consultarDietas();
  assert.deepEqual(Object.keys(resultado), ["items"]); assert.deepEqual(Object.keys(resultado.items[0]), ["comision"]);
  assert.deepEqual(Object.keys(resultado.items[0].comision), ["referencia", "estado", "fecha_inicio", "fecha_fin"]);
  assert.equal(Object.hasOwn(resultado.items[0].comision, "version"), false);
});

function dietasDevuelta() {
  const original = dietas();
  Object.assign(original.items[0].comision, { estado: "devuelta", version: 4,
    devolucion: { etapa: "revision", motivo: "Corrige el recorrido declarado.", version: 3, devuelta_en: "2026-10-03T10:00:00.123456Z" } });
  return original;
}

test("Dietas conserva la devolución propia exacta, separada del motivo de la comisión e inmutable", async () => {
  const original = dietasDevuelta();
  const resultado = await crearFuenteTramitesPropios({ listarComisiones: async () => original }).consultarDietas();
  const comision = resultado.items[0].comision;
  assert.equal(comision.estado, "devuelta");
  assert.deepEqual(comision.devolucion, original.items[0].comision.devolucion);
  assert.deepEqual(Object.keys(comision.devolucion), ["etapa", "motivo", "version", "devuelta_en"]);
  assert.ok(Object.isFrozen(comision.devolucion));
  assert.deepEqual(resultado.items[0].recibo, { referencia: `rcd_${"c".repeat(22)}`, version: 3,
    registrado_en: "2026-10-01T10:00:00.123456Z", repeticion: false });
  assert.doesNotMatch(JSON.stringify(resultado), /Dato reservado|relacion_ref|persona reservada|plazo|vencimiento/);
  original.items[0].comision.devolucion.motivo = "Alterado";
  assert.equal(comision.devolucion.motivo, "Corrige el recorrido declarado.");
  assert.throws(() => comision.devolucion.version = 4, TypeError);
});

test("la devolución respeta etapas, estado en corrección y texto sin transformarlo en HTML ni plazo", async () => {
  for (const estado of ["devuelta", "borrador"]) for (const etapa of ["revision", "autorizacion", "liquidacion", "fiscalizacion"]) {
    const original = dietasDevuelta();
    original.items[0].comision.estado = estado;
    Object.assign(original.items[0].comision.devolucion, { etapa, motivo: '<img src=x onerror="alert(1)"> & \'', version: 4 });
    const resultado = await crearFuenteTramitesPropios({ listarComisiones: async () => original }).consultarDietas();
    assert.equal(resultado.items[0].comision.estado, estado);
    assert.deepEqual(resultado.items[0].comision.devolucion, original.items[0].comision.devolucion);
  }
  const original = dietasDevuelta(); original.items[0].comision.devolucion.motivo = "é".repeat(300);
  const resultado = await crearFuenteTramitesPropios({ listarComisiones: async () => original }).consultarDietas();
  assert.equal(resultado.items[0].comision.devolucion.motivo, "é".repeat(300));
});

test("devoluciones ajenas al contrato, incompatibles con la comisión o malformadas se rechazan sin detalle", async () => {
  for (const modificar of [
    c => c.devolucion = null, c => c.devolucion = [], c => c.devolucion = "devuelta",
    c => c.devolucion.actor_ref = "dato reservado", c => delete c.devolucion.motivo,
    c => c.devolucion.etapa = "otra", c => c.devolucion.etapa = {},
    c => c.estado = "fiscalizada", c => c.estado = "enviado_pendiente_revision",
    c => c.devolucion.version = 2, c => c.devolucion.version = 5, c => c.devolucion.version = "3",
    c => c.devolucion.version = 3.1, c => c.devolucion.version = Number.MAX_SAFE_INTEGER + 1,
    c => delete c.version, c => c.devolucion.motivo = "ab", c => c.devolucion.motivo = {},
    c => c.devolucion.motivo = "x".repeat(601), c => c.devolucion.motivo = "é".repeat(301),
    c => c.devolucion.motivo = " motivo", c => c.devolucion.motivo = "motivo\u0085",
    c => c.devolucion.motivo = "motivo\ncontenido", c => c.devolucion.motivo = "motivo\u007fcontenido",
    c => c.devolucion.devuelta_en = "2026-02-30T10:00:00.123456Z",
    c => c.devolucion.devuelta_en = "2026-10-03T24:00:00.123456Z",
    c => c.devolucion.devuelta_en = "2026-10-03T10:00:00.123Z",
    c => c.devolucion.devuelta_en = "2026-10-03T10:00:00.123456+00:00",
    c => c.devolucion.devuelta_en = "2026-10-03", c => c.devolucion.devuelta_en = {},
  ]) {
    const original = dietasDevuelta(); modificar(original.items[0].comision);
    await assert.rejects(crearFuenteTramitesPropios({ listarComisiones: async () => original }).consultarDietas(), codigo("respuesta_incompatible"));
  }
});

test("no acepta identidad, relación, límite o cursores Cronos suministrados por el consumidor", async () => {
  let llamadas = 0;
  const fuente = crearFuenteTramitesPropios({ consultarPermisos: async () => { llamadas++; return cronos(); }, listarComisiones: async () => { llamadas++; return dietas(); } });
  for (const entrada of [{ anio: 2026, empleado_ref: "emp_x" }, { anio: 2026, cursor: "abc" }, { anio: 1999 }, { anio: 2101 }, { anio: "2026" }, {}])
    await assert.rejects(fuente.consultarCronos(entrada), codigo("peticion_invalida"));
  for (const entrada of [{ relacion_ref: "rel_x" }, { limit: 50 }, { actor: "otra persona" }, { cursor: "" }, { cursor: "x\n" }, { cursor: "é".repeat(201) }, { cursor: {} }])
    await assert.rejects(fuente.consultarDietas(entrada), codigo("peticion_invalida"));
  assert.equal(llamadas, 0);
});

test("reutiliza las validaciones Cronos de fechas, año, estado y circuito", async () => {
  for (const modificar of [v => v.anio = 2027, v => v.solicitudes[0].desde = "2026-02-30", v => v.solicitudes[0].hasta = "2026-10-04",
    v => v.solicitudes[0].estado = "pagado", v => v.solicitudes[0].version = 0, v => v.solicitudes[0].circuito = "otro",
    v => v.solicitudes[0].solicitada_en = "2026-02-30T10:00:00Z", v => v.permisos[0].nombre = "Nombre\n",
    v => v.solicitudes = Array(5001).fill(v.solicitudes[0]), v => v.solicitudes[0].identidad = "ajena"]) {
    const original = cronos(); modificar(original);
    await assert.rejects(crearFuenteTramitesPropios({ consultarPermisos: async () => original }).consultarCronos({ anio: 2026 }), codigo("respuesta_incompatible"));
  }
});

test("rechaza referencias, estados, fechas y recibos Dietas malformados", async () => {
  for (const modificar of [v => v.items[0].comision.referencia = "dco_corta", v => v.items[0].comision.version = 0,
    v => v.items[0].comision.estado = "pagada", v => v.items[0].comision.fecha_inicio = "2026-02-30",
    v => v.items[0].comision.fecha_fin = "2026-10-04", v => v.items[0].comision.fecha_apertura = "2026-02-30T10:00:00Z",
    v => v.items[0].comision.numero_documento = {}, v => v.items[0].recibo.referencia = "inventada",
    v => v.items[0].recibo.registrado_en = "2026-10-01T24:00:00Z", v => v.items[0].recibo.repeticion = "false",
    v => v.items[0].recibo = null, v => v.items = Array(21).fill(v.items[0]), v => v.items = Array(1),
    v => v.siguiente_cursor = "x\n", v => v.siguiente_cursor = "é".repeat(201), v => v.items = {}]) {
    const original = dietas(); modificar(original);
    await assert.rejects(crearFuenteTramitesPropios({ listarComisiones: async () => original }).consultarDietas(), codigo("respuesta_incompatible"));
  }
});

test("acepta los estados propios de los módulos sin traducirlos ni dar por pagada una fiscalización", async () => {
  for (const estado of ["solicitado", "pendiente_administracion", "concedido", "denegado", "cancelado"]) {
    const valor = cronos(); valor.solicitudes[0].estado = estado; delete valor.solicitudes[0].pendiente_asignacion;
    const resultado = await crearFuenteTramitesPropios({ consultarPermisos: async () => valor }).consultarCronos({ anio: 2026 });
    assert.equal(resultado.solicitudes[0].estado, estado);
  }
  for (const estado of ["borrador", "eliminado", "enviado_pendiente_revision", "pendiente_autorizacion", "pendiente_liquidacion", "pendiente_fiscalizacion", "fiscalizada", "devuelta"]) {
    const valor = dietas(); valor.items[0].comision.estado = estado;
    assert.equal((await crearFuenteTramitesPropios({ listarComisiones: async () => valor }).consultarDietas()).items[0].comision.estado, estado);
  }
});

test("la relación ambigua conserva un código seguro y no intenta elegir o recorrer relaciones", async () => {
  let llamadas = 0;
  const fuente = crearFuenteTramitesPropios({ listarComisiones: async () => { llamadas++; throw Object.assign(new Error("Datos personales del servidor"), { codigo: "relacion_ambigua", detalles: "reservado" }); } });
  await assert.rejects(fuente.consultarDietas(), error => {
    assert.equal(error.codigo, "relacion_ambigua"); assert.equal(error.message, ""); assert.equal(error.detalles, undefined); return true;
  });
  assert.equal(llamadas, 1);
  for (const esperado of ["acceso_denegado", "autenticacion_requerida", "sin_empleado"]) {
    await assert.rejects(crearFuenteTramitesPropios({ consultarPermisos: async () => { throw { codigo: esperado }; } }).consultarCronos({ anio: 2026 }), codigo(esperado));
  }
  await assert.rejects(crearFuenteTramitesPropios({ listarComisiones: async () => { throw { codigo: "dato_personal_inyectado" }; } }).consultarDietas(), codigo("servicio_no_disponible"));
});

test("la cancelación previa no llama al cliente y un resultado tardío queda descartado", async () => {
  for (const origen of ["cronos", "dietas"]) {
    let llamadas = 0; let resolver; const controlador = new AbortController();
    const leer = async (_consulta, opciones) => { llamadas++; assert.equal(opciones.signal, controlador.signal); return new Promise(resolve => { resolver = resolve; }); };
    const fuente = crearFuenteTramitesPropios({ [origen === "cronos" ? "consultarPermisos" : "listarComisiones"]: leer });
    const consultar = () => origen === "cronos" ? fuente.consultarCronos({ anio: 2026, signal: controlador.signal }) : fuente.consultarDietas({ signal: controlador.signal });
    const pendiente = consultar(); controlador.abort(); resolver(origen === "cronos" ? cronos() : dietas());
    await assert.rejects(pendiente, codigo("operacion_abortada"));
    await assert.rejects(consultar(), codigo("operacion_abortada")); assert.equal(llamadas, 1);
  }
});

test("un error tardío tras cancelar también queda como cancelación", async () => {
  const controlador = new AbortController(); let rechazar;
  const fuente = crearFuenteTramitesPropios({ listarComisiones: () => new Promise((_resolver, reject) => { rechazar = reject; }) });
  const pendiente = fuente.consultarDietas({ signal: controlador.signal }); controlador.abort(); rechazar(new Error("red"));
  await assert.rejects(pendiente, codigo("operacion_abortada"));
  await assert.rejects(fuente.consultarDietas({ signal: {} }), codigo("peticion_invalida"));
});
