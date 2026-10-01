import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import {
  crearClienteHTTPCircuitoFirma, crearGestorCircuitoFirma, renderizarCircuitoFirma,
  RUTA_CIRCUITO_FIRMA, validarCircuitoFirma,

} from "./circuito-firma.js?v=20261001-ct-firma-verificador-v2";
import { crearAccionesFirma, fusionarEstadoFirmas, renderizarAccionesPaso } from "./circuito-firma-acciones.js?v=20261001-ct-firma-verificador-v2";
import { crearTraductorCircuitoFirma, MENSAJES_CIRCUITO_FIRMA_ES, MENSAJES_CIRCUITO_FIRMA_EN } from "./i18n-circuito-firma.js?v=20261001-ct-firma-verificador-v2";
import { cargarTextos } from "../../../comun/textos.js";
import { IDIOMA_POR_DEFECTO } from "../../../comun/idioma.js";

// Los textos de la fase de firma se leen una vez; el gestor los recibe ya
// cargados para que el montaje no dependa de la lectura del fichero.
const textosFasePrueba = await cargarTextos("contratacion-temporal-firma", { idioma: IDIOMA_POR_DEFECTO });
const cargarTextosPrueba = async () => textosFasePrueba;

function paso(orden, total, extra = {}) {
  return {
    orden, cargo: `Cargo <${orden}>`, perfil_ref: "perfil:ct:jefatura_servicio_rrhh",
    accion: "firma", condicion: orden === 1 ? "borrador_generado" : "firma_paso_anterior",
    habilita: orden === total ? "remision_intervencion" : "siguiente_paso",
    devolucion: "vuelve_a_redaccion", sustitucion: "suplente_designado",
    estado: orden === 1 ? "pendiente_firma" : "en_espera",
    referencia: `vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p${orden}`, ...extra,
  };
}

function circuito() {
  return {
    esquema: "vec.contratacion_temporal.circuito_firma.v1",
    catalogo_ref: "vec.contratacion_temporal.circuito_firma:1",
    huella_sha256: "a".repeat(64), ejemplo: true, firma_eficaz: false,
    portafirmas: { conectado: false, motivo: "conexion_pendiente" },
    documentos: [{ documento: "informe_definitivo", etiqueta: "Informe definitivo", pasos: [paso(1, 2), paso(2, 2)] }],
  };
}

function respuestaJSON(cuerpo, estado = 200) {
  return new Response(JSON.stringify(cuerpo), { status: estado, headers: { "Content-Type": "application/json; charset=utf-8" } });
}

test("valida el contrato exacto y rechaza desviaciones", () => {
  assert.ok(validarCircuitoFirma(circuito()));
  const casos = [
    (c) => { c.firma_eficaz = true; },
    (c) => { c.extra = 1; },
    (c) => { c.documentos[0].pasos[1].orden = 3; },
    (c) => { c.documentos[0].pasos[0].estado = "aprobado"; },
    (c) => { c.documentos[0].pasos[1].habilita = "siguiente_paso"; },
    (c) => { c.documentos[0].pasos[0].habilita = "cierre_circuito"; },
    (c) => { c.documentos = []; },
    (c) => { c.huella_sha256 = "x"; },
    // Firmadoc: el servidor solo dice si está conectado; nada más se admite.
    (c) => { c.portafirmas = { conectado: false }; },
    (c) => { c.portafirmas = { conectado: false, motivo: "Conexión" }; },
    (c) => { c.portafirmas = { conectado: true, motivo: "conexion_pendiente" }; },
    (c) => { c.portafirmas = { conectado: false, motivo: "conexion_pendiente", estado: "firmado" }; },
    (c) => { c.portafirmas = { conectado: "no", motivo: "conexion_pendiente" }; },
  ];
  for (const alterar of casos) {
    const copia = circuito();
    alterar(copia);
    assert.equal(validarCircuitoFirma(copia), null);
  }
});

