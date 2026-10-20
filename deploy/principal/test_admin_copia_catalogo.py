"""Casos focales del preparador AUT58; solo datos inventados."""
import json
import os
import tempfile
import unittest
from pathlib import Path
from unittest.mock import Mock, patch

from admin_copia import catalogo_acciones as cat


def entrada(ref, version, fuente, fuente_version, huella):
    return {
        'referencia': ref, 'version': version, 'fuente_ref': fuente,
        'fuente_version': fuente_version, 'fuente_huella_sha256': huella,
        'concesion': {'accion': ref.removeprefix('accion:'), 'modulo_id': 'bolsa',
                      'tipo_recurso': 'ficticio', 'finalidades': ['prueba'],
                      'garantia_minima': 'alto'},
        'dimensiones_ambito': [], 'clase_control': 'administrable',
        'vigente_desde': '2026-01-01T00:00:00Z',
        'vigente_hasta': '2027-01-01T00:00:00Z',
    }


def fuente(ref, version, accion):
    e = entrada('accion:' + accion, version, ref, version, '')
    proy = {k: v for k, v in e.items() if k != 'fuente_huella_sha256'}
    # La proyección del productor tiene el orden canónico usado por Go.
    canon = cat.go_json([proy])
    h = cat.sha(canon.encode())
    e['fuente_huella_sha256'] = h
    return {'modulo_id': 'bolsa', 'referencia': ref, 'version': version,
            'huella_sha256': h, 'entradas_canon': canon, 'entradas': [e]}


class CatalogoAccionesTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.base = Path(self.tmp.name)
        self.ref = 'fuente:bolsa:origen_ct_ficticio'
        otra = fuente('fuente:bolsa:otra_ficticia', 1, 'bolsa.otra.consultar')
        vieja = fuente(self.ref, 1, 'bolsa.origen_ct.consultar')
        self.fuentes = [otra, vieja]
        raw = {'rol_id': 'rol_ficticio', 'version': 1, 'nombre': 'Rol ficticio',
               'estado': 'publicada', 'concesiones': [], 'publicada_por': 'actor:ficticio',
               'publicada_en': '2026-01-01T00:00:00Z'}
        self.censo = [{'rol_id': 'rol_ficticio', 'raw_role': raw, 'rol': raw,
                       'huella_sha256': cat.sha(cat.go_json(cat.rol_go(raw)).encode()),
                       'control_vigencia': {'version_rol_ref': 'rol:rol_ficticio:v1',
                                            'revision': 1, 'estado': 'habilitada',
                                            'actualizado_por': 'actor:ficticio',
                                            'actualizado_en': '2026-01-01T00:00:00Z'},
                       'tipo_perfil': 'administrable'}]
        perfiles = [cat.od(('rol', cat.rol_go(raw, completar=False)),
                           ('control_vigencia', cat.control_go(self.censo[0]['control_vigencia'])),
                           ('tipo_perfil', 'administrable'))]
        self.paquete = cat.od(('esquema', 'vec.admin.catalogo-acciones.paquete.v2'),
                              ('referencia', 'paquete:ficticio'), ('version', 1),
                              ('fuentes', self.fuentes), ('perfiles', perfiles))
        self.catalogo = cat.od(('referencia', cat.CATALOGO_REF), ('version', 1),
                               ('fuente_ref', 'paquete:ficticio'), ('fuente_version', 1),
                               ('fuente_huella_sha256', cat.sha(cat.go_json(self.paquete).encode())),
                               ('vigente_desde', '2026-01-01T00:00:00Z'),
                               ('vigente_hasta', '2027-01-01T00:00:00Z'),
                               ('entradas', [e for f in self.fuentes for e in f['entradas']]),
                               ('perfiles', perfiles))
        self.cabeza = {'version': 1, 'huella': cat.sha(cat.go_json(self.catalogo).encode()),
                       'canon': cat.go_json(self.catalogo), 'paquete': cat.go_json(self.paquete),
                       'paquete_ref': 'paquete:ficticio', 'paquete_version': 1}

    def escribir(self, nombre, f):
        raw = (json.dumps(f, ensure_ascii=False) + '\n').encode()
        (self.base / nombre).write_bytes(raw)
        return cat.sha(raw)

    def preparar(self, modo, nombre, archivo_sha):
        with patch.object(cat, 'psql', side_effect=[json.dumps(self.cabeza), json.dumps(self.censo)]):
            return cat.preparar_catalogo(str(self.base), modo, [(nombre, archivo_sha)])

    def test_sustituye_v1_por_v2_y_conserva_otra_fuente(self):
        nueva = fuente(self.ref, 2, 'bolsa.origen_ct.consultar')
        h = self.escribir('origen_ficticio_v2.json', nueva)
        paquete, catalogo, resumen = self.preparar('sustituir', 'origen_ficticio_v2.json', h)
        actualizado = json.loads(paquete)
        self.assertEqual(actualizado['fuentes'][0], self.fuentes[0])
        self.assertEqual(actualizado['fuentes'][1]['version'], 2)
        self.assertEqual(json.loads(catalogo)['entradas'][0], self.fuentes[0]['entradas'][0])
        self.assertEqual(resumen['prev_huella'], self.cabeza['huella'])

    def test_rechaza_sha_archivo_y_accion_ajena(self):
        nueva = fuente(self.ref, 2, 'bolsa.otra.consultar')
        h = self.escribir('origen_ficticio_v2.json', nueva)
        with self.assertRaises(SystemExit):
            self.preparar('sustituir', 'origen_ficticio_v2.json', 'a' * 64)
        with self.assertRaises(SystemExit):
            self.preparar('sustituir', 'origen_ficticio_v2.json', h)

    def test_rechaza_preimagen_cambiada(self):
        nueva = fuente(self.ref, 2, 'bolsa.origen_ct.consultar')
        h = self.escribir('origen_ficticio_v2.json', nueva)
        paquete, catalogo, _ = self.preparar('sustituir', 'origen_ficticio_v2.json', h)
        (self.base / 'paquete.json').write_text(paquete)
        (self.base / 'catalogo.json').write_text(catalogo)
        os.chmod(self.base / 'paquete.json', 0o600)
        os.chmod(self.base / 'catalogo.json', 0o600)
        meta = {'modo': 'sustituir', 'sueltas': [('origen_ficticio_v2.json', h)],
                'paquete_sha256': cat.sha(paquete.encode()), 'catalogo_sha256': cat.sha(catalogo.encode())}
        cambiada = dict(self.cabeza, version=2)
        with patch.object(cat, 'psql', side_effect=[json.dumps(cambiada), json.dumps(self.censo)]):
            with self.assertRaises(SystemExit):
                cat.revisar(type('O', (), {'dir': str(self.base), 'fuentes': str(self.base)})(), meta)

    def test_acuses_exige_replay_y_recibo_identico(self):
        a = self.base / 'acuse1.json'
        b = self.base / 'acuse2.json'
        for ruta, doc in ((a, {'recibo': {'operacion_ref': 'caa_ficticia', 'catalogo_version': 2}}),
                          (b, {'replay': True, 'recibo': {'operacion_ref': 'caa_ficticia', 'catalogo_version': 3}})):
            ruta.write_text(json.dumps(doc))
            os.chmod(ruta, 0o600)
        with self.assertRaises(SystemExit):
            cat.cmd_acuses(type('O', (), {'primero': str(a), 'replay': str(b)})())

    def test_configurar_rechaza_aprobacion_de_otra_copia(self):
        meta = {'catalogo_sha256': 'a' * 64, 'paquete_sha256': 'b' * 64,
                'backup_ref': 'copia:ficticia:uno', 'backup_sha256': 'c' * 64,
                'encargo_ref': 'F1-131', 'sid_copia': '123456789012345',
                'prev_version': 1, 'prev_huella': 'd' * 64,
                'censo_preparado_sha256': 'e' * 64}
        aprobacion = {'esquema': 'vec.admin.catalogo-acciones.aprobacion-tecnica-copia.v1',
                      'catalogo_sha256': meta['catalogo_sha256'],
                      'paquete_sha256': meta['paquete_sha256'],
                      'backup_ref': 'copia:ficticia:otra',
                      'backup_sha256': meta['backup_sha256'],
                      'encargo_ref': meta['encargo_ref'],
                      'sid_copia': meta['sid_copia'],
                      'esperado_version': meta['prev_version'],
                      'esperado_sha256': meta['prev_huella'],
                      'censo_preparado_sha256': meta['censo_preparado_sha256'],
                      'aprobacion_ref': 'aprobacion:ficticia',
                      'aprobador_ref': cat.APROBADOR_REF}
        for nombre, doc in (('preparado.json', meta), ('aprobacion.json', aprobacion)):
            ruta = self.base / nombre
            ruta.write_text(json.dumps(doc))
            os.chmod(ruta, 0o600)
        o = type('O', (), {'dir': str(self.base), 'fuentes': str(self.base),
                            'aprobacion': str(self.base / 'aprobacion.json')})()
        with patch.object(cat, 'revisar'):
            with self.assertRaises(SystemExit):
                cat.cmd_configurar(o)

    def test_sid_ajeno_rechaza_antes_de_crear_artefactos(self):
        meta = {'catalogo_sha256': 'a' * 64, 'paquete_sha256': 'b' * 64,
                'backup_ref': 'copia:ficticia:uno', 'backup_sha256': 'c' * 64,
                'encargo_ref': 'F1-131', 'sid_copia': '123456789012345',
                'prev_version': 1, 'prev_huella': 'd' * 64,
                'censo_preparado_sha256': 'e' * 64}
        aprobacion = {'esquema': 'vec.admin.catalogo-acciones.aprobacion-tecnica-copia.v1',
                      'catalogo_sha256': meta['catalogo_sha256'],
                      'paquete_sha256': meta['paquete_sha256'],
                      'backup_ref': meta['backup_ref'],
                      'backup_sha256': meta['backup_sha256'],
                      'encargo_ref': meta['encargo_ref'],
                      'sid_copia': meta['sid_copia'],
                      'esperado_version': meta['prev_version'],
                      'esperado_sha256': meta['prev_huella'],
                      'censo_preparado_sha256': meta['censo_preparado_sha256'],
                      'aprobacion_ref': 'aprobacion:ficticia',
                      'aprobador_ref': cat.APROBADOR_REF}
        for nombre, doc in (('preparado.json', meta), ('aprobacion.json', aprobacion)):
            ruta = self.base / nombre
            ruta.write_text(json.dumps(doc))
            os.chmod(ruta, 0o600)
        o = type('O', (), {'dir': str(self.base), 'fuentes': str(self.base),
                            'aprobacion': str(self.base / 'aprobacion.json')})()
        ajena = dict(aprobacion, aprobador_ref='alberto:direccion')
        (self.base / 'aprobacion.json').write_text(json.dumps(ajena))
        with patch.object(cat, 'revisar'), patch.object(cat, 'psql') as consulta:
            with self.assertRaises(SystemExit):
                cat.cmd_configurar(o)
            consulta.assert_not_called()
        (self.base / 'aprobacion.json').write_text(json.dumps(aprobacion))
        with patch.object(cat, 'revisar'), \
                patch.object(cat, 'psql', return_value='123456789012346\n') as consulta:
            with self.assertRaises(SystemExit):
                cat.cmd_configurar(o)
        consulta.assert_called_once_with(cat.SQL_SID)
        for nombre in ('plan.json', 'conexion.json', 'configurar.sql', 'configurado.json'):
            self.assertFalse((self.base / nombre).exists(), nombre)

    def test_aprobacion_con_dos_huellas_rechazada_antes_del_sid(self):
        meta = {'catalogo_sha256': 'a' * 64, 'paquete_sha256': 'b' * 64,
                'backup_ref': 'copia:ficticia:uno', 'backup_sha256': 'c' * 64,
                'encargo_ref': 'F1-131', 'sid_copia': '123456789012345',
                'prev_version': 1, 'prev_huella': 'd' * 64,
                'censo_preparado_sha256': 'e' * 64}
        aprobacion = {'esquema': 'vec.admin.catalogo-acciones.aprobacion-tecnica-copia.v1',
                      'catalogo_sha256': meta['catalogo_sha256'],
                      'paquete_sha256': meta['paquete_sha256'],
                      'backup_ref': meta['backup_ref'], 'backup_sha256': meta['backup_sha256'],
                      'encargo_ref': meta['encargo_ref'], 'sid_copia': meta['sid_copia'],
                      'esperado_version': meta['prev_version'],
                      'esperado_sha256': meta['prev_huella'],
                      'censo_preparado_sha256': meta['censo_preparado_sha256'],
                      'aprobacion_ref': 'aprobacion:ficticia',
                      'aprobador_ref': cat.APROBADOR_REF}
        preparado = self.base / 'preparado.json'
        preparado.write_text(json.dumps(meta))
        os.chmod(preparado, 0o600)
        aprobada = self.base / 'aprobacion.json'
        raw = json.dumps(aprobacion)
        raw = raw.replace('"catalogo_sha256":', '"catalogo_sha256": "' + 'f' * 64 + '", "catalogo_sha256":', 1)
        aprobada.write_text(raw)
        os.chmod(aprobada, 0o600)
        o = type('O', (), {'dir': str(self.base), 'fuentes': str(self.base),
                            'aprobacion': str(aprobada)})()
        with patch.object(cat, 'revisar'), patch.object(cat, 'psql') as consulta:
            with self.assertRaises(SystemExit):
                cat.cmd_configurar(o)
            consulta.assert_not_called()
        for nombre in ('plan.json', 'conexion.json', 'configurar.sql', 'configurado.json'):
            self.assertFalse((self.base / nombre).exists(), nombre)


class DestinoCopiaTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.base = Path(self.tmp.name) / 'copias'
        self.base.mkdir()
        self.raiz = self.base / 'l-focal'

    def test_comprueba_etiqueta_con_docker_local_sin_cambiar_entorno(self):
        self.raiz.mkdir(mode=0o700)
        (self.raiz / 'estado.json').write_text(json.dumps({
            'nombre': 'vec-copia-l-focal', 'raiz': str(self.raiz),
            'fuente': str(cat.copia_local.REPO)}))
        inspeccion = Mock(returncode=0, stdout=json.dumps([
            {'Config': {'Labels': {'vec.copia': str(self.raiz)}}}]))
        with patch.object(cat.copia_local, 'BASE', self.base), \
                patch.dict(os.environ, {'VEC_COPIA_RAIZ': str(self.raiz),
                                        'DOCKER_HOST': 'tcp://destino-ajeno.invalid:2375'}), \
                patch.object(cat.copia_local.subprocess, 'run', return_value=inspeccion) as ejecutar:
            self.assertEqual(cat.copia_destino().nombre, 'vec-copia-l-focal')
            self.assertEqual(os.environ['DOCKER_HOST'], 'tcp://destino-ajeno.invalid:2375')
        self.assertEqual(ejecutar.call_args.kwargs['env'], {
            'PATH': '/usr/bin:/bin', 'LANG': 'C',
            'DOCKER_HOST': 'unix:///var/run/docker.sock'})
        self.assertEqual(ejecutar.call_args.args[0], [
            'docker', 'container', 'inspect', 'vec-copia-l-focal'])

    def test_rechaza_raiz_ajena_y_enlazada_sin_procesos(self):
        with patch.object(cat.copia_local, 'BASE', self.base), \
                patch.object(cat.copia_local.Copia, 'existe') as existe:
            with patch.dict(os.environ, {'VEC_COPIA_RAIZ': str(Path(self.tmp.name) / 'ajena')}):
                with self.assertRaises(SystemExit):
                    cat.copia_destino()
            ajena = Path(self.tmp.name) / 'ajena'
            ajena.mkdir()
            self.raiz.symlink_to(ajena, target_is_directory=True)
            with patch.dict(os.environ, {'VEC_COPIA_RAIZ': str(self.raiz)}):
                with self.assertRaises(SystemExit):
                    cat.copia_destino()
            existe.assert_not_called()

    def test_argv_fijo_para_copia_con_estado_propio(self):
        self.raiz.mkdir()
        (self.raiz / 'estado.json').write_text(json.dumps({
            'nombre': 'vec-copia-l-focal', 'raiz': str(self.raiz),
            'fuente': str(cat.copia_local.REPO)}))
        resultado = Mock(returncode=0, stdout='lectura\n', stderr='')
        with patch.object(cat.copia_local, 'BASE', self.base), \
                patch.object(cat.copia_local.Copia, 'existe', return_value=True), \
                patch.object(cat.subprocess, 'run', return_value=resultado) as ejecutar, \
                patch.dict(os.environ, {'VEC_COPIA_RAIZ': str(self.raiz), 'HZ_PSQL': '["/bin/false"]'}):
            self.assertEqual(cat.psql(cat.SQL_CABEZA), 'lectura\n')
        self.assertEqual(ejecutar.call_args.args[0], [
            '/usr/bin/docker', 'exec', '-i', '-u', 'postgres', 'vec-copia-l-focal',
            'psql', '-X', '-At', '-q', '-v', 'ON_ERROR_STOP=1', '-U', 'postgres', '-d', 'postgres'])
        self.assertEqual(ejecutar.call_args.kwargs['input'], cat.SQL_CABEZA)


if __name__ == '__main__':
    unittest.main()
