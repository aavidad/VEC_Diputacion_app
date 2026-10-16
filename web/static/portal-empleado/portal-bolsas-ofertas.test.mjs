import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { accionesPlaza, crearClienteOfertas, crearSuperficieOfertasBolsa, crearTraductorOfertas, mensajeError, validarOfertasBolsa,
  ESQUEMA_OFERTAS_BOLSA, RUTA_OFERTAS_BOLSA, RUTA_RESOLUCIONES_OFERTA } from "./portal-bolsas-ofertas.js?v=20261010-ct-bolsa-cohorte-v10";

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

const notificacion = { notificada_en: "2026-09-25T08:00:00.000000Z", referencia_correo: "correo:oferta-1",
  huella_correo_sha256: "a".repeat(64), fuente: "correo_externo_declarado_rrhh" };

const politica = { llamada: "simultanea", respuesta_horas: 24, tras_renuncia: "siguiente_en_orden" };
const turno = () => new Promise((r) => setTimeout(r, 0));
const sobre = (ofertas, inicioPlazo) => ({ ok: true, datos: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas,
  ...(inicioPlazo ? { inicio_plazo: inicioPlazo } : {}) } });

function respuesta(status, cuerpo) {
  return { ok: status >= 200 && status < 300, status, json: async () => cuerpo };
}

function boton(dataset) {
  const b = { disabled: false, dataset };
  b.closest = () => b;
  return { target: b };
}

test("la entrada y la caché cargan la versión de ofertas con avisos", async () => {
  const [portal, html, cache] = await Promise.all(["portal.js", "index.html", "cache-publica-v1.json"]
    .map((nombre) => readFile(new URL(nombre, import.meta.url), "utf8")));
  assert.match(portal, /portal-bolsas-ofertas\.js\?v=20261016-f118-reintento-incierto-v1/);
  assert.match(html, /portal\.js\?v=20261016-f118-reintento-incierto-v1/);
  assert.ok(JSON.parse(cache).propios.includes("/portal-empleado/portal.js?v=20261016-f118-reintento-incierto-v1"));
});

test("la ficha abre el historial autorizado de la oferta elegida y lo desmonta al cerrar", async () => {
  const referencia = `oferta:${"a".repeat(64)}`;
  const llamadas = [];
  const clienteHistorial = {
    consultar: async (bolsa, ofertaRef) => { llamadas.push([bolsa, ofertaRef]); return { ok: true, datos: { contactos: [], cursor_siguiente: null } }; },
    buscar: async () => ({ ok: true, datos: { candidatos: [], cursor_siguiente: null } }),
  };
  const superficie = crearSuperficieOfertasBolsa({ cliente: { consultar: async () => sobre([oferta({ oferta_ref: referencia })]) }, clienteHistorial });
  superficie.activar("bolsa:1"); await turno();
  assert.match(superficie.renderizar(), /Historial de ofrecimientos/);
  superficie.manejarClick(boton({ ofertasAccion: "historial", ofertaRef: referencia }));
  await turno();
  assert.deepEqual(llamadas, [["bolsa:1", referencia]]);
  assert.match(superficie.renderizar(), /No hay ofrecimientos registrados/);
  superficie.manejarClick(boton({ ofertasAccion: "historial", ofertaRef: referencia }));
  assert.doesNotMatch(superficie.renderizar(), /No hay ofrecimientos registrados/);
  superficie.desmontar();
});

test("el contrato exige una entrada por plaza y rechaza estados, propuestas y actos ajenos", () => {
  assert.equal(validarOfertasBolsa({ data: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [oferta()] } }).ofertas.length, 1);
  assert.equal(validarOfertasBolsa({ data: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [oferta({ plazo: { ejemplo: false, notificacion } })] } }).ofertas.length, 1);
  const invalidas = [
    oferta({ recibo_ref: "" }),
    oferta({ recibo_ref: "recibo:oferta:<script>" }),
    oferta({ estado: "inventado" }),
    oferta({ numero_plazas: 2 }),
    oferta({ numero_plazas: 0, plazas: [] }),
    oferta({ plazas: [plaza(2)] }),
    oferta({ plazas: [plaza(1, { estado: "ocupada" })] }),
    oferta({ plazas: [plaza(1, { propuesta: { tipo: "adjudicar" } })] }),
    oferta({ plazas: [plaza(1, { historial: [{ secuencia: 1, tipo: "borrada", orden_vigente: 1 }] })] }),
    oferta({ plazas: [plaza(1, { responder_antes_de: "mañana" })] }),
    oferta({ plazo: { ejemplo: false, notificacion: { ...notificacion, fuente: "entrega_acreditada" } } }),
  ];
  for (const o of invalidas) assert.throws(() => validarOfertasBolsa({ data: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [o] } }));
});

test("el aviso es opcional y solo admite recuentos enteros coherentes", () => {
  const aviso = { total: 5, pendientes: 1, enviados: 2, sin_confirmacion: 1, fallidos: 1 };
  assert.equal(validarOfertasBolsa({ data: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [oferta()] } }).ofertas[0].aviso, undefined);
  assert.deepEqual(validarOfertasBolsa({ data: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [oferta({ aviso })] } }).ofertas[0].aviso, aviso);
  assert.equal(validarOfertasBolsa({ data: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [oferta({ aviso, aviso_configuracion_pendiente: true })] } }).ofertas[0].aviso_configuracion_pendiente, true);
  for (const invalido of [null, 0, "true"]) {
    assert.throws(() => validarOfertasBolsa({ data: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [oferta({ aviso_configuracion_pendiente: invalido })] } }));
  }
  for (const invalido of [null, {}, { ...aviso, pendientes: -1 }, { ...aviso, enviados: 1.5 },
    { ...aviso, sin_confirmacion: "1" }, { ...aviso, total: 4 }, { ...aviso, total: Number.MAX_SAFE_INTEGER + 1 }]) {
    assert.throws(() => validarOfertasBolsa({ data: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [oferta({ aviso: invalido })] } }));
  }
});

