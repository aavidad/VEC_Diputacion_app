import assert from "node:assert/strict";
import test from "node:test";

import { diezmilesimasDesdeHorasMinutos, montarFormularioAnalisisRRHH } from "./formulario-analisis.js";

const CLAVE = "123e4567-e89b-42d3-a456-426614174000";
const HUELLA = "a".repeat(64);
const FormDataOriginal = globalThis.FormData;

globalThis.FormData = class {
  constructor(formulario) { this.valores = formulario.valores; }
  get(nombre) { return this.valores[nombre] ?? null; }
};
test.after(() => { globalThis.FormData = FormDataOriginal; });

function montar() {
  const eventos = new Map();
  const solicitudes = [];
  const raiz = {
    innerHTML: "",
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo) { eventos.delete(tipo); },
    contains() { return true; },
    querySelector() { return { focus() {}, scrollIntoView() {} }; },
    replaceChildren() { this.innerHTML = ""; },
  };
  const desmontar = montarFormularioAnalisisRRHH({
    raiz,
    contexto: {
      operacion: "registrar", expediente_ref: "expediente:opaco:001",
      version_esperada: 1, artefacto_ref: "artefacto:opaco:001",
    },
    catalogos: {
      modalidades: ["sustitucion", "vacante", "acumulacion_tareas", "programa", "relevo"]
        .map((clave) => ({ clave, etiqueta: clave })),
      categorias: [{ referencia: "categoria:rrhh:001", etiqueta: "Categoría", grupos_subgrupos: [
        { clave: "A1", etiqueta: "A1" },
      ] }],
      causas: [{ clave: "sustitucion", etiqueta: "Sustitución" }],
      entradas_rc: [{ referencia: "entrada-rc:opaca:001", huella_sha256: HUELLA, etiqueta: "RC" }],
      motivos_rectificacion: [{ clave: "correccion_datos", etiqueta: "Corrección" }],
    },
    generarClaveIdempotencia: () => CLAVE,
    cliente: {
      registrarAnalisis(solicitud) {
        solicitudes.push(solicitud);
        return Promise.resolve({
          esquema: "vec.contratacion-temporal.recibo-analisis-rrhh.v1",
          operacion: "registrar", expediente_ref: "expediente:opaco:001",
          version_resultante: 2, recibo_ref: "recibo:opaco:analisis:001",
          confirmada_en: "2026-08-21T21:30:00Z",
        });
      },
    },
  });
  return {
    raiz, solicitudes, desmontar,
    enviar(jornada_horas, jornada_minutos, jornada_original) {
      const formulario = {
        valores: {
          modalidad_clave: "sustitucion", categoria_ref: "categoria:rrhh:001",
          grupo_subgrupo: "A1", causa_clave: "sustitucion",
          inicio: "2026-09-01", fin: "2027-08-31",
          jornada_horas, jornada_minutos, jornada_original,
          entrada_rc_referencia: "entrada-rc:opaca:001",
        },
        closest(selector) { return selector === "[data-ct-analisis-form]" ? this : null; },
      };
      return eventos.get("submit")({ target: formulario, preventDefault() {} });
    },
  };
}

test("acepta enteros decimales con ceros iniciales dentro de la jornada semanal", () => {
  for (const [horas, minutos, esperado] of [
    ["07", "05", "1889"], ["7", "00", "1867"], ["0", "01", "4"],
    ["00037", "030", "10000"],
  ]) {
    assert.equal(diezmilesimasDesdeHorasMinutos(horas, minutos), esperado);
  }
  for (const [horas, minutos] of [
    ["0", "00"], ["37", "31"], ["38", "00"], ["7", "60"],
    ["-7", "05"], ["+7", "05"], ["7.0", "05"], ["7e0", "05"],
    ["Infinity", "05"], ["07", "-5"], ["07", "+5"], ["07", "5.0"],
    ["07", "5e0"], ["07", "Infinity"],
  ]) {
    assert.equal(diezmilesimasDesdeHorasMinutos(horas, minutos), "", `${horas}h${minutos}`);
  }
});

test("07h05 registra el DTO exacto en diezmilésimas", async () => {
  const vista = montar();
  await vista.enviar("07", "05");
  assert.deepEqual(vista.solicitudes, [{
    expediente_ref: "expediente:opaco:001",
    version_esperada: 1,
    clave_idempotencia: CLAVE,
    artefacto_ref: "artefacto:opaco:001",
    analisis: {
      modalidad_clave: "sustitucion", categoria_ref: "categoria:rrhh:001",
      grupo_subgrupo: "A1", causa_clave: "sustitucion",
      periodo: { inicio: "2026-09-01T00:00:00Z", fin: "2027-08-31T00:00:00Z" },
      porcentaje_jornada: 1889,
      entrada_rc: { referencia: "entrada-rc:opaca:001", huella_sha256: HUELLA },
    },
  }]);
  vista.desmontar();
});

test("la jornada original no cambia con ceros iniciales y no admite exponentes", async () => {
  const valida = montar();
  await valida.enviar("07", "05", "1888");
  assert.equal(valida.solicitudes[0].analisis.porcentaje_jornada, 1888);
  valida.desmontar();

  const invalida = montar();
  await invalida.enviar("7e0", "05", "1888");
  assert.equal(invalida.solicitudes.length, 0);
  assert.match(invalida.raiz.innerHTML, /ct-analisis-porcentaje_jornada-error/u);
  invalida.desmontar();
});
