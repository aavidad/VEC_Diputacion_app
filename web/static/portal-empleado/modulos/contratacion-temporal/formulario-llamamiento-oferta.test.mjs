import assert from "node:assert/strict";
import test from "node:test";
import { crearLlamamientoClienteHTTP, RUTAS_LLAMAMIENTO } from "./cliente-http-llamamiento.js";
import { ESQUEMA_RECIBO_SELECCION_OFERTA, ESQUEMA_RECIBO_DESCARTE_OFERTA,
  validarSolicitudRecuperacionOferta, validarReciboSeleccion, validarReciboSeleccionOferta,
  validarReciboDescarteOferta, validarEstadoSeleccionOferta } from "./contrato-llamamiento.js";
import { EXPEDIENTE, seleccion, raizPrueba, montar as montarBase, recibo as reciboSingular } from "./formulario-llamamiento-pruebas.js?v=VEC_WEB_VERSION";

const HUELLA = "a".repeat(64);
const reciboOferta = Object.freeze({
  esquema: ESQUEMA_RECIBO_SELECCION_OFERTA, estado: "confirmado",
  organizacion_ref: "organizacion:sintetica:001", expediente_ref: EXPEDIENTE,
  bolsa_ref: "bolsa:sintetica:001", oferta_ref: `oferta:${HUELLA}`,
  recibo_oferta_ref: `recibo:oferta:${HUELLA}`, recibo_vinculo_ref: "recibo:vinculo:001",
  numero_plazas: 2, publicada_en: "2026-10-15T10:00:00Z", vence_antes_de: "2026-10-17T10:00:00Z",
});
const reciboDescarte = Object.freeze({ esquema: ESQUEMA_RECIBO_DESCARTE_OFERTA, estado: "descartada",
  organizacion_ref: "organizacion:sintetica:001", expediente_ref: EXPEDIENTE,
  recibo_cierre_bolsa_ref: "recibo:cierre:001", recibo_descarte_ref: "recibo:descarte:001",
  descartada_en: "2026-10-15T10:05:00Z" });
const errorOfertaIndeterminada = () => Object.assign(new Error("resultado pendiente"), {
  estado: 503, codigo: "seleccion_oferta_indeterminada", envelopeValido: true, resultadoIndeterminado: true,
});
const click = (raiz, atributo) => {
  const selector = `[${atributo}]`;
  const boton = { dataset: { [atributo.replace(/^data-/, "").replace(/-([a-z])/g, (_, letra) => letra.toUpperCase())]: "" },
    closest: (consulta) => consulta === selector ? boton : null };
  return raiz.eventos.get("click")({ target: boton, preventDefault() {} });
};
const asentarse = () => new Promise((resolver) => setImmediate(resolver));
const montar = (raiz, cliente, extras = {}) => {
  const enviar = raiz.enviar.bind(raiz);
  raiz.enviar = (...argumentos) => asentarse().then(() => enviar(...argumentos));
  return montarBase(raiz, { consultarEstadoSeleccionOferta: async () => estadoOferta("ausente"), ...cliente }, {
    contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6, numero_personas: 2 }, ...extras,
  });
};
const estadoOferta = (estado, extras = {}) => ({ esquema: "vec.contratacion-temporal.estado-seleccion-oferta.v1",
  estado, expediente_ref: EXPEDIENTE,
  ...(estado === "ausente" ? {} : { version_esperada: 6, clave_idempotencia: seleccion().clave_idempotencia }),
  ...extras });
const intencionSingular = (extras = {}) => estadoOferta("pendiente", {
  tipo_seleccion: "llamamiento", ...extras,
});

test("la intención singular se reanuda tras remontar sin crear clave ni POST automático", async () => {
  const escritas = [], originales = [];
  const cliente = {
    consultarEstadoSeleccionOferta: async () => intencionSingular({ efecto: "preparar_orden", ventana_abierta: true }),
    seleccionarLlamamiento: async (solicitud) => { escritas.push(solicitud); return reciboSingular; },
    recuperarOferta: async () => { throw new Error("la recuperación de oferta no corresponde"); },
  };
  for (const versionContexto of [6, 9]) {
    const raiz = raizPrueba();
    const cerrar = montar(raiz, cliente, { contexto: { expediente_ref: EXPEDIENTE,
      version_esperada: versionContexto, numero_personas: 1 },
    generarClaveIdempotencia: () => { throw new Error("no se debe generar otra clave"); } });
    await asentarse();
    assert.equal(escritas.length, originales.length);
    assert.match(raiz.innerHTML, /Reanudar llamamiento/u);
    assert.doesNotMatch(raiz.innerHTML, /123e4567-e89b-42d3-a456-426614174000|Recuperar operación/u);
    await raiz.enviar("seleccion", { ...seleccion(), clave_idempotencia: "clave:falsa", version_esperada: "99" });
    originales.push(escritas.at(-1));
    assert.deepEqual(escritas.at(-1), { expediente_ref: EXPEDIENTE, version_esperada: 6,
      clave_idempotencia: seleccion().clave_idempotencia });
    assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="seleccion"/u);
    cerrar();
  }
});

