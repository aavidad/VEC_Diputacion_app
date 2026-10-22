import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { prepararGuardado, crearEscritorBasesHTTP } from './guardar.js';
import { RUTA_GUARDAR_BASES } from './cliente-http.js';

const cli = JSON.parse(await readFile(new URL('./testdata/preparacion-incompleta.json', import.meta.url)));
const huella = 'a'.repeat(64);
function respuesta(solicitud, estado = 'guardada') {
  return { estado, preparacion: { ambito: { organizacion_ref: 'org_' + 'a'.repeat(16) },
    estado: { ...solicitud.esperada, revision: solicitud.esperada.revision + 1, huella_material_sha256: huella }, material: solicitud.material },
  pendientes: cli.preparacion.pendientes,
  recibo: { recibo_ref: 'recibo:original', historia_ref: 'historia:original', auditoria_ref: 'auditoria:original', evento_ref: 'evento:original', huella_intencion_sha256: huella, confirmada_en: '2026-10-01T12:00:00Z' },
  acceso: { decision_ref: 'decision:acceso', consumo_huella_sha256: huella, auditoria_ref: 'auditoria:acceso', recibo_ref: 'recibo:acceso', correlacion_ref: 'correlacion:acceso', accedida_en: '2026-10-02T12:00:00Z' } };
}
const http = (dto, status) => new Response(JSON.stringify(dto), { status, headers: { 'Content-Type': 'application/json; charset=utf-8' } });

test('alta mínima y continuación conservan preimagen y clave propia', () => {
  let n = 0; const uuid = () => `id-${++n}`;
  const alta = prepararGuardado(null, '  Bases de prueba  ', 'conjunta', uuid);
  assert.deepEqual(alta.esperada, { preparacion_ref: 'id-1', revision: 0, huella_material_sha256: '' });
  assert.equal(alta.clave_operacion, 'id-2'); assert.equal(alta.material.contenido.titulo, 'Bases de prueba');
  assert.equal(alta.material.contenido.modalidad_inscripcion, 'conjunta'); assert.deepEqual(alta.material.referencias, []);
  const base = respuesta(alta); const siguiente = prepararGuardado(base, 'Otro título', 'separada', uuid);
  assert.deepEqual(siguiente.esperada, base.preparacion.estado); assert.equal(siguiente.material.contenido.modalidad_inscripcion, 'separada');
  assert.equal(base.preparacion.material.contenido.titulo, 'Bases de prueba');
  assert.throws(() => prepararGuardado(null, '', 'conjunta', uuid), /solicitud_invalida/u);
});

test('tras respuesta perdida repite cuerpo y clave y sólo acepta recibo válido', async () => {
  const solicitud = prepararGuardado(null, 'Bases de prueba', 'conjunta', () => 'preparacion:test');
  const cuerpos = []; let n = 0;
  const escritor = crearEscritorBasesHTTP({ fetchImpl: async (ruta, opciones) => {
    assert.equal(ruta, RUTA_GUARDAR_BASES); cuerpos.push(opciones.body);
    assert.equal(opciones.credentials, 'same-origin'); assert.equal(opciones.cache, 'no-store');
    if (++n === 1) throw new TypeError('red perdida');
    return http(respuesta(solicitud, 'recuperada'), 200);
  } });
  await assert.rejects(escritor.guardar(solicitud), e => e.codigo === 'servicio_no_disponible');
  const recuperada = await escritor.guardar(solicitud);
  assert.equal(recuperada.estado, 'recuperada'); assert.equal(recuperada.recibo.recibo_ref, 'recibo:original');
  assert.deepEqual(cuerpos, [JSON.stringify(solicitud), JSON.stringify(solicitud)]);
  const falso = crearEscritorBasesHTTP({ fetchImpl: async () => http({ ...respuesta(solicitud), recibo: {} }, 201) });
  await assert.rejects(falso.guardar(solicitud), e => e.codigo === 'respuesta_incompatible');
});
