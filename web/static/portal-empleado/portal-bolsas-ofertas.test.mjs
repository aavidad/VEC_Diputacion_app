import test from "node:test";
import assert from "node:assert/strict";
import { accionesPlaza, crearClienteOfertas, crearSuperficieOfertasBolsa, crearTraductorOfertas, mensajeError, validarOfertasBolsa,
  ESQUEMA_OFERTAS_BOLSA, RUTA_OFERTAS_BOLSA, RUTA_RESOLUCIONES_OFERTA } from "./portal-bolsas-ofertas.js?v=20261001-codexf-ct-catalogos-v1";

function plaza(numero, extra = {}) {
  return { numero_de_plaza: numero, estado: "vacante", secuencia: 0, participacion_ref: null, orden_vigente: null,
    responder_antes_de: null, puede_sin_respuesta: false, propuesta: null, historial: [], ...extra };
}

function oferta(extra = {}) {
  return {
    oferta_ref: "oferta:1", recibo_ref: "recibo:oferta:1", bolsa_ref: "bolsa:1",
    datos: { categoria: "Auxiliar <b>", centro: "Residencia", fecha_inicio: "2026-10-01", descripcion: "Sustitución" },
    plazo: { regla_ref: "vec.bolsa.reglas:1:b10.plazo_publicacion", ejemplo: false, cantidad: 2 },
    publicada_en: "2026-09-25T10:00:00Z", vence_antes_de: "2026-09-29T22:00:00Z", estado: "abierta",
    disposiciones: [], disposiciones_total: 0, propuesta: null, resolucion: null,
    numero_plazas: 1, politica_plazas: null, plazas: [plaza(1)], ...extra,
  };
}

const politica = { llamada: "simultanea", respuesta_horas: 24, tras_renuncia: "siguiente_en_orden" };
const turno = () => new Promise((r) => setTimeout(r, 0));
const sobre = (ofertas) => ({ ok: true, datos: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas } });

function respuesta(status, cuerpo) {
  return { ok: status >= 200 && status < 300, status, json: async () => cuerpo };
}

function boton(dataset) {
  const b = { disabled: false, dataset };
  b.closest = () => b;
  return { target: b };
}

test("el contrato exige una entrada por plaza y rechaza estados, propuestas y actos ajenos", () => {
  assert.equal(validarOfertasBolsa({ data: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [oferta()] } }).ofertas.length, 1);
  const invalidas = [
    oferta({ estado: "inventado" }),
    oferta({ numero_plazas: 2 }),
    oferta({ numero_plazas: 0, plazas: [] }),
    oferta({ plazas: [plaza(2)] }),
    oferta({ plazas: [plaza(1, { estado: "ocupada" })] }),
    oferta({ plazas: [plaza(1, { propuesta: { tipo: "adjudicar" } })] }),
    oferta({ plazas: [plaza(1, { historial: [{ secuencia: 1, tipo: "borrada", orden_vigente: 1 }] })] }),
    oferta({ plazas: [plaza(1, { responder_antes_de: "mañana" })] }),
  ];
  for (const o of invalidas) assert.throws(() => validarOfertasBolsa({ data: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [o] } }));
});

test("el cliente envía bolsa, datos, número de plazas y el acto de la plaza, sin identidad", async () => {
  const llamadas = [];
  const cliente = crearClienteOfertas({ fetchImpl: async (ruta, opciones) => { llamadas.push({ ruta, opciones }); return respuesta(201, { data: oferta() }); } });
  const r = await cliente.publicar("bolsa:1", { categoria: "Aux", centro: "Res", fecha_inicio: "2026-10-01", descripcion: "Des" }, 3, "oferta-clave-1");
  assert.equal(r.ok, true);
  assert.equal(llamadas[0].ruta, RUTA_OFERTAS_BOLSA);
  assert.equal(llamadas[0].opciones.headers["Idempotency-Key"], "oferta-clave-1");
  assert.deepEqual(JSON.parse(llamadas[0].opciones.body).numero_plazas, 3);
  assert.deepEqual(Object.keys(JSON.parse(llamadas[0].opciones.body)), ["bolsa_ref", "datos", "numero_plazas"]);
  await cliente.registrarActo("bolsa:1", { oferta_ref: "oferta:1", numero_de_plaza: 2, tipo: "llamamiento_directo", secuencia_esperada: 3 }, "clave-acto-1");
  assert.equal(llamadas[1].ruta, RUTA_RESOLUCIONES_OFERTA);
  assert.deepEqual(JSON.parse(llamadas[1].opciones.body), { bolsa_ref: "bolsa:1", oferta_ref: "oferta:1", numero_de_plaza: 2,
    tipo: "llamamiento_directo", secuencia_esperada: 3, participacion_ref: null });
  for (const { opciones } of llamadas) {
    assert.equal(opciones.credentials, "same-origin");
    assert.equal(Object.keys(opciones.headers).some((c) => /persona|actor|usuario/iu.test(c)), false);
  }
});

