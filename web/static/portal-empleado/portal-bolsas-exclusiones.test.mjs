import assert from "node:assert/strict";
import test from "node:test";
import { validarExclusionesOrigen, validarRespuestaCandidatosBolsa } from "./portal-bolsas-contrato.js";
import { renderizarMotivosNoOfrecimiento, renderizarSeguimientoLlamamiento } from "./portal-bolsas-seguimiento.js";

const contexto = { bolsaRef: "bolsa:uno", llamamientoRef: "llamamiento:uno", exclusionesDesde: 0 };
const pagina = () => ({ bolsa_ref: contexto.bolsaRef, llamamiento_ref: contexto.llamamientoRef,
  exclusiones: [{ participacion_ref: "participacion:uno", orden_acta: 1, situacion_en_origen: "en_revision",
    motivo_clave: "pendiente_comprobacion_rrhh", eventos_relacion: ["evento:opaco"],
    hechos_relacion: [{ evento_ref: "evento:opaco", tipo: "incorporacion", ocurrido_en: "2026-10-14T08:00:00Z" }] }],
  total_exclusiones: 2, hay_mas: true, siguiente_orden_acta: 1,
  auditoria_ref: "auditoria:uno", correlacion_ref: "correlacion:uno", consultada_en: "2026-10-15T10:00:00Z",
  criterios_ofrecimiento: { version_politica_ofertas: 3, huella_politica_ofertas_sha256: "a".repeat(64),
    modalidad_ofrecida: "vacante", causa_ofrecida: "vacante",
    configuracion: { fuente_normativa: "Reglamento" } } });
const mostrar = (origen, exclusionesDesde = 0) => renderizarMotivosNoOfrecimiento(origen, {
  escaparHTML: (v) => String(v).replaceAll("&", "&amp;").replaceAll("<", "&lt;"),
  numero: (v) => String(v), instanteVisible: () => "14/10/2026, 10:00", exclusionesDesde,
});

test("motivos pendientes y hechos de relación conservan el estado provisional sin enseñar referencias", () => {
  const origen = validarExclusionesOrigen({ estado: "disponible", pagina: pagina() }, contexto);
  const html = mostrar(origen);
  assert.match(html, /Pendiente de comprobación por RRHH/u);
  assert.match(html, /Regla aplicada: versión 3/u);
  assert.match(html, /Ver versión y hechos consultados \(1\)/u);
  assert.match(html, /Incorporación:.*14\/10\/2026/u);
  assert.match(html, /data-bolsa-accion="pagina-exclusiones"/u);
  assert.doesNotMatch(html, /evento:opaco|participacion:uno|excluido/u);
  assert.match(mostrar(origen, 1), /data-bolsa-accion="primera-pagina-exclusiones"/u);
});

test("permisos y lectura fallida afectan solo al bloque de motivos", () => {
  assert.match(mostrar(validarExclusionesOrigen({ estado: "sin_permiso" }, contexto)), /No tiene permiso/u);
  assert.match(mostrar(validarExclusionesOrigen({ estado: "no_disponible", error_estado: "lectura_no_disponible" }, contexto)), /Vuelva a intentarlo/u);
  assert.match(mostrar(validarExclusionesOrigen({ estado: "no_disponible", error_estado: "lectura_no_disponible" }, contexto)), /data-bolsa-accion="reintentar-candidatos"/u);
  const historica = pagina(); historica.criterios_ofrecimiento = null;
  historica.exclusiones[0].eventos_relacion = []; historica.exclusiones[0].hechos_relacion = [];
  assert.match(mostrar(validarExclusionesOrigen({ estado: "disponible", pagina: historica }, contexto)), /Sin datos del contrato/u);
});

test("rechaza otra bolsa, eventos ausentes o paginación incongruente", () => {
  const otra = pagina(); otra.bolsa_ref = "bolsa:otra";
  assert.throws(() => validarExclusionesOrigen({ estado: "disponible", pagina: otra }, contexto));
  const sinHechos = pagina(); delete sinHechos.exclusiones[0].eventos_relacion;
  assert.throws(() => validarExclusionesOrigen({ estado: "disponible", pagina: sinHechos }, contexto));
  const hechoAjeno = pagina(); hechoAjeno.exclusiones[0].hechos_relacion[0].evento_ref = "evento:ajeno";
  assert.throws(() => validarExclusionesOrigen({ estado: "disponible", pagina: hechoAjeno }, contexto));
  const malCursor = pagina(); malCursor.siguiente_orden_acta = 2;
  assert.throws(() => validarExclusionesOrigen({ estado: "disponible", pagina: malCursor }, contexto));
});

