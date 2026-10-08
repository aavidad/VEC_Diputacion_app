import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { API_AJUSTES, ErrorAjustes, cambiosDesdeCabeza, crearClienteAjustes, normalizarFechaMadrid,
  renderizarAjustes, validarLecturaAjustes, validarReciboAjustes, valoresParaEditar } from "./ajustes.js";
import { cargarTextosAjustes } from "./ajustes-i18n.js";

await cargarTextosAjustes();

const lectura = () => ({ data: { esquema: "vec.contratacion_temporal.reglas.ajustes.v1",
  catalogo_id: "vec.contratacion_temporal.reglas.ajustes", version_esperada: 2, puede_ajustar: true,
  activacion: { estado: "activa" }, consultada_en: "2026-10-08T10:00:00Z", hay_mas_programados: false,
  cabeza: { version: 2, ajustes: { "c03.plazo_fiscalizacion": { cantidad: "10", cantidad_urgente: "5" } },
    vigente_desde: "2026-09-30T12:00:00Z", publicada_en: "2026-09-29T12:00:00Z" },
  vigente_hoy: { version: 2, ajustes: { "c03.plazo_fiscalizacion": { cantidad: "10", cantidad_urgente: "5" } },
    vigente_desde: "2026-09-30T12:00:00Z", publicada_en: "2026-09-29T12:00:00Z" }, programados: [],
  motivos: [{ clave: "respuesta_rrhh_duda", texto_clave: "ajustesMotivo_respuesta_rrhh_duda" }],
  reglas: [{ clave: "c03.plazo_fiscalizacion", etiqueta: "Plazo de fiscalización", unidad: "dias_habiles", cantidad: 10,
    valores: { cantidad: "10", cantidad_urgente: "5", unidad: "dias_habiles" },
    edicion: { campos: ["cantidad", "cantidad_urgente"], opciones_unidad: ["dias_habiles"], opciones_computo: [],
      cantidad_minima: 1, cantidad_maxima: 30 }, ajuste_no_aplicable: false }],
  historial: [{ version: 2, vigente_desde: "2026-09-30T12:00:00Z", publicada_en: "2026-09-29T12:00:00Z", motivo_clave: "respuesta_rrhh_duda",
    referencia: "Duda 63", nota: "Revisado", recibo_ref: "recibo:ejemplo",
    cambios: [{ regla_clave: "c03.plazo_fiscalizacion", campo: "cantidad", anterior: "12", nuevo: "10" }] }], hay_mas: false } });

const recibo = () => ({ data: { esquema: "vec.contratacion_temporal.reglas.ajustes.v1", recibo: { recibo_ref: "recibo:syntetico", version: 3, vigente_desde: "2026-09-30T13:00:00Z", publicada_en: "2026-09-29T13:00:00Z",
  clave_idempotencia: "c3102148-38bb-4bbc-9b9b-b60b34521ec2", huella_sha256: "a".repeat(64),
  consumo_huella_sha256: "b".repeat(64), decision_ref: "decision:syntetica",
  auditoria_ref: "auditoria:syntetica" }, replay: false } });

const reglaNoAplicable = () => ({ clave: "c04.plazo_subsanacion", etiqueta: "Plazo de subsanación",
  ajuste_no_aplicable: true, edicion: { campos: ["cantidad"], opciones_unidad: [], opciones_computo: [],
    cantidad_minima: 1, cantidad_maxima: 30 } });

test("la lectura valida versión, campos, motivos catalogados y evita valores ambiguos", () => {
  assert.equal(validarLecturaAjustes(lectura()).reglas[0].valores.cantidad_urgente, "5");
  const sinMotivo = lectura(); sinMotivo.data.motivos[0].texto_clave = "desconocida";
  assert.throws(() => validarLecturaAjustes(sinMotivo), ErrorAjustes);
  const sinUrgente = lectura(); delete sinUrgente.data.reglas[0].valores.cantidad_urgente;
  assert.throws(() => validarLecturaAjustes(sinUrgente), ErrorAjustes);
  const otraVersion = lectura(); otraVersion.data.version_esperada = -1;
  assert.throws(() => validarLecturaAjustes(otraVersion), ErrorAjustes);
  const sinActivacion = lectura(); delete sinActivacion.data.activacion;
  assert.throws(() => validarLecturaAjustes(sinActivacion), ErrorAjustes);
  const activacionConHuella = lectura(); activacionConHuella.data.activacion.huella_sha256 = "a".repeat(64);
  assert.throws(() => validarLecturaAjustes(activacionConHuella), ErrorAjustes);
});