test("la reserva singular sin efecto ni ventana también reanuda el POST original", async () => {
  const raiz = raizPrueba(), escritas = [];
  const lectura = intencionSingular();
  assert.deepEqual(validarEstadoSeleccionOferta(lectura, EXPEDIENTE), lectura);
  const cerrar = montar(raiz, {
    consultarEstadoSeleccionOferta: async () => lectura,
    seleccionarLlamamiento: async (solicitud) => { escritas.push(solicitud); return reciboSingular; },
  }, { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 9, numero_personas: 1 },
    generarClaveIdempotencia: () => { throw new Error("no debe generarse otra clave"); } });
  await asentarse();
  assert.equal(escritas.length, 0);
  assert.match(raiz.innerHTML, /Reanudar llamamiento/u);
  await raiz.enviar("seleccion", seleccion());
  assert.deepEqual(escritas, [{ expediente_ref: EXPEDIENTE, version_esperada: 6,
    clave_idempotencia: seleccion().clave_idempotencia }]);
  cerrar();
});

test("GET singular primero espera a comunicaciones; error permite recarga y denegación explica el bloqueo", async () => {
  for (const fallo of ["error", "denegado"]) {
    const raiz = raizPrueba(); let resolverLista, consultas = 0, post = 0;
    const cerrar = montar(raiz, {
      consultarEstadoSeleccionOferta: async () => intencionSingular(),
      consultarComunicacionesExpediente: async () => {
        consultas++;
        if (consultas === 1) return new Promise((resolve) => { resolverLista = resolve; });
        return { expediente_ref: EXPEDIENTE, comunicaciones: [] };
      },
      seleccionarLlamamiento: async () => { post++; return reciboSingular; },
    }, { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6, numero_personas: 1 } });
    await asentarse();
    assert.match(raiz.innerHTML, /Cargando los llamamientos/u);
    assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="seleccion"/u);
    await raiz.enviar("seleccion", seleccion());
    assert.equal(post, 0);
    resolverLista(fallo === "denegado"
      ? Promise.reject(Object.assign(new Error("denegado"), { estado: 403, envelopeValido: true }))
      : Promise.reject(new Error("sin conexión")));
    await asentarse();
    assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="seleccion"/u);
    if (fallo === "denegado") {
      assert.match(raiz.innerHTML, /No tiene permiso para ver estos llamamientos/u);
      assert.doesNotMatch(raiz.innerHTML, /data-ct-comunicaciones-reintentar/u);
    } else {
      assert.match(raiz.innerHTML, /data-ct-comunicaciones-reintentar/u);
      await click(raiz, "data-ct-comunicaciones-reintentar");
      await asentarse();
      assert.match(raiz.innerHTML, /data-ct-llamamiento-form="seleccion"/u);
      assert.match(raiz.innerHTML, /Reanudar llamamiento/u);
    }
    cerrar();
  }
});

test("un llamamiento anterior elegido no deja visible Reanudar llamamiento", async () => {
  const raiz = raizPrueba(); let post = 0;
  const fila = { organizacion_ref: "organizacion:sintetica:001", expediente_ref: EXPEDIENTE,
    llamamiento_ref: "llamamiento:anterior:001", comunicacion_ref: "comunicacion:anterior:001",
    version: 2, estado: "registrada_localmente", registrada_en: "2026-10-15T09:00:00Z",
    recibo_comunicacion_ref: "recibo:comunicacion:anterior:001",
    antecedente_tipo: "seleccion_confirmada", recibo_antecedente_ref: "recibo:seleccion:anterior:001",
    estado_respuesta: "sin_respuesta" };
  const cerrar = montar(raiz, {
    consultarEstadoSeleccionOferta: async () => intencionSingular(),
    consultarComunicacionesExpediente: async () => ({ expediente_ref: EXPEDIENTE, comunicaciones: [fila] }),
    consultarReciboRespuesta: async () => { throw Object.assign(new Error("ausente"), {
      estado: 404, codigo: "recurso_no_encontrado", envelopeValido: true,
    }); },
    seleccionarLlamamiento: async () => { post++; return reciboSingular; },
  }, { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6, numero_personas: 1 } });
  await asentarse();
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="seleccion"/u);
  await click(raiz, "data-ct-comunicacion-indice");
  await asentarse();
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="seleccion"|Reanudar llamamiento/u);
  await raiz.enviar("seleccion", seleccion());
  assert.equal(post, 0);
  cerrar();
});

