"""Pruebas focales de la semilla de Organización histórica (F1-103), sin Docker."""

import importlib.util
import json
from pathlib import Path
import sys
import tempfile
import unittest
from urllib.parse import parse_qs, urlsplit

AQUI = Path(__file__).parent
REPO = AQUI.parent.parent
sys.path.insert(0, str(AQUI))


def cargar(nombre):
    espec = importlib.util.spec_from_file_location(nombre, AQUI / (nombre + ".py"))
    modulo = importlib.util.module_from_spec(espec)
    espec.loader.exec_module(modulo)
    return modulo


oh = cargar("copia_local_organizacion_historica")
conexiones = cargar("copia_local_conexiones")

ORGANISMO = "org_0e4b1d56ba2e24fb8e4808cbc7abb660"
CONTEXTO = {"persona_ref": "per_" + "a" * 32, "cuenta_ref": "cta_" + "b" * 32,
            "procedencia_ref": "prc_" + "c" * 32, "procedencia_version": 1,
            "procedencia_huella_sha256": "d" * 64, "procedencia_autoridad": "no_autoritativa",
            "vigente_desde": "2026-10-01T00:00:00+00:00", "vigente_hasta": "2027-01-01T00:00:00+00:00"}
CORPORATIVO = {**{c: CONTEXTO[c] for c in ("persona_ref", "cuenta_ref")},
               "cuenta_version": 1, "persona_version": 1, "organizacion_ref": ORGANISMO,
               "organizacion_version": 1, "organizacion_procedencia_ref": "prc_" + "e" * 32,
               "organizacion_procedencia_version": 1, "organizacion_procedencia_huella_sha256": "f" * 64,
               "organizacion_procedencia_autoridad": "autoridad_maestra_acreditada",
               "superficie": "interna_corporativa", "uso": "consulta_rrhh",
               "procedencia_ref": "prc_" + "c" * 32, "procedencia_version": 1,
               "procedencia_huella_sha256": "d" * 64, "procedencia_autoridad": "no_autoritativa",
               "vigente_desde": "2026-10-01T00:00:00+00:00", "vigente_hasta": "2027-01-01T00:00:00+00:00"}
REFS = {"perfil": "prf_" + "1" * 32, "vinculo_contexto": "vca_" + "2" * 32,
        "vinculo_corporativo": "vcr_" + "3" * 32, "asignacion": "asg_" + "4" * 32}


class CopiaFalsa:
    nombre = "vec-copia-l-oh"

    def __init__(self, raiz):
        self.raiz = raiz
        self.material = raiz / "material"
        self.material.mkdir(mode=0o700)
        (self.material / "ca").mkdir(mode=0o700)
        (self.material / "ca" / "ca.crt").write_text("CA sintética\n")
        self.sql_ejecutado = []

    def leer(self):
        return {"nombre": self.nombre, "revision": "a" * 40, "preparada": False, "puerto_pg": 55481}

    def sql_bytes(self, contenido, log):
        self.sql_ejecutado.append((contenido.decode(), log))


