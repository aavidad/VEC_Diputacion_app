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
import { crearFuenteLecturaBolsasPresentacion } from "./portal-presentacion-adaptador.js";
import { obtenerDatosPresentacion } from "./datos-presentacion.js";

const rutaDemoJson = new URL("../../../data/demo/bolsa/v1.bolsas-demo.json", import.meta.url);
const demoJsonRaw = JSON.parse(await readFile(rutaDemoJson, "utf8"));

/**
 * Función que mapea los estados sintéticos de demo al catálogo cerrado de SituacionParticipacionBolsa:
 * trabajando -> ocupado
 * pendiente_incorporacion -> ocupado
 * disponible_desde -> no_disponible (con fecha disponible_desde)
 * renuncia -> renuncia_pendiente
 * disponible -> disponible
 * no_disponible -> no_disponible
 * excluido -> excluido
 */
function mapearSituacion(estadoClave) {
  switch (estadoClave) {
    case "trabajando":
    case "pendiente_incorporacion":
      return "ocupado";
    case "disponible_desde":
    case "no_disponible":
      return "no_disponible";
    case "renuncia":
      return "renuncia_pendiente";
    case "disponible":
      return "disponible";
    case "excluido":
      return "excluido";
    default:
      return "no_disponible";
  }
}

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
      ocupado: 0,
      no_disponible: 0,
      excluido: 0,
      renuncia_pendiente: 0,
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
        resultado: situacion === "ocupado" ? "aceptado" : "sin_respuesta",
      };
    }
    return {
      participacion_ref: `part_${c.candidatura_ref.replace(":demo:", ":sintetico:").replace(/[^a-zA-Z0-9]/g, "_")}`,
      orden: c.orden,
      nombre_visible: c.nombre_visible,
      documento_enmascarado: c.documento_enmascarado,
      estado_clave: situacion,
      estado_desde: new Date(c.estado_desde).toISOString(),
      disponible_desde: c.disponible_desde ? new Date(c.disponible_desde).toISOString() : null,
      ultimo_llamamiento: ultimoLlamamiento,
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
    assert.equal(opciones.credentials, "omit");
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
    assert.equal(opciones.credentials, "omit");
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
  assert.match(htmlListo, /Ver candidatos/);
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
  assert.match(htmlB5, /Recorrido de gestión de candidatos/);
  assert.match(htmlB5, /Resumen de la bolsa/);
  assert.match(htmlB5, /Consultar historial de contactos/);
  assert.match(htmlB5, /Nuevo llamamiento/);
  assert.match(htmlB5, /Registrar resultado/);
  assert.match(htmlB5, /dependen de C23/);
  assert.match(htmlB5, /disabled aria-disabled="true" title="Pendiente de composición C23"/);

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

test("presentadorPanelInterno muestra ficha B5 solo con los campos del contrato de candidatos", () => {
  const { envelopeCandidatos } = construirFixturesDesdeDemo();
  const datos = validarRespuestaCandidatosBolsa(envelopeCandidatos);
  const candidato = datos.candidatos[0];
  const presentador = crearPresentadorPanelInterno({
    claseEstado: (c) => `chip-${c}`,
    encabezadoVista: (_s, t, d, a = "") => `<header><h2>${t}</h2><p>${d}</p>${a}</header>`,
    escaparHTML: (v) => String(v ?? ""),
    numero: (n) => String(n ?? 0),
    obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }),
    tituloVista: (v) => v,
    obtenerDatosCandidatosBolsa: () => ({ carga: "listo", datos, error: "" }),
    obtenerEstadoCandidatos: () => ({ estado: "", texto: "" }),
    obtenerModalFicha: () => ({ abierto: true, candidato, bolsa: datos.bolsa }),
  });

  const html = presentador.renderizarVista("bolsa-candidatos");
  assert.match(html, /Ficha de participación/);
  assert.match(html, /Referencia de participación/);
  assert.match(html, new RegExp(candidato.participacion_ref));
  assert.match(html, /data-bolsa-accion="cerrar-ficha"/);
  const ficha = html.slice(html.indexOf('id="titulo-modal-ficha"'));
  assert.doesNotMatch(ficha, /correo|teléfono|puntuación|relación laboral/i);
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
    canal: "telefono",
    realizado_en: "2026-09-17T10:30:00Z",
    resultado_clave: "aceptado",
    anotacion: "Acepta incorporación inmediata",
  };

  const validado = validarContacto(contactoValido);
  assert.equal(validado.contacto_ref, "contacto:sintetico:001");
  assert.equal(validado.canal, "telefono");
  assert.equal(validado.resultado_clave, "aceptado");

  // Falla si canal no es válido
  assert.throws(() => validarContacto({ ...contactoValido, canal: "paloma_mensajera" }), /canal de contacto no reconocido/);

  // Falla si resultado_clave no es válido
  assert.throws(() => validarContacto({ ...contactoValido, resultado_clave: "indeciso" }), /resultado_clave de contacto no reconocido/);

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
    assert.match(url, /\/api\/vec\/bolsa\/candidatos\/part_123\/contactos/);
    assert.equal(opciones.credentials, "omit");
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
              canal: "sede",
              realizado_en: "2026-09-17T10:00:00Z",
              resultado_clave: "pendiente",
              anotacion: "Notificación telemática enviada",
            },
          ],
        },
      }),
    };
  };

  const resOk = await consultarContactosCandidato("part_123", { fetchImpl: mockFetchOk });
  assert.equal(resOk.ok, true);
  assert.equal(resOk.datos.contactos.length, 1);
  assert.equal(resOk.datos.contactos[0].canal, "sede");

  const mockFetchDenegado = async () => ({
    ok: false,
    status: 403,
  });
  const resDenegado = await consultarContactosCandidato("part_123", { fetchImpl: mockFetchDenegado });
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
          estado_clave: "ocupado",
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
  assert.equal(llamadaLlamar.opciones.credentials, "omit");
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
  const bodyResultado = JSON.parse(llamadaResultado.opciones.body);
  assert.equal(bodyResultado.esquema, ESQUEMA_ACCION_BOLSA);
  assert.equal(bodyResultado.accion, "registrar_resultado");
  assert.equal(bodyResultado.confirmacion, true);
  assert.equal(bodyResultado.payload.resultado_clave, "aceptado");
});