test("la intención singular congelada conserva clave tras error y doble pulsación", async () => {
  const raiz = raizPrueba(), escritas = []; let liberar;
  const cerrar = montar(raiz, {
    consultarEstadoSeleccionOferta: async () => intencionSingular(),
    seleccionarLlamamiento: (solicitud) => {
      escritas.push(solicitud);
      if (escritas.length === 1) return Promise.reject(new Error("sin respuesta"));
      return new Promise((resolve) => { liberar = resolve; });
    },
  }, { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 8, numero_personas: 1 },
    generarClaveIdempotencia: () => { throw new Error("no se debe generar otra clave"); } });
  await asentarse();
  await raiz.enviar("seleccion", seleccion());
  assert.match(raiz.innerHTML, /Volver a intentar/u);
  const vuelo = raiz.enviar("seleccion", { ...seleccion(), version_esperada: "99" });
  await asentarse();
  await raiz.enviar("seleccion", seleccion());
  assert.equal(escritas.length, 2);
  assert.deepEqual(escritas[0], escritas[1]);
  liberar(reciboSingular);
  await vuelo;
  cerrar();
});

test("el estado singular rechaza formas ambiguas y no habilita otro POST", async () => {
  const invalidas = [
    intencionSingular({ efecto: "preparar_orden" }),
    intencionSingular({ ventana_abierta: true }),
    intencionSingular({ efecto: "otro", ventana_abierta: true }),
    intencionSingular({ tipo_seleccion: "otro" }),
    estadoOferta("confirmada", { tipo_seleccion: "llamamiento", oferta: reciboOferta }),
    intencionSingular({ version_esperada: 0 }),
  ];
  for (const lectura of invalidas) {
    assert.throws(() => validarEstadoSeleccionOferta(lectura, EXPEDIENTE));
    const raiz = raizPrueba(); let post = 0;
    const cerrar = montar(raiz, { consultarEstadoSeleccionOferta: async () => lectura,
      seleccionarLlamamiento: async () => { post++; return reciboSingular; } },
    { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6, numero_personas: 1 } });
    await raiz.enviar("seleccion", seleccion());
    assert.equal(post, 0);
    cerrar();
  }
});

test("GET de estado usa consulta codificada y valida la identidad de la respuesta", async () => {
  const comandos = [];
  const cliente = crearLlamamientoClienteHTTP({ validarOpciones: () => ({ signal: undefined }),
    ejecutar: async (comando) => { comandos.push(comando); return comando.validarRespuesta(estadoOferta("pendiente")); } });
  assert.equal((await cliente.consultarEstadoSeleccionOferta(EXPEDIENTE)).estado, "pendiente");
  assert.equal(comandos[0].metodo, "GET");
  assert.equal(comandos[0].efecto, false);
  assert.equal(comandos[0].ruta, `${RUTAS_LLAMAMIENTO.estadoSeleccionOferta}?expediente_ref=expediente%3Act%3Asintetico%3A001`);
  assert.throws(() => validarEstadoSeleccionOferta(estadoOferta("pendiente", { expediente_ref: "expediente:otro:001" }), EXPEDIENTE));
  assert.throws(() => validarEstadoSeleccionOferta(estadoOferta("ausente", { clave_idempotencia: seleccion().clave_idempotencia }), EXPEDIENTE));
});

test("recarga restaura la UUID pendiente sin POST y recupera solo al pulsar", async () => {
  const raiz = raizPrueba();
  const post = [], recuperaciones = [];
  const cerrar = montar(raiz, { consultarEstadoSeleccionOferta: async () => estadoOferta("pendiente", { ventana_abierta: true }),
    seleccionarLlamamiento: async (solicitud) => { post.push(solicitud); return reciboOferta; },
    recuperarOferta: async (solicitud) => { recuperaciones.push(solicitud); return reciboOferta; } },
  { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6, numero_personas: 2 },
    generarClaveIdempotencia: () => { throw new Error("UUID nueva prohibida"); } });
  await asentarse();
  assert.deepEqual(post, []);
  assert.deepEqual(recuperaciones, []);
  assert.match(raiz.innerHTML, /Recuperar operación/);
  assert.doesNotMatch(raiz.innerHTML, /type="submit"[\s\S]*?>Volver a intentar<\/button>/);
  assert.doesNotMatch(raiz.innerHTML, /123e4567-e89b-42d3-a456-426614174000/);
  await raiz.enviar("seleccion", seleccion());
  assert.deepEqual(post, []);
  await click(raiz, "data-ct-llamamiento-recuperar-oferta");
  assert.deepEqual(recuperaciones, [{ expediente_ref: EXPEDIENTE, version_esperada: 6,
    clave_idempotencia: seleccion().clave_idempotencia }]);
  assert.deepEqual(post, []);
  assert.match(raiz.innerHTML, /Oferta publicada y vinculada/);
  cerrar();
});

