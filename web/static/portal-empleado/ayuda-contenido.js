import { traducirPortal } from "./portal-i18n.js?v=20260925-d5d6-v1";

/** Contenido de ayuda sustituible por catálogo o conector, sin lógica de negocio. */
export const AYUDA_PORTAL_BOLSA = Object.freeze({
  esquema: "vec.portal.ayuda.v1",
  titulo: traducirPortal("ayuda_contenido_001"),
  introduccion: traducirPortal("ayuda_contenido_002"),
  pasos: Object.freeze([
    traducirPortal("ayuda_contenido_259"),
    traducirPortal("ayuda_contenido_260"),
    traducirPortal("ayuda_contenido_261"),
    traducirPortal("ayuda_contenido_262"),
  ]),
  preguntas: Object.freeze([
    Object.freeze({
      pregunta: traducirPortal("ayuda_contenido_003"),
      respuesta: traducirPortal("ayuda_contenido_004"),
    }),
    Object.freeze({
      pregunta: traducirPortal("ayuda_contenido_005"),
      respuesta: traducirPortal("ayuda_contenido_006"),
    }),
    Object.freeze({
      pregunta: traducirPortal("ayuda_contenido_007"),
      respuesta: traducirPortal("ayuda_contenido_008"),
    }),
  ]),
  transcripcion: traducirPortal("ayuda_contenido_009"),
});

/**
 * Catálogo local del ayudante. Describe pasos de orientación, no reglas ni
 * operaciones administrativas: cada límite indica de forma expresa qué
 * conector o validación sigue pendiente.
 */
