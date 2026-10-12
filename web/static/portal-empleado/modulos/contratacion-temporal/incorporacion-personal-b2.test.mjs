import assert from "node:assert/strict";
import test from "node:test";
import { cargarTextos } from "../../../comun/textos.js";
import { montarIncorporacionPersonalB2 } from "./incorporacion-personal-b2.js?v=20261001-ct-a-i18n-v1";
import { crearClienteIncorporacionPersonalB2HTTP, RUTA_PLAN_B2, RUTA_CONFIRMAR_B2 } from "./cliente-http-incorporacion-personal-b2.js";
import { validarConsultaB2, validarSolicitudPlanB2, validarReciboB2, ESQUEMA_CONSULTA_B2, ESQUEMA_RECIBO_B2 } from "./contrato-incorporacion-personal-b2.js";

const expediente = "expediente:b2:1", clave = "99000000-0000-4000-8000-000000000001", sha = "a".repeat(64);
const solicitud = { expediente_ref: expediente, version_expediente: 7, puesto_ref: "puesto:1", plaza_ref: "plaza:1",
  version_plantilla_ref: "plantilla:1", version_rpt_ref: "rpt:1", regimen: { ref: "regimen:1", version: 1 },
  modalidad: { ref: "modalidad:1", version: 1 }, clase_ocupacion: "temporal", desde: "2026-10-01", hasta: "", motivo_clave: "incorporar",
  documento_ref: "documento:1", documento_sha256: sha, clave_idempotencia: clave };
const plan = { plan_ref: "plan:b2:1", version: 1, sha256: sha, intencion: solicitud };
const recibo = { esquema: ESQUEMA_RECIBO_B2, expediente_ref: expediente, plan_ref: plan.plan_ref, plan_version: 1,
  recibo_ref: "recibo:b2:original", registrada_en: "2026-10-01T10:00:00.123456Z", empleado_ref: "empleado:1",
  relacion_ref: "relacion:1", ocupacion_ref: "ocupacion:1", firma_oficial: false, eficacia_administrativa: false };
const inicial = () => ({ esquema: ESQUEMA_CONSULTA_B2, expediente_ref: expediente, version_expediente_actual: 7,
  estado: "sin_plan", prerrequisitos: [{ clave_i18n: "previo.aceptacion", cumplido: true }], opciones: {
    vacantes: [{ plaza_ref: "plaza:1", puesto_ref: "puesto:1", version_plantilla_ref: "plantilla:1", version_rpt_ref: "rpt:1",
      unidad_ref: "unidad:1", categoria_ref: "categoria:1", plaza_etiqueta: "Plaza uno", puesto_etiqueta: "Puesto uno" }],
    regimenes: [{ ref: "regimen:1", version: 1, denominacion: "Personal laboral" }],
    catalogo_clases_ocupacion: { ref: "catalogo:clases:1", version: 1, huella_sha256: sha },
    clases_ocupacion: [{ valor: "titular", texto_clave: "personal.ocupacion.clase.titular" },
      { valor: "provisional", texto_clave: "personal.ocupacion.clase.provisional" }, { valor: "temporal", texto_clave: "personal.ocupacion.clase.temporal" }],
    modalidades: [{ ref: "modalidad:1", version: 1, denominacion: "Sustitución" }], motivos: ["incorporar"],
    documentos: [{ documento_ref: "documento:1", documento_sha256: sha, etiqueta_clave_i18n: "documento.resolucion" }],
    periodo: { desde: "2026-10-01", hasta: "", fuente_ref: "periodo:1" } }, plan: null, recibo: null });