test("GET fallido bloquea la selección; reintento ausente la vuelve a permitir", async () => {
  const raiz = raizPrueba();
  let consultas = 0, post = 0;
  const cerrar = montar(raiz, { consultarEstadoSeleccionOferta: async () => {
    if (++consultas === 1) throw Object.assign(new Error("sin conexión"), { estado: 503 });
    return estadoOferta("ausente");
  }, seleccionarLlamamiento: async () => { post++; return reciboOferta; } },
  { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6, numero_personas: 2 } });
  await asentarse();
  assert.match(raiz.innerHTML, /data-ct-llamamiento-estado-oferta-reintentar/);
  await raiz.enviar("seleccion", seleccion());
  assert.equal(post, 0);
  await click(raiz, "data-ct-llamamiento-estado-oferta-reintentar");
  assert.equal(consultas, 2);
  cerrar();
});

test("N singular con GET fallido bloquea el envío", async () => {
  const raiz = raizPrueba();
  let consultas = 0, post = 0;
  const cerrar = montar(raiz, { consultarEstadoSeleccionOferta: async () => { consultas++; throw new Error("S2 sin configurar"); },
    seleccionarLlamamiento: async () => { post++; return reciboSingular; } },
  { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6, numero_personas: 1 } });
  await asentarse();
  assert.equal(consultas, 1);
  await raiz.enviar("seleccion", seleccion());
  assert.equal(post, 0);
  cerrar();
});

test("N plural con S2 sin configurar bloquea POST hasta consulta válida", async () => {
  const raiz = raizPrueba();
  let consultas = 0, post = 0;
  const cerrar = montar(raiz, { consultarEstadoSeleccionOferta: async () => { consultas++; throw Object.assign(new Error("S2 sin configurar"), { estado: 503 }); },
    seleccionarLlamamiento: async () => { post++; return reciboOferta; } });
  await asentarse();
  await raiz.enviar("seleccion", seleccion());
  assert.equal(consultas, 1);
  assert.equal(post, 0);
  assert.match(raiz.innerHTML, /Reintentar consulta/);
  cerrar();
});

test("sin método GET bloquea selección singular y plural", async () => {
  for (const numeroPersonas of [1, 2]) {
    const raiz = raizPrueba(); let post = 0;
    const cerrar = montar(raiz, { consultarEstadoSeleccionOferta: null,
      seleccionarLlamamiento: async () => { post++; return numeroPersonas === 1 ? reciboSingular : reciboOferta; } },
      { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6, numero_personas: numeroPersonas } });
    await raiz.enviar("seleccion", seleccion());
    assert.equal(post, 0);
    assert.match(raiz.innerHTML, /No se ha podido preparar el envío/);
    cerrar();
  }
});

test("CT140 caído deja recuperar intención plural sin habilitar oferta nueva", async () => {
  for (const situacion of ["pendiente", "ausente", "confirmada", "descartada"]) {
    const raiz = raizPrueba(), recuperadas = []; let post = 0;
    const lectura = estadoOferta(situacion, situacion === "confirmada" ? { oferta: reciboOferta }
      : situacion === "descartada" ? { descarte: reciboDescarte } : {});
    const cerrar = montar(raiz, { consultarEstadoSeleccionOferta: async () => lectura,
      consultarComunicacionesExpediente: async () => { throw new Error("CT140 no disponible"); },
      seleccionarLlamamiento: async () => { post++; return reciboOferta; },
      recuperarOferta: async (solicitud) => { recuperadas.push(solicitud); return reciboOferta; } });
    await asentarse();
    if (situacion === "pendiente") {
      assert.match(raiz.innerHTML, /data-ct-llamamiento-recuperar-oferta/);
      await click(raiz, "data-ct-llamamiento-recuperar-oferta");
      assert.deepEqual(recuperadas, [{ expediente_ref: EXPEDIENTE, version_esperada: 6,
        clave_idempotencia: seleccion().clave_idempotencia }]);
      assert.equal(post, 0);
    } else if (situacion === "ausente") {
      assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="seleccion"/);
      await raiz.enviar("seleccion", seleccion());
      assert.equal(post, 0);
    } else if (situacion === "confirmada") {
      assert.match(raiz.innerHTML, /Oferta publicada y vinculada/);
      assert.equal(post, 0);
    } else {
      assert.match(raiz.innerHTML, /recibo:descarte:001/);
      assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-nueva-oferta/);
      await click(raiz, "data-ct-llamamiento-nueva-oferta");
      assert.equal(post, 0);
    }
    cerrar();
  }
});

