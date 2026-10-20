"""Focales sin PostgreSQL ni HTTP de los roles ADMIN de la copia."""

import hashlib
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import Mock, patch
from datetime import datetime, timezone

sys.path.insert(0, str(Path(__file__).resolve().parent))
from recorrido_copia import roles_admin as roles


class ContextoFalso:
    def __init__(self, raiz):
        self.raiz = raiz
        self.carpeta = raiz / 'recorrido'
        self.carpeta.mkdir(mode=0o700)
        self.material = raiz / 'material'
        self.material.mkdir(mode=0o700)
        self.copia = Mock(nombre='vec-copia-l-prueba')
        self.copia.nombre = 'vec-copia-l-prueba'
        self.copia.leer.return_value = {'sid_copia': '123456789012345', 'puerto_pg': 55441}
        self.aplicado = []

    @staticmethod
    def guardar(ruta, dato):
        ruta.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
        raw = dato if isinstance(dato, bytes) else (json.dumps(dato) + '\n').encode()
        fd = os.open(ruta, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(fd, 'wb') as salida:
            salida.write(raw)

    @staticmethod
    def leer(ruta):
        return json.loads(ruta.read_bytes())

    @staticmethod
    def sha(ruta):
        return hashlib.sha256(ruta.read_bytes()).hexdigest()

    @staticmethod
    def ahora():
        return datetime(2026, 10, 20, tzinfo=timezone.utc)

    def sql(self, datos, log):
        self.aplicado.append((datos, log))


class RolesAdminTest(unittest.TestCase):
    def setUp(self):
        self.temporal = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporal.cleanup)
        self.c = ContextoFalso(Path(self.temporal.name))

    def test_exige_dos_aut58_con_replay_y_mismo_sid(self):
        base = self.c.carpeta / 'catalogo'
        actos = []
        for nombre in ('organizacion', 'origen_ct'):
            ruta = base / nombre
            ruta.mkdir(parents=True, mode=0o700)
            recibo = {'catalogo_sha256': 'a' * 64, 'auditoria_ref': nombre}
            primero = {'recibo': recibo}
            replay = {'recibo': recibo, 'replay': True}
            self.c.guardar(ruta / 'acuse-primero.json', primero)
            self.c.guardar(ruta / 'acuse-replay.json', replay)
            actos.append({'nombre': nombre, 'sid_copia': '123456789012345',
                          'catalogo_sha256': 'a' * 64,
                          'acuse_primero_sha256': self.c.sha(ruta / 'acuse-primero.json'),
                          'acuse_replay_sha256': self.c.sha(ruta / 'acuse-replay.json')})
        meta = {'version': 1, 'copia_ref': self.c.copia.nombre,
                'sid_copia': '123456789012345', 'actos': actos,
                'recibos_reales': 2, 'replays_cotejados': 2}
        self.c.guardar(base / 'admisiones.json', meta)
        self.assertEqual(roles._admisiones(self.c)['nombre'], 'origen_ct')
        replay = base / 'origen_ct/acuse-replay.json'
        replay.write_text(json.dumps({'recibo': {'catalogo_sha256': 'b' * 64}, 'replay': True}))
        with self.assertRaisesRegex(ValueError, 'Acuse AUT58'):
            roles._admisiones(self.c)

    def test_pools_guardan_sid_antes_de_crear_login(self):
        run = self.c.raiz / 'admin/h10'
        (run / 'vec-admin/pools').mkdir(parents=True, mode=0o700)
        (self.c.material / 'ca').mkdir()
        ca = self.c.material / 'ca/ca.crt'
        self.c.guardar(ca, b'CA ficticia')
        self.c.guardar(run / 'entrada-vec-admin.json', {
            'sid_copia': '123456789012345', 'dia_gobierno': '20261020f131',
            'runner': str(run), 'procesos': {'usuarios': 'vec-admin-usuarios'}})
        roles._sql_pools(self.c, run, '20261020f131', '123456789012345')
        self.assertEqual(len(self.c.aplicado), 1)
        sql = self.c.aplicado[0][0].decode()
        self.assertLess(sql.index('DO $sid$'), sql.index('CREATE ROLE'))
        self.assertIn('pg_catalog.pg_control_system()', sql)
        self.assertIn('123456789012345', sql)
        self.assertEqual(sql.count('CREATE ROLE'), 4)
        self.assertEqual(sql.count('configuracion_origen_consumos_v1'), 4)

    def test_overlay_rechazado_restaura_admin_previo(self):
        import copia_local_administracion as admin
        preparacion = self.c.raiz / 'admin_preparacion.json'
        self.c.guardar(preparacion, {'VEC_ADMIN_ESTADO': 'preparado_con_acuses'})
        b103 = self.c.raiz / 'b103.json'
        gobierno = self.c.raiz / 'gobierno.json'
        for ruta in (b103, gobierno):
            self.c.guardar(ruta, {})
        pasos = []

        def arrancar(_copia, fase):
            pasos.append(fase)
            if len(pasos) == 1:
                raise ValueError('overlay rechazado')

        with patch.object(admin, 'parar') as parar, \
                patch.object(admin, 'ejecutar', side_effect=arrancar):
            with self.assertRaisesRegex(ValueError, 'Administración recuperada'):
                roles._activar(self.c, b103, gobierno)
        self.assertEqual(parar.call_count, 2)
        self.assertEqual(pasos, ['arrancar', 'arrancar', 'comprobar'])
        self.assertNotIn('admin_extra_env_sha256', json.loads(preparacion.read_text()))
        self.assertFalse((self.c.raiz / 'admin_extra_env.json').exists())
        self.assertTrue((self.c.carpeta / 'admin-extra-fallido.json').exists())


if __name__ == '__main__':
    unittest.main()


class DestinoBolsa(unittest.TestCase):
    def test_no_declara_exito_si_la_cuenta_nueva_conserva_la_version_anterior(self):
        from types import SimpleNamespace
        from unittest.mock import Mock, patch
        from recorrido_copia import roles_admin as modulo
        perfil = 'prf_' + 'a' * 32
        c = SimpleNamespace(material=Path('/tmp/propio'), consultar=Mock(return_value='0'))
        with patch.object(modulo, '_documento', return_value={'perfil_ref': perfil}):
            with self.assertRaisesRegex(ValueError, 'no recibió la versión gobernada'):
                modulo.cotejar_destino_bolsa(c, 'rol:tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo:v11')
            c.consultar.return_value = '1'
            resultado = modulo.cotejar_destino_bolsa(c, 'rol:tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo:v11')
        self.assertEqual(resultado['perfil_ref'], perfil)
