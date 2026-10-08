import test from "node:test";
import assert from "node:assert/strict";
import {
  crearBorradorAlta, crearComandoAlta, validarBorradorAlta, validarCatalogosAlta,
} from "./contrato.js";
import { crearAltaClienteHTTP } from "./cliente-http-alta.js";
import { crearPresentadorAltaContratacionTemporal } from "./presentador.js";
import { renderizarAltaContratacionTemporal, seleccionarPuestoPublicadoRPT } from "./vista.js";

const CLAVE = "12345678-1234-4abc-8def-1234567890ab";
const HUELLA = "a".repeat(64);
const CAMPOS_RPT = ["puesto_codigo", "rpt_catalogo_ref", "rpt_catalogo_huella_sha256"];

function causa(clave, campos, fechaFin = "obligatoria", extras = {}) {
  return { clave, etiqueta_clave: `ct.necesidad.${clave}`,
    fuente_ref: "circular:ct:20260508", fuente_url: "https://www.dipgra.es/circular.pdf",
    regla_ref: `regla:ct:${clave}`, fecha_fin: fechaFin, maximo_meses: 36,
    campos_permitidos: campos, campos_obligatorios: campos, ...extras };
}

function catalogos() {
  return {
    esquema: "vec.contratacion_temporal.catalogos_alta.v2",
    numero_expediente_moad: { referencia: "catalogo:moad:ct", version: 1,
      patron: "^[0-9]{4}/[1-9][0-9]{0,9}$", ejemplo: "2026/12345" },
    centros: [{ referencia: "centro:sintetico:001", etiqueta: "Centro sintético",
      contactos: [{ referencia: "contacto:sintetico:001", etiqueta: "Elena Morales" }] }],
    categorias: [{ referencia: "categoria:sintetica:001", etiqueta: "Técnica",
      grupos_subgrupos: [{ clave: "A2", etiqueta: "A2" }] }],
    documentos: [{ referencia: "documento:peticion:001", etiqueta: "Petición" }],
    necesidades: { referencia: "catalogo:ct:necesidades:001", version: 2,
      huella_sha256: HUELLA, es_ejemplo: true,
      fuente_ref: "circular:ct:20260508", fuente_url: "https://www.dipgra.es/circular.pdf",
      jornada_referencia_minutos: 2100, jornada_fuente_ref: "operador:ct:provisional",
      causas: [
        causa("vacante", CAMPOS_RPT),
        causa("sustitucion", [...CAMPOS_RPT, "titular_ref"], "opcional",
          { causa_fin: "reincorporacion_titular", campos_obligatorios: CAMPOS_RPT }),
        causa("acumulacion_tareas", ["justificacion_temporal"]),
        causa("programa_temporal", ["programa_denominacion", "programa_fin",
          "proyecto_codigo", "financiacion_ref", "rc_ref", "intervencion_ref"],
        "obligatoria", { campos_obligatorios: ["programa_denominacion", "programa_fin",
          "proyecto_codigo", "financiacion_ref"], uno_de: [["rc_ref", "intervencion_ref"]] }),
      ] },
  };
}

function borrador(datos = {}) {
  return { ...crearBorradorAlta({ conNumeroMOAD: true, conNecesidad: true,
    jornadaReferenciaMinutos: 2100 }),
  numero_expediente_moad: "2026/12345", centro_ref: "centro:sintetico:001",
  contacto_ref: "contacto:sintetico:001", categoria_ref: "categoria:sintetica:001",
  grupo_subgrupo: "A2", motivo_clave: "vacante", detalle: "Refuerzo solicitado por el centro.",
  inicio: "2026-10-01", fin: "2026-11-01", rc_existe: false,
  documentos_adjuntos: ["documento:peticion:001"], ...datos };
}

test("el catálogo v2 conserva cuatro causas, jornada publicada y contrato cerrado", () => {
  const actual = validarCatalogosAlta(catalogos());
  assert.equal(actual.necesidades.jornada_referencia_minutos, 2100);
  assert.deepEqual(actual.motivos.map(({ clave }) => clave), ["vacante", "sustitucion",
    "acumulacion_tareas", "programa_temporal"]);
  assert.throws(() => validarCatalogosAlta({ ...catalogos(), motivos: [{ clave: "otra" }] }));
  assert.throws(() => validarCatalogosAlta({ ...catalogos(), necesidades: {
    ...catalogos().necesidades, jornada_referencia_minutos: 0 } }));
});

test("sin publicación RPT exacta no confirma vacante ni inventa versión", () => {
  const actual = borrador({ puesto_codigo: "217" });
  const validacion = validarBorradorAlta(actual, catalogos());
  assert.equal(validacion.valido, false);
  assert.equal(validacion.errores.rpt_catalogo_ref, "texto_obligatorio");
  assert.throws(() => crearComandoAlta(actual, catalogos(), CLAVE));
});

