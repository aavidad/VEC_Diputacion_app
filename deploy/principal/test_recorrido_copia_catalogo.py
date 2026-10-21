"""Recibos AUT58 de dos actos, con SQL y CLI interceptados."""

import hashlib
import json
import os
from pathlib import Path
import tempfile
import unittest
from types import SimpleNamespace

from recorrido_copia import catalogo


class CopiaFalsa:
    nombre = 'vec-copia-l-f131'

    def __init__(self, estado):
        self.estado = estado

    def leer(self):
        return self.estado


class ContextoFalso:
    SID = '1234567890123456789'

    def __init__(self, root, repo, replay_alterado=False):
        self.raiz = root
        self.carpeta = root / 'recorrido'
        self.carpeta.mkdir(mode=0o700)
        self.material = root / 'material'
        (self.material / 'ca').mkdir(parents=True, mode=0o700)
        self.guardar(self.material / 'ca/ca.crt', b'ca-ficticia')
        self.fuente = repo
        self.copia = CopiaFalsa({'admin_preparada': True, 'nombre': CopiaFalsa.nombre,
                                 'sid_copia': self.SID, 'revision': 'a' * 40,
                                 'puerto_pg': 55441})
        backup = root / 'admin/preimagen-admin.dump'
        self.guardar(backup, b'preimagen-ficticia')
        config = root / 'admin/h10/config.json'
        self.guardar(config, {'backup': {'ref': 'preimagen:admin:f131',
                                        'sha256': self.sha(backup)}})
        self.guardar(root / 'admin/h10/contrato-arranque.json',
                     {'sid_copia': self.SID, 'revision': 'a' * 40,
                      'config_sha256': self.sha(config)})
        self.version, self.head_sha = 1, 'a' * 64
        self.commands, self.sql_calls = [], []
        self.replay_alterado = replay_alterado

    @staticmethod
    def sha(path):
        return hashlib.sha256(Path(path).read_bytes()).hexdigest()

    @staticmethod
    def leer(path):
        return json.loads(Path(path).read_bytes())

    @staticmethod
    def guardar(path, value):
        path = Path(path)
        path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        raw = value if isinstance(value, bytes) else (json.dumps(value, ensure_ascii=False,
                separators=(',', ':')) + '\n').encode()
        if path.exists():
            if path.read_bytes() != raw:
                raise ValueError('material anterior divergente')
            return False
        fd = os.open(path, os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW, 0o600)
        with os.fdopen(fd, 'wb') as out:
            out.write(raw)
        return True

    def consultar(self, query):
        if query != 'SELECT system_identifier::text FROM pg_catalog.pg_control_system();':
            raise AssertionError('consulta SQL no prevista')
        return self.SID

    def cli(self, name):
        assert name == 'vec-catalogo-acciones-admitir'
        return Path('/usr/bin/true')

    def sql(self, data, log):
        self.sql_calls.append((data, log))

    def ejecutar(self, command, log, cwd=None, env=None):
        assert env == {'VEC_COPIA_RAIZ': str(self.raiz)}
        command = [str(x) for x in command]
        self.commands.append((command, log))
        if command[0] == '/usr/bin/true':
            return self._native(command)
        action = command[2]
        base = Path(command[command.index('--dir') + 1]) if '--dir' in command else None
        if action == 'preparar':
            mode = command[command.index('--modo') + 1]
            act = int(command[command.index('--acto') + 1])
            source = command[command.index('--suelta') + 1].split(':')
            package = {'version': self.version + 1, 'fuente': source[0]}
            cat = {'referencia': catalogo.CATALOGO, 'version': self.version + 1,
                   'fuente_huella_sha256': 'b' * 64, 'entradas': [source[0]],
                   'perfiles': []}
            self.guardar(base / 'paquete.json', package)
            self.guardar(base / 'catalogo.json', cat)
            self.guardar(base / 'preparado.json', {'acto': str(act), 'modo': mode,
                'sueltas': [source], 'sid_copia': self.SID, 'encargo_ref': 'F1-131',
                'backup_ref': 'preimagen:admin:f131',
                'backup_sha256': self.sha(self.raiz / 'admin/preimagen-admin.dump'),
                'catalogo_sha256': self.sha(base / 'catalogo.json'),
                'paquete_sha256': self.sha(base / 'paquete.json'),
                'prev_version': self.version, 'prev_huella': self.head_sha,
                'censo_preparado_sha256': 'c' * 64})
        elif action == 'configurar':
            meta = self.leer(base / 'preparado.json')
            approval = self.leer(base / 'aprobacion-tecnica.json')
            plan = {'operacion_ref': f'caa_{meta["acto"]}',
                    'catalogo_sha256': meta['catalogo_sha256'],
                    'aprobacion_sha256': self.sha(base / 'aprobacion-tecnica.json')}
            self.guardar(base / 'plan.json', plan)
            self.guardar(base / 'conexion.json', {'dsn': 'conexion-ficticia'})
            self.guardar(base / 'configurar.sql', b'SQL interceptado')
            self.guardar(base / 'configurado.json', {'plan_sha256': self.sha(base / 'plan.json'),
                'sid_copia': self.SID, 'encargo_ref': 'F1-131',
                'catalogo_sha256': meta['catalogo_sha256'],
                'paquete_sha256': meta['paquete_sha256'],
                'aprobacion_ref': approval['aprobacion_ref'],
                'aprobacion_sha256': self.sha(base / 'aprobacion-tecnica.json'),
                'aprobador_ref': 'fabrica:copias:desarrollo',
                'preimagen': {'catalogo_ref': catalogo.CATALOGO,
                             'version': meta['prev_version'],
                             'huella_sha256': meta['prev_huella'],
                             'censo_preparado_sha256': meta['censo_preparado_sha256']},
                'login': 'vec_caa_1234567890abcdef',
                'operacion_ref': plan['operacion_ref']})
        elif action == 'acuses':
            first = self.leer(command[command.index('--primero') + 1])
            second = self.leer(command[command.index('--replay') + 1])
            assert first['recibo'] == second['recibo'] and second['replay'] is True
        return SimpleNamespace(stdout=b'')

    def _native(self, command):
        phase = command[command.index('--fase') + 1]
        base = Path(command[command.index('--config') + 1]).parent
        meta = self.leer(base / 'preparado.json')
        if phase == 'consultar':
            return SimpleNamespace(stdout=json.dumps({'codigo': 'catalogo_consultado',
                'catalogo': self.leer(base / 'catalogo.json')}).encode())
        plan = self.leer(base / 'plan.json')
        approval = self.leer(base / 'aprobacion-tecnica.json')
        replay = self.version > meta['prev_version']
        receipt = {'esquema': 'vec.admin.catalogo-acciones.recibo.v1',
            'operacion_ref': plan['operacion_ref'], 'plan_sha256': self.sha(base / 'plan.json'),
            'catalogo_ref': catalogo.CATALOGO,
            'catalogo_version': str(meta['prev_version'] + 1),
            'catalogo_sha256': meta['catalogo_sha256'],
            'paquete_sha256': meta['paquete_sha256'],
            'aprobacion_ref': approval['aprobacion_ref'],
            'aprobacion_sha256': self.sha(base / 'aprobacion-tecnica.json'),
            'aprobador_ref': 'fabrica:copias:desarrollo',
            'auditoria_ref': f'aud_v3_caa_{meta["acto"]}',
            'confirmado_en': '2026-10-20T12:00:00Z'}
        if not replay:
            self.version += 1
            self.head_sha = meta['catalogo_sha256']
        elif self.replay_alterado and meta['acto'] == '12':
            receipt['auditoria_ref'] = 'aud_v3_caa_ajeno'
        response = {'codigo': 'catalogo_admitido', 'estado': 'permitido', 'recibo': receipt}
        if replay:
            response['replay'] = True
        return SimpleNamespace(stdout=json.dumps(response).encode())


