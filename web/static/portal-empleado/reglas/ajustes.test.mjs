import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { API_AJUSTES, ErrorAjustes, crearClienteAjustes, renderizarAjustes, validarLecturaAjustes, validarReciboAjustes } from "./ajustes.js";

const lectura = () => ({ data: { esquema: "vec.contratacion_temporal.reglas.ajustes.v1",
  catalogo_id: "vec.contratacion_temporal.reglas.ajustes", version_esperada: 2, puede_ajustar: true,
  activacion: { estado: "activa" },
  motivos: [{ clave: "respuesta_rrhh_duda", texto_clave: "ajustesMotivo_respuesta_rrhh_duda" }],
  reglas: [{ clave: "c03.plazo_fiscalizacion", etiqueta: "Plazo de fiscalización", unidad: "dias_habiles", cantidad: 10,
    valores: { cantidad: "10", cantidad_urgente: "5", unidad: "dias_habiles" },
    edicion: { campos: ["cantidad", "cantidad_urgente"], opciones_unidad: ["dias_habiles"], opciones_computo: [],
      cantidad_minima: 1, cantidad_maxima: 30 }, ajuste_no_aplicable: false }],
  historial: [{ version: 2, vigente_desde: "2026-09-30T12:00:00Z", motivo_clave: "respuesta_rrhh_duda",
    referencia: "Duda 63", nota: "Revisado", recibo_ref: "recibo:ejemplo",
    cambios: [{ regla_clave: "c03.plazo_fiscalizacion", campo: "cantidad", anterior: "12", nuevo: "10" }] }], hay_mas: false } });

const recibo = () => ({ data: { esquema: "vec.contratacion_temporal.reglas.ajustes.v1", recibo: { recibo_ref: "recibo:syntetico", version: 3, vigente_desde: "2026-09-30T13:00:00Z",
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
    assert.match(html, /Identificador de la regla: <code>c03\.plazo_fiscalizacion<\/code>/u);
    assert.match(html, /Cantidad: 12 → 10/u);
    assert.match(html, /Revisado/u);
    assert.doesNotMatch(html, /data-ajustes-editar/u);
    assert.doesNotMatch(html, /Plazo de fiscalización/u);
    assert.match(html, estado === "sin_publicar" ? /base de plazos aprobada y publicada/u : /base de plazos está desactivada/u);
    const falsa = lectura(); falsa.data.activacion.estado = estado;
    assert.throws(() => validarLecturaAjustes(falsa), ErrorAjustes);
  }
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
  assert.match(html, /Respuesta de RRHH a una duda/u);
  assert.match(html, /Acuerdo 12\/2026/u);
  assert.match(html, /Revisión anual/u);
  assert.match(html, /data-ajustes-recibo/u);
  assert.match(html, /data-ajustes-estado/u);
  assert.match(html, /Duda 63/u);
  assert.match(html, /Persona de RRHH/u);
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
});

test("los textos de ajuste existen en ambos idiomas y el módulo no incluye frases visibles", () => {
  const ficheros = ["es", "en"].map((idioma) => JSON.parse(readFileSync(new URL(`../../textos/${idioma}/reglas.json`, import.meta.url), "utf8")).general);
  assert.deepEqual(Object.keys(ficheros[0]).sort(), Object.keys(ficheros[1]).sort());
  for (const texto of ["ajustesMotivo_respuesta_rrhh_duda", "ajustesConflicto", "ajustesEfecto", "ajustesAuditoria"]) {
    assert.ok(ficheros.every((fichero) => fichero[texto]), texto);
  }
});