export const TRAMITES_AYUDANTE_PORTAL = Object.freeze([
  Object.freeze({
    id: "dietas-crear-borrador", titulo: traducirPortal("ayuda_contenido_010"), modulo: traducirPortal("ayuda_contenido_011"), vista: "dietas", selector: "[data-dietas-borradores-propios]",
    resumen: traducirPortal("ayuda_contenido_012"),
    pasos: Object.freeze([
      Object.freeze({ selector: "[data-dietas-borradores-propios]", bloqueado: true, titulo: traducirPortal("ayuda_contenido_013"), instruccion: traducirPortal("ayuda_contenido_014"), objetivo: traducirPortal("ayuda_contenido_015"), preparacion: traducirPortal("ayuda_contenido_016"), resultado: traducirPortal("ayuda_contenido_017"), actor: traducirPortal("ayuda_contenido_018"), limite: traducirPortal("ayuda_contenido_019") }),
      Object.freeze({ selector: "[data-dietas-recorridos]", bloqueado: true, titulo: traducirPortal("ayuda_contenido_020"), instruccion: traducirPortal("ayuda_contenido_021"), objetivo: traducirPortal("ayuda_contenido_022"), preparacion: traducirPortal("ayuda_contenido_023"), resultado: traducirPortal("ayuda_contenido_024"), actor: traducirPortal("ayuda_contenido_025"), limite: traducirPortal("ayuda_contenido_026") }),
      Object.freeze({ selector: "[data-dietas-borradores-propios]", bloqueado: true, titulo: traducirPortal("ayuda_contenido_027"), instruccion: traducirPortal("ayuda_contenido_028"), objetivo: traducirPortal("ayuda_contenido_029"), preparacion: traducirPortal("ayuda_contenido_030"), resultado: traducirPortal("ayuda_contenido_031"), actor: traducirPortal("ayuda_contenido_032"), limite: traducirPortal("ayuda_contenido_033") }),
    ]),
  }),
  Object.freeze({
    id: "dietas-consultar-borrador", titulo: traducirPortal("ayuda_contenido_034"), modulo: traducirPortal("ayuda_contenido_035"), vista: "dietas", selector: "[data-dietas-borradores-propios]",
    resumen: traducirPortal("ayuda_contenido_036"),
    pasos: Object.freeze([
      Object.freeze({ selector: "[data-dietas-borradores-propios]", bloqueado: true, titulo: traducirPortal("ayuda_contenido_037"), instruccion: traducirPortal("ayuda_contenido_038"), objetivo: traducirPortal("ayuda_contenido_039"), preparacion: traducirPortal("ayuda_contenido_040"), resultado: traducirPortal("ayuda_contenido_041"), actor: traducirPortal("ayuda_contenido_042"), limite: traducirPortal("ayuda_contenido_043") }),
      Object.freeze({ selector: "[data-dietas-borradores-propios]", bloqueado: true, titulo: traducirPortal("ayuda_contenido_044"), instruccion: traducirPortal("ayuda_contenido_045"), objetivo: traducirPortal("ayuda_contenido_046"), preparacion: traducirPortal("ayuda_contenido_047"), resultado: traducirPortal("ayuda_contenido_048"), actor: traducirPortal("ayuda_contenido_049"), limite: traducirPortal("ayuda_contenido_050") }),
    ]),
  }),
  Object.freeze({
    id: "dietas-ruta", titulo: traducirPortal("ayuda_contenido_051"), modulo: traducirPortal("ayuda_contenido_052"), vista: "dietas", selector: "[data-dietas-area-itinerario]",
    resumen: traducirPortal("ayuda_contenido_053"),
    pasos: Object.freeze([
      Object.freeze({ selector: "[data-dietas-area-itinerario]", bloqueado: false, titulo: traducirPortal("ayuda_contenido_054"), instruccion: traducirPortal("ayuda_contenido_055"), objetivo: traducirPortal("ayuda_contenido_056"), preparacion: traducirPortal("ayuda_contenido_057"), resultado: traducirPortal("ayuda_contenido_058"), actor: traducirPortal("ayuda_contenido_059"), limite: traducirPortal("ayuda_contenido_060") }),
      Object.freeze({ selector: "[data-dietas-area-itinerario]", bloqueado: false, titulo: traducirPortal("ayuda_contenido_061"), instruccion: traducirPortal("ayuda_contenido_062"), objetivo: traducirPortal("ayuda_contenido_063"), preparacion: traducirPortal("ayuda_contenido_064"), resultado: traducirPortal("ayuda_contenido_065"), actor: traducirPortal("ayuda_contenido_066"), limite: traducirPortal("ayuda_contenido_067") }),
    ]),
  }),
  Object.freeze({
    id: "dietas-revisar-documento", titulo: traducirPortal("ayuda_contenido_319"), modulo: traducirPortal("ayuda_contenido_320"), vista: "dietas", selector: "[data-dietas-bandeja-circuito]",
    resumen: traducirPortal("ayuda_contenido_321"),
    pasos: Object.freeze([
      Object.freeze({ selector: "[data-dietas-bandeja-circuito]", bloqueado: false, titulo: traducirPortal("ayuda_contenido_322"), instruccion: traducirPortal("ayuda_contenido_323"), objetivo: traducirPortal("ayuda_contenido_324"), preparacion: traducirPortal("ayuda_contenido_325"), resultado: traducirPortal("ayuda_contenido_326"), actor: traducirPortal("ayuda_contenido_327"), limite: traducirPortal("ayuda_contenido_328") }),
      Object.freeze({ selector: "[data-dietas-bandeja-circuito]", bloqueado: false, titulo: traducirPortal("ayuda_contenido_329"), instruccion: traducirPortal("ayuda_contenido_330"), objetivo: traducirPortal("ayuda_contenido_331"), preparacion: traducirPortal("ayuda_contenido_332"), resultado: traducirPortal("ayuda_contenido_333"), actor: traducirPortal("ayuda_contenido_334"), limite: traducirPortal("ayuda_contenido_335") }),
    ]),
  }),
  Object.freeze({
    id: "cronos-corregir-marcaje", titulo: traducirPortal("ayuda_contenido_068"), modulo: traducirPortal("ayuda_contenido_069"), vista: "cronos", selector: "#cronos-persona",
    resumen: traducirPortal("ayuda_contenido_070"),
    pasos: Object.freeze([
      Object.freeze({ selector: "#cronos-persona", bloqueado: true, titulo: traducirPortal("ayuda_contenido_071"), instruccion: traducirPortal("ayuda_contenido_072"), objetivo: traducirPortal("ayuda_contenido_073"), preparacion: traducirPortal("ayuda_contenido_074"), resultado: traducirPortal("ayuda_contenido_075"), actor: traducirPortal("ayuda_contenido_076"), limite: traducirPortal("ayuda_contenido_077") }),
      Object.freeze({ selector: "#cronos-correccion-ayuda", bloqueado: true, titulo: traducirPortal("ayuda_contenido_078"), instruccion: traducirPortal("ayuda_contenido_079"), objetivo: traducirPortal("ayuda_contenido_080"), preparacion: traducirPortal("ayuda_contenido_081"), resultado: traducirPortal("ayuda_contenido_082"), actor: traducirPortal("ayuda_contenido_083"), limite: traducirPortal("ayuda_contenido_084") }),
      Object.freeze({ selector: "#cronos-responsable", bloqueado: true, titulo: traducirPortal("ayuda_contenido_085"), instruccion: traducirPortal("ayuda_contenido_086"), objetivo: traducirPortal("ayuda_contenido_087"), preparacion: traducirPortal("ayuda_contenido_088"), resultado: traducirPortal("ayuda_contenido_089"), actor: traducirPortal("ayuda_contenido_090"), limite: traducirPortal("ayuda_contenido_091") }),
    ]),
  }),
  Object.freeze({
    id: "cronos-solicitar-permiso", titulo: traducirPortal("ayuda_contenido_092"), modulo: traducirPortal("ayuda_contenido_093"), vista: "cronos", selector: "#cronos-persona",
    resumen: traducirPortal("ayuda_contenido_094"),
    pasos: Object.freeze([
      Object.freeze({ selector: "#cronos-persona", bloqueado: true, titulo: traducirPortal("ayuda_contenido_095"), instruccion: traducirPortal("ayuda_contenido_096"), objetivo: traducirPortal("ayuda_contenido_097"), preparacion: traducirPortal("ayuda_contenido_098"), resultado: traducirPortal("ayuda_contenido_099"), actor: traducirPortal("ayuda_contenido_100"), limite: traducirPortal("ayuda_contenido_101") }),
      Object.freeze({ selector: "#cronos-persona", bloqueado: true, titulo: traducirPortal("ayuda_contenido_102"), instruccion: traducirPortal("ayuda_contenido_103"), objetivo: traducirPortal("ayuda_contenido_104"), preparacion: traducirPortal("ayuda_contenido_105"), resultado: traducirPortal("ayuda_contenido_106"), actor: traducirPortal("ayuda_contenido_107"), limite: traducirPortal("ayuda_contenido_108") }),
      Object.freeze({ selector: "#cronos-responsable", bloqueado: true, titulo: traducirPortal("ayuda_contenido_109"), instruccion: traducirPortal("ayuda_contenido_110"), objetivo: traducirPortal("ayuda_contenido_111"), preparacion: traducirPortal("ayuda_contenido_112"), resultado: traducirPortal("ayuda_contenido_113"), actor: traducirPortal("ayuda_contenido_114"), limite: traducirPortal("ayuda_contenido_115") }),
    ]),
  }),
  Object.freeze({
    id: "cronos-consultar-saldo", titulo: traducirPortal("ayuda_contenido_116"), modulo: traducirPortal("ayuda_contenido_117"), vista: "cronos", selector: "#cronos-persona",
    resumen: traducirPortal("ayuda_contenido_118"),
    pasos: Object.freeze([
      Object.freeze({ selector: "#cronos-persona", bloqueado: true, titulo: traducirPortal("ayuda_contenido_119"), instruccion: traducirPortal("ayuda_contenido_120"), objetivo: traducirPortal("ayuda_contenido_121"), preparacion: traducirPortal("ayuda_contenido_122"), resultado: traducirPortal("ayuda_contenido_123"), actor: traducirPortal("ayuda_contenido_124"), limite: traducirPortal("ayuda_contenido_125") }),
      Object.freeze({ selector: "#cronos-persona", bloqueado: true, titulo: traducirPortal("ayuda_contenido_126"), instruccion: traducirPortal("ayuda_contenido_127"), objetivo: traducirPortal("ayuda_contenido_128"), preparacion: traducirPortal("ayuda_contenido_129"), resultado: traducirPortal("ayuda_contenido_130"), actor: traducirPortal("ayuda_contenido_131"), limite: traducirPortal("ayuda_contenido_132") }),
    ]),
  }),
  Object.freeze({
    id: "personal-consultar-ficha", titulo: traducirPortal("ayuda_contenido_133"), modulo: traducirPortal("ayuda_contenido_134"), vista: "personal", selector: "[data-personal-ficha-integral]",
    resumen: traducirPortal("ayuda_contenido_135"),
    pasos: Object.freeze([
      Object.freeze({ selector: '[data-personal-ficha-tab="ficha"]', activar: '[data-personal-ficha-tab="ficha"]', bloqueado: true, titulo: traducirPortal("ayuda_contenido_136"), instruccion: traducirPortal("ayuda_contenido_137"), objetivo: traducirPortal("ayuda_contenido_138"), preparacion: traducirPortal("ayuda_contenido_139"), resultado: traducirPortal("ayuda_contenido_140"), actor: traducirPortal("ayuda_contenido_141"), limite: traducirPortal("ayuda_contenido_142") }),
      Object.freeze({ selector: "[data-personal-ficha-integral]", bloqueado: true, titulo: traducirPortal("ayuda_contenido_143"), instruccion: traducirPortal("ayuda_contenido_144"), objetivo: traducirPortal("ayuda_contenido_145"), preparacion: traducirPortal("ayuda_contenido_146"), resultado: traducirPortal("ayuda_contenido_147"), actor: traducirPortal("ayuda_contenido_148"), limite: traducirPortal("ayuda_contenido_149") }),
    ]),
  }),
  Object.freeze({
    id: "personal-relaciones", titulo: traducirPortal("ayuda_contenido_150"), modulo: traducirPortal("ayuda_contenido_151"), vista: "personal", selector: '[data-personal-ficha-tab="relaciones"]', activar: '[data-personal-ficha-tab="relaciones"]',
    resumen: traducirPortal("ayuda_contenido_152"),
    pasos: Object.freeze([
      Object.freeze({ selector: '[data-personal-ficha-tab="relaciones"]', activar: '[data-personal-ficha-tab="relaciones"]', bloqueado: true, titulo: traducirPortal("ayuda_contenido_153"), instruccion: traducirPortal("ayuda_contenido_154"), objetivo: traducirPortal("ayuda_contenido_155"), preparacion: traducirPortal("ayuda_contenido_156"), resultado: traducirPortal("ayuda_contenido_157"), actor: traducirPortal("ayuda_contenido_158"), limite: traducirPortal("ayuda_contenido_159") }),
      Object.freeze({ selector: '[data-personal-ficha-tab="servicios"]', activar: '[data-personal-ficha-tab="servicios"]', bloqueado: true, titulo: traducirPortal("ayuda_contenido_160"), instruccion: traducirPortal("ayuda_contenido_161"), objetivo: traducirPortal("ayuda_contenido_162"), preparacion: traducirPortal("ayuda_contenido_163"), resultado: traducirPortal("ayuda_contenido_164"), actor: traducirPortal("ayuda_contenido_165"), limite: traducirPortal("ayuda_contenido_166") }),
    ]),
  }),
  Object.freeze({
    id: "personal-catalogos", titulo: traducirPortal("ayuda_contenido_167"), modulo: traducirPortal("ayuda_contenido_168"), vista: "personal", selector: '[data-personal-ficha-tab="catalogos"]', activar: '[data-personal-ficha-tab="catalogos"]',
    resumen: traducirPortal("ayuda_contenido_169"),
    pasos: Object.freeze([
      Object.freeze({ selector: '[data-personal-ficha-tab="catalogos"]', activar: '[data-personal-ficha-tab="catalogos"]', bloqueado: false, titulo: traducirPortal("ayuda_contenido_170"), instruccion: traducirPortal("ayuda_contenido_171"), objetivo: traducirPortal("ayuda_contenido_172"), preparacion: traducirPortal("ayuda_contenido_173"), resultado: traducirPortal("ayuda_contenido_174"), actor: traducirPortal("ayuda_contenido_175"), limite: traducirPortal("ayuda_contenido_176") }),
      Object.freeze({ selector: "[data-personal-ficha-integral]", bloqueado: false, titulo: traducirPortal("ayuda_contenido_177"), instruccion: traducirPortal("ayuda_contenido_178"), objetivo: traducirPortal("ayuda_contenido_179"), preparacion: traducirPortal("ayuda_contenido_180"), resultado: traducirPortal("ayuda_contenido_181"), actor: traducirPortal("ayuda_contenido_182"), limite: traducirPortal("ayuda_contenido_183") }),
    ]),
  }),
  Object.freeze({
    id: "bolsa-consultar-candidatos", titulo: traducirPortal("ayuda_contenido_184"), modulo: traducirPortal("ayuda_contenido_185"), vista: "resumen", selector: "#titulo-cuadro-b12",
    resumen: traducirPortal("ayuda_contenido_186"),
    pasos: Object.freeze([
      Object.freeze({ vista: "resumen", selector: "#titulo-cuadro-b12", bloqueado: false, titulo: traducirPortal("ayuda_contenido_187"), instruccion: traducirPortal("ayuda_contenido_188"), objetivo: traducirPortal("ayuda_contenido_189"), preparacion: traducirPortal("ayuda_contenido_190"), resultado: traducirPortal("ayuda_contenido_191"), actor: traducirPortal("ayuda_contenido_192"), limite: traducirPortal("ayuda_contenido_193") }),
      Object.freeze({ vista: "resumen", selector: '[data-accion="ver-bolsa"][data-bolsa-ref]', bloqueado: false, titulo: traducirPortal("ayuda_contenido_194"), instruccion: traducirPortal("ayuda_contenido_195"), objetivo: traducirPortal("ayuda_contenido_196"), preparacion: traducirPortal("ayuda_contenido_197"), resultado: traducirPortal("ayuda_contenido_198"), actor: traducirPortal("ayuda_contenido_199"), limite: traducirPortal("ayuda_contenido_200") }),
      Object.freeze({ vista: "bolsa-candidatos", selector: '[data-bolsa-form="filtros"]', bloqueado: false, titulo: traducirPortal("ayuda_contenido_201"), instruccion: traducirPortal("ayuda_contenido_202"), objetivo: traducirPortal("ayuda_contenido_203"), preparacion: traducirPortal("ayuda_contenido_204"), resultado: traducirPortal("ayuda_contenido_205"), actor: traducirPortal("ayuda_contenido_206"), limite: traducirPortal("ayuda_contenido_207") }),
      Object.freeze({ vista: "bolsa-candidatos", selector: '[data-bolsa-accion="abrir-ficha"]', bloqueado: false, titulo: traducirPortal("ayuda_contenido_208"), instruccion: traducirPortal("ayuda_contenido_209"), objetivo: traducirPortal("ayuda_contenido_210"), preparacion: traducirPortal("ayuda_contenido_211"), resultado: traducirPortal("ayuda_contenido_212"), actor: traducirPortal("ayuda_contenido_213"), limite: traducirPortal("ayuda_contenido_214") }),
    ]),
  }),
  Object.freeze({
    id: "bolsa-gestionar-llamamiento", titulo: traducirPortal("ayuda_contenido_215"), modulo: traducirPortal("ayuda_contenido_216"), vista: "resumen", selector: "#titulo-cuadro-b12",
    resumen: traducirPortal("ayuda_contenido_217"),
    pasos: Object.freeze([
      Object.freeze({ vista: "resumen", selector: "#titulo-cuadro-b12", bloqueado: false, titulo: traducirPortal("ayuda_contenido_218"), instruccion: traducirPortal("ayuda_contenido_219"), objetivo: traducirPortal("ayuda_contenido_220"), preparacion: traducirPortal("ayuda_contenido_221"), resultado: traducirPortal("ayuda_contenido_222"), actor: traducirPortal("ayuda_contenido_223"), limite: traducirPortal("ayuda_contenido_224") }),
      Object.freeze({ vista: "resumen", selector: '[data-accion="ver-bolsa"][data-bolsa-ref]', bloqueado: false, titulo: traducirPortal("ayuda_contenido_225"), instruccion: traducirPortal("ayuda_contenido_226"), objetivo: traducirPortal("ayuda_contenido_227"), preparacion: traducirPortal("ayuda_contenido_228"), resultado: traducirPortal("ayuda_contenido_229"), actor: traducirPortal("ayuda_contenido_230"), limite: traducirPortal("ayuda_contenido_231") }),
      Object.freeze({ vista: "bolsa-candidatos", selector: '[data-bolsa-form="filtros"]', bloqueado: false, titulo: traducirPortal("ayuda_contenido_232"), instruccion: traducirPortal("ayuda_contenido_233"), objetivo: traducirPortal("ayuda_contenido_234"), preparacion: traducirPortal("ayuda_contenido_235"), resultado: traducirPortal("ayuda_contenido_236"), actor: traducirPortal("ayuda_contenido_237"), limite: traducirPortal("ayuda_contenido_238") }),
      Object.freeze({ vista: "bolsa-candidatos", selector: "[data-bolsa-c23-pendiente]", bloqueado: true, titulo: traducirPortal("ayuda_contenido_239"), instruccion: traducirPortal("ayuda_contenido_240"), objetivo: traducirPortal("ayuda_contenido_241"), preparacion: traducirPortal("ayuda_contenido_242"), resultado: traducirPortal("ayuda_contenido_243"), actor: traducirPortal("ayuda_contenido_244"), limite: traducirPortal("ayuda_contenido_245") }),
    ]),
  }),
]);

