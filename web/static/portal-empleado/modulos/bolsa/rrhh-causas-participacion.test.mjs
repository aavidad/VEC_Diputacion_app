import assert from "node:assert/strict";
import test from "node:test";
import {
  calcularHuellaCausa, crearClienteCausasParticipacion, RUTA_CAUSAS_PARTICIPACION,
  RUTA_PROPUESTAS_CAUSAS, validarCatalogoCausasRecibido, validarCausaEditable,
} from "./rrhh-causas-participacion-api.js";
import { crearSuperficieRRHHCausasParticipacion } from "./rrhh-causas-participacion-ui.js";
import { MENSAJES_RRHH_CAUSAS_PARTICIPACION_ES as mensajes } from "./rrhh-causas-participacion-i18n.js";

const entrada = Object.freeze({ codigo: "cambio_contacto", version: 1, etiqueta: "Cambio de contacto",
  aplica_situacion: false, aplica_contacto: true, publicable: true, activa: true });
const sha = async (texto) => Array.from(new Uint8Array(await crypto.subtle.digest("SHA-256",
  new TextEncoder().encode(texto))), (b) => b.toString(16).padStart(2, "0")).join("");
const huella = await calcularHuellaCausa(entrada);
const reciboCausa = `recibo:causa:${await sha(`bolsa.causa_participacion.recibo.v1\n${huella}`)}`;
const propuestaRef = `propuesta:causa:${"b".repeat(64)}`;
const reciboPropuesta = `recibo:propuesta:causa:${await sha(`bolsa.causa_participacion.propuesta.recibo.v1\n${propuestaRef}`)}`;
const causa = Object.freeze({ ...entrada, huella_sha256: huella });
const propuesta = Object.freeze({ ...causa, propuesta_ref: propuestaRef });
const json = (valor, status = 200) => new Response(JSON.stringify(valor), {
  status, headers: { "content-type": "application/json" },
});
const traducir = (clave, variables = {}) => mensajes[clave].replace(/\{([a-z_]+)\}/gu,
  (_todo, nombre) => String(variables[nombre] ?? ""));
const tick = () => new Promise((resolve) => setImmediate(resolve));
const pulsar = (superficie, accion) => {
  const boton = { disabled: false, dataset: { rrhhCausasAccion: accion },
    closest(selector) { return selector === "[data-rrhh-causas]" ? {} : this; } };
  superficie.manejarClick({ target: { closest: () => boton } });
};

test("canon causal B57 coincide con vector UTF-8 de PostgreSQL", async () => {
  assert.equal(await calcularHuellaCausa({ codigo: "prueba_utf8", version: 1,
    etiqueta: "Cambio: áéñ;|:", aplica_situacion: true, aplica_contacto: false,
    publicable: true, activa: true }),
  "9fc9fb53485b81871467f6093badad8dc2dfdd871448aec773a96ba9570389f4");
});

test("GET catálogo usa misma sesión, no-store y DTO mínimo sin claves técnicas visibles", async () => {
  const llamadas = [];
  const cliente = crearClienteCausasParticipacion({ fetchImpl: async (ruta, opciones) => {
    llamadas.push([ruta, opciones]);
    return json({ data: [{ codigo: causa.codigo, version: causa.version,
      huella_sha256: causa.huella_sha256, etiqueta: causa.etiqueta,
      aplica_situacion: causa.aplica_situacion, aplica_contacto: causa.aplica_contacto }] });
  } });
  const resultado = await cliente.consultar();
  assert.equal(resultado.causas[0].etiqueta, causa.etiqueta);
  assert.equal(resultado.causas[0].publicable, undefined);
  assert.equal(llamadas[0][0], RUTA_CAUSAS_PARTICIPACION);
  assert.equal(llamadas[0][1].method, "GET");
  assert.equal(llamadas[0][1].credentials, "same-origin");
  assert.equal(llamadas[0][1].cache, "no-store");
  assert.equal(llamadas[0][1].body, undefined);
  assert.equal(llamadas[0][1].headers.Authorization, undefined);
});

test("propuesta, consulta exacta y publicación usan rutas/recibos distintos", async () => {
  const llamadas = [];
  const cliente = crearClienteCausasParticipacion({ fetchImpl: async (ruta, opciones) => {
    llamadas.push([ruta, opciones]);
    if (ruta === RUTA_PROPUESTAS_CAUSAS) return json({ data: { propuesta, recibo: reciboPropuesta } }, 201);
    if (opciones.method === "GET") return json({ data: { propuesta, estado: "pendiente", recibo: reciboPropuesta } });
    return json({ data: { catalogo: causa, recibo: reciboCausa } }, 201);
  } });
  const creada = await cliente.proponer(entrada);
  assert.equal(creada.propuesta.propuesta_ref, propuestaRef);
  const consultada = await cliente.consultarPropuesta(propuestaRef);
  assert.equal(consultada.estado, "pendiente");
  const publicada = await cliente.publicar(propuestaRef, consultada.propuesta);
  assert.equal(publicada.recibo, reciboCausa);
  assert.equal(llamadas[0][0], RUTA_PROPUESTAS_CAUSAS);
  assert.equal(llamadas[1][0], `${RUTA_PROPUESTAS_CAUSAS}/${propuestaRef}`);
  assert.equal(llamadas[2][0], `${RUTA_PROPUESTAS_CAUSAS}/${propuestaRef}/publicar`);
  assert.deepEqual(JSON.parse(llamadas[0][1].body), entrada);
  assert.deepEqual(JSON.parse(llamadas[2][1].body), entrada);
  assert.equal(llamadas[1][1].body, undefined);
  for (const [, opciones] of llamadas) {
    assert.equal(opciones.cache, "no-store");
    assert.equal(opciones.credentials, "same-origin");
  }
});

