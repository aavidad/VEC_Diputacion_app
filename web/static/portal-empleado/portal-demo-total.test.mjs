import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { obtenerDatosPresentacion } from "./datos-presentacion.js";
import { crearAdaptadorPresentacion, OPERACIONES_PRESENTACION } from "./portal-presentacion-adaptador.js";
import { crearClienteBorradoresPresentacion } from "./portal-borradores-demo-cliente.js";
import { crearUtilidadesVista } from "./portal-vistas-utilidades.js";
import { crearVistasConvocatorias } from "./portal-vistas-convocatorias.js";
import { crearVistasBaremacion } from "./portal-vistas-baremacion.js";
import { crearVistasOperaciones } from "./portal-vistas-operaciones.js";
import { crearVistasGobierno } from "./portal-vistas-gobierno.js";

const directorio = new URL("./", import.meta.url);
const nombresModulos = [
  "portal-presentacion-adaptador.js", "portal-borradores-demo-cliente.js",
  "portal-vistas-utilidades.js", "portal-vistas-convocatorias.js",
  "portal-vistas-baremacion.js", "portal-vistas-operaciones.js", "portal-vistas-gobierno.js",
];
const fuentes = Object.fromEntries(await Promise.all(nombresModulos.map(async (nombre) => [
  nombre, await readFile(new URL(nombre, directorio), "utf8"),
])));
const [portal, eventos, html] = await Promise.all([
  readFile(new URL("portal.js", directorio), "utf8"),
  readFile(new URL("portal-eventos.js", directorio), "utf8"),
  readFile(new URL("index.html", directorio), "utf8"),
]);

