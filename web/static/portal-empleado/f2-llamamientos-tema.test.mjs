import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const leer = (nombre) => readFileSync(new URL(nombre, import.meta.url), "utf8");
const estilos = leer("portal-llamamientos.css");
const tema = leer("portal.css");
const portal = leer("index.html");

function luminancia(hex) {
  const canales = hex.match(/[\da-f]{2}/gi).map((canal) => {
    const valor = parseInt(canal, 16) / 255;
    return valor <= 0.04045 ? valor / 12.92 : ((valor + 0.055) / 1.055) ** 2.4;
  });
  return canales[0] * 0.2126 + canales[1] * 0.7152 + canales[2] * 0.0722;
}

function contraste(a, b) {
  const [mayor, menor] = [luminancia(a), luminancia(b)].sort((x, y) => y - x);
  return (mayor + 0.05) / (menor + 0.05);
}

test("llamamientos montado consume solo colores del tema común", () => {
  assert.match(portal, /<link rel="stylesheet" href="\/portal-empleado\/portal-llamamientos\.css\?v=[^"]+">/);
  assert.match(estilos, /\.estado-asistente\s*\{[^}]*background: var\(--portal-azul-100\);[^}]*color: var\(--portal-tinta\)/);
  assert.match(estilos, /\.resumen-errores\s*\{[^}]*background: var\(--portal-peligro-suave\);[^}]*color: var\(--portal-peligro\)/);
  assert.match(estilos, /\.recibo-llamamiento\s*\{[^}]*border: 2px solid var\(--portal-exito\);[^}]*background: var\(--portal-exito-suave\)/);
  assert.doesNotMatch(estilos, /#[\da-f]{3,8}\b|(?:rgb|hsl)a?\(/i);

  const definidos = new Set([...tema.matchAll(/(--portal-[\w-]+)\s*:/g)].map((coincidencia) => coincidencia[1]));
  const usados = [...estilos.matchAll(/var\((--portal-[\w-]+)(?:,\s*[^)]+)?\)/g)];
  assert.deepEqual(usados.filter((uso) => !definidos.has(uso[1]) && !uso[0].includes(","))
    .map((uso) => uso[1]), []);
});

test("texto de estado y error conserva contraste AA con tema normal y alto contraste", () => {
  const raiz = tema.match(/^:root\s*\{([^}]+)\}/)?.[1];
  assert.ok(raiz, "existe tema base");
  const normal = Object.fromEntries([...raiz.matchAll(/(--portal-[\w-]+):\s*(#[\da-f]{6})/gi)]
    .map((coincidencia) => [coincidencia[1], coincidencia[2]]));
  const alto = tema.match(/body\.portal-empleado-app\[data-contraste="true"\]\s*\{([^}]+)\}/)?.[1];
  assert.ok(alto, "existe variante común de alto contraste");
  const sobrescrituras = Object.fromEntries([...alto.matchAll(/(--portal-[\w-]+):\s*(#[\da-f]{6})/gi)]
    .map((coincidencia) => [coincidencia[1], coincidencia[2]]));
  for (const colores of [normal, { ...normal, ...sobrescrituras }]) {
    assert.ok(contraste(colores["--portal-tinta"], colores["--portal-azul-100"]) >= 4.5);
    assert.ok(contraste(colores["--portal-peligro"], colores["--portal-peligro-suave"]) >= 4.5);
    assert.ok(contraste(colores["--portal-tinta"], colores["--portal-exito-suave"]) >= 4.5);
  }
});

test("el asistente conserva marcos internos y adaptación móvil", () => {
  assert.match(estilos, /@media \(min-width: 1024px\)[\s\S]*overflow: hidden;[\s\S]*overflow: auto;/);
  assert.match(estilos, /@media \(max-width: 520px\)[\s\S]*grid-template-columns: 1fr;/);
  assert.match(estilos, /@media \(forced-colors: active\)[\s\S]*border: 2px solid CanvasText;/);
});