test("GET terminal devuelve recibo sin POST y respuesta tardía no cruza expedientes", async () => {
  const raiz = raizPrueba();
  let resolver, post = 0;
  const cerrar = montar(raiz, { consultarEstadoSeleccionOferta: () => new Promise((r) => { resolver = r; }),
    seleccionarLlamamiento: async () => { post++; return reciboOferta; } },
  { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6, numero_personas: 2 } });
  await asentarse();
  assert.equal(cerrar.actualizarContexto({ expediente_ref: "expediente:otro:001", version_esperada: 6 }), false);
  resolver(estadoOferta("confirmada", { oferta: reciboOferta }));
  await asentarse();
  assert.equal(post, 0);
  assert.doesNotMatch(raiz.innerHTML, /Oferta publicada y vinculada/);

  const confirmada = raizPrueba();
  const cerrarConfirmada = montar(confirmada, { consultarEstadoSeleccionOferta: async () => estadoOferta("confirmada", { oferta: reciboOferta }),
    seleccionarLlamamiento: async () => { post++; return reciboOferta; } },
  { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6, numero_personas: 2 } });
  await asentarse();
  assert.match(confirmada.innerHTML, /Oferta publicada y vinculada/);
  assert.equal(post, 0);
  cerrarConfirmada();

  const otra = raizPrueba();
  const cerrarOtra = montar(otra, { consultarEstadoSeleccionOferta: async () => estadoOferta("descartada", { descarte: reciboDescarte }),
    seleccionarLlamamiento: async () => { post++; return reciboOferta; } },
  { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6, numero_personas: 2 } });
  await asentarse();
  assert.match(otra.innerHTML, /recibo:descarte:001/);
  assert.match(otra.innerHTML, /Preparar nueva oferta/);
  assert.equal(post, 0);
  cerrarOtra();
});

test("el mismo POST acepta recibo de oferta vinculada y rechaza un recibo ajeno", async () => {
  const solicitud = { ...seleccion(), version_esperada: 6 };
  const comandos = [];
  const cliente = crearLlamamientoClienteHTTP({ validarOpciones: () => ({ signal: undefined }),
    ejecutar: async (comando) => { comandos.push(comando); return comando.validarRespuesta(reciboOferta); } });
  assert.deepEqual(await cliente.seleccionarLlamamiento(solicitud), reciboOferta);
  assert.equal(comandos[0].ruta, RUTAS_LLAMAMIENTO.seleccionLlamamiento);
  assert.deepEqual(Object.keys(comandos[0].entrada), ["expediente_ref", "version_esperada", "clave_idempotencia"]);
  assert.equal(validarReciboSeleccion(reciboSingular).esquema, "vec.contratacion-temporal.recibo-seleccion-llamamiento.v1");
  assert.throws(() => validarReciboSeleccionOferta({ ...reciboOferta, expediente_ref: "expediente:otro:001" }, solicitud));
  assert.throws(() => validarReciboSeleccionOferta({ ...reciboOferta, numero_plazas: 1 }, solicitud));
  assert.throws(() => validarReciboSeleccionOferta({ ...reciboOferta, aviso_enviado: true }, solicitud));
});

