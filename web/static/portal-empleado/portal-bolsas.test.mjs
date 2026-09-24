import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import {
  ESQUEMA_BOLSAS,
  ESQUEMA_CANDIDATOS,
  ESQUEMA_CONTACTOS,
  ESQUEMA_ESTADISTICAS,
  ESQUEMA_ACCION_BOLSA,
  SITUACIONES_PARTICIPACION_BOLSA,
  CANALES_LLAMAMIENTO,
  RESULTADOS_LLAMAMIENTO_BOLSA,
  RESULTADOS_REGISTRO_LLAMAMIENTO,
  validarDocumentoEnmascarado,
  extraerDatosEnvelopeCanonico,
  validarBolsa,
  validarRespuestaBolsas,
  validarCandidato,
  validarRespuestaCandidatosBolsa,
  validarContacto,
  validarRespuestaContactos,
  validarRespuestaEstadisticas,
  validarPayloadCrearLlamamiento,
  validarPayloadResultadoLlamamiento,
  construirEnvelopeAccionBolsa,
} from "./portal-bolsas-contrato.js";
import {
  consultarBolsas,
  consultarCandidatosBolsa,
  consultarContactosCandidato,
  consultarEstadisticasBolsa,
  registrarContactoCandidato,
  emitirLlamamiento,
  cambiarSituacionCandidato,
  crearLlamamientoCandidato,
  registrarResultadoLlamamiento,
  rutaCandidatosBolsa,
  seleccionarParticipacionesPorEstado,
  crearControladorBolsas,
} from "./portal-bolsas-api.js";
function comprobarTransporteInterno(opciones) {
  assert.equal(opciones.credentials, "same-origin");
  assert.equal(opciones.mode, "same-origin");
  assert.equal(opciones.cache, "no-store");
  assert.equal(opciones.redirect, "error");
}

test("P-WEB-10 selecciona todas las participaciones del filtro según el orden vigente B6", () => {
  const candidatas = [
    { participacion_ref: "participacion:03", estado_clave: "disponible", orden: 3 },
    { participacion_ref: "participacion:01", estado_clave: "disponible", orden: 1 },
    { participacion_ref: "participacion:02", estado_clave: "disponible_desde", orden: 2 },
    { participacion_ref: "participacion:pausada", estado_clave: "no_disponible", orden: 0 },
    { participacion_ref: "participacion:sin-turno", estado_clave: "disponible", orden: null },
  ];
  assert.deepEqual(
    seleccionarParticipacionesPorEstado(candidatas, ["disponible", "disponible_desde"]),
    ["participacion:01", "participacion:02", "participacion:03"],
  );
});
test("P-WEB-10 limita la selección masiva a las primeras cien por orden B6", () => {
  const candidatas = Array.from({ length: 105 }, (_, indice) => ({
    participacion_ref: `participacion:${String(105 - indice).padStart(3, "0")}`,
    estado_clave: "disponible",
    orden: 105 - indice,
  }));
  const seleccion = seleccionarParticipacionesPorEstado(candidatas, ["disponible"]);
  assert.equal(seleccion.length, 100);
  assert.deepEqual(seleccion.slice(0, 2), ["participacion:001", "participacion:002"]);
  assert.equal(seleccion.at(-1), "participacion:100");
});
test("B7 emite por la ruta exacta con idempotencia y conserva el recibo", async () => {
  let llamada;
  const huella = "a".repeat(64);
  const salida = await emitirLlamamiento({
    bolsa_ref: "bolsa:01",
    participaciones: ["participacion:01"],
    configuracion: { referencia: "NEC-01", descripcion: "Cobertura", categoria: "Auxiliar", centro: "Centro", modalidad: "Sustitución", fecha_inicio: "2026-10-01", plazo: "Hasta el 30 de septiembre, 14:00", plantilla_version: "bolsa-llamamiento-v1", asunto: "Llamamiento", cuerpo: "Texto" },
    clave_idempotencia: "b7-emision-0001",
  }, { fetchImpl: async (url, opciones) => {
    llamada = { url, opciones };
    return { ok: true, status: 201, json: async () => ({ data: { recibo_ref: `recibo:llamamiento:${huella}`, llamamiento_ref: `llamamiento:${huella}`, bolsa_ref: "bolsa:01", estado: "emitido_pendiente_respuesta", participaciones: ["participacion:01"], configuracion: { referencia: "NEC-01", descripcion: "Cobertura", categoria: "Auxiliar", centro: "Centro", modalidad: "Sustitución", fecha_inicio: "2026-10-01", plazo: "Hasta el 30 de septiembre, 14:00", plantilla_version: "bolsa-llamamiento-v1", asunto: "Llamamiento", cuerpo: "Texto" }, emitido_en: "2026-09-22T12:00:00Z", reutilizada: false } }) };
  }});
  assert.equal(salida.ok, true);
  assert.equal(salida.datos.recibo_ref, `recibo:llamamiento:${huella}`);
  assert.equal(llamada.url, "/api/vec/bolsa/llamamientos/emisiones");
  assert.equal(llamada.opciones.credentials, "same-origin");
  comprobarTransporteInterno(llamada.opciones);
  assert.equal(llamada.opciones.headers["Idempotency-Key"], "b7-emision-0001");
  assert.deepEqual(JSON.parse(llamada.opciones.body).participaciones, ["participacion:01"]);
});
test("P-WEB-08 valida y consulta las estadísticas agregadas de Bolsa", async () => {
  const envelope = { data: { esquema: ESQUEMA_ESTADISTICAS, generado_en: "2026-09-23T08:00:00Z", bolsas: { total: 2, vigentes: 1, sustituidas: 1 }, personas: { total: 3, por_estado: { disponible: 1, no_disponible: 0, trabajando: 1, pendiente_incorporacion: 0, renuncia: 0, excluido: 0, disponible_desde: 1 } }, llamamientos: { total: 2, por_canal: { correo: 2 }, por_resultado: { pendiente: 1, aceptado: 1 } }, por_bolsa: [{ bolsa_ref: "bolsa:01", categoria: "Auxiliar", tipo_lista: "ordinaria", vigente: true, total: 3, por_estado: { disponible: 1, no_disponible: 0, trabajando: 1, pendiente_incorporacion: 0, renuncia: 0, excluido: 0, disponible_desde: 1 } }] } };
  const validado = validarRespuestaEstadisticas(envelope);
  assert.equal(validado.personas.total, 3);
  let llamada = "";
  const resultado = await consultarEstadisticasBolsa({ fetchImpl: async (url, opciones) => { llamada = url; comprobarTransporteInterno(opciones); return { ok: true, status: 200, json: async () => envelope }; } });
  assert.equal(llamada, "/api/vec/bolsa/estadisticas");
  assert.equal(resultado.datos.por_bolsa[0].categoria, "Auxiliar");
});
test("cambiar situación B2 envía idempotencia y conserva el recibo", async () => {
  let observada;
  const resultado = await cambiarSituacionCandidato("bolsa:01", "participacion:01", { situacion: "no_disponible", motivo: "Pausa comunicada", fecha_disponible: null, clave_idempotencia: "b2-cambio-0001" }, { fetchImpl: async (url, opciones) => {
    observada = { url, opciones };
    return { ok: true, status: 201, json: async () => ({ data: { recibo_ref: "recibo:situacion:01", situacion: "no_disponible" } }) };
  }});
  assert.equal(resultado.ok, true);
  assert.equal(resultado.datos.recibo_ref, "recibo:situacion:01");
  assert.equal(observada.opciones.headers["Idempotency-Key"], "b2-cambio-0001");
  assert.equal(observada.opciones.credentials, "same-origin");
  comprobarTransporteInterno(observada.opciones);
  assert.match(observada.url, /\/bolsa:01\/candidatos\/participacion:01\/situacion$/);
});
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js";
import { crearFuenteLecturaBolsasPresentacion } from "./portal-presentacion-adaptador.js";
import { obtenerDatosPresentacion } from "./datos-presentacion.js";
const rutaDemoJson = new URL("../../../data/demo/bolsa/v1.bolsas-demo.json", import.meta.url);
const demoJsonRaw = JSON.parse(await readFile(rutaDemoJson, "utf8"));
/**
 * Función que mapea los estados sintéticos de demo al catálogo cerrado de SituacionParticipacionBolsa:
 * El fixture conserva el catálogo B2 sin agrupar situaciones.
 */