test("interfaz B5: deja solo la ficha mientras contactos y efectos no están compuestos", () => {
  const { envelopeCandidatos } = construirFixturesDesdeDemo();
  const datosCandidatosValidados = validarRespuestaCandidatosBolsa(envelopeCandidatos);

  let modalContactos = null;
  let modalLlamar = null;
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
    obtenerModalLlamar: () => modalLlamar,
    obtenerModalResultado: () => modalResultado,
  });

  const html = presentador.renderizarVista("bolsa-candidatos");

  // Columna Acciones en cabecera
  assert.match(html, /<th scope="col">Acciones<\/th>/);
  assert.match(html, /data-bolsa-accion="abrir-ficha"/);
  assert.match(html, /Acciones pendientes de composición/);
  assert.match(html, /dependen de C23/);
  assert.match(html, /Consultar historial de contactos/);
  assert.match(html, /Nuevo llamamiento/);
  assert.match(html, /Registrar resultado/);
  assert.match(html, /disabled aria-disabled="true" title="Pendiente de composición C23"/);
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

  // Modal de llamar (B7)
  modalContactos = null;
  modalLlamar = {
    abierto: true,
    participacionRef: "part_demo_1",
    nombreVisible: "Aspirante de Prueba",
    orden: 3,
    carga: "ocioso",
    error: "",
  };
  const htmlConLlamar = presentador.renderizarVista("bolsa-candidatos");
  assert.doesNotMatch(htmlConLlamar, /Nuevo llamamiento \(B7\)|data-bolsa-form="llamar"|llamar-canal/);

  // Modal de resultado (B3)
  modalLlamar = null;
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

test("la presentación reutiliza B12/B5 con envelopes cerrados, filtros en memoria y contactos sintéticos", () => {
  const fuente = crearFuenteLecturaBolsasPresentacion({
    datosIniciales: obtenerDatosPresentacion("tecnico"),
  });
  const bolsas = fuente.consultarBolsas();
  assert.equal(bolsas.ok, true);
  assert.equal(bolsas.datos.bolsas.length, 6);

  const bolsaRef = bolsas.datos.bolsas[0].bolsa_ref;
  assert.equal(bolsaRef, "DEMO-BOL-AUXILIAR-ADMIN");
  assert.equal(bolsas.datos.bolsas[0].tipo_lista, "Pendiente de confirmar");
  const disponibles = fuente.consultarCandidatosBolsa(bolsaRef, { estado: "disponible" });
  assert.equal(disponibles.ok, true);
  assert.equal(disponibles.datos.candidatos.length, 1);
  assert.match(disponibles.datos.candidatos[0].documento_enmascarado, /^\*{3}\d{4}\*{2}$/);
  assert.equal(disponibles.datos.candidatos[0].ultimo_llamamiento, null);

  const vacio = fuente.consultarCandidatosBolsa(bolsaRef, { texto: "sin coincidencia" });
  assert.equal(vacio.ok, true);
  assert.equal(vacio.datos.candidatos.length, 0);

  const historial = fuente.consultarContactosCandidato(disponibles.datos.candidatos[0].participacion_ref);
  assert.equal(historial.ok, true);
  assert.deepEqual(historial.datos.contactos, []);

  const presentador = crearPresentadorPanelInterno({
    claseEstado: () => "", encabezadoVista: (_s, t) => `<h2>${t}</h2>`, escaparHTML: (v) => String(v ?? ""),
    numero: (n) => String(n ?? 0), obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }),
    tituloVista: (v) => v, obtenerDatosCandidatosBolsa: () => ({ carga: "listo", datos: disponibles.datos, error: "" }),
    obtenerEstadoCandidatos: () => ({ estado: "", texto: "" }), esLecturaPresentacion: () => true,
  });
  const html = presentador.renderizarVista("bolsa-candidatos");
  assert.match(html, /Presentación sintética de solo lectura/);
  assert.match(html, /Acciones pendientes de composición/);
  assert.match(html, /abrir-ficha/);
  assert.doesNotMatch(html, /abrir-contactos|abrir-llamar|abrir-resultado/);
});
