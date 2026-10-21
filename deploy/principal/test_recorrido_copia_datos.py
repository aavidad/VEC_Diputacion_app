"""Contratos de preparación sin Docker ni PostgreSQL."""

import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parent))
from recorrido_copia import preparar_roles, preparar_datos
from recorrido_copia import datos, roles
from copia_local_conexiones import GRUPOS


class CopiaFalsa:
    def __init__(self, raiz, fuente):
        self.raiz = raiz
        self.nombre = "vec-copia-l-prueba"
        self.estado = raiz / "estado.json"
        self.material = raiz / "material"
        self.material.mkdir()
        (self.material / "identidad").mkdir()
        self._estado = {"nombre": self.nombre, "raiz": str(raiz), "fuente": str(fuente),
                        "sid_copia": "1234567890123456789", "preparada": False, "material_modo": "nuevo", "puerto_pg": 15432,
                        "recorrido_completo": True}
        self.estado.write_text(json.dumps(self._estado))
        self.sentencias = []

    def leer(self):
        return json.loads(self.estado.read_text())

    def existe(self):
        return True

    def sql_bytes(self, sql, log):
        self.sentencias.append((sql.decode(), log))


class DatosCopiaTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.raiz = Path(self.tmp.name)
        self.repo = Path(__file__).resolve().parents[2]
        self.copia = CopiaFalsa(self.raiz, self.repo)

    def test_limpieza_no_se_ejecuta_en_una_copia_preparada_o_sin_recorrido(self):
        from recorrido_copia.limpieza_bolsa import preparar
        for cambio in ({'preparada': True}, {'recorrido_completo': False},
                       {'sid_copia': "1'; SELECT 1; --"}):
            with self.subTest(cambio=cambio):
                estado = self.copia.leer()
                original = dict(estado)
                estado.update(cambio)
                self.copia.estado.write_text(json.dumps(estado))
                with self.assertRaisesRegex(ValueError, 'copia nueva'):
                    preparar(self.copia)
                self.assertEqual(self.copia.sentencias, [])
                self.copia.estado.write_text(json.dumps(original))

    def test_limpieza_rechaza_otro_volcado_sin_enviar_sql(self):
        from recorrido_copia.limpieza_bolsa import preparar
        (self.raiz / 'fuentes.json').write_text(json.dumps({
            'principal-hz37-20261014.dump': '0' * 64}))
        with self.assertRaisesRegex(ValueError, 'lote versionado'):
            preparar(self.copia)
        self.assertEqual(self.copia.sentencias, [])

    def test_reglas_disponibles_antes_de_preparar_la_identidad_del_aspirante(self):
        from recorrido_copia import bolsa
        from copia_local_caso40 import REGLAS_BOLSA
        archivo = self.raiz / 'entorno_modulos.json'
        archivo.write_text(json.dumps({k: 'conexion' for k in GRUPOS}))
        with patch.object(bolsa, '_historico', return_value=lambda *_: {}):
            bolsa.preparar_bolsa(self.copia)
        entorno = json.loads(archivo.read_bytes())
        self.assertEqual(entorno['VEC_BOLSA_REGLAS_SOURCE_PATH'], str(self.repo / REGLAS_BOLSA))
        self.assertNotIn('VEC_BOLSA_PORTAL_CANDIDATO_ENABLED', entorno)

    def test_roles_antes_de_ad278_guardan_claves_privadas_tras_sql(self):
        class Resultado:
            stdout = b"vec_ct_origen\n"
        with patch.object(roles.subprocess, "run", return_value=Resultado()) as ejecutar:
            # La segunda consulta debe devolver otro LOGIN.
            ejecutar.side_effect = [Resultado(), type("R", (), {"stdout": b"vec_bolsa_origen\n"})()]
            preparar_roles(self.copia)
        self.assertEqual(len(self.copia.sentencias), 1)
        sql, log = self.copia.sentencias[0]
        self.assertIn("VALID UNTIL", sql)
        self.assertIn("INHERIT TRUE,SET FALSE", sql)
        self.assertIn("ALTER ROLE vec_ct_origen PASSWORD", sql)
        self.assertEqual(log, "datos-roles-previos.log")
        archivo = self.raiz / "roles-previos.json"
        self.assertEqual(archivo.stat().st_mode & 0o077, 0)
        self.assertEqual(len(json.loads(archivo.read_text())), 2)

    def test_roles_no_publican_claves_si_falla_sql(self):
        self.copia.sql_bytes = lambda *_: (_ for _ in ()).throw(RuntimeError("sql"))
        with patch.object(roles.subprocess, "run", side_effect=[
                type("R", (), {"stdout": b"vec_ct_origen\n"})(),
                type("R", (), {"stdout": b"vec_bolsa_origen\n"})()]):
            with self.assertRaises(RuntimeError):
                preparar_roles(self.copia)
        self.assertFalse((self.raiz / "roles-previos.json").exists())

    def test_datos_rechazan_volcado_distinto_antes_de_sql(self):
        (self.raiz / "fuentes.json").write_text(json.dumps({
            "principal-hz37-20261014.dump": "0" * 64}))
        with self.assertRaisesRegex(ValueError, "HZ37"):
            preparar_datos(self.copia)
        self.assertEqual(self.copia.sentencias, [])

    def test_rechaza_material_original_antes_de_sql(self):
        estado = self.copia.leer()
        estado["material_modo"] = "rrhh_original"
        self.copia.estado.write_text(json.dumps(estado))
        (self.raiz / "fuentes.json").write_text(json.dumps({
            "principal-hz37-20261014.dump": datos.HUELLA_HZ37}))
        with self.assertRaisesRegex(ValueError, "nuevo"):
            preparar_datos(self.copia)
        self.assertEqual(self.copia.sentencias, [])

    def test_datos_rechazan_origenes_incompletos_antes_de_sql(self):
        (self.raiz / "fuentes.json").write_text(json.dumps({
            "principal-hz37-20261014.dump": datos.HUELLA_HZ37}))
        (self.raiz / "conexiones.json").write_text("{}")
        (self.raiz / "roles-previos.json").write_text("{}")
        with self.assertRaisesRegex(ValueError, "orígenes"):
            preparar_datos(self.copia)
        self.assertEqual(self.copia.sentencias, [])

    def test_datos_con_material_nuevo_sustituyen_origenes_y_limpian_punteros(self):
        (self.raiz / "fuentes.json").write_text(json.dumps({
            "principal-hz37-20261014.dump": datos.HUELLA_HZ37}))
        conexiones = {clave: "postgresql://vcl_temporal_" + str(i) +
                      ":ficticia@localhost:15432/postgres?sslmode=verify-full"
                      for i, clave in enumerate(GRUPOS)}
        (self.raiz / "conexiones.json").write_text(json.dumps(conexiones))
        (self.raiz / "roles-previos.json").write_text(json.dumps({
            "VEC_CT_DATABASE_URL": {"login": "vec_ct_origen", "password": "ct-ficticia"},
            "VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL": {
                "login": "vec_bolsa_origen", "password": "bolsa-ficticia"}}))
        for archivo in ("identidad.json", "intervencion.json"):
            (self.copia.material / "identidad" / archivo).write_text("{}")
        with patch.object(datos, "preparar_intentos"), patch.object(datos, "preparar_bolsa"):
            preparar_datos(self.copia)
        sql, log = self.copia.sentencias[0]
        self.assertEqual(log, "datos-copia-sql.log")
        self.assertIn("DROP ROLE vcl_temporal_0", sql)
        self.assertIn("system_identifier::text", sql)
        self.assertIn(self.copia.leer()["sid_copia"], sql)
        self.assertIn("rolvaliduntil=NULL", sql)
        self.assertIn("WHERE rolname='vcl_temporal_7'", sql)
        self.assertIn("DISABLE TRIGGER ALL", sql)
        self.assertIn("ENABLE TRIGGER ALL", sql)
        self.assertNotIn("clave:capacidad:ct:desarrollo:v2", sql)
        nuevo = json.loads((self.raiz / "entorno_modulos.json").read_text())
        self.assertIn("vec_ct_origen", nuevo["VEC_CT_DATABASE_URL"])
        self.assertIn("sslmode=verify-full", nuevo["VEC_CT_DATABASE_URL"])
        self.assertEqual((self.raiz / "entorno_modulos.json").stat().st_mode & 0o077, 0)
        self.assertEqual(json.loads(self.copia.estado.read_text())["entorno_modulos_sha256"],
                         datos._huella(self.raiz / "entorno_modulos.json"))


if __name__ == "__main__":
    unittest.main()