function mapearSituacion(estadoClave) { return estadoClave; }
function construirFixturesDesdeDemo() {
  const candidaturasPorBolsa = new Map();
  for (const c of demoJsonRaw.candidaturas) {
    if (!candidaturasPorBolsa.has(c.bolsa_ref)) {
      candidaturasPorBolsa.set(c.bolsa_ref, []);
    }
    candidaturasPorBolsa.get(c.bolsa_ref).push(c);
  }
  const bolsas = demoJsonRaw.bolsas.map((b) => {
    const candidaturas = candidaturasPorBolsa.get(b.bolsa_ref) || [];
    const porEstado = {
      disponible: 0,
      no_disponible: 0,
      trabajando: 0,
      pendiente_incorporacion: 0,
      renuncia: 0,
      excluido: 0,
      disponible_desde: 0,
    };
    for (const c of candidaturas) {
      const situacion = mapearSituacion(c.estado_clave);
      porEstado[situacion] += 1;
    }
    return {
      bolsa_ref: b.bolsa_ref.replace(":demo:", ":sintetico:"),
      categoria_clave: b.categoria_ref.replace(/^categoria:rpt:/, ""),
      categoria: b.categoria,
      tipo_lista: b.tipo_lista,
      vigente_desde: b.vigente_desde,
      vigente_hasta: b.vigente_hasta,
      total: candidaturas.length,
      por_estado: porEstado,
	  llamamientos_en_curso: 0,
	  politica_orden: { politica_ref: `politica:orden:${b.bolsa_ref}`, version: 1, criterio: "puntuacion_desc_acta", tipo_lista: "rotatoria", reposicion: "misma_posicion", provisional: true, rotulo: "Provisional, pendiente de RRHH (dudas 13–14)", actor: "sistema:prueba", vigente_desde: b.vigente_desde },
    };
  });
  const primeraBolsa = bolsas[0];
  const candidaturasPrimeraBolsa = candidaturasPorBolsa.get(demoJsonRaw.bolsas[0].bolsa_ref) || [];
  const candidatos = candidaturasPrimeraBolsa.map((c) => {
    const situacion = mapearSituacion(c.estado_clave);
    let ultimoLlamamiento = null;
    if (c.contactos_previos > 0) {
      ultimoLlamamiento = {
        llamamiento_ref: `llam_${c.candidatura_ref.replace(/[^a-zA-Z0-9]/g, "_")}`,
        comunicado_en: new Date(c.estado_desde).toISOString(),
        canal: "correo",
        resultado: situacion === "trabajando" ? "aceptado" : "sin_respuesta",
      };
    }
    return {
      participacion_ref: `part_${c.candidatura_ref.replace(":demo:", ":sintetico:").replace(/[^a-zA-Z0-9]/g, "_")}`,
      orden: c.orden,
      orden_acta: c.orden,
      razon_orden: "orden_acta",
      nombre_visible: c.nombre_visible,
      documento_enmascarado: c.documento_enmascarado,
      estado_clave: situacion,
      estado_desde: new Date(c.estado_desde).toISOString(),
      disponible_desde: c.disponible_desde ? new Date(c.disponible_desde).toISOString() : null,
      ultimo_llamamiento: ultimoLlamamiento,
      contactos_total: 0,
    };
  });
  return {
    envelopeBolsas: {
      data: {
        esquema: ESQUEMA_BOLSAS,
        generado_en: "2026-09-17T00:00:00Z",
        bolsas,
      },
    },
    envelopeCandidatos: {
      data: {
        esquema: ESQUEMA_CANDIDATOS,
        generado_en: "2026-09-17T00:00:00Z",
        bolsa: primeraBolsa,
        candidatos,
        contactos: [],
        hay_mas: false,
        cursor_siguiente: null,
      },
    },
  };
}
test("validarRespuestaBolsas acepta fixtures derivadas del dataset sintético", () => {
  const { envelopeBolsas } = construirFixturesDesdeDemo();
  const validado = validarRespuestaBolsas(envelopeBolsas);
  assert.equal(validado.esquema, ESQUEMA_BOLSAS);
  assert.equal(validado.bolsas.length, 12);
  assert.ok(Object.isFrozen(validado));
  assert.ok(Object.isFrozen(validado.bolsas));
  for (const b of validado.bolsas) {
    assert.ok(Object.isFrozen(b));
    assert.ok(Object.isFrozen(b.por_estado));
    assert.equal(typeof b.bolsa_ref, "string");
    assert.equal(typeof b.categoria, "string");
    const sumaEstados = Object.values(b.por_estado).reduce((acc, v) => acc + v, 0);
    assert.equal(sumaEstados, b.total, "la suma de estados debe coincidir con el total");
  }
});
test("B3 registra y consulta contactos con ruta, idempotencia y recibo", async () => {
  let peticion;
  const alta=await registrarContactoCandidato("bolsa:01","participacion:01",{canal:"telefono",resultado:"contactado",anotacion:"Se explicó la oferta",instante:"2026-09-22T00:10:00Z",llamamiento_ref:"",clave_idempotencia:"contacto-0001"},{fetchImpl:async(url,opciones)=>{peticion={url,opciones};return{ok:true,status:201,json:async()=>({data:{recibo_ref:"recibo:contacto:01"}})}}});
  assert.equal(alta.ok,true); assert.equal(peticion.opciones.headers["Idempotency-Key"],"contacto-0001"); assert.match(peticion.url,/\/bolsa:01\/candidatos\/participacion:01\/contactos$/);
  comprobarTransporteInterno(peticion.opciones);
  const lectura=await consultarContactosCandidato("bolsa:01","participacion:01",{fetchImpl:async()=>({ok:true,status:200,json:async()=>({data:{esquema:ESQUEMA_CONTACTOS,contactos:[{contacto_ref:"contacto:01",participacion_ref:"participacion:01",llamamiento_ref:null,canal:"telefono",instante:"2026-09-22T00:10:00Z",actor_ref:"per_0123456789abcdefghijkl",resultado:"contactado",anotacion:"Se explicó la oferta"}],cursor_siguiente:null,hay_mas:false}})})});
  assert.equal(lectura.ok,true); assert.equal(lectura.datos.contactos[0].resultado,"contactado");
});
test("validarRespuestaCandidatosBolsa acepta fixtures derivadas del dataset sintético", () => {
  const { envelopeCandidatos } = construirFixturesDesdeDemo();
  const validado = validarRespuestaCandidatosBolsa(envelopeCandidatos);
  assert.equal(validado.esquema, ESQUEMA_CANDIDATOS);
  assert.equal(validado.hay_mas, false);
  assert.equal(validado.cursor_siguiente, null);
  assert.ok(validado.candidatos.length > 0);
  assert.ok(Object.isFrozen(validado));
  assert.ok(Object.isFrozen(validado.candidatos));
  for (const c of validado.candidatos) {
    assert.ok(Object.isFrozen(c));
    assert.match(c.documento_enmascarado, /^\*{3}\d{4}\*{2}$/);
    assert.ok(SITUACIONES_PARTICIPACION_BOLSA.includes(c.estado_clave));
    assert.ok(Number.isSafeInteger(c.orden) && c.orden >= 1);
  }
});
test("validarRespuestaBolsas rechaza respuestas no canónicas o alteradas", () => {
  const { envelopeBolsas } = construirFixturesDesdeDemo();
  // Sin data
  assert.throws(() => validarRespuestaBolsas(envelopeBolsas.data), /la API debe responder con el envelope canónico/);
  // Esquema incorrecto
  assert.throws(() => validarRespuestaBolsas({
    data: { ...envelopeBolsas.data, esquema: "vec.bolsa.rrhh.bolsas.v2" },
  }), /esquema no compatible/);
  // Propiedad extraña (contrato cerrado)
  assert.throws(() => validarRespuestaBolsas({
    data: { ...envelopeBolsas.data, extra_invalido: 123 },
  }), /no respeta el contrato cerrado/);
  // Estado desconocido en por_estado
  const copiaBolsas = JSON.parse(JSON.stringify(envelopeBolsas));
  copiaBolsas.data.bolsas[0].por_estado.inventado = 1;
  assert.throws(() => validarRespuestaBolsas(copiaBolsas), /no respeta el contrato cerrado/);
});
test("validarRespuestaCandidatosBolsa rechaza fugas de datos personales y DNI sin enmascarar", () => {
  const { envelopeCandidatos } = construirFixturesDesdeDemo();
  // DNI en documento_enmascarado
  const copiaDNI = JSON.parse(JSON.stringify(envelopeCandidatos));
  copiaDNI.data.candidatos[0].documento_enmascarado = "12345678Z";
  assert.throws(() => validarRespuestaCandidatosBolsa(copiaDNI), /debe estar enmascarado/);
  // DNI en nombre_visible
  const copiaNombreDNI = JSON.parse(JSON.stringify(envelopeCandidatos));
  copiaNombreDNI.data.candidatos[0].nombre_visible = "Juan 12345678Z";
  assert.throws(() => validarRespuestaCandidatosBolsa(copiaNombreDNI), /contiene datos personales no permitidos/);
  // Correo electrónico en nombre_visible
  const copiaEmail = JSON.parse(JSON.stringify(envelopeCandidatos));
  copiaEmail.data.candidatos[0].nombre_visible = "aspirante@example.com";
  assert.throws(() => validarRespuestaCandidatosBolsa(copiaEmail), /contiene datos personales no permitidos/);
  // Teléfono en nombre_visible
  const copiaTel = JSON.parse(JSON.stringify(envelopeCandidatos));
  copiaTel.data.candidatos[0].nombre_visible = "Contacto 612345678";
  assert.throws(() => validarRespuestaCandidatosBolsa(copiaTel), /contiene datos personales no permitidos/);
  // Propiedad extraña como telefono
  const copiaCampoExtra = JSON.parse(JSON.stringify(envelopeCandidatos));
  copiaCampoExtra.data.candidatos[0].telefono = "958000000";
  assert.throws(() => validarRespuestaCandidatosBolsa(copiaCampoExtra), /no respeta el contrato cerrado/);
});
test("validarRespuestaCandidatosBolsa exige coherencia entre hay_mas y cursor_siguiente", () => {
  const { envelopeCandidatos } = construirFixturesDesdeDemo();
  // hay_mas true con cursor null
  const copia1 = JSON.parse(JSON.stringify(envelopeCandidatos));
  copia1.data.hay_mas = true;
  copia1.data.cursor_siguiente = null;
  assert.throws(() => validarRespuestaCandidatosBolsa(copia1), /cursor_siguiente debe ser una cadena/);
  // hay_mas false con cursor no null
  const copia2 = JSON.parse(JSON.stringify(envelopeCandidatos));
  copia2.data.hay_mas = false;
  copia2.data.cursor_siguiente = "cursor_123";
  assert.throws(() => validarRespuestaCandidatosBolsa(copia2), /cursor_siguiente debe ser null cuando hay_mas es false/);
});
test("consultarBolsas cliente HTTP maneja 200, 401, 403, 404 y errores de red", async () => {
  const { envelopeBolsas } = construirFixturesDesdeDemo();
  // Éxito 200
  const fetchOk = async (url, opciones) => {
    assert.equal(opciones.credentials, "same-origin");
    comprobarTransporteInterno(opciones);
    assert.equal(opciones.headers.Accept, "application/json");
    return {
      ok: true,
      status: 200,
      json: async () => envelopeBolsas,
    };
  };
  const resOk = await consultarBolsas({ fetchImpl: fetchOk });
  assert.equal(resOk.ok, true);
  assert.equal(resOk.datos.bolsas.length, 12);
  // 401
  const fetch401 = async () => ({ ok: false, status: 401 });
  const res401 = await consultarBolsas({ fetchImpl: fetch401 });
  assert.equal(res401.ok, false);
  assert.equal(res401.codigo, "no_autenticado");
  // 403
  const fetch403 = async () => ({ ok: false, status: 403 });
  const res403 = await consultarBolsas({ fetchImpl: fetch403 });
  assert.equal(res403.ok, false);
  assert.equal(res403.codigo, "acceso_denegado");
  // 404
  const fetch404 = async () => ({ ok: false, status: 404 });
  const res404 = await consultarBolsas({ fetchImpl: fetch404 });
  assert.equal(res404.ok, false);
  assert.equal(res404.codigo, "no_encontrado");
  // Error de red
  const fetchFallo = async () => { throw new Error("Fallo de conexión"); };
  const resFallo = await consultarBolsas({ fetchImpl: fetchFallo });
  assert.equal(resFallo.ok, false);
  assert.equal(resFallo.codigo, "error_red_o_contrato");
});
test("consultarCandidatosBolsa maneja parámetros, códigos de estado y cursor", async () => {
  const { envelopeCandidatos } = construirFixturesDesdeDemo();
  let urlLlamada = "";
  const fetchMock = async (url, opciones) => {
    urlLlamada = url;
    assert.equal(opciones.credentials, "same-origin");
    comprobarTransporteInterno(opciones);
    return {
      ok: true,
      status: 200,
      json: async () => envelopeCandidatos,
    };
  };
  const res = await consultarCandidatosBolsa("bolsa:demo:administrativo", {
    estado: "disponible",
    texto: "Adrián",
    cursor: "cur_abc123",
    limite: 20,
  }, { fetchImpl: fetchMock });
  assert.equal(res.ok, true);
  assert.match(urlLlamada, /\/api\/vec\/bolsa\/bolsas\/bolsa:demo:administrativo\/candidatos/);
  assert.match(urlLlamada, /estado=disponible/);
  assert.match(urlLlamada, /texto=Adri%C3%A1n/);
  assert.match(urlLlamada, /cursor=cur_abc123/);
  assert.match(urlLlamada, /limite=20/);
  // Referencia vacía rechazada
  const resVacia = await consultarCandidatosBolsa("");
  assert.equal(resVacia.ok, false);
  assert.equal(resVacia.codigo, "referencia_invalida");
});
test("el controlador absorbe AbortError y otros rechazos tardíos al desmontar", async () => {
  for (const error of [Object.assign(new Error("abortada"), { name: "AbortError" }), new Error("respuesta cancelada")]) {
    let rechazarLectura;
    let senal;
    let renders = 0;
    const estado = { datosBolsas: null };
    const controlador = crearControladorBolsas({
      estado,
      renderizar: () => { renders += 1; },
      navegar: () => {},
      obtenerFuenteLectura: () => ({
        consultarBolsas: ({ signal }) => {
          senal = signal;
          return new Promise((_resolver, rechazar) => { rechazarLectura = rechazar; });
        },
      }),
    });
    const carga = controlador.cargarBolsas();
    await Promise.resolve();
    assert.equal(senal.aborted, false);
    assert.equal(estado.datosBolsas.carga, "cargando");
    controlador.cancelarPeticiones();
    assert.equal(senal.aborted, true);
    assert.equal(estado.datosBolsas, null);
    rechazarLectura(error);
    await assert.doesNotReject(carga);
    assert.equal(estado.datosBolsas, null);
    assert.equal(renders, 1);
  }
});
test("un rechazo vigente de fuente inyectada termina en error con un único render final", async () => {
  let renders = 0;
  const estado = { datosBolsas: null };
  const controlador = crearControladorBolsas({
    estado,
    renderizar: () => { renders += 1; },
    navegar: () => {},
    obtenerFuenteLectura: () => ({
      consultarBolsas: async () => { throw new Error("fuente no disponible"); },
    }),
  });
  await assert.doesNotReject(controlador.cargarBolsas());
  assert.equal(estado.datosBolsas.carga, "error");
  assert.match(estado.datosBolsas.error, /fuente no disponible/);
  assert.equal(renders, 2, "un render de carga y uno final de error");
});
test("un AbortError vigente limpia la carga sin render tardío", async () => {
  let renders = 0;
  const estado = { datosBolsas: null };
  const controlador = crearControladorBolsas({
    estado,
    renderizar: () => { renders += 1; },
    navegar: () => {},
    obtenerFuenteLectura: () => ({
      consultarBolsas: async () => { throw Object.assign(new Error("abortada"), { name: "AbortError" }); },
    }),
  });
  await assert.doesNotReject(controlador.cargarBolsas());
  assert.equal(estado.datosBolsas, null);
  assert.equal(renders, 1, "solo se renderiza el inicio de la carga");
});
test("una petición A resuelta o rechazada después de B no pisa la bolsa seleccionada", async () => {
  for (const desenlaceAntiguo of ["resolver", "rechazar"]) {
    const pendientes = new Map();
    const estado = { datosCandidatos: null, filtrosBolsa: {} };
    const controlador = crearControladorBolsas({
      estado,
      renderizar: () => {},
      navegar: () => {},
      obtenerFuenteLectura: () => ({
        consultarCandidatosBolsa: (bolsaRef) => new Promise((resolver, rechazar) => {
          pendientes.set(bolsaRef, { resolver, rechazar });
        }),
      }),
    });
    const antigua = controlador.cargarCandidatosBolsa("bolsa:primera");
    await Promise.resolve();
    const nueva = controlador.cargarCandidatosBolsa("bolsa:segunda");
    await Promise.resolve();
    pendientes.get("bolsa:segunda").resolver({ ok: true, datos: { bolsa: { bolsa_ref: "bolsa:segunda" }, candidatos: [] } });
    await nueva;
    if (desenlaceAntiguo === "resolver") {
      pendientes.get("bolsa:primera").resolver({ ok: true, datos: { bolsa: { bolsa_ref: "bolsa:primera" }, candidatos: [] } });
    } else {
      pendientes.get("bolsa:primera").rechazar(new Error("A llegó tarde"));
    }
    await assert.doesNotReject(antigua);
    assert.equal(estado.bolsaSeleccionada, "bolsa:segunda");
    assert.equal(estado.datosCandidatos.datos.bolsa.bolsa_ref, "bolsa:segunda");
  }
});
test("presentadorPanelInterno renderiza el Cuadro B12 en resumen con sus columnas y estados", () => {
  const { envelopeBolsas } = construirFixturesDesdeDemo();
  envelopeBolsas.data.bolsas[1].vigente_hasta = "2025-12-31";
  envelopeBolsas.data.bolsas[2].vigente_desde = "2025-03-07T11:30:00Z";
  envelopeBolsas.data.bolsas[2].vigente_hasta = "2025-12-31T09:30:00Z";
  const datosBolsasValidadas = validarRespuestaBolsas(envelopeBolsas);
  const panelMock = {
    esquema: "vec.bolsa.panel.interno.v1",
    selector: { clase: "organizacion" },
    origen: { revision: "rev_1", actualizada_en: "2026-09-17T00:00:00Z" },
    prueba_lectura: {
      lectura_ref: "lec_1",
      auditoria_ref: "aud_1",
      auditoria_secuencia: 1,
      confirmada_en: "2026-09-17T00:00:00Z",
    },
    indicadores: {
      convocatorias_borrador: 0, convocatorias_revision: 0, convocatorias_pendientes_firma: 0,
      convocatorias_publicadas: 1, bolsas_activas: 12, bolsas_suspendidas: 0, bolsas_agotadas: 0,
      llamamientos_pendientes: 0, llamamientos_en_curso: 0, llamamientos_vencen_hoy: 0,
      documentos_pendientes_firma: 0, incidencias_abiertas: 0,
    },
    convocatorias: [],
    actuaciones_pendientes: [],
  };
  let estadoBolsas = { carga: "listo", datos: datosBolsasValidadas, error: "" };
  const presentador = crearPresentadorPanelInterno({
    claseEstado: (c) => `chip-${c}`,
    encabezadoVista: (_s, t, d, a = "") => `<header><h2>${t}</h2><p>${d}</p>${a}</header>`,
    escaparHTML: (v) => String(v ?? ""),
    numero: (n) => String(n ?? 0),
    obtenerDatosPanel: () => panelMock,
    tituloVista: (v) => v,
    obtenerDatosBolsas: () => estadoBolsas,
  });
  const htmlListo = presentador.renderizarVista("resumen");
  assert.match(htmlListo, /Cuadro B12/);
  assert.match(htmlListo, /Bolsas de trabajo activas \(Cuadro B12\)/);
  assert.match(htmlListo, /12 bolsas/);
  assert.match(htmlListo, /Resumen del Cuadro B12/);
  assert.match(htmlListo, /Bolsas visibles/);
  assert.match(htmlListo, /Aspirantes/);
  assert.match(htmlListo, /Disponibles/);
  assert.match(htmlListo, /ADMINISTRATIVO/);
  const bolsaRef = datosBolsasValidadas.bolsas[0].bolsa_ref;
  assert.match(htmlListo, new RegExp(`<button type="button" class="enlace-tabla" data-accion="ver-bolsa" data-bolsa-ref="${bolsaRef}" aria-label="Abrir candidatos de la bolsa [^"]+">`));
  assert.match(htmlListo, new RegExp(`<button type="button" class="estado-chip exito" data-accion="ver-bolsa" data-bolsa-ref="${bolsaRef}" data-estado="disponible" aria-label="Ver \\d+ candidatos disponibles de [^"]+">`));
  assert.doesNotMatch(htmlListo, /<th scope="col">Acciones<\/th>|Ver candidatos/);
  assert.match(htmlListo, /<time datetime="2025-02-04">4\/2\/25<\/time> \(vigente\)/);
  assert.match(htmlListo, /<time datetime="2025-03-07">7\/3\/25<\/time> — <time datetime="2025-12-31">31\/12\/25<\/time>/);
  assert.match(htmlListo, /<time datetime="2025-03-07T11:30:00Z">7\/3\/25, 12:30<\/time> — <time datetime="2025-12-31T09:30:00Z">31\/12\/25, 10:30<\/time>/);
  assert.doesNotMatch(htmlListo, /<time datetime="2025-02-04">[^<]*:<\/time>/);
  assert.doesNotMatch(htmlListo, />2025-02-04</);
  assert.match(htmlListo, /<time datetime="2026-09-17T00:00:00Z">17\/9\/26, 2:00<\/time>/);
  estadoBolsas = { carga: "listo", datos: { bolsas: [{ ...datosBolsasValidadas.bolsas[0], vigente_desde: "fecha-invalida" }] }, error: "" };
  const htmlFechaInvalida = presentador.renderizarVista("resumen");
  assert.match(htmlFechaInvalida, /<small>Fecha no disponible \(vigente\)<\/small>/);
  assert.doesNotMatch(htmlFechaInvalida, /datetime="fecha-invalida"/);
  // Estado cargando
  estadoBolsas = { carga: "cargando", datos: null, error: "" };
  const htmlCargando = presentador.renderizarVista("resumen");
  assert.match(htmlCargando, /Cargando bolsas de trabajo…/);
  // Estado error
  estadoBolsas = { carga: "error", datos: null, error: "Fallo de conexión 500" };
  const htmlError = presentador.renderizarVista("resumen");
  assert.match(htmlError, /No se pudieron cargar las bolsas de trabajo/);
  assert.match(htmlError, /reintentar-bolsas/);
  // Estado denegado
  estadoBolsas = { carga: "denegado", datos: null, error: "Sin permiso" };
  const htmlDenegado = presentador.renderizarVista("resumen");
  assert.match(htmlDenegado, /Acceso denegado a la consulta de bolsas/);
  // Estado vacío
  estadoBolsas = { carga: "listo", datos: { bolsas: [] }, error: "" };
  const htmlVacio = presentador.renderizarVista("resumen");
  assert.match(htmlVacio, /No hay bolsas de trabajo activas/);
});
test("presentadorPanelInterno renderiza Vista B5 de candidatos con filtros, chips y paginación", () => {
  const { envelopeCandidatos } = construirFixturesDesdeDemo();
  const datosCandidatosValidados = validarRespuestaCandidatosBolsa(envelopeCandidatos);
  const panelMock = {
    esquema: "vec.bolsa.panel.interno.v1",
    selector: { clase: "organizacion" },
    origen: { revision: "rev_1", actualizada_en: "2026-09-17T00:00:00Z" },
    prueba_lectura: { lectura_ref: "lec_1", auditoria_ref: "aud_1", auditoria_secuencia: 1, confirmada_en: "2026-09-17T00:00:00Z" },
    indicadores: {}, convocatorias: [], actuaciones_pendientes: [],
  };
  let estadoCandidatos = { carga: "listo", datos: datosCandidatosValidados, error: "" };
  let filtrosBolsa = { estado: "disponible", texto: "Claudio" };
  const presentador = crearPresentadorPanelInterno({
    claseEstado: (c) => `chip-${c}`,
    encabezadoVista: (_s, t, d, a = "") => `<header><h2>${t}</h2><p>${d}</p>${a}</header>`,
    escaparHTML: (v) => String(v ?? ""),
    numero: (n) => String(n ?? 0),
    obtenerDatosPanel: () => panelMock,
    tituloVista: (v) => v,
    obtenerDatosCandidatosBolsa: () => estadoCandidatos,
    obtenerEstadoCandidatos: () => filtrosBolsa,
  });
  const htmlB5 = presentador.renderizarVista("bolsa-candidatos");
  assert.match(htmlB5, /Vista B5/);
  assert.match(htmlB5, /Filtros y ordenación de aspirantes/);
  assert.match(htmlB5, /data-bolsa-form="filtros"/);
  assert.match(htmlB5, /Volver al cuadro/);
  assert.match(htmlB5, /Claudio/);
  assert.match(htmlB5, /\*\*\*0034\*\*/);
  assert.match(htmlB5, /data-bolsa-accion="abrir-ficha"/);
  assert.match(htmlB5, /class="rejilla-kpi kpi-candidatos"/);
  assert.match(htmlB5, /class="tarjeta-kpi kpi-filtro kpi--exito"/);
  assert.match(htmlB5, /Resumen de la bolsa/);
  assert.match(htmlB5, /Consultar historial de contactos/);
  assert.match(htmlB5, /Nuevo llamamiento/);
  assert.match(htmlB5, /Registrar resultado/);
  assert.match(htmlB5, /El llamamiento se registra con recibo/);
  assert.match(htmlB5, /data-bolsa-accion="iniciar-b7"/);
  assert.match(htmlB5, /title="Pendiente de RRHH"/);
  assert.match(htmlB5, /Criterios de orden/);
  assert.match(htmlB5, /Puntuación descendente; desempate estable por nº del acta/);
  assert.match(htmlB5, /Provisional, pendiente de RRHH \(dudas 13–14\)/);
  filtrosBolsa = { ...filtrosBolsa, pestana: "historico" };
  const htmlHistorico = presentador.renderizarVista("bolsa-candidatos");
  assert.match(htmlHistorico, /Histórico de contactos y llamamientos/);
  assert.match(htmlHistorico, /<th scope="col">Contacto<\/th>/);
  assert.match(htmlHistorico, /Sin llamamientos registrados|Llamamiento/);
  assert.match(htmlHistorico, /<section class="panel" hidden>/);
  // Con paginación
  const candidatosConPaginacion = {
    ...datosCandidatosValidados,
    hay_mas: true,
    cursor_siguiente: "token_siguiente_pag",
  };
  estadoCandidatos = { carga: "listo", datos: candidatosConPaginacion, error: "" };
  const htmlPag = presentador.renderizarVista("bolsa-candidatos");
  assert.match(htmlPag, /Cargar siguientes aspirantes/);
  assert.match(htmlPag, /token_siguiente_pag/);
  // Estados error y denegado
  estadoCandidatos = { carga: "error", datos: null, error: "Error de servidor 500" };
  const htmlErr = presentador.renderizarVista("bolsa-candidatos");
  assert.match(htmlErr, /Error al consultar candidatos/);
  assert.match(htmlErr, /reintentar-candidatos/);
  assert.match(htmlErr, /Volver al cuadro/);
});
test("P-WEB-06: B12 enlaza cada estado B2 y B5 señala la bolsa B9 vigente", () => {
  const { envelopeBolsas, envelopeCandidatos } = construirFixturesDesdeDemo();
  const bolsas = validarRespuestaBolsas(envelopeBolsas);
  const datosCandidatos = validarRespuestaCandidatosBolsa(envelopeCandidatos);
  const sustituida = { ...bolsas.bolsas[0], vigente_hasta: "2026-01-31" };
  const vigente = { ...bolsas.bolsas[1], categoria_clave: sustituida.categoria_clave, vigente_desde: "2026-02-01", vigente_hasta: null };
  const estadoBolsas = { carga: "listo", datos: { ...bolsas, bolsas: [sustituida, vigente] }, error: "" };
  const estadoCandidatos = { carga: "listo", datos: { ...datosCandidatos, bolsa: sustituida }, error: "" };
  const presentador = crearPresentadorPanelInterno({
    claseEstado: () => "neutro", encabezadoVista: (_s, t) => `<header><h2>${t}</h2></header>`,
    escaparHTML: (v) => String(v ?? ""), numero: (n) => String(n ?? 0),
    obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1", indicadores: {}, convocatorias: [], actuaciones_pendientes: [] }),
    tituloVista: (v) => v, obtenerDatosBolsas: () => estadoBolsas,
    obtenerDatosCandidatosBolsa: () => estadoCandidatos, obtenerEstadoCandidatos: () => ({ estado: "", texto: "" }),
  });
  const cuadro = presentador.renderizarSoloBolsas("resumen");
  for (const estado of ["disponible", "trabajando", "no_disponible", "excluido", "renuncia", "pendiente_incorporacion"]) {
    assert.match(cuadro, new RegExp(`data-estado="${estado}"`));
  }
  const detalle = presentador.renderizarVista("bolsa-candidatos");
  assert.match(detalle, /Sustituida por la bolsa vigente de la categoría el 01\/02\/2026\./);
  assert.match(detalle, new RegExp(`data-bolsa-ref="${vigente.bolsa_ref}"`));
});
test("P-WEB-06: el histórico muestra el contacto B3 completo y conserva el guion cuando no existe", () => {
  const { envelopeCandidatos } = construirFixturesDesdeDemo();
  const datos = validarRespuestaCandidatosBolsa(envelopeCandidatos);
  const candidato = datos.candidatos[0];
  const contacto = validarContacto({ contacto_ref: "contacto:p-web-06", participacion_ref: candidato.participacion_ref, llamamiento_ref: null, canal: "telefono", instante: "2026-09-22T10:00:00Z", actor_ref: "persona:rrhh:p-web-06", resultado: "contactado", anotacion: "Oferta comunicada" });
  const presentador = crearPresentadorPanelInterno({
    claseEstado: () => "neutro", encabezadoVista: (_s, t) => `<header><h2>${t}</h2></header>`,
    escaparHTML: (v) => String(v ?? ""), numero: (n) => String(n ?? 0),
    obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1", indicadores: {}, convocatorias: [], actuaciones_pendientes: [] }),
    tituloVista: (v) => v, obtenerDatosCandidatosBolsa: () => ({ carga: "listo", datos: { ...datos, contactos: [contacto] }, error: "" }),
    obtenerEstadoCandidatos: () => ({ estado: "", texto: "", pestana: "historico" }),
  });
  const html = presentador.renderizarVista("bolsa-candidatos");
  assert.match(html, /Teléfono · Contactado · Oferta comunicada/);
  assert.match(html, />—<\/td>/);
});
test("presentadorPanelInterno muestra la ficha B5 en línea junto al único candidato seleccionado", () => {
  const { envelopeCandidatos } = construirFixturesDesdeDemo();
  const datos = validarRespuestaCandidatosBolsa(envelopeCandidatos);
  const candidato = datos.candidatos[0];
  const segundoCandidato = datos.candidatos[1];
  let modalFicha = null;
  const presentador = crearPresentadorPanelInterno({
    claseEstado: (c) => `chip-${c}`,
    encabezadoVista: (_s, t, d, a = "") => `<header><h2>${t}</h2><p>${d}</p>${a}</header>`,
    escaparHTML: (v) => String(v ?? ""),
    numero: (n) => String(n ?? 0),
    obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }),
    tituloVista: (v) => v,
    obtenerDatosCandidatosBolsa: () => ({ carga: "listo", datos, error: "" }),
    obtenerEstadoCandidatos: () => ({ estado: "", texto: "" }),
    obtenerModalFicha: () => modalFicha,
  });
  const htmlInicial = presentador.renderizarVista("bolsa-candidatos");
  assert.doesNotMatch(htmlInicial, /Ficha de participación/);
  assert.match(htmlInicial, /data-bolsa-control-principal="true"/);
  assert.match(htmlInicial, /aria-expanded="false"/);
  assert.doesNotMatch(htmlInicial, /Ficha en aspirante|<th scope="col">Acciones<\/th>/);
  modalFicha = { abierto: true, candidato, bolsa: datos.bolsa };
  const htmlAbierto = presentador.renderizarVista("bolsa-candidatos");
  const fichaId = `ficha-participacion-${candidato.participacion_ref}`;
  assert.match(htmlAbierto, /Ficha de participación/);
  assert.match(htmlAbierto, /Referencia de participación/);
  assert.match(htmlAbierto, new RegExp(candidato.participacion_ref));
  assert.match(htmlAbierto, /data-bolsa-accion="cerrar-ficha"/);
  assert.match(htmlAbierto, new RegExp(`aria-expanded="true" aria-controls="${fichaId}"`));
  assert.match(htmlAbierto, new RegExp(`</tr>\\s*<tr class="fila-ficha-participacion" data-ficha-participacion-ref="${candidato.participacion_ref}"`));
  assert.doesNotMatch(htmlAbierto, /role="dialog"|aria-modal="true"|modal-fondo/);
  assert.match(htmlAbierto, new RegExp(`id="${fichaId}" class="panel" data-bolsa-ficha-inline="true" tabindex="-1"`));
  const ficha = htmlAbierto.match(new RegExp(`<section id="${fichaId}"[\\s\\S]*?</section>`))[0];
  assert.doesNotMatch(ficha, /correo|teléfono|puntuación|relación laboral/i);
  modalFicha = { abierto: true, candidato: segundoCandidato, bolsa: datos.bolsa };
  const htmlSegundo = presentador.renderizarVista("bolsa-candidatos");
  assert.equal((htmlSegundo.match(/fila-ficha-participacion/g) || []).length, 1);
  assert.match(htmlSegundo, new RegExp(`data-ficha-participacion-ref="${segundoCandidato.participacion_ref}"`));
  assert.match(htmlSegundo, new RegExp(`data-participacion-ref="${candidato.participacion_ref}"[^>]*>[\\s\\S]*?aria-expanded="false"`));
  modalFicha = null;
  const htmlCerrado = presentador.renderizarVista("bolsa-candidatos");
  assert.doesNotMatch(htmlCerrado, /Fila de participación|fila-ficha-participacion|Ficha de participación/);
});
test("B7 presenta cuatro pasos, paginación interna y controles de teclado nativos", () => {
  const { envelopeCandidatos } = construirFixturesDesdeDemo();
  const base = validarRespuestaCandidatosBolsa(envelopeCandidatos);
  const pausado = base.candidatos.find((c) => c.estado_clave === "no_disponible") || base.candidatos[1];
  const datos = { ...base, candidatos: base.candidatos.map((c) => c.participacion_ref === pausado.participacion_ref ? { ...c, orden: null, razon_orden: "pausa" } : c) };
  const flujo = { paso: 1, estados: ["disponible"], participaciones: [], configuracion: null };
  const presentador = crearPresentadorPanelInterno({
    claseEstado: (c) => `chip-${c}`,
    encabezadoVista: (_s, t, d, a = "") => `<header><h2>${t}</h2><p>${d}</p>${a}</header>`,
    escaparHTML: (v) => String(v ?? ""),
    numero: (n) => String(n ?? 0),
    obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }),
    tituloVista: (v) => v,
    obtenerDatosCandidatosBolsa: () => ({ carga: "listo", datos, error: "" }),
    obtenerEstadoCandidatos: () => ({ nuevo_llamamiento: flujo }),
  });
  let html = presentador.renderizarVista("bolsa-candidatos");
  assert.match(html, /1\. Seleccionar bolsa/);
  assert.match(html, /aria-current="step" tabindex="-1"/);
  assert.match(html, /type="submit">Seleccionar esta bolsa/);
  flujo.paso = 2;
  html = presentador.renderizarVista("bolsa-candidatos");
  assert.match(html, /2\. Seleccionar candidatos/);
  assert.match(html, /Respetar orden de prelación \(obligatorio\)/);
  assert.match(html, /no ocupan turno/);
  assert.doesNotMatch(html, new RegExp(pausado.nombre_visible));
  assert.match(html, /Mostrando 1 a 6 de/);
  assert.match(html, /data-bolsa-accion="b7-pagina"/);
  assert.match(html, /Seleccionar todas las que cumplen el filtro/);
  assert.match(html, /0 seleccionadas/);
  flujo.paso = 3;
  html = presentador.renderizarVista("bolsa-candidatos");
  assert.match(html, /3\. Configurar llamamiento/);
  assert.match(html, /El asunto y el texto son comunes a todas las personas seleccionadas/);
  assert.match(html, /No se ofrece personalización individual en este formulario/);
  assert.match(html, /name="plazo" required minlength="2" maxlength="160" value=""/);
  assert.match(html, /Plazo de respuesta indicado por RRHH/);
  assert.match(html, /no presupone un plazo legal/);
  assert.doesNotMatch(html, /48 horas|relay de desarrollo|value="Pendiente de definición por RRHH"/);
  assert.match(html, /bolsa-llamamiento-v1/);
  flujo.paso = 4;
  flujo.participaciones = [datos.candidatos.find((c) => c.estado_clave === "disponible").participacion_ref];
  flujo.configuracion = { referencia: "NEC-01", centro: "Centro", modalidad: "Sustitución", plazo: "Hasta el 30 de septiembre, 14:00" };
  html = presentador.renderizarVista("bolsa-candidatos");
  assert.match(html, /4\. Revisar y enviar/);
  assert.match(html, /name="confirmacion" required/);
  assert.match(html, /type="submit" class="boton-primario"/);
});
test("P-WEB-08 presenta estadísticas y enlaza cada cifra por bolsa con B5", () => {
  const estadisticas = validarRespuestaEstadisticas({ data: { esquema: ESQUEMA_ESTADISTICAS, generado_en: "2026-09-23T08:00:00Z", bolsas: { total: 1, vigentes: 1, sustituidas: 0 }, personas: { total: 2, por_estado: { disponible: 1, no_disponible: 0, trabajando: 1, pendiente_incorporacion: 0, renuncia: 0, excluido: 0, disponible_desde: 0 } }, llamamientos: { total: 1, por_canal: { correo: 1 }, por_resultado: { pendiente: 1 } }, por_bolsa: [{ bolsa_ref: "bolsa:01", categoria: "Auxiliar", tipo_lista: "ordinaria", vigente: true, total: 2, por_estado: { disponible: 1, no_disponible: 0, trabajando: 1, pendiente_incorporacion: 0, renuncia: 0, excluido: 0, disponible_desde: 0 } }] } });
  const presentador = crearPresentadorPanelInterno({ claseEstado: (c) => c, encabezadoVista: (_s, t, d, a = "") => `<header><h2>${t}</h2><p>${d}</p>${a}</header>`, escaparHTML: (v) => String(v ?? ""), numero: (n) => String(n ?? 0), obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }), tituloVista: (v) => v, obtenerDatosEstadisticas: () => ({ carga: "listo", datos: estadisticas, error: "" }) });
  const html = presentador.renderizarVista("estadisticas");
  assert.match(html, /Indicadores agregados del ámbito autorizado/);
  assert.match(html, /data-accion="ver-bolsa" data-bolsa-ref="bolsa:01" data-estado="disponible"/);
  assert.match(html, /Personas y llamamientos/);
});
test("controlador de B5 lleva el foco a la ficha inline y lo recupera en su control principal", () => {
  const { envelopeCandidatos } = construirFixturesDesdeDemo();
  const datos = validarRespuestaCandidatosBolsa(envelopeCandidatos);
  const candidato = datos.candidatos[0];
  const focos = [];
  const documento = {
    querySelector(selector) {
      assert.equal(selector, "[data-bolsa-ficha-inline='true']");
      return { focus: () => focos.push("ficha") };
    },
    querySelectorAll(selector) {
      assert.equal(selector, '[data-bolsa-accion="abrir-ficha"][data-bolsa-control-principal="true"]');
      return [
        { dataset: { participacionRef: "otra-participacion" }, focus: () => focos.push("otro") },
        { dataset: { participacionRef: candidato.participacion_ref }, focus: () => focos.push("control-principal") },
      ];
    },
  };
  let renderizados = 0;
  const estado = {
    datosCandidatos: { carga: "listo", datos, error: "" },
    modalFicha: null,
  };
  const controlador = crearControladorBolsas({
    estado,
    renderizar: () => { renderizados += 1; },
    navegar: () => {},
    documento,
  });
  controlador.abrirFicha(candidato.participacion_ref);
  assert.equal(estado.modalFicha.candidato.participacion_ref, candidato.participacion_ref);
  assert.deepEqual(focos, ["ficha"]);
  controlador.cerrarFicha();
  assert.equal(estado.modalFicha, null);
  assert.deepEqual(focos, ["ficha", "control-principal"]);
  assert.equal(renderizados, 2);
});
test("B7 lleva el foco al raíl al iniciarse desde teclado", () => {
  const escuchas = {};
  const focos = [];
  const boton = { dataset: { bolsaAccion: "iniciar-b7" } };
  const documento = {
    addEventListener(tipo, fn) { escuchas[tipo] = fn; },
    querySelector(selector) {
      assert.equal(selector, '[aria-current="step"]');
      return { focus: () => focos.push("paso") };
    },
  };
  const estado = { filtrosBolsa: {}, datosCandidatos: { carga: "listo", datos: { candidatos: [] } } };
  let renderizados = 0;
  const controlador = crearControladorBolsas({ estado, renderizar: () => { renderizados += 1; }, navegar: () => {}, documento });
  controlador.instalar();
  escuchas.click({
    preventDefault() {},
    target: { closest(selector) { return selector === "[data-bolsa-accion]" ? boton : null; } },
  });
  assert.equal(estado.filtrosBolsa.nuevo_llamamiento.paso, 1);
  assert.deepEqual(focos, ["paso"]);
  assert.equal(renderizados, 1);
});
