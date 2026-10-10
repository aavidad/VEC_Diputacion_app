import test from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { ESQUEMA_CANDIDATOS_TURNO, validarRespuestaCandidatosBolsa } from "./portal-bolsas-contrato.js?v=20261001-ct-a-i18n-v1";
import { consultarCandidatosBolsa } from "./portal-bolsas-api.js?v=20261010-ct-bolsa-cohorte-v10";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js?v=20261010-ct-bolsa-cohorte-v10";
import { cargarMensajesPortal } from "./portal-i18n.js?v=20261001-ct-a-i18n-v1";

const bolsa = {
  bolsa_ref: "bolsa:sintetica:1", categoria_clave: "auxiliar", categoria: "Auxiliar",
  tipo_lista: "rotatoria", vigente_desde: "2026-09-18T00:00:00Z", vigente_hasta: null,
  total: 2, llamamientos_en_curso: 0,
  por_estado: { disponible: 1, no_disponible: 1, trabajando: 0, pendiente_incorporacion: 0,
    renuncia: 0, excluido: 0, disponible_desde: 0 },
  politica_orden: { politica_ref: "politica:sintetica:1", version: 3, criterio: "puntuacion_desc_acta",
    tipo_lista: "rotatoria", reposicion: "misma_posicion", provisional: true,
    rotulo: "Provisional, pendiente de RRHH (dudas 13–14)", actor: "sistema:prueba",
    vigente_desde: "2026-09-18T00:00:00Z" },
};
const turno = {
  politica_ref: "politica:sintetica:1", politica_version: 3, provisional: true,
  ultimo_llamado: { participacion_ref: "participacion:sintetica:2", nombre_visible: "Lucía <Prueba>",
    orden: 2, comunicado_en: "2026-09-20T10:00:00Z", canal: "correo", resultado: "enviado" },
  siguiente: { participacion_ref: "participacion:sintetica:1", nombre_visible: "María & Prueba", orden: 1 },
  estado_siguiente: "primero_disponible",
};
function respuesta(turnoDato = turno, bolsaDato = bolsa) {
  return { data: { esquema: ESQUEMA_CANDIDATOS_TURNO, generado_en: "2026-09-21T08:00:00Z",
    bolsa: bolsaDato, turno: turnoDato, candidatos: [], contactos: [], hay_mas: false,
    cursor_siguiente: null } };
}
function presentar(datos) {
  return crearPresentadorPanelInterno({
    claseEstado: (valor) => valor,
    encabezadoVista: (_vista, titulo) => `<h2>${titulo}</h2>`,
    escaparHTML: (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;"),
    numero: (valor) => String(valor),
    obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }),
    tituloVista: (vista) => vista,
    obtenerDatosCandidatosBolsa: () => ({ carga: "listo", datos }),
    obtenerEstadoCandidatos: () => ({ estado: "", texto: "" }),
  }).renderizarVista("bolsa-candidatos");
}

test("la lectura v2 conserva el turno servido y lo sitúa antes de filtros y tabla", async () => {
  const leida = await consultarCandidatosBolsa("bolsa:sintetica:1", {}, {
    fetchImpl: async () => ({ ok: true, json: async () => respuesta() }),
  });
  assert.equal(leida.ok, true);
  assert.deepEqual(leida.datos.turno.siguiente, turno.siguiente);
  assert.ok(Object.isFrozen(leida.datos.turno));
  const html = presentar(leida.datos);
  assert.ok(html.indexOf("Turno de la bolsa") < html.indexOf('data-bolsa-form="filtros"'));
  assert.ok(html.indexOf("Turno de la bolsa") < html.indexOf("tabla-datos--candidatos"));
  assert.match(html, /Último intento de llamamiento registrado[\s\S]*Lucía &lt;Prueba&gt;/);
  assert.match(html, /Primero disponible según el orden vigente[\s\S]*María &amp; Prueba/);
  assert.match(html, /Orden pendiente de aprobar/);
  assert.match(html, /Por puntuación; si hay empate, por orden del acta · Lista rotatoria · Reposición: Misma posición/);
  assert.match(html, /Correo · Enviado/);
  assert.match(html, /La selección para un puesto requiere comprobar las condiciones del llamamiento/);
  assert.doesNotMatch(html, /Lucía <Prueba>|María & Prueba|politica:sintetica/);
});

test("distingue bolsa vacía, ausencia de llamadas y ausencia de disponibles", () => {
  const sinDisponibles = { ...turno, ultimo_llamado: null, siguiente: null, estado_siguiente: "sin_disponibles" };
  const sinContacto = validarRespuestaCandidatosBolsa(respuesta(sinDisponibles));
  const html = presentar(sinContacto);
  assert.match(html, /Aún no hay intentos de llamamiento registrados/);
  assert.match(html, /No hay personas disponibles en esta bolsa/);
  const bolsaVacia = { ...bolsa, total: 0, por_estado: { ...bolsa.por_estado, disponible: 0, no_disponible: 0 } };
  const vacia = validarRespuestaCandidatosBolsa(respuesta(sinDisponibles, bolsaVacia));
  assert.match(presentar(vacia), /Esta bolsa no tiene aspirantes/);
});

