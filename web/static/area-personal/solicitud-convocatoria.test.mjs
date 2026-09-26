import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearClienteSolicitudesPersona, RUTAS_SOLICITUDES_PERSONA } from "./cliente-http-solicitudes.js";
import { cantidadContrato, cuerpoBorrador, formularioDesdeBorrador, formularioVacio, renderizarSolicitudConvocatoria } from "./solicitud-convocatoria.js";
import { renderizarMisSolicitudes } from "./mis-solicitudes.js";
import { calcularAutobaremoOrientativo, leerReglasConvocatoria } from "./reglas-convocatoria.js";
import { MENSAJES_SOLICITUD_ES } from "./i18n-solicitud.js";
import { crearServidorSimulado } from "./solicitudes-servidor-simulado.test-helper.mjs";

const REF = "bolsa-operario-diputacion-2026";
let contador = 0;
function cliente(servidor) {
  const base = crearClienteSolicitudesPersona({ fetchImpl: servidor.fetch });
  return { ...base, nuevaClaveIdempotencia: () => `prueba-${++contador}-clave` };
}
async function rechazo(promesa, codigo) {
  await assert.rejects(promesa, (error) => error?.codigo === codigo);
}

test("el cliente usa las rutas de Selección, mismo origen y ninguna identidad en el cuerpo", async () => {
  const servidor = crearServidorSimulado();
  const api = cliente(servidor);
  const reglas = await api.convocatoria(REF);
  assert.equal(reglas.titulo.startsWith("Bolsa de empleo"), true);
  const guardado = await api.guardarBorrador({ convocatoria_ref: REF, version_esperada: 0, datos: { nombre: "Antonio" }, requisitos: [], meritos: [] }, "clave-guardado-1");
  assert.equal(guardado.version, 1);
  const peticion = servidor.peticiones.find((item) => item.metodo === "PUT");
  assert.equal(peticion.ruta, RUTAS_SOLICITUDES_PERSONA.borrador);
  assert.equal(peticion.cabeceras["Idempotency-Key"], "clave-guardado-1");
  assert.ok(Object.values(RUTAS_SOLICITUDES_PERSONA).every((ruta) => ruta.startsWith("/api/vec/seleccion/mis-solicitudes")));
  const fuente = await readFile(new URL("./cliente-http-solicitudes.js", import.meta.url), "utf8");
  assert.match(fuente, /credentials: "same-origin", cache: "no-store", redirect: "error",\s+referrerPolicy: "no-referrer"/u);
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|document\.cookie|persona_ref/u);
});

test("borrador: versión obsoleta, repetición idempotente y borrador recuperable", async () => {
  const servidor = crearServidorSimulado();
  const api = cliente(servidor);
  assert.equal(await api.obtenerBorrador(REF), null, "sin borrador responde null");
  const cuerpo = { convocatoria_ref: REF, version_esperada: 0, turno: "libre", datos: { nombre: "Antonio" }, requisitos: [{ clave: "nacionalidad", estado: "cumple" }], meritos: [] };
  const primero = await api.guardarBorrador(cuerpo, "clave-repeticion");
  const repetido = await api.guardarBorrador(cuerpo, "clave-repeticion");
  assert.equal(repetido.version, primero.version);
  assert.equal(repetido.repetida, true);
  await rechazo(api.guardarBorrador({ ...cuerpo, datos: { nombre: "Otro" } }, "clave-repeticion"), "clave_reutilizada");
  await rechazo(api.guardarBorrador({ ...cuerpo, version_esperada: 0, datos: { nombre: "Lucía" } }, "clave-otra"), "version_obsoleta");
  const leido = await api.obtenerBorrador(REF);
  assert.equal(leido.version, 1);
  assert.deepEqual(formularioDesdeBorrador(leido).requisitos, { nacionalidad: "cumple" });
});

