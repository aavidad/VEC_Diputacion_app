import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { obtenerDatosPresentacion } from "../datos-presentacion.js";
import {
  ESQUEMA_CONTEXTO_ACTOR_FRONTEND,
  compartirContextoActor,
  crearProveedorContextoActorFijo,
  exigirContextoParaModulo,
  validarYCongelarContextoActor,
} from "./contexto-actor.js";
import { crearContextoActorPresentacionDesdeSesion } from "./presentacion.js";

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

test("la sesion existente de Bolsa produce una unica identidad interna inmutable", () => {
  const sesionBolsa = obtenerDatosPresentacion("administrador").sesion;
  const contexto = crearContextoActorPresentacionDesdeSesion(sesionBolsa);

  assert.equal(contexto.actor.actor_ref, sesionBolsa.actor_ref);
  assert.equal(contexto.actor.nombre_visible, sesionBolsa.nombre);
  assert.match(contexto.persona_ref, /^per_demo_/);
  assert.match(contexto.cuenta_ref, /^cta_demo_/);
  assert.match(contexto.perfil_ref, /^prf_demo_/);
  assert.deepEqual(contexto.ambito.modulos, ["bolsa"]);
  assert.equal(contexto.demostracion, true);
  assert.equal(contexto.autenticacion.metodo, "demo");
  assert.equal(contexto.autenticacion.garantia, "bajo");
  assert.ok(Object.isFrozen(contexto));
  assert.ok(Object.isFrozen(contexto.actor));
  assert.ok(Object.isFrozen(contexto.ambito));
  assert.ok(Object.isFrozen(contexto.ambito.modulos));
});

test("Cronos y Dietas reciben exactamente el mismo ContextoActor del funcionario", () => {
  const contexto = crearContextoActorPresentacionDesdeSesion(
    obtenerDatosPresentacion("funcionario").sesion,
  );
  const proveedor = crearProveedorContextoActorFijo(contexto);
  const identidades = compartirContextoActor(proveedor, ["cronos", "dietas"]);

  assert.strictEqual(proveedor.obtenerContexto(), contexto);
  assert.strictEqual(identidades.cronos, identidades.dietas);
  assert.strictEqual(proveedor.obtenerContexto(), identidades.cronos);
  assert.throws(() => exigirContextoParaModulo(contexto, "bolsa"), /fuera del ambito/);
  assert.ok(Object.isFrozen(identidades));
});

test("un modulo fuera del ambito falla cerrado y no deriva otra identidad", () => {
  const contexto = crearContextoActorPresentacionDesdeSesion(
    obtenerDatosPresentacion("tecnico").sesion,
  );

  assert.strictEqual(exigirContextoParaModulo(contexto, "bolsa"), contexto);
  assert.throws(() => exigirContextoParaModulo(contexto, "cronos"), /fuera del ambito/);
  assert.throws(() => exigirContextoParaModulo(contexto, "nominas"), /fuera del ambito/);
  assert.throws(
    () => exigirContextoParaModulo(Object.freeze({ ambito: Object.freeze({ modulos: Object.freeze(["cronos"]) }) }), "cronos"),
    /validado e inmutable/,
  );
});

test("los perfiles internos conservan actores distintos y el funcionario queda en autoservicio", () => {
  const administrador = crearContextoActorPresentacionDesdeSesion(
    obtenerDatosPresentacion("administrador").sesion,
  );
  const tecnico = crearContextoActorPresentacionDesdeSesion(
    obtenerDatosPresentacion("tecnico").sesion,
  );
  const funcionario = crearContextoActorPresentacionDesdeSesion(
    obtenerDatosPresentacion("funcionario").sesion,
  );

  assert.notEqual(administrador.persona_ref, tecnico.persona_ref);
  assert.notEqual(funcionario.persona_ref, tecnico.persona_ref);
  assert.equal(administrador.rol.clave, "administrador_funcional_bolsa");
  assert.equal(tecnico.rol.clave, "tecnico_revisor_rrhh");
  assert.equal(funcionario.rol.clave, "funcionario_autoservicio");
  assert.deepEqual(funcionario.ambito.modulos, ["cronos", "dietas"]);
  assert.throws(() => exigirContextoParaModulo(funcionario, "bolsa"), /fuera del ambito/);
  assert.deepEqual(tecnico.ambito.modulos, ["bolsa"]);
});

test("la identidad no incorpora permisos globales ni datos identificativos civiles", () => {
  const sesion = obtenerDatosPresentacion("administrador").sesion;
  const contexto = crearContextoActorPresentacionDesdeSesion({
    ...sesion,
    vistas_permitidas: ["*", "valor_que_no_debe_copiarse"],
    operaciones_permitidas: ["*", "valor_que_no_debe_copiarse"],
  });
  const serializado = JSON.stringify(contexto);
  const claves = clavesRecursivas(contexto).map((clave) => clave.toLowerCase());

  assert.doesNotMatch(serializado, /valor_que_no_debe_copiarse/);
  assert.ok(!claves.includes("dni"));
  assert.ok(!claves.includes("correo"));
  assert.ok(!claves.includes("permisos"));
  assert.ok(!claves.includes("operaciones_permitidas"));
});

test("la presentacion rechaza actores desconocidos y atribuciones inconsistentes", () => {
  const sesion = obtenerDatosPresentacion("tecnico").sesion;
  assert.throws(
    () => crearContextoActorPresentacionDesdeSesion({ ...sesion, actor_ref: "DEMO-PERFIL-OTRO-01" }),
    /no reconocido/,
  );
  assert.throws(
    () => crearContextoActorPresentacionDesdeSesion({ ...sesion, nombre: "Otra persona" }),
    /no coincide/,
  );
  assert.throws(() => crearContextoActorPresentacionDesdeSesion(null), /no valida/);
});

test("el contrato productivo acepta la proyeccion de una sesion interna fuerte", () => {
  const contexto = validarYCongelarContextoActor(contextoProductivoValido());
  assert.equal(contexto.demostracion, false);
  assert.equal(contexto.autenticacion.metodo, "kerberos_ad");
  assert.equal(contexto.autenticacion.garantia, "alto");
  assert.strictEqual(exigirContextoParaModulo(contexto, "dietas"), contexto);
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

test("el adaptador de presentacion no persiste estado ni inicia comunicaciones", async () => {
  const archivos = await Promise.all([
    readFile(new URL("./contexto-actor.js", import.meta.url), "utf8"),
    readFile(new URL("./presentacion.js", import.meta.url), "utf8"),
  ]);
  const codigo = archivos.join("\n");
  for (const patron of [
    /localStorage/u, /sessionStorage/u, /document\.cookie/u, /fetch\s*\(/u,
    /XMLHttpRequest/u, /WebSocket/u, /EventSource/u, /sendBeacon/u,
  ]) {
    assert.doesNotMatch(codigo, patron);
  }
});