test("el cliente pide la ruta de solo lectura y falla cerrado", async () => {
  let peticion;
  const cliente = crearClienteHTTPCircuitoFirma({
    fetchImpl: async (ruta, opciones) => { peticion = { ruta, opciones }; return respuestaJSON({ data: circuito() }); },
  });
  const resultado = await cliente.obtenerCircuito();
  assert.equal(peticion.ruta, RUTA_CIRCUITO_FIRMA);
  assert.equal(peticion.opciones.method, "GET");
  assert.equal(peticion.opciones.redirect, "error");
  assert.equal(resultado.documentos[0].pasos.length, 2);
  for (const fetchImpl of [
    async () => respuestaJSON({ error: { codigo: "servicio_no_disponible" } }, 503),
    async () => respuestaJSON({ data: { ...circuito(), firma_eficaz: true } }),
    async () => new Response("x".repeat(70 * 1024), { headers: { "Content-Type": "application/json" } }),
    async () => { throw new TypeError("red"); },
  ]) {
    assert.equal(await crearClienteHTTPCircuitoFirma({ fetchImpl }).obtenerCircuito(), null);
  }
  for (const [codigo, esperado] of [[403, "denegado"], [503, "no_disponible"]]) {
    const conEstado = crearClienteHTTPCircuitoFirma({ fetchImpl: async () => respuestaJSON({ error: { codigo: "acceso_denegado" } }, codigo) });
    assert.deepEqual(await conEstado.obtenerCircuitoConEstado(), { estado: esperado });
    assert.equal(await conEstado.obtenerCircuito(), null);
  }
});

test("el bloque muestra cada paso con su estado, escapa el catálogo y marca el ejemplo", () => {
  const t = crearTraductorCircuitoFirma();
  const html = renderizarCircuitoFirma(validarCircuitoFirma(circuito()), t);
  assert.match(html, /aria-labelledby="ct-circuito-firma-titulo"/u);
  assert.match(html, /Firma oficial en Firmadoc/u);
  assert.match(html, /No conectado/u);
  assert.match(html, /<button[^>]*disabled[^>]*aria-describedby="ct-circuito-envio-motivo"[^>]*>Enviar a Firmadoc<\/button>/u);
  assert.match(html, /No disponible hasta que Informática conecte VEC con Firmadoc/u);
  assert.match(html, /Sin constancia de envío ni firma oficial en VEC/u);
  assert.match(html, /AutoFirma · PRUEBA sin eficacia administrativa/u);
  assert.match(html, /<details class="ct-circuito-limite">/u);
  assert.match(html, /<details class="ct-circuito-prueba" data-ct-firma-detalles>/u);
  assert.doesNotMatch(html, /<details[^>]*\sopen/u);
  assert.doesNotMatch(html, /Enviad[ao] a Firmadoc|Firma oficial completada/u);
  assert.doesNotMatch(html, /Circuito de ejemplo/u);
  assert.match(html, /Pendiente de firma de prueba por Cargo &lt;1&gt;/u);
  assert.match(html, /Prueba en espera del paso anterior/u);
  assert.match(html, /Permite remitir a Intervención/u);
  assert.equal((html.match(/aria-current="step"/gu) ?? []).length, 1);
  assert.doesNotMatch(html, /<1>/u);
});

test("si falla el catálogo, conserva visible la fase oficial sin afirmar estado de firma", () => {
  const t = crearTraductorCircuitoFirma();
  const html = renderizarCircuitoFirma(null, t, "no_disponible");
  assert.match(html, /Firma oficial en Firmadoc/u);
  assert.match(html, /No conectado/u);
  assert.match(html, /El estado de las firmas no está disponible/u);
  assert.doesNotMatch(html, /data-ct-firma-accion|Firma de prueba registrada por/u);
  const denegado = renderizarCircuitoFirma(null, t, "denegado");
  assert.match(denegado, /No dispone de permiso para consultar/u);
  assert.match(denegado, /class="ct-circuito-indisponible" role="alert"/u);
  assert.doesNotMatch(denegado, /El estado de las firmas no está disponible/u);
});

test("dos pasos CT118 firmados no convierten Firmadoc en envío o firma oficial", () => {
  const catalogo = validarCircuitoFirma(circuito());
  const estado = {
    huella_sha256: catalogo.huella_sha256, verificacion_disponible: true,
    documentos: [{ documento: "informe_definitivo", paso_pendiente: 0,
      pasos: [{ estado: "firmado" }, { estado: "firmado" }] }],
  };
  const unido = fusionarEstadoFirmas(catalogo, estado);
  const html = renderizarCircuitoFirma(unido, crearTraductorCircuitoFirma());
  assert.equal((html.match(/Firma de prueba registrada por/gu) ?? []).length, 2);
  assert.match(html, /No conectado/u);
  assert.match(html, /Sin constancia de envío ni firma oficial en VEC/u);
  assert.equal(fusionarEstadoFirmas(catalogo, { ...estado, huella_sha256: "b".repeat(64) }), null);
});