/** Ayuda contextual del módulo de contratación temporal para Recursos Humanos. */
export const AYUDA_CONTRATACION_TEMPORAL = Object.freeze({
  vistas: Object.freeze({
    cuadro: Object.freeze({
      titulo: traducirPortal("ayuda_contenido_246"),
      frases: Object.freeze([
        traducirPortal("ayuda_contenido_263"),
        traducirPortal("ayuda_contenido_264"),
        traducirPortal("ayuda_contenido_265"),
      ]),
    }),
    alta: Object.freeze({
      titulo: traducirPortal("ayuda_contenido_247"),
      frases: Object.freeze([
        traducirPortal("ayuda_contenido_266"),
        traducirPortal("ayuda_contenido_267"),
        traducirPortal("ayuda_contenido_268"),
      ]),
    }),
    expediente: Object.freeze({
      titulo: traducirPortal("ayuda_contenido_248"),
      frases: Object.freeze([
        traducirPortal("ayuda_contenido_269"),
        traducirPortal("ayuda_contenido_270"),
        traducirPortal("ayuda_contenido_271"),
      ]),
    }),
    documentos: Object.freeze({
      titulo: traducirPortal("ayuda_contenido_249"),
      frases: Object.freeze([
        traducirPortal("ayuda_contenido_272"),
        traducirPortal("ayuda_contenido_273"),
        traducirPortal("ayuda_contenido_274"),
      ]),
    }),
    auditoria: Object.freeze({
      titulo: traducirPortal("ayuda_contenido_250"),
      frases: Object.freeze([
        traducirPortal("ayuda_contenido_275"),
        traducirPortal("ayuda_contenido_276"),
        traducirPortal("ayuda_contenido_277"),
      ]),
    }),
  }),
  fases: Object.freeze({
    solicitud: Object.freeze({
      paso: 1,
      titulo: traducirPortal("ayuda_contenido_251"),
      frases: Object.freeze([
        traducirPortal("ayuda_contenido_278"),
        traducirPortal("ayuda_contenido_279"),
        traducirPortal("ayuda_contenido_280"),
      ]),
    }),
    analisis_rrhh: Object.freeze({
      paso: 2,
      titulo: traducirPortal("ayuda_contenido_252"),
      frases: Object.freeze([
        traducirPortal("ayuda_contenido_281"),
        traducirPortal("ayuda_contenido_282"),
        traducirPortal("ayuda_contenido_283"),
      ]),
    }),
    gestion_bolsa: Object.freeze({
      paso: 3,
      titulo: traducirPortal("ayuda_contenido_253"),
      frases: Object.freeze([
        traducirPortal("ayuda_contenido_284"),
        traducirPortal("ayuda_contenido_285"),
        traducirPortal("ayuda_contenido_286"),
      ]),
    }),
    fiscalizacion: Object.freeze({
      paso: 4,
      titulo: traducirPortal("ayuda_contenido_254"),
      frases: Object.freeze([
        traducirPortal("ayuda_contenido_287"),
        traducirPortal("ayuda_contenido_288"),
        traducirPortal("ayuda_contenido_289"),
      ]),
    }),
    obtencion_candidato: Object.freeze({
      paso: 5,
      titulo: traducirPortal("ayuda_contenido_255"),
      frases: Object.freeze([
        traducirPortal("ayuda_contenido_290"),
        traducirPortal("ayuda_contenido_291"),
        traducirPortal("ayuda_contenido_292"),
      ]),
    }),
    nombramiento: Object.freeze({
      paso: 6,
      titulo: traducirPortal("ayuda_contenido_256"),
      frases: Object.freeze([
        traducirPortal("ayuda_contenido_293"),
        traducirPortal("ayuda_contenido_294"),
        traducirPortal("ayuda_contenido_295"),
      ]),
    }),
    incorporacion: Object.freeze({
      paso: 7,
      titulo: traducirPortal("ayuda_contenido_257"),
      frases: Object.freeze([
        traducirPortal("ayuda_contenido_296"),
        traducirPortal("ayuda_contenido_297"),
        traducirPortal("ayuda_contenido_298"),
      ]),
    }),
    seguimiento: Object.freeze({
      paso: 8,
      titulo: traducirPortal("ayuda_contenido_258"),
      frases: Object.freeze([
        traducirPortal("ayuda_contenido_299"),
        traducirPortal("ayuda_contenido_300"),
        traducirPortal("ayuda_contenido_301"),
      ]),
    }),
  }),
});

