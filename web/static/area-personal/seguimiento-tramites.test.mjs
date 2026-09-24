import assert from "node:assert/strict";
import test from "node:test";

import {
  renderizarAlegaciones, renderizarLlamamientos, renderizarSeguimiento, renderizarSubsanaciones,
} from "./vistas/seguimiento-tramites.js";

// Doble mínimo de los datos del área personal que leen estas vistas.
function datosPrueba() {
  return {
    meta: { presentacion: false },
    solicitudes: [{ id: "SOL-0001", convocatoria_id: "CONV-001", referencia: "REG-0001", titulo: "Solicitud de prueba", estado: "Registrada", actualizado: "17/07/2026" }],
    actividad: [], llamamientos: [], subsanaciones: [], alegaciones: [],
    disponibilidad: { disponible: true, estado: "Disponible" },
  };
}

test("seguimiento y trámites informan de ámbitos vacíos sin datos aparentes", () => {
  const datos = datosPrueba();
  datos.solicitudes = [];
  datos.actividad = [];
  datos.llamamientos = [];
  datos.subsanaciones = [];
  datos.alegaciones = [];
  delete datos.posicion;

  const seguimiento = renderizarSeguimiento(datos, { expedienteSeleccionado: "SOL-INEXISTENTE" });
  assert.match(seguimiento, /Sin expedientes en el ámbito autorizado/u);
  assert.match(seguimiento, /No hay acciones disponibles hasta que el servicio facilite un expediente autorizado/u);
  assert.doesNotMatch(seguimiento, /undefined|\[object Object\]|Descargar expediente/u);
  assert.match(renderizarLlamamientos(datos), /Sin información de contratos\./u);
  assert.match(renderizarSubsanaciones(datos), /Sin subsanaciones/u);
  assert.match(renderizarAlegaciones(datos), /Sin alegaciones/u);
});

test("mi bolsa muestra tarjetas propias, provisionalidad y paginación", () => {
  const datos = datosPrueba();
  const participaciones = Array.from({ length: 7 }, (_, indice) => ({ bolsa: `bolsa:prueba:${indice}`, categoria: "Auxiliar", version: 3, orden_inicial: indice + 1, total_instantanea: 87, estado_bolsa: "vigente", vigente_desde: "2026-09-01T00:00:00Z", vigente_hasta: null }));
  const vista = renderizarLlamamientos(datos, { participaciones, paginaParticipaciones: 1, fuenteBolsa: "ejemplo" });
  assert.doesNotMatch(vista, /Datos de ejemplo|aviso-fuente-ejemplo/u);
  assert.match(vista, /Mi número de orden inicial[\s\S]*1 de 87/u);
  assert.match(vista, /Versión de la bolsa/u);
  assert.match(vista, /Mostrando 1 a 6 de 7/u);
  assert.doesNotMatch(vista, /Identificarse con certificado no firma documentos|Pendiente de integración/u);
  assert.match(vista, /Sin información de contratos\./u);
  assert.match(vista, /La situación actual de esta participación aún no está disponible/u);
  assert.doesNotMatch(vista, /Ensayar pausa|Ensayar reactivación/u);
});

test("mi bolsa distingue estado vigente de la participación y vigencia de la bolsa", () => {
  const datos = datosPrueba();
  const participaciones = [{ bolsa: "bolsa:prueba", categoria: "Auxiliar", version: 3, orden_inicial: 2, total_instantanea: 40, estado_bolsa: "vigente", vigente_desde: "2026-09-01T00:00:00Z", vigente_hasta: null,
    situacion_actual: { estado: "no_disponible", desde: "2026-09-20T10:00:00Z", hasta: null, fecha_disponible: null } }];
  const vista = renderizarLlamamientos(datos, { participaciones, fuenteBolsa: "real" });
  assert.match(vista, /Última situación registrada de mi participación/u);
  assert.match(vista, /No disponible/u);
  assert.match(vista, /20 sept 2026/u);
  assert.match(vista, /Sin fecha de fin registrada/u);
  assert.match(vista, /Consta temporalmente no disponible en Bolsa/u);
  assert.doesNotMatch(vista, /Datos de ejemplo|motivo|Pausa comunicada/u);
});

test("mi bolsa muestra el último resultado B7 propio sin respuesta ni plazo", () => {
  const datos = datosPrueba();
  const participaciones = [
    { bolsa: "bolsa:1", categoria: "Auxiliar", version: 1, orden_inicial: 2, total_instantanea: 3, estado_bolsa: "vigente", vigente_desde: "2026-09-01T00:00:00Z", vigente_hasta: null, ultimo_llamamiento: { emitido_en: "2026-09-20T10:00:00Z", canal: "correo", resultado: "enviado" } },
    { bolsa: "bolsa:2", categoria: "Administrativo", version: 1, orden_inicial: 1, total_instantanea: 3, estado_bolsa: "vigente", vigente_desde: "2026-09-01T00:00:00Z", vigente_hasta: null, ultimo_llamamiento: { emitido_en: "2026-09-21T10:00:00Z", canal: "correo", resultado: "no_enviado" } },
  ];
  const vista = renderizarLlamamientos(datos, { participaciones, fuenteBolsa: "real" });
  assert.match(vista, /Último resultado de correo[\s\S]*bolsa:2[\s\S]*Administrativo[\s\S]*No enviado/u);
  assert.match(vista, /no acredita recepción, respuesta ni plazo aprobado/u);
  assert.doesNotMatch(vista, /Aceptar llamamiento|Rechazar llamamiento|nota privada/u);
});

test("subsanaciones y alegaciones pendientes solo ofrecen la operación real", () => {
  const datos = datosPrueba();
  datos.subsanaciones = [{ id: "SUB-0001", solicitud_ref: "SOL-0001", motivo: "Acreditar jornada", plazo: "23/07/2026 14:00", estado: "Pendiente", documento_solicitado: "Certificado de jornada" }];
  datos.alegaciones = [{ id: "ALE-0001", solicitud_ref: "SOL-0001", asunto: "Revisión de formación", estado: "Borrador", fecha: "17/07/2026" }];
  const vistas = `${renderizarSubsanaciones(datos)}\n${renderizarAlegaciones(datos)}`;
  assert.match(vistas, /El fichero se custodiará solo si el servicio confirma la carga/u);
  assert.match(vistas, /La operación se firmará y registrará/u);
  assert.equal(vistas.match(/>Revisar, firmar y presentar<\/button>/gu)?.length, 2);
  assert.doesNotMatch(vistas, /DEMO|demostración/iu);
});
