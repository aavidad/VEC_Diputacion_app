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
import {
  contenerTabulacionMenu,
  serializarCamposFiltro,
  serializarCamposOperacion,
} from "./portal-eventos.js";

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
  assert.match(portal, /function cerrarMenuMovil\(/);
  assert.match(portal, /crearControladorPortal\(\{[\s\S]{0,220}cerrarMenuMovil/);
  assert.match(eventos, /const \{[\s\S]{0,160}cerrarMenuMovil/);
  assert.doesNotMatch(eventos, /function cerrarMenuMovil\(\)/);
});

test("el selector de perfiles solo entra en presentación y un perfil inválido falla cerrado", () => {
  assert.match(portal, /import\("\.\.\/presentacion\/selector-perfiles\.js\?v=/u);
  assert.match(portal, /\["administrador", "tecnico", "funcionario"\]\.includes/u);
  assert.match(portal, /perfilPresentacionSolicitado\(\) === null\) return vista === "portal"/u);
  assert.match(html, /<div class="sesion-presentacion" id="sesion-visible"/u);
  assert.doesNotMatch(html, /selector-perfiles\.css/u);
});

test("Escape y el velo devuelven el foco al control que abre el menú móvil", () => {
  assert.match(portal, /function cerrarMenuMovil\(\{ restaurarFoco = false \} = \{\}\)/);
  assert.match(eventos, /velo-menu[^\n]+cerrarMenuMovil\(\{ restaurarFoco: true \}\)/);
  assert.match(eventos, /evento\.key === "Escape"[\s\S]{0,180}cerrarMenuMovil\(\{ restaurarFoco: true \}\)/);
});

function utilidades(esPresentacion) {
  return crearUtilidadesVista({
    escaparHTML: (valor) => String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;"),
    numero: (valor) => String(valor ?? 0),
    claseEstado: () => "neutro",
    encabezadoVista: (_sobrelinea, titulo, descripcion, acciones = "") => `<header><h2>${titulo}</h2><p>${descripcion}</p>${acciones}</header>`,
    esPresentacion: () => esPresentacion.valor,
    operacionPermitida: () => true,
  });
}

test("la serialización de formularios administrativos es cerrada y canónica", () => {
  assert.deepEqual(serializarCamposOperacion([
    ["plantilla", "Resolución DEMO"], ["formato", "PDF/A"], ["finalidad", "Auditoría"],
  ]), { plantilla: "Resolución DEMO", formato: "PDF/A", finalidad: "Auditoría" });
  assert.throws(() => serializarCamposOperacion([["plantilla", "a"], ["plantilla", "b"]]), /duplicados/);
  assert.throws(() => serializarCamposOperacion([["Plantilla", "a"]]), /no canónico/);
  assert.throws(() => serializarCamposOperacion([["campo_desconocido", "a"]]), /no canónico/);
  assert.throws(() => serializarCamposOperacion([["plantilla", "x".repeat(2_001)]]), /no válido/);
  assert.throws(() => serializarCamposOperacion([["plantilla", { nombre: "fichero.pdf" }]]), /no admite ficheros/);
  const nombres = ["denominacion", "categoria", "expediente", "tipo_proceso", "apertura", "cierre",
    "subsanacion_desde", "subsanacion_hasta", "version_bases", "medio_publicacion", "plantilla",
    "circuito_firma", "criterio", "motivo_tipificado", "observacion", "unidad_tiempo", "puntos_unidad",
    "fraccion_jornada", "tope_bloque", "ambito_experiencia", "redondeo", "desempate_1", "desempate_2",
    "desempate_3", "ultimo_recurso", "bolsa", "destino", "jornada", "duracion", "regla", "plazo_respuesta",
    "canales", "formato"];
  assert.throws(() => serializarCamposOperacion(nombres.map((nombre) => [nombre, "a"])), /máximo/);
  assert.deepEqual(serializarCamposFiltro("solicitudes", [["referencia", " DEMO-SOL-002 "]]),
    { referencia: "DEMO-SOL-002" });
  assert.throws(() => serializarCamposFiltro("solicitudes", [["finalidad", "Auditoría"]]), /no canónico/);
});

test("la trampa de foco móvil contiene Tab y Shift+Tab en el lateral", () => {
  const enfocados = [];
  const primero = { focus: () => enfocados.push("primero"), getAttribute: () => null, closest: () => null };
  const ultimo = { focus: () => enfocados.push("ultimo"), getAttribute: () => null, closest: () => null };
  const lateral = { querySelectorAll: () => [primero, ultimo], contains: (elemento) => [primero, ultimo].includes(elemento) };
  const haciaDelante = { key: "Tab", shiftKey: false, preventDefault() { this.cancelado = true; } };
  assert.equal(contenerTabulacionMenu(haciaDelante, lateral, ultimo), true);
  assert.equal(haciaDelante.cancelado, true);
  const haciaAtras = { key: "Tab", shiftKey: true, preventDefault() { this.cancelado = true; } };
  assert.equal(contenerTabulacionMenu(haciaAtras, lateral, primero), true);
  assert.deepEqual(enfocados, ["primero", "ultimo"]);
});

test("el controlador revalida permiso y falla cerrado sin recibo aparente", () => {
  assert.match(portal, /function operacionPermitida\(operacion\)[\s\S]{0,220}operaciones\.includes\("\*"\)/);
  assert.match(portal, /function ejecutarOperacionPresentacion\(operacion, objetivo, motivo, campos\)/);
  assert.match(portal, /adaptadorPresentacion\.ejecutar\(\{ operacion, objetivo, motivo, campos \}\)/);
  assert.match(eventos, /comando !== operacion \|\| !operacionPermitida\(operacion\)/);
  assert.match(eventos, /try \{[\s\S]{0,700}serializarCamposOperacion\(new FormData\(formulario\)\)[\s\S]{0,300}ejecutarOperacionPresentacion/);
  assert.match(eventos, /catch \{[\s\S]{0,240}Actuación no realizada[\s\S]{0,320}No se ha emitido un recibo de éxito/);
});

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
    objetivo: "DEMO-MER-001", resultado: "Aceptado", motivo: "Prueba",
    campos_aplicados: 0, efectos_reales: false,
  });
  assert.equal(uno.obtenerDatos().meritos_revision[0].estado, "Aceptado");
  assert.equal(dos.obtenerDatos().meritos_revision[0].estado, "Pendiente",
    "una nueva composición equivale a recargar y no conserva el estado anterior");
  assert.throws(() => uno.ejecutar({ operacion: "operacion-no-permitida" }), /no permitida/);
  const copia = uno.obtenerDatos();
  copia.meritos_revision[0].estado = "Manipulado";
  assert.equal(uno.obtenerDatos().meritos_revision[0].estado, "Aceptado");
});

test("cada operación distingue crear, actualizar y conjuntos; un objetivo erróneo falla cerrado", () => {
  const fecha = new Date("2026-07-18T12:00:00Z");
  const adaptador = crearAdaptadorPresentacion({ datosIniciales: obtenerDatosPresentacion(), reloj: () => fecha });
  const actividadInicial = adaptador.obtenerDatos().actividad.length;
  assert.throws(() => adaptador.ejecutar({ operacion: "aceptar-merito", objetivo: "DEMO-MER-INEXISTENTE" }), /no existe/);
  assert.equal(adaptador.obtenerDatos().actividad.length, actividadInicial, "un fallo no puede emitir actividad ni recibo");

  adaptador.ejecutar({ operacion: "crear-convocatoria", objetivo: "DEMO-BOL-NUEVA-PRUEBA" });
  assert.equal(adaptador.obtenerDatos().elaboraciones[0].id, "DEMO-BOL-NUEVA-PRUEBA");
  assert.throws(() => adaptador.ejecutar({ operacion: "crear-convocatoria", objetivo: "DEMO-BOL-NUEVA-PRUEBA" }), /ya existe/);

  adaptador.ejecutar({ operacion: "publicar-lista-provisional", objetivo: "DEMO-BOL-014" });
  assert.ok(adaptador.obtenerDatos().ranking.every((item) => item.estado === "Lista provisional"));
});

test("los formularios aplican únicamente campos enumerados y conservan cambios observables", () => {
  const adaptador = crearAdaptadorPresentacion({ datosIniciales: obtenerDatosPresentacion(), reloj: () => new Date("2026-07-18T12:00:00Z") });
  const recibo = adaptador.ejecutar({
    operacion: "guardar-bases",
    objetivo: "DEMO-BOL-014",
    campos: {
      denominacion: "Convocatoria DEMO modificada",
      expediente: "DEMO-EXP-014",
      apertura: "2026-08-01T09:00",
      cierre: "2026-08-20T23:59",
      version_bases: "v4 DEMO",
    },
  });
  assert.equal(recibo.campos_aplicados, 5);
  const convocatoria = adaptador.obtenerDatos().elaboraciones.find((item) => item.id === "DEMO-BOL-014");
  assert.equal(convocatoria.nombre, "Convocatoria DEMO modificada");
  assert.match(convocatoria.calendario, /2026-08-01T09:00/);
  assert.throws(() => adaptador.ejecutar({
    operacion: "guardar-bases", objetivo: "DEMO-BOL-014", campos: { administrador: "sí" },
  }), /campo no permitido/);

  adaptador.ejecutar({
    operacion: "generar-documento", objetivo: "DEMO-DOC-CREADO",
    campos: { plantilla: "Resolución DEMO", formato: "PDF/A", circuito: "Circuito DEMO", cotejo: "CSV" },
  });
  const documento = adaptador.obtenerDatos().documentos.find((item) => item.referencia === "DEMO-DOC-CREADO");
  assert.equal(documento.formatos, "PDF/A");
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

test("los perfiles sintéticos separan actor, vistas y operaciones por mínimo privilegio", () => {
  const administrador = obtenerDatosPresentacion("administrador");
  const tecnico = obtenerDatosPresentacion("tecnico");
  const funcionario = obtenerDatosPresentacion("funcionario");
  assert.ok(!administrador.sesion.vistas_permitidas.includes("*"));
  assert.ok(!administrador.sesion.operaciones_permitidas.includes("*"));
  assert.ok(administrador.sesion.vistas_permitidas.includes("configuracion"));
  assert.deepEqual(new Set(administrador.sesion.operaciones_permitidas), new Set(OPERACIONES_PRESENTACION));
  assert.ok(!tecnico.sesion.vistas_permitidas.includes("configuracion"));
  assert.ok(tecnico.sesion.operaciones_permitidas.includes("aceptar-merito"));
  assert.ok(!tecnico.sesion.operaciones_permitidas.includes("crear-rol"));
  assert.deepEqual(funcionario.sesion.vistas_permitidas, ["portal", "cronos", "dietas"]);
  assert.deepEqual(funcionario.sesion.operaciones_permitidas, []);
  const adaptador = crearAdaptadorPresentacion({ datosIniciales: tecnico, reloj: () => new Date("2026-07-18T12:00:00Z") });
  const recibo = adaptador.ejecutar({ operacion: "aceptar-merito", objetivo: "DEMO-MER-001" });
  assert.equal(recibo.actor, "DEMO-PERFIL-TECNICO-RRHH-01");
  assert.throws(() => adaptador.ejecutar({ operacion: "crear-rol", objetivo: "DEMO-ROL-PROHIBIDO" }), /no permite/);
  assert.throws(() => obtenerDatosPresentacion("perfil-no-admitido"), /no permitido/);
});

test("el adaptador rechaza una identidad ausente o malformada sin atribuirla al administrador", () => {
  for (const actorRef of [undefined, "actor-malformado", "DEMO-PERFIL-"]) {
    const datos = obtenerDatosPresentacion("tecnico");
    if (actorRef === undefined) delete datos.sesion.actor_ref;
    else datos.sesion.actor_ref = actorRef;
    assert.throws(
      () => crearAdaptadorPresentacion({ datosIniciales: datos }),
      /identidad del actor de presentación no válida/,
    );
  }
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

test("los filtros locales preservan valores y no se disfrazan de exportación", () => {
  const modo = { valor: true };
  const u = utilidades(modo);
  const datos = obtenerDatosPresentacion();
  const convocatorias = crearVistasConvocatorias(u);
  const baremacion = crearVistasBaremacion(u);
  const estado = {
    elaboracionSeleccionada: "DEMO-BOL-014",
    filtros: {
      convocatorias: { texto: "Operario", estado: "Publicada", unidad: "Todas" },
      solicitudes: { referencia: "DEMO-SOL-002", convocatoria: "DEMO-BOL-014", estado: "Todos" },
      meritos: { referencia: "DEMO-MER-004", tipo: "Titulación", estado: "Pendiente" },
    },
  };
  const salidas = [
    convocatorias.renderizarConvocatorias(datos, estado),
    convocatorias.renderizarSolicitudes(datos, estado),
    baremacion.renderizarMeritos(datos, estado),
  ];
  assert.match(salidas[0], /data-total-filtro="convocatorias" data-total="1"/);
  assert.match(salidas[0], /value="Operario"/);
  assert.match(salidas[0], /value="Publicada" selected/);
  assert.doesNotMatch(salidas[0], /DEMO-BOL-014<\/button>/);
  assert.match(salidas[1], /data-total-filtro="solicitudes" data-total="1"/);
  assert.match(salidas[1], /DEMO-SOL-002/);
  assert.doesNotMatch(salidas[1], /<strong>DEMO-SOL-001<\/strong>/);
  assert.match(salidas[2], /data-total-filtro="meritos" data-total="1"/);
  assert.match(salidas[2], /DEMO-MER-004/);
  assert.doesNotMatch(salidas[2], /<strong>DEMO-MER-001<\/strong>/);
  for (const salida of salidas) {
    const formulario = salida.match(/<form class="barra-filtros"[\s\S]*?<\/form>/)?.[0] || "";
    assert.match(formulario, /data-filtro=/);
    assert.match(formulario, /<button type="submit"/);
    assert.doesNotMatch(formulario, /data-operacion|exportar-informe|DEMO-FILTRO/);
  }
});

test("la vista de convocatorias distingue la fuente pública del expediente sintético", () => {
  const modo = { valor: true };
  const salida = crearVistasConvocatorias(utilidades(modo)).renderizarConvocatorias(
    obtenerDatosPresentacion(),
    { elaboracionSeleccionada: "DEMO-BOL-009", filtros: {} },
  );
  assert.match(salida, /BOP-GRA-2026-043004/);
  assert.match(salida, /Publicado 05\/03\/2026/);
  assert.match(salida, /bases-operario-demo\.pdf/);
  assert.match(salida, /bases-operario-demo\.html/);
  assert.match(salida, /Fuente pública real/);
  assert.match(salida, /expediente y su tramitación son sintéticos/);
  assert.match(salida, /target="_blank" rel="noopener noreferrer"/);
});

test("los formularios gobernados envían nombres estables y comandos explícitos", () => {
  const modo = { valor: true };
  const u = utilidades(modo);
  const datos = obtenerDatosPresentacion();
  const estado = { elaboracionSeleccionada: "DEMO-BOL-014", filtros: {} };
  const salidas = [
    crearVistasConvocatorias(u).renderizarConvocatorias(datos, estado),
    crearVistasBaremacion(u).renderizarMeritos(datos, estado),
    crearVistasBaremacion(u).renderizarBaremacion(datos),
    crearVistasOperaciones(u).renderizarLlamamientos(datos),
    crearVistasOperaciones(u).renderizarDocumentos(datos),
    crearVistasOperaciones(u).renderizarComunicaciones(datos),
    crearVistasGobierno(u).renderizarEstadisticas(datos),
    crearVistasGobierno(u).renderizarConfiguracion(datos),
  ];
  const formularios = salidas.flatMap((salida) => [...salida.matchAll(/<form class="[^"]*formulario-gobernado[^"]*"[\s\S]*?<\/form>/g)]
    .map((coincidencia) => coincidencia[0]));
  assert.equal(formularios.length, 8, "ningún formulario operativo debe quedar como cascarón visual");
  for (const formulario of formularios) {
    for (const control of formulario.matchAll(/<(?:input|select|textarea)\b([^>]*)>/g)) {
      assert.match(control[1], /\bname="[a-z][a-z0-9_]*"/, `control sin nombre canónico: ${control[0]}`);
    }
    assert.match(formulario, /data-comando=/, "cada formulario debe exponer al menos un comando explícito");
  }
  const meritos = salidas[1];
  assert.doesNotMatch(meritos, />Resultado<\/span>/);
  for (const operacion of ["aceptar-merito", "rechazar-merito", "revocar-merito", "rehabilitar-merito"]) {
    assert.match(meritos, new RegExp(`data-comando="${operacion}"`));
  }
  assert.match(salidas[2], /data-operacion="guardar-reglas-baremo" data-objetivo="DEMO-CRI-001"/);
  assert.doesNotMatch(salidas[1], /REV-DEMO-MERITOS/);
  assert.match(salidas[1], /DEMO-REV-MERITOS/);
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