const EQUIVALENCIAS_FASES_AYUDA = Object.freeze({
  solicitud: "solicitud",
  solicitud_registrada: "solicitud",
  analisis: "analisis_rrhh",
  analisis_rrhh: "analisis_rrhh",
  gestion_bolsa: "gestion_bolsa",
  asignacion: "gestion_bolsa",
  asignacion_unidad: "gestion_bolsa",
  fiscalizacion: "fiscalizacion",
  informe_juridico: "fiscalizacion",
  subsanacion_unidad: "fiscalizacion",
  obtencion_candidato: "obtencion_candidato",
  llamamiento: "obtencion_candidato",
  nombramiento: "nombramiento",
  incorporacion: "incorporacion",
  seguimiento: "seguimiento",
});

function normalizarClaveFase(fase) {
  if (typeof fase !== "string") return null;
  const limpia = fase.trim().toLowerCase()
    .normalize("NFD").replace(/[\u0300-\u036f]/g, "")
    .replaceAll(" ", "_")
    .replaceAll("-", "_");
  if (limpia in EQUIVALENCIAS_FASES_AYUDA) return EQUIVALENCIAS_FASES_AYUDA[limpia];
  if (limpia.includes("solicitud")) return "solicitud";
  if (limpia.includes("analisis")) return "analisis_rrhh";
  if (limpia.includes("bolsa") || limpia.includes("asignacion")) return "gestion_bolsa";
  if (limpia.includes("fiscaliz") || limpia.includes("subsanac") || limpia.includes("juridic")) return "fiscalizacion";
  if (limpia.includes("candidat") || limpia.includes("llamamiento")) return "obtencion_candidato";
  if (limpia.includes("nombramiento") || limpia.includes("formaliz")) return "nombramiento";
  if (limpia.includes("incorporac")) return "incorporacion";
  if (limpia.includes("seguimiento")) return "seguimiento";
  return null;
}