test("una base sin publicar o inactiva muestra historia sin afirmar valores vigentes", () => {
  for (const estado of ["sin_publicar", "inactiva"]) {
    const respuesta = lectura();
    respuesta.data.activacion.estado = estado;
    respuesta.data.puede_ajustar = false;
    respuesta.data.reglas = [];
    const modelo = validarLecturaAjustes(respuesta);
    const html = renderizarAjustes(modelo);
    assert.match(html, /Ver historial/u);
    assert.match(html, /Duda 63/u);
    assert.match(html, /Regla anterior 1/u);
    assert.doesNotMatch(html, /<code>c03\.plazo_fiscalizacion<\/code>/u);
    assert.match(html, /Cantidad: 12 → 10/u);
    assert.match(html, /Revisado/u);
    assert.doesNotMatch(html, /data-ajustes-editar/u);
    assert.doesNotMatch(html, /Plazo de fiscalización/u);
    assert.match(html, estado === "sin_publicar" ? /base de plazos aprobada y publicada/u : /base de plazos está desactivada/u);
    const falsa = lectura(); falsa.data.activacion.estado = estado;
    assert.throws(() => validarLecturaAjustes(falsa), ErrorAjustes);
  }
});

test("la historia sin catálogo distingue reglas sin exponer sus claves", () => {
  const sinBase = lectura();
  sinBase.data.activacion.estado = "inactiva";
  sinBase.data.puede_ajustar = false;
  sinBase.data.reglas = [];
  sinBase.data.historial[0].cambios.push({ regla_clave: "c04.plazo_subsanacion", campo: "cantidad", anterior: "8", nuevo: "9" });
  const html = renderizarAjustes(validarLecturaAjustes(sinBase));
  assert.match(html, /Regla anterior 1/u);
  assert.match(html, /Regla anterior 2/u);
  assert.doesNotMatch(html, /<code>c0[34]\.plazo_/u);
});

test("la cabeza programada conserva CAS independiente del valor vigente", () => {
  const respuesta = lectura();
  respuesta.data.version_esperada = 3;
  respuesta.data.cabeza = { version: 3, ajustes: { "c03.plazo_fiscalizacion": { cantidad: "7", cantidad_urgente: "4" } },
    vigente_desde: "2027-01-15T09:30:00Z", publicada_en: "2026-10-08T10:00:00Z" };
  respuesta.data.programados = [respuesta.data.cabeza];
  const datos = validarLecturaAjustes(respuesta);
  assert.equal(datos.vigente_hoy.version, 2);
  const html = renderizarAjustes(datos);
  assert.match(html, /Cambios programados/u);
  assert.match(html, /Versión 2 en vigor/u);
  assert.match(html, /Última versión publicada: 3/u);
  assert.match(html, /Versión 3: se aplicará desde/u);
  assert.match(html, /10 Días hábiles/u);
  assert.match(html, /7 Días hábiles/u);
  assert.match(html, /Valor en vigor ahora/u);
  assert.match(html, /Valor programado desde/u);
  assert.match(html, /Valores de esta versión/u);
  assert.deepEqual(cambiosDesdeCabeza(datos, datos.reglas[0], { cantidad: "7", cantidad_urgente: "4" }), []);
  assert.deepEqual(cambiosDesdeCabeza(datos, datos.reglas[0], { cantidad: "7", cantidad_urgente: "3" }),
    [{ regla_clave: "c03.plazo_fiscalizacion", campo: "cantidad_urgente", nuevo: "3" }]);
  assert.deepEqual(cambiosDesdeCabeza(datos, datos.reglas[0], { cantidad: "10", cantidad_urgente: "4" }),
    [{ regla_clave: "c03.plazo_fiscalizacion", campo: "cantidad", nuevo: "10" }]);
  assert.deepEqual(valoresParaEditar(datos, datos.reglas[0]), { cantidad: "7", cantidad_urgente: "4", unidad: "dias_habiles" });
  const revision = renderizarAjustes(datos, { reglaActiva: "c03.plazo_fiscalizacion", fase: "revision",
    borrador: { cantidad: "6", cantidad_urgente: "4", motivo_clave: "respuesta_rrhh_duda", inicio_efecto: "futuro",
      fecha_madrid: "2027-01-15T10:30", vigente_desde: "2027-01-15T09:30:00Z" } });
  assert.match(revision, /7 Días hábiles → 6 Días hábiles/u);
  assert.match(revision, /name="inicio_efecto" value="ahora"[^>]*disabled/u);
  assert.doesNotMatch(revision, /10 Días hábiles → 6 Días hábiles/u);
  assert.throws(() => normalizarFechaMadrid("2027-01-14T10:30", Date.parse("2026-10-08"), Date.parse(datos.cabeza.vigente_desde)), ErrorAjustes);
  assert.equal(normalizarFechaMadrid("2027-01-15T10:30", Date.parse("2026-10-08"), Date.parse(datos.cabeza.vigente_desde)),
    datos.cabeza.vigente_desde);
  const parcial = { ...datos, hay_mas_programados: true };
  assert.match(renderizarAjustes(parcial), /Hay más cambios programados/u);
  assert.throws(() => validarLecturaAjustes({ data: { ...respuesta.data, version_esperada: 2 } }), ErrorAjustes);
});

