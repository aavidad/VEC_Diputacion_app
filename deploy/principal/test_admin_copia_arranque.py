"""Contratos del arranque ADMIN, con datos ficticios y sin servicios externos."""

import datetime as dt
import json
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

    def escenario_bootstrap(self, dias):
        inicio = dt.datetime(2026, 10, 20, tzinfo=dt.timezone.utc)
        for nombre in ('fuentes', 'unidad'):
            (self.root / nombre).mkdir(mode=0o700)
        refs = ('per_1', 'per_2')
        vencimiento = arranque.fmt(inicio + dt.timedelta(days=dias))
        fuente_plan = {
            'organizacion': {'organizacion_ref': 'org_ficticia',
                             'vigente_hasta': vencimiento},
            'personas': [{'persona_ref': ref, 'vigente_hasta': vencimiento}
                         for ref in refs],
            'politica_admin': {'politica_ref': 'pga_ficticia',
                               'vigente_hasta': vencimiento}}
        huella = arranque.sha(arranque.canon(fuente_plan))
        self.privado('fuentes/plan.json', arranque.canon({
            'plan': fuente_plan, 'huella_plan_sha256': huella}))
        ca = b'ca ficticia'
        self.privado('fuentes/ca.der', ca)
        self.privado('fuentes/fuente.json', arranque.canon({
            'procedencia': {'referencia': 'prc_ficticia', 'version': 1,
                            'huella_sha256': 'a' * 64},
            'politica_admin': {'politica_ref': 'pga_ficticia',
                               'ca_sha256': arranque.sha(ca)}}))
        certificados = []
        personas = []
        for i, ref in enumerate(refs, 1):
            der = ('certificado ficticio ' + ref).encode()
            self.privado(f'fuentes/cliente-{i}.der', der)
            certificados.append({'persona_ref': ref,
                                 'cert_der_sha256': arranque.sha(der),
                                 'ca_der_sha256': arranque.sha(ca)})
            personas.append({'persona_ref': ref,
                             'cuenta_privilegiada_ref': f'cta_{i}',
                             'proyeccion_cuenta_version': 1,
                             'persona_version': 1})
        self.privado('fuentes/certificados.json', arranque.canon({
            'certificados': certificados}))
        # La unidad se prepara un día después de las fuentes con el mismo plazo.
        fecha_unidad = (inicio + dt.timedelta(days=dias + 1)).date().isoformat()
        unidad = {'unidad_ref': 'unidad:ficticia', 'fuente_ref': 'prc_unidad',
                  'fuente_sha256': 'b' * 64, 'vigente_hasta': fecha_unidad}
        fuente_recibo = {'recibo_ref': 'rec_fuentes', 'plan_sha256': huella,
                         'ca': {'datos': {'organizacion_ref': 'org_ficticia',
                                          'personas': personas}}}
        unidad_recibo = {'recibo_ref': 'rec_unidad', 'plan_sha256': 'c' * 64,
                         'fuente_ref': unidad['fuente_ref'],
                         'fuente_sha256': unidad['fuente_sha256'], 'unidad': unidad}
        categoria = lambda tipo: {'categoria_administrativa': tipo,
                                   'fuente_ref': 'prc_rol', 'fuente_version': 1,
                                   'fuente_huella_sha256': 'd' * 64}
        rol = lambda tipo, version: {'version_ref': version, 'rol_id': tipo,
                                     'huella': 'e' * 64, 'estado': 'publicada',
                                     'control': {'estado': 'habilitada', 'revision': 1,
                                                 'huella_sha256': 'f' * 64},
                                     'categoria': categoria(tipo),
                                     'dimensiones': ['organizacion_ref']}
        metadata = {
            'control': {'bootstrap_estado': 'pendiente', 'revision': 1,
                        'bootstrap_acto_ref': None},
            'fuente': fuente_recibo, 'unidad': unidad_recibo,
            'politica': {'politica_ref': 'pga_ficticia',
                         'huella_aprobacion_sha256': 'a' * 64,
                         'vigente_hasta': vencimiento},
            'roles': [rol('aplicacion', 'rol:aplicacion'),
                      rol('sistemas', 'rol:sistemas')]}
        config = {'privado': str(self.root), 'bootstrap': {
            'ttl_plan_segundos': 7200, 'ttl_perfiles_segundos': dias * 86400,
            'duracion_propuesta_segundos': 3600,
            'app_role_ref': 'rol:aplicacion', 'sys_role_ref': 'rol:sistemas',
            'audiencia_selector': 'admin:ficticio',
            'audiencia_administrativa': 'admin:ficticio'},
            'kit': {'catalogos': {'bootstrap_preparar': '/ficticio/catalogo'}}}
        acuses = {'fuentes': (None, {'estado': 'permitido', 'recibo': fuente_recibo}),
                  'unidad': (None, {'estado': 'permitido', 'recibo': unidad_recibo})}
        return inicio, fuente_plan, config, metadata, acuses

    def test_bootstrap_acota_perfiles_a_autoridades_confirmadas(self):
        inicio, fuente, config, metadata, acuses = self.escenario_bootstrap(90)
        posterior = inicio + dt.timedelta(days=1)

        def cli_ficticia(_c, _nombre, _args):
            plan = json.loads(arranque.privado(self.root / 'bootstrap/material-v3.json'))
            self.privado('bootstrap/plan-v3.json', arranque.canon({
                'plan': plan, 'huella_plan_sha256': arranque.sha(arranque.canon(plan))}))

        with patch.object(arranque, 'now', return_value=posterior), \
             patch.object(arranque, 'confirmacion_vigente',
                          side_effect=lambda path, _fase: acuses[path.name]), \
             patch.object(arranque, 'metadata_bootstrap', return_value=metadata), \
             patch.object(arranque, 'getter_hash', return_value='a' * 64), \
             patch.object(arranque, 'cli', side_effect=cli_ficticia), \
             patch.object(arranque, 'estado'):
            arranque.preparar_bootstrap(config)
        plan = json.loads(arranque.privado(self.root / 'bootstrap/material-v3.json'))
        limite = arranque.parse(fuente['personas'][0]['vigente_hasta'])
        self.assertLess(limite, posterior + dt.timedelta(days=90))
        self.assertGreaterEqual(limite, posterior + dt.timedelta(days=60))
        self.assertLessEqual(limite, arranque.fecha_unidad_utc(
            metadata['unidad']['unidad']['vigente_hasta']))
        for persona in plan['personas']:
            hasta = arranque.parse(persona['vigente_hasta'])
            self.assertEqual(hasta, limite)
            # CA33 publica cada proyección de cuenta con la misma fecha de su persona.
            self.assertLessEqual(hasta, arranque.parse(fuente['organizacion']['vigente_hasta']))
            self.assertLessEqual(hasta, arranque.parse(next(
                p['vigente_hasta'] for p in fuente['personas']
                if p['persona_ref'] == persona['persona_ref'])))
            for sistema in persona['sistemas']:
                self.assertEqual(arranque.parse(sistema['vigente_hasta']), limite)
        self.assertTrue(all(arranque.parse(r['vigente_hasta']) == limite
                            for r in plan['gobierno']['roles']))

    def test_bootstrap_respeta_politica_vigente_y_fin_civil_de_unidad(self):
        inicio, fuente, config, metadata, _acuses = self.escenario_bootstrap(90)
        posterior = inicio + dt.timedelta(days=1)
        politica = {**metadata['politica'], 'vigente_hasta': arranque.fmt(
            posterior + dt.timedelta(days=61))}
        self.assertEqual(arranque.vencimiento_bootstrap(
            posterior, config['bootstrap'], fuente, metadata['unidad'], politica),
            posterior + dt.timedelta(days=61))
        unidad = {**metadata['unidad'], 'unidad': {
            **metadata['unidad']['unidad'],
            'vigente_hasta': (posterior + dt.timedelta(days=60)).date().isoformat()}}
        self.assertEqual(arranque.vencimiento_bootstrap(
            posterior, config['bootstrap'], fuente, unidad, metadata['politica']),
            posterior + dt.timedelta(days=60))

    def test_bootstrap_rechaza_autoridad_sin_60_dias_antes_del_plan(self):
        inicio, _fuente, config, metadata, acuses = self.escenario_bootstrap(60)
        posterior = inicio + dt.timedelta(seconds=1)
        with patch.object(arranque, 'now', return_value=posterior), \
             patch.object(arranque, 'confirmacion_vigente',
                          side_effect=lambda path, _fase: acuses[path.name]), \
             patch.object(arranque, 'metadata_bootstrap', return_value=metadata), \
             patch.object(arranque, 'getter_hash') as getter, \
             patch.object(arranque, 'cli') as cli:
            with self.assertRaisesRegex(arranque.Paro, 'bootstrap_autoridad_agotada'):
                arranque.preparar_bootstrap(config)
        getter.assert_not_called()
        cli.assert_not_called()
        self.assertFalse((self.root / 'bootstrap/material-v3.json').exists())
        self.assertFalse((self.root / 'bootstrap/plan-v3.json').exists())
        self.assertFalse((self.root / 'bootstrap/reparto-propuesto.json').exists())

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

    def test_aprobacion_rechaza_cualquier_clave_json_duplicada(self):
        plan = 'a' * 64
        aprobacion_ref = 'aprobacion:ensayo'
        backup_ref = 'backup:ensayo'
        c = {'dba': {'fuentes': {'aprobacion_ref': aprobacion_ref,
                                'aprobacion_externa': str(self.root / 'externa.json')}},
             'backup': {'ref': backup_ref}}
        casos = {
            'plan_ajeno_antes': [('huella_plan_sha256', 'b' * 64),
                                 ('huella_plan_sha256', plan),
                                 ('aprobacion_ref', aprobacion_ref), ('backup_ref', backup_ref)],
            'plan_ajeno_despues': [('huella_plan_sha256', plan),
                                   ('huella_plan_sha256', 'b' * 64),
                                   ('aprobacion_ref', aprobacion_ref), ('backup_ref', backup_ref)],
            'respaldo': [('huella_plan_sha256', plan),
                         ('aprobacion_ref', aprobacion_ref), ('backup_ref', 'otro'),
                         ('backup_ref', backup_ref)],
            'referencia': [('huella_plan_sha256', plan),
                           ('aprobacion_ref', 'otra'),
                           ('aprobacion_ref', aprobacion_ref), ('backup_ref', backup_ref)]}
        with patch.object(arranque, 'sql') as sql, patch.object(arranque, 'cli') as cli:
            for nombre, pares in casos.items():
                with self.subTest(nombre=nombre):
                    raw = ('{' + ','.join(json.dumps(clave) + ':' + json.dumps(valor)
                                        for clave, valor in pares) + '}').encode()
                    path = self.privado('externa.json', raw)
                    with self.assertRaisesRegex(arranque.Paro, 'json_clave_duplicada'):
                        arranque.aprobacion(c, 'fuentes', plan)
                    with self.assertRaisesRegex(arranque.Paro, 'json_clave_duplicada'):
                        arranque.json_privado(path)
            sql.assert_not_called()
            cli.assert_not_called()

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