class TestSemilla(unittest.TestCase):
    def test_concesion_de_la_fuente_versionada(self):
        c = oh.concesion(REPO)
        self.assertEqual(c["accion"], "personal.organizacion_historica.consultar")
        self.assertEqual(c["garantia_minima"], "sustancial")
        self.assertEqual(c["finalidades"], ["consultar_organizacion_historica"])
        self.assertEqual(len(c["campos_permitidos"]), 6)
        self.assertNotIn("obligaciones", c)
        self.assertEqual(oh.organismo(REPO), ORGANISMO)

    def test_motivos_estables_y_cuatro_usos(self):
        a, b = oh.motivos(), oh.motivos()
        self.assertEqual(a, b)
        huella, entradas, referencias = a
        self.assertRegex(huella, r"^[0-9a-f]{64}$")
        self.assertEqual(set(referencias), set(oh.USOS_MOTIVO))
        self.assertEqual(len({r["entrada_clave"] for r in referencias.values()}), 4)
        for entrada in entradas:
            self.assertRegex(entrada["clave"], r"^motivo_[0-9a-f]{32}$")
            self.assertRegex(entrada["vigente_desde"], r"\.\d{6}Z$")

    def test_sql_una_transaccion_con_ambitos_exactos(self):
        sql = oh.sql_semilla(REPO, CONTEXTO, CORPORATIVO, REFS, ORGANISMO)
        self.assertTrue(sql.startswith("BEGIN ISOLATION LEVEL SERIALIZABLE;"))
        self.assertTrue(sql.endswith("COMMIT;\n"))
        self.assertEqual(sql.count("COMMIT"), 1)
        self.assertIn('"clave":"organismo_ref","valores":["' + ORGANISMO + '"]', sql)
        self.assertIn('"clave":"unidad_clave","valores":["unidad:centro:520"]', sql)
        self.assertIn("vec_autorizacion.canon_asignacion_perfil_admin_v1(doc)", sql)
        self.assertIn("vec_autorizacion.canon_version_rol_admin_v1(rol)", sql)
        self.assertIn("interval '89 days'", sql)
        self.assertIn("'identidad:desarrollo:no-autoritativa'", sql)
        self.assertIn("semilla OH: el perfil ya existe", sql)
        self.assertIn("semilla OH: rol o asignación ya presentes", sql)
        self.assertIn("publicar_motivos_autorizacion_v2", sql)
        # El vínculo gemelo apunta al perfil nuevo, no al de Bolsa.
        self.assertIn("'" + REFS["vinculo_corporativo"] + "',1,'" + REFS["perfil"] + "',1,'"
                      + REFS["vinculo_contexto"] + "',1,'activo'", sql)
        self.assertNotIn("'obligaciones'", sql)

    def test_vinculo_de_otro_organismo_se_rechaza(self):
        ajeno = {**CORPORATIVO, "organizacion_ref": "org_" + "9" * 32}
        with self.assertRaisesRegex(ValueError, "organismo"):
            oh.sql_semilla(REPO, CONTEXTO, ajeno, REFS, ORGANISMO)

    def test_literal_rechaza_barra_nulo_y_dolar(self):
        self.assertEqual(oh._literal("o'x"), "'o''x'")
        for valor in ("a\\b", "a\x00b", "a$contexto$b"):
            with self.assertRaises(ValueError):
                oh._literal(valor)

    def test_escribir_privado_sin_sobrescribir(self):
        with tempfile.TemporaryDirectory() as temporal:
            ruta = Path(temporal) / "x.json"
            oh._escribir(ruta, "{}\n")
            self.assertEqual(ruta.stat().st_mode & 0o777, 0o600)
            with self.assertRaises(FileExistsError):
                oh._escribir(ruta, "{}\n")


class TestConexionesOH(unittest.TestCase):
    def setUp(self):
        self.temporal = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporal.cleanup)
        raiz = Path(self.temporal.name) / "instancia"
        raiz.mkdir(mode=0o700)
        self.copia = CopiaFalsa(raiz)

    def test_cuatro_login_con_su_grupo_y_origen_del_consumo(self):
        dsn = conexiones.preparar_organizacion_historica(self.copia)
        self.assertEqual(set(dsn), set(conexiones.GRUPOS_ORGANIZACION_HISTORICA))
        sql, log = self.copia.sql_ejecutado[0]
        self.assertEqual(log, "conexiones-oh.log")
        self.assertEqual(sql.count("CREATE ROLE "), 4)
        for campo, url in dsn.items():
            partes = urlsplit(url)
            self.assertEqual(parse_qs(partes.query)["sslmode"], ["verify-full"])
            self.assertIn("GRANT " + conexiones.GRUPOS_ORGANIZACION_HISTORICA[campo] + " TO "
                          + partes.username + " WITH ADMIN FALSE, INHERIT TRUE, SET FALSE", sql)
        personal = urlsplit(dsn["dsn_personal"]).username
        self.assertIn("configuracion_origen_consumos_v1", sql)
        self.assertIn("'" + personal + "'", sql.split("configuracion_origen_consumos_v1", 1)[1])
        self.assertNotIn("PASSWORD", json.dumps(dsn))

    def test_preparador_de_gobierno_efimero(self):
        url = conexiones.crear_preparador_gobierno(self.copia)
        sql = self.copia.sql_ejecutado[0][0]
        usuario = urlsplit(url).username
        self.assertIn("GRANT vec_autorizacion_atestada_v3_migrador TO " + usuario, sql)
        self.assertIn("VALID UNTIL", sql)
        conexiones.retirar_preparador_gobierno(self.copia)
        self.assertEqual(self.copia.sql_ejecutado[1][0], "DROP ROLE IF EXISTS " + usuario + ";\n")


if __name__ == "__main__":
    unittest.main()
