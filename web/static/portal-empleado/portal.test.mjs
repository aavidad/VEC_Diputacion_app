import assert from "node:assert/strict";
import { readFile, stat } from "node:fs/promises";
import test from "node:test";
import {
  extraerDatosEnvelopeCanonico,
  validarPanelBolsa,
} from "./portal-contrato.js";
import { validarPropuestaLlamamientoPresentacion } from "./portal-llamamientos-contrato.js";
import { obtenerDatosPresentacion, obtenerPropuestaPresentacion } from "./datos-presentacion.js";
import { AYUDA_PORTAL_BOLSA } from "./ayuda-contenido.js";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js";

const directorio = new URL("./", import.meta.url);
const [html, manifiestoProduccion, javascript, eventos, contrato, contratoLlamamientos, apiLlamamientos, flujoLlamamientos, vistaLlamamientos, panelInterno, resumenPresentacion, datos, ayuda, estilosBase, estilosComponentes, estilosFlujos, estilosCapacidades] = await Promise.all([
  readFile(new URL("index.html", directorio), "utf8"),
  readFile(new URL("../../produccion.manifest", directorio), "utf8"),
  readFile(new URL("portal.js", directorio), "utf8"),
  readFile(new URL("portal-eventos.js", directorio), "utf8"),
  readFile(new URL("portal-contrato.js", directorio), "utf8"),
  readFile(new URL("portal-llamamientos-contrato.js", directorio), "utf8"),
  readFile(new URL("portal-llamamientos-api.js", directorio), "utf8"),
  readFile(new URL("portal-llamamientos-flujo.js", directorio), "utf8"),
  readFile(new URL("portal-llamamientos-vista.js", directorio), "utf8"),
  readFile(new URL("portal-panel-interno.js", directorio), "utf8"),
  readFile(new URL("portal-resumen-presentacion.js", directorio), "utf8"),
  readFile(new URL("datos-presentacion.js", directorio), "utf8"),
  readFile(new URL("ayuda-contenido.js", directorio), "utf8"),
  readFile(new URL("portal.css", directorio), "utf8"),
  readFile(new URL("portal-componentes.css", directorio), "utf8"),
  readFile(new URL("portal-flujos.css", directorio), "utf8"),
  readFile(new URL("portal-capacidades.css", directorio), "utf8"),
]);
const codigo = `${javascript}\n${eventos}\n${contrato}\n${contratoLlamamientos}\n${apiLlamamientos}\n${flujoLlamamientos}\n${vistaLlamamientos}\n${panelInterno}\n${resumenPresentacion}`;
const estilos = `${estilosBase}\n${estilosComponentes}\n${estilosFlujos}\n${estilosCapacidades}`;

function panelInternoReal() {
  return {
    esquema: "vec.bolsa.panel.interno.v1",
    selector: { clase: "organizacion", organizacion_ref: "org_0123456789abcdef" },
    origen: {
      revision: "rev_0123456789abcdef",
      actualizada_en: "2026-07-17T08:59:00Z",
      demostracion: false,
    },
    prueba_lectura: {
      lectura_ref: "lec_0123456789abcdef",
      auditoria_ref: "aud_0123456789abcdef",
      auditoria_secuencia: 17,
      decision_ref: "dec_0123456789abcdef",
      huella_decision_sha256: "a".repeat(64),
      correlacion_ref: "cor_0123456789abcdef",
      confirmada_en: "2026-07-17T09:00:00Z",
    },
    indicadores: {
      convocatorias_borrador: 2,
      convocatorias_revision: 1,
      convocatorias_pendientes_firma: 1,
      convocatorias_publicadas: 4,
      bolsas_activas: 3,
      bolsas_suspendidas: 0,
      bolsas_agotadas: 0,
      llamamientos_pendientes: 5,
      llamamientos_en_curso: 2,
      llamamientos_vencen_hoy: 1,
      documentos_pendientes_firma: 3,
      incidencias_abiertas: 1,
    },
    convocatorias: [{
      convocatoria_ref: "cnv_0123456789abcdef",
      categoria_clave: "auxiliar_administrativo",
      estado_clave: "revision",
      plazo_cierra_en: "2026-07-19T09:00:00Z",
      numero_solicitudes: 120,
      numero_pendientes: 7,
    }],
    actuaciones_pendientes: [{
      actuacion_ref: "act_0123456789abcdef",
      recurso_ref: "cnv_0123456789abcdef",
      tipo_clave: "revisar_bases",
      estado_clave: "pendiente",
      prioridad_clave: "alta",
      fecha_limite: "2026-07-18T09:00:00Z",
      numero_elementos: 1,
    }],
  };
}

