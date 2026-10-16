import test from 'node:test';
import assert from 'node:assert/strict';
import { eleccionTrasConsulta, enfocarEstadoSiConfirmacionOculta, crearIntentoGuardado } from './vista.js';

test('el conflicto conserva la elección solo para la misma preparación y permite una nueva revisión', () => {
  const deseo = { referencia: 'preparacion:una', modalidad: 'conjunta' };
  const actual = eleccionTrasConsulta(deseo, 'preparacion:una', 'separada');
  assert.deepEqual(actual, { deseo, modalidad: 'conjunta' });
  assert.deepEqual(eleccionTrasConsulta(deseo, 'preparacion:otra', 'separada'),
    { deseo: null, modalidad: 'separada' });
  const carga = (revision) => ({ dto: { preparacion: { estado: { preparacion_ref: 'preparacion:una', revision },
    material: { contenido: { modalidad_inscripcion: 'separada' } } } } });
  let numero = 0;
  const clave = () => String(++numero);
  const anterior = crearIntentoGuardado(carga(1), 'conjunta', clave);
  const posterior = crearIntentoGuardado(carga(2), actual.modalidad, clave);
  assert.equal(anterior.clave_operacion, 'modalidad:1');
  assert.equal(posterior.clave_operacion, 'modalidad:2');
  assert.equal(posterior.esperada.revision, 2);
  assert.equal(posterior.material.contenido.modalidad_inscripcion, 'conjunta');
});

test('la confirmación oculta devuelve el foco al estado solo si seguía en ella', () => {
  let movimientos = 0;
  const confirmar = { hidden: true };
  const estado = { focus(opciones) { assert.deepEqual(opciones, { preventScroll: true }); movimientos += 1; } };
  const cuerpo = {};
  const documento = { activeElement: confirmar, body: cuerpo };
  enfocarEstadoSiConfirmacionOculta(documento, confirmar, estado, true);
  assert.equal(movimientos, 1);
  documento.activeElement = {};
  enfocarEstadoSiConfirmacionOculta(documento, confirmar, estado, true);
  enfocarEstadoSiConfirmacionOculta(documento, confirmar, estado, false);
  assert.equal(movimientos, 1);
});
