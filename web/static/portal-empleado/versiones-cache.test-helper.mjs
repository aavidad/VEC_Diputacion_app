import assert from "node:assert/strict";

// Comprobaciones compartidas de caché immutable: cuando cambia un módulo, su
// URL (`?v=`) cambia y todos sus importadores piden la misma URL nueva. Así las
// pruebas no persiguen el literal de cada renovación, sino la invariante.

function escapar(texto) {
  return texto.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
}

/** Devuelve todas las versiones con que `fuente` referencia `recurso`. */
export function versionesDe(fuente, recurso) {
  const patron = new RegExp(`(?<![\\w-])${escapar(recurso)}\\?v=([\\w.-]+)`, "gu");
  return [...fuente.matchAll(patron)].map(([, version]) => version);
}

/** Devuelve la única versión con que `fuente` referencia `recurso`. */
export function versionDe(fuente, recurso) {
  const versiones = new Set(versionesDe(fuente, recurso));
  assert.equal(versiones.size, 1, `${recurso} debe referenciarse con una sola versión: ${[...versiones].join(", ") || "ninguna"}`);
  return [...versiones][0];
}

/**
 * Exige que todas las `fuentes` referencien `recurso` con una misma versión y
 * que no sea ninguna de las `anteriores` ya publicadas (una caché immutable
 * las conservaría). Devuelve esa versión vigente.
 */
export function exigirRenovado(fuentes, recurso, anteriores) {
  const lista = Array.isArray(fuentes) ? fuentes : [fuentes];
  const versiones = new Set();
  for (const fuente of lista) {
    const propias = versionesDe(fuente, recurso);
    assert.ok(propias.length > 0, `${recurso} debe referenciarse versionado en cada importador`);
    propias.forEach((version) => versiones.add(version));
  }
  assert.equal(versiones.size, 1, `${recurso} no admite referencias mixtas: ${[...versiones].join(", ")}`);
  const [vigente] = versiones;
  for (const anterior of Array.isArray(anteriores) ? anteriores : [anteriores])
    assert.notEqual(vigente, anterior, `${recurso} no puede reutilizar la versión publicada ${anterior}`);
  return vigente;
}

/** Marca una versión esperada como «cualquiera posterior a `anterior`». */
export const posterior = (anterior) => Object.freeze({ posterior: anterior });

/**
 * Comprueba una arista importador→recurso: `cantidad` referencias con una
 * única versión, que es exactamente `esperada` (recurso sin cambios) o
 * distinta de `esperada.posterior` (recurso renovado). Devuelve la vigente.
 */
export function exigirVersiones(codigo, recurso, esperada, cantidad = 1) {
  const encontradas = versionesDe(codigo, recurso);
  assert.equal(encontradas.length, cantidad, `${recurso}: número de referencias versionadas`);
  assert.equal(new Set(encontradas).size, 1, `${recurso} no admite referencias mixtas: ${encontradas.join(", ")}`);
  const [vigente] = encontradas;
  if (typeof esperada === "string") assert.equal(vigente, esperada, recurso);
  else assert.notEqual(vigente, esperada.posterior, `${recurso} no puede reutilizar la versión publicada ${esperada.posterior}`);
  return vigente;
}