test("una regla con ajuste no aplicable omite valores sin ocultar las reglas válidas", async () => {
  const mixta = lectura();
  mixta.data.reglas.push(reglaNoAplicable());
  const cliente = crearClienteAjustes(async () => new Response(JSON.stringify(mixta), { status: 200 }));
  const datos = await cliente.leer();
  assert.equal(datos.reglas.length, 2);
  const html = renderizarAjustes(datos, { reglaActiva: "c04.plazo_subsanacion" });
  assert.match(html, /Plazo de fiscalización/u);
  assert.match(html, /10 Días hábiles/u);
  assert.match(html, /Plazo de subsanación/u);
  assert.match(html, /RRHH debe revisarlo/u);
  assert.match(html, /data-ajustes-editar="c04.plazo_subsanacion" disabled/u);
  assert.doesNotMatch(html, /data-ajustes-form="c04.plazo_subsanacion"/u);
  const sinEdicion = lectura();
  sinEdicion.data.reglas.push({ clave: "c04.plazo_subsanacion", etiqueta: "Plazo de subsanación", ajuste_no_aplicable: true });
  assert.equal(validarLecturaAjustes(sinEdicion).reglas.length, 2);
  for (const campo of ["valores", "cantidad", "cantidad_urgente", "unidad", "computo"]) {
    const filtrada = lectura();
    filtrada.data.reglas.push({ ...reglaNoAplicable(), [campo]: campo === "valores" ? { cantidad: "10" } : 10 });
    assert.throws(() => validarLecturaAjustes(filtrada), ErrorAjustes, campo);
  }
});

test("el cliente GET y POST conserva origen, clave y señal; distingue conflicto y dependencia", async () => {
  const pedidos = [];
  const cliente = crearClienteAjustes(async (url, opciones) => {
    pedidos.push({ url, opciones });
    return new Response(JSON.stringify(opciones.method === "GET" ? lectura() : recibo()), { status: opciones.method === "GET" ? 200 : 201 });
  });
  await cliente.leer({ antesDeVersion: 2 });
  assert.equal(pedidos[0].url, `${API_AJUSTES}?limite=20&antes_de_version=2`);
  assert.equal(pedidos[0].opciones.credentials, "same-origin");
  assert.equal(pedidos[0].opciones.cache, "no-store");
  const comando = { clave_idempotencia: "c3102148-38bb-4bbc-9b9b-b60b34521ec2", version_esperada: 2,
    cambios: [{ regla_clave: "c03.plazo_fiscalizacion", campo: "cantidad", nuevo: "7" }], motivo_clave: "respuesta_rrhh_duda" };
  assert.equal((await cliente.guardar(comando)).recibo.version, 3);
  const respuestaAjena = crearClienteAjustes(async () => new Response(JSON.stringify({ data: { ...recibo().data,
    recibo: { ...recibo().data.recibo, clave_idempotencia: "otra" } } }), { status: 201 }));
  await assert.rejects(respuestaAjena.guardar(comando), ErrorAjustes);
  assert.deepEqual(Object.keys(pedidos[1].opciones.headers), ["Accept", "Content-Type"]);
  assert.deepEqual(JSON.parse(pedidos[1].opciones.body), comando);
  for (const [estado, codigo] of [[409, "ajustesConflicto"], [422, "ajustesValorInvalido"], [403, "ajustesSinPermiso"], [503, "ajustesNoDisponible"]]) {
    const fallido = crearClienteAjustes(async () => new Response("{}", { status: estado }));
    await assert.rejects(fallido.guardar(comando), (error) => error.codigo === codigo);
  }
  assert.equal(validarReciboAjustes(recibo()).recibo.recibo_ref, "recibo:syntetico");
});