test("la oferta real muestra plazas, plazo y justificantes sin abrir comunicación singular", async () => {
  const raiz = raizPrueba();
  const solicitudes = [];
  const cerrar = montar(raiz, { seleccionarLlamamiento: async (solicitud) => {
    solicitudes.push(solicitud); return reciboOferta;
  } });
  await asentarse();
  assert.match(raiz.innerHTML, /Publicar oferta de varias plazas/);
  await raiz.enviar("seleccion", seleccion());
  assert.equal(solicitudes.length, 1);
  assert.deepEqual(Object.keys(solicitudes[0]), ["expediente_ref", "version_esperada", "clave_idempotencia"]);
  assert.match(raiz.innerHTML, /Oferta publicada y vinculada al expediente/);
  assert.match(raiz.innerHTML, /Plazas publicadas<\/dt><dd>2/);
  assert.match(raiz.innerHTML, /recibo:oferta:/);
  assert.match(raiz.innerHTML, /recibo:vinculo:001/);
  assert.match(raiz.innerHTML, /ofertas_bolsa_ref=bolsa%3Asintetica%3A001#bolsa\/llamamientos/);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-comunicacion|data-ct-llamamiento-form="comunicacion"/);
  assert.match(raiz.innerHTML, /no acredita el envío de un correo/);
  cerrar();
});

test("tras fallo incierto recupera la oferta con la misma petición y el singular sigue abriendo comunicación", async () => {
  const raiz = raizPrueba();
  const solicitudes = [];
  const cerrar = montar(raiz, { seleccionarLlamamiento: async (solicitud) => {
    solicitudes.push(solicitud);
    if (solicitudes.length === 1) throw new Error("red interrumpida");
    return reciboOferta;
  } });
  await raiz.enviar("seleccion", seleccion());
  assert.match(raiz.innerHTML, /Volver a intentar/);
  await raiz.enviar("seleccion", { ...seleccion(), expediente_ref: "expediente:otro:001" });
  assert.deepEqual(solicitudes[0], solicitudes[1]);
  assert.match(raiz.innerHTML, /Oferta publicada y vinculada/);
  cerrar();

  const raizSingular = raizPrueba();
  const cerrarSingular = montar(raizSingular, { seleccionarLlamamiento: async () => reciboSingular });
  await raizSingular.enviar("seleccion", seleccion());
  assert.match(raizSingular.innerHTML, /data-ct-llamamiento-comunicacion open/);
  cerrarSingular();
});

test("503, 403 y 403 tras un intento incierto conservan cuerpo y UUID hasta recibo válido", async () => {
  for (const reciboFinal of [reciboOferta, reciboSingular]) {
    const raiz = raizPrueba();
    const solicitudes = [];
    const cerrar = montar(raiz, { seleccionarLlamamiento: async (solicitud) => {
      solicitudes.push(solicitud);
      if (solicitudes.length === 1) throw new Error("503 sin respuesta concluyente");
      if (solicitudes.length < 4) throw Object.assign(new Error("denegado"), {
        estado: 403, codigo: "acceso_denegado", envelopeValido: true, resultadoIndeterminado: false,
      });
      return reciboFinal;
    } });
    for (let intento = 0; intento < 4; intento++) {
      await raiz.enviar("seleccion", { ...seleccion(), expediente_ref: intento ? "expediente:otro:001" : EXPEDIENTE });
      if (intento < 3) assert.match(raiz.innerHTML, /Volver a intentar/u);
    }
    assert.equal(solicitudes.length, 4);
    solicitudes.forEach((solicitud) => assert.deepEqual(solicitud, solicitudes[0]));
    assert.equal(solicitudes[0].expediente_ref, EXPEDIENTE);
    assert.match(raiz.innerHTML, reciboFinal === reciboOferta ? /Oferta publicada y vinculada/ : /data-ct-llamamiento-comunicacion open/u);
    cerrar();
  }
});

test("401, 422 y 409 tras un intento incierto no liberan la selección original", async () => {
  const raiz = raizPrueba();
  const solicitudes = [];
  const rechazos = [[401, "autenticacion_requerida"], [422, "contenido_no_valido"], [409, "conflicto_no_reintentable"]];
  const cerrar = montar(raiz, { seleccionarLlamamiento: async (solicitud) => {
    solicitudes.push(solicitud);
    if (solicitudes.length === 1) throw new Error("red interrumpida");
    if (solicitudes.length <= 4) {
      const [estado, codigo] = rechazos[solicitudes.length - 2];
      throw Object.assign(new Error(codigo), { estado, codigo, envelopeValido: true, resultadoIndeterminado: false });
    }
    return reciboOferta;
  } });
  for (let intento = 0; intento < 5; intento++) {
    await raiz.enviar("seleccion", { ...seleccion(), expediente_ref: intento ? "expediente:otro:001" : EXPEDIENTE });
    if (intento < 4) assert.match(raiz.innerHTML, /Volver a intentar/u);
  }
  assert.equal(solicitudes.length, 5);
  solicitudes.forEach((solicitud) => assert.deepEqual(solicitud, solicitudes[0]));
  assert.match(raiz.innerHTML, /Oferta publicada y vinculada/);
  cerrar();
});

test("recuperar añade acción al final sin mutar el comando normal y valida ambos terminales", async () => {
  const original = { ...seleccion(), version_esperada: 6 };
  const antes = JSON.stringify(original);
  const derivada = validarSolicitudRecuperacionOferta(original);
  assert.deepEqual(Object.keys(derivada), ["expediente_ref", "version_esperada", "clave_idempotencia", "accion"]);
  assert.equal(derivada.accion, "recuperar_oferta");
  assert.equal(JSON.stringify(original), antes);
  const comandos = [];
  const cliente = crearLlamamientoClienteHTTP({ validarOpciones: () => ({ signal: undefined }),
    ejecutar: async (comando) => { comandos.push(comando); return comando.validarRespuesta(reciboDescarte); } });
  assert.deepEqual(await cliente.recuperarOferta(original), reciboDescarte);
  assert.equal(comandos[0].ruta, RUTAS_LLAMAMIENTO.seleccionLlamamiento);
  assert.equal(JSON.stringify(comandos[0].entrada), `${antes.slice(0, -1)},"accion":"recuperar_oferta"}`);
  assert.deepEqual(validarReciboDescarteOferta(reciboDescarte, original), reciboDescarte);
  assert.throws(() => validarReciboDescarteOferta({ ...reciboDescarte, expediente_ref: "expediente:otro:001" }, original));
  assert.throws(() => validarReciboDescarteOferta({ ...reciboDescarte, oferta_ref: reciboOferta.oferta_ref }, original));
});

test("recuperación puede cerrar sin publicar; solo tras decisión nueva se crea otra UUID", async () => {
  const raiz = raizPrueba();
  const publicadas = [], recuperadas = [];
  let confirmada = true, clavesNuevas = 0;
  const cerrar = montar(raiz, {
    seleccionarLlamamiento: async (solicitud) => {
      publicadas.push(solicitud);
      if (publicadas.length === 1) throw errorOfertaIndeterminada();
      return reciboOferta;
    },
    recuperarOferta: async (solicitud) => { recuperadas.push(solicitud); return reciboDescarte; },
  }, { confirmarOperacion: () => confirmada,
    generarClaveIdempotencia: () => { clavesNuevas++; return clavesNuevas === 1
      ? "123e4567-e89b-42d3-a456-426614174099" : "123e4567-e89b-42d3-a456-426614174098"; } });
  await raiz.enviar("seleccion", seleccion());
  assert.match(raiz.innerHTML, /Recuperar operación/);
  await click(raiz, "data-ct-llamamiento-recuperar-oferta");
  assert.deepEqual(recuperadas, publicadas);
  assert.match(raiz.innerHTML, /Intento cerrado sin publicar oferta/);
  assert.match(raiz.innerHTML, /recibo:cierre:001/);
  assert.match(raiz.innerHTML, /recibo:descarte:001/);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="comunicacion"|data-ct-llamamiento-recuperar-oferta/);
  const clavesAntes = clavesNuevas;
  confirmada = false;
  await click(raiz, "data-ct-llamamiento-nueva-oferta");
  assert.match(raiz.innerHTML, /Preparar nueva oferta/);
  assert.match(raiz.innerHTML, /recibo:descarte:001/);
  assert.equal(clavesNuevas, clavesAntes);
  confirmada = true;
  await click(raiz, "data-ct-llamamiento-nueva-oferta");
  assert.equal(clavesNuevas, clavesAntes);
  assert.match(raiz.innerHTML, /recibo:cierre:001/);
  assert.match(raiz.innerHTML, /recibo:descarte:001/);
  confirmada = false;
  await raiz.enviar("seleccion", { expediente_ref: EXPEDIENTE, version_esperada: "6" });
  assert.equal(clavesNuevas, clavesAntes);
  assert.equal(publicadas.length, 1);
  assert.match(raiz.innerHTML, /recibo:cierre:001/);
  assert.match(raiz.innerHTML, /recibo:descarte:001/);
  confirmada = true;
  await raiz.enviar("seleccion", { expediente_ref: EXPEDIENTE, version_esperada: "6" });
  assert.equal(clavesNuevas, clavesAntes + 1);
  assert.equal(publicadas.length, 2);
  assert.notEqual(publicadas[1].clave_idempotencia, publicadas[0].clave_idempotencia);
  assert.match(raiz.innerHTML, /Oferta publicada y vinculada/);
  assert.match(raiz.innerHTML, /recibo:cierre:001/);
  assert.match(raiz.innerHTML, /recibo:descarte:001/);
  cerrar();
});

