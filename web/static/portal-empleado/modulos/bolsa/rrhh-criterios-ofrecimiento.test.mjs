import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { criteriosOfrecimientoValidos, validarPoliticaEditable,
  cargarCatalogoCriteriosOfrecimiento, RUTA_CATALOGO_CRITERIOS_OFRECIMIENTO } from "./rrhh-plazos-api.js";
import { renderizarCriteriosOfrecimiento, cambiarCriteriosOfrecimiento } from "./rrhh-criterios-ofrecimiento.js";
import { crearSuperficieRRHHPlazos } from "./rrhh-plazos-ui.js";
import { crearTraductorRRHHPlazos } from "./rrhh-plazos-i18n.js";

const textos = JSON.parse(readFileSync(new URL("../../../textos/es/bolsa-ofertas.json", import.meta.url))).politica_criterios_ofrecimiento;
const criterio = () => ({
  fuente_normativa: "Reglamento publicado",
  mejora: { habilitada: true, modalidades_destino: ["vacante"], causas_destino: ["vacante"], maximo_vacante_meses: 36,
    reduccion_por_modalidades_previas: ["sustitucion"], condiciones_actuales: [
    { modalidad: "sustitucion", requiere_baja_comprobada: true,
      meses_transcurridos: { meses: 9, incluye_limite: true } },
    { modalidad: "programa", meses_duracion_inicial: { meses: 9, incluye_limite: false } },
  ] },
});
const politica = () => ({
  plazo: { unidad: "dias_habiles", cantidad: 2, inicio: "notificacion", computo: "administrativo", municipio_sede: "18087" },
  adjudicacion: { criterio: "orden_vigente", elegibilidad: "disposicion_en_plazo" },
  no_cubierta: { accion: "llamamiento_directo", condicion: "sin_disposiciones_elegibles" },
});

test("conserva criterio publicado, regla desactivada y límites al validar la política", () => {
  const actual = politica();
  assert.deepEqual(validarPoliticaEditable(actual), actual);
  actual.criterios_ofrecimiento = criterio();
  actual.criterios_ofrecimiento.mejora.habilitada = false;
  assert.deepEqual(validarPoliticaEditable(actual), actual);
  assert.equal(actual.criterios_ofrecimiento.mejora.condiciones_actuales[0].meses_transcurridos.meses, 9);
  for (const invalido of [0, -1, 1.5, NaN]) {
    const otro = structuredClone(actual);
    otro.criterios_ofrecimiento.mejora.condiciones_actuales[0].meses_transcurridos.meses = invalido;
    assert.equal(criteriosOfrecimientoValidos(otro.criterios_ofrecimiento), false);
    assert.throws(() => validarPoliticaEditable(otro));
  }
  const sinTope = criterio(); delete sinTope.mejora.maximo_vacante_meses;
  assert.equal(criteriosOfrecimientoValidos(sinTope), false);
  const sinCausa = criterio(); delete sinCausa.mejora.causas_destino;
  assert.equal(criteriosOfrecimientoValidos(sinCausa), false);
  const sinCondicion = criterio(); sinCondicion.mejora.reduccion_por_modalidades_previas = ["acumulacion_tareas"];
  assert.equal(criteriosOfrecimientoValidos(sinCondicion), false);
  const legada = politica(); legada.criterios_ofrecimiento = { ...criterio(), modalidad_no_aceptada: "renuncia", modalidades_aceptables: ["vacante"] };
  assert.equal(validarPoliticaEditable(legada).criterios_ofrecimiento.modalidad_no_aceptada, undefined);
  assert.equal(validarPoliticaEditable(legada).criterios_ofrecimiento.modalidades_aceptables, undefined);
});

test("renderiza solo el criterio publicado con rótulos y controles deshabilitados", () => {
  assert.equal(renderizarCriteriosOfrecimiento(undefined, textos), "");
  const html = renderizarCriteriosOfrecimiento(criterio(), textos, true);
  assert.match(html, /Sustitución/u);
  assert.match(html, /Programa temporal/u);
  assert.match(html, /name="criterio_mejora_habilitada"[^>]*checked[^>]*disabled/u);
  assert.match(html, /name="criterio_meses_transcurridos_0"[^>]*disabled/u);
  assert.match(html, /name="criterio_maximo_vacante_meses"[^>]*value="36"[^>]*disabled/u);
  assert.match(html, /Causas de la oferta/u);
  assert.doesNotMatch(html, /criterio_modalidad_no_aceptada/u);
  assert.doesNotMatch(html, /sustitucion/u);
  const ajeno = criterio(); ajeno.mejora.modalidades_destino = ["tipo_nuevo"];
  assert.match(renderizarCriteriosOfrecimiento(ajeno, textos), /Modalidad publicada 1/u);
});

test("desactivar y reactivar mejora conserva condiciones; solo edita límites existentes", () => {
  const actual = criterio();
  const cambio = (name, value, checked) => cambiarCriteriosOfrecimiento(actual, { name, value, checked });
  assert.equal(cambio("criterio_mejora_habilitada", "", false), true);
  assert.equal(actual.mejora.habilitada, false);
  assert.equal(cambio("criterio_meses_transcurridos_0", "8"), true);
  assert.equal(cambio("criterio_meses_transcurridos_incluye_0", "", false), true);
  assert.equal(cambio("criterio_meses_duracion_inicial_0", "8"), false);
  assert.equal(cambio("criterio_mejora_habilitada", "", true), true);
  assert.deepEqual(actual.mejora.condiciones_actuales[0].meses_transcurridos,
    { meses: 8, incluye_limite: false });
  assert.equal(cambio("criterio_maximo_vacante_meses", "30"), true);
  assert.equal(actual.mejora.maximo_vacante_meses, 30);
  assert.equal(cambio("criterio_modalidad_no_aceptada", "renuncia"), false);
});

