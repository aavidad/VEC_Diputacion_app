import test from 'node:test';
import assert from 'node:assert/strict';
import { seleccionarOpciones, resumirRecuperacionFirmas } from './recorrer.mjs';
import { validarReciboFirma, origen } from './config.mjs';

test('B2 selecciona referencias del expediente consultado y corta ambigüedades', () => {
  const consulta = { opciones: { vacantes: [{ plaza_ref: 'plaza:1' }, { plaza_ref: 'plaza:2' }],
    regimenes: [{ ref: 'regimen:1' }], modalidades: [{ ref: 'modalidad:1' }],
    clases_ocupacion: [{ valor: 'titular' }], motivos: ['incorporacion'],
    documentos: [{ documento_ref: 'documento:1' }], periodo: { fuente_ref: '', desde: '', hasta: '' } } };
  const seleccion = { vacante: 'plaza:2', regimen: 'regimen:1', modalidad: 'modalidad:1',
    clase_ocupacion: 'titular', motivo: 'incorporacion', documento: 'documento:1',
    desde: '2026-10-02', hasta: '' };
  assert.equal(seleccionarOpciones(consulta, seleccion).vacante, '1');
  assert.throws(() => seleccionarOpciones(consulta, { ...seleccion, vacante: 'plaza:otra' }), /b2_opcion/u);
  consulta.opciones.vacantes.push({ plaza_ref: 'plaza:2' });
  assert.throws(() => seleccionarOpciones(consulta, seleccion), /b2_opcion/u);
});

test('recuperación E3 exige pasos firmados, recibo y fecha idénticos', () => {
  const firma = { documento: 'resolucion', orden: 1, recibo_ref: 'recibo:firma:1',
    registrada_en: '2026-10-01T10:00:00Z', custodiado: null };
  const estado = { documentos: [{ documento: 'resolucion', completo: true, pasos: [{ orden: 1,
    estado: 'firmado', recibo_ref: firma.recibo_ref, registrada_en: firma.registrada_en }] }] };
  assert.equal(resumirRecuperacionFirmas(estado, [firma]).recibos_cotejados, 1);
  assert.throws(() => resumirRecuperacionFirmas(estado, [{ ...firma, recibo_ref: 'recibo:otro' }]), /recuperacion/u);
  assert.throws(() => resumirRecuperacionFirmas({ documentos: [{ ...estado.documentos[0], completo: false }] }, [firma]), /firma_incompleta/u);
});

test('el recibo E3 exige verificación positiva y conserva límite de eficacia', () => {
  const recibido = { esquema: 'vec.contratacion-temporal.recibo-firma-documento.v1', expediente_ref: 'expediente:b2:1',
    documento: 'resolucion', paso_orden: 1, resultado: 'firmado', firma_verificada: true, firma_eficaz: false,
    verificacion: { estado: 'valida', motivo: 'verificada', firmado_sha256: 'a'.repeat(64) },
    recibo_ref: 'recibo:firma:1', registrada_en: '2026-10-01T10:00:00Z' };
  assert.equal(validarReciboFirma(recibido, 'expediente:b2:1', 'resolucion', 1).firmado_sha256, 'a'.repeat(64));
  assert.throws(() => validarReciboFirma({ ...recibido, firma_eficaz: true }, 'expediente:b2:1', 'resolucion', 1), /entrada/u);
});

test('la red auxiliar solo admite un origen local exacto', () => {
  assert.equal(origen('https://127.0.0.1:8443'), 'https://127.0.0.1:8443');
  assert.equal(origen('wss://127.0.0.1:63117', { auxiliar: true }), 'wss://127.0.0.1:63117');
  assert.throws(() => origen('https://example.org:443'), /entrada/u);
  assert.throws(() => origen('https://127.0.0.1:8443/api'), /entrada/u);
});