test("rechaza payload o recibo incompatible, 403 y límite de respuesta", async () => {
  assert.throws(() => validarCausaEditable({ ...entrada, etiqueta: " Texto " }));
  assert.throws(() => validarCausaEditable({ ...entrada, etiqueta: "Sin\ncontrol" }));
  assert.throws(() => validarCausaEditable({ ...entrada, aplica_contacto: false }));
  assert.throws(() => validarCatalogoCausasRecibido({ data: [causa, causa] }), /duplicado/u);
  const denegado = crearClienteCausasParticipacion({ fetchImpl: async () =>
    json({ error: { codigo: "acceso_denegado" } }, 403) });
  assert.deepEqual(await denegado.consultarPropuesta(propuestaRef), { ok: false, status: 403 });
  const falsa = crearClienteCausasParticipacion({ fetchImpl: async () =>
    json({ data: { propuesta: { ...propuesta, huella_sha256: "a".repeat(64) },
      recibo: reciboPropuesta } }, 201) });
  await assert.rejects(falsa.proponer(entrada), /huella/u);
  const largo = crearClienteCausasParticipacion({ fetchImpl: async () => new Response("x".repeat(262145), {
    status: 200, headers: { "content-type": "application/json" },
  }) });
  await assert.rejects(largo.consultar(), /grande/u);
});

test("vista muestra carga, lista sin claves redundantes, ayuda cerrada y 403", async () => {
  let resolver;
  const superficie = crearSuperficieRRHHCausasParticipacion({ cliente: {
    consultar: () => new Promise((res) => { resolver = res; }),
  }, traducir });
  superficie.activar();
  assert.match(superficie.renderizar(), /Cargando causas/u);
  resolver({ ok: true, causas: [causa] }); await tick();
  const html = superficie.renderizar();
  assert.match(html, /Cambio de contacto/u);
  assert.doesNotMatch(html.split("<table")[1].split("<\/table>")[0], /cambio_contacto|[a-f0-9]{64}/u);
  assert.match(html, /id="rrhh-causas-ayuda"[^>]*hidden/u);
  assert.match(html, /data-rrhh-causas-form="propuesta"/u);
  assert.doesNotMatch(html, /data-rrhh-causas-form="publicacion"/u);
  pulsar(superficie, "modo-publicar");
  assert.match(superficie.renderizar(), /data-rrhh-causas-form="publicacion"/u);
  assert.doesNotMatch(superficie.renderizar(), /data-rrhh-causas-form="propuesta"/u);
  superficie.desmontar();
  const denegada = crearSuperficieRRHHCausasParticipacion({ cliente: {
    consultar: async () => ({ ok: false, status: 403 }),
  }, traducir });
  denegada.activar(); await tick();
  assert.match(denegada.renderizar(), /Su sesión no permite consultar/u);
  assert.doesNotMatch(denegada.renderizar(), /data-rrhh-causas-form/u);
});

test("503 de propuesta retiene los siete campos y reintenta mismo cuerpo", async () => {
  const cuerpos = [];
  let llamadas = 0;
  const superficie = crearSuperficieRRHHCausasParticipacion({ cliente: {
    consultar: async () => ({ ok: true, causas: [] }),
    proponer: async (cuerpo) => {
      cuerpos.push(structuredClone(cuerpo)); llamadas++;
      return llamadas === 1 ? { ok: false, status: 503 }
        : { ok: true, propuesta, recibo: reciboPropuesta };
    },
  }, traducir });
  superficie.activar(); await tick();
  const formulario = { dataset: { rrhhCausasForm: "propuesta" } };
  for (const [name, value] of Object.entries(entrada)) superficie.manejarCambio({ target: {
    name, value, checked: value, type: typeof value === "boolean" ? "checkbox" : "text",
    closest: () => formulario,
  } });
  const evento = { target: { closest: () => formulario, reportValidity: () => true }, preventDefault() {} };
  superficie.manejarSubmit(evento); await tick();
  assert.deepEqual(superficie.estado().pendiente.entrada, entrada);
  assert.match(superficie.renderizar(), /Reintentar el mismo envío/u);
  superficie.manejarSubmit(evento); await tick();
  assert.deepEqual(cuerpos, [entrada, entrada]);
  assert.equal(superficie.estado().pendiente, null);
  assert.ok(superficie.renderizar().includes(propuestaRef));
  assert.doesNotMatch(superficie.renderizar(), /data-rrhh-causas-form="publicacion"[\s\S]*Cambio de contacto/u);
});