test("las acciones activas de AutoFirma indican que son PRUEBA", () => {
  const catalogo = validarCircuitoFirma(circuito());
  const real = fusionarEstadoFirmas(catalogo, {
    huella_sha256: catalogo.huella_sha256, verificacion_disponible: true,
    documentos: [{ documento: "informe_definitivo", paso_pendiente: 1,
      pasos: [{ estado: "pendiente_firma" }, { estado: "en_espera" }] }],
  });
  const html = renderizarCircuitoFirma(real, crearTraductorCircuitoFirma());
  assert.match(html, /<details class="ct-circuito-prueba" data-ct-firma-detalles>/u);
  assert.match(html, />Firmar en PRUEBA<\/button>/u);
  assert.match(html, />Devolver en PRUEBA<\/button>/u);
  assert.match(html, />Registrar devolución de PRUEBA<\/button>/u);
  assert.match(html, /Pendiente de firma de prueba por/u);
});

test("verificación apagada mantiene Firmar enfocable y explica por qué no puede usarse", () => {
  const catalogo = validarCircuitoFirma(circuito());
  const estado = (disponible) => fusionarEstadoFirmas(catalogo, {
    huella_sha256: catalogo.huella_sha256, verificacion_disponible: disponible,
    documentos: [{ documento: "informe_definitivo", paso_pendiente: 1,
      pasos: [{ estado: "pendiente_firma" }, { estado: "en_espera" }] }],
  });
  const t = crearTraductorCircuitoFirma();
  const apagado = estado(false);
  const html = renderizarAccionesPaso(apagado, apagado.documentos[0], apagado.documentos[0].pasos[0], t);
  assert.match(html, /data-ct-firma-accion="firmar"[^>]*aria-disabled="true" aria-describedby="ct-firma-resultado-informe_definitivo-1"/u);
  assert.match(html, /id="ct-firma-resultado-informe_definitivo-1"[^>]*>La verificación de firmas no está activada/u);
  assert.match(html, /data-ct-firma-accion="devolver"/u);
  assert.doesNotMatch(html, /data-ct-firma-accion="firmar"[^>]*\sdisabled(?:\s|>)/u);
  const activo = estado(true);
  const disponible = renderizarAccionesPaso(activo, activo.documentos[0], activo.documentos[0].pasos[0], t);
  assert.doesNotMatch(disponible, /aria-disabled="true"|La verificación de firmas no está activada/u);
  assert.match(disponible, /data-ct-firma-accion="firmar"/u);
});

test("al activar firma conocida imposible no descarga ni lanza AutoFirma; devolución sigue disponible", async () => {
  const ref = "expediente:ct:001";
  const estado = { vista: "expediente", carga: "listo", expediente_ref: ref,
    expediente: { expediente_ref: ref, version: 7, demostracion: false },
    cuadro: { demostracion: false, expedientes: [{ expediente_ref: ref, version: 7,
      fase_clave: "nombramiento", estado_clave: "en_curso" }] } };
  const salida = { textContent: "" };
  const contenedor = { querySelector: (selector) => selector === "[data-ct-firma-resultado]" ? salida
    : selector === "[data-ct-firma-motivo]" ? { value: "Falta la fecha de efectos" } : null,
  querySelectorAll: () => [] };
  const boton = { dataset: { ctFirmaAccion: "firmar", ctFirmaDocumento: "informe_definitivo", ctFirmaOrden: "1" },
    getAttribute: () => "true", disabled: false,
    closest: (selector) => selector === "[data-ct-firma-accion]" ? boton : contenedor };
  let descargas = 0; let lanzamientos = 0; const registros = [];
  const acciones = crearAccionesFirma({ obtenerEstado: () => estado, t: crearTraductorCircuitoFirma(),
    clienteBorrador: { descargarBorrador: async () => { descargas += 1; return new Blob(["%PDF-1.7"]); } },
    autofirma: { firmarPDF: async () => { lanzamientos += 1; return new Uint8Array([1]); } },
    clienteFirma: { registrar: async (orden) => { registros.push(orden); return { recibo_ref: "recibo:firma:1" }; } },
    aleatorio: (n) => new Uint8Array(n) });
  await acciones.manejarClic({ target: { closest: () => boton } });
  assert.match(salida.textContent, /verificación de firmas no está activada/u);
  assert.equal(descargas, 0); assert.equal(lanzamientos, 0); assert.equal(registros.length, 0);
  assert.equal(boton.disabled, false, "el botón conserva el foco posible");

  boton.getAttribute = () => null;
  await acciones.manejarClic({ target: { closest: () => boton } });
  assert.equal(descargas, 1); assert.equal(lanzamientos, 1); assert.equal(registros[0].resultado, "firmado");

  boton.getAttribute = () => "true";
  boton.dataset.ctFirmaAccion = "confirmar-devolucion";
  await acciones.manejarClic({ target: { closest: () => boton } });
  assert.equal(descargas, 1); assert.equal(lanzamientos, 1);
  assert.equal(registros[1].resultado, "devuelto");
});