const preparado = () => ({ ...inicial(), estado: "plan_preparado", plan });
const historia = () => ({ ...preparado(), estado: "incorporacion_confirmada", version_expediente_actual: 8, recibo });
const textos = await cargarTextos("contratacion-temporal-incorporacion-personal-b2");
const etiquetas = { "previo.aceptacion": "Aceptación confirmada", "motivo.incorporar": "Incorporación", "documento.resolucion": "Resolución" };
function raiz() {
  const eventos = new Map();
  return { innerHTML: "", isConnected: true, addEventListener(k, f) { eventos.set(k, f); }, removeEventListener(k) { eventos.delete(k); },
    replaceChildren() { this.innerHTML = ""; },
    revisar(valores = { desde: "2026-10-01", hasta: "", clase_ocupacion: "2" }) {
      return eventos.get("submit")({ preventDefault() {}, target: { matches: () => true,
        elements: Object.fromEntries(Object.entries(valores).map(([k, value]) => [k, { value }])) } });
    },
    salir(campo, valor) {
      return eventos.get("focusout")({ target: { name: campo, value: valor, setAttribute() {} } });
    },
    async click(accion) { await eventos.get("click")({ target: { closest: () => ({ getAttribute: () => accion }) } }); },
  };
}
async function montar(cliente, opciones = {}) {
  const r = raiz();
  const desmontar = montarIncorporacionPersonalB2({ raiz: r, cliente, expedienteRef: expediente, versionEsperada: 7,
    textos, resolverEtiqueta: (k) => etiquetas[k], generarClave: () => clave, ...opciones });
  await new Promise((resolve) => setImmediate(resolve)); return { r, desmontar };
}
function clienteBase(extra = {}) {
  return { consultar: async () => inicial(), preparar: async () => preparado(), confirmar: async () => recibo, ...extra };
}
const response = (data, estado = 200) => new Response(JSON.stringify({ data }), { status: estado, headers: { "Content-Type": "application/json; charset=utf-8" } });

test("contrato cerrado: persona, permiso, catálogo extra, fechas imposibles y cruce de recibo se rechazan", () => {
  assert.deepEqual(validarSolicitudPlanB2(solicitud), solicitud);
  for (const extra of [{ persona_ref: "persona:intrusa" }, { actor_ref: "actor:intruso" }, { permisos: [] }]) {
    assert.throws(() => validarSolicitudPlanB2({ ...solicitud, ...extra }));
  }
  assert.throws(() => validarSolicitudPlanB2({ ...solicitud, motivo_clave: false }));
  assert.throws(() => validarSolicitudPlanB2({ ...solicitud, clase_ocupacion: false }));
  for (const desde of ["2026-02-30", "0000-10-01", "2026-10-01T00:00:00Z"]) assert.throws(() => validarSolicitudPlanB2({ ...solicitud, desde }));
  assert.throws(() => validarSolicitudPlanB2({ ...solicitud, regimen: { ...solicitud.regimen, etiqueta: "extra" } }));
  assert.throws(() => validarConsultaB2({ ...historia(), recibo: { ...recibo, plan_ref: "plan:otro" } }, expediente));
  assert.throws(() => validarReciboB2({ ...recibo, firma_oficial: true }, solicitud));
});

test("la consulta vincula las clases al catálogo de Personal y admite ausencia sin análisis", () => {
  const consulta = inicial();
  assert.equal(validarConsultaB2(consulta, expediente).opciones.catalogo_clases_ocupacion.ref, "catalogo:clases:1");
  for (const catalogo of [undefined, { ref: "", version: 0, huella_sha256: "" },
    { ref: "catalogo:clases:1", version: 1, huella_sha256: "x".repeat(64) }]) {
    const alterada = inicial();
    alterada.opciones.catalogo_clases_ocupacion = catalogo;
    assert.throws(() => validarConsultaB2(alterada, expediente));
  }
  const sinAnalisis = inicial();
  sinAnalisis.opciones.catalogo_clases_ocupacion = { ref: "", version: 0, huella_sha256: "" };
  sinAnalisis.opciones.clases_ocupacion = null;
  assert.deepEqual(validarConsultaB2(sinAnalisis, expediente).opciones.clases_ocupacion, []);
});

