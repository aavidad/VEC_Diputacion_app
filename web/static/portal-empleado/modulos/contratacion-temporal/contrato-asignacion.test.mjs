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

function datosAsignacion(cambios = {}) {
  return {
    expediente_ref: "expediente:asignacion:001",
    version_esperada: 1,
    clave_idempotencia: UUID,
    unidad_ref: "unidad:gestora:001",
    responsable_ref: "responsable:interno:001",
    ...cambios,
  };
}

function datosReasignacion(cambios = {}) {
  return {
    ...datosAsignacion({ version_esperada: 2 }),
    motivo_reasignacion_clave: "cambio_unidad",
    observaciones: "Cambio motivado\ncon detalle\tinterno.",
    ...cambios,
  };
}

function datosRecibo(cambios = {}) {
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

function canon(datos) {
  return JSON.stringify(datos);
}

function exigirTypeError(funcion, entrada) {
  assert.throws(
    () => funcion(entrada),
    (error) => error instanceof TypeError
      && error.message === "contrato JSON no válido",
  );
}

function bytesUTF8(texto) {
  let bytes = 0;
  for (let indice = 0; indice < texto.length; indice += 1) {
    const unidad = texto.charCodeAt(indice);
    if (unidad <= 0x7F) bytes += 1;
    else if (unidad <= 0x7FF) bytes += 2;
    else if (unidad >= 0xD800 && unidad <= 0xDBFF
      && indice + 1 < texto.length
      && texto.charCodeAt(indice + 1) >= 0xDC00
      && texto.charCodeAt(indice + 1) <= 0xDFFF) {
      bytes += 4;
      indice += 1;
    } else bytes += 3;
  }
  return bytes;
}

test("exporta solo los tres validadores del contrato", () => {
  assert.deepEqual(Object.keys(contrato).sort(), [
    "validarReciboAsignacion",
    "validarSolicitudAsignacion",
    "validarSolicitudReasignacion",
  ]);
});

test("valida JSON nominal como salidas nuevas ordinarias, primitivas y congeladas", () => {
  const casos = [
    [validarSolicitudAsignacion, datosAsignacion()],
    [validarSolicitudReasignacion, datosReasignacion()],
    [validarReciboAsignacion, datosRecibo()],
  ];
  for (const [validar, esperado] of casos) {
    const texto = canon(esperado);
    const primera = validar(texto);
    const segunda = validar(texto);
    assert.deepEqual(primera, esperado);
    assert.deepEqual(segunda, esperado);
    assert.notEqual(primera, esperado);
    assert.notEqual(primera, segunda);
    assert.equal(Object.getPrototypeOf(primera), Object.prototype);
    assert.equal(Object.isFrozen(primera), true);
    for (const valor of Object.values(primera)) {
      assert.equal(["string", "number", "boolean"].includes(typeof valor), true);
    }
  }
});

test("rechaza objetos y proxies por tipo sin trampas, getters ni coerción", () => {
  const casos = [
    [validarSolicitudAsignacion, canon(datosAsignacion())],
    [validarSolicitudReasignacion, canon(datosReasignacion())],
    [validarReciboAsignacion, canon(datosRecibo())],
  ];
  for (const [validar, texto] of casos) {
    exigirTypeError(validar, new Proxy(JSON.parse(texto), {}));

    let trampas = 0;
    const trampa = () => {
      trampas += 1;
      throw new Error("no debe ejecutarse una trampa");
    };
    const hostil = new Proxy({}, {
      defineProperty: trampa,
      deleteProperty: trampa,
      get: trampa,
      getOwnPropertyDescriptor: trampa,
      getPrototypeOf: trampa,
      has: trampa,
      isExtensible: trampa,
      ownKeys: trampa,
      preventExtensions: trampa,
      set: trampa,
      setPrototypeOf: trampa,
    });
    exigirTypeError(validar, hostil);
    assert.equal(trampas, 0);

    const revocable = Proxy.revocable({}, {});
    revocable.revoke();
    exigirTypeError(validar, revocable.proxy);

    const cadenaObjeto = new Proxy(new String(texto), { get: trampa });
    exigirTypeError(validar, cadenaObjeto);

    let accesos = 0;
    const conGetter = {};
    Object.defineProperty(conGetter, "valor", {
      enumerable: true,
      get() {
        accesos += 1;
        return texto;
      },
    });
    Object.defineProperty(conGetter, Symbol.toPrimitive, {
      value() {
        accesos += 1;
        return texto;
      },
    });
    exigirTypeError(validar, conGetter);
    assert.equal(accesos, 0);
    assert.equal(trampas, 0);
  }
});

test("rechaza campos ausentes, extra, de autoridad y datos internos", () => {
  const casos = [
    [validarSolicitudAsignacion, datosAsignacion, [
      "expediente_ref", "version_esperada", "clave_idempotencia",
      "unidad_ref", "responsable_ref",
    ]],
    [validarSolicitudReasignacion, datosReasignacion, [
      "expediente_ref", "version_esperada", "clave_idempotencia",
      "unidad_ref", "responsable_ref", "motivo_reasignacion_clave",
      "observaciones",
    ]],
    [validarReciboAsignacion, datosRecibo, [
      "esquema", "operacion", "expediente_ref", "version_resultante",
      "recibo_ref", "confirmada_en",
    ]],
  ];
  for (const [validar, crear, campos] of casos) {
    for (const campo of campos) {
      const datos = crear();
      delete datos[campo];
      exigirTypeError(validar, canon(datos));
    }
    exigirTypeError(validar, canon({ ...crear(), campo_extra: "valor" }));
  }

  for (const campo of [
    "autenticacion_ref", "sesion_ref", "actor_ref", "identidad_ref",
    "perfil_ref", "organizacion_ref", "rol_ref", "permiso_ref",
    "decision_ref",
  ]) {
    exigirTypeError(
      validarSolicitudAsignacion,
      canon({ ...datosAsignacion(), [campo]: "referencia:inyectada:001" }),
    );
    exigirTypeError(
      validarSolicitudReasignacion,
      canon({ ...datosReasignacion(), [campo]: "referencia:inyectada:001" }),
    );
  }
  for (const campo of [
    "organizacion_ref", "unidad_ref", "responsable_ref", "notificacion_ref",
    "auditoria_ref", "evento_ref", "decision_ref", "hmac",
  ]) {
    exigirTypeError(
      validarReciboAsignacion,
      canon({ ...datosRecibo(), [campo]: "referencia:interna:001" }),
    );
  }
});

test("impone canon único: sin duplicados, espacios, BOM, reordenación ni escapes", () => {
  const asignacion = canon(datosAsignacion());
  const reasignacion = canon(datosReasignacion());
  const recibo = canon(datosRecibo());
  for (const [validar, texto] of [
    [validarSolicitudAsignacion, asignacion],
    [validarSolicitudReasignacion, reasignacion],
    [validarReciboAsignacion, recibo],
  ]) {
    exigirTypeError(validar, ` ${texto}`);
    exigirTypeError(validar, `${texto}\n`);
    exigirTypeError(validar, `\uFEFF${texto}`);
    exigirTypeError(
      validar,
      texto.replace("{", "{\"expediente_ref\":\"duplicada:001\","),
    );
  }

  exigirTypeError(
    validarSolicitudAsignacion,
    canon({
      unidad_ref: "unidad:gestora:001",
      expediente_ref: "expediente:asignacion:001",
      version_esperada: 1,
      clave_idempotencia: UUID,
      responsable_ref: "responsable:interno:001",
    }),
  );
  exigirTypeError(
    validarSolicitudAsignacion,
    asignacion.replace(
      "expediente:asignacion:001",
      "expediente:\\u0061signacion:001",
    ),
  );
  const conBarra = canon(datosAsignacion({
    expediente_ref: "expediente/asignacion/001",
  }));
  exigirTypeError(
    validarSolicitudAsignacion,
    conBarra.replace("expediente/asignacion/001", "expediente\\/asignacion\\/001"),
  );
  exigirTypeError(
    validarSolicitudAsignacion,
    asignacion.replace("\"version_esperada\":1", "\"version_esperada\":1.0"),
  );
});

test("rechaza JSON malformado, raíces no objeto, valores múltiples y anidados", () => {
  for (const validar of [
    validarSolicitudAsignacion,
    validarSolicitudReasignacion,
    validarReciboAsignacion,
  ]) {
    for (const entrada of [
      null, undefined, 1, true, Symbol("entrada"), 1n, {}, [],
      "", "{", "null", "[]", "true", "1", "\"texto\"", "{}{}",
    ]) {
      exigirTypeError(validar, entrada);
    }
  }

  for (const valor of [{ valor: "anidado" }, ["anidado"], null, true]) {
    exigirTypeError(
      validarSolicitudAsignacion,
      canon(datosAsignacion({ expediente_ref: valor })),
    );
    exigirTypeError(
      validarSolicitudReasignacion,
      canon(datosReasignacion({ observaciones: valor })),
    );
    exigirTypeError(
      validarReciboAsignacion,
      canon(datosRecibo({ recibo_ref: valor })),
    );
  }
});

test("rechaza claves especiales aunque el JSON sea sintácticamente válido", () => {
  const base = canon(datosAsignacion());
  for (const campo of ["__proto__", "constructor", "prototype"]) {
    const texto = `${base.slice(0, -1)},${JSON.stringify(campo)}:"valor"}`;
    exigirTypeError(validarSolicitudAsignacion, texto);
  }
});

test("aplica límites semánticos de referencias, versiones, UUID y motivo", () => {
  const referenciaMinima = "a.b";
  const referenciaMaxima = `a${"b".repeat(159)}`;
  for (const valor of [referenciaMinima, referenciaMaxima, "A0._:/#-z"]) {
    assert.equal(validarSolicitudAsignacion(canon(datosAsignacion({
      expediente_ref: valor,
      unidad_ref: valor,
      responsable_ref: valor,
    }))).expediente_ref, valor);
    assert.equal(validarReciboAsignacion(canon(datosRecibo({
      expediente_ref: valor,
      recibo_ref: valor,
    }))).recibo_ref, valor);
  }
  for (const valor of [
    "ab", `a${"b".repeat(160)}`, "_ab", "a b", "ábc", "a?b", "", 123,
  ]) {
    for (const campo of ["expediente_ref", "unidad_ref", "responsable_ref"]) {
      exigirTypeError(
        validarSolicitudAsignacion,
        canon(datosAsignacion({ [campo]: valor })),
      );
    }
    for (const campo of ["expediente_ref", "recibo_ref"]) {
      exigirTypeError(
        validarReciboAsignacion,
        canon(datosRecibo({ [campo]: valor })),
      );
    }
  }

  for (const version of [1, Number.MAX_SAFE_INTEGER - 1]) {
    assert.equal(validarSolicitudAsignacion(canon(datosAsignacion({
      version_esperada: version,
    }))).version_esperada, version);
  }
  for (const version of [
    0, -1, 1.5, NaN, Infinity, Number.MAX_SAFE_INTEGER,
  ]) {
    exigirTypeError(
      validarSolicitudAsignacion,
      canon(datosAsignacion({ version_esperada: version })),
    );
  }
  for (const version of [2, Number.MAX_SAFE_INTEGER]) {
    assert.equal(validarReciboAsignacion(canon(datosRecibo({
      version_resultante: version,
    }))).version_resultante, version);
  }
  for (const version of [
    1, 2.5, NaN, Infinity, Number.MAX_SAFE_INTEGER + 1,
  ]) {
    exigirTypeError(
      validarReciboAsignacion,
      canon(datosRecibo({ version_resultante: version })),
    );
  }

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
      canon(datosAsignacion({ clave_idempotencia: valor })),
    );
  }
  for (const valor of ["a1", `a${"b".repeat(79)}`, "cambio.unidad-1"]) {
    assert.equal(validarSolicitudReasignacion(canon(datosReasignacion({
      motivo_reasignacion_clave: valor,
    }))).motivo_reasignacion_clave, valor);
  }
  for (const valor of [
    "a", `a${"b".repeat(80)}`, "Ajuste", "a:b", "", 7,
  ]) {
    exigirTypeError(
      validarSolicitudReasignacion,
      canon(datosReasignacion({ motivo_reasignacion_clave: valor })),
    );
  }
});

