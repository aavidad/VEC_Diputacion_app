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

    def test_base_propia_con_cuatro_cuentas_y_custodia(self):
        destino = documentos_base.preparar(self.c)
        base = self.c.leer(destino)
        self.assertEqual(len(base['cuentas']), 4)
        self.assertEqual(set(base['custodia_firmado']['documentos']), {'informe_definitivo', 'resolucion'})
        self.assertEqual(documentos_base.preparar(self.c), destino)

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