test("presentación: justificante interno, repetición con el mismo recibo y servicios no realizados", async () => {
  const servidor = crearServidorSimulado();
  const api = cliente(servidor);
  const borrador = await api.guardarBorrador({ convocatoria_ref: REF, version_esperada: 0, turno: "libre", datos: { nombre: "Antonio", apellidos: "Reyes Álvarez" }, requisitos: [{ clave: "nacionalidad", estado: "cumple" }, { clave: "titulacion", estado: "pendiente" }], meritos: [{ clave_grupo: "experiencia", clave_merito: "meses_diputacion", descripcion: "", cantidad: "14" }] }, "clave-b1");
  assert.equal(borrador.puntuacion_autobaremo, "1.4");
  const presentada = await api.presentar({ solicitudRef: borrador.solicitud_ref, versionEsperada: 1 }, "clave-p1");
  assert.match(presentada.numero_justificante, /^2026\/SOL-\d{6}$/u);
  assert.equal(presentada.servicios.firma, "no_disponible");
  const repetida = await api.presentar({ solicitudRef: borrador.solicitud_ref, versionEsperada: 1 }, "clave-p1");
  assert.equal(repetida.recibo_ref, presentada.recibo_ref);
  assert.equal(repetida.numero_justificante, presentada.numero_justificante);
  await rechazo(api.presentar({ solicitudRef: borrador.solicitud_ref, versionEsperada: 1 }, "clave-p2"), "ya_presentada");
  const lista = await api.listar();
  assert.equal(lista[0].estado, "presentada");
  assert.equal(lista[0].numero_justificante, presentada.numero_justificante);
});

test("fuera de plazo, requisito que impide presentar y servicio no montado llegan como códigos traducibles", async () => {
  let abierto = true;
  const servidor = crearServidorSimulado({ plazoAbierto: () => abierto });
  const api = cliente(servidor);
  const b = await api.guardarBorrador({ convocatoria_ref: REF, version_esperada: 0, datos: {}, requisitos: [{ clave: "nacionalidad", estado: "no_cumple" }], meritos: [] }, "clave-x1");
  await rechazo(api.presentar({ solicitudRef: b.solicitud_ref, versionEsperada: 1 }, "clave-x2"), "requisito_no_cumplido");
  abierto = false;
  await rechazo(api.presentar({ solicitudRef: b.solicitud_ref, versionEsperada: 1 }, "clave-x3"), "fuera_de_plazo");
  await rechazo(cliente(crearServidorSimulado({ montado: false })).listar(), "recurso_no_encontrado");
  const sinRuta = crearClienteSolicitudesPersona({ fetchImpl: async () => ({ status: 404, headers: { get: () => "text/plain" }, text: async () => "" }) });
  await rechazo(sinRuta.listar(), "no_disponible");
  for (const codigo of ["fuera_de_plazo", "requisito_no_cumplido", "version_obsoleta", "no_disponible", "clave_reutilizada", "convocatoria_actualizada"]) {
    assert.equal(typeof MENSAJES_SOLICITUD_ES[`error_${codigo}`], "string", codigo);
  }
});

test("el cuerpo del borrador sigue el contrato: tres estados, méritos opcionales y cantidades decimales", async () => {
  const reglas = leerReglasConvocatoria((await crearServidorSimulado().atender("GET", `${RUTAS_SOLICITUDES_PERSONA.convocatoria}?convocatoria_ref=${REF}`)).cuerpo.data);
  const formulario = formularioVacio();
  formulario.requisitos = { nacionalidad: "cumple", edad: "pendiente", titulacion: "no_cumple" };
  formulario.turno = "libre";
  const sinMeritos = cuerpoBorrador({ convocatoriaRef: REF, reglas, formulario, versionEsperada: 0 });
  assert.deepEqual(sinMeritos.meritos, [], "no se exige ningún mérito");
  assert.deepEqual(sinMeritos.requisitos.map((item) => item.estado), ["cumple", "pendiente", "no_cumple"]);
  assert.equal(sinMeritos.turno, "libre");
  assert.equal("puntos_autobaremo" in sinMeritos, false);
  formulario.meritos["experiencia|meses_diputacion"] = { cantidad: "14,5", descripcion: "Peón" };
  const conMerito = cuerpoBorrador({ convocatoriaRef: REF, reglas, formulario, versionEsperada: 2 });
  assert.deepEqual(conMerito.meritos, [{ clave_grupo: "experiencia", clave_merito: "meses_diputacion", descripcion: "Peón", cantidad: "14.5" }]);
  assert.equal(cantidadContrato("1.2345"), "");
  assert.equal(cantidadContrato("abc"), "");
  const calculo = calcularAutobaremoOrientativo(reglas.baremo, new Map([["experiencia|meses_diputacion", 200], ["experiencia|meses_otras", 200]]));
  assert.equal(calculo.porMerito.get("experiencia|meses_diputacion"), 8, "máximo del mérito");
  assert.equal(calculo.total, 12, "máximo del grupo");
});