test("valida observaciones NFC por puntos de código y sus bordes astrales", () => {
  const astrales1000 = "😀".repeat(1000);
  for (const valor of ["Texto", "Línea uno\nLínea dos\tfin"]) {
    assert.equal(validarSolicitudReasignacion(canon(datosReasignacion({
      observaciones: valor,
    }))).observaciones, valor);
  }
  const maxima = canon(datosReasignacion({
    expediente_ref: `a${"b".repeat(159)}`,
    version_esperada: Number.MAX_SAFE_INTEGER - 1,
    unidad_ref: `u${"n".repeat(159)}`,
    responsable_ref: `r${"p".repeat(159)}`,
    motivo_reasignacion_clave: `m${"c".repeat(79)}`,
    observaciones: astrales1000,
  }));
  assert.equal(bytesUTF8(maxima), 4764);
  assert.equal(validarSolicitudReasignacion(maxima).observaciones, astrales1000);

  for (const valor of [
    "", " texto", "texto ", "\ntexto", "texto\t", "Cafe\u0301",
    "texto\u0000interno", "texto\u0085interno", "texto\uD800interno",
    "😀".repeat(1001),
  ]) {
    exigirTypeError(
      validarSolicitudReasignacion,
      canon(datosReasignacion({ observaciones: valor })),
    );
  }
});

