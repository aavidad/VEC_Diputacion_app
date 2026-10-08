import assert from "node:assert/strict";
import test from "node:test";
import { cargarTextos } from "../../../comun/textos.js";
import { crearGestorIncorporacion, crearResolverEtiquetasIncorporacionB2 } from "./vista-expedientes-incorporacion.js?v=20261008-alta-rpt-circular-v5";
import { ESQUEMA_CONSULTA_B2, ESQUEMA_RECIBO_B2 } from "./contrato-incorporacion-personal-b2.js";

const expedienteRef = "expediente:b2:montaje";
const sha = "a".repeat(64);
const consulta = (version = 7, cumplido = true) => ({
  esquema: ESQUEMA_CONSULTA_B2, expediente_ref: expedienteRef, version_expediente_actual: version,
  estado: "sin_plan", prerrequisitos: [{ clave_i18n: "ct_incorporacion_b2_previo_aceptacion_persona", cumplido }],
  opciones: {
    vacantes: [{ plaza_ref: "plaza:1", puesto_ref: "puesto:1", version_plantilla_ref: "plantilla:1", version_rpt_ref: "rpt:1",
      unidad_ref: "unidad:1", categoria_ref: "categoria:1", plaza_etiqueta: "Plaza uno", puesto_etiqueta: "Puesto uno" }],
    regimenes: [{ ref: "regimen:1", version: 1, denominacion: "Personal laboral" }],
    modalidades: [{ ref: "modalidad:1", version: 1, denominacion: "Sustitución" }],
    catalogo_clases_ocupacion: { ref: "catalogo:clases:1", version: 1, huella_sha256: sha },
    clases_ocupacion: [{ valor: "temporal", texto_clave: "personal.ocupacion.clase.temporal" }],
    motivos: ["sustitucion"], documentos: [{ documento_ref: "documento:1", documento_sha256: sha,
      etiqueta_clave_i18n: "ct_incorporacion_b2_documento_formalizacion" }],
    periodo: { desde: "2026-10-01", hasta: "", fuente_ref: "periodo:1" },
  }, plan: null, recibo: null,
});

function superficie(version = 7) {
  const contenedor = { innerHTML: "", isConnected: true,
    addEventListener() {}, removeEventListener() {}, replaceChildren() { this.innerHTML = ""; },
    querySelector() { return null; } };
  const raiz = { contains: (nodo) => nodo === contenedor,
    querySelector: (selector) => selector === "[data-ct-exp-incorporacion-ejercicio]" ? contenedor : null };
  const estado = { carga: "listo", vista: "expediente", expediente: { expediente_ref: expedienteRef, version, demostracion: false },
    cuadro: { expedientes: [{ expediente_ref: expedienteRef, version, fase_clave: "nombramiento" }] } };
  return { contenedor, raiz, estado };
}

const siguienteCiclo = () => new Promise((resolver) => setImmediate(resolver));

test("motivos gobernados y reserva tienen etiqueta ES/EN sin depender de textos del código", async () => {
  for (const [idioma, motivo, vacante, reserva] of [["es", "Incorporación del expediente", "Vacante", "Reserva"],
    ["en", "Commencement for this case", "Vacant post", "Reserved"]]) {
    const [textos, personal] = await Promise.all([
      cargarTextos("contratacion-temporal-incorporacion-personal-b2", { idioma, porDefecto: "es" }),
      cargarTextos("personal", { idioma, porDefecto: "es" }),
    ]);
    const resolver = crearResolverEtiquetasIncorporacionB2(textos, personal);
    assert.equal(resolver("motivo.incorporar"), motivo);
    assert.equal(resolver("motivo.vacante"), vacante);
    assert.equal(resolver("personal.ocupacion.clase.reserva"), reserva);
    assert.equal(resolver("motivo.clave_sin_catalogo"), null);
  }
});

test("ficha real B2 hace un único GET, muestra formulario con decisión positiva y no abre legado", async () => {
  const s = superficie(); let lecturas = 0, escrituras = 0, legado = 0;
  const gestor = crearGestorIncorporacion({ raiz: s.raiz, presentador: { obtenerEstado: () => s.estado },
    incorporacionPersonalB2: { consultar: async () => { lecturas++; return consulta(); }, preparar: () => { escrituras++; }, confirmar: () => { escrituras++; } },
    incorporacionEjercicioDisponible: true, clienteLlamamiento: { prepararIncorporacionEjercicio: () => { legado++; } },
  });
  await gestor.ofrecerIncorporacionEjercicio(); await siguienteCiclo();
  assert.equal(lecturas, 1); assert.equal(escrituras, 0); assert.equal(legado, 0);
  assert.match(s.contenedor.innerHTML, /data-b2-form/u);
  assert.doesNotMatch(s.contenedor.innerHTML, /data-ct-exp-accion="consultar-incorporacion"/u);
  await gestor.ofrecerIncorporacionEjercicio(); assert.equal(lecturas, 1);
  gestor.retirar();
});

