import assert from "node:assert/strict";
import test from "node:test";

import { crearAdaptadorPresentacion } from "./adaptador-presentacion.js";

const BORRADOR_COMPLETO = {
  convocatoria_id: "DEMO-CONV-001",
  requisitos_confirmados: true,
  datos_confirmados: true,
  meritos_ids: ["DEMO-MER-001", "DEMO-MER-002"],
  autobaremo_revisado: true,
};
const SOLICITUD_BORRADOR = "DEMO-SOL-BORRADOR-0001";

const CASOS = [
  ["actualizar_contacto", { correo: "nuevo@vec-demo.test", telefono: "000 000 001", domicilio: "Domicilio sintético actualizado" }],
  ["incorporar_merito", { tipo: "Formación", titulo: "Mérito añadido durante la prueba" }],
  ["guardar_borrador", BORRADOR_COMPLETO],
  ["calcular_autobaremo", { convocatoria_id: "DEMO-CONV-001" }],
  ["iniciar_pago", { id: SOLICITUD_BORRADOR }],
  ["firmar_solicitud", { id: SOLICITUD_BORRADOR }],
  ["registrar_solicitud", { id: SOLICITUD_BORRADOR, declaracion_final: true }],
  ["cambiar_disponibilidad", { disponible: false }],
  ["responder_llamamiento", { id: "DEMO-LLA-0045", respuesta: "aceptar" }],
  ["presentar_subsanacion", { id: "DEMO-SUB-0008" }],
  ["presentar_alegacion", { id: "DEMO-ALE-0003" }],
  ["marcar_mensaje", { id: "DEMO-MSG-001" }],
  ["actualizar_notificaciones", { correo: "on" }],
  ["solicitar_certificado", { id: "DEMO-CER-001", formato: "JSON" }],
  ["solicitar_descarga", { id: "DEMO-DOC-001" }],
];

test("todas las operaciones de presentación devuelven recibo DEMO y estado válido", async () => {
  const adaptador = crearAdaptadorPresentacion();
  for (const [accion, payload] of CASOS) {
    const resultado = await adaptador.ejecutar({ accion, payload, confirmacion: true, capacidad: true });
    assert.equal(resultado.recibo.presentacion, true, accion);
    assert.match(resultado.recibo.referencia, /^DEMO-REC-\d{4}$/u, accion);
    assert.match(resultado.recibo.advertencia, /RECIBO DEMO/u, accion);
    assert.equal(resultado.datos.meta.presentacion, true, accion);
  }
});

test("el estado de la demo vive en la instancia y desaparece al crear otra", async () => {
  const primera = crearAdaptadorPresentacion();
  await primera.ejecutar({
    accion: "actualizar_contacto",
    payload: { correo: "cambio@vec-demo.test" },
    confirmacion: true,
    capacidad: true,
  });
  assert.equal((await primera.cargar()).perfil.correo, "cambio@vec-demo.test");
  assert.equal((await crearAdaptadorPresentacion().cargar()).perfil.correo, "aspirante@vec-demo.test");
});

test("la demo falla cerrada sin capacidad, confirmación o acción conocida", async () => {
  const adaptador = crearAdaptadorPresentacion();
  await assert.rejects(() => adaptador.ejecutar({ accion: "guardar_borrador", confirmacion: false, capacidad: true }), /confirmación explícita/);
  await assert.rejects(() => adaptador.ejecutar({ accion: "guardar_borrador", confirmacion: true, capacidad: false }), /capacidad no está concedida/);
  await assert.rejects(() => adaptador.ejecutar({ accion: "accion_inventada", confirmacion: true, capacidad: true }), /no pertenece/);
});

test("el borrador exige requisitos, datos, méritos y autobaremación antes de guardarse", async () => {
  const adaptador = crearAdaptadorPresentacion();
  const ejecutar = (payload) => adaptador.ejecutar({
    accion: "guardar_borrador", payload, confirmacion: true, capacidad: true,
  });
  await assert.rejects(() => ejecutar({ ...BORRADOR_COMPLETO, requisitos_confirmados: false }), /declaración de requisitos/iu);
  await assert.rejects(() => ejecutar({ ...BORRADOR_COMPLETO, datos_confirmados: false }), /confirmación de datos/iu);
  await assert.rejects(() => ejecutar({ ...BORRADOR_COMPLETO, meritos_ids: [] }), /al menos un mérito/iu);
  await assert.rejects(() => ejecutar({ ...BORRADOR_COMPLETO, autobaremo_revisado: false }), /autobaremación/iu);
  assert.equal((await adaptador.cargar()).solicitudes.some((item) => /borrador/i.test(item.estado)), false);
});

test("pago, firma y registro actúan sobre el borrador indicado y respetan la secuencia", async () => {
  const adaptador = crearAdaptadorPresentacion();
  const ejecutar = (accion, payload) => adaptador.ejecutar({ accion, payload, confirmacion: true, capacidad: true });
  await ejecutar("guardar_borrador", BORRADOR_COMPLETO);

  await assert.rejects(() => ejecutar("firmar_solicitud", { id: SOLICITUD_BORRADOR }), /pago o la exención/iu);
  await assert.rejects(() => ejecutar("registrar_solicitud", { id: SOLICITUD_BORRADOR, declaracion_final: true }), /pago o exención/iu);
  await assert.rejects(() => ejecutar("iniciar_pago", { id: "DEMO-SOL-INEXISTENTE" }), /no existe/iu);

  await ejecutar("iniciar_pago", { id: SOLICITUD_BORRADOR });
  await assert.rejects(() => ejecutar("registrar_solicitud", { id: SOLICITUD_BORRADOR, declaracion_final: true }), /firma confirmada/iu);
  await ejecutar("firmar_solicitud", { id: SOLICITUD_BORRADOR });
  await assert.rejects(() => ejecutar("registrar_solicitud", { id: SOLICITUD_BORRADOR }), /declaración final/iu);
  await ejecutar("registrar_solicitud", { id: SOLICITUD_BORRADOR, declaracion_final: true });

  const datos = await adaptador.cargar();
  const original = datos.solicitudes.find((item) => item.id === "DEMO-SOL-0027");
  const registrada = datos.solicitudes.find((item) => item.id === SOLICITUD_BORRADOR);
  assert.equal(original.pago, "Tasa simulada abonada");
  assert.equal(original.referencia, "DEMO-REG-2026-0027");
  assert.match(registrada.estado, /Registro DEMO completado/u);
  assert.match(registrada.pago, /confirmado/u);
  assert.match(registrada.firma, /confirmada/u);
});