test("acredita máximos canónicos de asignación y recibo", () => {
  const referencia = `a${"b".repeat(159)}`;
  const asignacion = canon(datosAsignacion({
    expediente_ref: referencia,
    version_esperada: Number.MAX_SAFE_INTEGER - 1,
    unidad_ref: referencia,
    responsable_ref: referencia,
  }));
  const recibo = canon(datosRecibo({
    operacion: "reasignar",
    expediente_ref: referencia,
    version_resultante: Number.MAX_SAFE_INTEGER,
    recibo_ref: referencia,
    confirmada_en: "9999-12-31T23:59:59.999999Z",
  }));
  assert.equal(bytesUTF8(asignacion), 634);
  assert.equal(bytesUTF8(recibo), 524);
  assert.deepEqual(validarSolicitudAsignacion(asignacion), JSON.parse(asignacion));
  assert.deepEqual(validarReciboAsignacion(recibo), JSON.parse(recibo));
});

test("cierra esquema, operación e instante UTC canónico del recibo", () => {
  for (const operacion of ["asignar", "reasignar"]) {
    assert.equal(validarReciboAsignacion(canon(datosRecibo({
      operacion,
    }))).operacion, operacion);
  }
  for (const cambios of [
    { esquema: "vec.contratacion-temporal.recibo-asignacion.v2" },
    { operacion: "registrar" },
    { operacion: "ASIGNAR" },
  ]) {
    exigirTypeError(validarReciboAsignacion, canon(datosRecibo(cambios)));
  }
  for (const valor of [
    "0001-01-01T00:00:00.000001Z",
    "2000-02-29T23:59:59.000001Z",
    "9999-12-31T23:59:59.999999Z",
  ]) {
    assert.equal(validarReciboAsignacion(canon(datosRecibo({
      confirmada_en: valor,
    }))).confirmada_en, valor);
  }
  for (const valor of [
    "0001-01-01T00:00:00Z", "0000-01-01T00:00:00Z",
    "10000-01-01T00:00:00Z",
    "2026-02-29T00:00:00Z", "2026-04-31T00:00:00Z",
    "2026-01-01T24:00:00Z", "2026-01-01T00:60:00Z",
    "2026-01-01T00:00:60Z", "2026-01-01T00:00:00+00:00",
    "2026-01-01T00:00:00z", "2026-01-01T00:00:00.1234567Z",
    "2026-01-01T00:00:00.120Z", "2026-1-01T00:00:00Z",
  ]) {
    exigirTypeError(
      validarReciboAsignacion,
      canon(datosRecibo({ confirmada_en: valor })),
    );
  }
});

