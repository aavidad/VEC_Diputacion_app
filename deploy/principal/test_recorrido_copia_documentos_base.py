"""Base Documentos: fuentes de copia, motivo publicado y fallo cerrado."""

import hashlib
import json
from pathlib import Path
import tempfile
from unittest.mock import patch
import unittest

from recorrido_copia import admisiones, configuracion_firma, documentos_base
from recorrido_copia.documentos import preparar_documentos
from test_recorrido_copia_configuracion import Contexto

ROLES = ('tecnico', 'jefatura', 'diputacion', 'secretaria')


class DocumentosBaseTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.c = Contexto(Path(self.tmp.name))
        self.c.fuente = Path(__file__).resolve().parents[2]
        self.c.copia.leer = lambda: {'sid_copia': '12345678901234567', 'recorrido_completo': True}
        original = self.c.consultar
        self.c.consultar = lambda sql: ('1' if 'motivo_v2_catalogo_publicado' in sql else
                                        '4' if 'WITH esperado(rol,perfil,persona,cuenta,vinculo,version_rol)' in sql
                                        else original(sql))
        self.c.parametros['documentos_motivo_listar'] = {
            'catalogo_id': 'motivos_documentos', 'catalogo_version': 1,
            'catalogo_huella_sha256': 'a' * 64, 'entrada_clave': 'motivo_' + 'b' * 32}
        f = self.c.carpeta / 'firmantes'
        semilla_ruta = f / 'semilla-final/firmantes-datos.json'
        semilla = self.c.leer(semilla_ruta)
        self.c.guardar(self.c.material / 'identidad/identidad.json', {
            'version': 1, 'autoridad': 'no_autoritativo', 'roles': ['tecnico_rrhh'],
            'subject': 'desarrollo:operador-rrhh',
            'certificate_sha256': 'e' * 64})
        self.cuenta_tramitacion = {'sujeto': 'desarrollo:operador-rrhh',
                                  'certificado_sha256': 'e' * 64,
                                  'cuenta_ref': 'cta_' + 'e' * 32,
                                  'perfil_ref': 'prf_' + 'e' * 32}
        productor = patch.object(documentos_base, 'preparar_tramitacion',
                                 return_value=self.cuenta_tramitacion)
        self.preparar_tramitacion = productor.start()
        self.addCleanup(productor.stop)
        for rol in ROLES:
            firmante = next(x for x in semilla['firmantes']
                            if x['perfil_firma_ref'] == semilla['referencias'][rol]['perfil']['firma'])
            semilla['referencias'][rol]['perfil']['documentos'] = firmante['perfil_documentos_ref']
            semilla['referencias'][rol]['vinculo']['documentos'] = 'vinculo:docs:' + rol
        semilla_ruta.write_text(json.dumps(semilla))
        ids = ['ct_cargo_' + rol + '_' + clase + '_dev' for rol in ROLES
               for clase in ('firma', 'consulta', 'documentos', 'circuito')]
        plan = f / 'roles-plan.json'
        acuse = f / 'roles-acuse.json'
        plan.write_text(json.dumps({'cargos': [{'rol_id': i, 'version': 1,
                    'version_rol_sha256': 'b' * 64} for i in ids]}))
        acuse.write_text(json.dumps({'estado': 'permitido', 'recibo': {
            'auditoria_ref': 'auditoria:aut53',
            'plan_sha256': hashlib.sha256(plan.read_bytes()).hexdigest(),
            'cargos': [{'version_rol_ref': f'rol:{i}:v1',
                        'version_rol_sha256': 'b' * 64, 'control_sha256': 'c' * 64}
                       for i in ids]}}))
        meta = self.c.leer(f / 'metadata.json')
        meta['roles_acuse_sha256'] = hashlib.sha256(acuse.read_bytes()).hexdigest()
        (f / 'metadata.json').write_text(json.dumps(meta))
        admisiones.preparar(self.c)
        configuracion_firma.preparar(self.c)

    def test_base_propia_con_identidad_rrhh_y_custodia_nominal(self):
        destino = documentos_base.preparar(self.c)
        base = self.c.leer(destino)
        self.assertEqual(len(base['cuentas']), 1)
        self.assertEqual(base['cuentas'][0]['sujeto'], 'desarrollo:operador-rrhh')
        self.assertEqual(base['cuentas'], [self.cuenta_tramitacion])
        self.assertNotEqual(base['cuentas'][0]['sujeto'],
                            self.c.leer(self.c.carpeta / 'firmantes/semilla-final/firmantes-datos.json')['referencias']['tecnico']['persona_ref'])
        self.assertGreater(base['almacen']['tamano_maximo'], 0)
        self.assertLessEqual(base['almacen']['tamano_maximo'], 64 * 1024 * 1024)
        self.assertEqual(base['almacen']['retencion_minima_dias'], 0)
        self.assertEqual(set(base['custodia_firmado']['documentos']), {'informe_definitivo', 'resolucion'})
        self.assertEqual(documentos_base.preparar(self.c), destino)

    def test_hoja_rrhh_distinta_cierra_antes_de_publicar_configuracion(self):
        ruta = self.c.material / 'identidad/identidad.json'
        identidad = self.c.leer(ruta)
        identidad['certificate_sha256'] = 'f' * 64
        ruta.write_text(json.dumps(identidad))
        with self.assertRaisesRegex(ValueError, 'identidad o perfil ajenos'):
            documentos_base.preparar(self.c)
        self.assertFalse((self.c.carpeta / 'documentos/base.json').exists())

    def test_no_reutiliza_perfil_de_firma_o_custodia_para_tramitacion(self):
        datos = self.c.leer(self.c.carpeta / 'firmantes/semilla-final/firmantes-datos.json')
        self.cuenta_tramitacion['perfil_ref'] = datos['firmantes'][0]['perfil_documentos_ref']
        with self.assertRaisesRegex(ValueError, 'identidad o perfil ajenos'):
            documentos_base.preparar(self.c)
        self.assertFalse((self.c.carpeta / 'documentos/base.json').exists())

    def test_certificado_nominal_no_se_admite_como_tramitacion(self):
        datos = self.c.leer(self.c.carpeta / 'firmantes/semilla-final/firmantes-datos.json')
        ruta = self.c.material / 'identidad/identidad.json'
        identidad = self.c.leer(ruta)
        identidad['certificate_sha256'] = datos['referencias']['tecnico']['certificado_sha256']
        ruta.write_text(json.dumps(identidad))
        with self.assertRaisesRegex(ValueError, 'identidad RRHH original'):
            documentos_base.preparar(self.c)
        self.preparar_tramitacion.assert_not_called()

    def test_productor_recibe_rol_aprobado_y_los_dieciseis_perfiles_reservados(self):
        documentos_base.preparar(self.c)
        opciones = self.preparar_tramitacion.call_args.kwargs
        self.assertEqual(opciones['rol_documental']['version_rol_ref'],
                         'rol:ct_cargo_tecnico_documentos_dev:v1')
        self.assertEqual(len(opciones['perfiles_reservados']), 16)
        self.assertNotIn(self.cuenta_tramitacion['perfil_ref'], opciones['perfiles_reservados'])

    def test_publicacion_nativa_del_motivo_propio(self):
        del self.c.parametros['documentos_motivo_listar']
        anterior = self.c.consultar
        estado = {'publicado': False, 'sql': None}
        def consultar(sql):
            if sql.startswith('SELECT count(*) FROM vec_autorizacion.motivo_v2_entrada'):
                return '1'
            if sql.startswith('SELECT catalogo_huella_publicada_sha256'):
                return self.c.sha(self.c.carpeta / 'documentos/motivo-catalogo.json') if estado['publicado'] else ''
            return anterior(sql)
        def ejecutar_sql(raw, log):
            self.assertEqual(log, 'documentos-motivo-publicar.log')
            self.assertIn(b'vec_autorizacion.publicar_motivos_autorizacion_v2', raw)
            estado['sql'] = raw
            estado['publicado'] = True
        self.c.consultar = consultar
        self.c.sql = ejecutar_sql
        destino = documentos_base.preparar(self.c)
        self.assertTrue(estado['sql'])
        self.assertEqual(self.c.leer(destino)['motivos']['listar']['catalogo_id'],
                         'motivos_documentos_copia_desarrollo')
        self.assertEqual(documentos_base.preparar(self.c), destino)

    def test_motivo_sin_publicacion_cierra(self):
        original = self.c.consultar
        self.c.consultar = lambda sql: '0' if 'motivo_v2_catalogo_publicado' in sql else original(sql)
        with self.assertRaisesRegex(ValueError, 'no publicado'):
            documentos_base.preparar(self.c)
        self.assertFalse((self.c.carpeta / 'documentos/base.json').exists())

    def test_documentos_preparada_del_recorrido_admite_base_propia(self):
        from recorrido_copia.documentos import REUTILIZA
        base = self.c.leer(documentos_base.preparar(self.c))
        puerto = 55441
        dsn = ('postgresql://usuario:secreto@localhost:55441/postgres?'
               'sslmode=verify-full&sslrootcert=' + str(self.c.material / 'ca/ca.crt'))
        entorno = self.c.raiz / 'entorno_modulos.json'
        self.c.guardar(entorno, {v: dsn for v in REUTILIZA.values()})
        class CopiaCompleta:
            nombre = self.c.copia.nombre
            raiz = self.c.raiz
            material = self.c.material
            sql = None
            def leer(self_inner):
                return {'preparada': True, 'recorrido_completo': True,
                        'sid_copia': '12345678901234567', 'puerto_pg': puerto,
                        'entorno_modulos_sha256': self.c.sha(entorno)}
            def existe(self_inner):
                return True
            def sql_bytes(self_inner, raw, nombre):
                self_inner.sql = (raw, nombre)
        copia = CopiaCompleta()
        self.c.copia = copia
        with patch('recorrido_copia.comun.Contexto', Contexto):
            preparar_documentos(copia, base, self.c)
        self.assertIsNotNone(copia.sql)
        self.assertEqual(self.c.leer(self.c.material / 'identidad/documentos.json')['cuentas'], base['cuentas'])

    def test_documentos_preparada_ordinaria_se_deniega(self):
        class CopiaOrdinaria:
            raiz = self.c.raiz
            material = self.c.material
            def leer(self):
                return {'preparada': True, 'recorrido_completo': False}
            def existe(self):
                return True
        with self.assertRaisesRegex(ValueError, 'recorrido completo'):
            preparar_documentos(CopiaOrdinaria(), {'version': 1, 'autoridad': 'no_autoritativo',
                'cuentas': [{'sujeto': 'prueba'}], 'motivos': {'listar': {'catalogo_id': 'x'}}})
        self.assertFalse((self.c.material / 'identidad/documentos.json').exists())

    def test_cuenta_documentos_sin_asignacion_cierra(self):
        original = self.c.consultar
        self.c.consultar = lambda sql: '3' if 'WITH esperado(rol,perfil,persona,cuenta,vinculo,version_rol)' in sql else original(sql)
        with self.assertRaisesRegex(ValueError, 'Asignaciones Documentos'):
            documentos_base.preparar(self.c)
        self.assertFalse((self.c.carpeta / 'documentos/base.json').exists())


if __name__ == '__main__':
    unittest.main()
