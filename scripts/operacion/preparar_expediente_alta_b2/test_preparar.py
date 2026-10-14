import json
import hashlib
import os
import subprocess
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

from preparar import Corte, RPT_PUBLICA, comando_catalogo_faltante, comando_psql, comprobar_ct215, comprobar_fuente_bolsa_ct, comprobar_relevo_cese, data_http, datos_ejercicio, guardar, cargar, http, lectura_db, necesidad_contexto_centro, necesidad_rpt_publicada, nueva_peticion, nuevo_numero, preparar_rpt, vacante_personal_compatible
from preparar_configuracion import leer_conexiones_privadas


class CanonTrasReinicio(unittest.TestCase):
    def test_huella_rpt_publicada_procede_del_fichero_real(self):
        publicado = necesidad_rpt_publicada()
        self.assertEqual(publicado["rpt_catalogo_huella_sha256"], hashlib.sha256(RPT_PUBLICA.read_bytes()).hexdigest())

    def test_ct215_exige_huellas_prosrc_y_frontera_instalada(self):
        from preparar import CT215_PROSRC
        filas = [f"{nombre}|{huella}|11|t|vec_contratacion_temporal_propietario|t|t"
                 for nombre, huella in sorted(CT215_PROSRC.items())]
        with patch("preparar.lectura_db", return_value=filas):
            comprobar_ct215(None)
        with patch("preparar.lectura_db", return_value=[fila.replace("|11|", "|10|") for fila in filas]):
            with self.assertRaisesRegex(Corte, "firma o ACL"):
                comprobar_ct215(None)

    def test_peticion_v3_conserva_necesidad_original_tras_respuesta_incierta(self):
        datos = {"plaza_codigo": "0033", "organica_codigo": "520", "funcional_codigo": "920",
                 "proyecto_gasto_codigo": "2026-520", "porcentaje_financiacion": "100"}
        rpt = {"puesto_codigo": "9", "rpt_catalogo_ref": "rpt:publicada", "rpt_catalogo_huella_sha256": "b" * 64,
               "denominacion": "ADMINISTRATIVO"}
        obligatorios = ["numero_personas", *datos, "puesto_codigo", "rpt_catalogo_ref", "rpt_catalogo_huella_sha256"]
        contexto = {"actor": {"puede_presentar": True}, "catalogos": {
            "esquema": "vec.contratacion_temporal.catalogos_alta.v2", "necesidades": {
                "referencia": "catalogo:ct:necesidades", "version": 3, "huella_sha256": "a" * 64,
                "jornada_referencia_minutos": 2250, "causas": [{"clave": "vacante", "campos_obligatorios": obligatorios}]}},
            "vinculos_rpt": {"categorias": [{"categoria_ref": "categoria:rpt:administrativo", "rpt_categoria_clave": "administrativo"}],
                             "centros": [{"centro_ref": "centro-520", "rpt_centro_codigo": "520"}]}}
        with tempfile.TemporaryDirectory() as folder:
            ruta = Path(folder) / "estado.json"
            args = SimpleNamespace(estado=ruta)
            state = {"esquema": "vec.preparar-alta-b2.v1", "operaciones": {},
                     "valores": {"rpt_necesidad": rpt, "datos_ejercicio": datos}}
            with patch("preparar.data_http", side_effect=[contexto, Corte("sin respuesta")]):
                with self.assertRaises(Corte):
                    nueva_peticion(args, state)
            congelado = cargar(ruta)
            cuerpo = congelado["operaciones"]["presentar"]["cuerpo"]
            self.assertEqual(cuerpo["esquema"], "vec.ct.alta_necesidad.v1")
            self.assertEqual(set(cuerpo["necesidad"]), {"esquema", "catalogo_ref", "catalogo_version", "catalogo_huella_sha256", "causa_clave", "jornada_minutos", "campos"})
            self.assertEqual(cuerpo["necesidad"]["campos"]["rpt_catalogo_huella_sha256"], "b" * 64)
            with patch("preparar.data_http", return_value={"peticion_ref": "peticion:centro:uno"}) as post:
                nueva_peticion(args, congelado)
            self.assertEqual(post.call_count, 1)
            self.assertEqual(post.call_args.args[4], cuerpo)
            self.assertEqual(necesidad_contexto_centro(contexto, cuerpo["solicitud"], rpt, datos), cuerpo["necesidad"])

    def test_vacante_de_plaza_ajena_no_habilita_peticion(self):
        pagina = {"pagina": {"cobertura": "completa", "vacantes": [{
            "codigo_plaza_fuente": "2026/9999", "unidad_ref": "unidad:centro:520",
            "puesto_ref": "puesto:uno", "version_plantilla_ref": "version:uno",
            "version_rpt_ref": "version:rpt", "estado_cobertura": "vacante_sin_ocupacion"}]}}
        with patch("preparar.data_http", return_value=pagina):
            with self.assertRaisesRegex(Corte, "plaza del ejercicio"):
                vacante_personal_compatible(None, "0033", __import__("datetime").date(2026, 12, 1))

    def test_vacante_personal_de_otro_puesto_no_pasa_anclaje_historico(self):
        state = {"operaciones": {}, "valores": {}}
        with patch("preparar.rpt_compatible", return_value=(1, "org:uno", ("puesto:9", "plaza:33"))), \
             patch("preparar.necesidad_rpt_publicada", return_value={"puesto_codigo": "9"}), \
             patch("preparar.datos_ejercicio", return_value={"plaza_codigo": "0033"}), \
             patch("preparar.vacante_personal_compatible", return_value={"puesto_ref": "puesto:otro", "plaza_ref": "plaza:33"}):
            with self.assertRaisesRegex(Corte, "no coincide"):
                preparar_rpt(None, state)

    def test_cinco_datos_privados_no_aceptan_campos_extra(self):
        with tempfile.TemporaryDirectory() as folder:
            ruta = Path(folder) / "necesidad_ejemplo_b2.json"
            args = SimpleNamespace(config=Path(folder) / "personal-b2/servidor.json")
            valores = {"plaza_codigo": "0033", "organica_codigo": "520", "funcional_codigo": "920",
                       "proyecto_gasto_codigo": "2026-520", "porcentaje_financiacion": "100"}
            ruta.write_text(json.dumps(valores))
            ruta.chmod(0o600)
            self.assertEqual(datos_ejercicio(args), valores)
            ruta.write_text(json.dumps({**valores, "rpt_catalogo_ref": "otro"}))
            with self.assertRaisesRegex(Corte, "cinco códigos"):
                datos_ejercicio(args)

    def test_fuente_nativa_exige_b110_ad261_bolsa_unica_y_m10(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            config = root / "config/personal-b2/servidor.json"
            config.parent.mkdir(parents=True)
            config.write_text('{"personal_b2":{"organismo_ref":"org:uno"}}')
            (root / "config/cliente_m10_ct.json").write_text(json.dumps({
                "version": 1, "organismo_ref": "org:uno", "origen": "https://127.0.0.1:18446",
                "directorio": "/privado", "autoridad_ca": "/privado/ca",
                "certificado_cliente": "/privado/cert", "clave_cliente": "/privado/key"}))
            launcher = root / "arrancar.sh"
            launcher.write_text("export VEC_CT_BOLSA_ORGANIZACION_SOURCE_PATH=/vec-incorporacion/cliente_m10_ct.json\n")
            args = SimpleNamespace(config=config, launcher=launcher, origen="https://127.0.0.1:18446")
            with patch("preparar.lectura_db", return_value=["1|1|1"]) as lectura:
                comprobar_fuente_bolsa_ct(args)
            self.assertIn("sustitucion_bolsa", lectura.call_args.args[1])
            self.assertIn("bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk", lectura.call_args.args[1])
            with patch("preparar.lectura_db", return_value=["1|1|2"]):
                with self.assertRaisesRegex(Corte, "Bolsa Administrativo"):
                    comprobar_fuente_bolsa_ct(args)

    def test_relevo_cese_exige_selectores_y_dsn_sin_imprimirlo(self):
        lanzador = """export VEC_CT_SEGUIMIENTO_CESE_ENABLED=true
export VEC_BOLSA_CESE_CT_ENABLED=true
export VEC_BOLSA_CESE_CT_DATABASE_URL='postgres://privado:55461/base?sslmode=verify-full'
"""
        comprobar_relevo_cese(lanzador)
        with self.assertRaisesRegex(Corte, "relevo de cese"):
            comprobar_relevo_cese(lanzador + "export VEC_BOLSA_CONTRATOS_CT_INTERVALO=0\n")
        with self.assertRaisesRegex(Corte, "relevo de cese"):
            comprobar_relevo_cese(lanzador + "export VEC_BOLSA_CONTRATOS_CT_INTERVALO=0s\n")
        with self.assertRaisesRegex(Corte, "varias asignaciones"):
            comprobar_relevo_cese(lanzador + "export VEC_BOLSA_CESE_CT_ENABLED=false\n")
        with self.assertRaisesRegex(Corte, "DSN nominal"):
            comprobar_relevo_cese(lanzador.replace("VEC_BOLSA_CESE_CT_DATABASE_URL", "OTRA_VARIABLE"))

    def test_dsn_importado_solo_desde_archivo_privado_explicito(self):
        lanzador = "export VEC_CT_SEGUIMIENTO_CESE_ENABLED=true\nexport VEC_BOLSA_CESE_CT_ENABLED=true\nsource conexiones.sh\n"
        with tempfile.TemporaryDirectory() as carpeta:
            ruta = Path(carpeta) / "conexiones.sh"
            ruta.write_text("export VEC_BOLSA_CESE_CT_DATABASE_URL='postgres://nominal:55461/base?sslmode=verify-full'\n")
            ruta.chmod(0o600)
            comprobar_relevo_cese(lanzador, leer_conexiones_privadas(ruta))
            with self.assertRaisesRegex(Corte, "varias asignaciones"):
                comprobar_relevo_cese(lanzador + "export VEC_BOLSA_CESE_CT_DATABASE_URL='postgres://otro:55461/base?sslmode=verify-full'\n", leer_conexiones_privadas(ruta))
            for dsn in ("postgres://nominal:55461/base?host=%2Fvar%2Frun%2Fpostgresql&sslmode=verify-full",
                        "host=/var/run/postgresql port=55461 sslmode=verify-full",
                        "dbname=postgres sslmode=verify-full"):
                with self.subTest(dsn=dsn), self.assertRaisesRegex(Corte, "TLS verify-full"):
                    comprobar_relevo_cese(lanzador, f"export VEC_BOLSA_CESE_CT_DATABASE_URL='{dsn}'\n")
            ruta.chmod(0o644)
            with self.assertRaisesRegex(ValueError, "0600"):
                leer_conexiones_privadas(ruta)

    def test_error_http_malformado_conserva_corte_nominal(self):
        for error in (None, "cadena", [], 4):
            with self.subTest(error=error), patch("preparar.http", return_value=(422, {"error": error})):
                with self.assertRaisesRegex(Corte, "422 respuesta_invalida"):
                    data_http(None, "RRHH", "POST", "/api/vec/prueba", {})

    def test_numero_usa_politica_y_cuatro_fuentes_sin_repetir_consulta(self):
        from datetime import date
        year = date.today().year
        politica = {"numero_expediente_moad": {"patron": rf"^{year}/[1-9][0-9]{{0,9}}$", "ejemplo": f"{year}/5487"}}
        state = {"valores": {}}
        with patch("preparar.data_http", return_value=politica) as catalogo, \
             patch("preparar.lectura_db", return_value=["15"]) as lectura:
            self.assertEqual(nuevo_numero(None, state, ""), f"{year}/5487")
            self.assertEqual(nuevo_numero(None, state, ""), f"{year}/5487")
        self.assertEqual(catalogo.call_count, 1)
        self.assertEqual(lectura.call_count, 1)
        sql = lectura.call_args.args[1]
        for tabla in ("expediente_alta", "identidad_reserva_alta", "numeracion_anual_asignada", "candidatura_alta_tecnica"):
            self.assertIn(tabla, sql)

    def test_entorno_hostil_no_cambia_destinos_locales(self):
        with tempfile.TemporaryDirectory() as folder:
            material = Path(folder)
            (material / "ca").mkdir()
            (material / "ca/ca.crt").write_text("CA de prueba")
            args = SimpleNamespace(motor="docker", contenedor="pg-prueba",
                                   material=material, origen="https://127.0.0.1:29445",
                                   estado=material / "privado/estado.json")
            with patch.dict(os.environ, {"CURL_HOME": "/tmp/curl-ajeno",
                                      "DOCKER_HOST": "tcp://remoto:2375", "DOCKER_CONTEXT": "remoto"}):
                self.assertEqual(comando_psql(args)[:6],
                                 ["docker", "--host", "unix:///var/run/docker.sock", "exec", "-i", "pg-prueba"])
                with patch("preparar.subprocess.run", return_value=subprocess.CompletedProcess([], 0, b"BEGIN\nSET\n1\nROLLBACK\n", b"")) as sql_call:
                    self.assertEqual(lectura_db(args, "SELECT 1"), ["1"])
                self.assertNotIn("DOCKER_HOST", sql_call.call_args.kwargs["env"])
                self.assertNotIn("DOCKER_CONTEXT", sql_call.call_args.kwargs["env"])
                with patch("preparar.mTLS", return_value=(material / "actor.crt", material / "actor.key")), \
                     patch("preparar.subprocess.run", return_value=subprocess.CompletedProcess([], 0, b'{"data":{}}\n200', b"")) as call:
                    self.assertEqual(http(args, "RRHH", "GET", "/api/vec/prueba")[0], 200)
                argv = call.call_args.args[0]
                self.assertEqual(argv[:4], ["curl", "-q", "--noproxy", "*"])
                self.assertEqual(argv[-1], "https://127.0.0.1:29445/api/vec/prueba")
                with patch("preparar.mTLS", return_value=(material / "actor.crt", material / "actor.key")):
                    impreso = comando_catalogo_faltante(args, "modalidad:vacante")
                self.assertIn("curl -q --noproxy '*'", impreso)


if __name__ == "__main__":
    unittest.main()
