import assert from "node:assert/strict";
import test from "node:test";
import { montarFormularioIncorporacionEjercicio } from "./formulario-incorporacion-ejercicio.js";

const periodo = { desde: "2026-09-09T00:00:00Z", hasta: "2026-10-09T00:00:00Z" };
const expedienteRef = "expediente:ejercicio:fechas";
const solicitudRef = "solicitud:personal:fechas";
const recibo = {
  esquema: "vec.contratacion-temporal.incorporacion-ejercicio.recibo.v2",
  expediente_ref: expedienteRef, solicitud_personal_ref: solicitudRef,
  relacion_ref: "relacion:personal:fechas", recibo_ref: "recibo:ct:fechas",
  actuacion_ref: "actuacion:ct:fechas", registrada_en: "2026-09-09T01:00:00Z",
  periodo_incorporacion: periodo, version_solicitud_personal: 7, version_actual_expediente: 8,
  seguimiento_ref: "seguimiento:ct:fechas", version_seguimiento_anterior: 0,
  version_seguimiento_resultante: 1, auditoria_ref: "auditoria:ct:fechas",
  outbox_ref: "outbox:ct:fechas", ejercicio_sintetico: true,
  firma_oficial: false, eficacia_administrativa: false,
};
const preparacion = {
  esquema: "vec.contratacion-temporal.incorporacion-ejercicio.preparacion.v2",
  expediente_ref: expedienteRef, version_actual_expediente: 8, recibo: null,
  preparacion: {
    solicitud_personal_ref: solicitudRef, version_solicitud_personal: 7,
    version_seguimiento_esperada: 0, periodo_incorporacion: periodo,
    motivos: ["confirmar_incorporacion"], documentos_refs: [], disponible: true,
  },
};

function pintar(proyeccion, zonaHoraria) {
  const raiz = {
    innerHTML: "", addEventListener() {}, removeEventListener() {},
    replaceChildren() { this.innerHTML = ""; },
  };
  const cliente = {
    prepararIncorporacionEjercicio() { assert.fail("GET inesperado"); },
    confirmarIncorporacionEjercicio() { assert.fail("POST inesperado"); },
  };
  const desmontar = montarFormularioIncorporacionEjercicio({
    raiz, cliente, preparacion: proyeccion, zonaHoraria,
  });
  const html = raiz.innerHTML;
  desmontar();
  return html;
}

test("el período conserva su día civil en Madrid y Bogotá; el registro conserva su hora local", () => {
  const original = JSON.stringify({ preparacion, recibo });
  const fechaCivil = new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeZone: "UTC" });
  const inicio = fechaCivil.format(new Date(periodo.desde));
  const fin = fechaCivil.format(new Date(periodo.hasta));
  const registros = [];

  for (const zonaHoraria of ["Europe/Madrid", "America/Bogota"]) {
    const fechaRegistro = new Intl.DateTimeFormat("es-ES", {
      dateStyle: "medium", timeStyle: "medium", timeZone: zonaHoraria,
    }).format(new Date(recibo.registrada_en));
    registros.push(fechaRegistro);

    for (const proyeccion of [preparacion, { ...preparacion, preparacion: null, recibo }]) {
      const html = pintar(proyeccion, zonaHoraria);
      assert.ok(html.includes(`Inicio del período</dt><dd>${inicio}</dd>`), zonaHoraria);
      assert.ok(html.includes(`Fin del período</dt><dd>${fin}</dd>`), zonaHoraria);
      assert.doesNotMatch(html, /(?:Inicio|Fin) del período<\/dt><dd>[^<]*\d{1,2}:\d{2}/u);
      if (proyeccion.recibo) {
        assert.ok(html.includes(
          `Fecha original de registro</dt><dd><time datetime="${recibo.registrada_en}">${fechaRegistro}</time></dd>`,
        ), zonaHoraria);
      }
    }
  }

  assert.notEqual(registros[0], registros[1]);
  assert.equal(JSON.stringify({ preparacion, recibo }), original);
});