test("rechaza referencias y UUID inválidas sin filtrar intrínsecos alterados", () => {
  const ejecutarOriginal = RegExp.prototype.exec;
  const probarOriginal = RegExp.prototype.test;
  let aceptacionesAtacantes = 0;
  let erroresAtacantes = 0;
  const rechazarEntradasInvalidas = () => {
    exigirTypeError(
      validarSolicitudAsignacion,
      canon(datosAsignacion({ expediente_ref: "??" })),
    );
    exigirTypeError(
      validarSolicitudAsignacion,
      canon(datosAsignacion({ clave_idempotencia: "uuid-inválida" })),
    );
    exigirTypeError(
      validarReciboAsignacion,
      canon(datosRecibo({ recibo_ref: "??" })),
    );
  };
  try {
    const aceptarAtacante = () => {
      aceptacionesAtacantes += 1;
      return ["falso positivo atacante"];
    };
    RegExp.prototype.exec = aceptarAtacante;
    RegExp.prototype.test = aceptarAtacante;
    rechazarEntradasInvalidas();

    const fallarAtacante = () => {
      erroresAtacantes += 1;
      throw new Error("error atacante no debe escapar");
    };
    RegExp.prototype.exec = fallarAtacante;
    RegExp.prototype.test = fallarAtacante;
    rechazarEntradasInvalidas();
  } finally {
    RegExp.prototype.exec = ejecutarOriginal;
    RegExp.prototype.test = probarOriginal;
  }
  assert.equal(aceptacionesAtacantes, 0);
  assert.equal(erroresAtacantes, 0);
});