export function detectarContextoContratacionTemporal(doc = (typeof document !== "undefined" ? document : null)) {
  if (!doc) return { vista: "cuadro", fase: null };
  const botonVista = doc.querySelector?.(".ct-exp-navegacion button[data-ct-exp-vista][aria-current='page']");
  let vista = botonVista?.getAttribute?.("data-ct-exp-vista") || null;
  if (!vista) {
    if (doc.querySelector?.("form[data-ct-alta-formulario]")) vista = "alta";
    else if (doc.querySelector?.(".ct-exp-cabecera-expediente")) vista = "expediente";
    else vista = "cuadro";
  }
  let fase = null;
  if (vista === "expediente") {
    const paso = doc.querySelector?.(".ct-exp-progreso li[aria-current='step']")
      || doc.querySelector?.(".ct-exp-progreso li.en_curso")
      || doc.querySelector?.(".ct-exp-progreso li.incidencia");
    fase = paso?.getAttribute?.("data-ct-fase")
      || paso?.querySelector?.("span:not(.ct-exp-numero-fase)")?.textContent?.trim()
      || null;
    if (!fase) {
      const dts = doc.querySelectorAll?.(".ct-exp-cabecera-expediente dt") || [];
      for (const dt of dts) {
        if (dt.textContent?.trim().toLowerCase().includes("fase actual")) {
          fase = dt.nextElementSibling?.textContent?.trim() || null;
          break;
        }
      }
    }
  }
  return { vista, fase };
}