test("todas las claves de vocabulario tienen traducción", () => {
  const claves = Object.keys(MENSAJES_CIRCUITO_FIRMA_ES);
  for (const prefijo of ["accion_firma", "accion_visto_bueno", "habilita_siguiente_paso", "habilita_remision_intervencion",
    "habilita_envio_notificacion", "habilita_envio_comunicacion", "habilita_cierre_circuito",
    "devolucion_vuelve_a_redaccion", "devolucion_vuelve_paso_anterior", "sustitucion_suplente_designado",
    "sustitucion_no_admitida", "estado_pendiente_firma", "estado_en_espera", "estado_firmado", "estado_devuelto"]) {
    assert.ok(claves.includes(`circuito_firma_${prefijo}`), prefijo);
  }
});

test("la fase Firmadoc usa el idioma del portal", () => {
  assert.deepEqual(Object.keys(MENSAJES_CIRCUITO_FIRMA_EN).sort(), Object.keys(MENSAJES_CIRCUITO_FIRMA_ES).sort());
  for (const [clave, valor] of Object.entries(MENSAJES_CIRCUITO_FIRMA_EN)) {
    assert.ok(valor.trim(), clave);
    const variables = (texto) => [...texto.matchAll(/\{([a-z_]+)\}/gu)].map((m) => m[1]).sort();
    assert.deepEqual(variables(valor), variables(MENSAJES_CIRCUITO_FIRMA_ES[clave]), clave);
  }
  const traductor = crearTraductorCircuitoFirma({}, "en-GB");
  const html = renderizarCircuitoFirma(validarCircuitoFirma(circuito()), traductor);
  assert.match(html, /Official signing in Firmadoc/u);
  assert.match(html, /Not connected/u);
  assert.match(html, /Send to Firmadoc<\/button>/u);
  assert.match(html, /Not available until IT connects VEC to Firmadoc/u);
  assert.match(html, /No recorded submission or official signature in VEC/u);
  assert.match(html, /Awaiting test signature by/u);
  assert.match(html, /Allows referral to Financial Control/u);
  assert.match(html, /If returned, goes back to drafting/u);
  assert.doesNotMatch(html, /Pendiente de firma|Permite remitir|Si se devuelve/u);
  assert.match(html, /AutoFirma · TEST with no administrative effect/u);
  assert.match(MENSAJES_CIRCUITO_FIRMA_EN.circuito_firma_firmar, /TEST/u);
  assert.match(MENSAJES_CIRCUITO_FIRMA_EN.circuito_firma_devolver, /TEST/u);
});

test("el catálogo de prueba traduce documentos y cargos conocidos sin alterar valores ajenos", () => {
  const dato = circuito();
  dato.documentos[0].pasos[0].cargo = "Técnico/a de RRHH responsable del expediente";
  const html = renderizarCircuitoFirma(validarCircuitoFirma(dato), crearTraductorCircuitoFirma({}, "en-GB"));
  assert.match(html, /Final report/u);
  assert.match(html, /HR officer responsible for the case/u);
  assert.doesNotMatch(html, /Informe definitivo|Técnico\/a de RRHH/u);
  dato.documentos[0].etiqueta = "Nombre ajeno <x>";
  const otro = renderizarCircuitoFirma(validarCircuitoFirma(dato), crearTraductorCircuitoFirma({}, "en-GB"));
  assert.match(otro, /Nombre ajeno &lt;x&gt;/u);
});

test("el gestor inserta el bloque después de siguiente paso, con fallback tras las fases", async () => {
  const insertados = [];
  const siguiente = { insertAdjacentHTML: (posicion, html) => insertados.push({ ancla: "siguiente", posicion, html }) };
  const fases = { insertAdjacentHTML: (posicion, html) => insertados.push({ ancla: "fases", posicion, html }) };
  let estado = { vista: "expediente", expediente: { expediente_ref: "exp:1" } };
  const raiz = { querySelector: (selector) => selector === ".ct-exp-siguiente-paso" ? siguiente : selector === ".ct-exp-progreso" ? fases : null };
  let consultas = 0;
  const cliente = { obtenerCircuito: async () => { consultas += 1; return validarCircuitoFirma(circuito()); } };
  const gestor = crearGestorCircuitoFirma({ cargarTextos: cargarTextosPrueba, raiz, obtenerEstado: () => estado, cliente });
  gestor.montarSiProcede(estado);
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.equal(insertados.length, 1);
  assert.equal(insertados[0].ancla, "siguiente");
  assert.equal(insertados[0].posicion, "afterend");
  gestor.montarSiProcede({ vista: "cuadro" });
  const anterior = estado;
  gestor.montarSiProcede(anterior);
  estado = { vista: "expediente", expediente: { expediente_ref: "exp:2" } };
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.equal(insertados.length, 1, "un expediente ya sustituido no recibe el bloque");
  assert.equal(consultas, 1, "el catálogo se consulta una vez por montaje");
  const sinSiguiente = { querySelector: (selector) => selector === ".ct-exp-progreso" ? fases : null };
  const segundo = crearGestorCircuitoFirma({ cargarTextos: cargarTextosPrueba, raiz: sinSiguiente, obtenerEstado: () => estado, cliente });
  segundo.montarSiProcede(estado);
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.equal(insertados.at(-1).ancla, "fases");
  segundo.retirar();
});

