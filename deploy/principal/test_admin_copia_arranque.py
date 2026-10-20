"""Contratos del arranque ADMIN, con datos ficticios y sin servicios externos."""

import datetime as dt
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from admin_copia import arranque


class ArranqueAdminTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.root.chmod(0o700)

    def privado(self, nombre, data):
        path = self.root / nombre
        path.write_bytes(data)
        path.chmod(0o600)
        return path

    def config_vigencias(self, dias):
        login = lambda fase: {'login': 'vec_adm_' + fase,
                               'aprobacion_ref': 'aprobacion:' + fase,
                               'conexion': str(self.root / (fase + '-conexion.json')),
                               'aprobacion_externa': str(self.root / (fase + '-aprobacion.json'))}
        return {
            'version': 1, 'entorno': 'desarrollo',
            'alcance_fuente': 'sintetico_declarado', 'destino': 'principal_desarrollo',
            'backup': {'ref': 'backup:ficticio', 'sha256': 'a' * 64},
            'kit': {}, 'privado': str(self.root),
            'psql_comando': ['/usr/bin/psql', '-X', '-v', 'ON_ERROR_STOP=1'],
            'proveedor': {'incluir_cuenta_ordinaria': False},
            'fuentes': {'nombres': ['Irene Robles Vega', 'Daniel Ortega Soria'],
                        'ttl_plan_segundos': 7200, 'ttl_personas_segundos': dias * 86400,
                        'ttl_organizacion_segundos': dias * 86400,
                        'ca_dias': 400, 'maxima_edad_revocacion_segundos': 300},
            'unidad': {'ttl_plan_segundos': 7200, 'vigencia_dias': dias,
                       'clase': 'centro'},
            'bootstrap': {'ttl_plan_segundos': 7200,
                          'ttl_perfiles_segundos': dias * 86400,
                          'duracion_propuesta_segundos': 3600,
                          'app_role_ref': 'rol:aplicacion',
                          'sys_role_ref': 'rol:sistemas'},
            'dba': {'fuentes': {**login('fuentes'), 'proceso': 'proceso:fuentes',
                                 'motivo_ref': 'motivo:fuentes'},
                    'unidad': login('unidad'),
                    'bootstrap': {**login('bootstrap'), 'proceso': 'proceso:bootstrap'}}}

    def test_vigencias_autoridad_entre_60_y_90_dias(self):
        for dias in (60, 90):
            with self.subTest(dias=dias):
                config = self.config_vigencias(dias)
                path = self.privado('config.json', arranque.canon(config))
                cargada = arranque.cargar_config(path)
                self.assertEqual(cargada['fuentes']['ca_dias'], 400)
                self.assertEqual(cargada['fuentes']['ttl_plan_segundos'], 7200)

    def test_vigencias_autoridad_rechazan_corto_excesivo_y_booleanos(self):
        campos = (('fuentes', 'ttl_personas_segundos', 86400),
                  ('fuentes', 'ttl_organizacion_segundos', 86400),
                  ('unidad', 'vigencia_dias', 1),
                  ('bootstrap', 'ttl_perfiles_segundos', 86400))
        for seccion, campo, unidad in campos:
            for valor in (59 * unidad, 400 * unidad, True, False):
                with self.subTest(campo=campo, valor=valor):
                    config = self.config_vigencias(90)
                    config[seccion][campo] = valor
                    path = self.privado('config.json', arranque.canon(config))
                    with self.assertRaisesRegex(
                            arranque.Paro, 'vigencia_autoridad_fuera_de_60_90_dias'):
                        arranque.cargar_config(path)

    def test_aprobacion_externa_exige_plan_y_backup_exactos(self):
        plan = 'a' * 64
        d = {'aprobacion_ref': 'aprobacion:ensayo',
             'aprobacion_externa': str(self.root / 'externa.json')}
        c = {'dba': {'fuentes': d}, 'backup': {'ref': 'backup:ensayo'}}
        documento = {'huella_plan_sha256': plan, 'aprobacion_ref': d['aprobacion_ref'],
                     'backup_ref': c['backup']['ref']}
        raw = arranque.canon(documento)
        self.privado('externa.json', raw)
        self.assertEqual(arranque.aprobacion(c, 'fuentes', plan), arranque.sha(raw))
        for campo, valor in [('huella_plan_sha256', 'b' * 64),
                             ('aprobacion_ref', 'otra'), ('backup_ref', 'otro')]:
            with self.subTest(campo=campo):
                cambiado = {**documento, campo: valor}
                self.privado('externa.json', arranque.canon(cambiado))
                with self.assertRaisesRegex(arranque.Paro, 'aprobacion_externa_divergente'):
                    arranque.aprobacion(c, 'fuentes', plan)

    def test_preimagen_ya_comprometida_no_se_sustituye(self):
        fase = self.root / 'fuentes'
        fase.mkdir(mode=0o700)
        material = b'{"personas":[]}'
        self.privado('fuentes/material-hmac.raw', material)
        plan = {'caduca_en': arranque.fmt(arranque.now() + dt.timedelta(hours=1)),
                'fuente_hmac': {'huella_sha256': arranque.sha(material)}}
        self.privado('fuentes/plan.json', arranque.canon({
            'plan': plan, 'huella_plan_sha256': arranque.sha(arranque.canon(plan))}))
        self.privado('fuentes/preimagen.sha256', ('b' * 64 + '\n').encode())
        c = {'privado': str(self.root), 'dba': {'fuentes': {
            'login': 'vec_admin_ensayo', 'proceso': 'ensayo', 'motivo_ref': 'motivo:ensayo',
            'aprobacion_ref': 'aprobacion:ensayo'}}}
        with patch.object(arranque, 'aprobacion', return_value='c' * 64), \
             patch.object(arranque, 'getter_hash', return_value='a' * 64), \
             patch.object(arranque, 'configurar_login') as configurar:
            with self.assertRaisesRegex(arranque.Paro, 'artefacto_divergente'):
                arranque.configurar_fuentes(c)
            configurar.assert_not_called()

    def test_recuperar_reusa_plan_y_exige_replay_del_mismo_recibo(self):
        fase = self.root / 'fuentes'
        fase.mkdir(mode=0o700)
        plan = {'caduca_en': arranque.fmt(arranque.now() + dt.timedelta(hours=1)),
                'organizacion': {'organizacion_ref': 'org_ficticia'},
                'personas': [{'persona_ref': 'per_1'}, {'persona_ref': 'per_2'}]}
        huella = arranque.sha(arranque.canon(plan))
        self.privado('fuentes/plan.json', arranque.canon({'plan': plan,
                                                          'huella_plan_sha256': huella}))
        recibo = {'auditoria_ref': 'aud_1', 'plan_sha256': huella, 'recibo_ref': 'rec_1',
                  'ca': {'plan_sha256': huella, 'datos': {
                      'organizacion_ref': 'org_ficticia', 'personas': plan['personas']}},
                  'is': {'plan_sha256': huella}}
        primero = {'estado': 'permitido', 'recibo': recibo,
                   'auditoria_intento': {'ref': 'intento_1'}}
        self.privado('fuentes/acuse-primero.json', arranque.canon(primero))
        c = {'privado': str(self.root), 'kit': {'cli': {'fuentes_aplicar': '/bin/false'},
             'catalogos': {'fuentes_aplicar': '/catalogo/ficticio'}}}
        llamadas = []

        def cli_ficticia(_c, name, args, acuse):
            llamadas.append((name, args, acuse))
            repetido = {**primero, 'replay': True,
                        'auditoria_intento': {'ref': 'intento_2'}}
            self.privado(str(Path(acuse).relative_to(self.root)), arranque.canon(repetido))
            return repetido

        with patch.object(arranque, 'exigir_config_confirmada'), \
             patch.object(arranque, 'cli', side_effect=cli_ficticia), \
             patch.object(arranque, 'estado'):
            arranque.aplicar_fase(c, 'fuentes', replay=True)
        self.assertEqual(len(llamadas), 1)
        self.assertEqual(llamadas[0][0], 'fuentes_aplicar')
        self.assertIn(fase / 'plan.json', llamadas[0][1])
        self.assertNotEqual(llamadas[0][2], fase / 'acuse-primero.json')
        self.assertEqual(arranque.confirmacion_vigente(fase, 'fuentes')[1]['recibo'], recibo)

    def test_recuperar_rechaza_acto_nuevo(self):
        fase = self.root / 'fuentes'
        fase.mkdir(mode=0o700)
        plan = {'caduca_en': arranque.fmt(arranque.now() + dt.timedelta(hours=1)),
                'organizacion': {'organizacion_ref': 'org_ficticia'},
                'personas': [{'persona_ref': 'per_1'}, {'persona_ref': 'per_2'}]}
        huella = arranque.sha(arranque.canon(plan))
        self.privado('fuentes/plan.json', arranque.canon({'plan': plan,
                                                          'huella_plan_sha256': huella}))
        recibo = {'auditoria_ref': 'aud_1', 'plan_sha256': huella,
                  'ca': {'plan_sha256': huella, 'datos': {
                      'organizacion_ref': 'org_ficticia', 'personas': plan['personas']}},
                  'is': {'plan_sha256': huella}}
        self.privado('fuentes/acuse-primero.json', arranque.canon({
            'estado': 'permitido', 'recibo': recibo,
            'auditoria_intento': {'ref': 'intento_1'}}))
        c = {'privado': str(self.root), 'kit': {'catalogos': {
            'fuentes_aplicar': '/catalogo/ficticio'}}}
        with patch.object(arranque, 'exigir_config_confirmada'), \
             patch.object(arranque, 'cli', return_value={
                 'estado': 'permitido', 'recibo': recibo,
                 'auditoria_intento': {'ref': 'intento_2'}}):
            with self.assertRaisesRegex(arranque.Paro, 'recuperacion_no_replay'):
                arranque.aplicar_fase(c, 'fuentes', replay=True)


if __name__ == '__main__':
    unittest.main()