test("motivos malformados se descartan sin ocultar el seguimiento validado", () => {
  const candidato = { participacion_ref: "participacion:uno", orden: 1, orden_acta: 1, razon_orden: "orden_acta",
    nombre_visible: "Ana de Prueba", documento_enmascarado: "***1234**", estado_clave: "disponible",
    estado_desde: "2026-10-15T08:00:00Z", disponible_desde: null, ultimo_llamamiento: null, contactos_total: 1 };
  const contacto = { contacto_ref: "contacto:uno", participacion_ref: candidato.participacion_ref,
    llamamiento_ref: contexto.llamamientoRef, canal: "telefono", instante: "2026-10-15T09:00:00Z",
    actor_ref: "actor:rrhh", resultado: "no_contesta", anotacion: "" };
  const bolsa = { bolsa_ref: contexto.bolsaRef, categoria_clave: "auxiliar", categoria: "Auxiliar",
    tipo_lista: "rotatoria", vigente_desde: "2026-10-01T00:00:00Z", vigente_hasta: null,
    total: 1, llamamientos_en_curso: 1,
    por_estado: { disponible: 1, no_disponible: 0, trabajando: 0, pendiente_incorporacion: 0,
      renuncia: 0, excluido: 0, disponible_desde: 0 },
    politica_orden: { politica_ref: "politica:uno", version: 1, criterio: "puntuacion_desc_acta",
      tipo_lista: "rotatoria", reposicion: "misma_posicion", provisional: true, rotulo: "Provisional, pendiente de RRHH (dudas 13–14)",
      actor: "sistema:prueba", vigente_desde: "2026-10-01T00:00:00Z" } };
  const malformada = { estado: "disponible", pagina: { ...pagina(), campo_privado: "NO MOSTRAR" } };
  const datos = { esquema: "vec.bolsa.rrhh.candidatos.v1", generado_en: "2026-10-15T10:00:00Z",
    bolsa, candidatos: [candidato], contactos: [contacto], hay_mas: false, cursor_siguiente: null,
    canales_llamamiento: [{ canal: "correo", modo: "automatico", al_emitir: true, seguimiento: false },
      { canal: "telefono", modo: "manual", al_emitir: false, seguimiento: true,
        resultados: ["no_contesta", "acepta"], resultados_cierre: ["acepta"] }],
    respuestas_portal: { llamamiento_ref: contexto.llamamientoRef, respuestas: [] },
    exclusiones_origen: malformada };
  const registrados = []; const anterior = console.error;
  let validados;
  try { console.error = (...args) => registrados.push(args);
    validados = validarRespuestaCandidatosBolsa({ data: datos }, contexto);
  } finally { console.error = anterior; }
  assert.equal(validados.exclusiones_origen.estado, "no_disponible");
  assert.equal(validados.candidatos[0].nombre_visible, candidato.nombre_visible);
  assert.equal(validados.contactos[0].contacto_ref, contacto.contacto_ref);
  assert.deepEqual(registrados, [["bolsa.exclusiones_contrato_invalido", { tipo: "Error" }]]);
  const html = renderizarSeguimientoLlamamiento({ estadoCandidatos: { carga: "listo", datos: validados },
    seguimiento: { bolsa_ref: contexto.bolsaRef, llamamiento_ref: contexto.llamamientoRef }, modalFicha: null,
    encabezadoVista: () => "<h2>Seguimiento</h2>", escaparHTML: (v) => String(v), numero: (v) => String(v),
    instanteVisible: () => "15/10/2026", renderizarFicha: () => "" });
  assert.match(html, /Ana de Prueba/u);
  assert.match(html, /No contesta|Sin llamadas/u);
  assert.match(html, /Ahora no se pueden consultar estos motivos/u);
  assert.doesNotMatch(html, /NO MOSTRAR|campo_privado|pendiente_comprobacion_rrhh/u);
});