test("la revisión y el recibo manual son opcionales y rechazan datos ajenos", () => {
  const aviso = { total: 2, pendientes: 0, enviados: 1, sin_confirmacion: 1, fallidos: 0, revision: 3 };
  const reintento_aviso = { recibo_ref: "recibo:reintento:1", registrado_en: "2026-10-16T06:10:00Z", rearmados: 1 };
  assert.equal(validarOfertasBolsa({ data: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [oferta({ aviso, reintento_aviso })] } }).ofertas[0].aviso.revision, 3);
  for (const invalido of [-1, 1.5, "3", null]) {
    assert.throws(() => validarOfertasBolsa({ data: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [oferta({ aviso: { ...aviso, revision: invalido } })] } }));
  }
  for (const invalido of [null, { ...reintento_aviso, recibo_ref: "<script>" }, { ...reintento_aviso, registrado_en: "mañana" },
    { ...reintento_aviso, rearmados: -1 }]) {
    assert.throws(() => validarOfertasBolsa({ data: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [oferta({ reintento_aviso: invalido })] } }));
  }
});

test("el cliente envía solo la variante tipada y su propia clave de reintento", async () => {
  const llamadas = [];
  const cliente = crearClienteOfertas({ fetchImpl: async (ruta, opciones) => { llamadas.push({ ruta, opciones }); return respuesta(201, {
    data: oferta({ reintento_aviso: { recibo_ref: "recibo:reintento:1", registrado_en: "2026-10-16T06:10:00Z", rearmados: 1 } }),
  }); } });
  const cuerpo = { tipo: "reintentar_avisos_sin_confirmacion", bolsa_ref: "bolsa:1", oferta_ref: "oferta:1",
    revision_avisos: 3, confirmar_posible_duplicado: true };
  assert.equal((await cliente.reintentarAvisosSinConfirmacion(cuerpo, "clave-manual-1")).ok, true);
  assert.equal(llamadas[0].ruta, RUTA_OFERTAS_BOLSA);
  assert.equal(llamadas[0].opciones.headers["Idempotency-Key"], "clave-manual-1");
  assert.deepEqual(JSON.parse(llamadas[0].opciones.body), cuerpo);
  assert.equal(llamadas[0].opciones.credentials, "same-origin");
  assert.equal((await cliente.reintentarAvisosSinConfirmacion({ ...cuerpo, confirmar_posible_duplicado: false }, "clave-erronea")).ok, false);
  assert.equal(llamadas.length, 1);
});

