"""Orden rol-aspirante: DBA mínimo, actos reanudables y cabeceras de la frontera ADMIN."""

import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

import copia_local_rol_aspirante as rol


class CopiaFalsa:
    def __init__(self, raiz):
        self.nombre = 'vec-copia-l-rolasp'
        self.raiz = raiz
        self.material = raiz / 'material'
        self.sql = []

    def leer(self):
        return {'puerto_pg': 55481}

    def sql_bytes(self, contenido, log):
        self.sql.append(contenido.decode())


def plan_falso():
    plan = dict(paquete_ref='paquete:bolsa:carga_convoca:b1', paquete_version='7', paquete_sha256='a' * 64,
                preparado_en='2026-10-20T10:00:00Z', caduca_en='2026-10-21T08:00:00Z')
    return dict(plan=json.dumps(plan), plan_sha256='b' * 64)


class TestRolAspirante(unittest.TestCase):
    def setUp(self):
        temporal = tempfile.TemporaryDirectory()
        self.addCleanup(temporal.cleanup)
        self.carpeta = Path(temporal.name)
        self.copia = CopiaFalsa(self.carpeta)

    def test_literal_rechaza_lo_que_no_es_una_referencia(self):
        self.assertEqual(rol.literal('catalogo:bolsa:carga_convoca:b1'), "'catalogo:bolsa:carga_convoca:b1'")
        for malo in ("x'; DROP", 'a b', '', 'ñ'):
            with self.assertRaises(ValueError):
                rol.literal(malo)

    def test_dba_solo_crea_login_del_grupo_y_su_aprobacion(self):
        aprobacion = dict(ref='aprobacion:copia-local:rol-aspirante:0123456789abcdef', sha256='c' * 64)
        rol.dba_catalogo(self.copia, self.carpeta, plan_falso(), aprobacion)
        sql = self.copia.sql[0]
        self.assertEqual(sql.count('GRANT '), 1)
        self.assertIn(f'GRANT {rol.GRUPO_CATALOGO} TO ', sql)
        self.assertIn('WITH INHERIT TRUE, SET FALSE, ADMIN FALSE', sql)
        self.assertIn('SET LOCAL ROLE vec_autorizacion_propietario;', sql)
        self.assertIn("'desarrollo','admitida'", sql)
        # Ninguna tabla de roles, concesiones ni asignaciones.
        for tabla in ('version_rol', 'asignacion_perfil', 'control_vigencia'):
            self.assertNotIn(tabla, sql)
        # La contraseña nunca llega al registro de PostgreSQL y el LOGIN caduca con el plan.
        self.assertLess(sql.index("log_min_error_statement='panic'"), sql.index('PASSWORD'))
        self.assertIn("VALID UNTIL '2026-10-21T08:00:00Z'", sql)
        self.assertEqual((self.carpeta / 'catalogo-config.json').stat().st_mode & 0o777, 0o600)
        self.assertIn('sslmode=verify-full', (self.carpeta / 'catalogo-config.json').read_text())
        # Repetir tras un corte no falla por un LOGIN ya creado: se crea o se actualiza.
        self.assertIn('IF EXISTS(SELECT 1 FROM pg_catalog.pg_roles', sql)
        rol.dba_catalogo(self.copia, self.carpeta, plan_falso(), aprobacion)
        self.assertEqual(sql, self.copia.sql[1])
        rol.revocar_aprobacion(self.copia, self.carpeta)
        self.assertIn("SET estado='revocada'", self.copia.sql[2])
        self.assertIn('NOLOGIN', self.copia.sql[2])

    def test_material_releido_con_otra_forma_no_entra_en_el_sql(self):
        (self.carpeta / 'catalogo-acceso.json').write_text(json.dumps(dict(login="x'; DROP", clave='a' * 32)))
        with self.assertRaises(ValueError):
            rol.dba_catalogo(self.copia, self.carpeta, plan_falso(), dict(ref='r', sha256='c' * 64))
        self.assertEqual(self.copia.sql, [])

    def test_acto_ya_publicado_no_envia_nada(self):
        cabeza = dict(ref='rol:x:v3', rol={'concesiones': [{'accion': 'otra'}] * 13 + [
            {'accion': 'bolsa.inscripcion.presentar'}]})
        admin = SimpleNamespace(perfil_aplicacion=lambda: self.fail('no debe seleccionar perfil'),
                                pedir=lambda *a: self.fail('no debe enviar'))
        with patch.object(rol, 'cabeza_rol', return_value=cabeza):
            hecho = rol.acto(self.copia, self.carpeta, None, 'inscripcion', 'bolsa.inscripcion.presentar',
                             12, admin, admin, {})
        self.assertEqual(hecho['ya_publicado'], 'rol:x:v3')

    def test_preferencias_sin_modalidades_no_se_dan_por_hechas(self):
        concesiones = [{'accion': 'otra'}] * 12 + [
            {'accion': 'bolsa.preferencias_bolsas.consultar_propias', 'campos_permitidos': ['version']},
            {'accion': 'bolsa.preferencias_bolsas.ordenar_propias'}]
        with patch.object(rol, 'cabeza_rol', return_value=dict(ref='rol:x:v3', rol={'concesiones': concesiones})):
            with self.assertRaises(ValueError):
                rol.acto(self.copia, self.carpeta, None, 'preferencias',
                         'bolsa.preferencias_bolsas.ordenar_propias', 14, None, None, {})

    def test_acto_reanuda_con_los_mismos_bytes_y_exige_replay(self):
        cuerpo = {'operacion_ref': 'propuesta_admin:' + '1' * 32}
        (self.carpeta / 'acto-preferencias.json').write_text(json.dumps(dict(
            cuerpo=cuerpo, objetivo='rol:x:v3', cierre_ref='cierre_admin:' + '2' * 32)))
        enviados = []

        def pedir(metodo, ruta, datos=None):
            enviados.append((ruta, datos))
            if ruta.endswith('propuestas'):
                replay = len([e for e in enviados if e[0].endswith('propuestas')]) > 1
                return (200 if replay else 201), dict(propuesta_ref=cuerpo['operacion_ref'], huella_sha256='d' * 64,
                                                      version_rol_objetivo_ref='rol:x:v3', replay=replay)
            return 200, dict(recibo_ref='recibo_admin:' + '3' * 32, confirmado_en='2026-10-20T10:00:00Z')

        admin1 = SimpleNamespace(perfil_aplicacion=lambda: None, pedir=pedir)
        admin2 = SimpleNamespace(perfil_aplicacion=lambda: None, pedir=pedir)
        cabeza = dict(ref='rol:x:v3', rol={'concesiones': [{}] * 14})
        with patch.object(rol, 'cabeza_rol', return_value=cabeza):
            hecho = rol.acto(self.copia, self.carpeta, None, 'preferencias', 'marca', 14, admin1, admin2,
                             {'cierre': {'catalogo_id': 'motivos_autorizacion'}})
        self.assertEqual([e[1] for e in enviados[:2]], [cuerpo, cuerpo])
        self.assertEqual(enviados[2][1]['operacion_ref'], 'cierre_admin:' + '2' * 32)
        self.assertEqual(enviados[2][1], enviados[3][1])
        self.assertEqual(hecho['cierre']['recibo_ref'], 'recibo_admin:' + '3' * 32)

    def test_publicar_exige_administracion_en_marcha(self):
        copia = SimpleNamespace(leer=lambda: {'puerto_pg': 1})
        with self.assertRaises(ValueError):
            rol.publicar(copia, repo=Path('/no'), go='go', entorno_go={})

    def test_peticion_lleva_origen_y_cabeceras_de_navegador(self):
        capturado = {}

        class Conexion:
            def __init__(self, *a, **k):
                self.sock = None

            def request(self, metodo, ruta, body=None, headers=None):
                capturado.update(headers)

            def getresponse(self):
                return SimpleNamespace(status=200, read=lambda: b'{}')

            def close(self):
                pass

        admin = rol.Administrador.__new__(rol.Administrador)
        admin.puerto, admin.host = 19481, 'admin.localhost'
        admin.ctx = SimpleNamespace(wrap_socket=lambda *a, **k: None)
        with patch.object(rol.http.client, 'HTTPSConnection', Conexion), \
                patch.object(rol.socket, 'create_connection', lambda *a, **k: None):
            admin.pedir('POST', rol.RUTA_ACTOS + 'cierres', {})
        self.assertEqual(capturado['Origin'], 'https://admin.localhost:19481')
        self.assertEqual(capturado['Host'], 'admin.localhost:19481')
        self.assertEqual((capturado['Sec-Fetch-Site'], capturado['Sec-Fetch-Mode'], capturado['Sec-Fetch-Dest']),
                         ('same-origin', 'cors', 'empty'))
        self.assertFalse(any('cert' in c.lower() or c.lower() == 'cookie' for c in capturado))


if __name__ == '__main__':
    unittest.main()
