import assert from "node:assert/strict";
import test from "node:test";

import { crearBorradorAlta, crearComandoPeticionCentro, validarBorradorAlta }
  from "../modulos/contratacion-temporal/contrato.js?v=VEC_WEB_VERSION";
import { operacionRCDeclaradaValida } from "../modulos/contratacion-temporal/rc-declarada.js?v=VEC_WEB_VERSION";
import { renderizarPeticionCentro } from "./peticiones-centro.js?v=VEC_WEB_VERSION";

// El catálogo de documentos está vacío, como en desarrollo: el «Sí» no depende de él.
const catalogos = {
  esquema: "vec.contratacion_temporal.catalogos_alta.v1",
  centros: [{ referencia: "cen_sintetico_001", etiqueta: "Centro sintético", contactos: [{ referencia: "con_sintetico_001", etiqueta: "Contacto" }] }],
  categorias: [{ referencia: "cat_sintetica_001", etiqueta: "C2 sintética", grupos_subgrupos: [{ clave: "C2", etiqueta: "C2" }] }],
  motivos: [{ clave: "sustitucion", etiqueta: "Sustitución" }],
  documentos: [],
};
const contexto = { actor: { referencia: "actor:sintetico:001", nombre: "Marta Ruiz", cargo: "Secretaría", centro: "Centro sintético",
  puede_presentar: true, puede_ratificar: false }, catalogos };
const CLAVE = "f3134ee2-61af-467d-aa58-dc71f07553b6";

function borrador(rc) {
  return { ...crearBorradorAlta({ conPeticionCentro: true }), centro_ref: "cen_sintetico_001", contacto_ref: "con_sintetico_001",
    categoria_ref: "cat_sintetica_001", grupo_subgrupo: "C2", motivo_clave: "sustitucion", detalle: "Sustitución de personal",
    inicio: "2026-11-02", fin: "2026-12-31", jornada_minutos: "2250", numero_personas: "1", puesto_solicitado: "Administrativo C2", ...rc };
}
const conRC = (numero) => borrador({ rc_existe: true, rc_numero: numero, rc_fecha: "2026-10-01", rc_importe: "1234,56" });
const formulario = (b, errores = {}) => renderizarPeticionCentro({ contexto, modo: "formulario", estado: { fase: "edicion", disponible: true,
  ocupado: false, borrador: b, catalogos, errores, mensaje_clave: "estado_disponible", tipo_mensaje: "informacion" } });

test("con «Sí» y número de operación válido se envía la RC declarada sin elegir documento", () => {
  const b = conRC("220260001234");
  assert.deepEqual(validarBorradorAlta(b, catalogos).errores, {});
  const rc = crearComandoPeticionCentro(b, catalogos, CLAVE).solicitud.rc;
  assert.equal(rc.documento_ref, "rc-declarado:220260001234");
  assert.equal(rc.numero, "220260001234");
  const html = formulario(b);
  assert.match(html, /Número de operación en SICAL/u);
  assert.match(html, /RRHH o Intervención lo comprueban en SICAL/u);
  assert.match(html, /aria-describedby="ct-rc_numero-ayuda"/u);
  assert.match(html, /name="rc_numero"[^>]*maxlength="147"/u);
  assert.doesNotMatch(html, /name="rc_documento_ref"|Documento de RC incorporado/u);
});

test("se rechazan DNI, NIE, espacios y números demasiado largos", () => {
  for (const numero of ["12345678Z", "X1234567L", "1234.5678-Z", "dni-2026", "2202 6000", "ab", `A${"1".repeat(147)}`]) {
    assert.equal(operacionRCDeclaradaValida(numero), false, numero);
    const b = conRC(numero);
    assert.equal(validarBorradorAlta(b, catalogos).errores.rc_numero, "rc_operacion", numero);
    assert.throws(() => crearComandoPeticionCentro(b, catalogos, CLAVE));
  }
  assert.equal(operacionRCDeclaradaValida(`A${"1".repeat(146)}`), true);
  const html = formulario(conRC("12345678Z"), { rc_numero: "rc_operacion" });
  assert.match(html, /No escriba un DNI ni un NIE\./u);
  assert.match(html, /aria-describedby="ct-rc_numero-ayuda ct-rc_numero-error"/u);
});

test("con «No» la petición no lleva datos de RC", () => {
  const b = borrador({ rc_existe: false });
  assert.deepEqual(crearComandoPeticionCentro(b, catalogos, CLAVE).solicitud.rc, { existe: false });
  assert.equal(validarBorradorAlta({ ...b, rc_numero: "220260001234" }, catalogos).errores.rc_existe, "rc_residual");
});

test("ratificador y RRHH ven la RC como declarada y pendiente de comprobar", () => {
  const peticion = { referencia: "peticion:centro:001", version: 1, estado: "pendiente_ratificacion",
    configuracion: { solicitante: { actor_ref: "actor:sintetico:001", puesto_ref: "puesto:sintetico:001" } },
    solicitud: { centro_ref: "cen_sintetico_001", categoria_ref: "cat_sintetica_001", grupo_subgrupo: "C2", motivo_clave: "sustitucion",
      detalle: "Necesidad", periodo: { inicio: "2026-11-02T00:00:00Z", fin: "2026-12-31T00:00:00Z" },
      rc: { existe: true, numero: "220260001234", fecha: "2026-10-01T00:00:00Z", importe: { centimos: 123456, moneda: "EUR" },
        documento_ref: "rc-declarado:220260001234" } },
    creada_en: "2026-10-19T08:00:00Z" };
  const html = renderizarPeticionCentro({ contexto: { ...contexto, actor: { ...contexto.actor, puede_presentar: false, puede_ratificar: true } },
    modo: "ratificacion", peticion });
  assert.match(html, /Declarada; la comprueba RRHH · 220260001234/u);
  assert.doesNotMatch(html, /rc-declarado:|acreditad/u);
});
