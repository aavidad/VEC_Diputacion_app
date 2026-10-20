"""La copia enciende Mis preferencias y Mi imagen con conexiones propias (F1-113)."""
import importlib.util
import json
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).parent))
spec = importlib.util.spec_from_file_location(
    'usuarios', Path(__file__).with_name('copia_local_usuarios.py'))
usuarios = importlib.util.module_from_spec(spec)
spec.loader.exec_module(usuarios)

MOTIVO = {'catalogo_id': 'motivos_ficticios', 'entrada_clave': 'motivo_1',
          'catalogo_version': 1, 'catalogo_huella_sha256': 'a' * 64}


def original(superficie):
    datos = {'version': 1, 'autoridad': 'no_autoritativo', 'superficie': superficie,
             'cuentas': [{'certificado_sha256': 'b' * 64, 'sujeto': 'desarrollo:ficticio',
                          'cuenta_ref': 'cta_ficticia', 'perfil_ref': 'prf_ficticio'}],
             'motivo_consulta': MOTIVO, 'motivo_actualizacion': MOTIVO}
    datos.update({clave: 'postgresql://principal:secreto@principal/postgres' for clave in usuarios.CONEXIONES})
    return datos


class UsuariosCopiaTest(unittest.TestCase):
    def test_dieciseis_conexiones_distintas_con_su_grupo(self):
        filas = usuarios.plan()
        self.assertEqual(len(filas), 16)
        self.assertEqual(len({(s, c) for s, c, _ in filas}), 16)
        grupos = {(s, c): g for s, c, g in filas}
        self.assertEqual(grupos[('interna', 'dsn_usuarios')], 'vec_usuarios_ejecutor_interno')
        self.assertEqual(grupos[('externa', 'dsn_usuarios_frontera')], 'vec_usuarios_registrador_frontera_externo')
        self.assertEqual(grupos[('externa', 'dsn_contexto')], 'vec_contexto_actor_v1_runtime')

    def test_no_traslada_ninguna_conexion_de_la_principal(self):
        dsns = {clave: 'postgresql://copia@localhost/postgres' for clave in usuarios.CONEXIONES}
        nuevo = usuarios.configuracion(original('interna_corporativa'), dsns, 'interna')
        self.assertNotIn('principal', json.dumps(nuevo))
        self.assertEqual(nuevo['cuentas'], original('interna_corporativa')['cuentas'])
        self.assertEqual(nuevo['motivo_consulta'], MOTIVO)

    def test_rechaza_superficie_cambiada_o_campos_desconocidos(self):
        dsns = {clave: 'x' for clave in usuarios.CONEXIONES}
        with self.assertRaises(ValueError):
            usuarios.configuracion(original('externa_personal'), dsns, 'interna')
        raro = original('interna_corporativa') | {'dsn_otra': 'x'}
        with self.assertRaises(ValueError):
            usuarios.configuracion(raro, dsns, 'interna')

    def test_sql_una_membresia_sin_set_y_sin_reutilizar_logins(self):
        texto = usuarios.sql(['vcl_a', 'vcl_b'], ['c1', 'c2'],
                             ['vec_usuarios_ejecutor_interno', 'vec_autorizacion_fuente'], '2027-01-01T00:00:00Z').decode()
        self.assertTrue(texto.startswith('BEGIN;') and texto.rstrip().endswith('COMMIT;'))
        self.assertIn("Un LOGIN de Usuarios ya existe", texto)
        self.assertIn('GRANT vec_usuarios_ejecutor_interno TO vcl_a WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;', texto)
        self.assertNotIn('INSERT', texto.upper())
        self.assertNotIn('configuracion_origen_consumos_v1', texto)

    def test_la_fila_de_origen_solo_se_lee(self):
        for superficie, canal in usuarios.CANAL.items():
            consulta = usuarios.consulta_login_origen(superficie).decode()
            self.assertTrue(consulta.startswith('SELECT login_nombre FROM'))
            self.assertIn(f"vec_usuarios.preferencias.consultar.{canal}.v1", consulta)

    def test_enciende_selectores_y_respeta_un_false(self):
        entorno = usuarios.entorno_usuarios({'VEC_CT_DATABASE_URL': 'x'})
        self.assertEqual(entorno['VEC_USUARIOS_PREFERENCIAS_ENABLED'], 'true')
        self.assertEqual(entorno['VEC_USUARIOS_IMAGEN_ENABLED'], 'true')
        apagado = usuarios.entorno_usuarios({'VEC_USUARIOS_IMAGEN_ENABLED': 'false'})
        self.assertEqual(apagado['VEC_USUARIOS_IMAGEN_ENABLED'], 'false')
        sin_preferencias = usuarios.entorno_usuarios({'VEC_USUARIOS_PREFERENCIAS_ENABLED': 'false'})
        self.assertEqual(sin_preferencias['VEC_USUARIOS_PREFERENCIAS_ENABLED'], 'false')
        self.assertEqual(sin_preferencias['VEC_USUARIOS_IMAGEN_ENABLED'], 'false')

    def test_sin_material_original_no_se_enciende(self):
        self.assertFalse(usuarios.disponible(None))
        with tempfile.TemporaryDirectory() as carpeta:
            self.assertFalse(usuarios.disponible(carpeta))
            (Path(carpeta) / 'identidad').mkdir()
            for superficie in usuarios.SUPERFICIES:
                (Path(carpeta) / usuarios.FICHERO.format(superficie)).write_text('{}')
            self.assertTrue(usuarios.disponible(carpeta))

    def test_la_copia_lo_llama_en_los_dos_modos(self):
        codigo = Path(__file__).with_name('copia_local.py').read_text()
        self.assertEqual(codigo.count('preparar_usuarios(self, args.rrhh_original, guardar, sha256)'), 2)


if __name__ == '__main__':
    unittest.main()
