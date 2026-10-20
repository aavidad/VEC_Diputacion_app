"""Pruebas sin SQL, red ni certificados reales del montaje ADMIN."""

import hashlib
import importlib.util
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import Mock, patch
from datetime import datetime, timezone

from recorrido_copia import montajes_firma as m


class Contexto:
    def __init__(self, raiz):
        self.raiz = raiz
        self.carpeta = raiz / 'recorrido'
        self.carpeta.mkdir(mode=0o700)
        self.material = raiz / 'material'
        (self.material / 'ca').mkdir(parents=True, mode=0o700)
        self.copia = Mock()
        self.copia.nombre = 'vec-copia-l-prueba'
        self.copia.leer.return_value = {'sid_copia': '123456789012345',
                                        'puerto_pg': 55441, 'admin_preparada': True,
                                        'recorrido_completo': True}
        self.sqls = []

    @staticmethod
    def guardar(ruta, dato):
        ruta.parent.mkdir(parents=True, mode=0o700, exist_ok=True)
        raw = dato if isinstance(dato, bytes) else (json.dumps(dato, ensure_ascii=False) + '\n').encode()
        fd = os.open(ruta, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, 'wb') as salida:
            salida.write(raw)

    @staticmethod
    def leer(ruta):
        return json.loads(ruta.read_bytes())

    @staticmethod
    def ahora():
        return datetime(2026, 10, 20, tzinfo=timezone.utc)

    def consultar(self, _sql):
        return '123456789012345'

    def sql(self, dato, log):
        self.sqls.append((dato, log))


class MontajesTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.c = Contexto(Path(self.temp.name))
        self.run = self.c.raiz / 'admin/h10'
        (self.run / 'vec-admin/pools').mkdir(parents=True, mode=0o700)
        (self.run / 'vec-admin/config').mkdir(mode=0o700)
        self.c.guardar(self.c.material / 'ca/ca.crt', b'CA ficticia')

    def test_sql_liga_cuatro_pools_al_sid_antes_de_crear_rol(self):
        entrada = {'procesos': {'usuarios': 'vec-admin-usuarios'}}
        _, logins = m.preparar_sql(self.c, self.c.copia.leer(), self.run,
                                  '20261020f131', entrada)
        self.assertEqual(set(logins), set(m.AUDIENCIAS))
        self.assertEqual(len(self.c.sqls), 1)
        sql = self.c.sqls[0][0].decode()
        self.assertLess(sql.index('DO $sid$'), sql.index('CREATE ROLE'))
        self.assertEqual(sql.count('CREATE ROLE'), 4)
        self.assertEqual(sql.count('configuracion_origen_consumos_v1'), 8)
        self.assertIn('123456789012345', sql)
        for nombre in m.AUDIENCIAS:
            ruta = self.run / 'vec-admin/pools' / (nombre + '.json')
            self.assertEqual(ruta.stat().st_mode & 0o777, 0o600)
            self.assertIn('host=localhost port=55441 dbname=postgres', json.loads(ruta.read_text())['dsn'])

    def test_preflight_detiene_sid_ajeno_antes_de_montar(self):
        self.c.consultar = Mock(return_value='123456789012346')
        with self.assertRaisesRegex(ValueError, 'SID'):
            m.preflight(self.c)
        self.assertFalse((self.c.carpeta / 'montajes_firma').exists())

    def test_configs_separan_lote_plan_cargos_y_certificados(self):
        dia = '20261020f131'
        gov = self.run / 'vec-admin/claves' / dia
        gov.mkdir(parents=True, mode=0o700)
        material = {'claves': []}
        for i, audiencia in enumerate(m.AUDIENCIAS.values()):
            secreto = bytes([i + 1]) * 32
            self.c.guardar(gov / (audiencia + '.bin'), secreto)
            material['claves'].append({'audiencia': audiencia, 'clave_id': 'clave:ficticia:' + str(i),
                'emisor_id': 'emisor:ficticio:' + str(i), 'huella_gobierno_sha256': 'a' * 64,
                'version': 1, 'revision_gobierno': 1,
                'huella_secreto_sha256': hashlib.sha256(secreto).hexdigest(),
                'valida_desde': '2026-10-20T00:00:00Z', 'valida_hasta': '2027-01-18T00:00:00Z'})
        raiz = {'clave_id': 'clave:raiz', 'publica_base64': 'cHVibGljYQ=='}
        usuarios = {'organizacion_ref': 'org_ficticia', 'unidad_ref': 'unidad:ficticia',
                    'confianza': {'cabecera': {'formato_version': 3}, 'raiz': raiz,
                                  'gobierno': {'revision': 'rev-ficticia'}}, 'destinos': {}}
        cm = {'Raiz': {'ClaveID': raiz['clave_id'], 'Publica': raiz['publica_base64']},
              'Gobierno': {'Revision': 'rev-ficticia'}}
        plan = {'plan': {'personas': [{'ambitos': [
            {'dimension': 'organizacion_ref', 'valores': ['org_ficticia'], 'fuente': 'fuente:org'},
            {'dimension': 'unidad_ref', 'valores': ['unidad:ficticia'], 'fuente': 'fuente:unidad'}]}]}}
        self.c.guardar(self.run / 'bootstrap/plan-v3.json', plan)
        motivo = {'catalogo_id': 'motivos', 'catalogo_version': 1,
                  'catalogo_huella_sha256': 'b' * 64, 'entrada_clave': 'motivo_ficticio'}
        resultado = m.preparar_configuraciones(self.c, self.run, dia, {}, material, cm,
                                               usuarios, motivo, {'gobierno': ({}, 'c' * 64)})
        self.assertEqual(set(resultado['archivos_sha256']), set(m.ENTORNO))
        config = self.run / 'vec-admin/config'
        rutas = [json.loads((config / m.ENTORNO[k]).read_text())['confianza']['entradas_capacidad'][0]['material_archivo']
                 for k in m.ENTORNO if k != 'VEC_ADMIN_USUARIOS_CONFIG_FILE']
        self.assertEqual(len(set(rutas)), 4)
        self.assertTrue(all(Path(p).is_file() for p in rutas))


if __name__ == '__main__':
    unittest.main()