test("la vista no muestra referencias internas ni texto de ayuda, y usa solo claves del catálogo", async () => {
  const servidor = crearServidorSimulado();
  const reglas = leerReglasConvocatoria(servidor.atender("GET", `${RUTAS_SOLICITUDES_PERSONA.convocatoria}?convocatoria_ref=${REF}`).cuerpo.data);
  const base = { convocatoriaRef: REF, fase: "listo", reglas, formulario: formularioVacio(), borrador: null, error: "" };
  for (const paso of [1, 2, 3, 4]) {
    const html = renderizarSolicitudConvocatoria({ ...base, paso });
    assert.match(html, /data-ayuda-solicitud="ayuda_/u, "cada paso tiene su botón «?»");
    assert.doesNotMatch(html, new RegExp(MENSAJES_SOLICITUD_ES.ayuda_requisitos.slice(0, 40), "u"));
    assert.doesNotMatch(html, new RegExp(MENSAJES_SOLICITUD_ES.ayuda_meritos.slice(0, 40), "u"));
  }
  const revision = renderizarSolicitudConvocatoria({ ...base, paso: 4 });
  for (const servicio of ["firma", "registro_sede", "tasas", "notificacion"]) assert.match(revision, new RegExp(`aria-describedby="solicitud-motivo-${servicio}"`, "u"));
  assert.match(renderizarSolicitudConvocatoria({ ...base, paso: 3 }), /Se aportarán más adelante/u);
  const justificante = renderizarSolicitudConvocatoria({ ...base, fase: "presentada", justificante: { numero_justificante: "2026/SOL-000031", presentada_en: "2026-09-26T10:15:00.000000Z", recibo_ref: "recibo:sol_000001", puntuacion_autobaremo: "1.4", servicios: { firma: "no_disponible" } } });
  assert.match(justificante, /2026\/SOL-000031/u);
  assert.doesNotMatch(justificante.replace(/data-copiar-referencia="[^"]*"/gu, ""), /recibo:sol_000001/u, "el recibo solo viaja en el botón de copiar");
  assert.doesNotMatch(justificante, /registro administrativo|asiento/iu);
  const lista = renderizarMisSolicitudes({ fase: "listo", solicitudes: [{ solicitud_ref: "sol_000001", convocatoria_ref: REF, convocatoria_titulo: "Bolsa", estado: "presentada", version: 1, presentada_en: "2026-09-26T10:15:00.000000Z", numero_justificante: "2026/SOL-000031", puntuacion_autobaremo: "1.4" }] });
  assert.doesNotMatch(lista.replace(/data-copiar-referencia="[^"]*"/gu, ""), /sol_000001/u);
  for (const archivo of ["solicitud-convocatoria.js", "mis-solicitudes.js"]) {
    const fuente = await readFile(new URL(`./${archivo}`, import.meta.url), "utf8");
    for (const [, clave] of fuente.matchAll(/\bt\("([a-z_]+)"/gu)) assert.equal(typeof MENSAJES_SOLICITUD_ES[clave], "string", `${archivo}: ${clave}`);
    assert.doesNotMatch(fuente, />[¿¡A-ZÁÉÍÓÚÑ][a-záéíóúñ]+[^<$]*</u, `${archivo}: texto literal en plantilla`);
    assert.doesNotMatch(fuente, /(?:aria-label|title|placeholder)="[A-ZÁÉÍÓÚÑ]/u, `${archivo}: atributo literal`);
  }
});
