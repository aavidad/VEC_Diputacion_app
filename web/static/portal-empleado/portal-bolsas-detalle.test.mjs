import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";

import {
  ESQUEMA_BOLSAS,
  ESQUEMA_CANDIDATOS,
  ESQUEMA_CONTACTOS,
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
  validarPayloadCrearLlamamiento,
  validarPayloadResultadoLlamamiento,
  construirEnvelopeAccionBolsa,
} from "./portal-bolsas-contrato.js";

import {
  consultarBolsas,
  consultarCandidatosBolsa,
  consultarContactosCandidato,
  crearLlamamientoCandidato,
  registrarResultadoLlamamiento,
  rutaCandidatosBolsa,
  crearControladorBolsas,
} from "./portal-bolsas-api.js";

import { crearPresentadorPanelInterno } from "./portal-panel-interno.js";

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


test("el acceso B12 abre la lectura B5 de su propia bolsa sin ejecutar mutaciones", async () => {
  const { envelopeBolsas, envelopeCandidatos } = construirFixturesDesdeDemo();
  const bolsas = validarRespuestaBolsas(envelopeBolsas);
  const candidatos = validarRespuestaCandidatosBolsa(envelopeCandidatos);
  const bolsaRef = bolsas.bolsas[0].bolsa_ref;
  const oyentes = new Map();
  const navegaciones = [];
  const consultas = [];
  const focos = [];
  let prevenido = false;
  const documento = {
    addEventListener(tipo, oyente) { oyentes.set(tipo, oyente); },
    querySelector(selector) {
      assert.equal(selector, "[data-bolsa-b5-destino='true']");
      return { focus: () => focos.push("b5") };
    },
    querySelectorAll() { return []; },
  };
  const estado = {
    bolsaSeleccionada: "",
    filtrosBolsa: { estado: "", texto: "" },
    datosCandidatos: null,
  };
  const controlador = crearControladorBolsas({
    estado,
    renderizar: () => {},
    navegar: (vista) => navegaciones.push(vista),
    obtenerFuenteLectura: () => ({
      consultarCandidatosBolsa: async (referencia, opciones) => {
        consultas.push({ referencia, opciones });
        return { ok: true, datos: candidatos };
      },
    }),
    documento,
  });

  controlador.instalar();
  oyentes.get("click")({
    target: {
      closest(selector) {
        return selector === '[data-accion="ver-bolsa"], [data-bolsa-abrir="true"]'
          ? { dataset: { bolsaRef } }
          : null;
      },
    },
    preventDefault() { prevenido = true; },
  });
  await new Promise((resolver) => setImmediate(resolver));

  assert.equal(prevenido, true);
  assert.equal(estado.bolsaSeleccionada, bolsaRef);
  assert.deepEqual(navegaciones, ["bolsa-candidatos"]);
  assert.deepEqual(consultas, [{ referencia: bolsaRef, opciones: { estado: "", texto: "", cursor: "" } }]);
  assert.equal(estado.datosCandidatos.datos.bolsa.bolsa_ref, bolsaRef);
  assert.deepEqual(focos, ["b5"]);
});

test("las vistas de bolsa no contienen la palabra demo en sus textos visibles", () => {
  const { envelopeBolsas, envelopeCandidatos } = construirFixturesDesdeDemo();
  const panelMock = {
    esquema: "vec.bolsa.panel.interno.v1",
    selector: { clase: "organizacion" },
    origen: { revision: "rev_1", actualizada_en: "2026-09-17T00:00:00Z" },
    prueba_lectura: { lectura_ref: "lec_1", auditoria_ref: "aud_1", auditoria_secuencia: 1, confirmada_en: "2026-09-17T00:00:00Z" },
    indicadores: {}, convocatorias: [], actuaciones_pendientes: [],
  };

  const presentador = crearPresentadorPanelInterno({
    claseEstado: (c) => `chip-${c}`,
    encabezadoVista: (_s, t, d, a = "") => `<header><h2>${t}</h2><p>${d}</p>${a}</header>`,
    escaparHTML: (v) => String(v ?? ""),
    numero: (n) => String(n ?? 0),
    obtenerDatosPanel: () => panelMock,
    tituloVista: (v) => v,
    obtenerDatosBolsas: () => ({ carga: "listo", datos: validarRespuestaBolsas(envelopeBolsas), error: "" }),
    obtenerDatosCandidatosBolsa: () => ({ carga: "listo", datos: validarRespuestaCandidatosBolsa(envelopeCandidatos), error: "" }),
    obtenerEstadoCandidatos: () => ({ estado: "", texto: "" }),
  });

  const resumenHtml = presentador.renderizarVista("resumen");
  const candidatosHtml = presentador.renderizarVista("bolsa-candidatos");

  // Textos visibles no deben incluir "demo"
  assert.doesNotMatch(resumenHtml, /\bdemo\b/i);
  assert.doesNotMatch(candidatosHtml, /\bdemo\b/i);
});

