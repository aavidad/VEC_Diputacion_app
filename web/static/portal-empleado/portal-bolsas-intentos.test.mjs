import test from "node:test";
import assert from "node:assert/strict";
import {
  consultarIntentosContacto,
  crearControladorIntentosContacto,
  registrarContactoIntento,
  renderizarIntentosContacto,
  rutaContactosCandidato,
} from "./portal-bolsas-intentos.js";
import { MENSAJES_INTENTOS_ES, crearTraductorIntentos } from "./portal-i18n-intentos.js";

const candidato = { participacion_ref: "participacion:1", estado_clave: "disponible", ultimo_llamamiento: { llamamiento_ref: "llamamiento:1" } };
const intentos = {
  configurado: true, llamamiento_ref: "llamamiento:1", sin_contacto: 1, maximo: 4, proceso: 1, intento: 2,
  intentos_por_proceso: 2, procesos: 2, contactado: false, baja_propuesta: false, completo: true,
  ultimo_intento: "2026-09-28T08:00:00Z", siguiente_permitido_desde: "2026-09-28T10:00:00Z", avisos: ["antes_de_separacion"],
  franja: { valor: "09:00-14:00", solo_dias_habiles: true, control: "advertir" },
  reglas: [
    { clave: "b02.intentos_contacto", etiqueta: "Dos intentos", referencia: "vec.bolsa.reglas:1:b02.intentos_contacto", ejemplo: false },
    { clave: "b04.franja_llamadas", etiqueta: "Franja <b>", referencia: "vec.bolsa.reglas:1:b04.franja_llamadas", ejemplo: true },
  ],
};
const respuesta = (status, cuerpo) => ({ ok: status >= 200 && status < 300, status, json: async () => cuerpo });

test("el catálogo i18n de intentos está completo y rechaza claves ajenas", () => {
  const t = crearTraductorIntentos();
  assert.equal(t("registrado", { recibo: "r:1" }), "Contacto registrado. Recibo r:1.");
  assert.throws(() => t("inexistente"));
  assert.throws(() => crearTraductorIntentos({ ...MENSAJES_INTENTOS_ES, titulo: "" }));
});

test("la ficha muestra proceso, avisos y reglas con su procedencia, escapando textos", () => {
  const salida = renderizarIntentosContacto({ candidato, estado: { carga: "listo", datos: intentos } });
  assert.match(salida, /Proceso 1 de 2 · intento 2 de 2/);
  assert.match(salida, /Antes de la separación mínima/);
  assert.match(salida, /1 de 4/);
  assert.match(salida, /Regla de ejemplo<\/span> Franja &lt;b&gt;/);
  assert.match(salida, /data-intentos-form="intento"/);
  assert.match(salida, /data-intentos-form="rebote"/);
  assert.doesNotMatch(salida, /data-b8-accion/);
});

test("con los procesos agotados propone la baja con la exclusión existente y no ofrece otro intento", () => {
  const agotado = { ...intentos, sin_contacto: 4, proceso: 0, intento: 0, baja_propuesta: true, avisos: [] };
  const salida = renderizarIntentosContacto({ candidato, estado: { carga: "listo", datos: agotado } });
  assert.match(salida, /Baja propuesta/);
  assert.match(salida, /data-b8-accion="seleccionar" data-operacion="excluir"/);
  assert.doesNotMatch(salida, /data-intentos-form="intento"/);
  assert.match(salida, /data-intentos-form="rebote"/);
  const excluido = renderizarIntentosContacto({ candidato: { ...candidato, estado_clave: "excluido" }, estado: { carga: "listo", datos: agotado } });
  assert.doesNotMatch(excluido, /data-b8-accion/);
});

test("sin llamamiento o sin catálogo lo dice sin inventar reglas", () => {
  assert.match(renderizarIntentosContacto({ candidato: { participacion_ref: "p" } }), /Sin llamamiento en curso/);
  const sin = renderizarIntentosContacto({ candidato, estado: { carga: "listo", datos: { configurado: false } } });
  assert.match(sin, /Sin reglas de intentos en el catálogo/);
  assert.doesNotMatch(sin, /Reglas aplicadas/);
});

test("consulta con llamamiento_ref y valida el contrato", async () => {
  let url = "";
  const ok = await consultarIntentosContacto("bolsa:1", "participacion:1", "llamamiento:1", {
    fetchImpl: async (u) => { url = u; return respuesta(200, { data: { esquema: "vec.bolsa.rrhh.contactos.v1", contactos: [], intentos } }); },
  });
  assert.equal(url, `${rutaContactosCandidato("bolsa:1", "participacion:1")}?llamamiento_ref=llamamiento%3A1`);
  assert.equal(ok.ok, true);
  const roto = await consultarIntentosContacto("b", "p", "l", { fetchImpl: async () => respuesta(200, { data: { esquema: "vec.bolsa.rrhh.contactos.v1", intentos: { configurado: true } } }) });
  assert.equal(roto.ok, false);
});

test("traduce los rechazos de las reglas al registrar", async () => {
  const res = await registrarContactoIntento("b", "p", {}, "k", { fetchImpl: async () => respuesta(409, { error: { codigo: "intento_antes_de_separacion" } }) });
  assert.equal(res.ok, false);
  assert.equal(res.mensaje, "No ha pasado la separación mínima desde el último intento.");
});

test("el controlador registra un rebote de correo ligado al llamamiento y recarga el estado", async () => {
  const anterior = globalThis.FormData;
  globalThis.FormData = class { constructor(f) { this.v = f.valores; } get(c) { return this.v[c]; } };
  try {
    const peticiones = [];
    const fetchImpl = async (url, opciones) => {
      peticiones.push({ url, opciones });
      if (opciones.method === "POST") return respuesta(201, { data: { recibo_ref: "recibo:contacto:1" } });
      return respuesta(200, { data: { esquema: "vec.bolsa.rrhh.contactos.v1", contactos: [], intentos } });
    };
    const modal = { candidato };
    const estado = { bolsaSeleccionada: "bolsa:1", modalFicha: modal };
    const controlador = crearControladorIntentosContacto({ estado, renderizar() {}, fetchImpl });
    const formulario = { dataset: { intentosForm: "rebote" }, valores: { instante: "2026-09-28T10:00", anotacion: "Rebote del aviso" }, closest() { return this; } };
    assert.equal(controlador.manejarSubmit({ target: formulario, preventDefault() {} }), true);
    await new Promise((r) => setTimeout(r, 0));
    await new Promise((r) => setTimeout(r, 0));
    const post = peticiones.find((p) => p.opciones.method === "POST");
    const cuerpo = JSON.parse(post.opciones.body);
    assert.equal(cuerpo.canal, "correo");
    assert.equal(cuerpo.resultado, "no_entregado");
    assert.equal(cuerpo.llamamiento_ref, "llamamiento:1");
    assert.ok(post.opciones.headers["Idempotency-Key"]);
    assert.equal(modal.intentosContacto.recibo, "recibo:contacto:1");
    assert.equal(modal.intentosContacto.carga, "listo");
  } finally {
    globalThis.FormData = anterior;
  }
});
