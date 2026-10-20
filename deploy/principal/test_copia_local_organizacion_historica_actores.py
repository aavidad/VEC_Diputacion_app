"""Pruebas focales de la segunda cuenta OH, sin servicios ni material real."""

import json
import hashlib
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parent))
import copia_local_organizacion_historica_actores as actores
REPO = Path(__file__).parent.parent.parent


ORG = 'org_' + 'a' * 32
CUENTA = 'cta_' + 'b' * 32
PERSONA = 'per_' + 'c' * 32
PERFIL_CT = 'prf_' + 'd' * 32
CONTEXTO = {'cuenta_ref': CUENTA, 'persona_ref': PERSONA, 'perfil_ref': PERFIL_CT,
            'procedencia_ref': 'prc_' + 'e' * 32, 'procedencia_version': 1,
            'procedencia_huella_sha256': 'f' * 64, 'procedencia_autoridad': 'no_autoritativa',
            'vigente_desde': '2026-10-01T00:00:00Z', 'vigente_hasta': '2027-01-01T00:00:00Z'}
CORPORATIVO = {**{k: CONTEXTO[k] for k in ('cuenta_ref', 'persona_ref')},
               'cuenta_version': 1, 'persona_version': 1, 'organizacion_ref': ORG,
               'organizacion_version': 1, 'organizacion_procedencia_ref': 'prc_' + '1' * 32,
               'organizacion_procedencia_version': 1,
               'organizacion_procedencia_huella_sha256': '2' * 64,
               'organizacion_procedencia_autoridad': 'autoridad_maestra_acreditada',
               'superficie': 'interna_corporativa', 'uso': 'consulta_rrhh',
               'procedencia_ref': CONTEXTO['procedencia_ref'], 'procedencia_version': 1,
               'procedencia_huella_sha256': CONTEXTO['procedencia_huella_sha256'],
               'procedencia_autoridad': 'no_autoritativa',
               'vigente_desde': CONTEXTO['vigente_desde'], 'vigente_hasta': CONTEXTO['vigente_hasta']}
REFS = {'perfil': 'prf_' + '3' * 32, 'vinculo_contexto': 'vca_' + '4' * 32,
        'vinculo_corporativo': 'vcr_' + '5' * 32, 'asignacion': 'asg_' + '6' * 32}


class Copia:
    def __init__(self, raiz):
        self.raiz = raiz
        self.material = raiz / 'material'
        self.material.mkdir(mode=0o700)
        (self.material / 'identidad').mkdir(mode=0o700)
        (raiz / 'organizacion-historica/entrada').mkdir(parents=True, mode=0o700)
        self.sql = []

    def leer(self):
        return {'organizacion_historica_semilla_sha256': 'a' * 64,
                'rrhh_adicional_semilla_sha256': 'b' * 64}

    def sql_bytes(self, contenido, log):
        self.sql.append((contenido.decode(), log))


def privado(ruta, dato):
    ruta.write_text(json.dumps(dato) + '\n')
    ruta.chmod(0o600)