test("segunda identidad consulta propuesta y reintenta publicación con cuerpo exacto", async () => {
  const cuerpos = [];
  let publicaciones = 0;
  const superficie = crearSuperficieRRHHCausasParticipacion({ cliente: {
    consultar: async () => ({ ok: true, causas: [] }),
    consultarPropuesta: async (ref) => ({ ok: true, propuesta: { ...propuesta },
      estado: "pendiente", recibo: reciboPropuesta, ref }),
    publicar: async (ref, cuerpo) => {
      cuerpos.push([ref, structuredClone(cuerpo)]); publicaciones++;
      return publicaciones === 1 ? { ok: false, status: 503 }
        : { ok: true, catalogo: causa, recibo: reciboCausa };
    },
  }, traducir });
  superficie.activar(); await tick();
  pulsar(superficie, "modo-publicar");
  superficie.manejarCambio({ target: { value: propuestaRef,
    matches: (selector) => selector === "[data-rrhh-causas-ref]" } });
  await superficie.consultarPropuesta();
  assert.match(superficie.renderizar(), /Pendiente de publicación/u);
  assert.match(superficie.renderizar(), /Cambio de contacto/u);
  const formulario = { dataset: { rrhhCausasForm: "publicacion" } };
  const evento = { target: { closest: () => formulario, reportValidity: () => true }, preventDefault() {} };
  superficie.manejarSubmit(evento); await tick();
  assert.equal(superficie.estado().pendiente.propuestaRef, propuestaRef);
  superficie.manejarSubmit(evento); await tick();
  assert.deepEqual(cuerpos, [[propuestaRef, entrada], [propuestaRef, entrada]]);
  assert.equal(superficie.estado().pendiente, null);
  assert.ok(superficie.renderizar().includes(reciboCausa));
});

test("propuesta publicada o superada no habilita otro POST; conflicto admite salida explícita", async () => {
  let publicaciones = 0;
  const superficie = crearSuperficieRRHHCausasParticipacion({ cliente: {
    consultar: async () => ({ ok: true, causas: [] }),
    consultarPropuesta: async () => ({ ok: true, propuesta, estado: "superada", recibo: reciboPropuesta }),
    publicar: async () => { publicaciones++; return { ok: false, status: publicaciones === 1 ? 503 : 409 }; },
    proponer: async () => ({ ok: false, status: 503 }),
  }, traducir, confirmarDescarte: () => true });
  superficie.activar(); await tick();
  pulsar(superficie, "modo-publicar");
  superficie.manejarCambio({ target: { value: propuestaRef,
    matches: (selector) => selector === "[data-rrhh-causas-ref]" } });
  await superficie.consultarPropuesta();
  assert.match(superficie.renderizar(), /Superada por otra propuesta/u);
  assert.match(superficie.renderizar(), /data-rrhh-causas-form="publicacion"[\s\S]*button type="submit" class="boton-primario" disabled/u);
  pulsar(superficie, "modo-proponer");
  const formulario = { dataset: { rrhhCausasForm: "propuesta" } };
  const evento = { target: { closest: () => formulario, reportValidity: () => true }, preventDefault() {} };
  for (const [name, value] of Object.entries(entrada)) superficie.manejarCambio({ target: {
    name, value, checked: value, type: typeof value === "boolean" ? "checkbox" : "text",
    closest: () => formulario,
  } });
  superficie.manejarSubmit(evento); await tick();
  assert.ok(superficie.estado().pendiente);
  pulsar(superficie, "descartar");
  assert.equal(superficie.estado().pendiente, null);
  assert.match(superficie.renderizar(), /resultado anterior no se ha acreditado/u);
});

test("403 y desmontaje retiran propuesta, recibo y envío pendiente de la sesión", async () => {
  const superficie = crearSuperficieRRHHCausasParticipacion({ cliente: {
    consultar: async () => ({ ok: true, causas: [] }),
    consultarPropuesta: async () => ({ ok: true, propuesta, estado: "pendiente", recibo: reciboPropuesta }),
    publicar: async () => ({ ok: false, status: 403 }),
  }, traducir });
  superficie.activar(); await tick(); pulsar(superficie, "modo-publicar");
  superficie.manejarCambio({ target: { value: propuestaRef,
    matches: (selector) => selector === "[data-rrhh-causas-ref]" } });
  await superficie.consultarPropuesta();
  assert.ok(superficie.renderizar().includes(reciboPropuesta));
  const formulario = { dataset: { rrhhCausasForm: "publicacion" } };
  superficie.manejarSubmit({ target: { closest: () => formulario, reportValidity: () => true }, preventDefault() {} });
  await tick();
  assert.equal(superficie.estado().propuestaConsultada, null);
  assert.equal(superficie.estado().pendiente, null);
  assert.doesNotMatch(superficie.renderizar(), /recibo:propuesta:causa:/u);
  superficie.desmontar();
  assert.equal(superficie.estado().propuestaRef, "");
  assert.equal(superficie.estado().propuestaConfirmada, null);
  assert.equal(superficie.estado().confirmacion, null);
});
