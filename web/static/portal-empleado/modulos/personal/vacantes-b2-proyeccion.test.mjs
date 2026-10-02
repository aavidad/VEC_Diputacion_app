import assert from "node:assert/strict";
import test from "node:test";
import { proyectarPaginaVacantesB2 } from "./vacantes-b2-proyeccion.js";

function pagina(puesto = "puesto_sintetico") {
  const version = "plantilla_version_sintetica";
  return { organismo_ref: "organismo_sintetico", corte: { vigente_en: "2026-10-01", conocido_en: "2026-10-01T12:00:00Z" }, cobertura: "completa", vacantes: [{
    plaza_ref: "plaza_sintetica", puesto_ref: puesto, puesto_denominacion: "Técnico/a", unidad_ref: "unidad_sintetica", unidad_denominacion: "Unidad de prueba", codigo_plaza_fuente: "00041", estado_cobertura: "vacante_sin_ocupacion", version_plantilla_ref: version, version_rpt_ref: "rpt_version_sintetica",
    traza: { desde: "2024-01-01", hasta: "2027-01-01", registrada_en: "2026-09-30T11:00:00Z", revision_estructural: 3, version_plantilla_ref: version, acto_ref: "acto_sintetico", fuente_ref: "fuente_sintetica", fuente_huella_sha256: "a".repeat(64) },
  }] };
}

test("tres ejes: plaza sin ocupación no acredita puesto libre ni necesidad cubrible", () => {
  const dto = pagina(); dto.vacantes[0].cubrible = true; dto.vacantes[0].puesto_libre = true; dto.vacantes[0].persona_ref = "persona_no_publicable";
  const modelo = proyectarPaginaVacantesB2(dto); const fila = modelo.filas[0];
  assert.equal(fila.dotacionMensaje, "sin_ocupacion"); assert.equal(fila.puestoMensaje, "puesto_vinculado"); assert.equal(fila.necesidadMensaje, "cubrible_pendiente");
  assert.equal(fila.codigoPlaza, "00041", "conserva el código opaco y los ceros");
  assert.ok(!JSON.stringify(modelo).includes("persona_no_publicable"));
  assert.ok(Object.isFrozen(modelo) && Object.isFrozen(fila));
  dto.vacantes[0].traza.revision_estructural = 20;
  assert.equal(fila.origenVisible.find((dato) => dato.codigoMensaje === "revision_estructural").valor, 3);
});

test("ausencia de vínculo no se traduce a un puesto libre ni conserva una denominación incongruente", () => {
  const fila = proyectarPaginaVacantesB2(pagina("")).filas[0];
  assert.equal(fila.puestoMensaje, "puesto_no_consta"); assert.equal(fila.puesto, ""); assert.equal(fila.necesidadMensaje, "cubrible_pendiente");
  assert.ok(!fila.origenTecnico.some((dato) => dato.codigoMensaje === "puesto_ref"));
});

test("origen conserva revisión estructural, versiones, acto y huella sin convertirlos en aprobación", () => {
  const fila = proyectarPaginaVacantesB2(pagina()).filas[0];
  assert.equal(fila.origenVisible.find((dato) => dato.codigoMensaje === "revision_estructural").valor, 3);
  assert.equal(fila.origenTecnico.find((dato) => dato.codigoMensaje === "fuente_huella_sha256").valor, "a".repeat(64));
  for (const codigoMensaje of ["version_plantilla_ref", "version_rpt_ref", "acto_ref", "fuente_ref"]) assert.ok(fila.origenTecnico.some((dato) => dato.codigoMensaje === codigoMensaje));
  assert.ok(!JSON.stringify(fila).includes("firma_oficial"));
});

test("cobertura incompleta o trazas incompatibles rechazan toda la página, incluida una página vacía", () => {
  assert.throws(() => proyectarPaginaVacantesB2({ ...pagina(), cobertura: "parcial", vacantes: [] }), TypeError);
  const otro = pagina(); otro.vacantes[0].traza.version_plantilla_ref = "otra_version";
  assert.throws(() => proyectarPaginaVacantesB2(otro), TypeError);
  const reservado = pagina(); reservado.vacantes[0].estado_cobertura = "reservada";
  assert.throws(() => proyectarPaginaVacantesB2(reservado), TypeError);
});


test("las denominaciones válidas de la fuente conservan 257 ASCII y 300 puntos Unicode; 301 se rechaza", () => {
  for (const nombre of ["P".repeat(257), "😀".repeat(300)]) {
    const original = pagina(); original.vacantes[0].puesto_denominacion = nombre; original.vacantes[0].unidad_denominacion = nombre;
    const fila = proyectarPaginaVacantesB2(original).filas[0];
    assert.equal(fila.puesto, nombre); assert.equal(fila.unidad, nombre);
    assert.equal(fila.necesidadMensaje, "cubrible_pendiente");
  }
  for (const campo of ["puesto_denominacion", "unidad_denominacion"]) {
    const original = pagina(); original.vacantes[0][campo] = "😀".repeat(301);
    assert.throws(() => proyectarPaginaVacantesB2(original), TypeError);
  }
  const referencia = pagina(); referencia.vacantes[0].plaza_ref = "r".repeat(257);
  assert.throws(() => proyectarPaginaVacantesB2(referencia), TypeError, "las referencias conservan su límite anterior");
});
