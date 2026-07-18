import assert from "node:assert/strict";
import test from "node:test";

import { crearAdaptadorPresentacion } from "./adaptador-presentacion.js";

const CASOS = [
  ["actualizar_contacto", { correo: "nuevo@vec-demo.test", telefono: "000 000 001", domicilio: "Domicilio sintético actualizado" }],
  ["incorporar_merito", { tipo: "Formación", titulo: "Mérito añadido durante la prueba" }],
  ["guardar_borrador", { convocatoria_id: "DEMO-CONV-001" }],
  ["calcular_autobaremo", { convocatoria_id: "DEMO-CONV-001" }],
  ["iniciar_pago", { id: "DEMO-CONV-001" }],
  ["firmar_solicitud", { id: "DEMO-CONV-001" }],
  ["registrar_solicitud", { id: "DEMO-CONV-001" }],
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