class TestSegundaOH(unittest.TestCase):
    def test_semilla_original_no_acredita_rol_sin_cierre(self):
        with tempfile.TemporaryDirectory() as t:
            copia = Copia(Path(t))
            with patch.object(actores.oh, '_consulta', return_value=['ausente']):
                with self.assertRaisesRegex(ValueError, 'cierre AUT60'):
                    actores.extender_contextos(copia, REPO)
            self.assertEqual(copia.sql, [])

    def test_sql_perfil_propio_sin_republicar_rol_ni_motivos(self):
        sql = actores.sql_segunda(CONTEXTO, CORPORATIVO, REFS, ORG, REPO)
        self.assertEqual(sql.count('BEGIN ISOLATION LEVEL SERIALIZABLE'), 1)
        self.assertEqual(sql.count('COMMIT;'), 1)
        self.assertIn("'" + CUENTA + "'", sql)
        self.assertIn("'" + PERSONA + "'", sql)
        self.assertIn('"clave":"unidad_clave","valores":["unidad:centro:520"]', sql)
        self.assertIn('canon_asignacion_perfil_admin_v1', sql)
        self.assertNotIn('INSERT INTO vec_autorizacion.version_rol', sql)
        self.assertIn('cierre_gobierno_rol_nuevo_v1', sql)
        self.assertIn('c.aprobador_persona_ref<>p.proponente_persona_ref', sql)
        self.assertNotIn('publicar_motivos_autorizacion_v2', sql)

    def test_organizacion_original_no_presta_identidad_a_segundo(self):
        with tempfile.TemporaryDirectory() as t:
            copia = Copia(Path(t))
            original = {'cuenta_ref': 'cta_' + '0' * 32, 'persona_ref': 'per_' + '0' * 32,
                        'perfil_ref': 'prf_' + '0' * 32, 'organismo_ref': ORG,
                        'gobierno_real_acreditado': False}
            privado(copia.raiz / actores.oh.SEMILLA, original)
            plantilla = {**CORPORATIVO, 'cuenta_ref': original['cuenta_ref'],
                         'persona_ref': original['persona_ref'], 'cuenta_version': 7,
                         'persona_version': 8, 'procedencia_autoridad': 'autoridad_maestra_acreditada'}
            fila = json.dumps({'contexto': CONTEXTO, 'corporativo': plantilla,
                               'cuenta_version': 1, 'persona_version': 1})
            with patch.object(actores.oh, '_consulta', return_value=[fila]):
                contexto, propio = actores._contexto_f1(copia,
                    {'cuenta_ref': CUENTA, 'persona_ref': PERSONA, 'perfil_ref': PERFIL_CT}, ORG)
            self.assertEqual(contexto['cuenta_ref'], CUENTA)
            self.assertEqual((propio['cuenta_ref'], propio['persona_ref']), (CUENTA, PERSONA))
            self.assertEqual((propio['cuenta_version'], propio['persona_version']), (1, 1))
            self.assertEqual(propio['organizacion_ref'], ORG)
            self.assertEqual(propio['organizacion_procedencia_ref'], plantilla['organizacion_procedencia_ref'])
            self.assertEqual(propio['procedencia_autoridad'], CONTEXTO['procedencia_autoridad'])

    def test_otra_cuenta_se_rechaza_antes_de_sql(self):
        with self.assertRaisesRegex(ValueError, 'segunda cuenta'):
            actores.sql_segunda(CONTEXTO, {**CORPORATIVO, 'cuenta_ref': 'cta_' + '9' * 32}, REFS, ORG, REPO)

    def test_sin_proyeccion_f1_no_cambia_entrada(self):
        with tempfile.TemporaryDirectory() as t:
            copia = Copia(Path(t))
            privado(copia.material / 'identidad/tecnico-rrhh-adicional.json',
                    {'version': 1, 'autoridad': 'no_autoritativo',
                     'identidad': 'identidad/tecnico-rrhh-adicional-identidad.json',
                     'certificado': 'mtls/tecnico-rrhh-adicional.crt',
                     'sujeto': 'desarrollo:rrhh:adicional:1', 'cuenta_ref': CUENTA,
                     'persona_ref': PERSONA, 'perfil_ref': PERFIL_CT})
            privado(copia.material / 'identidad/tecnico-rrhh-adicional-identidad.json',
                    {'subject': 'desarrollo:rrhh:adicional:1', 'roles': ['tecnico_rrhh'],
                     'certificate_sha256': hashlib.sha256(b'der-sintetico').hexdigest()})
            certificado = copia.material / 'mtls/tecnico-rrhh-adicional.crt'
            certificado.parent.mkdir(mode=0o700)
            certificado.write_bytes(b'certificado-sintetico')
            certificado.chmod(0o600)
            privado(copia.raiz / 'rrhh-adicional-semilla.json',
                    {'persona_ref': PERSONA, 'perfil_ref': PERFIL_CT,
                     'gobierno_real_acreditado': False,
                     'rol_ref': 'rol:seguimiento_cese_ct_desarrollo:v2',
                     'organizacion_ref': 'organizacion:desarrollo:dipgra'})
            privado(copia.raiz / actores.oh.SEMILLA,
                    {'cuenta_ref': 'cta_' + '0' * 32, 'persona_ref': 'per_' + '0' * 32,
                     'perfil_ref': 'prf_' + '0' * 32, 'organismo_ref': ORG,
                     'gobierno_real_acreditado': False})
            entrada = copia.raiz / 'organizacion-historica/entrada/organizacion_historica_v3.json'
            privado(entrada, {'contextos': {'cta_' + '0' * 32: {}}})
            with patch.object(actores.oh, 'comprobar_rol_publicado'), \
                    patch.object(actores.oh, 'organismo', return_value=ORG), \
                    patch.object(actores.oh, '_consulta', return_value=[]), \
                    patch.object(actores.subprocess, 'run') as openssl:
                openssl.return_value.stdout = b'der-sintetico'
                with self.assertRaisesRegex(ValueError, 'proyección F1'):
                    actores.extender_contextos(copia, Path(t))
            self.assertEqual(len(json.loads(entrada.read_text())['contextos']), 1)
            self.assertEqual(copia.sql, [])

    def test_extiende_dos_cuentas_con_perfil_distinto(self):
        with tempfile.TemporaryDirectory() as t:
            copia = Copia(Path(t))
            entrada = copia.raiz / 'organizacion-historica/entrada/organizacion_historica_v3.json'
            privado(entrada, {'version': 1, 'contextos': {'cta_' + '0' * 32: {
                'perfil_activo_ref': 'prf_' + '0' * 32, 'perfil_version': 1,
                'organismo_ref': ORG, 'unidad_clave': ''}}})
            segundo = {'cuenta_ref': CUENTA, 'persona_ref': PERSONA}
            with patch.object(actores.oh, 'comprobar_rol_publicado'), \
                    patch.object(actores.oh, 'organismo', return_value=ORG), \
                    patch.object(actores, '_segundo', return_value=segundo), \
                    patch.object(actores, '_contexto_f1', return_value=(CONTEXTO, CORPORATIVO)):
                huella = actores.extender_contextos(copia, REPO)
            contextos = json.loads(entrada.read_text())['contextos']
            self.assertEqual(len(contextos), 2)
            self.assertNotEqual(contextos[CUENTA]['perfil_activo_ref'],
                                contextos['cta_' + '0' * 32]['perfil_activo_ref'])
            self.assertEqual(contextos[CUENTA]['organismo_ref'], ORG)
            self.assertEqual(copia.sql[0][1], 'organizacion-historica-segunda-semilla.log')
            self.assertEqual(len(huella), 64)


if __name__ == '__main__':
    unittest.main()