test("la pantalla muestra resumen, historia y recibo, escapa datos; desactiva cambios sin motivos", () => {
  const datos = validarLecturaAjustes(lectura());
  const html = renderizarAjustes(datos, { reglaActiva: "c03.plazo_fiscalizacion", fase: "revision",
    borrador: { cantidad: "7", cantidad_urgente: "4", motivo_clave: "respuesta_rrhh_duda",
      referencia: "Acuerdo 12/2026", nota: "Revisión anual" }, recibo: recibo().data.recibo, aviso: "Cambio guardado" });
  assert.match(html, /Plazo de fiscalización/u);
  assert.match(html, /10 Días hábiles/u);
  assert.match(html, /10 Días hábiles → 7 Días hábiles/u);
  assert.match(html, /data-ajustes-revision/u);
  assert.match(html, /Desde ahora/u);
  assert.match(html, /fecha futura/u);
  assert.match(html, /Respuesta de RRHH a una duda/u);
  assert.match(html, /Acuerdo 12\/2026/u);
  assert.match(html, /Revisión anual/u);
  assert.match(html, /data-ajustes-recibo/u);
  assert.match(html, /data-ajustes-estado/u);
  assert.match(html, /Duda 63/u);
  assert.match(html, /Autor no disponible/u);
  assert.match(html, /<summary>Ver justificante<\/summary>/u);
  assert.match(html, /Auditoría:/u);
  const sinMotivos = { ...datos, motivos: [] };
  assert.match(renderizarAjustes(sinMotivos), /data-ajustes-editar="c03.plazo_fiscalizacion" disabled/u);
  assert.match(renderizarAjustes(datos, { bloqueado: true, error: true, aviso: "Conflicto" }), /data-ajustes-reintentar/u);
  const motivoRetirado = renderizarAjustes({ ...datos, motivos: [{ clave: "acuerdo_instruccion", texto_clave: "ajustesMotivo_acuerdo_instruccion" }] },
    { reglaActiva: "c03.plazo_fiscalizacion", fase: "revision", borrador: { cantidad: "7", cantidad_urgente: "4", motivo_clave: "respuesta_rrhh_duda" } });
  assert.match(motivoRetirado, /Motivo anterior no disponible/u);
  assert.match(motivoRetirado, /data-ajustes-enviar disabled/u);
  const revision = { ...datos, reglas: [reglaNoAplicable()] };
  assert.doesNotMatch(renderizarAjustes(revision), /0 Días hábiles/u);
  assert.match(renderizarAjustes(revision), /RRHH debe revisarlo/u);
  const malicioso = { ...datos, reglas: [{ ...datos.reglas[0], etiqueta: "<img src=x>" }] };
  assert.ok(!renderizarAjustes(malicioso).includes("<img src=x>"));
  assert.match(renderizarAjustes(malicioso), /&lt;img src=x&gt;/u);
  const camposMaliciosos = renderizarAjustes(datos, { reglaActiva: "c03.plazo_fiscalizacion", fase: "revision",
    borrador: { cantidad: "7", cantidad_urgente: "4", motivo_clave: "respuesta_rrhh_duda",
      referencia: "<script>x</script>", nota: "<img src=x>" } });
  assert.ok(!camposMaliciosos.includes("<script>x</script>"));
  assert.ok(!camposMaliciosos.includes("<img src=x>"));
  const futuro = renderizarAjustes(datos, { reglaActiva: "c03.plazo_fiscalizacion", fase: "revision",
    borrador: { cantidad: "7", cantidad_urgente: "4", motivo_clave: "respuesta_rrhh_duda", inicio_efecto: "futuro",
      fecha_madrid: "2027-01-15T10:30", vigente_desde: "2027-01-15T09:30:00Z" } });
  assert.match(futuro, /15 ene 2027/u);
  assert.match(futuro, /type="datetime-local"/u);
  const autor = { ...datos, historial: [{ ...datos.historial[0], actor_nombre: "<img src=x>" }] };
  assert.match(renderizarAjustes(autor), /&lt;img src=x&gt;/u);
  assert.doesNotMatch(renderizarAjustes(autor), /<img src=x>/u);
});

