import test from "node:test";
import assert from "node:assert/strict";
import {
  crearBorradorAlta, crearComandoAlta, jornadaVisibleDesdeMinutos,
  minutosDesdeJornadaVisible, validarBorradorAlta, validarCatalogosAlta,
} from "./contrato.js?v=20261008-alta-circular-v3";
import { crearAltaClienteHTTP } from "./cliente-http-alta.js?v=20261008-alta-circular-v3";
import { crearPresentadorAltaContratacionTemporal } from "./presentador.js?v=20261008-alta-circular-v3";
import { montarAltaContratacionTemporal, renderizarAltaContratacionTemporal,
  seleccionarPuestoPublicadoRPT } from "./vista.js?v=20261008-alta-circular-v3";
import { cargarMensajesNecesidadesAlta } from "./i18n.js?v=20261008-alta-circular-v3";

const CLAVE = "12345678-1234-4abc-8def-1234567890ab";
const HUELLA = "a".repeat(64);
const CAMPOS_RPT = ["puesto_codigo", "rpt_catalogo_ref", "rpt_catalogo_huella_sha256"];

function causa(clave, campos, fechaFin = "obligatoria", extras = {}) {
  const { campos_obligatorios: obligatorios = campos, ...otros } = extras;
  return { clave, etiqueta_clave: `ct.necesidad.${clave}`,
    fuente_ref: "circular:ct:20260219", fuente_url: "https://www.dipgra.es/circular.pdf",
    regla_ref: `regla:ct:${clave}`, fecha_fin: fechaFin, maximo_meses: 36,
    campos_permitidos: ["numero_personas", ...campos],
    campos_obligatorios: ["numero_personas", ...obligatorios], ...otros };
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
      fuente_ref: "circular:ct:20260219", fuente_url: "https://www.dipgra.es/circular.pdf",
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
  numero_personas: "1",
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

test("el número de personas lo aporta el usuario y viaja dentro de necesidad.campos", () => {
  const vacio = crearBorradorAlta({ conNumeroMOAD: true, conNecesidad: true,
    jornadaReferenciaMinutos: 2100 });
  assert.equal(vacio.numero_personas, "");
  for (const cuenta of ["1", "2", "4294967295"]) {
    const actual = borrador({ numero_personas: cuenta, puesto_codigo: "217",
      rpt_catalogo_ref: "rpt-dipgra-2026", rpt_catalogo_huella_sha256: HUELLA });
    assert.equal(validarBorradorAlta(actual, catalogos()).valido, true);
    const comando = crearComandoAlta(actual, catalogos(), CLAVE);
    assert.equal(comando.necesidad.campos.numero_personas, cuenta);
    assert.equal(Object.hasOwn(comando, "numero_personas"), false);
    assert.equal(Object.hasOwn(comando.solicitud, "numero_personas"), false);
  }
  for (const cuenta of ["", "0", "1.5", "abc", "01", "4294967296"]) {
    const actual = borrador({ numero_personas: cuenta });
    const validacion = validarBorradorAlta(actual, catalogos());
    assert.equal(validacion.valido, false, cuenta);
    assert.equal(validacion.errores.numero_personas,
      cuenta === "" ? "texto_obligatorio" : "numero_personas", cuenta);
    assert.throws(() => crearComandoAlta(actual, catalogos(), CLAVE), undefined, cuenta);
  }
});

test("la obligatoriedad de plaza procede del catálogo de la causa", () => {
  const actual = catalogos();
  actual.necesidades.causas = actual.necesidades.causas.map((dato) => dato.clave === "vacante"
    ? { ...dato, campos_permitidos: [...dato.campos_permitidos, "plaza_codigo"],
      campos_obligatorios: [...dato.campos_obligatorios, "plaza_codigo"] } : dato);
  const sinPlaza = borrador({ puesto_codigo: "217", rpt_catalogo_ref: "rpt-dipgra-2026",
    rpt_catalogo_huella_sha256: HUELLA });
  assert.equal(validarBorradorAlta(sinPlaza, actual).errores.plaza_codigo, "texto_obligatorio");
  assert.equal(validarBorradorAlta({ ...sinPlaza, plaza_codigo: "PL-217" }, actual).valido, true);
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

test("programa acepta exactamente una vía de financiación y un fin que cubra el periodo", () => {
  const programa = borrador({ motivo_clave: "programa_temporal", programa_denominacion: "Archivo",
    programa_fin: "2026-11-01", proyecto_codigo: "PR-2026-7",
    financiacion_ref: "financiacion:sintetica:001", rc_ref: "rc:sintetica:001" });
  assert.equal(validarBorradorAlta(programa, catalogos()).valido, true);
  assert.equal(validarBorradorAlta({ ...programa,
    intervencion_ref: "intervencion:sintetica:001" }, catalogos()).errores.rc_ref, "uno_de");
  assert.equal(validarBorradorAlta({ ...programa, programa_fin: "2026-10-15" },
    catalogos()).errores.programa_fin, "campo_necesidad");
  assert.equal(validarBorradorAlta({ ...programa, programa_fin: "2026-09-30" },
    catalogos()).errores.programa_fin, "campo_necesidad");
});

test("el máximo de meses de cada causa usa fechas civiles y no admite el día exclusivo", () => {
  const actual = catalogos();
  actual.necesidades.causas = actual.necesidades.causas.map((dato) => dato.clave === "acumulacion_tareas"
    ? { ...dato, maximo_meses: 9 } : dato);
  const acumulacion = borrador({ motivo_clave: "acumulacion_tareas", inicio: "2026-05-31",
    fin: "2027-02-28", justificacion_temporal: "Trabajo temporal del centro" });
  assert.equal(validarBorradorAlta(acumulacion, actual).valido, true);
  assert.equal(validarBorradorAlta({ ...acumulacion, fin: "2027-03-01" }, actual).errores.fin,
    "periodo_maximo_necesidad");
});

test("la jornada visible conserva minutos enteros sin redondear una fracción", () => {
  assert.equal(minutosDesdeJornadaVisible("37,5"), 2250);
  assert.equal(minutosDesdeJornadaVisible("37.5"), 2250);
  assert.equal(minutosDesdeJornadaVisible("37:30"), 2250);
  assert.equal(jornadaVisibleDesdeMinutos("2250"), "37:30");
  assert.equal(minutosDesdeJornadaVisible("37,01"), null);
  assert.equal(minutosDesdeJornadaVisible("0:01"), 1);
  const actual = borrador({ puesto_codigo: "217", rpt_catalogo_ref: "rpt-dipgra-2026",
    rpt_catalogo_huella_sha256: HUELLA, jornada_minutos: "2250" });
  assert.equal(crearComandoAlta(actual, catalogos(), CLAVE).necesidad.jornada_minutos, 2250);
});

test("los textos de la necesidad se cargan del catálogo del idioma solicitado", async () => {
  const es = await cargarMensajesNecesidadesAlta("es");
  const en = await cargarMensajesNecesidadesAlta("en");
  assert.equal(es.jornada_minutos, "Jornada semanal (horas y minutos)");
  assert.equal(en.jornada_minutos, "Weekly working time (hours and minutes)");
  assert.equal(es.numero_personas, "Número de personas solicitadas");
  assert.equal(en.numero_personas, "Number of people requested");
  assert.match(es.necesidad_ayuda, /Selección Temporal decidirá/u);
  assert.match(en.necesidad_ayuda, /Temporary Staff Selection will decide/u);
});

test("los formularios CT existentes pintan sin leer necesidades y Alta v3 espera su catálogo", async () => {
  const v2 = validarCatalogosAlta(catalogos());
  const v1 = { esquema: "vec.contratacion_temporal.catalogos_alta.v1",
    centros: v2.centros, categorias: v2.categorias, documentos: v2.documentos,
    motivos: v2.motivos, numero_expediente_moad: v2.numero_expediente_moad };
  const raiz = () => ({ innerHTML: "", addEventListener() {}, removeEventListener() {},
    querySelector: () => null, setAttribute() {}, removeAttribute() {} });
  const base = { capacidad: "contratacion_temporal.solicitud.crear",
    ejecutor: async () => { throw new Error("el montaje no debe enviar"); } };
  let lecturas = 0;
  const raizAnterior = raiz();
  const desmontarAnterior = montarAltaContratacionTemporal({ raiz: raizAnterior,
    presentador: crearPresentadorAltaContratacionTemporal({ ...base, catalogos: v1 }),
    cargarTextosNecesidades: () => { lecturas += 1; throw new Error("lectura ajena"); } });
  assert.match(raizAnterior.innerHTML, /data-modulo="contratacion-temporal"/u);
  assert.equal(lecturas, 0);
  desmontarAnterior();

  let resolver;
  const textosPendientes = new Promise((confirmar) => { resolver = confirmar; });
  const raizAlta = raiz();
  const desmontarAlta = montarAltaContratacionTemporal({ raiz: raizAlta,
    presentador: crearPresentadorAltaContratacionTemporal({ ...base, catalogos: v2 }),
    cargarTextosNecesidades: () => { lecturas += 1; return textosPendientes; } });
  assert.equal(raizAlta.innerHTML, "", "no pinta claves de necesidades antes de leerlas");
  await Promise.resolve();
  assert.equal(lecturas, 1);
  resolver(await cargarMensajesNecesidadesAlta());
  await new Promise((confirmar) => setImmediate(confirmar));
  assert.match(raizAlta.innerHTML, /Jornada semanal \(horas y minutos\)/u);
  desmontarAlta();
});

test("si fallan los textos del alta se ofrece reintento sin mostrar claves crudas", async () => {
  const textos = await cargarMensajesNecesidadesAlta();
  const escuchas = new Map();
  const raiz = { innerHTML: "", addEventListener: (tipo, manejar) => escuchas.set(tipo, manejar),
    removeEventListener: (tipo) => escuchas.delete(tipo), querySelector: () => null,
    contains: () => true, setAttribute() {}, removeAttribute() {} };
  let lecturas = 0;
  const desmontar = montarAltaContratacionTemporal({ raiz,
    presentador: crearPresentadorAltaContratacionTemporal({ catalogos: catalogos(),
      capacidad: "contratacion_temporal.solicitud.crear", ejecutor: async () => null }),
    cargarTextosNecesidades: async () => {
      lecturas += 1;
      if (lecturas === 1) throw new Error("lectura fallida");
      return textos;
    } });
  await new Promise((confirmar) => setImmediate(confirmar));
  assert.match(raiz.innerHTML, /data-ct-accion="reintentar-textos"/u);
  assert.doesNotMatch(raiz.innerHTML, /necesidad_leyenda|ct\.necesidad\./u);
  const boton = { dataset: { ctAccion: "reintentar-textos" } };
  await escuchas.get("click")({ target: { closest: (selector) => selector === "[data-ct-accion]" ? boton : null },
    preventDefault() {} });
  await new Promise((confirmar) => setImmediate(confirmar));
  assert.equal(lecturas, 2);
  assert.match(raiz.innerHTML, /Jornada semanal \(horas y minutos\)/u);
  desmontar();
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
  const mensajes = await cargarMensajesNecesidadesAlta();
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
  assert.equal(inicial.borrador.numero_personas, "");
  presentador.actualizarBorrador({ ...inicial.borrador, motivo_clave: "vacante" });
  assert.throws(() => renderizarAltaContratacionTemporal(presentador.obtenerEstado()),
    /textos del alta de necesidades no preparados/u);
  const html = renderizarAltaContratacionTemporal(presentador.obtenerEstado(), { mensajes });
  assert.match(html, /Cobertura de un puesto vacante/);
  assert.match(html, /Buscar puesto/);
  assert.match(html, /name="jornada_horas"[^>]*value="35:00"/u);
  assert.match(html, /name="numero_personas" type="number" min="1" max="4294967295" step="1" inputmode="numeric" required/u);
  assert.ok(html.indexOf('name="numero_personas"') < html.indexOf('name="puesto_busqueda"'),
    "conserva el orden publicado de campos");
  assert.doesNotMatch(html, /modalidad jurídica/i);
  assert.equal(presentador.prepararRevision(borrador({ puesto_codigo: "217" })), false);
  assert.equal(presentador.obtenerEstado().errores.rpt_catalogo_ref, "texto_obligatorio");
  const listo = borrador({ puesto_codigo: "217", rpt_catalogo_ref: "rpt-dipgra-2026",
    rpt_catalogo_huella_sha256: HUELLA });
  assert.equal(presentador.prepararRevision(listo), true);
  assert.match(renderizarAltaContratacionTemporal(presentador.obtenerEstado(), { mensajes }),
    /<dt>Número de personas solicitadas<\/dt><dd>1<\/dd>/u);
  await presentador.enviar();
  assert.equal(enviados.length, 1);
  assert.equal(presentador.obtenerEstado().recibo.recibo_ref, "recibo:ct:001");
});
