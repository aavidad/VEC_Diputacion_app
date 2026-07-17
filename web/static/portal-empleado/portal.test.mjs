import assert from "node:assert/strict";
import { readFile, stat } from "node:fs/promises";
import test from "node:test";
import {
  extraerDatosEnvelopeCanonico,
  validarPanelBolsa,
  validarPropuestaLlamamiento,
} from "./portal-contrato.js";
import { AYUDA_PORTAL_BOLSA } from "./ayuda-contenido.js";

const directorio = new URL("./", import.meta.url);
const [html, javascript, eventos, contrato, datos, ayuda, estilosBase, estilosComponentes, estilosFlujos] = await Promise.all([
  readFile(new URL("index.html", directorio), "utf8"),
  readFile(new URL("portal.js", directorio), "utf8"),
  readFile(new URL("portal-eventos.js", directorio), "utf8"),
  readFile(new URL("portal-contrato.js", directorio), "utf8"),
  readFile(new URL("datos-presentacion.js", directorio), "utf8"),
  readFile(new URL("ayuda-contenido.js", directorio), "utf8"),
  readFile(new URL("portal.css", directorio), "utf8"),
  readFile(new URL("portal-componentes.css", directorio), "utf8"),
  readFile(new URL("portal-flujos.css", directorio), "utf8"),
]);
const codigo = `${javascript}\n${eventos}\n${contrato}`;
const estilos = `${estilosBase}\n${estilosComponentes}\n${estilosFlujos}`;

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
  const panel = {
    esquema: "vec.bolsa.panel.v1", demostracion: false,
    bolsas: [], necesidades_llamamiento: [], elaboraciones: [], proximos: [],
    actividad: [], contratos: [], reglas: [], documentos: [], canales: [], avisos: [],
  };
  assert.throws(() => extraerDatosEnvelopeCanonico(panel), /envelope canónico/);
  assert.deepEqual(extraerDatosEnvelopeCanonico({ data: panel }), panel);
  assert.equal(validarPanelBolsa(extraerDatosEnvelopeCanonico({ data: panel })).esquema, "vec.bolsa.panel.v1");
});

test("el panel global prohíbe candidatos y la propuesta es un contrato separado", () => {
  const panel = {
    esquema: "vec.bolsa.panel.v1", demostracion: false,
    bolsas: [], necesidades_llamamiento: [], elaboraciones: [], proximos: [],
    actividad: [], contratos: [], reglas: [], documentos: [], canales: [], avisos: [], candidatos: [],
  };
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