test("la ruta normal usa API protegida sin cookies y no cae a datos sintéticos", () => {
  assert.match(javascript, /const API_PANEL_BOLSA = "\/api\/vec\/bolsa\/panel"/);
  assert.equal(`${javascript}\n${apiLlamamientos}`.match(/credentials: "omit"/g)?.length, 2, "todas las llamadas internas deben omitir cookies");
  assert.doesNotMatch(`${javascript}\n${apiLlamamientos}`, /credentials: "(?:same-origin|include)"/);
  assert.doesNotMatch(javascript, /document\.cookie|localStorage.*(?:token|sesion|auth)/i);
  assert.match(javascript, /extraerDatosEnvelopeCanonico\(envelope\)/);
  assert.match(contrato, /la API interna no puede responder con datos de demostración/);
  assert.match(javascript, /if \(respuesta\.status === 401\)/);
  assert.match(javascript, /if \(respuesta\.status === 403\)/);
  assert.match(javascript, /let DATOS_PANEL = DATOS_VACIOS/);
  assert.match(javascript, /superficieBorradores\.comprobarDisponibilidad\(\)/);
  assert.match(javascript, /superficieBorradores\.obtenerAcceso\(\)/);
  assert.doesNotMatch(javascript, /resolverAcceso\(clave, estado\.fuenteLista\)/);
  assert.doesNotMatch(codigo, /María Pérez|García López|Auxiliar Administrativo|BOL-2026|CON-2026|DOC-[A-Z]{2}|20\/07\/2026/);
});

test("el contrato real exige envelope canónico y rechaza una raíz raw", () => {
  const panel = panelInternoReal();
  assert.throws(() => extraerDatosEnvelopeCanonico(panel), /envelope canónico/);
  assert.deepEqual(extraerDatosEnvelopeCanonico({ data: panel }), panel);
  const validado = validarPanelBolsa(extraerDatosEnvelopeCanonico({ data: panel }));
  assert.equal(validado.esquema, "vec.bolsa.panel.interno.v1");
  assert.equal(validado.convocatorias[0].numero_pendientes, 7);
  assert.equal(validado.actuaciones_pendientes[0].tipo_clave, "revisar_bases");
});

test("el panel global prohíbe candidatos y la propuesta es un contrato separado", () => {
  const panel = { ...panelInternoReal(), candidatos: [] };
  assert.throws(() => validarPanelBolsa(panel), /no admite listados/);
  assert.doesNotMatch(datos, /\bcandidatos\s*:/);
  assert.doesNotMatch(datos, /\bdni\s*:/i);
  assert.doesNotMatch(codigo, /data-candidato|Nombre o DNI parcial|filtros-candidatos/);
  const propuesta = validarPropuestaLlamamientoPresentacion(obtenerPropuestaPresentacion("DEMO-NEC-0045"));
  assert.deepEqual(Object.keys(propuesta.evaluaciones[0]), ["orden", "resultado", "motivos"]);
  assert.throws(() => validarPropuestaLlamamientoPresentacion({
    ...obtenerPropuestaPresentacion("DEMO-NEC-0045"),
    nombre: "dato no permitido",
  }), /contrato cerrado/);
});

test("el contrato real falla cerrado y no completa datos ausentes con ceros o listas", () => {
  const sinIndicador = panelInternoReal();
  delete sinIndicador.indicadores.llamamientos_en_curso;
  assert.throws(() => validarPanelBolsa(sinIndicador), /indicadores no respeta el contrato cerrado/);

  const sinActuaciones = panelInternoReal();
  delete sinActuaciones.actuaciones_pendientes;
  assert.throws(() => validarPanelBolsa(sinActuaciones), /panel interno no respeta el contrato cerrado/);

  const demostracion = panelInternoReal();
  demostracion.origen.demostracion = true;
  assert.throws(() => validarPanelBolsa(demostracion), /no puede responder con datos de demostración/);

  const sinFechasOpcionales = panelInternoReal();
  delete sinFechasOpcionales.convocatorias[0].plazo_cierra_en;
  delete sinFechasOpcionales.actuaciones_pendientes[0].fecha_limite;
  const validado = validarPanelBolsa(sinFechasOpcionales);
  assert.equal(Object.hasOwn(validado.convocatorias[0], "plazo_cierra_en"), false);
  assert.equal(Object.hasOwn(validado.actuaciones_pendientes[0], "fecha_limite"), false);
});