test("un fallo de consulta deja el estado pendiente visible en el expediente actual", async () => {
  const insertados = [];
  const fases = { insertAdjacentHTML: (_, html) => insertados.push(html) };
  const estado = { vista: "expediente", expediente: { expediente_ref: "exp:1" } };
  const raiz = { querySelector: (selector) => (selector === ".ct-exp-progreso" ? fases : null) };
  const gestor = crearGestorCircuitoFirma({ cargarTextos: cargarTextosPrueba, raiz, obtenerEstado: () => estado,
    cliente: { obtenerCircuito: async () => null }, clienteFirma: { consultar: () => { throw new Error("no debe consultarse"); } } });
  gestor.montarSiProcede(estado);
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.equal(insertados.length, 1);
  assert.match(insertados[0], /No conectado/u);
  assert.match(insertados[0], /El estado de las firmas no está disponible/u);
  gestor.retirar();
});

test("el gestor distingue denegación 403 de indisponibilidad 503 de CT118", async () => {
  const ref = "expediente:ct:001";
  const estado = { vista: "expediente", carga: "listo", expediente_ref: ref,
    expediente: { expediente_ref: ref, version: 7, demostracion: false },
    cuadro: { demostracion: false, expedientes: [{ expediente_ref: ref, version: 7,
      fase_clave: "nombramiento", estado_clave: "en_curso" }] } };
  for (const [resultado, texto] of [["denegado", "No dispone de permiso"], ["no_disponible", "Ahora no se puede saber en qué estado"]]) {
    const insertados = [];
    const fases = { insertAdjacentHTML: (_, html) => insertados.push(html) };
    const raiz = { querySelector: (selector) => (selector === ".ct-exp-progreso" ? fases : null) };
    const gestor = crearGestorCircuitoFirma({ cargarTextos: cargarTextosPrueba, raiz, obtenerEstado: () => estado,
      cliente: { obtenerCircuitoConEstado: async () => ({ estado: "disponible", circuito: validarCircuitoFirma(circuito()) }) },
      clienteFirma: { consultarConEstado: async (expedienteRef) => {
        assert.equal(expedienteRef, ref);
        return { estado: resultado };
      } },
    });
    gestor.montarSiProcede(estado);
    await new Promise((resolver) => setTimeout(resolver, 0));
    assert.equal(insertados.length, 1);
    assert.match(insertados[0], new RegExp(texto, "u"));
    assert.doesNotMatch(insertados[0], /data-ct-firma-accion|ct-circuito-paso/u);
    // La fase de firma sigue visible con el primer firmante y sin estado.
    assert.match(insertados[0], /data-ct-fase-firma-estado="no_disponible"/u);
    // Un solo aviso: o el de permiso (sin el de la fase) o el de la fase.
    assert.equal((insertados[0].match(/ct-fase-firma-aviso|ct-circuito-indisponible/gu) ?? []).length, 1);
    gestor.retirar();
  }
});