test("RRHH publica el borrador de criterios con la operación y el recibo existentes", async () => {
  const inicial = politica(); inicial.criterios_ofrecimiento = criterio();
  inicial.plazas = { llamada: "simultanea", respuesta_horas: 24, tras_renuncia: "siguiente_en_orden" };
  const sobre = (version, politicaPublicada, recibo) => ({ esquema: "vec.bolsa.rrhh.politica-ofertas.v1",
    bolsa_ref: "bolsa:1", version, configurada: true, ejemplo: true,
    huella_sha256: "a".repeat(64), publicada_en: "2026-10-15T10:00:00Z",
    politica: structuredClone(politicaPublicada), ...(recibo ? { recibo_ref: recibo } : {}) });
  let comando;
  const superficie = crearSuperficieRRHHPlazos({
    cliente: { consultar: async () => ({ ok: true, politica: sobre(1, inicial) }),
      consultarCapacidad: async () => ({ ok: true, puede_publicar: true }),
      publicar: async (recibido) => { comando = recibido; return { ok: true,
        politica: sobre(2, recibido.politica, "recibo:criterios:2") }; } },
    traducir: crearTraductorRRHHPlazos(), cargarEjemplo: async () => null,
    cargarPlazo: async () => null, cargarConfirmacion: async () => null,
    generarClave: () => "criterios-2",
  });
  superficie.activar("bolsa:1");
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.match(superficie.renderizar(), /name="criterio_mejora_habilitada"/u);
  const control = (name, value, checked) => ({ target: { name, value, checked, closest: () => ({}) } });
  superficie.manejarCambio(control("criterio_mejora_habilitada", "", false));
  superficie.manejarCambio(control("criterio_meses_transcurridos_0", "8"));
  superficie.manejarCambio(control("criterio_fuente_normativa", "Nueva norma publicada"));
  const formulario = { closest: () => formulario, reportValidity: () => true };
  superficie.manejarSubmit({ target: formulario, preventDefault() {} });
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.equal(comando.version_esperada, 1);
  assert.equal(comando.clave_idempotencia, "criterios-2");
  assert.equal(comando.politica.criterios_ofrecimiento.mejora.habilitada, false);
  assert.equal(comando.politica.criterios_ofrecimiento.mejora.condiciones_actuales[0].meses_transcurridos.meses, 8);
  assert.match(superficie.renderizar(), /recibo:criterios:2/u);
});

test("la propuesta sale solo del catálogo público validado y sin credenciales", async () => {
  const catalogo = JSON.parse(readFileSync(new URL("./criterios-ofrecimiento/v1.json", import.meta.url)));
  let pedido;
  const leer = (valor) => cargarCatalogoCriteriosOfrecimiento({ fetchImpl: async (ruta, opciones) => {
    pedido = { ruta, opciones };
    return { status: 200, redirected: false, headers: { get: () => "application/json" }, json: async () => valor };
  } });
  assert.deepEqual(await leer(catalogo), catalogo.criterios_ofrecimiento);
  assert.equal(pedido.ruta, RUTA_CATALOGO_CRITERIOS_OFRECIMIENTO);
  assert.equal(pedido.opciones.credentials, "omit");
  assert.equal(pedido.opciones.cache, "no-store");
  assert.equal(pedido.opciones.redirect, "error");
  assert.equal(pedido.opciones.referrerPolicy, "no-referrer");
  await assert.rejects(leer({ ...catalogo, criterios_ofrecimiento: { fuente_normativa: "Sin regla" } }));
  await assert.rejects(cargarCatalogoCriteriosOfrecimiento({ fetchImpl: async () => { throw new Error("sin red"); } }));
});

test("RRHH añade la propuesta solo a su borrador; fallo del catálogo conserva el resto del formulario", async () => {
  const base = politica(); base.plazas = { llamada: "simultanea", respuesta_horas: 24, tras_renuncia: "siguiente_en_orden" };
  const sobre = { esquema: "vec.bolsa.rrhh.politica-ofertas.v1", bolsa_ref: "bolsa:1", version: 1,
    configurada: true, ejemplo: true, huella_sha256: "a".repeat(64), publicada_en: "2026-10-15T10:00:00Z",
    politica: base };
  let falla = true;
  const superficie = crearSuperficieRRHHPlazos({ cliente: {
    consultar: async () => ({ ok: true, politica: sobre }),
    consultarCapacidad: async () => ({ ok: true, puede_publicar: true }),
  }, traducir: crearTraductorRRHHPlazos(), cargarEjemplo: async () => null,
  cargarPlazo: async () => null, cargarConfirmacion: async () => null,
  cargarCriterios: async () => { if (falla) throw new Error("catálogo inaccesible"); return criterio(); } });
  superficie.activar("bolsa:1");
  await new Promise((resolver) => setTimeout(resolver, 0));
  const click = () => superficie.manejarClick({ target: { closest: () => ({ disabled: false,
    dataset: { rrhhPlazosAccion: "anadir-criterios" } }) } });
  assert.match(superficie.renderizar(), /Añadir criterios para vacantes/u);
  click(); await new Promise((resolver) => setTimeout(resolver, 0));
  assert.match(superficie.renderizar(), /No se han podido cargar los criterios/u);
  assert.equal(superficie.estado().borrador.plazo.cantidad, 2);
  falla = false; click(); await new Promise((resolver) => setTimeout(resolver, 0));
  assert.equal(superficie.estado().borrador.criterios_ofrecimiento.mejora.maximo_vacante_meses, 36);
  assert.equal(superficie.estado().vigente.politica.criterios_ofrecimiento, undefined);
  assert.match(superficie.renderizar(), /Comprobar la excepción para vacantes/u);
});
