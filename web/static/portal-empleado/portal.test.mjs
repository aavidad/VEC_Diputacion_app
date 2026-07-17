import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const directorio = new URL("./", import.meta.url);
const [html, javascript, eventos, datos, estilosBase, estilosComponentes, estilosFlujos] = await Promise.all([
  readFile(new URL("index.html", directorio), "utf8"),
  readFile(new URL("portal.js", directorio), "utf8"),
  readFile(new URL("portal-eventos.js", directorio), "utf8"),
  readFile(new URL("datos-presentacion.js", directorio), "utf8"),
  readFile(new URL("portal.css", directorio), "utf8"),
  readFile(new URL("portal-componentes.css", directorio), "utf8"),
  readFile(new URL("portal-flujos.css", directorio), "utf8"),
]);
const codigo = `${javascript}\n${eventos}`;
const estilos = `${estilosBase}\n${estilosComponentes}\n${estilosFlujos}`;

test("la ruta normal usa API protegida y no cae a datos sintéticos", () => {
  assert.match(javascript, /const API_PANEL_BOLSA = "\/api\/vec\/bolsa\/panel"/);
  assert.match(javascript, /credentials: "same-origin"/);
  assert.match(javascript, /la API interna no puede responder con datos de demostración/);
  assert.match(javascript, /if \(respuesta\.status === 401\)/);
  assert.match(javascript, /if \(respuesta\.status === 403\)/);
  assert.match(javascript, /let DATOS_PANEL = DATOS_VACIOS/);
  assert.doesNotMatch(codigo, /María Pérez|García López|Auxiliar Administrativo|BOL-2026|CON-2026|DOC-[A-Z]{2}|20\/07\/2026/);
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
