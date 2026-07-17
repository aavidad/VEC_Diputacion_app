import assert from "node:assert/strict";
import { readFile, stat } from "node:fs/promises";
import test from "node:test";
import {
  extraerDatosEnvelopeCanonico,
  validarPanelBolsa,
  validarPropuestaLlamamiento,
} from "./portal-contrato.js";
import { obtenerDatosPresentacion } from "./datos-presentacion.js";
import { AYUDA_PORTAL_BOLSA } from "./ayuda-contenido.js";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js";

const directorio = new URL("./", import.meta.url);
const [html, javascript, eventos, contrato, panelInterno, datos, ayuda, estilosBase, estilosComponentes, estilosFlujos] = await Promise.all([
  readFile(new URL("index.html", directorio), "utf8"),
  readFile(new URL("portal.js", directorio), "utf8"),
  readFile(new URL("portal-eventos.js", directorio), "utf8"),
  readFile(new URL("portal-contrato.js", directorio), "utf8"),
  readFile(new URL("portal-panel-interno.js", directorio), "utf8"),
  readFile(new URL("datos-presentacion.js", directorio), "utf8"),
  readFile(new URL("ayuda-contenido.js", directorio), "utf8"),
  readFile(new URL("portal.css", directorio), "utf8"),
  readFile(new URL("portal-componentes.css", directorio), "utf8"),
  readFile(new URL("portal-flujos.css", directorio), "utf8"),
]);
const codigo = `${javascript}\n${eventos}\n${contrato}\n${panelInterno}`;
const estilos = `${estilosBase}\n${estilosComponentes}\n${estilosFlujos}`;

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

test("la ruta normal usa API protegida y no cae a datos sintéticos", () => {
  assert.match(javascript, /const API_PANEL_BOLSA = "\/api\/vec\/bolsa\/panel"/);
  assert.match(javascript, /credentials: "same-origin"/);
  assert.match(javascript, /extraerDatosEnvelopeCanonico\(envelope\)/);
  assert.match(contrato, /la API interna no puede responder con datos de demostración/);
  assert.match(javascript, /if \(respuesta\.status === 401\)/);
  assert.match(javascript, /if \(respuesta\.status === 403\)/);
  assert.match(javascript, /let DATOS_PANEL = DATOS_VACIOS/);
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
  const propuesta = validarPropuestaLlamamiento({
    esquema: "vec.bolsa.propuesta-llamamiento.presentacion.v1", demostracion: true,
    id: "propuesta-sintetica", necesidad_id: "necesidad-sintetica", personas_incluidas: 1,
    evaluaciones: [{ secuencia: 1, resultado: "Elegible", puntuacion: 1, regla: "R1", fundamento: "Caso sintético" }],
  }, true);
  assert.deepEqual(Object.keys(propuesta.evaluaciones[0]), ["secuencia", "resultado", "puntuacion", "regla", "fundamento"]);
  assert.throws(() => validarPropuestaLlamamiento({
    esquema: "vec.bolsa.propuesta-llamamiento.presentacion.v1", demostracion: true,
    evaluaciones: [{ secuencia: 1, nombre: "dato no permitido" }],
  }, true), /no admite identidad ni contacto/);
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
  assert.match(html, /portal\.js\?v=20260717-panel-interno-v1/);
  assert.match(panelInterno, /export function crearPresentadorPanelInterno/);
});

test("solicitar una propuesta está preparado con idempotencia y cerrado por capacidad", () => {
  assert.match(javascript, /const API_PROPUESTAS_LLAMAMIENTO = "\/api\/vec\/bolsa\/propuestas-llamamiento"/);
  assert.match(javascript, /if \(!puedeSolicitarPropuesta\(\)\)[\s\S]{0,180}return/);
  assert.match(javascript, /"Idempotency-Key": estado\.claveIdempotenciaPropuesta/);
  assert.match(javascript, /esquema: "vec\.bolsa\.propuesta-llamamiento\.solicitud\.v1"/);
  assert.match(datos, /solicitar_propuesta_llamamiento: false/);
  assert.match(codigo, /el navegador no elige personas/i);
  assert.match(javascript, /La propuesta sintética se carga localmente: no ejecuta el POST/);
  assert.match(javascript, /Propuesta sintética de elegibilidad/);
});

test("los datos de presentación están aislados y se activan de forma explícita", () => {
  const presentacion = validarPanelBolsa(obtenerDatosPresentacion(), true);
  assert.equal(presentacion.esquema, "vec.bolsa.panel.presentacion.v1");
  assert.equal(presentacion.demostracion, true);
  assert.ok(presentacion.bolsas.length > 0);
  assert.match(javascript, /get\("presentacion"\) === "rrhh"/);
  assert.match(javascript, /import\("\.\/datos-presentacion\.js/);
  assert.match(datos, /ADAPTADOR EXCLUSIVO DE PRESENTACIÓN RRHH/);
  assert.match(datos, /esquema: "vec\.bolsa\.panel\.presentacion\.v1"/);
  assert.match(datos, /demostracion: true/);
  assert.match(html, /class="aviso-presentacion" role="status" hidden/);
  assert.match(html, /Datos íntegramente sintéticos/);
  assert.doesNotMatch(datos, /\b\d{8}[A-Z]\b/);
});

test("ningún dato de negocio se guarda en localStorage", () => {
  const usos = [...codigo.matchAll(/localStorage\.(?:getItem|setItem)\(([^\n]+)/g)].map((coincidencia) => coincidencia[1]);
  assert.ok(usos.length > 0, "deben existir preferencias visuales comprobables");
  for (const uso of usos) assert.match(uso, /vec_portal_(?:\$\{nombre\}|texto|contraste)/);
  assert.doesNotMatch(codigo, /localStorage.*(?:bolsa|candidato|llamamiento|expediente)/i);
});

test("el portal expone solo Bolsa y conserva los diez paneles solicitados", () => {
  assert.match(html, /Bolsas de trabajo[\s\S]*etiqueta-menu">Activo/);
  for (const modulo of ["Personal", "Nóminas", "Cronos", "Dietas", "Solicitudes y certificados"]) {
    assert.match(html, new RegExp(`${modulo}[\\s\\S]{0,180}No habilitado`));
  }
  const vistas = ["elaboracion", "llamamientos", "contratos", "reglas", "consulta", "resumen", "estadisticas", "documentos", "comunicaciones", "auditoria"];
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