test("respuesta perdida al cerrar reintenta recuperación y sin reserva conserva publicación original", async () => {
  const raiz = raizPrueba();
  const intentos = [];
  let recuperaciones = 0;
  const cerrar = montar(raiz, { seleccionarLlamamiento: async (solicitud) => {
    intentos.push(solicitud);
    throw errorOfertaIndeterminada();
  }, recuperarOferta: async (solicitud) => {
    recuperaciones++;
    if (recuperaciones === 1) throw new Error("cierre sin respuesta");
    if (recuperaciones === 2) throw Object.assign(new Error("sin reserva"), { estado: 409, codigo: "seleccion_oferta_sin_reserva" });
    return reciboDescarte;
  } });
  await raiz.enviar("seleccion", seleccion());
  await click(raiz, "data-ct-llamamiento-recuperar-oferta");
  assert.match(raiz.innerHTML, /Reintente la recuperación/);
  await click(raiz, "data-ct-llamamiento-recuperar-oferta");
  assert.match(raiz.innerHTML, /Repita la solicitud anterior sin cambiarla/);
  assert.match(raiz.innerHTML, /Volver a intentar/);
  await raiz.enviar("seleccion", { ...seleccion(), expediente_ref: "expediente:otro:001" });
  assert.deepEqual(intentos[0], intentos[1]);
  await click(raiz, "data-ct-llamamiento-recuperar-oferta");
  assert.match(raiz.innerHTML, /Intento cerrado sin publicar oferta/);
  cerrar();
});

