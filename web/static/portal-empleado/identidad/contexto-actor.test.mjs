import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import {
  ESQUEMA_CONTEXTO_ACTOR_FRONTEND,
  compartirContextoActor,
  crearProveedorContextoActorFijo,
  exigirContextoParaModulo,
  validarYCongelarContextoActor,
} from "./contexto-actor.js";

function contextoProductivoValido() {
  return {
    esquema: ESQUEMA_CONTEXTO_ACTOR_FRONTEND,
    revision: 8,
    demostracion: false,
    persona_ref: "per_persona_interna_opaca_000001",
    cuenta_ref: "cta_cuenta_interna_opaca_000001",
    perfil_ref: "prf_perfil_interno_activo_000001",
    actor: {
      actor_ref: "prf_perfil_interno_activo_000001",
      nombre_visible: "Persona interna",
      iniciales: "PI",
    },
    rol: {
      clave: "empleado_publico",
      etiqueta: "Empleado público",
    },
    ambito: {
      clase: "personal_interno",
      organizacion_ref: "org_organizacion_interna_000001",
      unidad_ref: "uni_unidad_interna_destino_000001",
      modulos: ["bolsa", "cronos", "dietas"],
    },
    autenticacion: {
      sesion_ref: "ses_sesion_interna_opaca_000001",
      metodo: "kerberos_ad",
      garantia: "alto",
    },
    resuelto_en: "2026-07-19T08:30:00.000Z",
  };
}

function clavesRecursivas(valor, resultado = []) {
  if (!valor || typeof valor !== "object") return resultado;
  for (const [clave, contenido] of Object.entries(valor)) {
    resultado.push(clave);
    clavesRecursivas(contenido, resultado);
  }
  return resultado;
}

test("el contrato productivo acepta la proyeccion de una sesion interna fuerte", () => {
  const contexto = validarYCongelarContextoActor(contextoProductivoValido());
  assert.equal(contexto.demostracion, false);
  assert.equal(contexto.autenticacion.metodo, "kerberos_ad");
  assert.equal(contexto.autenticacion.garantia, "alto");
  assert.strictEqual(exigirContextoParaModulo(contexto, "dietas"), contexto);
});

test("el contexto validado se comparte por referencia solo con módulos de su ámbito", () => {
  const contexto = validarYCongelarContextoActor(contextoProductivoValido());
  const proveedor = crearProveedorContextoActorFijo(contexto);
  const identidades = compartirContextoActor(proveedor, ["bolsa", "cronos", "dietas"]);
  assert.strictEqual(identidades.bolsa, contexto);
  assert.strictEqual(identidades.cronos, identidades.dietas);
  assert.ok(Object.isFrozen(identidades));
  assert.throws(() => exigirContextoParaModulo(contexto, "nominas"), /fuera del ambito/);
  assert.throws(() => compartirContextoActor(proveedor, ["cronos", "cronos"]), /repetido/);
});

test("el contexto real no añade permisos globales ni identificadores civiles", () => {
  const contexto = validarYCongelarContextoActor(contextoProductivoValido());
  const claves = clavesRecursivas(contexto).map((clave) => clave.toLowerCase());
  for (const clave of ["dni", "correo", "permisos", "operaciones_permitidas"]) {
    assert.ok(!claves.includes(clave), clave);
  }
});

test("el contrato cerrado rechaza campos extra, referencias o ambitos ambiguos", () => {
  const base = contextoProductivoValido();
  assert.throws(
    () => validarYCongelarContextoActor({ ...base, campo_inesperado: true }),
    /contrato cerrado/,
  );
  assert.throws(
    () => validarYCongelarContextoActor({ ...base, persona_ref: "persona-visible" }),
    /persona_ref no valida/,
  );
  assert.throws(
    () => validarYCongelarContextoActor({
      ...base,
      ambito: { ...base.ambito, modulos: ["cronos", "cronos"] },
    }),
    /ambito.modulos no valido/,
  );
  assert.throws(
    () => validarYCongelarContextoActor({
      ...base,
      autenticacion: { ...base.autenticacion, metodo: "demo" },
    }),
    /autenticacion no valida/,
  );
});

test("la proyección de contexto no persiste estado ni inicia comunicaciones", async () => {
  const codigo = await readFile(new URL("./contexto-actor.js", import.meta.url), "utf8");
  for (const patron of [
    /localStorage/u, /sessionStorage/u, /document\.cookie/u, /fetch\s*\(/u,
    /XMLHttpRequest/u, /WebSocket/u, /EventSource/u, /sendBeacon/u,
  ]) {
    assert.doesNotMatch(codigo, patron);
  }
});
