import test from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, writeFileSync, readFileSync, existsSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const guion = fileURLToPath(new URL("./publicar.sh", import.meta.url));
const estado = {
  secuencia: 2, activa: true, version: 2,
  huella: "6ecf24d17b889c8bbd4400580977ac8b1c2c863c824f09ac5498d70a3d4a7141",
  aprobacion: "operacion:datos-inventados:20261012",
  fuente: "a9192d1a68e79a21f17650be6d57632ad3520fa91cfeb820fee84e5d6b89768a",
  ajustes: 0,
};

test("la publicación v3 comprueba la base y rechaza ajustes antes de escribir", () => {
  const privado = mkdtempSync(join(tmpdir(), "vec-plazos-v3-"));
  try {
    const consulta = join(privado, "consulta.sql");
    const respuesta = join(privado, "respuesta.json");
    const psql = join(privado, "psql");
    writeFileSync(psql, '#!/bin/sh\ncat > "$CT_PRUEBA_CONSULTA"\ncat "$CT_PRUEBA_RESPUESTA"\n', { mode: 0o700 });
    const ejecutar = (dato, argumentos) => {
      rmSync(consulta, { force: true });
      writeFileSync(respuesta, JSON.stringify(dato));
      return spawnSync("bash", [guion, ...argumentos], {
        encoding: "utf8",
        env: { PATH: process.env.PATH, LC_ALL: "C", TMPDIR: privado,
          VEC_REGLAS_CT_PSQL: psql, CT_PRUEBA_CONSULTA: consulta, CT_PRUEBA_RESPUESTA: respuesta },
      });
    };
    const listo = ejecutar(estado, ["--comprobar"]);
    assert.equal(listo.status, 0, listo.stderr);
    assert.equal(JSON.parse(listo.stdout).version, 3);
    assert.match(readFileSync(consulta, "utf8"), /REPEATABLE READ READ ONLY/);
    assert.doesNotMatch(readFileSync(consulta, "utf8"), /\b(?:INSERT|UPDATE|DELETE)\b/);

    for (const dato of [{ ...estado, ajustes: 1 }, { ...estado, activa: false }, { ...estado, version: 3 }]) {
      const resultado = ejecutar(dato, ["--aplicar", "PUBLICA-PLAZOS-CT-V3"]);
      assert.notEqual(resultado.status, 0, "una base incompatible no puede publicarse");
      assert.match(readFileSync(consulta, "utf8"), /READ ONLY/);
    }
    assert.notEqual(ejecutar(estado, ["--aplicar", "PUBLICA-PLAZOS-CT-V3"]).status, 0, "requiere el preparador");
    assert.notEqual(ejecutar(estado, ["--aplicar", "otro"]).status, 0);
    assert.equal(existsSync(consulta), false, "el argumento inválido no consulta la base");
  } finally {
    rmSync(privado, { recursive: true, force: true });
  }
});
