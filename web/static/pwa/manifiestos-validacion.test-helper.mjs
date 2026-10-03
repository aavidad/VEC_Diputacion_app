import assert from "node:assert/strict";

const RUTAS = Object.freeze({
  "pwa-portal-empleado": "/portal-empleado/",
  "pwa-area-personal": "/area-personal/",
  "pwa-admin": "/administracion-perfiles/",
});
const CLAVES = ["id", "name", "short_name", "description", "lang", "start_url", "scope", "display", "background_color", "theme_color", "icons"].sort();
const ICONOS = [
  ["vec-192", "192x192", "any"],
  ["vec-512", "512x512", "any"],
  ["vec-maskable-192", "192x192", "maskable"],
  ["vec-maskable-512", "512x512", "maskable"],
];

export function esManifiestoPWA(modulo) {
  return Object.hasOwn(RUTAS, modulo);
}

// Los iconos son metadatos cerrados del manifiesto, no mensajes traducibles.
export function validarManifiestoPWA(manifiesto, modulo, idioma) {
  assert.ok(esManifiestoPWA(modulo), "manifiesto PWA no registrado");
  assert.ok(manifiesto && typeof manifiesto === "object" && !Array.isArray(manifiesto));
  assert.deepEqual(Object.keys(manifiesto).sort(), CLAVES);
  const ruta = RUTAS[modulo];
  assert.equal(manifiesto.id, ruta);
  assert.equal(manifiesto.scope, ruta);
  assert.equal(manifiesto.start_url, `${ruta}?lang=${idioma}`);
  assert.equal(manifiesto.lang, idioma);
  assert.equal(manifiesto.display, "standalone");
  for (const clave of ["name", "short_name", "description"]) {
    assert.ok(typeof manifiesto[clave] === "string" && manifiesto[clave].trim().length > 0);
  }
  for (const clave of ["theme_color", "background_color"]) {
    assert.match(manifiesto[clave], /^#[0-9a-f]{6}$/u);
  }
  assert.ok(Array.isArray(manifiesto.icons));
  assert.equal(manifiesto.icons.length, ICONOS.length);
  for (const [indice, [nombre, tamano, finalidad]] of ICONOS.entries()) {
    const icono = manifiesto.icons[indice];
    assert.ok(icono && typeof icono === "object" && !Array.isArray(icono));
    assert.deepEqual(Object.keys(icono).sort(), ["purpose", "sizes", "src", "type"]);
    assert.equal(icono.src, `/pwa/icons/${nombre}.png?v=20261002-pwa-v1`);
    assert.equal(icono.sizes, tamano);
    assert.equal(icono.type, "image/png");
    assert.equal(icono.purpose, finalidad);
  }
}
