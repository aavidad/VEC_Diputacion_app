import json
import tempfile
import unittest
import hashlib
from pathlib import Path

import recorrer


class RecorridoFirmaTest(unittest.TestCase):
    def test_salida_rechaza_git_worktree_permisos_y_enlaces(self):
        with tempfile.TemporaryDirectory() as tmp:
            raiz = Path(tmp)
            privado = raiz / "privado"
            privado.mkdir(mode=0o700)
            recorrer.guardar_privado(privado / "valido.json", b"sintetico")
            self.assertEqual((privado / "valido.json").read_bytes(), b"sintetico")
            for marcador in ("directorio", "fichero"):
                repo = raiz / marcador
                repo.mkdir(mode=0o700)
                if marcador == "directorio":
                    (repo / ".git").mkdir()
                else:
                    (repo / ".git").write_text("gitdir: otro", encoding="utf-8")
                hijo = repo / "capturas"
                hijo.mkdir(mode=0o700)
                with self.assertRaises(OSError):
                    recorrer.guardar_privado(hijo / "rechazada.png", b"sintetico")
                self.assertFalse((hijo / "rechazada.png").exists())
            bare = raiz / "bare"
            bare.mkdir(mode=0o700)
            (bare / "HEAD").write_text("ref: refs/heads/main", encoding="utf-8")
            (bare / "config").write_text("[core]\nbare = true", encoding="utf-8")
            (bare / "objects").mkdir()
            with self.assertRaises(OSError):
                recorrer.guardar_privado(bare / "rechazada.png", b"sintetico")
            compartido = raiz / "compartido"
            compartido.mkdir(mode=0o755)
            with self.assertRaises(OSError):
                recorrer.guardar_privado(compartido / "rechazada.png", b"sintetico")
            enlace = raiz / "enlace"
            enlace.symlink_to(privado, target_is_directory=True)
            with self.assertRaises(OSError):
                recorrer.guardar_privado(enlace / "rechazada.png", b"sintetico")
            with self.assertRaises(OSError):
                recorrer.guardar_privado(privado / ".." / "rechazada.png", b"sintetico")

    def test_capturas_del_primer_corte_son_privadas_y_no_sobrescriben(self):
        class Pagina:
            def set_viewport_size(self, dimensiones):
                self.ancho = dimensiones["width"]

            def screenshot(self, *, full_page):
                return b"captura sintetica"

            def evaluate(self, codigo):
                return {"ancho": self.ancho, "contenido": self.ancho}

        with tempfile.TemporaryDirectory() as tmp:
            carpeta = Path(tmp)
            escritorio = carpeta / "1440.png"
            movil = carpeta / "390.png"
            args = recorrer.argumentos(["--captura-escritorio", str(escritorio),
                                       "--captura-movil", str(movil)])
            informe = {"corte": "propuesta"}
            recorrer.capturar_corte(Pagina(), args, informe)
            for ruta in (escritorio, movil):
                self.assertEqual(ruta.stat().st_mode & 0o777, 0o600)
                self.assertEqual(ruta.read_bytes(), b"captura sintetica")
            self.assertTrue(informe["escritorio_1440"]["captura_guardada"])
            self.assertTrue(informe["movil_390"]["sin_desbordamiento"])
            segundo = {"corte": "propuesta"}
            recorrer.capturar_corte(Pagina(), args, segundo)
            self.assertFalse(segundo["escritorio_1440"]["captura_guardada"])
            self.assertEqual(segundo["escritorio_1440"]["error"], "FileExistsError")
            self.assertEqual(escritorio.read_bytes(), b"captura sintetica")

    def test_sin_clon_no_ejecuta_navegador(self):
        with self.assertRaises(recorrer.Corte) as error:
            recorrer.validar_entrada(recorrer.argumentos([]))
        self.assertEqual(error.exception.paso, "precondiciones")

    def test_binario_que_no_coincide_con_inventario_corta_antes_de_chrome(self):
        with tempfile.TemporaryDirectory() as tmp:
            carpeta = Path(tmp)
            binario = carpeta / "vec-server"
            binario.write_bytes(b"binario sintetico")
            binario.chmod(0o700)
            certificado = carpeta / "cliente.crt"
            clave = carpeta / "cliente.key"
            certificado.write_bytes(b"certificado sintetico")
            clave.write_bytes(b"clave sintetica")
            inventario = carpeta / "inventario.json"
            inventario.write_text(json.dumps({"clon": "local", "datos": "sinteticos",
                "hitos": ["H3", "H4", "H5"], "origen": "https://localhost:8443",
                "binario_sha256": hashlib.sha256(b"otro binario").hexdigest()}), encoding="utf-8")
            a = recorrer.argumentos(["--entorno", str(inventario), "--binario", str(binario),
                "--origen", "https://localhost:8443", "--certificado", str(certificado),
                "--clave", str(clave), "--expediente-ref", "expediente:sintetico",
                "--version-propuesta", "7"])
            with self.assertRaises(recorrer.Corte) as error:
                recorrer.validar_entrada(a)
            self.assertEqual(error.exception.paso, "precondiciones")
            self.assertIn("huella", error.exception.motivo)

    def test_solo_acepta_propuesta_en_version_indicada(self):
        expediente = {"resumen": {"expediente_ref": "expediente:sintetico", "version": 8,
                                  "fase_clave": "nombramiento", "estado_clave": "en_curso"},
                      "hitos": [{"accion_clave": "otro"}] * 6 +
                      [{"accion_clave": "registrar_propuesta_formalizacion", "version_expediente": 7},
                       {"accion_clave": "otro", "version_expediente": 8}]}
        self.assertEqual(recorrer.comprobar_propuesta(expediente, "expediente:sintetico", 7)["version_propuesta"], 7)
        with self.assertRaises(recorrer.Corte):
            recorrer.comprobar_propuesta(expediente, "expediente:ajeno", 7)
        with self.assertRaises(recorrer.Corte):
            recorrer.comprobar_propuesta(expediente, "expediente:sintetico", 8)

    def test_recibo_de_firma_exige_verificacion_y_no_eficacia(self):
        recibo = {"expediente_ref": "expediente:sintetico", "documento": "informe_definitivo",
                  "resultado": "firmado", "firma_eficaz": False, "firma_verificada": True,
                  "registrada_en": "2026-09-29T20:00:00Z", "paso_orden": 1,
                  "recibo_ref": "recibo:prueba", "verificacion": {"estado": "valida", "motivo": "verificada",
                  "firmado_sha256": "a" * 64}}
        self.assertEqual(recorrer.resumen_firma(recibo, "expediente:sintetico", "informe_definitivo")["recibo_ref"], "recibo:prueba")
        with self.assertRaises(recorrer.Corte):
            recorrer.resumen_firma({**recibo, "firma_eficaz": True}, "expediente:sintetico", "informe_definitivo")
        with self.assertRaises(recorrer.Corte):
            recorrer.resumen_firma({**recibo, "verificacion": {**recibo["verificacion"], "estado": "indeterminada"}},
                                   "expediente:sintetico", "informe_definitivo")

    def test_recibo_de_propuesta_sale_de_consulta_separada(self):
        datos = {"estado": {"expediente_ref": "expediente:sintetico", "propuestas": [
            {"version_resultante": 7, "recibo_ref": "recibo:propuesta", "confirmada_en": "2026-09-06T00:00:00Z"}]}}
        self.assertEqual(recorrer.comprobar_recibo_propuesta(datos, "expediente:sintetico", 7)["recibo_ref"], "recibo:propuesta")
        with self.assertRaises(recorrer.Corte):
            recorrer.comprobar_recibo_propuesta(datos, "expediente:sintetico", 8)
        with self.assertRaises(recorrer.Corte):
            recorrer.comprobar_recibo_propuesta({"estado": {**datos["estado"], "propuestas": datos["estado"]["propuestas"] * 2}},
                                               "expediente:sintetico", 7)

    def test_recuperacion_exige_huellas_y_recibo_iguales(self):
        firma = {"recibo_ref": "recibo:prueba", "registrada_en": "2026-09-29T20:00:00Z",
                 "estado": "firmado", "paso_orden": 1, "firma_ref": "firma:origen", "firmado_sha256": "b" * 64}
        recuperada = {"recibo_ref": "recibo:prueba", "registrada_en": "2026-09-29T20:00:00Z",
                      "estado": "firmado", "paso_orden": 1}
        previo = {"expediente_ref": "expediente:sintetico", "propuesta": {"version_propuesta": 7},
                  "pdf": {"informe-definitivo": {"sha256": "a" * 64}},
                  "firma": firma}
        actual = {**previo, "firma": None, "firma_recuperada": recuperada}
        with tempfile.TemporaryDirectory() as tmp:
            ruta = Path(tmp) / "anterior.json"
            ruta.write_text(json.dumps(previo), encoding="utf-8")
            self.assertTrue(recorrer.comparar_recuperacion(actual, ruta))
            with self.assertRaises(recorrer.Corte):
                recorrer.comparar_recuperacion({**actual, "pdf": {}}, ruta)
            with self.assertRaises(recorrer.Corte):
                recorrer.comparar_recuperacion({**actual, "firma_recuperada": None}, ruta)
            with self.assertRaises(recorrer.Corte):
                recorrer.comparar_recuperacion({**actual, "firma_recuperada": {
                    **recuperada, "registrada_en": "2026-09-29T20:00:01Z"}}, ruta)
            self.assertEqual(recorrer.firma_recuperada({"firma_eficaz": False, "documentos": [
                {"documento": "informe_definitivo", "pasos": [{"orden": 1, "estado": "firmado",
                "recibo_ref": "recibo:prueba", "registrada_en": "2026-09-29T20:00:00Z"}]}]},
                "recibo:prueba"), recuperada)

    def test_redireccion_local_a_otro_puerto_se_corta(self):
        class Respuesta:
            status = 302
            url = "https://localhost:8443/portal-empleado/"
            headers = {"location": "https://localhost:8444/otro"}

        class Ruta:
            request = type("Peticion", (), {"url": "https://localhost:8443/portal-empleado/"})()
            abortada = False
            entregada = False

            def fetch(self, *, max_redirects, timeout):
                self.max_redirects = max_redirects
                return Respuesta()

            def abort(self):
                self.abortada = True

            def fulfill(self, *, response):
                self.entregada = True

        ruta = Ruta()
        recorrer.limitar_origen(ruta, "https://localhost:8443")
        self.assertEqual(ruta.max_redirects, 0)
        self.assertTrue(ruta.abortada)
        self.assertFalse(ruta.entregada)

        externa = Ruta()
        externa.request = type("Peticion", (), {"url": "https://localhost:8444/otro"})()
        recorrer.limitar_origen(externa, "https://localhost:8443")
        self.assertTrue(externa.abortada)
        self.assertFalse(hasattr(externa, "max_redirects"))

    def test_websocket_solo_autofirma_explicita(self):
        class Ruta:
            def __init__(self, url):
                self.url = url
                self.conectada = False
                self.cerrada = False

            def connect_to_server(self):
                self.conectada = True

            def close(self):
                self.cerrada = True

        permitida = Ruta("wss://127.0.0.1:63117")
        recorrer.limitar_websocket(permitida, True)
        self.assertTrue(permitida.conectada)
        denegada = Ruta("wss://127.0.0.1:63117")
        recorrer.limitar_websocket(denegada, False)
        self.assertTrue(denegada.cerrada)
        otro_puerto = Ruta("wss://127.0.0.1:63118")
        recorrer.limitar_websocket(otro_puerto, True)
        self.assertTrue(otro_puerto.cerrada)


if __name__ == "__main__":
    unittest.main()
