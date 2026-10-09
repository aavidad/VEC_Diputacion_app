const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,511}$/u;
const LLAMAMIENTO = /^llamamiento:[0-9a-f]{64}$/u;
const RECIBO_EMISION = /^recibo:llamamiento:[0-9a-f]{64}$/u;
const RESPUESTAS = new Set(["acepta", "renuncia", "renuncia_justificada"]);
const MODOS = new Set(["firme", "propuesta_rrhh"]);

function objeto(valor, claves) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor)
    || Object.keys(valor).sort().join("|") !== [...claves].sort().join("|")) {
    throw new TypeError("resultado de Bolsa no válido");
  }
  return valor;
}

function referencia(valor, patron = REFERENCIA) {
  if (typeof valor !== "string" || !patron.test(valor)) throw new TypeError("referencia Bolsa no válida");
  return valor;
}

function fecha(valor) {
  if (typeof valor !== "string" || valor.length > 40 || !Number.isFinite(Date.parse(valor))) {
    throw new TypeError("fecha Bolsa no válida");
  }
  return valor;
}

function opcional(valor, validar) { return valor === null ? null : validar(valor); }

function participacion(valor) {
  objeto(valor, ["participacion_ref", "respuesta", "modo", "recibo_respuesta_ref", "respondida_en",
    "justificante_ref", "contacto_resultado", "recibo_contacto_ref", "contacto_en",
    "situacion_actual", "recibo_situacion_ref", "situacion_desde"]);
  referencia(valor.participacion_ref);
  if (valor.respuesta !== null && (!RESPUESTAS.has(valor.respuesta) || !MODOS.has(valor.modo)
    || valor.recibo_respuesta_ref === null || valor.respondida_en === null)) {
    throw new TypeError("respuesta Bolsa no válida");
  }
  if (valor.respuesta === null && [valor.modo, valor.recibo_respuesta_ref, valor.respondida_en,
    valor.justificante_ref].some((campo) => campo !== null)) throw new TypeError("respuesta Bolsa incompleta");
  opcional(valor.recibo_respuesta_ref, referencia);
  opcional(valor.respondida_en, fecha);
  opcional(valor.justificante_ref, referencia);
  if (valor.contacto_resultado === null && (valor.recibo_contacto_ref !== null || valor.contacto_en !== null)) {
    throw new TypeError("contacto Bolsa incompleto");
  }
  opcional(valor.contacto_resultado, (x) => referencia(x, /^[a-z][a-z0-9_]{1,63}$/u));
  opcional(valor.recibo_contacto_ref, referencia);
  opcional(valor.contacto_en, fecha);
  if (valor.situacion_actual === null && (valor.recibo_situacion_ref !== null || valor.situacion_desde !== null)) {
    throw new TypeError("situación Bolsa incompleta");
  }
  opcional(valor.situacion_actual, (x) => referencia(x, /^[a-z][a-z0-9_]{1,63}$/u));
  opcional(valor.recibo_situacion_ref, referencia);
  opcional(valor.situacion_desde, fecha);
  return Object.freeze({ ...valor });
}

function vinculo(valor) {
  objeto(valor, ["bolsa_ref", "llamamiento_ref", "recibo_emision_ref", "recibo_vinculo_ref",
    "vinculado_en", "emitido_en", "participaciones"]);
  referencia(valor.bolsa_ref);
  referencia(valor.llamamiento_ref, LLAMAMIENTO);
  referencia(valor.recibo_emision_ref, RECIBO_EMISION);
  referencia(valor.recibo_vinculo_ref);
  fecha(valor.vinculado_en);
  fecha(valor.emitido_en);
  if (!Array.isArray(valor.participaciones) || valor.participaciones.length < 1 || valor.participaciones.length > 100) {
    throw new TypeError("participaciones Bolsa no válidas");
  }
  const participaciones = valor.participaciones.map(participacion);
  if (new Set(participaciones.map((p) => p.participacion_ref)).size !== participaciones.length) {
    throw new TypeError("participaciones Bolsa repetidas");
  }
  return Object.freeze({ ...valor, participaciones: Object.freeze(participaciones) });
}

function emision(valor) {
  objeto(valor, ["bolsa_ref", "llamamiento_ref", "recibo_emision_ref", "referencia_visible", "emitido_en"]);
  referencia(valor.bolsa_ref);
  referencia(valor.llamamiento_ref, LLAMAMIENTO);
  referencia(valor.recibo_emision_ref, RECIBO_EMISION);
  if (typeof valor.referencia_visible !== "string" || valor.referencia_visible.length > 80) {
    throw new TypeError("referencia visible Bolsa no válida");
  }
  fecha(valor.emitido_en);
  return Object.freeze({ ...valor });
}

export function validarResultadoBolsaCT(valor) {
  objeto(valor, ["vinculos", "emisiones_vinculables", "siguiente_cursor"]);
  if (!Array.isArray(valor.vinculos) || valor.vinculos.length > 100
    || !Array.isArray(valor.emisiones_vinculables) || valor.emisiones_vinculables.length > 20
    || valor.siguiente_cursor !== null) throw new TypeError("resultado Bolsa no válido");
  const vinculos = valor.vinculos.map(vinculo);
  const emisiones = valor.emisiones_vinculables.map(emision);
  if (new Set(vinculos.map((v) => v.llamamiento_ref)).size !== vinculos.length
    || new Set(emisiones.map((e) => e.llamamiento_ref)).size !== emisiones.length) {
    throw new TypeError("llamamientos Bolsa repetidos");
  }
  return Object.freeze({ vinculos: Object.freeze(vinculos), emisiones_vinculables: Object.freeze(emisiones),
    siguiente_cursor: null });
}
