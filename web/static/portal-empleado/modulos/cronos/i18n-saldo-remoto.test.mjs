import assert from "node:assert/strict";
import test from "node:test";

import { crearTraductorCronos, MENSAJES_CRONOS_ES } from "./i18n.js";

const CLAVES_SALDO = [
  "titulo", "ayuda", "periodos", "hoy", "semana", "mes", "anio", "rango",
  "desde", "hasta", "consultar", "seleccionar_rango", "rango_invalido",
  "cargando", "vacio", "error", "denegado",
  "no_disponible", "previsto", "trabajado", "diferencia", "detalle", "fecha",
  "estado", "marcajes", "sin_marcajes", "estado_calculado",
  "estado_disponible", "estado_no_disponible", "estado_incompleto",
  "estado_sin_fuente", "exceso_semanal", "pausas", "canal", "movimiento_entrada",
  "movimiento_salida", "movimiento_inicio_pausa", "movimiento_fin_pausa",
  "origen_remoto", "origen_terminal", "origen_sin_verificar", "origen_manual", "origen_dispositivo",
].map((clave) => `saldo_${clave}`);

const CLAVES_REMOTO = [
  "titulo", "origen", "periodo", "sin_periodo", "consultando", "autorizado",
  "motivo_sin_autorizacion", "motivo_fuera_periodo", "motivo_no_disponible",
  "error_consulta", "autenticacion_requerida", "acceso_denegado", "sin_movimientos",
  "secuencia_no_permitida",
  "continuidad_pendiente", "servicio_no_disponible", "error_registro",
  "conflicto", "enviando", "incierto", "recuperando", "recuperacion_no_disponible",
  "reintentar", "registrado", "replay", "hora_servidor", "recibo",
  "actualizar", "ayuda",
].map((clave) => `remoto_${clave}`);

const CLAVES_MOVIMIENTOS = [
  "titulo", "periodos", "detalle", "vacio", "cargando", "error", "denegado",
  "fecha", "hora", "tipo", "origen", "estado", "correccion",
  "correccion_pendiente", "sin_marcajes", "rango_invalido", "seleccionar_rango",
].map((clave) => `movimientos_${clave}`);

test("el traductor real cubre saldo, movimientos y fichaje remoto en todos sus estados", () => {
  const traducir = crearTraductorCronos();
  for (const clave of [...CLAVES_SALDO, ...CLAVES_MOVIMIENTOS, ...CLAVES_REMOTO]) {
    if (clave !== "remoto_periodo") assert.equal(traducir(clave), MENSAJES_CRONOS_ES[clave], clave);
    assert.ok(traducir(clave).trim(), clave);
  }
  assert.match(traducir("saldo_estado_incompleto"), /incompleto/i);
  assert.equal(traducir("movimientos_correccion"), "Solicitar corrección");
  assert.match(traducir("movimientos_correccion_pendiente"), /no está disponible/);
  assert.match(traducir("remoto_incierto"), /misma clave/i);
  assert.match(traducir("remoto_continuidad_pendiente"), /último marcaje/);
  assert.match(traducir("remoto_sin_movimientos"), /no hay movimientos disponibles/i);
  assert.match(traducir("remoto_secuencia_no_permitida"), /no está permitido/i);
  assert.match(traducir("remoto_recuperando"), /recibo del marcaje/i);
  assert.match(traducir("remoto_recuperacion_no_disponible"), /vuelva a intentarlo/i);
  assert.equal(traducir("remoto_periodo", { desde: "1 oct 2026", hasta: "31 oct 2026" }),
    "Periodo autorizado: desde 1 oct 2026 hasta antes de 31 oct 2026");
  assert.equal(traducir("abrir_ayuda", { asunto: traducir("remoto_titulo") }), "Abrir ayuda sobre Fichaje remoto");
});

test("un catálogo incompleto o una clave no declarada fallan de forma cerrada", () => {
  assert.throws(
    () => crearTraductorCronos({ ...MENSAJES_CRONOS_ES, saldo_denegado: "" }),
    /catálogo i18n de Cronos incompleto/,
  );
  assert.throws(() => crearTraductorCronos()("saldo_inventado"), /clave i18n de Cronos desconocida/);
});
