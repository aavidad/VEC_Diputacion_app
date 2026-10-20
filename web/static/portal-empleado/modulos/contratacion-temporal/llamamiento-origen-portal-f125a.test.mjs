import assert from "node:assert/strict";
import test from "node:test";
import { raizPrueba, montar, EXPEDIENTE } from "./formulario-llamamiento-pruebas.js?v=VEC_WEB_VERSION";
import { validarResultadoBolsaCT } from "./contrato-resultado-bolsa.js?v=VEC_WEB_VERSION";
import { contextoLlamamientoDesdeEstado } from "./vista-expedientes-render.js?v=VEC_WEB_VERSION";

const llamamiento = `llamamiento:${"a".repeat(64)}`;
const emision = "2026-10-20T09:00:00Z";
const respuesta = "2026-10-20T09:05:00Z";
const persona = (origen_ct_confirmado = true) => ({ participacion_ref: "participacion:prueba:001",
  respuesta: "acepta", modo: "firme", recibo_respuesta_ref: "recibo:respuesta:prueba:001",
  respondida_en: respuesta, justificante_ref: null, contacto_resultado: null,
  recibo_contacto_ref: null, contacto_en: null, situacion_actual: "disponible",
  recibo_situacion_ref: null, situacion_desde: null, origen_ct_confirmado });
const resultado = (participaciones = [persona()]) => ({ vinculos: [{ bolsa_ref: "bolsa:prueba:001",
  llamamiento_ref: llamamiento, recibo_emision_ref: `recibo:${llamamiento}`,
  recibo_vinculo_ref: "recibo:vinculo:prueba:001", vinculado_en: emision,
  emitido_en: emision, participaciones }], emisiones_vinculables: [], siguiente_cursor: null,
  total_vinculos: 1, personas_solicitadas: 1, aceptaciones_firmes: 1 });
const contexto = (bolsa) => ({ expediente_ref: EXPEDIENTE, version_esperada: 6,
  numero_personas: 1, resultado_bolsa: bolsa });
const esperar = () => new Promise(setImmediate);

test("la ficha muestra origen O6, fechas y justificante de Mi Bolsa sin resolver ni pedir correo", () => {
  const raiz = raizPrueba(); let escrituras = 0;
  const cerrar = montar(raiz, { resolverLlamamiento: async () => { escrituras += 1; } },
    { contexto: contexto(resultado()) });
  const panel = raiz.innerHTML.match(/<section class="ct-alta ct-llamamiento" data-ct-portal-origen[\s\S]*$/u)?.[0];
  assert.ok(panel);
  assert.match(panel, /Aceptación en Mi Bolsa/u);
  assert.match(panel, /Iniciado desde este expediente/u);
  assert.match(panel, /datetime="2026-10-20T09:00:00Z"/u);
  assert.match(panel, /datetime="2026-10-20T09:05:00Z"/u);
  assert.match(panel, /recibo:respuesta:prueba:001/u);
  assert.doesNotMatch(panel, /<form|data-ct-portal-resolucion|\.eml|correo_ref|resolucion_ct|recibo:ct:/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="seleccion"/u);
  assert.equal(escrituras, 0);
  cerrar();
});

test("aceptación B7 o múltiple no se presenta como origen O6", () => {
  const raiz = raizPrueba();
  const cerrar = montar(raiz, {}, { contexto: contexto(resultado([persona(false)])) });
  assert.doesNotMatch(raiz.innerHTML, /data-ct-portal-origen/u);
  cerrar();
  assert.throws(() => validarResultadoBolsaCT(resultado([persona(), {
    ...persona(false), participacion_ref: "participacion:prueba:002" }])));
  assert.throws(() => validarResultadoBolsaCT(resultado([{
    ...persona(), respondida_en: "2026-10-20T08:59:59Z" }])));
});

test("la versión de la ficha conserva el origen de Bolsa en el contexto del formulario", () => {
  const bolsa = validarResultadoBolsaCT(resultado());
  const estado = { vista: "expediente", carga: "listo", ocupado: false,
    actualizacion_pendiente: false, resultado_indeterminado: false,
    expediente_ref: EXPEDIENTE,
    expediente: { demostracion: false, expediente_ref: EXPEDIENTE, version: 6,
      numero_personas: 1, resultado_bolsa: bolsa },
    cuadro: { demostracion: false, expedientes: [{ expediente_ref: EXPEDIENTE,
      version: 6, fase_clave: "llamamiento" }] } };
  assert.equal(contextoLlamamientoDesdeEstado(estado).resultado_bolsa, bolsa);
  estado.cuadro.expedientes[0].version = 7;
  assert.equal(contextoLlamamientoDesdeEstado(estado), null);
});

for (const escenario of [
  { nombre: "pendiente", codigo: null, texto: /Cargando los llamamientos/u },
  { nombre: "error", codigo: 503, texto: /No se han podido cargar los llamamientos/u },
  { nombre: "denegado", codigo: 403, texto: /No tiene permiso para ver estos llamamientos/u },
]) test(`origen O6 conserva la consulta ${escenario.nombre} sin formulario inoperativo`, async () => {
  const raiz = raizPrueba(); let escrituras = 0, liberar;
  const pendiente = new Promise((resolver) => { liberar = resolver; });
  const cerrar = montar(raiz, {
    consultarComunicacionesExpediente: async () => {
      if (escenario.codigo === null) return pendiente;
      throw Object.assign(new Error("consulta"), { envelopeValido: true, estado: escenario.codigo });
    },
    seleccionarLlamamiento: async () => { escrituras += 1; },
  }, { contexto: contexto(resultado()) });
  await esperar();
  assert.match(raiz.innerHTML, escenario.texto);
  assert.match(raiz.innerHTML, /data-ct-portal-origen/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form=/u);
  if (escenario.codigo === 503) assert.match(raiz.innerHTML, /data-ct-comunicaciones-reintentar/u);
  await raiz.enviar("seleccion", {});
  assert.equal(escrituras, 0);
  cerrar();
  liberar?.({ expediente_ref: EXPEDIENTE, comunicaciones: [] });
});

test("una comunicación CT seleccionada conserva el formulario legado", async () => {
  const raiz = raizPrueba();
  const fila = { organizacion_ref: "organizacion:prueba:001", expediente_ref: EXPEDIENTE,
    llamamiento_ref: "llamamiento:legado:001", comunicacion_ref: "comunicacion:legado:001",
    version: 2, estado: "registrada_localmente", registrada_en: emision,
    recibo_comunicacion_ref: "recibo:comunicacion:001", antecedente_tipo: "seleccion_confirmada",
    recibo_antecedente_ref: "recibo:seleccion:001", estado_respuesta: "sin_respuesta" };
  const cerrar = montar(raiz, {
    consultarComunicacionesExpediente: async () => ({ expediente_ref: EXPEDIENTE, comunicaciones: [fila] }),
    consultarReciboRespuesta: async () => { throw Object.assign(new Error("sin respuesta"), {
      envelopeValido: true, estado: 404, codigo: "recurso_no_encontrado" }); },
  }, { contexto: contexto(resultado()) });
  await esperar();
  assert.match(raiz.innerHTML, /data-ct-comunicacion-indice="0"/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form=/u);
  raiz.eventos.get("click")({ preventDefault() {}, target: { closest: (selector) =>
    selector === "[data-ct-comunicacion-indice]" ? { dataset: { ctComunicacionIndice: "0" } } : null } });
  await esperar();
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="respuesta"/u);
  assert.match(raiz.innerHTML, /data-ct-portal-origen/u);
  cerrar();
});