class CatalogoRecorridoTest(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(dir=os.environ.get('VEC_TEST_SCRATCH'))
        self.addCleanup(self.scratch.cleanup)
        self.root = Path(self.scratch.name)
        os.chmod(self.root, 0o700)
        self.repo = Path(__file__).parents[2]

    def test_dos_admisiones_recibos_reales_y_replay(self):
        c = ContextoFalso(self.root, self.repo)
        meta = catalogo.preparar(c)
        self.assertEqual(meta['recibos_reales'], 2)
        self.assertEqual([a['nombre'] for a in meta['actos']], ['organizacion', 'origen_ct'])
        self.assertEqual(len(c.sql_calls), 2)
        self.assertEqual([a['catalogo_version'] for a in meta['actos']], [2, 3])
        self.assertEqual(c.leer(c.carpeta / 'catalogo/admisiones.json'), meta)
        for act in meta['actos']:
            base = c.carpeta / 'catalogo' / act['nombre']
            self.assertEqual(c.leer(base / 'acuse-primero.json')['recibo'],
                             c.leer(base / 'acuse-replay.json')['recibo'])
            approval = c.leer(base / 'aprobacion-tecnica.json')
            self.assertEqual(approval['sid_copia'], c.SID)
            self.assertEqual(approval['backup_sha256'], c.sha(c.raiz / 'admin/preimagen-admin.dump'))
            self.assertEqual(approval['aprobador_ref'], 'fabrica:copias:desarrollo')

    def test_replay_distinto_no_publica_manifiesto_final(self):
        c = ContextoFalso(self.root, self.repo, replay_alterado=True)
        with self.assertRaisesRegex(ValueError, 'replay AUT58 cambió'):
            catalogo.preparar(c)
        self.assertFalse((c.carpeta / 'catalogo/admisiones.json').exists())


    def test_dietas_admite_solo_fuentes_opcionales_y_conserva_admisiones_base(self):
        c = ContextoFalso(self.root, self.repo)
        base = catalogo.preparar(c)
        antes = (c.carpeta / 'catalogo/admisiones.json').read_bytes()
        nuevas = catalogo.preparar_dietas(c)
        self.assertEqual([x['nombre'] for x in nuevas['actos']], ['personal_dietas', 'dietas'])
        self.assertEqual([x['catalogo_version'] for x in nuevas['actos']], [4, 5])
        self.assertEqual((c.carpeta / 'catalogo/admisiones.json').read_bytes(), antes)
        self.assertEqual(base['recibos_reales'], 2)
        self.assertEqual(len(c.sql_calls), 4)
        for acto in nuevas['actos']:
            carpeta = c.carpeta / 'catalogo' / acto['nombre']
            self.assertEqual(c.leer(carpeta / 'acuse-primero.json')['recibo'],
                             c.leer(carpeta / 'acuse-replay.json')['recibo'])

    def test_fuentes_dietas_canonicas_cinco_acciones_y_vigencia_de_90_dias(self):
        from admin_copia.catalogo_acciones import fuente_nueva
        from datetime import datetime
        acciones = []
        for _, _, nombre, huella in catalogo.ACTOS_DIETAS:
            ruta = self.repo / 'deploy/principal/fuentes'
            fuente_nueva(ruta, nombre, huella)
            documento = json.loads((ruta / nombre).read_bytes())
            for entrada in documento['entradas']:
                acciones.append(entrada['concesion']['accion'])
                inicio = datetime.fromisoformat(entrada['vigente_desde'])
                fin = datetime.fromisoformat(entrada['vigente_hasta'])
                self.assertEqual((fin - inicio).days, 90)
                self.assertEqual(entrada['clase_control'], 'ordinario')
                self.assertEqual(entrada['concesion']['modulo_id'], documento['modulo_id'])
        self.assertEqual(acciones, ['personal.relacion.propia.consultar_dietas',
            'dietas.borrador.propio.crear', 'dietas.borrador.propio.consultar',
            'dietas.ruta.catalogo.consultar', 'dietas.ruta.calculo.solicitar'])