export function obtenerAyudaContratacionTemporal(vista = "cuadro", fase = null) {
  const vistaLimpia = typeof vista === "string" ? vista.trim().toLowerCase() : "cuadro";
  if (vistaLimpia === "expediente") {
    const claveFase = normalizarClaveFase(fase);
    if (claveFase && AYUDA_CONTRATACION_TEMPORAL.fases[claveFase]) {
      return AYUDA_CONTRATACION_TEMPORAL.fases[claveFase];
    }
    return AYUDA_CONTRATACION_TEMPORAL.vistas.expediente;
  }
  if (vistaLimpia in AYUDA_CONTRATACION_TEMPORAL.vistas) {
    return AYUDA_CONTRATACION_TEMPORAL.vistas[vistaLimpia];
  }
  if (vistaLimpia === "nueva_peticion" || vistaLimpia === "peticion") {
    return AYUDA_CONTRATACION_TEMPORAL.vistas.alta;
  }
  return AYUDA_CONTRATACION_TEMPORAL.vistas.cuadro;
}

export function renderizarAyudaContratacionTemporal(ayuda, escapar = (s) => s) {
  if (!ayuda || !Array.isArray(ayuda.frases)) return "";
  const contenido = `<section class="ayuda-contextual ayuda-contratacion-temporal" tabindex="-1">
  <div class="ayuda-descripcion">
${ayuda.frases.map((frase) => `    <p>${escapar(frase)}</p>`).join("\n")}
  </div>
</section>`;
  return {
    titulo: ayuda.titulo,
    contenido,
    toString() { return this.contenido; },
  };
}