test("el modo real renderiza solo indicadores, convocatorias y actuaciones acreditadas", () => {
  let fuente = validarPanelBolsa(panelInternoReal());
  const escapar = (valor) => String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;");
  const presentador = crearPresentadorPanelInterno({
    claseEstado: () => "neutro",
    encabezadoVista: (_sobrelinea, titulo, descripcion) => `<header><h2>${escapar(titulo)}</h2><p>${escapar(descripcion)}</p></header>`,
    escaparHTML: escapar,
    numero: (valor) => String(valor),
    obtenerDatosPanel: () => fuente,
    tituloVista: (vista) => `Vista ${vista}`,
  });
  assert.equal(presentador.esActivo(), true);
  const resumen = presentador.renderizarVista("resumen");
  assert.match(resumen, /cnv_0123456789abcdef/);
  assert.match(resumen, /120/);
  assert.match(resumen, /act_0123456789abcdef/);
  assert.match(resumen, /Prueba de lectura/);
  assert.match(presentador.renderizarVista("contratos"), /Funcionalidad no conectada/);

  fuente = validarPanelBolsa(obtenerDatosPresentacion(), true);
  assert.equal(presentador.esActivo(), false);
  assert.throws(() => presentador.renderizarVista("resumen"), /requiere un panel interno válido/);

  assert.match(javascript, /crearPresentadorPanelInterno/);
  assert.match(javascript, /portal-panel-interno\.js\?v=20260717-panel-interno-v1/);
  for (const indicador of [
    "convocatorias_borrador", "convocatorias_revision", "convocatorias_pendientes_firma",
    "convocatorias_publicadas", "bolsas_activas", "bolsas_suspendidas", "bolsas_agotadas",
    "llamamientos_pendientes", "llamamientos_en_curso", "llamamientos_vencen_hoy",
    "documentos_pendientes_firma", "incidencias_abiertas",
  ]) assert.match(panelInterno, new RegExp(`i\\.${indicador}`));
  assert.match(panelInterno, /No se muestran valores cero, tablas vacías ni controles aparentes/);
  assert.match(javascript, /estado\.modoPresentacion && datos/);
  assert.match(javascript, /datos-presentacion\.js/);
});

test("el coordinador respeta DEC-051 y carga el presentador con versión de caché", () => {
  assert.ok(javascript.split(/\r?\n/).length - 1 < 800, "portal.js debe mantenerse por debajo de 800 líneas");
  assert.match(html, /portal\.js\?v=20260721-acceso-real-v2/);
  assert.match(javascript, /portal-eventos\.js\?v=20260721-acceso-real-v2/);
  assert.match(javascript, /import\("\.\/portal-resumen-presentacion\.js\?v=20260721-acceso-real-v2"\)/);
  assert.doesNotMatch(javascript, /^import .*portal-resumen-presentacion/m);
  assert.doesNotMatch(manifiestoProduccion, /portal-resumen-presentacion\.js/);
  assert.match(manifiestoProduccion, /portal-i18n\.js/);
  assert.match(javascript, /if \(acceso\.disponible !== true \|\| vistaPermitida\(acceso\.vista\)\) return acceso/);
  assert.match(eventos, /case "reintentar-borradores"/);
  assert.match(eventos, /comprobarDisponibilidadBorradores\(\{ forzar: true \}\)/);
  assert.match(panelInterno, /export function crearPresentadorPanelInterno/);
});