test("una clase reserva recibida de Personal usa la etiqueta de su catálogo", async () => {
  const s = superficie(); const c = consulta();
  c.opciones.clases_ocupacion = [{ valor: "reserva", texto_clave: "personal.ocupacion.clase.reserva" }];
  const gestor = crearGestorIncorporacion({ raiz: s.raiz, presentador: { obtenerEstado: () => s.estado },
    incorporacionPersonalB2: { consultar: async () => c, preparar() { assert.fail(); }, confirmar() { assert.fail(); } },
  });
  await gestor.ofrecerIncorporacionEjercicio(); await siguienteCiclo();
  assert.match(s.contenedor.innerHTML, /<option value="0">Reserva<\/option>/u);
  gestor.retirar();
});

test("v8 sin decisión positiva B2 ofrece solo consulta del protocolo anterior", async () => {
  const s = superficie(8); let lecturas = 0;
  const gestor = crearGestorIncorporacion({ raiz: s.raiz, presentador: { obtenerEstado: () => s.estado },
    incorporacionPersonalB2: { consultar: async () => { lecturas++; return consulta(8, false); }, preparar() { assert.fail(); }, confirmar() { assert.fail(); } },
    incorporacionEjercicioDisponible: true, clienteLlamamiento: { prepararIncorporacionEjercicio() { assert.fail("sin GET legado automático"); } },
  });
  await gestor.ofrecerIncorporacionEjercicio();
  assert.equal(lecturas, 1); assert.match(s.contenedor.innerHTML, /data-ct-exp-accion="consultar-incorporacion"/u);
  assert.match(s.contenedor.innerHTML, /Aceptación y persona comprobadas/u);
  assert.doesNotMatch(s.contenedor.innerHTML, /data-b2-form/u);
  gestor.retirar();
});

test("B2 recuperado en v8 conserva recibo y no ofrece el formulario anterior", async () => {
  const s = superficie(8); const c = consulta(8);
  const intencion = { expediente_ref: expedienteRef, version_expediente: 7, puesto_ref: "puesto:1", plaza_ref: "plaza:1",
    version_plantilla_ref: "plantilla:1", version_rpt_ref: "rpt:1", regimen: { ref: "regimen:1", version: 1 },
    modalidad: { ref: "modalidad:1", version: 1 }, clase_ocupacion: "temporal", desde: "2026-10-01", hasta: "",
    motivo_clave: "sustitucion", documento_ref: "documento:1", documento_sha256: sha,
    clave_idempotencia: "99000000-0000-4000-8000-000000000001" };
  c.estado = "incorporacion_confirmada";
  c.plan = { plan_ref: "plan:b2:1", version: 1, sha256: sha, intencion };
  c.recibo = { esquema: ESQUEMA_RECIBO_B2, expediente_ref: expedienteRef, plan_ref: c.plan.plan_ref, plan_version: 1,
    recibo_ref: "recibo:b2:original", registrada_en: "2026-10-01T10:00:00.123456Z", empleado_ref: "empleado:1",
    relacion_ref: "relacion:1", ocupacion_ref: "ocupacion:1", firma_oficial: false, eficacia_administrativa: false };
  const gestor = crearGestorIncorporacion({ raiz: s.raiz, presentador: { obtenerEstado: () => s.estado },
    incorporacionPersonalB2: { consultar: async () => c, preparar() { assert.fail(); }, confirmar() { assert.fail(); } },
    incorporacionEjercicioDisponible: true, clienteLlamamiento: { prepararIncorporacionEjercicio() { assert.fail(); } },
  });
  await gestor.ofrecerIncorporacionEjercicio(); await siguienteCiclo();
  assert.match(s.contenedor.innerHTML, /recibo:b2:original/u);
  assert.doesNotMatch(s.contenedor.innerHTML, /data-b2-form|consultar-incorporacion/u);
  gestor.retirar();
});

test("errores 403, 409 y 503 B2 no abren el POST legado ni filtran el error", async () => {
  for (const estadoHTTP of [403, 409, 503]) {
    const s = superficie(8); let lecturas = 0;
    const gestor = crearGestorIncorporacion({ raiz: s.raiz, presentador: { obtenerEstado: () => s.estado },
      incorporacionPersonalB2: { consultar: async () => { lecturas++; throw Object.assign(new Error("dato privado"),
        { estado: estadoHTTP, envelopeValido: true }); }, preparar() { assert.fail(); }, confirmar() { assert.fail(); } },
      incorporacionEjercicioDisponible: true, clienteLlamamiento: { prepararIncorporacionEjercicio() { assert.fail(); } },
    });
    await gestor.ofrecerIncorporacionEjercicio(); await siguienteCiclo();
    assert.equal(lecturas, 1);
    assert.doesNotMatch(s.contenedor.innerHTML, /dato privado|data-b2-form|consultar-incorporacion/u);
    assert.match(s.contenedor.innerHTML, /data-b2-mensaje/u);
    gestor.retirar();
  }
});

test("retirar la ficha antes del GET B2 impide montar datos tardíos", async () => {
  const s = superficie(); let resolver;
  const gestor = crearGestorIncorporacion({ raiz: s.raiz, presentador: { obtenerEstado: () => s.estado },
    incorporacionPersonalB2: { consultar: () => new Promise((terminar) => { resolver = terminar; }),
      preparar() { assert.fail(); }, confirmar() { assert.fail(); } },
  });
  const pendiente = gestor.ofrecerIncorporacionEjercicio();
  gestor.retirar(); resolver(consulta()); await pendiente;
  assert.doesNotMatch(s.contenedor.innerHTML, /data-b2-form/u);
});