test("recuperar oferta publicada tras 403 conserva la clave original y omite comunicación", async () => {
  const raiz = raizPrueba();
  let claves = 0, llamadasRecuperacion = 0;
  const originales = [], recuperadas = [];
  const cerrar = montar(raiz, {
    seleccionarLlamamiento: async (solicitud) => { originales.push(solicitud); throw errorOfertaIndeterminada(); },
    recuperarOferta: async (solicitud) => {
      recuperadas.push(solicitud);
      if (++llamadasRecuperacion === 1) throw Object.assign(new Error("denegado"), {
        estado: 403, codigo: "acceso_denegado", envelopeValido: true, resultadoIndeterminado: false,
      });
      return reciboOferta;
    },
  }, { generarClaveIdempotencia: () => { claves++; return "123e4567-e89b-42d3-a456-426614174099"; } });
  await raiz.enviar("seleccion", seleccion());
  await click(raiz, "data-ct-llamamiento-recuperar-oferta");
  assert.match(raiz.innerHTML, /Recuperar operación/);
  await click(raiz, "data-ct-llamamiento-recuperar-oferta");
  assert.deepEqual(recuperadas, [originales[0], originales[0]]);
  assert.equal(claves, 1);
  assert.match(raiz.innerHTML, /Oferta publicada y vinculada/);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="comunicacion"/);
  cerrar();
});

test("sin cantidad se bloquea el envío y N singular no ofrece cierre de oferta", async () => {
  for (const [numeroPersonas, error] of [[undefined, new Error("red sin respuesta")], [1, errorOfertaIndeterminada()]]) {
    const raiz = raizPrueba();
    let recuperaciones = 0, selecciones = 0;
    const cerrar = montar(raiz, {
      seleccionarLlamamiento: async () => { selecciones++; throw error; },
      recuperarOferta: async () => { recuperaciones++; return reciboDescarte; },
    }, { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6,
      ...(numeroPersonas === undefined ? {} : { numero_personas: numeroPersonas }) } });
    await raiz.enviar("seleccion", seleccion());
    assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-recuperar-oferta/);
    await click(raiz, "data-ct-llamamiento-recuperar-oferta");
    assert.equal(recuperaciones, 0);
    if (numeroPersonas === undefined) {
      assert.equal(selecciones, 0);
      assert.match(raiz.innerHTML, /No se pudo consultar el número de personas solicitadas/);
    } else {
      assert.equal(selecciones, 1);
      assert.match(raiz.innerHTML, /Volver a intentar/);
    }
    cerrar();
  }
});

test("A descartada conserva recibos mientras B incierta recupera su propia UUID", async () => {
  const raiz = raizPrueba();
  const selecciones = [], recuperaciones = [];
  let claves = 0;
  const cerrar = montar(raiz, {
    seleccionarLlamamiento: async (solicitud) => { selecciones.push(solicitud); throw errorOfertaIndeterminada(); },
    recuperarOferta: async (solicitud) => {
      recuperaciones.push(solicitud);
      return recuperaciones.length === 1 ? reciboDescarte : reciboOferta;
    },
  }, { generarClaveIdempotencia: () => {
    claves++;
    return claves === 1 ? "123e4567-e89b-42d3-a456-426614174091" : "123e4567-e89b-42d3-a456-426614174092";
  } });
  await raiz.enviar("seleccion", seleccion());
  await click(raiz, "data-ct-llamamiento-recuperar-oferta");
  assert.match(raiz.innerHTML, /recibo:descarte:001/);
  await click(raiz, "data-ct-llamamiento-nueva-oferta");
  await raiz.enviar("seleccion", { expediente_ref: EXPEDIENTE, version_esperada: "6" });
  assert.notEqual(selecciones[0].clave_idempotencia, selecciones[1].clave_idempotencia);
  assert.match(raiz.innerHTML, /recibo:descarte:001/);
  assert.match(raiz.innerHTML, /Recuperar operación/);
  await click(raiz, "data-ct-llamamiento-recuperar-oferta");
  assert.deepEqual(recuperaciones, [selecciones[0], selecciones[1]]);
  assert.match(raiz.innerHTML, /Oferta publicada y vinculada/);
  assert.match(raiz.innerHTML, /recibo:descarte:001/);
  cerrar();
});
