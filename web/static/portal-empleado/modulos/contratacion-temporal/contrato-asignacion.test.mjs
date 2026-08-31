import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import * as contrato from "./contrato-asignacion.js";

const {
  validarReciboAsignacion,
  validarSolicitudAsignacion,
  validarSolicitudReasignacion,
} = contrato;

const UUID = "a1111111-b222-4c33-8d44-e55555555555";
const ESQUEMA = "vec.contratacion-temporal.recibo-asignacion.v1";

function solicitudAsignacion(cambios = {}) {
  return {
    expediente_ref: "expediente:asignacion:001",
    version_esperada: 1,
    clave_idempotencia: UUID,
    unidad_ref: "unidad:gestora:001",
    responsable_ref: "responsable:interno:001",
    ...cambios,
  };
}

function solicitudReasignacion(cambios = {}) {
  return {
    ...solicitudAsignacion({ version_esperada: 2 }),
    motivo_reasignacion_clave: "cambio_unidad",
    observaciones: "Cambio motivado\ncon detalle\tinterno.",
    ...cambios,
  };
}

function reciboAsignacion(cambios = {}) {
  return {
    esquema: ESQUEMA,
    operacion: "asignar",
    expediente_ref: "expediente:asignacion:001",
    version_resultante: 2,
    recibo_ref: "recibo:asignacion:001",
    confirmada_en: "2026-08-31T10:20:30.123456Z",
    ...cambios,
  };
}

function exigirTypeError(funcion, entrada) {
  assert.throws(() => funcion(entrada), TypeError);
}

test("exporta solo los tres validadores del contrato", () => {
  assert.deepEqual(Object.keys(contrato).sort(), [
    "validarReciboAsignacion",
    "validarSolicitudAsignacion",
    "validarSolicitudReasignacion",
  ]);
});

test("valida las tres formas nominales como copias nuevas y congeladas", () => {
  const asignacion = solicitudAsignacion();
  const reasignacion = solicitudReasignacion();
  const recibo = reciboAsignacion();

  const asignacionValidada = validarSolicitudAsignacion(asignacion);
  const reasignacionValidada = validarSolicitudReasignacion(reasignacion);
  const reciboValidado = validarReciboAsignacion(recibo);

  assert.deepEqual(asignacionValidada, asignacion);
  assert.deepEqual(reasignacionValidada, reasignacion);
  assert.deepEqual(reciboValidado, recibo);
  assert.notEqual(asignacionValidada, asignacion);
  assert.notEqual(reasignacionValidada, reasignacion);
  assert.notEqual(reciboValidado, recibo);
  assert.equal(Object.isFrozen(asignacionValidada), true);
  assert.equal(Object.isFrozen(reasignacionValidada), true);
  assert.equal(Object.isFrozen(reciboValidado), true);

  asignacion.expediente_ref = "expediente:alterado:001";
  reasignacion.observaciones = "Contenido alterado";
  recibo.recibo_ref = "recibo:alterado:001";
  assert.equal(asignacionValidada.expediente_ref, "expediente:asignacion:001");
  assert.equal(
    reasignacionValidada.observaciones,
    "Cambio motivado\ncon detalle\tinterno.",
  );
  assert.equal(reciboValidado.recibo_ref, "recibo:asignacion:001");
});

test("rechaza cada campo ausente y cualquier campo extra", () => {
  const casos = [
    [validarSolicitudAsignacion, solicitudAsignacion, [
      "expediente_ref", "version_esperada", "clave_idempotencia",
      "unidad_ref", "responsable_ref",
    ]],
    [validarSolicitudReasignacion, solicitudReasignacion, [
      "expediente_ref", "version_esperada", "clave_idempotencia",
      "unidad_ref", "responsable_ref", "motivo_reasignacion_clave",
      "observaciones",
    ]],
    [validarReciboAsignacion, reciboAsignacion, [
      "esquema", "operacion", "expediente_ref", "version_resultante",
      "recibo_ref", "confirmada_en",
    ]],
  ];
  for (const [validar, crear, campos] of casos) {
    for (const campo of campos) {
      const entrada = crear();
      delete entrada[campo];
      exigirTypeError(validar, entrada);
    }
    exigirTypeError(validar, { ...crear(), campo_extra: "valor:sintetico:001" });
  }
});

