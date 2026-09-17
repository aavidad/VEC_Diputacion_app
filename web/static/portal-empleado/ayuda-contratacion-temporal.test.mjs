import test from "node:test";
import assert from "node:assert/strict";
import {
  AYUDA_PORTAL_BOLSA,
  AYUDA_CONTRATACION_TEMPORAL,
  detectarContextoContratacionTemporal,
  obtenerAyudaContratacionTemporal,
  renderizarAyudaContratacionTemporal,
} from "./ayuda-contenido.js";

test("la ayuda del portal de bolsa preexistente permanece intacta y conforme", () => {
  assert.equal(AYUDA_PORTAL_BOLSA.esquema, "vec.portal.ayuda.v1");
  assert.equal(typeof AYUDA_PORTAL_BOLSA.titulo, "string");
  assert.ok(AYUDA_PORTAL_BOLSA.pasos.length >= 4);
  assert.ok(AYUDA_PORTAL_BOLSA.preguntas.length >= 3);
  assert.ok(AYUDA_PORTAL_BOLSA.audio.src.length > 0);
});

test("la ayuda de contratación temporal cubre todas las vistas del módulo", () => {
  const vistasRequeridas = ["cuadro", "alta", "expediente", "documentos", "auditoria"];
  for (const vista of vistasRequeridas) {
    const item = AYUDA_CONTRATACION_TEMPORAL.vistas[vista];
    assert.ok(item, `falta ayuda para la vista: ${vista}`);
    assert.equal(typeof item.titulo, "string", `título no válido para vista ${vista}`);
    assert.ok(item.titulo.length > 5, `título demasiado corto para vista ${vista}`);
    assert.ok(Array.isArray(item.frases), `frases no es array para vista ${vista}`);
    assert.ok(
      item.frases.length >= 3 && item.frases.length <= 4,
      `la vista ${vista} debe tener entre 3 y 4 frases (tiene ${item.frases.length})`,
    );
    for (const frase of item.frases) {
      assert.equal(typeof frase, "string");
      assert.ok(frase.length > 20, `frase demasiado corta en vista ${vista}: ${frase}`);
      assert.ok(frase.endsWith("."), `la frase debe terminar en punto: ${frase}`);
    }
  }
});

test("la ayuda de contratación temporal cubre los 8 pasos procedimentales de RRHH", () => {
  const fasesRequeridas = [
    { clave: "solicitud", paso: 1 },
    { clave: "analisis_rrhh", paso: 2 },
    { clave: "gestion_bolsa", paso: 3 },
    { clave: "fiscalizacion", paso: 4 },
    { clave: "obtencion_candidato", paso: 5 },
    { clave: "nombramiento", paso: 6 },
    { clave: "incorporacion", paso: 7 },
    { clave: "seguimiento", paso: 8 },
  ];

  for (const { clave, paso } of fasesRequeridas) {
    const item = AYUDA_CONTRATACION_TEMPORAL.fases[clave];
    assert.ok(item, `falta ayuda para la fase: ${clave}`);
    assert.equal(item.paso, paso, `paso incorrecto para fase ${clave}: esperado ${paso}, recibido ${item.paso}`);
    assert.equal(typeof item.titulo, "string", `título no válido para fase ${clave}`);
    assert.ok(item.titulo.includes(String(paso)), `el título debe indicar el paso ${paso}: ${item.titulo}`);
    assert.ok(Array.isArray(item.frases), `frases no es array para fase ${clave}`);
    assert.ok(
      item.frases.length >= 3 && item.frases.length <= 4,
      `la fase ${clave} debe tener entre 3 y 4 frases (tiene ${item.frases.length})`,
    );
    for (const frase of item.frases) {
      assert.equal(typeof frase, "string");
      assert.ok(frase.length > 20, `frase demasiado corta en fase ${clave}: ${frase}`);
      assert.ok(frase.endsWith("."), `la frase debe terminar en punto: ${frase}`);
    }
  }
});