test("transportes fijos GET, plan y confirmar: mismos datos, sin credenciales externas ni almacenamiento", async () => {
  assert.equal(RUTA_PLAN_B2, "/api/vec/contratacion-temporal/incorporacion-personal-b2/plan/v1");
  assert.equal(RUTA_CONFIRMAR_B2, "/api/vec/contratacion-temporal/incorporacion-personal-b2/confirmar/v1");
  const calls = [];
  const cliente = crearClienteIncorporacionPersonalB2HTTP({ fetchImpl: async (ruta, o) => {
    calls.push({ ruta, ...o }); return response(calls.length === 1 ? inicial() : calls.length === 2 ? preparado() : recibo);
  } });
  await cliente.consultar(expediente); await cliente.preparar(solicitud);
  await cliente.confirmar({ expediente_ref: expediente, plan_ref: plan.plan_ref, version_plan: 1, clave_idempotencia: clave });
  assert.deepEqual(calls.map((c) => [c.ruta, c.method]), [[`${RUTA_PLAN_B2}?expediente_ref=expediente%3Ab2%3A1`, "GET"], [RUTA_PLAN_B2, "POST"], [RUTA_CONFIRMAR_B2, "POST"]]);
  for (const c of calls) for (const [k, v] of Object.entries({ credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer" })) assert.equal(c[k], v);
  assert.deepEqual(JSON.parse(calls[1].body), solicitud); assert.equal(calls[0].body, undefined);
});

test("cliente aborta pronto y cancela la respuesta tardía", async () => {
  let resolver, cancelada = false;
  const cliente = crearClienteIncorporacionPersonalB2HTTP({ fetchImpl: () => new Promise((r) => { resolver = r; }) });
  const controlador = new AbortController();
  const pendiente = cliente.consultar(expediente, { signal: controlador.signal }); controlador.abort();
  await assert.rejects(pendiente, (e) => e.codigo === "operacion_abortada");
  resolver({ body: { cancel: async () => { cancelada = true; } } });
  await new Promise((r) => setImmediate(r)); assert.equal(cancelada, true);
});

test("UI: GET no escribe, revisión sin efectos, un plan y una confirmación exactos con recibo", async () => {
  const calls = [];
  const x = await montar(clienteBase({ preparar: async (s) => { calls.push(["plan", s]); return preparado(); },
    confirmar: async (s) => { calls.push(["confirmar", s]); return recibo; } }));
  assert.equal(calls.length, 0); assert.doesNotMatch(x.r.innerHTML, /name="persona|name=".*sha256|name=".*permiso/u);
  x.r.revisar(); assert.equal(calls.length, 0); assert.match(x.r.innerHTML, /Cambiar datos/u);
  await x.r.click("registrar");
  assert.deepEqual(calls, [["plan", solicitud], ["confirmar", { expediente_ref: expediente, plan_ref: plan.plan_ref, version_plan: 1, clave_idempotencia: clave }]]);
  assert.match(x.r.innerHTML, /recibo:b2:original/u); assert.doesNotMatch(x.r.innerHTML, /data-b2-form|data-b2-accion="registrar"/u);
  x.desmontar();
});

test("UI: falta un requisito, vacantes o documento; no se ofrece una incorporación", async () => {
  for (const c of [(() => { const c = inicial(); c.prerrequisitos[0].cumplido = false; return c; })(),
    (() => { const c = inicial(); c.opciones.vacantes = []; return c; })(),
    (() => { const c = inicial(); c.opciones.documentos = []; return c; })()]) {
    const x = await montar(clienteBase({ consultar: async () => c, preparar: () => assert.fail() }));
    assert.doesNotMatch(x.r.innerHTML, /data-b2-form|data-b2-accion="registrar"/u); x.desmontar();
  }
});

test("la opción única de documento conserva su texto completo y escapado sin campo recortable", async () => {
  const c = inicial();
  c.opciones.documentos[0].etiqueta_clave_i18n = "documento.resolucion";
  const x = await montar(clienteBase({ consultar: async () => c }), { resolverEtiqueta: (k) =>
    k === "documento.resolucion" ? "Documento de formalización extenso <sin recorte>" : etiquetas[k] });
  assert.match(x.r.innerHTML, /class="ct-b2-valor-solo-lectura">Documento de formalización extenso &lt;sin recorte&gt;<\/span>/u);
  assert.doesNotMatch(x.r.innerHTML, /<input[^>]+value="Documento de formalización/u);
  x.desmontar();
});

test("UI: fechas validadas y periodo conocido conservado; ningún POST si cambia", async () => {
  const x = await montar(clienteBase({ preparar: () => assert.fail() }));
  for (const v of [{ desde: "2026-10-02", hasta: "" }, { desde: "2026-10-01", hasta: "2026-09-30" }]) {
    x.r.revisar(v); assert.match(x.r.innerHTML, /data-b2-errores/u); assert.doesNotMatch(x.r.innerHTML, /data-b2-accion="registrar"/u);
  }
  x.desmontar();
});

test("actualizar conserva selecciones por identidad y fechas editadas aunque cambie el orden", async () => {
  const antes = inicial(); antes.opciones.periodo.fuente_ref = "";
  antes.opciones.vacantes.push({ ...antes.opciones.vacantes[0], version_plantilla_ref: "plantilla:2",
    version_rpt_ref: "rpt:2", plaza_etiqueta: "Plaza dos" });
  antes.opciones.regimenes.push({ ref: "regimen:2", version: 2, denominacion: "Funcionario interino" });
  const despues = structuredClone(antes);
  despues.opciones.vacantes.reverse(); despues.opciones.regimenes.reverse(); despues.opciones.clases_ocupacion.reverse();
  let lecturas = 0, preparadoCon;
  const x = await montar(clienteBase({ consultar: async () => {
    if (++lecturas === 2) throw Object.assign(new Error("detalle privado"), { estado: 503, envelopeValido: true });
    return lecturas === 1 ? antes : despues;
  },
    preparar: async (s) => { preparadoCon = s; return { ...despues, estado: "plan_preparado", plan: { ...plan, intencion: s } }; } }));
  x.r.salir("vacante", "1"); x.r.salir("regimen", "1"); x.r.salir("clase_ocupacion", "2"); x.r.salir("desde", "2026-11-01");
  await x.r.click("consultar");
  assert.doesNotMatch(x.r.innerHTML, /detalle privado|data-b2-accion="registrar"/u);
  await x.r.click("consultar");
  assert.equal(lecturas, 3);
  assert.match(x.r.innerHTML, /<option value="0" selected>Puesto uno · Plaza dos/u);
  assert.match(x.r.innerHTML, /<option value="0" selected>Funcionario interino/u);
  x.r.revisar({ desde: "2026-11-01", hasta: "" });
  assert.match(x.r.innerHTML, /Plaza dos|Funcionario interino/u);
  await x.r.click("registrar");
  assert.equal(preparadoCon.version_plantilla_ref, "plantilla:2");
  assert.equal(preparadoCon.version_rpt_ref, "rpt:2");
  assert.equal(preparadoCon.regimen.ref, "regimen:2");
  assert.equal(preparadoCon.clase_ocupacion, "temporal");
  assert.equal(preparadoCon.desde, "2026-11-01");
  x.desmontar();
});

test("tras revisar, actualizar y cambiar datos mantiene la vacante elegida aunque se reordene", async () => {
  const antes = inicial(); antes.opciones.vacantes.push({ ...antes.opciones.vacantes[0],
    version_plantilla_ref: "plantilla:2", version_rpt_ref: "rpt:2", plaza_etiqueta: "Plaza dos" });
  const despues = structuredClone(antes); despues.opciones.vacantes.reverse();
  let lecturas = 0, preparadoCon;
  const x = await montar(clienteBase({ consultar: async () => ++lecturas === 1 ? antes : despues,
    preparar: async (s) => { preparadoCon = s; return { ...despues, estado: "plan_preparado", plan: { ...plan, intencion: s } }; } }));
  x.r.salir("vacante", "1"); x.r.revisar();
  await x.r.click("consultar"); await x.r.click("cambiar");
  assert.match(x.r.innerHTML, /<option value="0" selected>Puesto uno · Plaza dos/u);
  x.r.revisar(); await x.r.click("registrar");
  assert.equal(preparadoCon.version_plantilla_ref, "plantilla:2");
  x.desmontar();
});

test("si desaparece la clase elegida, actualizar exige nueva selección incluso tras otro GET", async () => {
  const antes = inicial(); const despues = inicial(); despues.opciones.clases_ocupacion.pop();
  let lecturas = 0, posts = 0;
  const x = await montar(clienteBase({ consultar: async () => ++lecturas === 1 ? antes : despues,
    preparar: async () => { posts++; return preparado(); } }));
  x.r.salir("clase_ocupacion", "2");
  await x.r.click("consultar"); await x.r.click("consultar");
  assert.match(x.r.innerHTML, /La opción elegida ha cambiado/u);
  assert.doesNotMatch(x.r.innerHTML, /name="clase_ocupacion"[^>]*><option value="0" selected/u);
  x.r.revisar({ desde: "2026-10-01", hasta: "" }); await x.r.click("registrar");
  assert.equal(posts, 0);
  x.desmontar();
});

test("una opción única nueva no se elige sola tras validar en blanco y volver a consultar", async () => {
  const antes = inicial(); const despues = inicial();
  despues.opciones.vacantes[0] = { ...despues.opciones.vacantes[0], plaza_ref: "plaza:2", plaza_etiqueta: "Plaza dos" };
  let lecturas = 0, posts = 0;
  const x = await montar(clienteBase({ consultar: async () => ++lecturas === 1 ? antes : despues,
    preparar: async () => { posts++; return preparado(); } }));
  await x.r.click("consultar");
  x.r.revisar({ desde: "2026-10-01", hasta: "", clase_ocupacion: "2", vacante: "" });
  await x.r.click("consultar");
  assert.match(x.r.innerHTML, /name="vacante"[^>]*aria-invalid="true"/u);
  assert.doesNotMatch(x.r.innerHTML, /class="ct-b2-valor-solo-lectura">Puesto uno · Plaza dos/u);
  x.r.revisar({ desde: "2026-10-01", hasta: "", clase_ocupacion: "2", vacante: "" });
  await x.r.click("registrar"); assert.equal(posts, 0);
  x.desmontar();
});

test("cambiar el catálogo durante la revisión impide preparar una intención obsoleta", async () => {
  const antes = inicial(); const despues = inicial();
  despues.opciones.catalogo_clases_ocupacion.version = 2;
  despues.opciones.catalogo_clases_ocupacion.huella_sha256 = "b".repeat(64);
  let lecturas = 0, posts = 0;
  const x = await montar(clienteBase({ consultar: async () => ++lecturas === 1 ? antes : despues,
    preparar: async () => { posts++; return preparado(); } }));
  x.r.revisar(); await x.r.click("consultar"); await x.r.click("registrar");
  assert.equal(posts, 0);
  assert.match(x.r.innerHTML, /La opción elegida ha cambiado/u);
  assert.doesNotMatch(x.r.innerHTML, /data-b2-accion="registrar"/u);
  x.desmontar();
});

test("una operación incierta con selección cambiada conserva la clave y solo permite comprobar", async () => {
  const antes = inicial(); const despues = inicial();
  despues.opciones.catalogo_clases_ocupacion.version = 2;
  despues.opciones.catalogo_clases_ocupacion.huella_sha256 = "b".repeat(64);
  let lecturas = 0; const claves = [];
  const x = await montar(clienteBase({ consultar: async () => ++lecturas === 1 ? antes : despues,
    preparar: async (s) => { claves.push(s.clave_idempotencia); throw Object.assign(new Error("privado"), { estado: 503 }); } }));
  x.r.revisar(); await x.r.click("registrar");
  assert.deepEqual(claves, [clave]);
  assert.match(x.r.innerHTML, /data-b2-accion="consultar"/u);
  assert.doesNotMatch(x.r.innerHTML, /data-b2-accion="retomar"|data-b2-accion="registrar"|privado/u);
  await x.r.click("retomar"); assert.deepEqual(claves, [clave]);
  x.desmontar();
});

test("denegación tras revisar borra el borrador antes de admitir otra consulta", async () => {
  let lecturas = 0, posts = 0;
  const x = await montar(clienteBase({ consultar: async () => {
    if (++lecturas === 2) throw Object.assign(new Error("privado"), { estado: 403, envelopeValido: true });
    return inicial();
  }, preparar: async () => { posts++; return preparado(); } }));
  x.r.revisar(); await x.r.click("consultar");
  assert.doesNotMatch(x.r.innerHTML, /data-b2-accion="registrar"|privado/u);
  await x.r.click("consultar"); await x.r.click("registrar");
  assert.equal(posts, 0); assert.match(x.r.innerHTML, /data-b2-form/u);
  x.desmontar();
});

test("UI: resultado incierto consulta recibo y no vuelve a escribir", async () => {
  let lecturas = 0, posts = 0;
  const x = await montar(clienteBase({ consultar: async () => ++lecturas === 1 ? inicial() : historia(),
    confirmar: async () => { posts++; throw Object.assign(new Error("causa privada"), { estado: 503 }); } }));
  x.r.revisar(); await x.r.click("registrar"); await x.r.click("retomar");
  assert.equal(posts, 1); assert.equal(lecturas, 2); assert.match(x.r.innerHTML, /recibo:b2:original/u); assert.doesNotMatch(x.r.innerHTML, /causa privada|503/u); x.desmontar();
});

test("UI: respuesta incierta y GET fallido exige comprobar antes de retomar", async () => {
  let lecturas = 0, confirmaciones = 0;
  const x = await montar(clienteBase({ consultar: async () => ++lecturas === 1 ? inicial() : Promise.reject(new Error("lectura no disponible")),
    confirmar: async () => { confirmaciones++; throw Object.assign(new Error("resultado privado"), { estado: 503 }); } }));
  x.r.revisar(); await x.r.click("registrar");
  assert.doesNotMatch(x.r.innerHTML, /data-b2-accion="registrar"/u);
  assert.match(x.r.innerHTML, /data-b2-accion="consultar"/u);
  assert.doesNotMatch(x.r.innerHTML, /data-b2-accion="retomar"/u);
  await x.r.click("retomar"); assert.equal(confirmaciones, 1);
  assert.doesNotMatch(x.r.innerHTML, /resultado privado|lectura no disponible|503/u);
  x.desmontar();
});

test("UI: recuperación sin plan conserva la misma clave y el material original", async () => {
  const solicitudes = [];
  const x = await montar(clienteBase({ preparar: async (s) => {
    solicitudes.push(s); if (solicitudes.length === 1) throw new Error(); return preparado();
  } }));
  x.r.revisar(); await x.r.click("registrar");
  assert.match(x.r.innerHTML, /Comprobar la operación/u);
  await x.r.click("retomar"); assert.equal(solicitudes.length, 2); assert.deepEqual(solicitudes[0], solicitudes[1]);
  assert.match(x.r.innerHTML, /recibo:b2:original/u); x.desmontar();
});

test("UI: denegación retira datos; expediente obsoleto impide preparar", async () => {
  for (const resultado of [Object.assign(new Error("datosprivados"), { estado: 403, envelopeValido: true }), { ...inicial(), version_expediente_actual: 8 }]) {
    const x = await montar(clienteBase({ consultar: async () => { if (resultado instanceof Error) throw resultado; return resultado; }, preparar: () => assert.fail() }));
    assert.doesNotMatch(x.r.innerHTML, /Puesto uno|Personal laboral|datosprivados|data-b2-form/u); x.desmontar();
  }
});

test("UI: el GET pendiente no atribuye un cambio al expediente; otros errores conservan su aviso", async () => {
  for (const idioma of ["es", "en"]) {
    const traduccion = await cargarTextos("contratacion-temporal-incorporacion-personal-b2", { idioma, porDefecto: "es" });
    for (const [estado, codigo, aviso] of [[409, "preparacion_pendiente", "preparacion_pendiente"],
      [409, "conflicto", "conflicto"], [503, "servicio_no_disponible", "no_disponible"]]) {
      const cliente = crearClienteIncorporacionPersonalB2HTTP({ fetchImpl: async () => new Response(JSON.stringify({ error: {
        codigo, clave_i18n: `api.contratacion_temporal.incorporacion_personal_b2.error.${codigo}`,
        correlacion_ref: "correlacion:1",
      } }), { status: estado, headers: { "Content-Type": "application/json; charset=utf-8" } }) });
      const x = await montar(cliente, { textos: traduccion });
      assert.ok(x.r.innerHTML.includes(traduccion.traducir(aviso)), `${idioma} ${codigo}`);
      if (codigo === "preparacion_pendiente") assert.ok(!x.r.innerHTML.includes(traduccion.traducir("conflicto")));
      // El estado bloqueante va en tono de aviso (ámbar), no informativo.
      assert.equal(/class="ct-estado ct-estado-aviso"[^>]*data-b2-mensaje/u.test(x.r.innerHTML), codigo === "preparacion_pendiente", `${idioma} ${codigo} tono`);
      assert.match(x.r.innerHTML, /role="status"[^>]*data-b2-mensaje/u);
      assert.match(x.r.innerHTML, /data-b2-accion="consultar"/u);
      assert.doesNotMatch(x.r.innerHTML, /data-b2-form|data-b2-accion="registrar"/u);
      x.desmontar();
    }
  }
});

test("UI: un montaje retirado o detalle cambiado descarta el resultado tardío", async () => {
  for (const desmontada of [true, false]) {
    let resolver, vigente = true;
    const x = await montar(clienteBase({ consultar: () => new Promise((r) => { resolver = r; }) }), { esVigente: () => vigente });
    if (desmontada) x.desmontar(); else vigente = false;
    const antes = x.r.innerHTML; resolver(historia()); await new Promise((r) => setImmediate(r)); assert.equal(x.r.innerHTML, antes);
    if (!desmontada) x.desmontar();
  }
});

test("catálogos ES/EN completos con traductor real y escapar las etiquetas del propietario", async () => {
  for (const idioma of ["es", "en"]) {
    const traduccion = await cargarTextos("contratacion-temporal-incorporacion-personal-b2", { idioma, porDefecto: "es" });
    assert.deepEqual(traduccion.faltantes, []);
    const c = inicial(); c.opciones.vacantes[0].puesto_etiqueta = "<img src=x onerror=alert(1)>";
    const x = await montar(clienteBase({ consultar: async () => c }), { textos: traduccion });
    assert.match(x.r.innerHTML, /&lt;img/u); assert.doesNotMatch(x.r.innerHTML, /<img/u); x.desmontar();
  }
});

test("cliente admite escapes de json.Marshal y conserva el dato propietario", async () => {
  const c = inicial(); c.opciones.vacantes[0].puesto_etiqueta = "Gestión & archivo <prueba>";
  const wireGo = JSON.stringify({ data: c }).replaceAll("&", "\\u0026").replaceAll("<", "\\u003c").replaceAll(">", "\\u003e");
  const cliente = crearClienteIncorporacionPersonalB2HTTP({ fetchImpl: async () => new Response(wireGo, {
    status: 200, headers: { "Content-Type": "application/json; charset=utf-8" },
  }) });
  const consulta = await cliente.consultar(expediente); assert.equal(consulta.opciones.vacantes[0].puesto_etiqueta, c.opciones.vacantes[0].puesto_etiqueta);
});

test("cliente acepta sólo errores reconocidos y no incluye el cuerpo privado en su error", async () => {
  for (const [estado, codigo] of [[400, "peticion_no_valida"], [422, "contenido_no_valido"], [403, "acceso_denegado"], [409, "preparacion_pendiente"], [503, "servicio_no_disponible"]]) {
    const cliente = crearClienteIncorporacionPersonalB2HTTP({ fetchImpl: async () => new Response(JSON.stringify({ error: {
      codigo, clave_i18n: `api.contratacion_temporal.incorporacion_personal_b2.error.${codigo}`, correlacion_ref: "correlacion:1",
    } }), { status: estado, headers: { "Content-Type": "application/json; charset=utf-8" } }) });
    await assert.rejects(cliente.consultar(expediente), (e) => e.estado === estado && e.envelopeValido === true && e.codigo === codigo);
  }
});


test("tipo de ocupación exige elección expresa, también si Personal ofrece una sola opción", async () => {
  for (const clases of [inicial().opciones.clases_ocupacion, [inicial().opciones.clases_ocupacion[2]]]) {
    const c = inicial(); c.opciones.clases_ocupacion = clases;
    const x = await montar(clienteBase({ consultar: async () => c, preparar: () => assert.fail() }));
    assert.match(x.r.innerHTML, /<select id="ct-b2-clase_ocupacion" name="clase_ocupacion" required/u);
    assert.doesNotMatch(x.r.innerHTML, /<option value="[012]" selected/u);
    x.r.revisar({ desde: "2026-10-01", hasta: "" });
    assert.match(x.r.innerHTML, /aria-invalid="true"/u); assert.doesNotMatch(x.r.innerHTML, /data-b2-accion="registrar"/u);
    x.desmontar();
  }
});

test("la clase elegida procede del subconjunto propietario y no se deriva de la modalidad", async () => {
  const c = inicial(); c.opciones.clases_ocupacion = [c.opciones.clases_ocupacion[1]];
  const posts = [], intencion = { ...solicitud, clase_ocupacion: "provisional" };
  const x = await montar(clienteBase({ consultar: async () => c, preparar: async (s) => {
    posts.push(s); return { ...c, estado: "plan_preparado", plan: { ...plan, intencion: s } };
  } }));
  x.r.revisar({ desde: "2026-10-01", hasta: "", clase_ocupacion: "0" });
  assert.match(x.r.innerHTML, /Tipo de ocupación \(obligatorio\)<\/dt><dd>Provisional/u);
  await x.r.click("registrar"); assert.deepEqual(posts, [intencion]); x.desmontar();
});

test("sin clases o con etiqueta desconocida se bloquea el formulario", async () => {
  for (const clases of [[], [{ valor: "otra", texto_clave: "personal.ocupacion.clase.desconocida" }]]) {
    const c = inicial(); c.opciones.clases_ocupacion = clases;
    const x = await montar(clienteBase({ consultar: async () => c, preparar: () => assert.fail() }));
    assert.doesNotMatch(x.r.innerHTML, /data-b2-form|data-b2-accion="registrar"/u); x.desmontar();
  }
});

test("recuperar una intención con otra clase no confirma el contenido divergente", async () => {
  let lecturas = 0, confirmaciones = 0;
  const x = await montar(clienteBase({ consultar: async () => ++lecturas === 1 ? inicial() : { ...preparado(), plan: {
    ...plan, intencion: { ...solicitud, clase_ocupacion: "titular" },
  } }, preparar: async () => { throw new Error(); }, confirmar: async () => { confirmaciones++; return recibo; } }));
  x.r.revisar(); await x.r.click("registrar");
  assert.equal(confirmaciones, 0); assert.match(x.r.innerHTML, /Comprobar la operación/u); x.desmontar();
});

test("catálogo propio resuelve las clases ES/EN sin depender de traducciones antiguas del montaje", async () => {
  for (const [idioma, titular] of [["es", "Titular"], ["en", "Permanent"]]) {
    const traduccion = await cargarTextos("contratacion-temporal-incorporacion-personal-b2", { idioma, porDefecto: "es" });
    const x = await montar(clienteBase(), { textos: traduccion });
    assert.match(x.r.innerHTML, new RegExp(titular, "u")); x.desmontar();
  }
});

test("régimen y modalidad se rotulan por referencia en cada idioma; sin traducción, la denominación publicada", async () => {
  const esperado = { es: ["Funcionario interino", "Interinidad por sustitución del titular"],
    en: ["Interim civil servant", "Interim appointment to replace the post holder"] };
  for (const idioma of ["es", "en"]) {
    const traduccion = await cargarTextos("contratacion-temporal-incorporacion-personal-b2", { idioma, porDefecto: "es" });
    const c = inicial();
    c.opciones.regimenes = [{ ref: "regimen:funcionario-interino", version: 1, denominacion: "Funcionario interino" },
      { ref: "regimen:otro-propio", version: 1, denominacion: "Régimen <propio>" }];
    c.opciones.modalidades = [{ ref: "modalidad:interino-sustitucion-titular", version: 1, denominacion: "Interinidad por sustitución del titular" }];
    const x = await montar(clienteBase({ consultar: async () => c }), { textos: traduccion });
    for (const rotulo of esperado[idioma]) assert.ok(x.r.innerHTML.includes(rotulo), `${idioma}: falta ${rotulo}`);
    assert.match(x.r.innerHTML, /Régimen &lt;propio&gt;/u);
    if (idioma === "en") assert.doesNotMatch(x.r.innerHTML, /Funcionario interino/u);
    x.desmontar();
  }
});