test("tras registrar una firma, el repintado conserva el detalle abierto y devuelve el foco al aviso", async () => {
  const ref = "expediente:ct:001";
  const estado = { vista: "expediente", carga: "listo", expediente_ref: ref,
    expediente: { expediente_ref: ref, version: 7, demostracion: false },
    cuadro: { demostracion: false, expedientes: [{ expediente_ref: ref, version: 7,
      fase_clave: "nombramiento", estado_clave: "en_curso" }] } };
  let actual = null;
  let focoRestaurado = false;
  function seccion() {
    const detalles = { open: true };
    const limite = { open: false };
    const aviso = { textContent: "", focus: () => { focoRestaurado = true; } };
    return {
      detalles, aviso,
      set outerHTML(html) { assert.match(html, /AutoFirma · PRUEBA/u); actual = seccion(); actual.detalles.open = false; },
      querySelector: (selector) => selector === "[data-ct-firma-detalles]" ? detalles
        : selector === ".ct-circuito-limite" ? limite
          : selector === "[data-ct-firma-aviso]" ? aviso : null,
      addEventListener: (tipo, callback) => { if (tipo === "click") actual.manejar = callback; },
    };
  }
  const fases = { insertAdjacentHTML: () => { actual = seccion(); } };
  const raiz = { querySelector: (selector) => selector === ".ct-exp-progreso" ? fases
    : selector === "[data-ct-circuito-firma]" ? actual : null };
  const catalogo = validarCircuitoFirma(circuito());
  const registro = { huella_sha256: catalogo.huella_sha256, verificacion_disponible: true,
    documentos: [{ documento: "informe_definitivo", paso_pendiente: 1,
      pasos: [{ estado: "pendiente_firma" }, { estado: "en_espera" }] }] };
  const gestor = crearGestorCircuitoFirma({ cargarTextos: cargarTextosPrueba, raiz, obtenerEstado: () => estado,
    cliente: { obtenerCircuitoConEstado: async () => ({ estado: "disponible", circuito: catalogo }) },
    clienteFirma: { consultarConEstado: async () => ({ estado: "disponible", datos: registro }),
      registrar: async () => ({ recibo_ref: "recibo:firma:1" }) },
    dependenciasAcciones: { clienteBorrador: { descargarBorrador: async () => new Blob(["%PDF-1.7"]) },
      autofirma: { firmarPDF: async () => new Uint8Array([1, 2, 3]) },
      aleatorio: (n) => new Uint8Array(n).fill(1) },
  });
  gestor.montarSiProcede(estado);
  await new Promise((resolver) => setTimeout(resolver, 0));
  const salida = { textContent: "" };
  const contenedor = { querySelector: (selector) => selector === "[data-ct-firma-resultado]" ? salida : null,
    querySelectorAll: () => [] };
  const boton = { dataset: { ctFirmaAccion: "firmar", ctFirmaDocumento: "informe_definitivo", ctFirmaOrden: "1" },
    closest: (selector) => selector === "[data-ct-firma-accion]" ? boton : contenedor };
  actual.manejar({ target: { closest: () => boton } });
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.equal(actual.detalles.open, true);
  assert.equal(focoRestaurado, true);
  assert.match(actual.aviso.textContent, /recibo:firma:1/u);
  gestor.retirar();
});

test("un rechazo del verificador no muestra sus códigos internos", async () => {
  const ref = "expediente:ct:001";
  const estado = { vista: "expediente", carga: "listo", expediente_ref: ref,
    expediente: { expediente_ref: ref, version: 7, demostracion: false },
    cuadro: { demostracion: false, expedientes: [{ expediente_ref: ref, version: 7,
      fase_clave: "nombramiento", estado_clave: "en_curso" }] } };
  const salida = { textContent: "" };
  const contenedor = { querySelector: (selector) => selector === "[data-ct-firma-resultado]" ? salida : null,
    querySelectorAll: () => [] };
  const boton = { dataset: { ctFirmaAccion: "firmar", ctFirmaDocumento: "informe_definitivo", ctFirmaOrden: "1" },
    closest: (selector) => selector === "[data-ct-firma-accion]" ? boton : contenedor };
  const acciones = crearAccionesFirma({ obtenerEstado: () => estado, t: crearTraductorCircuitoFirma({}, "en-GB"),
    clienteBorrador: { descargarBorrador: async () => new Blob(["%PDF-1.7"]) },
    autofirma: { firmarPDF: async () => new Uint8Array([1]) },
    clienteFirma: { registrar: async () => { throw { codigo: "firma_no_verificada", motivo: "revocacion_no_acreditada" }; } },
    aleatorio: (n) => new Uint8Array(n) });
  await acciones.manejarClic({ target: { closest: () => boton } });
  assert.match(salida.textContent, /could not be verified/u);
  assert.doesNotMatch(salida.textContent, /firma_no_verificada|revocacion_no_acreditada/u);

  for (const motivo of ["validador_no_disponible", "credencial_rechazada"]) {
    const servicioCaido = crearAccionesFirma({ obtenerEstado: () => estado, t: (clave) => clave,
      clienteBorrador: { descargarBorrador: async () => new Blob(["%PDF-1.7"]) },
      autofirma: { firmarPDF: async () => new Uint8Array([1]) },
      clienteFirma: { registrar: async () => { throw { codigo: "servicio_no_disponible", motivo }; } },
      aleatorio: (n) => new Uint8Array(n) });
    await servicioCaido.manejarClic({ target: { closest: () => boton } });
    assert.equal(salida.textContent, "circuito_firma_error_validador_no_disponible");
  }
});