test("todos los textos usan castellano llano, sin jerga técnica ni identificadores y con reglas pendientes explícitas", () => {
  const todasLasEntradas = [
    ...Object.values(AYUDA_CONTRATACION_TEMPORAL.vistas),
    ...Object.values(AYUDA_CONTRATACION_TEMPORAL.fases),
  ];

  // Patrones prohibidos: jerga técnica, identificadores de base de datos o endpoints, interpolaciones rotas
  const patronesProhibidos = [
    /expediente:ct:/i,
    /centro:rpt:/i,
    /sha256/i,
    /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}/i, // UUID
    /\bPOST\b/,
    /\bGET\b/,
    /\bJSON\b/,
    /\bAPI\b/,
    /\bHTTP\b/,
    /\bSQL\b/,
    /\bendpoint\b/i,
    /\bpayload\b/i,
    /\bundefined\b/,
    /\bnull\b/,
    /\bNaN\b/,
    /\[object /i,
    /\{\{?/,
    /\}?}/,
  ];

  for (const entrada of todasLasEntradas) {
    const textoCompleto = [entrada.titulo, ...entrada.frases].join(" ");

    for (const patron of patronesProhibidos) {
      assert.doesNotMatch(
        textoCompleto,
        patron,
        `texto contiene patrón técnico o clave sin resolver (${patron}) en: "${entrada.titulo}"`,
      );
    }

    // Regla obligatoria: donde falte la regla confirmada, decir explícitamente "pendiente de definir por RRHH"
    assert.ok(
      textoCompleto.includes("pendiente de definir por RRHH")
      || textoCompleto.includes("pendientes de definir por RRHH"),
      `la entrada "${entrada.titulo}" debe indicar explícitamente que las reglas están pendientes de definir por RRHH`,
    );
  }
});

test("obtenerAyudaContratacionTemporal resuelve vistas directas y alias", () => {
  assert.equal(obtenerAyudaContratacionTemporal("cuadro"), AYUDA_CONTRATACION_TEMPORAL.vistas.cuadro);
  assert.equal(obtenerAyudaContratacionTemporal("alta"), AYUDA_CONTRATACION_TEMPORAL.vistas.alta);
  assert.equal(obtenerAyudaContratacionTemporal("nueva_peticion"), AYUDA_CONTRATACION_TEMPORAL.vistas.alta);
  assert.equal(obtenerAyudaContratacionTemporal("peticion"), AYUDA_CONTRATACION_TEMPORAL.vistas.alta);
  assert.equal(obtenerAyudaContratacionTemporal("documentos"), AYUDA_CONTRATACION_TEMPORAL.vistas.documentos);
  assert.equal(obtenerAyudaContratacionTemporal("auditoria"), AYUDA_CONTRATACION_TEMPORAL.vistas.auditoria);
  assert.equal(obtenerAyudaContratacionTemporal("desconocida"), AYUDA_CONTRATACION_TEMPORAL.vistas.cuadro);
  assert.equal(obtenerAyudaContratacionTemporal(null), AYUDA_CONTRATACION_TEMPORAL.vistas.cuadro);
});

test("obtenerAyudaContratacionTemporal resuelve fases del expediente con alias canónicos y etiquetas legibles", () => {
  // 1. Solicitud
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "solicitud"), AYUDA_CONTRATACION_TEMPORAL.fases.solicitud);
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "Solicitud"), AYUDA_CONTRATACION_TEMPORAL.fases.solicitud);
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "solicitud_registrada"), AYUDA_CONTRATACION_TEMPORAL.fases.solicitud);

  // 2. Análisis RRHH
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "analisis_rrhh"), AYUDA_CONTRATACION_TEMPORAL.fases.analisis_rrhh);
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "analisis"), AYUDA_CONTRATACION_TEMPORAL.fases.analisis_rrhh);
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "Análisis RRHH"), AYUDA_CONTRATACION_TEMPORAL.fases.analisis_rrhh);

  // 3. Gestión de bolsa
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "gestion_bolsa"), AYUDA_CONTRATACION_TEMPORAL.fases.gestion_bolsa);
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "asignacion"), AYUDA_CONTRATACION_TEMPORAL.fases.gestion_bolsa);
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "asignacion_unidad"), AYUDA_CONTRATACION_TEMPORAL.fases.gestion_bolsa);
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "Gestión de bolsa"), AYUDA_CONTRATACION_TEMPORAL.fases.gestion_bolsa);

  // 4. Fiscalización
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "fiscalizacion"), AYUDA_CONTRATACION_TEMPORAL.fases.fiscalizacion);
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "subsanacion_unidad"), AYUDA_CONTRATACION_TEMPORAL.fases.fiscalizacion);
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "informe_juridico"), AYUDA_CONTRATACION_TEMPORAL.fases.fiscalizacion);
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "Fiscalización"), AYUDA_CONTRATACION_TEMPORAL.fases.fiscalizacion);

  // 5. Obtención del candidato
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "obtencion_candidato"), AYUDA_CONTRATACION_TEMPORAL.fases.obtencion_candidato);
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "llamamiento"), AYUDA_CONTRATACION_TEMPORAL.fases.obtencion_candidato);
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "Obtención del candidato"), AYUDA_CONTRATACION_TEMPORAL.fases.obtencion_candidato);

  // 6. Nombramiento
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "nombramiento"), AYUDA_CONTRATACION_TEMPORAL.fases.nombramiento);
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "Nombramiento"), AYUDA_CONTRATACION_TEMPORAL.fases.nombramiento);

  // 7. Incorporación
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "incorporacion"), AYUDA_CONTRATACION_TEMPORAL.fases.incorporacion);
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "Incorporación"), AYUDA_CONTRATACION_TEMPORAL.fases.incorporacion);

  // 8. Seguimiento
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "seguimiento"), AYUDA_CONTRATACION_TEMPORAL.fases.seguimiento);
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "Seguimiento"), AYUDA_CONTRATACION_TEMPORAL.fases.seguimiento);

  // Sin fase en expediente devuelve la ayuda general de expediente
  assert.equal(obtenerAyudaContratacionTemporal("expediente", null), AYUDA_CONTRATACION_TEMPORAL.vistas.expediente);
  assert.equal(obtenerAyudaContratacionTemporal("expediente", ""), AYUDA_CONTRATACION_TEMPORAL.vistas.expediente);
  assert.equal(obtenerAyudaContratacionTemporal("expediente", "fase_inexistente"), AYUDA_CONTRATACION_TEMPORAL.vistas.expediente);
});