test("la propuesta real usa el cliente cerrado y no habilita un detalle inexistente", () => {
  assert.match(apiLlamamientos, /const RUTA_PROPUESTAS_LLAMAMIENTO = "\/api\/vec\/bolsa\/propuestas-llamamiento"/);
  assert.match(apiLlamamientos, /if \(capacidad !== true\)/);
  assert.match(apiLlamamientos, /esquema: "vec\.bolsa\.propuesta-llamamiento\.solicitud\.v1"/);
  assert.doesNotMatch(`${javascript}\n${apiLlamamientos}`, /Idempotency-Key|randomUUID|claveIdempotenciaPropuesta/);
  assert.match(datos, /solicitar_propuesta_llamamiento: false/);
  assert.match(flujoLlamamientos, /conoce el cliente HTTP/);
  assert.match(javascript, /import\("\.\/portal-presentacion-adaptador\.js/);
  assert.doesNotMatch(javascript, /^import .*portal-presentacion-adaptador/m);
  assert.match(`${flujoLlamamientos}\n${vistaLlamamientos}`, /Detalle no disponible/);
  assert.match(eventos, /if \(resultado\.avanzar === true\) estado\.pasoLlamamiento = 2/);
  assert.doesNotMatch(datos, /puntuacion|Puntuación/);
  assert.doesNotMatch(contratoLlamamientos, /evaluaciones.*confirmacion|camposEvaluacion/i);
});

test("los datos de presentación están aislados y se activan de forma explícita", () => {
  const presentacion = validarPanelBolsa(obtenerDatosPresentacion(), true);
  assert.equal(presentacion.esquema, "vec.bolsa.panel.presentacion.v1");
  assert.equal(presentacion.demostracion, true);
  assert.ok(presentacion.bolsas.length > 0);
  assert.match(javascript, /getAll\("presentacion"\)/);
  assert.match(javascript, /import\("\.\/datos-presentacion\.js/);
  assert.match(datos, /ADAPTADOR EXCLUSIVO DE PRESENTACIÓN RRHH/);
  assert.match(datos, /esquema: "vec\.bolsa\.panel\.presentacion\.v1"/);
  assert.match(datos, /demostracion: true/);
  assert.match(html, /class="aviso-presentacion" role="status" hidden/);
  assert.match(html, /Referencias públicas de convocatoria y BOP reales/);
  assert.match(html, /Personas, expedientes y actuaciones internas sintéticos/);
  assert.doesNotMatch(datos, /\b\d{8}[A-Z]\b/);
});

test("la presentación RRHH usa referencias públicas reales y bases adaptadas locales", async () => {
  const elaboraciones = obtenerDatosPresentacion().elaboraciones;
  assert.deepEqual(elaboraciones.map((item) => item.cve_bop), [
    "BOP-GRA-2025-125002",
    "BOP-GRA-2024-244002",
    "BOP-GRA-2026-043004",
  ]);
  assert.deepEqual(elaboraciones.map((item) => item.publicacion_bop), [
    "04/07/2025",
    "19/12/2024",
    "05/03/2026",
  ]);
  assert.match(elaboraciones[0].nombre, /Auxiliar de Servicios Generales/);
  assert.equal(elaboraciones[1].nombre, "Ingreso en la Subescala de Gestión de Administración General");
  assert.equal(elaboraciones[1].identificador_publico, "gestion-administracion-general-2024");
  assert.match(elaboraciones[2].nombre, /Bolsa de empleo de Operario/);
  for (const elaboracion of elaboraciones) {
    assert.match(elaboracion.expediente, /^DEMO-/);
    assert.match(elaboracion.fase, /DEMO/);
    assert.equal(elaboracion.documentos_publicos.length, 2);
    for (const documento of elaboracion.documentos_publicos) {
      assert.match(documento.url, /^\/bolsa\/documentos\/bases-(?:auxiliar|gestion|operario)-demo\.(?:pdf|html)$/);
      const ruta = new URL(`../${documento.url.slice(1)}`, directorio);
      assert.ok((await stat(ruta)).size > 1_000, `${documento.url} debe ser un documento real de la presentación`);
    }
  }
});

test("el selector de perfil es cerrado y la navegación aplica mínimo privilegio", () => {
  assert.match(javascript, /getAll\("presentacion"\)/);
  assert.match(javascript, /getAll\("perfil"\)/);
  assert.match(javascript, /valores\.length !== 1[\s\S]{0,100}return null/);
  assert.doesNotMatch(javascript, /valores\.length === 0\) return "administrador"/);
  assert.match(javascript, /\["administrador", "tecnico", "funcionario"\]\.includes/);
  assert.match(javascript, /perfilPresentacionSolicitado\(\) === null\) return vista === "portal"/);
  assert.match(javascript, /adaptador\.obtenerDatosPresentacion\(perfil\)/);
  assert.match(javascript, /function vistaPermitida\(vista\)/);
  assert.match(javascript, /history\.replaceState\(null, "", hashSeguro\)/);
  assert.match(javascript, /control\.disabled = true/);
  assert.match(javascript, /querySelectorAll\("\[data-vista\], \[data-requiere-vista\]"\)/);
  assert.equal((`${javascript}\n${resumenPresentacion}`.match(/data-requiere-vista="llamamientos"/g) || []).length, 2);
  assert.match(eventos, /navegar\(vista, \{ enfocar: false \}\)/);
});

test("el portal interno no usa cookies ni almacenamiento del navegador", () => {
  assert.doesNotMatch(codigo, /localStorage|sessionStorage|document\.cookie/);
  assert.match(eventos, /document\.body\.dataset\.textoGrande/);
  assert.match(eventos, /document\.documentElement\.dataset\.textoGrande/);
  assert.match(estilosBase, /html\[data-texto-grande="true"\][\s\S]{0,80}font-size: 125%/);
  assert.match(eventos, /document\.body\.dataset\.contraste/);
});

test("texto ampliado y contraste siguen disponibles en resoluciones compactas", () => {
  assert.match(html, /id="boton-texto"[^>]+aria-label="Aumentar o restablecer el tamaño del texto"/);
  assert.match(html, /id="boton-contraste"[^>]+aria-label="Activar o desactivar el alto contraste"/);
  assert.doesNotMatch(estilosFlujos, /boton-cabecera:not\(\.boton-avisos\)[^{]*\{[^}]*display:\s*none/);
  assert.match(estilosCapacidades, /body\.portal-empleado-app\s*\{[^}]*font-size:\s*1rem/);
});

test("el portal conserva el shell rico y delega el catálogo sin fijar módulos en la plantilla", () => {
  assert.match(html, /id="navegacion-modulos-dinamica"/);
  assert.match(javascript, /crearCoordinadorModulosPortal/);
  assert.match(javascript, /VISTAS_MODULOS_PERSONALES/);
  assert.match(html, /modulos\/cronos\/cronos\.css/);
  assert.match(html, /modulos\/dietas\/dietas\.css/);
  assert.doesNotMatch(html, /Bolsas de trabajo[\s\S]{0,180}etiqueta-menu/);
  const vistas = ["resumen", "elaboracion", "convocatorias", "solicitudes", "meritos",
    "baremacion", "alegaciones", "importacion", "llamamientos", "contratos",
    "documentos", "comunicaciones", "estadisticas", "auditoria", "configuracion"];
  for (const vista of vistas) assert.match(html, new RegExp(`data-vista="${vista}"`));
});

test("la interfaz es semántica, adaptable y no contiene CSS inline", () => {
  assert.doesNotMatch(html.toLowerCase(), /<style\b|\sstyle=/);
  assert.match(html, /Saltar al contenido principal/);
  assert.match(html, /aria-live="polite"/);
  assert.match(estilos, /@media \(max-width: 1040px\)/);
  assert.match(estilos, /@media \(max-width: 780px\)/);
  assert.match(estilos, /@media \(max-width: 520px\)/);
  assert.match(estilos, /prefers-reduced-motion/);
});

test("la ayuda configurable incluye audio local, FAQ y transcripción accesible", async () => {
  assert.equal(AYUDA_PORTAL_BOLSA.esquema, "vec.portal.ayuda.v1");
  assert.ok(AYUDA_PORTAL_BOLSA.pasos.length >= 4);
  assert.ok(AYUDA_PORTAL_BOLSA.preguntas.length >= 3);
  assert.match(javascript, /<audio controls preload="metadata" aria-describedby="transcripcion-ayuda">/);
  assert.match(javascript, /Transcripción del audio/);
  assert.doesNotMatch(AYUDA_PORTAL_BOLSA.audio.src, /^https?:/);
  const rutaAudio = new URL(`.${AYUDA_PORTAL_BOLSA.audio.src.replace("/portal-empleado", "")}`, directorio);
  assert.ok((await stat(rutaAudio)).size > 10_000, "el audio local debe ser reproducible, no un marcador vacío");
  assert.match(ayuda, /Contenido de ayuda sustituible por catálogo o conector/);
});

test("la cabecera usa el logo institucional local, dimensionado y sin hotlink", async () => {
  assert.match(html, /data-identidad-institucional="diputacion-granada"/);
  assert.match(html, /src="\/portal-empleado\/assets\/logo-diputacion-granada\.svg" width="250" height="84" alt="Diputación de Granada"/);
  assert.match(estilosBase, /\.logo-institucional[\s\S]{0,260}width: min\(100%, 218px\)[\s\S]{0,160}height: auto/);
  assert.doesNotMatch(html, /<img[^>]+src="https?:/i);
  const rutaLogo = new URL("assets/logo-diputacion-granada.svg", directorio);
  assert.ok((await stat(rutaLogo)).size > 10_000);
  assert.doesNotMatch(await readFile(rutaLogo, "utf8"), /<script\b|<foreignObject\b|\sonload=/i);
});
