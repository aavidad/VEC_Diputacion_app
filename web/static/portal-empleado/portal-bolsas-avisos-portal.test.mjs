import assert from "node:assert/strict";
import test from "node:test";
import { renderizarBloqueAvisos, validarAvisosBolsa } from "./portal-bolsas-avisos.js";

const sobre = (items, conteos) => ({ data: { esquema: "vec.bolsa.rrhh.avisos.v1", provisionalidad: "Pendiente", items, conteos, paginacion: { desde: 1, hasta: items.length, total: items.length } } });

test("la bandeja de RRHH muestra solicitudes y respuestas del portal del candidato", () => {
  const datos = validarAvisosBolsa(sobre([
    { tipo: "solicitud_portal", bolsa: "bolsa:demo:1", referencia: "solicitud-portal:" + "a".repeat(64), fecha: "2026-09-25T10:00:00Z", detalle: { participacion_ref: "participacion:1", solicitud: "pausa", pausa_hasta: "2026-12-31T22:59:59Z" } },
    { tipo: "respuesta_portal", bolsa: "bolsa:demo:1", referencia: "respuesta-portal:" + "b".repeat(64), fecha: "2026-09-25T11:00:00Z", detalle: { participacion_ref: "participacion:1", respuesta: "renuncia_justificada", modo: "firme", causa: "enfermedad", justificante_ref: "justificante:1" } },
  ], { salto_orden: 0, tres_anos: 0, solicitud_portal: 1, respuesta_portal: 1 }));
  const html = renderizarBloqueAvisos({ estado: "listo", datos });
  assert.match(html, /Solicitud desde «Mi bolsa»/u);
  assert.match(html, /Pausa voluntaria hasta/u);
  assert.match(html, /Renuncia justificada\. Respuesta firme\. Causa: enfermedad/u);
  assert.match(html, /1 solicitudes del portal/u);
  assert.match(html, /data-accion="abrir-ficha-b5"/u);
  // Solicitud y justificante se copian desde su botón; no se leen en pantalla.
  const visible = html.replace(/data-[a-z-]+="[^"]*"/gu, "");
  assert.doesNotMatch(visible, /solicitud-portal:|justificante:1|participacion:/u);
  assert.match(html, /data-copiar-justificante="solicitud-portal:a{64}" data-texto-copiado="Referencia copiada" aria-label="Copiar la referencia de la solicitud"/u);
  assert.match(html, /data-copiar-justificante="justificante:1"/u);
});

test("sin portal compuesto la bandeja conserva su contrato anterior", () => {
  assert.doesNotThrow(() => validarAvisosBolsa(sobre([], { salto_orden: 0, tres_anos: 0 })));
  assert.throws(() => validarAvisosBolsa(sobre([], { salto_orden: 0, tres_anos: 0, solicitud_portal: -1 })), /no válido/u);
  assert.throws(() => validarAvisosBolsa(sobre([{ tipo: "otro", bolsa: "bolsa:1", referencia: "ref:1", fecha: "2026-09-25T10:00:00Z", detalle: {} }], { salto_orden: 0, tres_anos: 0 })), /no válido/u);
});