test("contrato de contactos y acciones: validación estricta de contacto y respuesta", () => {
  const contactoValido = {
    contacto_ref: "contacto:sintetico:001",
    participacion_ref: "part:001",
    llamamiento_ref: null,
    canal: "telefono",
    instante: "2026-09-17T10:30:00Z",
    actor_ref: "actor:rrhh:001",
    resultado: "acepta",
    anotacion: "Acepta incorporación inmediata",
  };

  const validado = validarContacto(contactoValido);
  assert.equal(validado.contacto_ref, "contacto:sintetico:001");
  assert.equal(validado.canal, "telefono");
  assert.equal(validado.resultado, "acepta");

  // Falla si canal no es válido
  assert.throws(() => validarContacto({ ...contactoValido, canal: "paloma_mensajera" }), /canal de contacto no reconocido/);

  // Falla si resultado no es válido
  assert.throws(() => validarContacto({ ...contactoValido, resultado: "indeciso" }), /resultado de contacto no reconocido/);

  // Falla ante datos personales en anotación
  assert.throws(() => validarContacto({ ...contactoValido, anotacion: "Llamar a test@diputacion.es" }), /contiene datos personales/);

  // Falla si faltan campos o hay campos extra
  assert.throws(() => validarContacto({ ...contactoValido, extra: "no_permitido" }), /no respeta el contrato cerrado/);

  // Envelope canónico de contactos
  const envelope = {
    data: {
      esquema: ESQUEMA_CONTACTOS,
      generado_en: "2026-09-17T12:00:00Z",
      participacion_ref: "part:001",
      contactos: [contactoValido],
    },
  };
  const respuestaValidada = validarRespuestaContactos(envelope);
  assert.equal(respuestaValidada.esquema, ESQUEMA_CONTACTOS);
  assert.equal(respuestaValidada.contactos.length, 1);
  assert.equal(respuestaValidada.contactos[0].contacto_ref, "contacto:sintetico:001");
});

test("contrato de acciones: validación de payload de crear llamamiento y resultado", () => {
  const payloadLlamar = {
    canal: "correo",
    comunicado_en: "2026-09-17T09:00:00Z",
    plazo_respuesta_hasta: "2026-09-19T23:59:59Z",
    anotacion: "Primer llamamiento para plaza vacante",
  };
  const llamamientoValidado = validarPayloadCrearLlamamiento(payloadLlamar);
  assert.equal(llamamientoValidado.canal, "correo");

  assert.throws(() => validarPayloadCrearLlamamiento({ ...payloadLlamar, canal: "fax" }), /canal de llamamiento no válido/);
  assert.throws(() => validarPayloadCrearLlamamiento({ ...payloadLlamar, comunicado_en: "fecha_invalida" }), /no es una fecha válida|debe ser un instante válido/);

  const payloadResultado = {
    resultado_clave: "renuncia",
    anotacion: "Renuncia por incompatibilidad horaria",
  };
  const resultadoValidado = validarPayloadResultadoLlamamiento(payloadResultado);
  assert.equal(resultadoValidado.resultado_clave, "renuncia");

  assert.throws(() => validarPayloadResultadoLlamamiento({ resultado_clave: "otra_cosa" }), /resultado_clave no válido/);

  // Construcción de envelope de acción
  const accion = construirEnvelopeAccionBolsa("crear_llamamiento", llamamientoValidado, { confirmacion: true });
  assert.equal(accion.esquema, ESQUEMA_ACCION_BOLSA);
  assert.equal(accion.accion, "crear_llamamiento");
  assert.equal(accion.confirmacion, true);
  assert.deepEqual(accion.payload, llamamientoValidado);

  // Falla sin confirmación explícita
  assert.throws(() => construirEnvelopeAccionBolsa("crear_llamamiento", llamamientoValidado, { confirmacion: false }), /confirmación explícita/);
});