test("los errores del servidor se traducen sin mostrar códigos y los plurales usan el catálogo", () => {
  const t = crearTraductorOfertas();
  assert.match(mensajeError(t, { status: 409, codigo: "propuesta_cambiada" }), /La plaza ha cambiado/);
  assert.match(mensajeError(t, { status: 409, codigo: "respuesta_abierta" }), /en plazo para responder/);
  assert.match(mensajeError(t, { status: 409, codigo: "politica_sin_plazas" }), /no admite varias plazas/);
  assert.match(mensajeError(t, { status: 503, codigo: "plazo_no_configurado" }), /no tiene plazo/);
  assert.match(mensajeError(t, { status: 500, codigo: "x" }), /No se pudo completar/);
  assert.equal(t("plazas_total", { cuenta: 1 }), "1 plaza");
  assert.equal(t("plazas_cubiertas", { cuenta: 3, cubiertas: 2 }), "2 de 3 cubiertas");
});

test("cada estado de plaza ofrece solo las acciones que admite", () => {
  assert.deepEqual(accionesPlaza(plaza(1, { propuesta: { tipo: "adjudicar", participacion_ref: "p:3", orden_vigente: 2 } })).map((a) => a.tipo), ["adjudicada"]);
  assert.deepEqual(accionesPlaza(plaza(1, { propuesta: { tipo: "llamamiento_directo" } })).map((a) => a.tipo), ["llamamiento_directo"]);
  const pendiente = accionesPlaza(plaza(1, { estado: "pendiente_respuesta", participacion_ref: "p:3", orden_vigente: 2, secuencia: 1 }));
  assert.deepEqual(pendiente.map((a) => [a.tipo, Boolean(a.deshabilitada)]), [["aceptada", false], ["renuncia", false], ["sin_respuesta", true]]);
  assert.equal(accionesPlaza(plaza(1, { estado: "cubierta", participacion_ref: "p:3", orden_vigente: 2 })).length, 0);
  assert.equal(accionesPlaza(plaza(1)).length, 0);
});

test("la superficie publica con número de plazas y clave estable al reintentar, y escapa los datos", async () => {
  let claves = 0; const enviadas = [];
  let fallar = true;
  const cliente = {
    consultar: async () => sobre([oferta()]),
    publicar: async (_b, _d, plazas, clave) => { enviadas.push([clave, plazas]); if (fallar) { fallar = false; return { ok: false, status: 503, codigo: "servicio_no_disponible" }; } return { ok: true, status: 201, oferta: oferta() }; },
    registrarActo: async () => ({ ok: true, status: 201, oferta: oferta() }),
  };
  const s = crearSuperficieOfertasBolsa({ cliente, generarClave: () => `clave-${++claves}` });
  s.activar("bolsa:1");
  await turno();
  const html = s.renderizar();
  assert.match(html, /Auxiliar &lt;b&gt;/);
  assert.doesNotMatch(html, /<b>/);
  assert.match(html, /name="numero_plazas"[^>]*value="1"[^>]*required|name="numero_plazas"[^>]*required/);
  assert.match(html, /1 plaza/);
  assert.doesNotMatch(html, /plazas-oferta/, "una oferta abierta no despliega sus plazas");
  const datos = { categoria: "Aux", centro: "Res", fecha_inicio: "2026-10-01", descripcion: "Des", numero_plazas: "3" };
  const formulario = { closest: () => formulario, reportValidity: () => true };
  globalThis.FormData = class { get(k) { return datos[k] ?? ""; } };
  s.manejarSubmit({ target: formulario, preventDefault() {} });
  await turno();
  assert.match(s.renderizar(), /Reintentar la misma publicación/);
  s.manejarSubmit({ target: formulario, preventDefault() {} });
  await turno();
  assert.deepEqual(enviadas, [["clave-1", 3], ["clave-1", 3]]);
  datos.numero_plazas = "101";
  s.manejarSubmit({ target: formulario, preventDefault() {} });
  await turno();
  assert.equal(enviadas.length, 2, "no se publica fuera del límite de plazas");
});