test("la ayuda explica el circuito de ejemplo y la falta de eficacia sin portafirmas", async () => {
  const ayuda = await readFile(new URL("../../../textos/es/portal-ayuda.json", import.meta.url), "utf8");
  assert.match(ayuda, /"ayuda_contenido_421": "El «Circuito de firma»/u);
  assert.match(ayuda, /"ayuda_contenido_422": ".*no tiene eficacia administrativa.*portafirmas corporativo/u);
});

test("la fase de firma se ve en cualquier fase del expediente real, sin botones fuera de nombramiento", async () => {
  const ref = "expediente:ct:002";
  const estado = { vista: "expediente", carga: "listo", expediente_ref: ref,
    expediente: { expediente_ref: ref, version: 9, demostracion: false },
    cuadro: { demostracion: false, expedientes: [{ expediente_ref: ref, version: 9,
      fase_clave: "formalizacion", estado_clave: "en_curso" }] } };
  const catalogo = validarCircuitoFirma(circuito());
  const insertados = [];
  const fases = { insertAdjacentHTML: (_, html) => insertados.push(html) };
  const raiz = { querySelector: (selector) => (selector === ".ct-exp-progreso" ? fases : null) };
  let consultado = "";
  const gestor = crearGestorCircuitoFirma({ cargarTextos: cargarTextosPrueba, raiz, obtenerEstado: () => estado,
    cliente: { obtenerCircuitoConEstado: async () => ({ estado: "disponible", circuito: catalogo }) },
    clienteFirma: { consultarConEstado: async (expedienteRef) => {
      consultado = expedienteRef;
      return { estado: "disponible", datos: { huella_sha256: catalogo.huella_sha256, verificacion_disponible: true,
        documentos: [{ documento: "informe_definitivo", paso_pendiente: 2,
          pasos: [{ estado: "firmado", registrada_en: "2026-09-28T08:30:00Z" }, { estado: "pendiente_firma" }] }] } };
    } },
  });
  gestor.montarSiProcede(estado);
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.equal(consultado, ref);
  assert.equal(insertados.length, 1);
  assert.match(insertados[0], /data-ct-fase-firma-estado="pendiente_firma"/u);
  assert.match(insertados[0], /Cargo &lt;2&gt; \(paso 2 de 2\)/u);
  assert.doesNotMatch(insertados[0], /data-ct-firma-accion/u, "firmar solo cuando se pueden descargar los borradores");
  gestor.retirar();
});

test("sin el campo portafirmas el circuito vale y Firmadoc cuenta como no conectado", () => {
  const datos = circuito();
  delete datos.portafirmas;
  const valido = validarCircuitoFirma(datos);
  assert.ok(valido);
  assert.deepEqual({ ...valido.portafirmas }, { conectado: false, motivo: "conexion_pendiente" });
  assert.equal(validarCircuitoFirma(circuito()).portafirmas.conectado, false);
});

test("descargar el PDF firmado pide a Documentos la terna exacta y avisa en lenguaje llano", async () => {
  const ref = "expediente:ct:001";
  const estado = { vista: "expediente", carga: "listo", expediente_ref: ref,
    expediente: { expediente_ref: ref, version: 7, demostracion: false }, cuadro: { demostracion: false, expedientes: [] } };
  const aviso = { textContent: "" };
  let bloque = null;
  let manejar = null;
  const fases = { insertAdjacentHTML: () => {
    bloque = { querySelector: (s) => s === "[data-ct-firma-aviso]" ? aviso : null,
      addEventListener: (tipo, f) => { if (tipo === "click") manejar = f; } };
  } };
  const raiz = { querySelector: (s) => s === ".ct-exp-progreso" ? fases : s === "[data-ct-circuito-firma]" ? bloque : null };
  const pedidas = [];
  let fallo = null;
  let esperarDescarga = null;
  const crearDocumentos = ({ expedienteRef }) => ({
    async descargar(documento, { version, mime, huella }) {
      pedidas.push({ expedienteRef, documento, version, mime, huella });
      if (esperarDescarga) await esperarDescarga;
      if (fallo) throw fallo;
      return { contenido: new Uint8Array([37, 80, 68, 70]), nombre: "documento-ref.pdf", tipo: "application/pdf" };
    },
  });
  const enlaces = [];
  const entornoDescarga = {
    Blob, URL: { createObjectURL: () => "blob:x", revokeObjectURL() {} },
    document: { body: { append(e) { enlaces.push(e); } }, createElement: () => ({ click() { this.pulsado = true; }, remove() {} }) },
  };
  const gestor = crearGestorCircuitoFirma({ cargarTextos: cargarTextosPrueba, raiz, obtenerEstado: () => estado,
    cliente: { obtenerCircuitoConEstado: async () => ({ estado: "no_disponible" }) }, crearDocumentos, entornoDescarga });
  gestor.montarSiProcede(estado);
  await new Promise((r) => setTimeout(r, 0));
  assert.equal(typeof manejar, "function");
  const atributos = new Map();
  const fila = { querySelector: (s) => s === ".ct-fase-firma-documento strong" ? { textContent: "Resolución de nombramiento" } : null };
  const boton = {
    disabled: false,
    setAttribute: (nombre, valor) => atributos.set(nombre, valor),
    removeAttribute: (nombre) => atributos.delete(nombre),
    dataset: { ctDescargarFirmado: "", ctFirmadoExpediente: `ref:${"e".repeat(64)}`, ctFirmadoDocumento: `ref:${"d".repeat(64)}`,
      ctFirmadoVersion: "1", ctFirmadoHuella: "1".repeat(64) },
    closest: (s) => s === "[data-ct-descargar-firmado]" ? boton : s === "[data-ct-circuito-firma]" ? bloque
      : s === ".ct-fase-firma-fila" ? fila : null,
  };
  const pulsar = async () => { manejar({ target: boton }); for (let i = 0; i < 3; i += 1) await new Promise((r) => setTimeout(r, 0)); };
  let terminarDescarga;
  esperarDescarga = new Promise((resolver) => { terminarDescarga = resolver; });
  await pulsar();
  assert.equal(atributos.get("aria-disabled"), "true", "la descarga en curso conserva el botón en el orden de foco");
  assert.equal(atributos.get("aria-describedby"), "ct-firma-aviso");
  assert.equal(boton.disabled, false);
  await pulsar();
  assert.equal(pedidas.length, 1, "un segundo clic durante la descarga no inicia otra petición");
  terminarDescarga();
  esperarDescarga = null;
  await new Promise((r) => setTimeout(r, 0));
  assert.deepEqual(pedidas[0], { expedienteRef: `ref:${"e".repeat(64)}`, documento: `ref:${"d".repeat(64)}`, version: 1,
    mime: "application/pdf", huella: "1".repeat(64) });
  assert.equal(enlaces[0]?.download, "documento-ref.pdf");
  assert.equal(enlaces[0]?.pulsado, true);
  assert.match(aviso.textContent, /PDF firmado descargado: Resolución de nombramiento/u);
  assert.doesNotMatch(aviso.textContent, /documento-ref\.pdf/u);
  assert.equal(boton.disabled, false);
  assert.equal(atributos.has("aria-disabled"), false);
  fallo = Object.assign(new Error("denegado"), { codigo: "denegado", estado: 403 });
  await pulsar();
  assert.match(aviso.textContent, /No tiene permiso/u);
  fallo = Object.assign(new Error("consulta_fallida"), { codigo: "consulta_fallida", estado: 503 });
  await pulsar();
  assert.match(aviso.textContent, /No se ha podido descargar el PDF firmado/u);
  assert.doesNotMatch(aviso.textContent, /503|consulta_fallida/u);
  gestor.retirar();
});

test("los importadores locales de la vista y el circuito evitan las URLs immutable anteriores", async () => {
  const [vista, pruebas] = await Promise.all([
    readFile(new URL("./vista-expedientes.js", import.meta.url), "utf8"),
    readFile(new URL("./formulario-llamamiento-pruebas.js", import.meta.url), "utf8"),
  ]);
  const versiones = new Map([
    ["circuito-firma.js", "20261001-ct-firma-verificador-v2"],
    ["vista-expedientes.js", "20261001-ct-firma-verificador-v2"],
  ]);
  const anterior = "20260929-custodia-506-v1";
  const importadores = [
    [vista, "circuito-firma.js"],
    [pruebas, "vista-expedientes.js"],
  ];
  for (const [codigo, modulo] of importadores) {
    const rutas = [...codigo.matchAll(new RegExp(`\\./${modulo.replace(".", "\\.")}\\?v=([^"']+)`, "gu"))];
    assert.equal(rutas.length, 1, modulo);
    assert.equal(rutas[0][1], versiones.get(modulo), modulo);
    assert.notEqual(`${modulo}?v=${rutas[0][1]}`, `${modulo}?v=${anterior}`);
  }
});