test("cliente API: consultarContactosCandidato maneja 200, 403 y errores", async () => {
  const mockFetchOk = async (url, opciones) => {
    assert.match(url, /\/api\/vec\/bolsa\/bolsas\/bolsa_123\/candidatos\/part_123\/contactos/);
    assert.equal(opciones.credentials, "same-origin");
    assert.deepEqual([opciones.mode, opciones.cache, opciones.redirect], ["same-origin", "no-store", "error"]);
    assert.equal(opciones.headers.Accept, "application/json");
    return {
      ok: true,
      status: 200,
      json: async () => ({
        data: {
          esquema: ESQUEMA_CONTACTOS,
          generado_en: "2026-09-17T12:00:00Z",
          participacion_ref: "part_123",
          contactos: [
            {
              contacto_ref: "c_1",
              participacion_ref: "part_123",
              llamamiento_ref: null,
              canal: "correo",
              instante: "2026-09-17T10:00:00Z",
              actor_ref: "actor:rrhh:001",
              resultado: "contactado",
              anotacion: "Notificación telemática enviada",
            },
          ],
        },
      }),
    };
  };

  const resOk = await consultarContactosCandidato("bolsa_123", "part_123", { fetchImpl: mockFetchOk });
  assert.equal(resOk.ok, true);
  assert.equal(resOk.datos.contactos.length, 1);
  assert.equal(resOk.datos.contactos[0].canal, "correo");

  const mockFetchDenegado = async () => ({
    ok: false,
    status: 403,
  });
  const resDenegado = await consultarContactosCandidato("bolsa_123", "part_123", { fetchImpl: mockFetchDenegado });
  assert.equal(resDenegado.ok, false);
  assert.equal(resDenegado.status, 403);
  assert.equal(resDenegado.codigo, "acceso_denegado");
});

test("cliente API: crearLlamamientoCandidato y registrarResultadoLlamamiento emiten envelope correcto", async () => {
  let llamadaLlamar = null;
  const mockFetchLlamar = async (url, opciones) => {
    llamadaLlamar = { url, opciones };
    return {
      ok: true,
      status: 200,
      json: async () => ({
        data: {
          recibo_ref: "recibo:llamamiento:001",
          estado_clave: "trabajando",
        },
      }),
    };
  };

  const resLlamar = await crearLlamamientoCandidato("part_456", {
    canal: "telefono",
    comunicado_en: "2026-09-17T10:00:00Z",
    plazo_respuesta_hasta: "2026-09-19T10:00:00Z",
    anotacion: "Llamada telefónica realizada",
  }, { fetchImpl: mockFetchLlamar });

  assert.equal(resLlamar.ok, true);
  assert.match(llamadaLlamar.url, /\/api\/vec\/bolsa\/candidatos\/part_456\/llamamientos/);
  assert.equal(llamadaLlamar.opciones.method, "POST");
  assert.equal(llamadaLlamar.opciones.credentials, "same-origin");
  assert.deepEqual([llamadaLlamar.opciones.mode, llamadaLlamar.opciones.cache, llamadaLlamar.opciones.redirect], ["same-origin", "no-store", "error"]);
  const bodyLlamar = JSON.parse(llamadaLlamar.opciones.body);
  assert.equal(bodyLlamar.esquema, ESQUEMA_ACCION_BOLSA);
  assert.equal(bodyLlamar.accion, "crear_llamamiento");
  assert.equal(bodyLlamar.confirmacion, true);

  // Registrar resultado
  let llamadaResultado = null;
  const mockFetchResultado = async (url, opciones) => {
    llamadaResultado = { url, opciones };
    return {
      ok: true,
      status: 200,
      json: async () => ({
        data: {
          recibo_ref: "recibo:resultado:001",
          resultado_clave: "aceptado",
        },
      }),
    };
  };

  const resResultado = await registrarResultadoLlamamiento("llam_789", {
    resultado_clave: "aceptado",
    anotacion: "Acepta la vacante ofrecida",
  }, { fetchImpl: mockFetchResultado });

  assert.equal(resResultado.ok, true);
  assert.match(llamadaResultado.url, /\/api\/vec\/bolsa\/llamamientos\/llam_789\/resultado/);
  assert.deepEqual([llamadaResultado.opciones.mode, llamadaResultado.opciones.cache, llamadaResultado.opciones.redirect], ["same-origin", "no-store", "error"]);
  const bodyResultado = JSON.parse(llamadaResultado.opciones.body);
  assert.equal(bodyResultado.esquema, ESQUEMA_ACCION_BOLSA);
  assert.equal(bodyResultado.accion, "registrar_resultado");
  assert.equal(bodyResultado.confirmacion, true);
  assert.equal(bodyResultado.payload.resultado_clave, "aceptado");
});