test("rechaza autoridad inyectada en las intenciones y datos internos en el recibo", () => {
  const autoridad = [
    "autenticacion_ref", "sesion_ref", "actor_ref", "identidad_ref",
    "perfil_ref", "organizacion_ref", "rol_ref", "permiso_ref",
    "decision_ref",
  ];
  for (const campo of autoridad) {
    exigirTypeError(validarSolicitudAsignacion, {
      ...solicitudAsignacion(),
      [campo]: "referencia:inyectada:001",
    });
    exigirTypeError(validarSolicitudReasignacion, {
      ...solicitudReasignacion(),
      [campo]: "referencia:inyectada:001",
    });
  }
  for (const campo of [
    "organizacion_ref", "unidad_ref", "responsable_ref", "notificacion_ref",
    "auditoria_ref", "evento_ref", "decision_ref", "hmac",
  ]) {
    exigirTypeError(validarReciboAsignacion, {
      ...reciboAsignacion(),
      [campo]: "referencia:interna:001",
    });
  }
});

test("solo admite objetos ordinarios con propiedades de datos enumerables", () => {
  const casos = [
    [validarSolicitudAsignacion, solicitudAsignacion, "expediente_ref"],
    [validarSolicitudReasignacion, solicitudReasignacion, "observaciones"],
    [validarReciboAsignacion, reciboAsignacion, "confirmada_en"],
  ];
  for (const [validar, crear, campo] of casos) {
    exigirTypeError(validar, null);
    exigirTypeError(validar, []);

    const sinPrototipo = Object.assign(Object.create(null), crear());
    exigirTypeError(validar, sinPrototipo);

    const prototipoPropio = Object.create({ marca: true });
    Object.assign(prototipoPropio, crear());
    exigirTypeError(validar, prototipoPropio);

    const conSimbolo = crear();
    conSimbolo[Symbol("campo")] = "valor";
    exigirTypeError(validar, conSimbolo);

    const noEnumerable = crear();
    Object.defineProperty(noEnumerable, campo, {
      value: noEnumerable[campo],
      enumerable: false,
    });
    exigirTypeError(validar, noEnumerable);

    let accesos = 0;
    const conAccesor = crear();
    Object.defineProperty(conAccesor, campo, {
      enumerable: true,
      get() {
        accesos += 1;
        throw new Error("el accesor no debe ejecutarse");
      },
    });
    exigirTypeError(validar, conAccesor);
    assert.equal(accesos, 0);
  }
});

test("aplica alfabeto y límites exactos a todas las referencias", () => {
  const minima = "a.b";
  const maxima = `a${"b".repeat(159)}`;
  const alfa = "A0._:/#-z";
  for (const valor of [minima, maxima, alfa]) {
    assert.equal(validarSolicitudAsignacion(solicitudAsignacion({
      expediente_ref: valor,
      unidad_ref: valor,
      responsable_ref: valor,
    })).expediente_ref, valor);
    assert.equal(validarReciboAsignacion(reciboAsignacion({
      expediente_ref: valor,
      recibo_ref: valor,
    })).recibo_ref, valor);
  }

  const invalidas = [
    "ab", `a${"b".repeat(160)}`, "_ab", "a b", "ábc", "a?b", "", 123,
  ];
  for (const valor of invalidas) {
    for (const campo of ["expediente_ref", "unidad_ref", "responsable_ref"]) {
      exigirTypeError(
        validarSolicitudAsignacion,
        solicitudAsignacion({ [campo]: valor }),
      );
    }
    for (const campo of ["expediente_ref", "recibo_ref"]) {
      exigirTypeError(
        validarReciboAsignacion,
        reciboAsignacion({ [campo]: valor }),
      );
    }
  }
});

test("distingue las fronteras de versión de intención y recibo", () => {
  for (const version of [1, Number.MAX_SAFE_INTEGER - 1]) {
    assert.equal(
      validarSolicitudAsignacion(
        solicitudAsignacion({ version_esperada: version }),
      ).version_esperada,
      version,
    );
  }
  for (const version of [0, -1, 1.5, NaN, Infinity, Number.MAX_SAFE_INTEGER]) {
    exigirTypeError(
      validarSolicitudAsignacion,
      solicitudAsignacion({ version_esperada: version }),
    );
  }
  for (const version of [2, Number.MAX_SAFE_INTEGER]) {
    assert.equal(
      validarReciboAsignacion(
        reciboAsignacion({ version_resultante: version }),
      ).version_resultante,
      version,
    );
  }
  for (const version of [1, 2.5, Number.MAX_SAFE_INTEGER + 1, Infinity]) {
    exigirTypeError(
      validarReciboAsignacion,
      reciboAsignacion({ version_resultante: version }),
    );
  }
});

test("acepta solo UUIDv4 canónica minúscula y no nula", () => {
  assert.equal(
    validarSolicitudAsignacion(solicitudAsignacion()).clave_idempotencia,
    UUID,
  );
  for (const valor of [
    UUID.toUpperCase(),
    "11111111-2222-5333-8444-555555555555",
    "11111111-2222-4333-7444-555555555555",
    "00000000-0000-4000-8000-000000000000",
    "11111111222243338444555555555555",
    "",
    123,
  ]) {
    exigirTypeError(
      validarSolicitudAsignacion,
      solicitudAsignacion({ clave_idempotencia: valor }),
    );
  }
});

