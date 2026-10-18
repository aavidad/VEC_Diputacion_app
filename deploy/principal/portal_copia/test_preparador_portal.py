import fcntl
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

FUENTE = Path(__file__).with_name('preparar_portal.py')
ESPEC = importlib.util.spec_from_file_location('portal_fabrica', FUENTE)
with tempfile.TemporaryDirectory() as directorio_importacion:
    with mock.patch.dict(os.environ, {'VEC_COPIA_PORTAL_ESTADO': directorio_importacion}):
        portal = importlib.util.module_from_spec(ESPEC)
        ESPEC.loader.exec_module(portal)

CLASIFICACION = Path(__file__).with_name('preparar_clasificacion_externa.py')
ESPEC_CLAS = importlib.util.spec_from_file_location('clasificacion_fabrica', CLASIFICACION)
clasificacion = importlib.util.module_from_spec(ESPEC_CLAS)
ESPEC_CLAS.loader.exec_module(clasificacion)


class PortalFabricaTest(unittest.TestCase):
    def test_candado_heredado_es_misma_descripcion_y_standalone_excluye(self):
        with tempfile.TemporaryDirectory() as temporal:
            base = Path(temporal)
            raiz = base / 'equipo-instancia'
            raiz.mkdir(mode=0o700)
            ruta = base / '.vec-copia-equipo-instancia.lock'
            with mock.patch.object(portal, 'BASE', base):
                with ruta.open('a+') as abierto:
                    fcntl.flock(abierto, fcntl.LOCK_EX | fcntl.LOCK_NB)
                    with portal.candado_copia(raiz, abierto.fileno()):
                        with self.assertRaisesRegex(RuntimeError, 'copia_ocupada'):
                            with portal.candado_copia(raiz):
                                pass
                with portal.candado_copia(raiz):
                    pass

    def test_sid_y_puerto_se_fijan_antes_del_sql_y_cambio_falla(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            copia = mock.Mock(raiz=raiz)
            copia.existe.return_value = True
            estado = {'nombre': 'vec-copia-equipo-instancia', 'puerto_pg': 55441}
            control = raiz / 'control-sid.json'
            inspeccion = {'Config': {'Labels': {'vec.copia': str(raiz)}},
                          'NetworkSettings': {'Ports': {'5432/tcp': [{'HostIp': '127.0.0.1', 'HostPort': '55441'}]}},
                          'State': {'Running': True}}
            with mock.patch.object(portal.subprocess, 'run', side_effect=[
                mock.Mock(stdout=json.dumps([inspeccion]).encode()),
                mock.Mock(stdout=b'1234567890123456789\n'),
                mock.Mock(stdout=json.dumps([inspeccion]).encode()),
                mock.Mock(stdout=b'9876543210987654321\n'),
            ]):
                self.assertEqual(portal.verificar_copia(copia, estado, control), '1234567890123456789')
                with self.assertRaisesRegex(RuntimeError, 'fuente_privada_divergente'):
                    portal.verificar_copia(copia, estado, control)
            self.assertEqual(json.loads(control.read_text())['sid'], '1234567890123456789')
            self.assertEqual(control.stat().st_mode & 0o777, 0o600)

    def test_commit_perdido_recupera_el_mismo_documento_y_acto(self):
        documento = {'asignacion_id': 'asg_' + 'a' * 32, 'perfil_activo_ref': 'prf_' + 'b' * 32,
                     'principal_id': 'per_' + 'c' * 32, 'version_rol_ref': 'rol:candidato_bolsa_portal_historial_propio_desarrollo:v1',
                     'ambitos': [{'clave': 'candidato_ref', 'valores': ['can_' + 'd' * 32]}]}
        canon = json.dumps(documento, separators=(',', ':'))
        intento = {'documento': canon, 'huella_sha256': hashlib.sha256(canon.encode()).hexdigest(),
                   'asignacion_ref': 'asignacion:asg_' + 'a' * 32 + ':v1', 'acto_ref': 'act_' + 'e' * 32,
                   'perfil_ref': documento['perfil_activo_ref'], 'principal_id': documento['principal_id'],
                   'rol_ref': documento['version_rol_ref'], 'candidato_ref': 'can_' + 'd' * 32}
        actual = {'asignacion_ref': intento['asignacion_ref'], 'huella_sha256': intento['huella_sha256'],
                  'documento': documento, 'acto_ref': intento['acto_ref'], 'perfil_ref': intento['perfil_ref'],
                  'principal_id': intento['principal_id'], 'rol_ref': intento['rol_ref'],
                  'evento': {'objeto_ref': intento['asignacion_ref'], 'huella_nueva': intento['huella_sha256'], 'version': 1}}
        self.assertIsNone(portal.confirmar_asignacion_retenida(actual, intento, False))
        ajena = dict(actual, acto_ref='act_' + 'f' * 32)
        with self.assertRaisesRegex(RuntimeError, 'asignacion_ajena_no_repetir'):
            portal.confirmar_asignacion_retenida(ajena, intento, False)
        with self.assertRaisesRegex(RuntimeError, 'reconciliar'):
            portal.confirmar_asignacion_retenida(None, intento, False)

    def test_clasificacion_no_publica_selector_aunque_hijo_se_interrumpa(self):
        entorno = portal.entorno_portal({'VEC_PORTAL_PROCESO': 'interno'}, {}, '/material', True)
        self.assertNotIn('VEC_BOLSA_INSCRIPCIONES_ENABLED', entorno)
        self.assertEqual(portal.entorno_portal({}, {}, '/material', False)['VEC_BOLSA_INSCRIPCIONES_ENABLED'], 'true')
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            archivo = raiz / 'entorno_modulos.json'
            archivo.write_text(json.dumps(entorno))
            antes = archivo.read_bytes()
            with (mock.patch.object(sys, 'argv', ['clasificacion', '--raiz', str(raiz), '--candado-fd', '7']),
                  mock.patch.object(clasificacion.subprocess, 'run', side_effect=subprocess.CalledProcessError(1, 'hijo')) as hijo):
                with self.assertRaises(subprocess.CalledProcessError):
                    clasificacion.main()
            self.assertEqual(archivo.read_bytes(), antes)
        self.assertIn('--sin-inscripcion', hijo.call_args.args[0])
        self.assertEqual(hijo.call_args.kwargs['pass_fds'], (7,))
        self.assertNotIn('VEC_BOLSA_INSCRIPCIONES_ENABLED', entorno)


if __name__ == '__main__':
    unittest.main()