test("fecha futura de Madrid se normaliza sin horas ambiguas ni inexistentes", () => {
  assert.equal(normalizarFechaMadrid("2027-01-15T10:30", Date.parse("2026-01-01")), "2027-01-15T09:30:00Z");
  assert.equal(normalizarFechaMadrid("2027-01-15T10:30:30", Date.parse("2026-01-01")), "2027-01-15T09:30:30Z");
  assert.equal(normalizarFechaMadrid("2027-07-15T10:30", Date.parse("2026-01-01")), "2027-07-15T08:30:00Z");
  for (const local of ["2027-03-28T02:30", "2027-10-31T02:30", "2026-01-01T10:30", "invalida"]) {
    assert.throws(() => normalizarFechaMadrid(local, Date.parse("2026-10-08")), ErrorAjustes);
  }
});

test("la fecha prellenada conserva segundos y supera fracciones de la cabeza", () => {
  const futuro = lectura();
  futuro.data.version_esperada = 3;
  futuro.data.cabeza = { version: 3, ajustes: { "c03.plazo_fiscalizacion": { cantidad: "7" } },
    vigente_desde: "2027-01-15T09:30:30.123456Z", publicada_en: "2026-10-08T10:00:00Z" };
  futuro.data.programados = [futuro.data.cabeza];
  const html = renderizarAjustes(validarLecturaAjustes(futuro), { reglaActiva: "c03.plazo_fiscalizacion" });
  assert.match(html, /value="2027-01-15T10:30:31"/u);
  assert.match(html, /type="datetime-local" step="1"/u);
  assert.equal(normalizarFechaMadrid("2027-01-15T10:30:31", Date.parse("2026-10-08"), Date.parse(futuro.data.cabeza.vigente_desde)),
    "2027-01-15T09:30:31Z");
});

test("el cliente admite fecha futura sólo como instante UTC y usa una ruta fija por módulo", async () => {
  const pedidos = [];
  const cliente = crearClienteAjustes(async (url, opciones) => {
    pedidos.push({ url, opciones }); return new Response(JSON.stringify(recibo()), { status: 201 });
  });
  const comando = { clave_idempotencia: "c3102148-38bb-4bbc-9b9b-b60b34521ec2", version_esperada: 2,
    cambios: [{ regla_clave: "c03.plazo_fiscalizacion", campo: "cantidad", nuevo: "7" }],
    motivo_clave: "respuesta_rrhh_duda", vigente_desde: "2027-01-15T09:30:00Z" };
  await cliente.guardar(comando);
  assert.equal(JSON.parse(pedidos[0].opciones.body).vigente_desde, comando.vigente_desde);
  await assert.rejects(cliente.guardar({ ...comando, vigente_desde: "2027-01-15T10:30:00+01:00" }), ErrorAjustes);
  await assert.rejects(cliente.guardar({ ...comando, actor_ref: "actor:falso" }), ErrorAjustes);
  await assert.rejects(cliente.guardar({ ...comando, cambios: [{ regla_clave: undefined, campo: "cantidad", nuevo: "7" }] }), ErrorAjustes);
  assert.throws(() => crearClienteAjustes(fetch, 10000, { modulo: "bolsa", ruta: "https://externo.invalid" }), TypeError);
});

test("los textos de ajuste existen en ambos idiomas y el módulo no incluye frases visibles", () => {
  const ficheros = ["es", "en"].map((idioma) => JSON.parse(readFileSync(new URL(`../../textos/${idioma}/reglas-plazos.json`, import.meta.url), "utf8")).general);
  assert.deepEqual(Object.keys(ficheros[0]).sort(), Object.keys(ficheros[1]).sort());
  for (const texto of ["ajustesMotivo_respuesta_rrhh_duda", "ajustesConflicto", "ajustesEfecto", "ajustesEfectoFuturo", "ajustesAuditoria"]) {
    assert.ok(ficheros.every((fichero) => fichero[texto]), texto);
  }
});