test("programa exige una referencia publicada y sustitución admite causa de fin", () => {
  const programa = borrador({ motivo_clave: "programa_temporal", programa_denominacion: "Refuerzo de archivo",
    programa_fin: "2027-03-31", proyecto_codigo: "PR-2026-7",
    financiacion_ref: "financiacion:sintetica:001" });
  assert.equal(validarBorradorAlta(programa, catalogos()).errores.rc_ref, "uno_de");
  assert.equal(validarBorradorAlta({ ...programa, intervencion_ref: "intervencion:sintetica:001" },
    catalogos()).valido, true);
  const sustitucion = borrador({ motivo_clave: "sustitucion", fin: "", puesto_codigo: "217",
    rpt_catalogo_ref: "rpt-dipgra-2026", rpt_catalogo_huella_sha256: HUELLA });
  assert.equal(validarBorradorAlta(sustitucion, catalogos()).valido, true);
  assert.deepEqual(crearComandoAlta(sustitucion, catalogos(), CLAVE).solicitud.periodo,
    { inicio: "2026-10-01T00:00:00Z", causa_fin: "reincorporacion_titular" });
});

test("el alta v3 envía necesidad estructurada y conserva la clave de reintento", () => {
  const actual = borrador({ puesto_codigo: "217", rpt_catalogo_ref: "rpt-dipgra-2026",
    rpt_catalogo_huella_sha256: HUELLA });
  const comando = crearComandoAlta(actual, catalogos(), CLAVE);
  assert.equal(comando.esquema, "vec.ct.alta_necesidad.v1");
  assert.equal(comando.clave_idempotencia, CLAVE);
  assert.equal(comando.necesidad.jornada_minutos, 2100);
  assert.equal(comando.necesidad.campos.puesto_codigo, "217");
  assert.equal(comando.necesidad.campos.rpt_catalogo_ref, "rpt-dipgra-2026");
  assert.equal(Object.hasOwn(comando.necesidad.campos, "rpt_catalogo_version"), false);
  assert.equal(comando.solicitud.motivo_clave, "vacante");
  assert.equal(Object.hasOwn(comando.solicitud, "necesidad"), false);
});

test("la consulta v2 solo usa la ruta declarada y no degrada un fallo", async () => {
  const llamadas = [];
  const cliente = crearAltaClienteHTTP({ ejecutar: async (opcion) => {
    llamadas.push(opcion.ruta);
    throw new Error("fuente caída");
  }, validarOpciones: () => ({ signal: undefined }) });
  await assert.rejects(cliente.obtenerCatalogosNecesidadesAlta());
  assert.deepEqual(llamadas, ["/api/vec/contratacion-temporal/catalogos-alta?version=2"]);
});

test("el selector toma el par publicado de Personal sin fabricar versión", () => {
  const pagina = { total: 1, items: [{ codigo: "217", denominacion: "Administrativo/a" }],
    fuente: { importacion: "rpt-dipgra-2026", huella_sha256: HUELLA } };
  assert.deepEqual(seleccionarPuestoPublicadoRPT(pagina, "217"), {
    codigo: "217", denominacion: "Administrativo/a",
    rpt_catalogo_ref: "rpt-dipgra-2026", rpt_catalogo_huella_sha256: HUELLA,
  });
  assert.equal(seleccionarPuestoPublicadoRPT(pagina, "218"), null);
  assert.throws(() => seleccionarPuestoPublicadoRPT({ ...pagina, fuente: {
    ...pagina.fuente, huella_sha256: "" } }, "217"));
});

test("la pantalla v2 ofrece causas y campos publicados y conserva recibo real", async () => {
  const enviados = [];
  const presentador = crearPresentadorAltaContratacionTemporal({ catalogos: catalogos(),
    capacidad: "contratacion_temporal.solicitud.crear",
    ejecutor: async (comando) => {
      enviados.push(comando);
      return { expediente_ref: "expediente:ct:001", numero_visible: "2026/12345",
        version: 1, recibo_ref: "recibo:ct:001", confirmada_en: "2026-10-08T08:00:00Z" };
    }, generarClaveIdempotencia: () => CLAVE });
  const inicial = presentador.obtenerEstado();
  assert.equal(inicial.borrador.jornada_minutos, "2100");
  presentador.actualizarBorrador({ ...inicial.borrador, motivo_clave: "vacante" });
  const html = renderizarAltaContratacionTemporal(presentador.obtenerEstado());
  assert.match(html, /Cobertura de un puesto vacante/);
  assert.match(html, /Buscar puesto/);
  assert.doesNotMatch(html, /modalidad jurídica/i);
  assert.equal(presentador.prepararRevision(borrador({ puesto_codigo: "217" })), false);
  assert.equal(presentador.obtenerEstado().errores.rpt_catalogo_ref, "texto_obligatorio");
  const listo = borrador({ puesto_codigo: "217", rpt_catalogo_ref: "rpt-dipgra-2026",
    rpt_catalogo_huella_sha256: HUELLA });
  assert.equal(presentador.prepararRevision(listo), true);
  await presentador.enviar();
  assert.equal(enviados.length, 1);
  assert.equal(presentador.obtenerEstado().recibo.recibo_ref, "recibo:ct:001");
});