test("aplica 8192 bytes antes de analizar y conserva intrínsecos capturados", async () => {
  const ruta = new URL("./contrato-asignacion.js", import.meta.url);
  const analizarOriginal = JSON.parse;
  let analisis = 0;
  JSON.parse = (...argumentos) => {
    analisis += 1;
    return analizarOriginal(...argumentos);
  };
  try {
    const fresco = await import(`${ruta.href}?limite=${Date.now()}`);
    exigirTypeError(
      fresco.validarSolicitudAsignacion,
      `"${"😀".repeat(2048)}"`,
    );
    assert.equal(analisis, 0);
    fresco.validarSolicitudAsignacion(canon(datosAsignacion()));
    assert.equal(analisis, 1);
  } finally {
    JSON.parse = analizarOriginal;
  }

  const asignacion = canon(datosAsignacion());
  const analizarCapturado = JSON.parse;
  const serializarCapturado = JSON.stringify;
  JSON.parse = () => { throw new Error("parse global alterado"); };
  JSON.stringify = () => { throw new Error("stringify global alterado"); };
  try {
    assert.deepEqual(validarSolicitudAsignacion(asignacion), datosAsignacion());
  } finally {
    JSON.parse = analizarCapturado;
    JSON.stringify = serializarCapturado;
  }
});

test("los TypeError son opacos y no incorporan el dato rechazado", () => {
  const dato = "dato personal sintético no repetir";
  assert.throws(
    () => validarSolicitudAsignacion(canon(datosAsignacion({
      expediente_ref: dato,
    }))),
    (error) => error instanceof TypeError
      && error.message === "contrato JSON no válido"
      && !error.message.includes(dato),
  );
});

test("la importación no produce efectos y el productivo no abre superficies", async () => {
  const ruta = new URL("./contrato-asignacion.js", import.meta.url);
  const fuente = await readFile(ruta, "utf8");
  const globalesAntes = Reflect.ownKeys(globalThis);
  await import(`${ruta.href}?sin-efectos=${Date.now()}`);
  assert.deepEqual(Reflect.ownKeys(globalThis), globalesAntes);

  for (const patron of [
    /TextEncoder|\bBlob\b|structuredClone/u,
    /\bfetch\b|XMLHttpRequest|WebSocket|EventSource/u,
    /\bdocument\b|\bwindow\b|\bnavigator\b|\bDOM\b/u,
    /localStorage|sessionStorage|indexedDB|\bcookie\b/u,
    /node:|\brequire\s*\(|\bprocess\b|\bBuffer\b/u,
    /autenticacion|autorizacion|autoridad|identidad|sesion|actor|perfil|organizacion|rol|permiso|decision/iu,
    /Object\.(?:keys|values|entries|assign)|Reflect\.ownKeys|structuredClone/u,
  ]) {
    assert.doesNotMatch(fuente, patron);
  }
});