test("interfaz B5: conecta el nuevo llamamiento y mantiene pendiente la respuesta", () => {
  const { envelopeCandidatos } = construirFixturesDesdeDemo();
  const datosCandidatosValidados = validarRespuestaCandidatosBolsa(envelopeCandidatos);

  let modalContactos = null;
  let modalResultado = null;

  const presentador = crearPresentadorPanelInterno({
    claseEstado: (c) => `chip-${c}`,
    encabezadoVista: (_s, t, d, a = "") => `<header><h2>${t}</h2><p>${d}</p>${a}</header>`,
    escaparHTML: (v) => String(v ?? ""),
    numero: (n) => String(n ?? 0),
    obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }),
    tituloVista: (v) => v,
    obtenerDatosBolsas: () => ({ carga: "listo", datos: { bolsas: [] }, error: "" }),
    obtenerDatosCandidatosBolsa: () => ({ carga: "listo", datos: datosCandidatosValidados, error: "" }),
    obtenerEstadoCandidatos: () => ({ estado: "", texto: "" }),
    obtenerModalContactos: () => modalContactos,
    obtenerModalResultado: () => modalResultado,
  });

  const html = presentador.renderizarVista("bolsa-candidatos");

  // El aspirante es el único control principal para abrir la ficha.
  assert.doesNotMatch(html, /<th scope="col">Acciones<\/th>/);
  assert.match(html, /data-bolsa-accion="abrir-ficha"/);
  assert.doesNotMatch(html, /Operaciones conectadas|Vista B5|en esta página/);
  assert.match(html, /Consultar historial de contactos/);
  assert.match(html, /Nuevo llamamiento/);
  assert.match(html, /Registrar resultado/);
  assert.match(html, /data-bolsa-accion="iniciar-b7"/);
  assert.match(html, /disabled aria-disabled="true" title="Pendiente de RRHH"/);
  assert.doesNotMatch(html, /abrir-contactos|abrir-llamar|abrir-resultado/);

  // Modal de contactos abierto con datos
  modalContactos = {
    abierto: true,
    participacionRef: "part_demo_1",
    nombreVisible: "Aspirante de Prueba",
    carga: "listo",
    contactos: [
      {
        contacto_ref: "ct_1",
        canal: "telefono",
        realizado_en: "2026-09-17T11:00:00Z",
        resultado_clave: "aceptado",
        anotacion: "Llamada satisfactoria",
      },
    ],
  };
  const htmlConContactos = presentador.renderizarVista("bolsa-candidatos");
  assert.doesNotMatch(htmlConContactos, /Historial de contactos|Llamada satisfactoria|cerrar-contactos/);

  // Modal de resultado (B3)
  modalContactos = null;
  modalResultado = {
    abierto: true,
    llamamientoRef: "llam_1",
    participacionRef: "part_demo_1",
    nombreVisible: "Aspirante de Prueba",
    orden: 3,
    carga: "ocioso",
    error: "",
  };
  const htmlConResultado = presentador.renderizarVista("bolsa-candidatos");
  assert.doesNotMatch(htmlConResultado, /Registrar resultado de llamamiento \(B3\)|data-bolsa-form="resultado"|resultado-clave/);
});

test("sin fuente B5 configurada no ofrece candidaturas ni acciones de muestra", () => {
  const presentador = crearPresentadorPanelInterno({
    claseEstado: () => "", encabezadoVista: (_s, titulo) => `<h2>${titulo}</h2>`,
    escaparHTML: (valor) => String(valor ?? ""), numero: (valor) => String(valor ?? 0),
    obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }), tituloVista: (valor) => valor,
    obtenerDatosCandidatosBolsa: () => null,
    obtenerEstadoCandidatos: () => ({ estado: "", texto: "" }),
  });
  const html = presentador.renderizarVista("bolsa-candidatos");
  assert.match(html, /Consulta no configurada/);
  assert.match(html, /No hay una fuente autorizada/);
  assert.doesNotMatch(html, /DEMO-BOL|Historial sintético|data-bolsa-accion="iniciar-b7"|data-bolsa-accion="abrir-ficha"/);
});