test("la revisión manual exige confirmación y conserva el recibo de publicación", async () => {
  const aviso = { total: 2, pendientes: 0, enviados: 1, sin_confirmacion: 1, fallidos: 0, revision: 3 };
  const reintento_aviso = { recibo_ref: "recibo:reintento:1", registrado_en: "2026-10-16T06:10:00Z", rearmados: 1 };
  let vista = oferta({ aviso });
  const llamadas = [];
  const superficie = crearSuperficieOfertasBolsa({ cliente: {
    consultar: async () => sobre([vista], "publicacion"),
    reintentarAvisosSinConfirmacion: async (cuerpo, clave) => { llamadas.push({ cuerpo, clave });
      vista = oferta({ aviso: { ...aviso, pendientes: 0, enviados: 2, sin_confirmacion: 0, revision: 4 }, reintento_aviso });
      return { ok: true, status: 201, oferta: vista }; },
  }, generarClave: () => "clave-manual-1" });
  superficie.activar("bolsa:1"); await turno();
  assert.match(superficie.renderizar(), /Reintentar envíos sin confirmación/);
  superficie.manejarClick(boton({ ofertasAccion: "preparar-reintento-incierto", ofertaRef: "oferta:1" }));
  assert.match(superficie.renderizar(), /El correo puede haberse recibido/);
  assert.match(superficie.renderizar(), /Auxiliar &lt;b&gt;.*1 persona/);
  assert.equal(llamadas.length, 0);
  superficie.manejarClick(boton({ ofertasAccion: "cancelar-reintento-incierto" }));
  assert.equal(llamadas.length, 0);
  superficie.manejarClick(boton({ ofertasAccion: "preparar-reintento-incierto", ofertaRef: "oferta:1" }));
  superficie.manejarClick(boton({ ofertasAccion: "confirmar-reintento-incierto" })); await turno();
  assert.deepEqual(llamadas, [{ cuerpo: { tipo: "reintentar_avisos_sin_confirmacion", bolsa_ref: "bolsa:1", oferta_ref: "oferta:1",
    revision_avisos: 3, confirmar_posible_duplicado: true }, clave: "clave-manual-1" }]);
  assert.match(superficie.renderizar(), /Recibo del reintento de avisos/);
  assert.match(superficie.renderizar(), /Reintento registrado el/);
  assert.match(superficie.renderizar(), /Publicada el/);
  assert.match(superficie.renderizar(), /recibo:reintento:1/);
  assert.match(superficie.renderizar(), /recibo:oferta:1/);
  assert.doesNotMatch(superficie.renderizar().replaceAll(/data-copiar-justificante="[^"]*"/g, ""), /recibo:reintento:1/);
  assert.equal(superficie.estado().intencionAvisoIncierto, null);
  superficie.desmontar();
});

test("sin revisión o con configuración pendiente no aparece el reintento manual", async () => {
  const base = { total: 1, pendientes: 0, enviados: 0, sin_confirmacion: 1, fallidos: 0 };
  const superficie = crearSuperficieOfertasBolsa({ cliente: { consultar: async () => sobre([
    oferta({ aviso: base }), oferta({ oferta_ref: "oferta:2", aviso: { ...base, revision: 1 }, aviso_configuracion_pendiente: true }),
  ]) } });
  superficie.activar("bolsa:1"); await turno();
  assert.doesNotMatch(superficie.renderizar(), /Reintentar envíos sin confirmación/);
  superficie.manejarClick(boton({ ofertasAccion: "preparar-reintento-incierto", ofertaRef: "oferta:1" }));
  assert.equal(superficie.estado().revisionAvisoIncierto, null);
  superficie.desmontar();
});

test("la red incierta conserva cuerpo y clave; la recuperación expresa no rearma otra vez", async () => {
  const aviso = { total: 1, pendientes: 0, enviados: 0, sin_confirmacion: 1, fallidos: 0, revision: 2 };
  const reintento_aviso = { recibo_ref: "recibo:reintento:estable", registrado_en: "2026-10-16T06:20:00Z", rearmados: 1 };
  let vista = oferta({ aviso });
  const llamadas = [];
  let claves = 0;
  const superficie = crearSuperficieOfertasBolsa({ cliente: {
    consultar: async () => sobre([vista], "publicacion"),
    reintentarAvisosSinConfirmacion: async (cuerpo, clave) => { llamadas.push({ cuerpo: { ...cuerpo }, clave });
      if (llamadas.length === 1) throw new Error("red interrumpida");
      vista = oferta({ aviso: { ...aviso, pendientes: 0, enviados: 1, sin_confirmacion: 0, revision: 3 }, reintento_aviso });
      return { ok: true, status: 200, oferta: vista }; },
  }, generarClave: () => `clave-${++claves}` });
  superficie.activar("bolsa:1"); await turno();
  superficie.manejarClick(boton({ ofertasAccion: "preparar-reintento-incierto", ofertaRef: "oferta:1" }));
  superficie.manejarClick(boton({ ofertasAccion: "confirmar-reintento-incierto" })); await turno();
  assert.equal(claves, 1);
  assert.match(superficie.renderizar(), /Recuperar el resultado del reintento/);
  assert.equal(superficie.estado().intencionAvisoIncierto.incierta, true);
  superficie.manejarClick(boton({ ofertasAccion: "repetir-reintento-incierto" })); await turno();
  assert.equal(claves, 1);
  assert.deepEqual(llamadas[1], llamadas[0]);
  assert.equal(superficie.estado().intencionAvisoIncierto, null);
  assert.match(superficie.renderizar(), /recibo:reintento:estable/);
  superficie.desmontar();
});

test("un reintento con pendientes continúa con la misma clave y un incierto nuevo exige otra revisión", async () => {
  const aviso = { total: 2, pendientes: 0, enviados: 1, sin_confirmacion: 1, fallidos: 0, revision: 4 };
  const recibo = { recibo_ref: "recibo:reintento:2", registrado_en: "2026-10-16T06:30:00Z", rearmados: 1 };
  let vista = oferta({ aviso });
  const llamadas = [];
  let claves = 0;
  const superficie = crearSuperficieOfertasBolsa({ cliente: {
    consultar: async () => sobre([vista], "publicacion"),
    reintentarAvisosSinConfirmacion: async (cuerpo, clave) => { llamadas.push({ cuerpo: { ...cuerpo }, clave });
      vista = oferta({ aviso: llamadas.length === 1 ? { ...aviso, pendientes: 1, sin_confirmacion: 0, revision: 5 }
        : { ...aviso, pendientes: 0, sin_confirmacion: 1, revision: 6 }, reintento_aviso: recibo });
      return { ok: true, status: llamadas.length === 1 ? 201 : 200, oferta: vista }; },
  }, generarClave: () => `clave-${++claves}` });
  superficie.activar("bolsa:1"); await turno();
  superficie.manejarClick(boton({ ofertasAccion: "preparar-reintento-incierto", ofertaRef: "oferta:1" }));
  superficie.manejarClick(boton({ ofertasAccion: "confirmar-reintento-incierto" })); await turno();
  assert.match(superficie.renderizar(), /Continuar el envío pendiente/);
  superficie.manejarClick(boton({ ofertasAccion: "repetir-reintento-incierto" })); await turno();
  assert.deepEqual(llamadas[1], llamadas[0]);
  assert.equal(claves, 1);
  assert.equal(superficie.estado().intencionAvisoIncierto, null);
  assert.match(superficie.renderizar(), /Reintentar envíos sin confirmación/);
  assert.equal(llamadas.length, 2, "el nuevo sin confirmar no genera otro POST");
  superficie.manejarClick(boton({ ofertasAccion: "preparar-reintento-incierto", ofertaRef: "oferta:1" }));
  assert.match(superficie.renderizar(), /El correo puede haberse recibido/);
  assert.equal(llamadas.length, 2);
  superficie.desmontar();
});

test("409 consulta el estado actual sin otra clave y 401 borra la intención manual", async () => {
  const aviso = { total: 1, pendientes: 0, enviados: 0, sin_confirmacion: 1, fallidos: 0, revision: 2 };
  for (const status of [409, 401]) {
    let consultas = 0; let envios = 0;
    const superficie = crearSuperficieOfertasBolsa({ cliente: {
      consultar: async () => { consultas++; return sobre([oferta({ aviso: { ...aviso, revision: consultas > 1 ? 3 : 2 } })], "publicacion"); },
      reintentarAvisosSinConfirmacion: async () => { envios++; return { ok: false, status,
        codigo: status === 409 ? "aviso_revision_obsoleta" : "sin_permiso" }; },
    }, generarClave: () => "clave-unica" });
    superficie.activar("bolsa:1"); await turno();
    superficie.manejarClick(boton({ ofertasAccion: "preparar-reintento-incierto", ofertaRef: "oferta:1" }));
    superficie.manejarClick(boton({ ofertasAccion: "confirmar-reintento-incierto" })); await turno();
    assert.equal(envios, 1);
    assert.equal(superficie.estado().intencionAvisoIncierto, null);
    if (status === 409) {
      assert.equal(consultas, 2);
      assert.equal(superficie.estado().ofertas[0].aviso.revision, 3);
      assert.match(superficie.renderizar(), /Los avisos han cambiado/);
    } else {
      assert.deepEqual(superficie.estado().ofertas, []);
      assert.equal(superficie.estado().carga, "error");
    }
    superficie.desmontar();
  }
});

test("la configuración pendiente explica el bloqueo y no ofrece reintento hasta que la consulta la confirme lista", async () => {
  const anteriorFormData = globalThis.FormData;
  globalThis.FormData = class { get(clave) { return { categoria: "Aux", centro: "Res", fecha_inicio: "2026-10-01", descripcion: "Cobertura", numero_plazas: "1" }[clave] ?? ""; } };
  const formulario = { closest: () => formulario, reportValidity: () => true };
  const aviso = { total: 1, pendientes: 1, enviados: 0, sin_confirmacion: 0, fallidos: 0 };
  let configuracionPendiente = true;
  let publicaciones = 0;
  const superficie = crearSuperficieOfertasBolsa({ cliente: {
    consultar: async () => sobre([oferta({ aviso, aviso_configuracion_pendiente: configuracionPendiente })], "publicacion"),
    publicar: async () => { publicaciones++; return { ok: true, status: 201, oferta: oferta({ aviso, aviso_configuracion_pendiente: true }) }; },
  }, generarClave: () => "clave-original" });
  try {
    superficie.activar("bolsa:1"); await turno();
    assert.match(superficie.renderizar(), /Falta configurar el envío de avisos de las ofertas/);
    superficie.manejarSubmit({ target: formulario, preventDefault() {} });
    superficie.manejarClick(boton({ ofertasAccion: "confirmar-publicacion" })); await turno();
    assert.equal(superficie.estado().reintentoAvisos.clave, "clave-original");
    assert.doesNotMatch(superficie.renderizar(), /Reintentar avisos pendientes/);
    assert.match(superficie.renderizar(), /data-ofertas-accion="recargar">Actualizar estado/);
    superficie.manejarClick(boton({ ofertasAccion: "recuperar-avisos" }));
    assert.equal(publicaciones, 1, "no reintenta con la configuración pendiente");
    configuracionPendiente = false;
    superficie.manejarClick(boton({ ofertasAccion: "recargar" })); await turno();
    assert.match(superficie.renderizar(), /Reintentar avisos pendientes/);
    assert.doesNotMatch(superficie.renderizar(), /Falta configurar el envío de avisos de las ofertas/);
  } finally { superficie.desmontar(); globalThis.FormData = anteriorFormData; }
});

test("la ficha muestra solo el aviso recibido y distingue los envíos sin confirmar", async () => {
  const aviso = { total: 4, pendientes: 1, enviados: 1, sin_confirmacion: 1, fallidos: 1 };
  const superficie = crearSuperficieOfertasBolsa({ cliente: { consultar: async () => sobre([oferta(), oferta({ oferta_ref: "oferta:2", aviso })]) } });
  superficie.activar("bolsa:1"); await turno();
  const html = superficie.renderizar();
  assert.equal((html.match(/Avisos por correo para 4 personas/g) || []).length, 2, "una etiqueta y un título solo en la oferta con aviso");
  assert.match(html, /Enviados/);
  assert.match(html, /Sin confirmación del servicio de correo/);
  assert.match(html, /<summary aria-label="¿Qué significa «Enviados»\?"[^>]*>\?<\/summary>/);
  assert.doesNotMatch(html, /correo electrónico de una persona|Reintentar avisos/);
  superficie.desmontar();
});

test("el recibo reintenta solo con la intención original mientras haya avisos pendientes", async () => {
  const anteriorFormData = globalThis.FormData;
  const valores = { categoria: "Aux", centro: "Res", fecha_inicio: "2026-10-01", descripcion: "Cobertura", numero_plazas: "2" };
  globalThis.FormData = class { get(clave) { return valores[clave] ?? ""; } };
  const formulario = { closest: () => formulario, reportValidity: () => true };
  const llamadas = [];
  let completar;
  const avisoPendiente = { total: 2, pendientes: 1, enviados: 1, sin_confirmacion: 0, fallidos: 0 };
  const avisoSinConfirmacion = { total: 2, pendientes: 0, enviados: 1, sin_confirmacion: 1, fallidos: 0 };
  const superficie = crearSuperficieOfertasBolsa({ cliente: {
    consultar: async () => sobre([], "publicacion"),
    publicar: async (_bolsa, datos, plazas, notificacion, clave) => {
      llamadas.push({ datos: structuredClone(datos), plazas, notificacion, clave });
      if (llamadas.length === 1) return { ok: true, status: 201, oferta: oferta({ aviso: avisoPendiente }) };
      return new Promise((resolver) => { completar = resolver; });
    },
  }, generarClave: () => "clave-estable" });
  const focos = [];
  superficie.instalar({ addEventListener() {}, querySelector(selector) { return { focus() { focos.push(selector); } }; } });
  try {
    superficie.activar("bolsa:1"); await turno();
    superficie.manejarSubmit({ target: formulario, preventDefault() {} });
    superficie.manejarClick(boton({ ofertasAccion: "confirmar-publicacion" })); await turno();
    assert.match(superficie.renderizar(), /Reintentar avisos pendientes/);
    valores.descripcion = "Texto alterado en el formulario";
    superficie.manejarClick(boton({ ofertasAccion: "recuperar-avisos" }));
    superficie.manejarClick(boton({ ofertasAccion: "recuperar-avisos" }));
    assert.equal(llamadas.length, 2, "el segundo clic no envía otra petición");
    assert.deepEqual(llamadas[1], llamadas[0], "cuerpo y clave son los de la publicación original");
    completar({ ok: true, status: 200, oferta: oferta({ aviso: avisoSinConfirmacion }) }); await turno();
    assert.doesNotMatch(superficie.renderizar(), /Reintentar avisos pendientes/);
    assert.match(superficie.renderizar(), /Se ha recuperado el estado de los avisos/);
    assert.equal(superficie.estado().reintentoAvisos, null, "los avisos sin confirmación no se reintentan");
    assert.equal(focos.at(-1), '[data-ofertas-aviso="exito"]', "el foco vuelve al recibo después del GET y el repintado");
  } finally { superficie.desmontar(); globalThis.FormData = anteriorFormData; }
});

test("un reintento incierto bloquea otra operación y los rechazos de acceso retiran los datos", async () => {
  const anteriorFormData = globalThis.FormData;
  globalThis.FormData = class { get(clave) { return { categoria: "Aux", centro: "Res", fecha_inicio: "2026-10-01", descripcion: "Cobertura", numero_plazas: "1" }[clave] ?? ""; } };
  const formulario = { closest: () => formulario, reportValidity: () => true };
  const aviso = { total: 1, pendientes: 1, enviados: 0, sin_confirmacion: 0, fallidos: 0 };
  try {
    for (const fallo of ["red", 401, 403]) {
      let llamadas = 0;
      const superficie = crearSuperficieOfertasBolsa({ cliente: {
        consultar: async () => sobre([], "publicacion"),
        publicar: async () => { llamadas++; if (llamadas === 1) return { ok: true, status: 201, oferta: oferta({ aviso }) };
          if (fallo === "red") throw new Error("conexión perdida");
          return { ok: false, status: fallo, codigo: "sin_permiso" }; },
      }, generarClave: () => "clave-original" });
      superficie.activar("bolsa:1"); await turno();
      superficie.manejarSubmit({ target: formulario, preventDefault() {} });
      superficie.manejarClick(boton({ ofertasAccion: "confirmar-publicacion" })); await turno();
      superficie.manejarClick(boton({ ofertasAccion: "recuperar-avisos" })); await turno();
      assert.doesNotMatch(superficie.renderizar(), /Reintentar avisos pendientes/);
      if (fallo === "red") {
        assert.equal(superficie.estado().reintentoAvisos.clave, "clave-original");
        assert.equal(superficie.estado().reintentoAvisos.incierta, true);
        assert.match(superficie.renderizar(), /No se pudo confirmar el resultado/);
        superficie.manejarClick(boton({ ofertasAccion: "recuperar-avisos" }));
        assert.equal(llamadas, 2);
      } else {
        assert.equal(superficie.estado().reintentoAvisos, null);
        assert.deepEqual(superficie.estado().ofertas, []);
        assert.match(superficie.renderizar(), fallo === 401 ? /Su sesión ha terminado/ : /Su sesión no permite gestionar/);
      }
      superficie.desmontar();
    }
  } finally { globalThis.FormData = anteriorFormData; }
});

test("tras leer ofertas, un POST o GET denegado retira datos, recibo e intención", async () => {
  const anteriorFormData = globalThis.FormData;
  globalThis.FormData = class { get(clave) { return { categoria: "Aux", centro: "Res", fecha_inicio: "2026-10-01", descripcion: "Cobertura", numero_plazas: "1" }[clave] ?? ""; } };
  const formulario = { closest: () => formulario, reportValidity: () => true };
  try {
    for (const status of [401, 403]) {
      const superficie = crearSuperficieOfertasBolsa({ cliente: {
        consultar: async () => sobre([oferta()], "publicacion"),
        publicar: async () => ({ ok: false, status, codigo: "sin_permiso" }),
      }, generarClave: () => "clave-denegada" });
      superficie.activar("bolsa:1"); await turno();
      assert.equal(superficie.estado().ofertas.length, 1);
      superficie.manejarSubmit({ target: formulario, preventDefault() {} });
      superficie.manejarClick(boton({ ofertasAccion: "confirmar-publicacion" })); await turno();
      const estado = superficie.estado();
      assert.deepEqual(estado.ofertas, []);
      assert.equal(estado.reciboPublicacion, null);
      assert.equal(estado.reintentoAvisos, null);
      assert.equal(estado.claveEnCurso, null);
      assert.equal(estado.borrador, null);
      assert.equal(estado.carga, "error");
      assert.doesNotMatch(superficie.renderizar(), /Auxiliar &lt;b&gt;/);
      superficie.desmontar();
    }
    let autorizado = true;
    const lectura = crearSuperficieOfertasBolsa({ cliente: { consultar: async () => autorizado ? sobre([oferta()], "publicacion") : { ok: false, status: 403 } } });
    lectura.activar("bolsa:1"); await turno();
    autorizado = false; lectura.recargarPoliticaSegura(); await turno();
    assert.deepEqual(lectura.estado().ofertas, []);
    assert.equal(lectura.estado().carga, "error");
    lectura.desmontar();
  } finally { globalThis.FormData = anteriorFormData; }
});

test("el cliente envía bolsa, datos, número de plazas y el acto de la plaza, sin identidad", async () => {
  const llamadas = [];
  const cliente = crearClienteOfertas({ fetchImpl: async (ruta, opciones) => { llamadas.push({ ruta, opciones }); return respuesta(201, { data: oferta() }); } });
  const r = await cliente.publicar("bolsa:1", { categoria: "Aux", centro: "Res", fecha_inicio: "2026-10-01", descripcion: "Des" }, 3, notificacion, "oferta-clave-1");
  assert.equal(r.ok, true);
  assert.equal(llamadas[0].ruta, RUTA_OFERTAS_BOLSA);
  assert.equal(llamadas[0].opciones.headers["Idempotency-Key"], "oferta-clave-1");
  assert.deepEqual(JSON.parse(llamadas[0].opciones.body).numero_plazas, 3);
  assert.deepEqual(Object.keys(JSON.parse(llamadas[0].opciones.body)), ["bolsa_ref", "datos", "numero_plazas", "notificacion"]);
  assert.deepEqual(JSON.parse(llamadas[0].opciones.body).notificacion, notificacion);
  const invalida = await cliente.publicar("bolsa:1", { categoria: "Aux" }, 1,
    { ...notificacion, referencia_correo: "nie:X1234567L" }, "clave-privada");
  assert.deepEqual(invalida, { ok: false, status: 400, codigo: "solicitud_invalida" });
  assert.equal(llamadas.length, 1, "el cliente no envía identificadores personales");
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
  assert.match(mensajeError(t, { status: 409, codigo: "aviso_revision_obsoleta" }), /Los avisos han cambiado/);
  assert.match(mensajeError(t, { status: 409, codigo: "aviso_envio_en_curso" }), /Hay un envío en curso/);
  assert.match(mensajeError(t, { status: 409, codigo: "respuesta_abierta" }), /en plazo para responder/);
  assert.match(mensajeError(t, { status: 409, codigo: "politica_sin_plazas" }), /no admite varias plazas/);
  assert.match(mensajeError(t, { status: 409, codigo: "personas_elegibles_insuficientes" }), /Reduzca las plazas o revise la disponibilidad/);
  assert.match(mensajeError(t, { status: 503, codigo: "plazo_no_configurado" }), /no tiene plazo/);
  assert.match(mensajeError(t, { status: 422, codigo: "oferta_invalida" }), /fecha del correo no puede ser futura/);
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
    consultar: async () => sobre([oferta({ plazo: { ejemplo: false, notificacion } })]),
    publicar: async (_b, _d, plazas, n, clave) => { enviadas.push([clave, plazas, n]); if (fallar) { fallar = false; return { ok: false, status: 503, codigo: "servicio_no_disponible" }; } return { ok: true, status: 201, oferta: oferta() }; },
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
  assert.match(html, /RRHH declaró un correo externo/);
  assert.match(html, /correo:oferta-1/);
  assert.match(html, /Fecha y hora del correo/);
  assert.match(html, /Compruebe la entrega por el canal correspondiente/);
  assert.doesNotMatch(html, /plazas-oferta/, "una oferta abierta no despliega sus plazas");
  const datos = { categoria: "Aux", centro: "Res", fecha_inicio: "2026-10-01", descripcion: "Des", numero_plazas: "3",
    notificada_en: "2026-09-25T10:00", referencia_correo: "correo:oferta-1", huella_correo_sha256: "a".repeat(64) };
  const formulario = { closest: () => formulario, reportValidity: () => true };
  globalThis.FormData = class { get(k) { return datos[k] ?? ""; } };
  datos.huella_correo_sha256 = "invalida";
  s.manejarSubmit({ target: formulario, preventDefault() {} });
  await turno();
  assert.equal(enviadas.length, 0, "no publica sin huella verificable");
  datos.huella_correo_sha256 = "a".repeat(64);
  datos.referencia_correo = "dni:12345678Z";
  s.manejarSubmit({ target: formulario, preventDefault() {} });
  await turno();
  assert.equal(enviadas.length, 0, "no envía un identificador personal como referencia opaca");
  assert.match(s.renderizar(), /no debe incluir DNI, NIE ni otros datos de identidad/);
  datos.referencia_correo = "correo:oferta-1";
  datos.notificada_en = "2026-03-29T02:30";
  s.manejarSubmit({ target: formulario, preventDefault() {} });
  await turno();
  assert.equal(enviadas.length, 0, "no convierte una hora inexistente en Madrid");
  datos.notificada_en = "2026-09-25T10:00";
  s.manejarSubmit({ target: formulario, preventDefault() {} });
  await turno();
  assert.equal(enviadas.length, 0, "la revisión no publica todavía");
  assert.match(s.renderizar(), /Revise la oferta antes de publicarla/);
  assert.match(s.renderizar(), /Fecha y hora declaradas del correo \(Madrid\)/);
  assert.match(s.renderizar(), /25 sept 2026/);
  assert.match(s.renderizar(), /Ver código de comprobación del correo/);
  s.manejarClick(boton({ ofertasAccion: "confirmar-publicacion" }));
  await turno();
  assert.match(s.renderizar(), /Reintentar la misma publicación/);
  assert.equal(s.recargarPoliticaSegura(), false, "no sustituye el cuerpo incierto al guardar la política");
  datos.numero_plazas = "5";
  s.manejarClick(boton({ ofertasAccion: "confirmar-publicacion" }));
  await turno();
  assert.equal(enviadas.length, 2);
  assert.deepEqual(enviadas.map(([clave, plazas]) => [clave, plazas]), [["clave-1", 3], ["clave-1", 3]]);
  assert.deepEqual(enviadas[0][2], enviadas[1][2]);
  assert.equal(enviadas[0][2].notificada_en, "2026-09-25T08:00:00.000000Z", "el navegador no determina la zona");
  const conRecibo = s.renderizar();
  assert.match(conRecibo, /Oferta publicada\. El plazo termina el/);
  assert.match(conRecibo, /<details open><summary>Recibo<\/summary>/, "el recibo y su fecha quedan visibles tras publicar");
  assert.equal((conRecibo.match(/<summary>Recibo<\/summary>/g) || []).length, 2, "el recibo real aparece en la confirmación y en la ficha");
  assert.match(conRecibo, /<dt>Referencia del recibo<\/dt><dd><button[^>]+data-copiar-justificante="recibo:oferta:1"/);
  assert.match(conRecibo, /<dt>Publicada el<\/dt><dd>25 sept 2026/);
  datos.numero_plazas = "101";
  s.manejarSubmit({ target: formulario, preventDefault() {} });
  await turno();
  assert.equal(enviadas.length, 2, "no se publica fuera del límite de plazas");
});

test("el reintento muestra el mismo recibo y un GET sin política indica dónde configurarla", async () => {
  const recibo = oferta({ recibo_ref: "recibo:oferta:original", publicada_en: "2026-10-15T09:30:00Z" });
  const valores = { categoria: "Aux", centro: "Res", fecha_inicio: "2026-10-20", descripcion: "Cobertura", numero_plazas: "2" };
  const anteriorFormData = globalThis.FormData;
  globalThis.FormData = class { get(clave) { return valores[clave] ?? ""; } };
  const formulario = { closest: () => formulario, reportValidity: () => true };
  try {
    const s = crearSuperficieOfertasBolsa({ cliente: {
      consultar: async () => sobre([recibo], "publicacion"),
      publicar: async () => ({ ok: true, status: 200, oferta: recibo }),
    }, generarClave: () => "clave-original" });
    s.activar("bolsa:1"); await turno();
    s.manejarSubmit({ target: formulario, preventDefault() {} });
    s.manejarClick(boton({ ofertasAccion: "confirmar-publicacion" })); await turno();
    const html = s.renderizar();
    assert.match(html, /ya estaba publicada con esta misma solicitud/);
    assert.equal((html.match(/recibo:oferta:original/g) || []).length, 2);
    assert.match(html, /15 oct 2026/);

    const sinPolitica = crearSuperficieOfertasBolsa({ cliente: {
      consultar: async () => ({ ok: false, status: 503, codigo: "plazo_no_configurado" }),
    } });
    sinPolitica.activar("bolsa:1"); await turno();
    assert.match(sinPolitica.renderizar(), /Configure el inicio y el plazo en «Política de ofertas»/);
  } finally { globalThis.FormData = anteriorFormData; }
});

test("una publicación incierta conserva clave y cuerpo tras 422 y 409 hasta recibir el recibo", async () => {
  const valores = { categoria: "Aux", centro: "Res", fecha_inicio: "2026-10-20", descripcion: "Cobertura original", numero_plazas: "2" };
  const anteriorFormData = globalThis.FormData;
  globalThis.FormData = class { get(clave) { return valores[clave] ?? ""; } };
  const formulario = { closest: () => formulario, reportValidity: () => true };
  const llamadas = [];
  let consultas = 0;
  const respuestas = [503, 422, 409, 200];
  try {
    const s = crearSuperficieOfertasBolsa({ cliente: {
      consultar: async () => { consultas++; return sobre([], "publicacion"); },
      publicar: async (_bolsa, datos, plazas, notificacion, clave) => {
        llamadas.push({ datos: { ...datos }, plazas, notificacion, clave });
        const status = respuestas.shift();
        return status === 200 ? { ok: true, status, oferta: oferta({ recibo_ref: "recibo:oferta:estable" }) } :
          { ok: false, status, codigo: "solicitud_invalida" };
      },
    }, generarClave: () => "clave-original" });
    s.activar("bolsa:1"); await turno();
    s.manejarSubmit({ target: formulario, preventDefault() {} });
    valores.descripcion = "Cobertura cambiada tras confirmar";
    for (let intento = 0; intento < 3; intento++) {
      s.manejarClick(boton({ ofertasAccion: "confirmar-publicacion" })); await turno();
      assert.equal(s.estado().incierta, true);
      assert.equal(s.estado().claveEnCurso, "clave-original");
      assert.ok(s.estado().revisionPublicacion, "el rechazo del reintento no libera la operación previa");
      assert.match(s.renderizar(), /Reintente aquí la misma operación antes de salir/);
    }
    assert.equal(consultas, 1, "un 409 posterior no sustituye la política de una operación pendiente");
    s.manejarClick(boton({ ofertasAccion: "confirmar-publicacion" })); await turno();
    assert.equal(s.estado().incierta, false);
    assert.equal(s.estado().claveEnCurso, null);
    assert.equal(llamadas.length, 4);
    assert.deepEqual(llamadas.slice(1), Array(3).fill(llamadas[0]), "todos los reintentos conservan cuerpo, clave y número de plazas");
    assert.match(s.renderizar(), /recibo:oferta:estable/);
  } finally { globalThis.FormData = anteriorFormData; }
});

test("corregir antes de publicar o tras 422 envía los datos revisados con clave nueva", async () => {
  const valores = { categoria: "Aux", centro: "Res", fecha_inicio: "2026-10-01", descripcion: "A", numero_plazas: "2" };
  const anteriorFormData = globalThis.FormData;
  globalThis.FormData = class { get(clave) { return valores[clave] ?? ""; } };
  const formulario = { closest: () => formulario, reportValidity: () => true };
  const submit = (superficie) => superficie.manejarSubmit({ target: formulario, preventDefault() {} });
  const confirmar = (superficie) => superficie.manejarClick(boton({ ofertasAccion: "confirmar-publicacion" }));
  try {
    const primeras = [];
    const superficie = crearSuperficieOfertasBolsa({ cliente: {
      consultar: async () => sobre([], "publicacion"),
      publicar: async (_bolsa, datos, _plazas, _notificacion, clave) => {
        primeras.push({ descripcion: datos.descripcion, clave }); return { ok: true, status: 201, oferta: oferta() };
      },
    }, generarClave: () => "clave-b" });
    superficie.activar("bolsa:1"); await turno();
    submit(superficie);
    superficie.manejarClick(boton({ ofertasAccion: "corregir-publicacion" }));
    valores.descripcion = "B";
    submit(superficie); confirmar(superficie); await turno();
    assert.deepEqual(primeras, [{ descripcion: "B", clave: "clave-b" }]);

    valores.descripcion = "A";
    const rechazadas = [];
    let intento = 0;
    const trasRechazo = crearSuperficieOfertasBolsa({ cliente: {
      consultar: async () => sobre([], "publicacion"),
      publicar: async (_bolsa, datos, _plazas, _notificacion, clave) => {
        rechazadas.push({ descripcion: datos.descripcion, clave });
        return ++intento === 1 ? { ok: false, status: 422, codigo: "solicitud_invalida" } : { ok: true, status: 201, oferta: oferta() };
      },
    }, generarClave: () => `clave-${intento + 1}` });
    trasRechazo.activar("bolsa:1"); await turno();
    submit(trasRechazo); confirmar(trasRechazo); await turno();
    assert.match(trasRechazo.renderizar(), /name="descripcion"/);
    valores.descripcion = "B";
    submit(trasRechazo); confirmar(trasRechazo); await turno();
    assert.deepEqual(rechazadas, [{ descripcion: "A", clave: "clave-1" }, { descripcion: "B", clave: "clave-2" }]);
  } finally { globalThis.FormData = anteriorFormData; }
});

test("el inicio por publicación omite el correo y exige revisión antes del POST", async () => {
  const llamadas = [];
  const clienteHTTP = crearClienteOfertas({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return respuesta(201, { data: oferta() });
  } });
  await clienteHTTP.publicar("bolsa:1", { categoria: "Aux", centro: "Res", fecha_inicio: "2026-10-01", descripcion: "Des" }, 2, null, "clave-1");
  assert.deepEqual(Object.keys(JSON.parse(llamadas[0].opciones.body)), ["bolsa_ref", "datos", "numero_plazas"]);
  assert.equal(validarOfertasBolsa({ data: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [], inicio_plazo: "publicacion" } }).inicio_plazo, "publicacion");
  assert.throws(() => validarOfertasBolsa({ data: { esquema: ESQUEMA_OFERTAS_BOLSA, ofertas: [], inicio_plazo: "otro" } }));

  const publicaciones = [];
  const superficie = crearSuperficieOfertasBolsa({ cliente: {
    consultar: async () => sobre([], "publicacion"),
    publicar: async (...args) => { publicaciones.push(args); return { ok: true, status: 201, oferta: oferta() }; },
  }, generarClave: () => "clave-publicacion" });
  superficie.activar("bolsa:1"); await turno();
  assert.match(superficie.renderizar(), /El plazo comienza al publicar la oferta/);
  assert.doesNotMatch(superficie.renderizar(), /name="referencia_correo"/);
  const valores = { categoria: "Aux", centro: "Res", fecha_inicio: "2026-10-01", descripcion: "Des", numero_plazas: "2" };
  globalThis.FormData = class { get(clave) { return valores[clave] ?? ""; } };
  const formulario = { closest: () => formulario, reportValidity: () => true };
  superficie.manejarSubmit({ target: formulario, preventDefault() {} });
  assert.equal(publicaciones.length, 0);
  assert.match(superficie.renderizar(), /Confirmar y publicar oferta/);
  superficie.manejarClick(boton({ ofertasAccion: "confirmar-publicacion" })); await turno();
  assert.equal(publicaciones.length, 1);
  assert.equal(publicaciones[0][3], null);
  assert.equal(publicaciones[0][4], "clave-publicacion");
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


test("la adjudicación telemática confirma la aceptación previa sin pedir otra respuesta", async () => {
  let adjudicada = false;
  const pendiente = oferta({ confirmacion_adjudicacion: "aceptacion_previa", estado: "pendiente_resolucion",
    plazas: [plaza(1, { propuesta: { tipo: "adjudicar", participacion_ref: "p:2", orden_vigente: 2 } })] });
  const cubierta = oferta({ confirmacion_adjudicacion: "aceptacion_previa", estado: "adjudicada",
    plazas: [plaza(1, { estado: "cubierta", secuencia: 1, participacion_ref: "p:2", orden_vigente: 2 })] });
  const anuncios = [];
  const cliente = { consultar: async () => sobre([adjudicada ? cubierta : pendiente]),
    registrarActo: async () => { adjudicada = true; return { ok: true, oferta: cubierta }; } };
  const s = crearSuperficieOfertasBolsa({ cliente, anunciar: (m) => anuncios.push(m) });
  s.activar("bolsa:1"); await turno();
  s.manejarClick(boton({ ofertasAccion: "preparar", ofertaRef: "oferta:1", plaza: "1", tipo: "adjudicada", secuencia: "0", participacionRef: "p:2", orden: "2" }));
  assert.match(s.renderizar(), /que ya aceptó en plazo/u);
  assert.doesNotMatch(s.renderizar(), /Tendrá que responder dentro de plazo/u);
  s.manejarClick(boton({ ofertasAccion: "confirmar" })); await turno();
  assert.deepEqual(anuncios, ["Plaza 1 adjudicada al n.º de orden 2, que aceptó en plazo."]);
  assert.doesNotMatch(s.renderizar(), /data-tipo="(?:aceptada|renuncia|sin_respuesta)"/u);
});
