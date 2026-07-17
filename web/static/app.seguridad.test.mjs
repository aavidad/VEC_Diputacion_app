import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";

const directorio = dirname(fileURLToPath(import.meta.url));
const javascript = readFileSync(join(directorio, "app.js"), "utf8");

function extraerLlamadasFetch(codigo) {
  const llamadas = [];
  const inicioFetch = /\bfetch\s*\(/g;
  let coincidencia;

  while ((coincidencia = inicioFetch.exec(codigo)) !== null) {
    const inicio = codigo.indexOf("(", coincidencia.index);
    let profundidad = 0;
    let estado = "codigo";
    let escapado = false;
    let fin = -1;

    for (let indice = inicio; indice < codigo.length; indice += 1) {
      const caracter = codigo[indice];
      const siguiente = codigo[indice + 1];

      if (estado === "comentario-linea") {
        if (caracter === "\n") estado = "codigo";
        continue;
      }
      if (estado === "comentario-bloque") {
        if (caracter === "*" && siguiente === "/") {
          estado = "codigo";
          indice += 1;
        }
        continue;
      }
      if (estado !== "codigo") {
        if (escapado) {
          escapado = false;
          continue;
        }
        if (caracter === "\\") {
          escapado = true;
          continue;
        }
        const cierre = estado === "cadena-simple" ? "'" : estado === "cadena-doble" ? '"' : "`";
        if (caracter === cierre) estado = "codigo";
        continue;
      }

      if (caracter === "/" && siguiente === "/") {
        estado = "comentario-linea";
        indice += 1;
      } else if (caracter === "/" && siguiente === "*") {
        estado = "comentario-bloque";
        indice += 1;
      } else if (caracter === "'") {
        estado = "cadena-simple";
      } else if (caracter === '"') {
        estado = "cadena-doble";
      } else if (caracter === "`") {
        estado = "plantilla";
      } else if (caracter === "(") {
        profundidad += 1;
      } else if (caracter === ")") {
        profundidad -= 1;
        if (profundidad === 0) {
          fin = indice + 1;
          break;
        }
      }
    }

    assert.notEqual(fin, -1, `llamada fetch sin cierre desde el carácter ${inicio}`);
    llamadas.push(codigo.slice(coincidencia.index, fin));
    inicioFetch.lastIndex = fin;
  }

  return llamadas;
}

test("DEC-053: toda llamada fetch omite explícitamente las credenciales ambientales", () => {
  const llamadas = extraerLlamadasFetch(javascript);
  assert.ok(llamadas.length > 0, "la prueba debe observar las llamadas de red de la SPA");

  for (const llamada of llamadas) {
    assert.match(
      llamada,
      /\bcredentials\s*:\s*["']omit["']/,
      `fetch sin credentials: \"omit\": ${llamada.slice(0, 120)}`,
    );
  }
});

test("DEC-053: la SPA no crea sesiones web ni persiste credenciales", () => {
  assert.doesNotMatch(javascript, /document\s*\.\s*cookie/i);
  assert.doesNotMatch(javascript, /\bcredentials\s*:\s*["'](?:same-origin|include)["']/i);
  assert.doesNotMatch(
    javascript,
    /localStorage[^\n]*(?:token|sesi[oó]n|auth|credencial|identidad)|(?:token|sesi[oó]n|auth|credencial|identidad)[^\n]*localStorage/i,
  );
});

test("el almacenamiento local heredado queda limitado a datos sintéticos de demostración", () => {
  const clavesPermitidas = new Set([
    "DIETAS_SHEETS_STORAGE_KEY",
    "EMPLOYEE_DIRECTORY_STORAGE_KEY",
    '"vec_demo_bolsa_offers"',
    '"vec_demo_bolsa_employee_applications"',
  ]);
  const usos = [
    ...javascript.matchAll(/\b(?:readStoredArray|storedArraySnapshot|writeStoredArray)\s*\(\s*([^,\)\n]+)/g),
  ]
    .filter((coincidencia) => !javascript.slice(Math.max(0, coincidencia.index - 12), coincidencia.index).includes("function "))
    .map((coincidencia) => coincidencia[1].trim());

  assert.ok(usos.length > 0, "la clasificación debe cubrir el almacenamiento heredado existente");
  for (const clave of usos) {
    assert.ok(clavesPermitidas.has(clave), `clave local no clasificada como demo: ${clave}`);
  }

  for (const nombreConstante of ["DIETAS_SHEETS_STORAGE_KEY", "EMPLOYEE_DIRECTORY_STORAGE_KEY"]) {
    assert.match(
      javascript,
      new RegExp(`const ${nombreConstante} = ["']vec_demo_[a-z0-9_]+["']`),
      `${nombreConstante} debe seguir siendo inequívocamente sintética`,
    );
  }
});