test("aplica al motivo de reasignación su alfabeto y longitudes", () => {
  for (const valor of ["a1", `a${"b".repeat(79)}`, "cambio.unidad-1"] ) {
    assert.equal(
      validarSolicitudReasignacion(solicitudReasignacion({
        motivo_reasignacion_clave: valor,
      })).motivo_reasignacion_clave,
      valor,
    );
  }
  for (const valor of ["a", `a${"b".repeat(80)}`, "Ajuste", "a:b", "", 7]) {
    exigirTypeError(
      validarSolicitudReasignacion,
      solicitudReasignacion({ motivo_reasignacion_clave: valor }),
    );
  }
});

test("valida observaciones NFC por puntos de código y permite newline y tab internos", () => {
  const astrales1000 = "😀".repeat(1000);
  for (const valor of ["Texto", "Línea uno\nLínea dos\tfin", astrales1000]) {
    assert.equal(
      validarSolicitudReasignacion(
        solicitudReasignacion({ observaciones: valor }),
      ).observaciones,
      valor,
    );
  }
  for (const valor of [
    "",
    " texto",
    "texto ",
    "\ntexto",
    "texto\t",
    "Cafe\u0301",
    "texto\u0000interno",
    "texto\u0085interno",
    "texto\uD800interno",
    "😀".repeat(1001),
  ]) {
    exigirTypeError(
      validarSolicitudReasignacion,
      solicitudReasignacion({ observaciones: valor }),
    );
  }
});

test("cierra esquema y operación del recibo", () => {
  for (const operacion of ["asignar", "reasignar"]) {
    assert.equal(
      validarReciboAsignacion(reciboAsignacion({ operacion })).operacion,
      operacion,
    );
  }
  for (const cambios of [
    { esquema: "vec.contratacion-temporal.recibo-asignacion.v2" },
    { operacion: "registrar" },
    { operacion: "ASIGNAR" },
  ]) {
    exigirTypeError(validarReciboAsignacion, reciboAsignacion(cambios));
  }
});

test("acepta solo instantes UTC canónicos, civiles reales y con microsegundos", () => {
  for (const valor of [
    "0001-01-01T00:00:00Z",
    "2000-02-29T23:59:59.000001Z",
    "9999-12-31T23:59:59.999999Z",
  ]) {
    assert.equal(
      validarReciboAsignacion(
        reciboAsignacion({ confirmada_en: valor }),
      ).confirmada_en,
      valor,
    );
  }
  for (const valor of [
    "0000-01-01T00:00:00Z",
    "10000-01-01T00:00:00Z",
    "2026-02-29T00:00:00Z",
    "2026-04-31T00:00:00Z",
    "2026-01-01T24:00:00Z",
    "2026-01-01T00:60:00Z",
    "2026-01-01T00:00:60Z",
    "2026-01-01T00:00:00+00:00",
    "2026-01-01T00:00:00z",
    "2026-01-01T00:00:00.1234567Z",
    "2026-01-01T00:00:00.120Z",
    "2026-1-01T00:00:00Z",
  ]) {
    exigirTypeError(
      validarReciboAsignacion,
      reciboAsignacion({ confirmada_en: valor }),
    );
  }
});

test("los TypeError no incorporan datos rechazados", () => {
  const dato = "dato personal sintético no repetir";
  assert.throws(
    () => validarSolicitudAsignacion(
      solicitudAsignacion({ expediente_ref: dato }),
    ),
    (error) => error instanceof TypeError && !error.message.includes(dato),
  );
});

test("la importación no produce efectos y el productivo carece de superficies prohibidas", async () => {
  const ruta = new URL("./contrato-asignacion.js", import.meta.url);
  const fuente = await readFile(ruta, "utf8");
  const globalesAntes = Reflect.ownKeys(globalThis);
  await import(`${ruta.href}?sin-efectos=${Date.now()}`);
  assert.deepEqual(Reflect.ownKeys(globalThis), globalesAntes);

  for (const patron of [
    /\bfetch\b|XMLHttpRequest|WebSocket|EventSource/u,
    /\bdocument\b|\bwindow\b|\bnavigator\b|\bDOM\b/u,
    /localStorage|sessionStorage|indexedDB|\bcookie\b/u,
    /autenticacion|autorizacion|autoridad|identidad|sesion|actor|perfil|organizacion|rol|permiso|decision/iu,
  ]) {
    assert.doesNotMatch(fuente, patron);
  }
});