test("tras el plazo muestra cada plaza, pide confirmación y registra el acto con su secuencia", async () => {
  const vencida = oferta({ oferta_ref: "oferta:2", estado: "en_curso", numero_plazas: 3, politica_plazas: politica, disposiciones_total: 3,
    disposiciones: [{ participacion_ref: "p:3", manifestada_en: "2026-09-26T08:00:00Z", orden_vigente: 2, situacion: "disponible" }],
    plazas: [
      plaza(1, { estado: "pendiente_respuesta", secuencia: 1, participacion_ref: "p:1", orden_vigente: 1, responder_antes_de: "2026-09-30T22:00:00Z",
        historial: [{ secuencia: 1, tipo: "adjudicada", orden_vigente: 1, registrado_en: "2026-09-29T22:00:00Z", recibo_ref: "recibo:plaza-oferta:1" }] }),
      plaza(2, { secuencia: 2, propuesta: { tipo: "adjudicar", participacion_ref: "p:5", orden_vigente: 5 },
        historial: [{ secuencia: 1, tipo: "adjudicada", orden_vigente: 3 }, { secuencia: 2, tipo: "renuncia", orden_vigente: 3 }] }),
      plaza(3, { propuesta: { tipo: "llamamiento_directo" } }),
    ] });
  const actos = [];
  let claves = 0;
  const cliente = {
    consultar: async () => sobre([vencida]),
    publicar: async () => ({ ok: false, status: 500 }),
    registrarActo: async (_b, acto, clave) => { actos.push([acto, clave]); return actos.length === 1 ? { ok: false, status: 503 } : { ok: true, status: 201, oferta: vencida }; },
  };
  const anuncios = [];
  const s = crearSuperficieOfertasBolsa({ cliente, generarClave: () => `clave-acto-${++claves}`, anunciar: (m) => anuncios.push(m) });
  s.activar("bolsa:1");
  await turno();
  const html = s.renderizar();
  assert.match(html, /0 de 3 cubiertas/);
  assert.match(html, /Plaza 1/);
  assert.match(html, /Esperando respuesta/);
  assert.match(html, /Responde antes del/);
  assert.match(html, />Aceptó</);
  assert.match(html, /data-tipo="sin_respuesta"[^>]*aria-describedby="[^"]+"[^>]*disabled/);
  assert.match(html, /Adjudicar al n\.º 5/);
  assert.match(html, /Antes: n\.º 3 renunció/);
  assert.match(html, /Pasar a llamamiento directo/);
  assert.match(html, /No quedan personas que se ofrecieran/);
  assert.doesNotMatch(html.replace(/<[^>]+>/g, " "), /p:5|p:1|recibo:plaza-oferta|oferta:2/, "sin referencias internas en el texto visible");
  // Primer clic: solo pide confirmación.
  s.manejarClick(boton({ ofertasAccion: "preparar", ofertaRef: "oferta:2", plaza: "2", tipo: "adjudicada", secuencia: "2", participacionRef: "p:5", orden: "5" }));
  assert.equal(actos.length, 0);
  assert.match(s.renderizar(), /¿Adjudicar la plaza 2 al n\.º de orden 5\? Tendrá que responder/);
  assert.match(html, /Podrá marcar «No respondió» cuando venza el plazo/);
  s.manejarClick(boton({ ofertasAccion: "cancelar" }));
  assert.doesNotMatch(s.renderizar(), /¿Adjudicar/);
  s.manejarClick(boton({ ofertasAccion: "preparar", ofertaRef: "oferta:2", plaza: "2", tipo: "adjudicada", secuencia: "2", participacionRef: "p:5", orden: "5" }));
  s.manejarClick(boton({ ofertasAccion: "confirmar" }));
  await turno();
  // El fallo transitorio conserva la confirmación y la clave del mismo acto.
  assert.match(s.renderizar(), /¿Adjudicar la plaza 2/);
  s.manejarClick(boton({ ofertasAccion: "confirmar" }));
  await turno();
  assert.deepEqual(actos.map(([a, c]) => [a.numero_de_plaza, a.tipo, a.participacion_ref, a.secuencia_esperada, c]),
    [[2, "adjudicada", "p:5", 2, "clave-acto-1"], [2, "adjudicada", "p:5", 2, "clave-acto-1"]]);
  assert.deepEqual(anuncios, ["Plaza 2 adjudicada al n.º de orden 5. Registre su respuesta cuando llegue."]);
  s.manejarClick(boton({ ofertasAccion: "preparar", ofertaRef: "oferta:2", plaza: "3", tipo: "llamamiento_directo", secuencia: "0" }));
  s.manejarClick(boton({ ofertasAccion: "confirmar" }));
  await turno();
  assert.deepEqual([actos[2][0].tipo, actos[2][0].participacion_ref, actos[2][1]], ["llamamiento_directo", null, "clave-acto-2"]);
});

test("un conflicto descarta la confirmación, recarga y explica el cambio", async () => {
  const vencida = oferta({ estado: "pendiente_resolucion", plazas: [plaza(1, { propuesta: { tipo: "adjudicar", participacion_ref: "p:2", orden_vigente: 2 } })] });
  let consultas = 0;
  const cliente = {
    consultar: async () => { consultas++; return sobre([vencida]); },
    registrarActo: async () => ({ ok: false, status: 409, codigo: "propuesta_cambiada" }),
  };
  const s = crearSuperficieOfertasBolsa({ cliente, generarClave: () => "clave-x" });
  s.activar("bolsa:1");
  await turno();
  s.manejarClick(boton({ ofertasAccion: "preparar", ofertaRef: "oferta:1", plaza: "1", tipo: "adjudicada", secuencia: "0", participacionRef: "p:2", orden: "2" }));
  s.manejarClick(boton({ ofertasAccion: "confirmar" }));
  await turno();
  assert.equal(consultas, 2);
  assert.equal(s.estado().confirmacion, null);
  assert.match(s.renderizar(), /La plaza ha cambiado mientras la consultaba/);
});
