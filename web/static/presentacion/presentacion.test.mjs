import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";

const directorio = dirname(fileURLToPath(import.meta.url));
const html = readFileSync(join(directorio, "index.html"), "utf8");
const estilos = readFileSync(join(directorio, "presentacion.css"), "utf8");
const catalogoPortalInterno = readFileSync(join(directorio, "../portal-empleado/portal-catalogo-presentacion.js"), "utf8");
const catalogo = html.match(/<section class="presentacion-catalogo"[\s\S]*?<\/section>/)?.[0] || "";

const modulos = [
  "Bolsas de trabajo",
  "Personal",
  "Nóminas",
  "Cronos",
  "Dietas",
  "Solicitudes y certificados",
  "Méritos y formación",
  "Comunicaciones",
  "Documentos y firma",
  "Aprobaciones y portafirmas",
  "Auditoría",
  "Administración y configuración",
];

test("el lanzador conserva la visión completa del Portal de Recursos Humanos", () => {
  assert.match(catalogo, /id="modulos-portal">Módulos del portal/);
  for (const modulo of modulos) {
    assert.match(catalogo, new RegExp(`<h3[^>]*>${modulo}<\\/h3>`));
    assert.match(catalogoPortalInterno, new RegExp(`titulo: "${modulo}"`));
  }
});

test("Bolsa Cronos y Dietas ofrecen acceso y el resto queda inequívocamente pendiente", () => {
  assert.equal((catalogo.match(/presentacion-modulo-activo/g) || []).length, 3);
  assert.equal((catalogo.match(/presentacion-modulo-pendiente/g) || []).length, modulos.length - 3);
  assert.equal((catalogo.match(/Pendiente · No disponible/g) || []).length, modulos.length - 3);
  assert.equal((catalogo.match(/<a\b/g) || []).length, 3);
  assert.match(catalogo, /<a href="#elegir-recorrido">Elegir acceso<\/a>/);
  assert.match(catalogo, /perfil=administrador#cronos">Abrir módulo<\/a>/);
  assert.match(catalogo, /perfil=administrador#dietas">Abrir módulo<\/a>/);
  assert.doesNotMatch(catalogo, /href="\/(?:nominas|personal)/);
});

test("el catálogo de módulos se adapta a escritorio, tableta y móvil", () => {
  assert.match(estilos, /\.presentacion-rejilla-modulos\s*\{[^}]*grid-template-columns:\s*repeat\(4,/s);
  assert.match(estilos, /@media \(max-width: 52rem\)[\s\S]*?\.presentacion-rejilla-modulos\s*\{[^}]*grid-template-columns:\s*repeat\(2,/);
  assert.match(estilos, /@media \(max-width: 36rem\)[\s\S]*?\.presentacion-rejilla-modulos\s*\{[^}]*grid-template-columns:\s*1fr/);
});

test("las rutas de presentación de Bolsa permanecen disponibles", () => {
  assert.match(html, /href="\/bolsa\/">Abrir consulta pública/);
  assert.match(html, /href="\/area-personal\/\?presentacion=rrhh">Abrir área personal/);
  assert.match(html, /perfil=tecnico#portal">Abrir como técnico revisor/);
  assert.match(html, /perfil=administrador#portal">Abrir como administrador/);
});