test("v2 exige turno completo, política coincidente y estado siguiente coherente", () => {
  const sinTurno = respuesta(); delete sinTurno.data.turno;
  assert.throws(() => validarRespuestaCandidatosBolsa(sinTurno));
  assert.throws(() => validarRespuestaCandidatosBolsa(respuesta({ ...turno, politica_version: 2 })));
  assert.throws(() => validarRespuestaCandidatosBolsa(respuesta({ ...turno, siguiente: null })));
  assert.throws(() => validarRespuestaCandidatosBolsa(respuesta({ ...turno, siguiente: { ...turno.siguiente, orden: 0 } })));
  assert.throws(() => validarRespuestaCandidatosBolsa(respuesta({ ...turno, siguiente: { ...turno.siguiente, nombre_visible: "alguien@example.com" } })));
  assert.throws(() => validarRespuestaCandidatosBolsa(respuesta({ ...turno, ultimo_llamado: { ...turno.ultimo_llamado, resultado: "codigo_no_catalogado" } })));
});

test("traduce los resultados de intentos que no llegaron a su destinatario", () => {
  const dato = respuesta({ ...turno, ultimo_llamado: { ...turno.ultimo_llamado, orden: null, canal: "telefono", resultado: "numero_erroneo" } });
  const html = presentar(validarRespuestaCandidatosBolsa(dato));
  assert.match(html, /Teléfono · Número erróneo/);
  assert.match(html, /Lucía &lt;Prueba&gt;<\/strong> · sin puesto vigente/);
  assert.doesNotMatch(html, /Lucía &lt;Prueba&gt; · puesto 0/);
});

test("un turno inválido se informa sin exponer nombres de campos del contrato", async () => {
  const dato = respuesta(); delete dato.data.turno;
  const leida = await consultarCandidatosBolsa("bolsa:sintetica:1", {}, {
    fetchImpl: async () => ({ ok: true, json: async () => dato }),
  });
  assert.equal(leida.ok, false);
  assert.equal(leida.codigo, "error_red_o_contrato");
  assert.equal(leida.mensaje, "No se pudo cargar la relación de aspirantes.");
});

test("la lectura anterior no inventa un turno y las claves nuevas existen en ambos idiomas", async () => {
  const anterior = respuesta();
  anterior.data.esquema = "vec.bolsa.rrhh.candidatos.v1";
  delete anterior.data.turno;
  const datos = validarRespuestaCandidatosBolsa(anterior);
  assert.equal(datos.turno, null);
  assert.doesNotMatch(presentar(datos), /Turno de la bolsa/);
  for (const idioma of ["es", "en"]) {
    const mensajes = await cargarMensajesPortal(idioma);
    assert.equal(mensajes.txt_n_registros, idioma === "es" ? "Registros: {numero}" : "Records: {numero}");
    for (const clave of ["bolsa_turno_titulo", "bolsa_turno_ultimo", "bolsa_turno_siguiente",
      "bolsa_turno_regla_provisional", "bolsa_turno_sin_disponibles", "bolsa_turno_aviso",
      "bolsa_historico_estadisticas_pendiente"]) {
      assert.ok(mensajes[clave], `${idioma}: ${clave}`);
    }
  }
});

test("la lista inglesa traduce canal, resultado, falta de turno y tipo desde el catálogo activo", () => {
  const dato = respuesta().data;
  dato.candidatos = [{ participacion_ref: "participacion:sintetica:3", orden: null, orden_acta: 3,
    razon_orden: "sin_turno", nombre_visible: "Persona sintética", documento_enmascarado: "***0003**",
    estado_clave: "en_revision", estado_desde: "2026-09-21T08:00:00Z", disponible_desde: null,
    ultimo_llamamiento: { llamamiento_ref: "llamamiento:sintetico:3", comunicado_en: "2026-09-20T10:00:00Z",
      canal: "correo", resultado: "enviado" }, contactos_total: 1 }];
  const modulo = new URL("./portal-panel-interno.js", import.meta.url).href;
  const programa = `globalThis.location={href:"https://vec.example/portal-empleado/?lang=en",search:"?lang=en"};
    const {crearPresentadorPanelInterno}=await import(${JSON.stringify(modulo)});
    const datos=${JSON.stringify(dato)};
    const presentador=crearPresentadorPanelInterno({
      claseEstado:(valor)=>valor, encabezadoVista:(_vista,titulo)=>'<h2>'+titulo+'</h2>',
      escaparHTML:(valor)=>String(valor??'').replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;'),
      numero:(valor)=>String(valor), obtenerDatosPanel:()=>({esquema:'vec.bolsa.panel.interno.v1'}),
      tituloVista:(vista)=>vista, obtenerDatosCandidatosBolsa:()=>({carga:'listo',datos}),
      obtenerEstadoCandidatos:()=>({estado:'',texto:''})});
    process.stdout.write(presentador.renderizarVista('bolsa-candidatos'));`;
  const ejecutado = spawnSync(process.execPath, ["--input-type=module", "-e", programa],
    { encoding: "utf8", maxBuffer: 2 * 1024 * 1024 });
  assert.equal(ejecutado.status, 0, ejecutado.stderr);
  assert.match(ejecutado.stdout, /Email · Sent/u);
  assert.match(ejecutado.stdout, /no current position/u);
  assert.match(ejecutado.stdout, /Rotating list/u);
  assert.doesNotMatch(ejecutado.stdout, /Correo · Enviado|Sin turno|Rotatoria/u);
});