test("renderizarAyudaContratacionTemporal produce marcado semántico accesible y escapado", () => {
  const ayuda = AYUDA_CONTRATACION_TEMPORAL.vistas.cuadro;
  const escapador = (t) => t.replaceAll("&", "&amp;").replaceAll("<", "&lt;");
  const resultado = renderizarAyudaContratacionTemporal(ayuda, escapador);

  assert.equal(resultado.titulo, "Ayuda — Cuadro de mando de contratación temporal");
  assert.match(resultado.contenido, /<section class="ayuda-contextual ayuda-contratacion-temporal">/);
  assert.match(resultado.contenido, /<div class="ayuda-descripcion">/);
  for (const frase of ayuda.frases) {
    assert.match(resultado.contenido, new RegExp(escapador(frase)));
  }
  // Coerción a string
  assert.equal(String(resultado), resultado.contenido);
});

test("detectarContextoContratacionTemporal deduce la vista y fase según los elementos del DOM", () => {
  // Caso 1: navegación indica alta
  const docAlta = {
    querySelector: (selector) => {
      if (selector === ".ct-exp-navegacion button[data-ct-exp-vista][aria-current='page']") {
        return { getAttribute: (attr) => (attr === "data-ct-exp-vista" ? "alta" : null) };
      }
      return null;
    },
    querySelectorAll: () => [],
  };
  assert.deepEqual(detectarContextoContratacionTemporal(docAlta), { vista: "alta", fase: null });

  // Caso 2: navegación indica expediente y raíl marca fase con data-ct-fase
  const docExpedienteFase = {
    querySelector: (selector) => {
      if (selector === ".ct-exp-navegacion button[data-ct-exp-vista][aria-current='page']") {
        return { getAttribute: (attr) => (attr === "data-ct-exp-vista" ? "expediente" : null) };
      }
      if (selector === ".ct-exp-progreso li[aria-current='step']") {
        return {
          getAttribute: (attr) => (attr === "data-ct-fase" ? "fiscalizacion" : null),
          querySelector: () => ({ textContent: "Fiscalización" }),
        };
      }
      return null;
    },
    querySelectorAll: () => [],
  };
  assert.deepEqual(
    detectarContextoContratacionTemporal(docExpedienteFase),
    { vista: "expediente", fase: "fiscalizacion" },
  );

  // Caso 3: navegación indica cuadro
  const docCuadro = {
    querySelector: (selector) => {
      if (selector === ".ct-exp-navegacion button[data-ct-exp-vista][aria-current='page']") {
        return { getAttribute: (attr) => (attr === "data-ct-exp-vista" ? "cuadro" : null) };
      }
      return null;
    },
    querySelectorAll: () => [],
  };
  assert.deepEqual(detectarContextoContratacionTemporal(docCuadro), { vista: "cuadro", fase: null });

  // Caso 4: fallback a cuadro si documento nulo
  assert.deepEqual(detectarContextoContratacionTemporal(null), { vista: "cuadro", fase: null });
});
