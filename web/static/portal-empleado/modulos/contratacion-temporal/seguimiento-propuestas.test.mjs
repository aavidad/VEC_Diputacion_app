import assert from "node:assert/strict";
import test from "node:test";
import { validarConsultaSeguimientoCese } from "./cliente-http-seguimiento-cese.js";
import { montarPanelSeguimientoCese } from "./seguimiento-cese.js";
import { ofrecerNoIncorporacion } from "./seguimiento-no-incorporacion.js";
import { filasPropuestas, noIncorporacionVigente, validarPropuestasSeguimiento } from "./seguimiento-propuestas.js";

const EXP = "expediente:ps:001";
const reglaNo = { motivos: [{ clave: "no_presentado", etiqueta: "No se presenta el día y en el lugar indicados" }], segunda_persona: true };
const opciones = { causas_cese: [{ clave: "fin_sustitucion", etiqueta: "Fin de la sustitución", clave_i18n: "x.y", justificante_tipo: "comunicacion_reincorporacion" }],
  condiciones_cierre: ["cese_registrado", "ginpix_confirmado"], fase_retorno_modificacion: "fiscalizacion",
  motivos_modificacion: [{ clave: "cambio_jornada", etiqueta: "Cambio de jornada", clave_i18n: "x.z" }], confirmacion_ginpix: true, no_incorporacion: reglaNo };
const noIncA = { motivo_clave: "no_presentado", resolucion_ref: "resolucion:rrhh:2026/0142", resolucion_sha256: "b".repeat(64), resuelta_por: "per_segunda",
  fecha_notificacion: "2026-09-20", recibo_ref: "recibo:ni:a", registrada_en: "2026-09-21T09:00:00.123456Z" };
const propuestaA = { orden: 1, version_resultante: 7, confirmada_en: "2026-09-06T01:28:30.697897Z", recibo_ref: "recibo:propuesta:a", vigente: false,
  sustitucion: { no_incorporacion_recibo_ref: "recibo:ni:a", motivo_clave: "no_presentado", registrada_en: "2026-09-21T09:00:00.123456Z" } };
const propuestaB = { orden: 2, version_resultante: 9, confirmada_en: "2026-09-24T10:15:00.5Z", recibo_ref: "recibo:propuesta:b", vigente: true, sustitucion: null };
const consulta = (estado = {}) => ({ esquema: "vec.contratacion-temporal.seguimiento-cese.v1", opciones,
  estado: { expediente_ref: EXP, incorporacion: null, cese: null, cierre: null, ginpix: null, confirmacion_centro: null, ...estado } });
const contexto = Object.freeze({ expediente_ref: EXP, version: 9, fase_clave: "nombramiento", estado_clave: "en_curso" });
const esperar = () => new Promise((r) => setImmediate(r));
function contenedorFalso() {
  const eventos = new Map();
  return { innerHTML: "", eventos, addEventListener: (n, f) => eventos.set(n, f), removeEventListener: (n) => eventos.delete(n), querySelector: () => null };
}

test("la historia de propuestas se valida: una sola vigente, la última, y las anteriores con su no incorporación", () => {
  assert.equal(validarPropuestasSeguimiento([propuestaA, propuestaB]).length, 2);
  assert.equal(validarPropuestasSeguimiento([]).length, 0);
  assert.equal(validarPropuestasSeguimiento([{ ...propuestaB, orden: 1 }]).length, 1);
  for (const lista of [
    [{ ...propuestaA, vigente: true, sustitucion: null }, propuestaB],
    [propuestaA, { ...propuestaB, vigente: false }],
    [propuestaA, { ...propuestaB, orden: 3 }],
    [propuestaA, { ...propuestaB, version_resultante: 7 }],
    [propuestaA, { ...propuestaB, recibo_ref: "" }],
    [{ ...propuestaA, sustitucion: { ...propuestaA.sustitucion, motivo_clave: "No" } }, propuestaB],
    [propuestaA, { ...propuestaB, extra: 1 }],
    [propuestaA, { ...propuestaB, confirmada_en: "ayer" }],
  ]) assert.throws(() => validarPropuestasSeguimiento(lista), TypeError, JSON.stringify(lista));
  assert.throws(() => validarPropuestasSeguimiento(Array.from({ length: 51 }, () => propuestaB)), TypeError);
  assert.equal(validarConsultaSeguimientoCese(consulta({ propuestas: [propuestaA, propuestaB], no_incorporacion: noIncA }), EXP).estado.propuestas.length, 2);
  assert.throws(() => validarConsultaSeguimientoCese(consulta({ propuestas: [propuestaB] }), EXP), TypeError, "orden inválido");
});

test("tras la propuesta de la siguiente persona, la no incorporación anterior es historia y se vuelve a ofrecer", () => {
  const conB = { incorporacion: null, cese: null, no_incorporacion: noIncA, propuestas: [propuestaA, propuestaB] };
  assert.equal(noIncorporacionVigente(conB), null);
  assert.equal(ofrecerNoIncorporacion(conB, opciones), true);
  const sinB = { incorporacion: null, cese: null, no_incorporacion: noIncA, propuestas: [{ ...propuestaA, orden: 1, vigente: true, sustitucion: null }] };
  assert.equal(noIncorporacionVigente(sinB), noIncA);
  assert.equal(ofrecerNoIncorporacion(sinB, opciones), false);
  const t = (clave, v = {}) => `${clave}:${JSON.stringify(v)}`;
  const filas = filasPropuestas(conB, opciones, t, (f) => f);
  assert.deepEqual(filas.map(([a]) => a), ['propuesta_anterior:{"orden":1}', "propuesta_vigente:{}"]);
  assert.match(filas[0][1], /"motivo":"No se presenta el día y en el lugar indicados"/u);
  assert.match(filas[0][1], /"desde":"2026-09-06"/u, "día civil en Madrid");
  assert.match(filas[1][1], /"fecha":"2026-09-24"/u);
  assert.deepEqual(filasPropuestas({}, opciones, t, (f) => f), []);
});

test("el panel muestra la propuesta vigente y la historia, sin referencias internas", async () => {
  const c = contenedorFalso();
  const quitar = montarPanelSeguimientoCese({ contenedor: c, contexto, cliente: { consultarSeguimientoCese: async () =>
    validarConsultaSeguimientoCese(consulta({ no_incorporacion: noIncA, propuestas: [propuestaA, propuestaB] }), EXP) } });
  await esperar();
  assert.match(c.innerHTML, /Propuesta de nombramiento vigente/u);
  assert.match(c.innerHTML, /Registrada el 24 de septiembre de 2026\./u);
  assert.match(c.innerHTML, /Propuesta anterior \(1\)/u);
  assert.match(c.innerHTML, /sustituida por no incorporación \(No se presenta el día y en el lugar indicados\) el 21 de septiembre de 2026\./u);
  assert.doesNotMatch(c.innerHTML, /No se incorpora:/u, "la no incorporación de la persona anterior no es la vigente");
  assert.match(c.innerHTML, /data-ct-seg-form="no_incorporacion"/u, "la nueva persona puede no incorporarse");
  assert.doesNotMatch(c.innerHTML, /recibo:|resolucion:rrhh/u, "sin referencias internas en pantalla");
  quitar();
});
