"""Guardas focales del recorrido de actos y plan; no abren red ni SQL."""

from datetime import datetime, timedelta, timezone
import base64
from types import SimpleNamespace
import json
from pathlib import Path
import tempfile
import unittest

from recorrido_copia.actos import preparar_actos
from recorrido_copia.materiales_actos import PASOS, verificar_materiales
from recorrido_copia.plan import preparar_plan


class ContextoFalso:
    def __init__(self, base):
        self.raiz = base / 'l-f131'
        self.carpeta = self.raiz / 'recorrido'
        self.copia = SimpleNamespace(nombre=self.raiz.name)
        self.material = self.raiz / 'material'
        self.fuente = base / 'fuente'
        self.parametros = {'organizacion_operativa_ref': 'organizacion:desarrollo:dipgra'}
        self.llamadas = []
        self.fallar = False
        self.admin_igual = False
        self.perfiles = ['prf_' + 'a' * 32, 'prf_' + 'b' * 32]
        self.activos = [None, None]
        self.raiz.mkdir(mode=0o700)
        for p in (self.carpeta, self.material, self.fuente):
            p.mkdir(mode=0o700)
        self._semilla()

    def guardar(self, ruta, dato):
        ruta = Path(ruta)
        ruta.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
        b = dato if isinstance(dato, bytes) else json.dumps(dato, ensure_ascii=False).encode()
        if ruta.exists():
            if ruta.read_bytes() != b:
                raise ValueError('sobrescritura')
            return
        ruta.write_bytes(b)
        ruta.chmod(0o600)

    def leer(self, ruta):
        return json.loads(Path(ruta).read_bytes())

    def sha(self, raw):
        import hashlib
        return hashlib.sha256(raw).hexdigest()

    def ahora(self):
        return datetime(2026, 10, 20, 12, tzinfo=timezone.utc)

    def admin(self, n, metodo, ruta, cuerpo):
        self.llamadas.append((n, metodo, ruta, cuerpo))
        if ruta.endswith('/propios'):
            perfil = self.perfiles[0] if self.admin_igual else self.perfiles[n - 1]
            return 200, {'revision': 0, 'perfil_activo_ref': self.activos[n - 1],
                         'perfiles': [{'perfil_ref': perfil, 'categoria_admin': 'aplicacion'}]}
        if ruta.endswith('/seleccion'):
            self.activos[n - 1] = cuerpo['perfil_ref']
            return 200, {}
        if self.fallar:
            raise OSError('respuesta perdida')
        m = json.loads(base64.b64decode(cuerpo['material_base64']))
        tipo = m.get('operacion', 'certificado')
        if ruta.endswith('/plan-firma/gobierno'):
            return 200, {'recibo': {'catalogo_ref': 'ct.plan.firma.hz39.operativa:1',
                    'estado': 'borrador' if tipo == 'crear' else 'publicado',
                    'recibo_ref': 'plan:' + tipo, 'auditoria_ref': 'auditoria:plan',
                    'consumo_auditoria_ref': 'consumo:plan'}}
        recibo = {'recibo_ref': 'recibo:' + str(len(self.llamadas)),
                  'auditoria_ref': 'auditoria:uno', 'version': 1, 'estado': 'vigente'}
        if tipo == 'certificado':
            recibo.update(vinculo_ref=m['vinculo_ref'], certificado_der_sha256=m['certificado_der_sha256'])
        else:
            recibo['objeto_ref'] = m['objeto_ref']
        return 200, {'recibo': recibo, 'consumo_auditoria_ref': 'consumo:uno'}


    def consultar(self, sql):
        return 'org_' + 'a' * 32 + '|correspondencia:uno|1\n'

    def sql(self, contenido, log):
        raise AssertionError('El mapeo ya existe en el ensayo')

    def cli(self, nombre):
        self.assert_cli = nombre
        return Path('/bin/true')

    def ejecutar(self, argv, log, cwd=None, env=None):
        from recorrido_copia.materiales_actos import canon
        if argv[0] == '/usr/bin/openssl':
            return SimpleNamespace(returncode=0, stdout=Path(argv[argv.index('-in') + 1]).read_bytes() if '-outform' in argv else b'')
        if '--config' in argv:
            cfg = self.leer(argv[2])
            salida = Path(argv[4])
            entradas = self.leer(cfg['entradas_archivo'])
            for e in entradas:
                e['atributos']['circuito_ref'] = 'vec.contratacion_temporal.circuito_firma:2'
            borrador = {'id': 'ct.plan.firma.hz39.operativa', 'version': 1,
                        'estado': 'borrador', 'creado_por': cfg['creado_por'],
                        'entradas': entradas}
            publicado = {**borrador, 'estado': 'publicado',
                         'publicado_por': cfg['publicado_por'],
                         'publicado_en': '2026-10-20T12:00:00Z'}
            crear_sha = self.sha(canon(borrador))
            for nombre, catalogo in (('crear', borrador), ('publicar', publicado)):
                self.guardar(salida / (nombre + '-catalogo.json'), canon(catalogo))
                material = {'operacion': nombre,
                            'catalogo_canonico_base64': base64.b64encode(canon(catalogo)).decode(),
                            'catalogo_sha256': self.sha(canon(catalogo)),
                            'huella_esperada': crear_sha if nombre == 'publicar' else None}
                raw = canon(material)
                self.guardar(salida / (nombre + '-material.json'), raw)
                self.guardar(salida / (nombre + '-material.sha256'), (self.sha(raw) + '\n').encode())
            return SimpleNamespace(returncode=0, stdout=b'{"estado":"preparado_sin_autorizacion"}')
        return SimpleNamespace(returncode=0, stdout=b'{"estado":"material_validado_sin_autorizacion"}')

    def _semilla(self):
        h10 = self.raiz / 'admin/h10'
        self.guardar(h10 / 'entrada-vec-admin.json', {'motivo': 'Ensayo local'})
        personas = [{'persona_ref': 'per_' + c * 32} for c in 'ab']
        self.guardar(h10 / 'bootstrap/plan-v3.json', {'plan': {'personas': personas}})
        for i, p in enumerate(personas, 1):
            self.guardar(h10 / f'bootstrap/acreditacion-certificado-{i}.json', p)
        self.guardar(h10 / 'unidad/acuse-primero.json', {'estado': 'permitido',
            'recibo': {'auditoria_ref': 'auditoria:unidad', 'unidad': {
                'organismo_ref': 'org_' + 'a' * 32, 'unidad_ref': 'unidad:desarrollo:rrhh',
                'nodo_ref': 'nodo:rrhh', 'revision': 1}}})
        refs = {}
        for c in ('tecnico', 'jefatura', 'diputacion', 'secretaria'):
            self.guardar(self.carpeta / 'certificados' / (c + '.pem'), c.encode())
            refs[c] = {'persona_ref': 'per_' + c[0] * 32, 'cuenta_ref': 'cta_' + c[0] * 32,
                       'certificado_sha256': self.sha(c.encode()), 'certificado': str(self.carpeta / 'certificados' / (c + '.pem')),
                       'vinculo': {'firma': 'vinculo:firma:' + c},
                       'perfil': {'firma': 'prf_' + c[0] * 32}}
        s = self.carpeta / 'firmantes/semilla-final'
        self.guardar(s / 'firmantes-datos.json', {'unidad_ref': 'unidad:desarrollo:rrhh',
                    'organizacion_ca_ref': 'org_' + 'a' * 32, 'nodo_ref': 'nodo:rrhh',
                    'retirada_en': (self.ahora() + timedelta(days=90)).strftime('%Y-%m-%dT%H:%M:%SZ'),
                    'referencias': refs})
        self.guardar(s / 'preparacion.json', {'actos_ejecutados': 0, 'sql_ejecutadas': 0})
        raiz = self.carpeta / 'firmantes'
        for prefijo, ids in (('roles', list(dict.fromkeys(p[4] for p in PASOS)) +
                              ['extra_' + str(i) for i in range(12)]),
                             ('consulta-v2', ['consulta_' + str(i) for i in range(4)])):
            plan = raiz / (prefijo + '-plan.json')
            acuse = raiz / (prefijo + '-acuse.json')
            cargos = [{'rol_id': rol, 'version': 1, 'version_rol_sha256': 'a' * 64}
                      for rol in ids]
            self.guardar(plan, {'cargos': cargos})
            self.guardar(acuse, {'estado': 'permitido', 'recibo': {
                'auditoria_ref': 'auditoria:aut53', 'plan_sha256': self.sha(plan.read_bytes()),
                'cargos': [{'version_rol_ref': 'rol:' + rol + ':v1',
                            'version_rol_sha256': 'a' * 64, 'control_sha256': 'b' * 64}
                           for rol in ids]}})
        self.guardar(raiz / 'metadata.json', {
            'copia_ref': self.copia.nombre,
            'semilla_copia_ref': 'f131-' + self.sha(self.copia.nombre.encode())[:12],
            'firmantes_datos': str(s / 'firmantes-datos.json'),
            'preparacion': str(s / 'preparacion.json'),
            'roles_plan': str(raiz / 'roles-plan.json'),
            'roles_acuse': str(raiz / 'roles-acuse.json'),
            'consulta_v2_plan': str(raiz / 'consulta-v2-plan.json'),
            'consulta_v2_acuse': str(raiz / 'consulta-v2-acuse.json'),
            'unidad_h10_acuse_sha256': self.sha((h10 / 'unidad/acuse-primero.json').read_bytes()),
            'roles_acuse_sha256': self.sha((raiz / 'roles-acuse.json').read_bytes()),
            'consulta_v2_acuse_sha256': self.sha((raiz / 'consulta-v2-acuse.json').read_bytes()),
            'roles_iniciales': 16, 'sucesoras_consulta': 4,
            'alcance': 'semilla técnica ficticia; sin aprobación de dos administradores ni firma'})


class ActosTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.c = ContextoFalso(Path(self.tmp.name))

    def test_catorce_recibos_y_replay_sin_reenvio(self):
        meta = preparar_actos(self.c)
        self.assertEqual(meta['actos_confirmados'], 14)
        anteriores = len([x for x in self.c.llamadas if x[1] == 'POST' and x[2].endswith('/publicacion')])
        self.assertEqual(anteriores, 14)
        self.assertEqual(preparar_actos(self.c), meta)
        self.assertEqual(len([x for x in self.c.llamadas if x[1] == 'POST' and x[2].endswith('/publicacion')]), 14)

    def test_material_alterado_se_rechaza_antes_de_http(self):
        from recorrido_copia.materiales_actos import preparar_materiales
        m = preparar_materiales(self.c)
        ruta = Path(m['acciones'][0]['material'])
        ruta.write_bytes(ruta.read_bytes() + b' ')
        with self.assertRaisesRegex(ValueError, 'alterado'):
            verificar_materiales(self.c, m)
        self.assertFalse(self.c.llamadas)

    def test_respuesta_incierta_no_repite(self):
        self.c.fallar = True
        with self.assertRaisesRegex(ValueError, 'incierto'):
            preparar_actos(self.c)
        antes = len(self.c.llamadas)
        self.c.fallar = False
        with self.assertRaisesRegex(ValueError, 'incierto'):
            preparar_actos(self.c)
        self.assertEqual(len(self.c.llamadas), antes + 2)  # Solo GET de los perfiles.

    def test_fuente_firmantes_alterada_no_reenvia(self):
        preparar_actos(self.c)
        fuente = self.c.carpeta / 'firmantes/semilla-final/firmantes-datos.json'
        fuente.write_bytes(fuente.read_bytes() + b' ')
        antes = len(self.c.llamadas)
        with self.assertRaisesRegex(ValueError, 'otra copia o fuente'):
            preparar_actos(self.c)
        self.assertEqual(len(self.c.llamadas), antes)

    def test_mismo_admin_se_rechaza_antes_de_actos(self):
        self.c.admin_igual = True
        with self.assertRaisesRegex(ValueError, 'comparten perfil'):
            preparar_actos(self.c)
        self.assertFalse(any(x[2].endswith('/publicacion') for x in self.c.llamadas))



class PlanTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.c = ContextoFalso(Path(self.tmp.name))
        origen = self.c.fuente / 'data/catalogos/contratacion-temporal/circuito_firma.v2.json'
        self.c.guardar(origen, {'fuente': {'demostracion': False},
            'catalogo': {'id': 'vec.contratacion_temporal.circuito_firma', 'version': 2,
                         'estado': 'publicado', 'publicado_en': '2026-10-01T00:00:00Z',
                         'aprobacion_ref': 'aprobacion:uno'}})
        self.c.guardar(self.c.fuente / 'scripts/operacion/firma_ct_tres_pasos/publicar_correspondencia.sql', b'')

    def test_plan_dos_admin_y_replay(self):
        preparar_actos(self.c)
        meta = preparar_plan(self.c)
        self.assertEqual(meta['recibos'], ['plan:crear', 'plan:publicar'])
        antes = len([x for x in self.c.llamadas if x[2].endswith('/plan-firma/gobierno')])
        self.assertEqual(antes, 2)
        self.assertEqual(preparar_plan(self.c), meta)
        self.assertEqual(len([x for x in self.c.llamadas if x[2].endswith('/plan-firma/gobierno')]), 2)

    def test_plan_incierto_no_repite_publicacion(self):
        preparar_actos(self.c)
        self.c.fallar = True
        with self.assertRaisesRegex(ValueError, 'incierto'):
            preparar_plan(self.c)
        antes = len([x for x in self.c.llamadas if x[2].endswith('/plan-firma/gobierno')])
        self.c.fallar = False
        with self.assertRaisesRegex(ValueError, 'incierto'):
            preparar_plan(self.c)
        self.assertEqual(len([x for x in self.c.llamadas if x[2].endswith('/plan-firma/gobierno')]), antes)

    def test_plan_material_alterado_falla_antes_de_post(self):
        preparar_actos(self.c)
        meta = preparar_plan(self.c)
        ruta = self.c.carpeta / 'plan/material/crear-material.json'
        ruta.write_bytes(ruta.read_bytes() + b' ')
        with self.assertRaisesRegex(ValueError, 'modificado'):
            preparar_plan(self.c)
        self.assertEqual(len([x for x in self.c.llamadas if x[2].endswith('/plan-firma/gobierno')]), 2)

if __name__ == '__main__':
    unittest.main()