test("la navegación comparte el cierre de menú y no referencia una función fuera de ámbito", () => {
  assert.match(portal, /function cerrarMenuMovil\(\)/);
  assert.match(portal, /crearControladorPortal\(\{[\s\S]{0,220}cerrarMenuMovil/);
  assert.match(eventos, /const \{[\s\S]{0,160}cerrarMenuMovil/);
  assert.doesNotMatch(eventos, /function cerrarMenuMovil\(\)/);
});

function utilidades(esPresentacion) {
  return crearUtilidadesVista({
    escaparHTML: (valor) => String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;"),
    numero: (valor) => String(valor ?? 0),
    claseEstado: () => "neutro",
    encabezadoVista: (_sobrelinea, titulo, descripcion, acciones = "") => `<header><h2>${titulo}</h2><p>${descripcion}</p>${acciones}</header>`,
    esPresentacion: () => esPresentacion.valor,
  });
}

function referenciasSinteticas(valor, ruta = "datos", salida = []) {
  if (Array.isArray(valor)) {
    valor.forEach((item, indice) => referenciasSinteticas(item, `${ruta}[${indice}]`, salida));
    return salida;
  }
  if (valor === null || typeof valor !== "object") return salida;
  for (const [clave, contenido] of Object.entries(valor)) {
    const esReferencia = clave === "id" || clave === "referencia" || clave === "expediente"
      || clave === "recibo" || clave === "evidencia" || clave === "objetivo"
      || clave === "objeto" || clave === "destinatario" || clave === "necesidad"
      || clave.endsWith("_ref") || clave.endsWith("_id");
    if (esReferencia && typeof contenido === "string") salida.push([`${ruta}.${clave}`, contenido]);
    referenciasSinteticas(contenido, `${ruta}.${clave}`, salida);
  }
  return salida;
}

test("el adaptador DEMO es volátil, cerrado y genera recibos reconstruibles", () => {
  const fecha = new Date("2026-07-18T12:00:00Z");
  const iniciales = obtenerDatosPresentacion();
  const uno = crearAdaptadorPresentacion({ datosIniciales: iniciales, reloj: () => fecha });
  const dos = crearAdaptadorPresentacion({ datosIniciales: iniciales, reloj: () => fecha });
  const recibo = uno.ejecutar({ operacion: "aceptar-merito", objetivo: "DEMO-MER-001", motivo: "Prueba" });
  assert.deepEqual(recibo, {
    referencia: "DEMO-REC-000001", actor: "DEMO-PERFIL-ADMIN-FUNCIONAL-BOLSA-01",
    instante: "2026-07-18T12:00:00.000Z", operacion: "aceptar-merito",
    objetivo: "DEMO-MER-001", resultado: "Aceptado", motivo: "Prueba", efectos_reales: false,
  });
  assert.equal(uno.obtenerDatos().meritos_revision[0].estado, "Aceptado");
  assert.equal(dos.obtenerDatos().meritos_revision[0].estado, "Pendiente",
    "una nueva composición equivale a recargar y no conserva el estado anterior");
  assert.throws(() => uno.ejecutar({ operacion: "operacion-no-permitida" }), /no permitida/);
  const copia = uno.obtenerDatos();
  copia.meritos_revision[0].estado = "Manipulado";
  assert.equal(uno.obtenerDatos().meritos_revision[0].estado, "Aceptado");
});

test("toda referencia sintética del modelo empieza literalmente por DEMO-", () => {
  const referencias = referenciasSinteticas(obtenerDatosPresentacion());
  assert.ok(referencias.length > 60);
  for (const [ruta, valor] of referencias) {
    assert.match(valor, /^DEMO-/, `${ruta} no identifica inequívocamente la demostración`);
  }
  const serializado = JSON.stringify(obtenerDatosPresentacion());
  assert.doesNotMatch(serializado, /\b\d{8}[A-Z]\b/);
  assert.doesNotMatch(serializado, /María Pérez|Juan López|Ana Ruiz/);
});

test("las operaciones de todas las vistas están enumeradas en el adaptador", () => {
  const codigoVistas = Object.entries(fuentes)
    .filter(([nombre]) => nombre.startsWith("portal-vistas-") && nombre !== "portal-vistas-utilidades.js")
    .map(([, codigo]) => codigo).join("\n");
  const declaradas = new Set(OPERACIONES_PRESENTACION);
  const usadas = new Set([
    ...codigoVistas.matchAll(/botonOperacion\("[^"]+",\s*"([^"]+)"/g),
    ...codigoVistas.matchAll(/data-operacion="([^"]+)"/g),
  ].map((coincidencia) => coincidencia[1]));
  assert.ok(usadas.size >= 25);
  for (const operacion of usadas) assert.ok(declaradas.has(operacion), `falta ${operacion}`);
});

test("la misma vista se conserva y solo cambia la capacidad del adaptador", () => {
  const modo = { valor: true };
  const u = utilidades(modo);
  const datos = obtenerDatosPresentacion();
  const vistas = crearVistasConvocatorias(u);
  const demo = vistas.renderizarSolicitudes(datos);
  modo.valor = false;
  const real = vistas.renderizarSolicitudes({ ...datos, solicitudes: [] });
  for (const texto of ["Solicitudes, admisión y subsanación", "Bandeja de solicitudes", "Solicitudes presentadas"]) {
    assert.match(demo, new RegExp(texto));
    assert.match(real, new RegExp(texto));
  }
  assert.match(demo, /data-operacion="admitir-solicitud"/);
  assert.match(real, /data-operacion="publicar-lista-provisional"[^>]+disabled/);
  assert.match(real, /Funcionalidad no conectada/);
});

test("todas las capacidades producen una pantalla completa con datos sintéticos", () => {
  const modo = { valor: true };
  const u = utilidades(modo);
  const datos = obtenerDatosPresentacion();
  const estado = { elaboracionSeleccionada: "DEMO-BOL-014" };
  const convocatorias = crearVistasConvocatorias(u);
  const baremacion = crearVistasBaremacion(u);
  const operaciones = crearVistasOperaciones(u);
  const gobierno = crearVistasGobierno(u);
  const salidas = [
    convocatorias.renderizarConvocatorias(datos, estado), convocatorias.renderizarSolicitudes(datos),
    baremacion.renderizarMeritos(datos), baremacion.renderizarBaremacion(datos),
    baremacion.renderizarAlegaciones(datos), operaciones.renderizarImportacion(datos),
    operaciones.renderizarLlamamientos(datos), operaciones.renderizarContratos(datos),
    operaciones.renderizarDocumentos(datos), operaciones.renderizarComunicaciones(datos),
    gobierno.renderizarEstadisticas(datos), gobierno.renderizarAuditoria(datos),
    gobierno.renderizarConfiguracion(datos),
  ];
  assert.equal(salidas.length, 13);
  for (const salida of salidas) {
    assert.match(salida, /<h2>/);
    assert.match(salida, /<section/);
    assert.match(salida, /DEMO|sintétic|presentación/i);
  }
});

test("el cliente DEMO de borradores cumple el mismo contrato sin red", async () => {
  const cliente = crearClienteBorradoresPresentacion({ reloj: () => new Date("2026-07-18T12:00:00Z") });
  const [opciones, lista] = await Promise.all([cliente.obtenerOpciones(), cliente.listar()]);
  assert.equal(opciones.capacidades.crear, true);
  assert.equal(lista.elementos[0].referencia_estado.referencia, "DEMO-BORRADOR-001");
  const detalle = await cliente.obtenerDetalle("DEMO-BORRADOR-001", opciones.limites);
  const recibo = await cliente.actualizar("DEMO-BORRADOR-001", {
    contenido_editable: { ...detalle.contenido_editable, titulo: "Borrador DEMO actualizado" },
  });
  assert.match(recibo.evento_outbox_ref, /^DEMO-REC-BOR-/);
  assert.equal((await cliente.obtenerDetalle()).contenido_editable.titulo, "Borrador DEMO actualizado");
  assert.doesNotMatch(fuentes["portal-borradores-demo-cliente.js"], /fetch\(|XMLHttpRequest|WebSocket/);
});

test("presentación no carga adaptadores reales ni usa cookies o storage", () => {
  const codigoDemo = Object.values(fuentes).join("\n");
  assert.doesNotMatch(codigoDemo, /localStorage|sessionStorage|document\.cookie/);
  assert.doesNotMatch(codigoDemo, /fetch\(|XMLHttpRequest|WebSocket/);
  assert.match(portal, /estado\.modoPresentacion = modoPresentacionSolicitado\(\);[\s\S]{0,120}controlador\.restaurarPreferencias\(\);[\s\S]{0,80}renderizar\(\)/);
  assert.match(portal, /if \(estado\.modoPresentacion\) \{[\s\S]+import\("\.\/portal-presentacion-adaptador\.js/);
  assert.doesNotMatch(portal, /^import .*portal-presentacion-adaptador/m);
  assert.doesNotMatch(`${portal}\n${eventos}`, /localStorage|sessionStorage|document\.cookie/);
});

test("menú, títulos y renderizadores cubren las mismas capacidades", () => {
  const vistas = [...html.matchAll(/data-vista="([a-z]+)"/g)].map((item) => item[1]);
  const bolsa = [...new Set(vistas.filter((vista) => vista !== "portal"))];
  for (const vista of bolsa) {
    assert.match(portal, new RegExp(`\\b${vista}: \\[`), `falta título para ${vista}`);
    if (vista !== "resumen" && vista !== "elaboracion") {
      assert.match(portal, new RegExp(`\\b${vista}: \\(`), `falta renderizador para ${vista}`);
    }
  }
  for (const [nombre, codigo] of Object.entries(fuentes)) {
    assert.ok(codigo.split(/\r?\n/).length - 1 < 800, `${nombre} supera 800 líneas`);
  }
  assert.ok(portal.split(/\r?\n/).length - 1 < 800, "portal.js supera 800 líneas");
});
