"""Pruebas focales de las conexiones de una copia nueva, sin Docker."""

import importlib.util
import json
from datetime import datetime, timedelta, timezone
from pathlib import Path
import re
import tempfile
import unittest
from urllib.parse import parse_qs, urlsplit


RUTA = Path(__file__).with_name("copia_local_conexiones.py")
ESPEC = importlib.util.spec_from_file_location("copia_local_conexiones", RUTA)
modulo = importlib.util.module_from_spec(ESPEC)
ESPEC.loader.exec_module(modulo)


class CopiaFalsa:
    nombre = "vec-copia-l-segunda"

    def __init__(self, raiz):
        self.raiz = raiz
        self.material = raiz / "material"
        self.material.mkdir(mode=0o700)
        (self.material / "ca").mkdir(mode=0o700)
        (self.material / "ca" / "ca.crt").write_text("CA sintética\n")
        self.sql_ejecutado = []

    def leer(self):
        return {"nombre": self.nombre, "revision": "a" * 40,
                "preparada": False, "puerto_pg": 55481}

    def sql_bytes(self, contenido, log):
        self.sql_ejecutado.append((contenido.decode(), log))


class TestConexiones(unittest.TestCase):
    def setUp(self):
        self.temporal = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporal.cleanup)
        raiz = Path(self.temporal.name) / "instancia"
        raiz.mkdir(mode=0o700)
        self.copia = CopiaFalsa(raiz)

    def test_nueva_instancia_genera_identidades_y_env_privado(self):
        env = modulo.preparar(self.copia)
        self.assertEqual(set(env), set(modulo.GRUPOS))
        self.assertEqual(len(env), 20)
        sql, log = self.copia.sql_ejecutado[0]
        self.assertEqual(log, "conexiones.log")
        self.assertTrue(sql.startswith("BEGIN;"))
        self.assertTrue(sql.endswith("COMMIT;\n"))
        self.assertEqual(sql.count("CREATE ROLE "), len(env))
        self.assertEqual(sql.count("WITH ADMIN FALSE, INHERIT TRUE, SET FALSE"), len(env) - 1)
        self.assertEqual(sql.count("WITH ADMIN FALSE, INHERIT TRUE, SET TRUE"), 1)
        self.assertIn("Arco AD3 heredado distinto", sql)
        self.assertIn("Propietario AD3 distinto", sql)
        self.assertIn("Propietario AD3 con membresía", sql)
        self.assertIn("Propietario AD3 con miembro ajeno", sql)
        self.assertIn("Arcos AD3 duplicados", sql)
        self.assertIn("Migrador AD3 ya tiene miembros", sql)
        self.assertIn("Rol AD3 INHERIT con membresías", sql)
        self.assertIn("DO $atributo$ BEGIN IF EXISTS", sql)
        self.assertIn("EXECUTE 'ALTER ROLE vec_autorizacion_atestada_v3_propietario NOINHERIT'", sql)
        self.assertIn("EXECUTE 'ALTER ROLE vec_autorizacion_atestada_v3_migrador NOINHERIT'", sql)
        self.assertIn("DO $arco$ BEGIN IF NOT EXISTS", sql)
        self.assertIn("EXECUTE 'GRANT vec_autorizacion_atestada_v3_propietario "
                      "TO vec_autorizacion_atestada_v3_migrador "
                      "WITH ADMIN FALSE, INHERIT FALSE, SET TRUE'", sql)
        self.assertIn("Arco AD3 no restaurado", sql)
        self.assertLess(sql.index("Rol AD3 INHERIT con membresías"), sql.index("DO $atributo$"))
        self.assertLess(sql.index("Arco AD3 heredado distinto"), sql.index("DO $atributo$"))
        self.assertLess(sql.index("DO $atributo$"), sql.index("DO $arco$"))
        self.assertLess(sql.index("DO $arco$"), sql.index("CREATE ROLE "))
        fechas = re.findall(r"VALID UNTIL '([^']+)'", sql)
        self.assertEqual(len(fechas), len(env))
        self.assertEqual(len(set(fechas)), 1)
        caducidad = datetime.strptime(fechas[0], "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)
        self.assertLess(datetime.now(timezone.utc) + timedelta(days=89), caducidad)
        self.assertLess(caducidad, datetime.now(timezone.utc) + timedelta(days=91))
        self.assertIn("Grupo nominal ausente o sin CONNECT seguro", sql)
        self.assertIn("Un LOGIN nominal ya existe", sql)
        usuarios = set()
        for variable, dsn in env.items():
            parsed = urlsplit(dsn)
            self.assertEqual((parsed.scheme, parsed.hostname, parsed.port, parsed.path),
                             ("postgresql", "localhost", 55481, "/postgres"))
            self.assertEqual(parse_qs(parsed.query),
                             {"sslmode": ["verify-full"],
                              "sslrootcert": [str(self.copia.material / "ca" / "ca.crt")]})
            self.assertRegex(parsed.username, r"^vcl_l_segunda_[0-9a-f]{12}_[0-9]{2}$")
            self.assertGreaterEqual(len(parsed.password), 32)
            usuarios.add(parsed.username)
            set_option = "TRUE" if variable == modulo.GOBIERNO else "FALSE"
            self.assertIn("GRANT " + modulo.GRUPOS[variable] + " TO " + parsed.username +
                          " WITH ADMIN FALSE, INHERIT TRUE, SET " + set_option, sql)
        self.assertEqual(len(usuarios), len(env))
        archivo = self.copia.raiz / modulo.RESULTADO
        self.assertEqual(archivo.stat().st_mode & 0o777, 0o600)
        self.assertEqual(self.copia.raiz.stat().st_mode & 0o777, 0o700)
        texto = archivo.read_text()
        self.assertEqual(json.loads(texto), env)
        self.assertNotIn("PASSWORD '", texto)
        with self.assertRaisesRegex(ValueError, "ya existen"):
            modulo.preparar(self.copia)
        self.assertEqual(len(self.copia.sql_ejecutado), 1)

    def test_estado_parcial_y_resultado_enlazado_se_rechazan_antes_de_sql(self):
        original = self.copia.leer
        self.copia.leer = lambda: {**original(), "preparada": True}
        with self.assertRaisesRegex(ValueError, "no es nueva"):
            modulo.preparar(self.copia)
        self.copia.leer = original
        (self.copia.raiz / modulo.RESULTADO).symlink_to(self.copia.material / "ca" / "ca.crt")
        with self.assertRaisesRegex(ValueError, "ya existen"):
            modulo.preparar(self.copia)
        self.assertEqual(self.copia.sql_ejecutado, [])

    def test_una_instancia_distinta_no_reutiliza_login(self):
        otra_raiz = Path(self.temporal.name) / "segunda"
        otra_raiz.mkdir(mode=0o700)
        otra = CopiaFalsa(otra_raiz)
        otra.nombre = "vec-copia-l-otra"
        self.assertTrue(set(urlsplit(d).username for d in modulo.preparar(self.copia).values())
                        .isdisjoint(urlsplit(d).username for d in modulo.preparar(otra).values()))

    def test_error_de_sql_no_publica_configuracion(self):
        def fallar(contenido, log):
            raise RuntimeError("SQL denegada")
        self.copia.sql_bytes = fallar
        with self.assertRaisesRegex(RuntimeError, "SQL denegada"):
            modulo.preparar(self.copia)
        self.assertFalse((self.copia.raiz / modulo.RESULTADO).exists())


if __name__ == "__main__":
    unittest.main()
