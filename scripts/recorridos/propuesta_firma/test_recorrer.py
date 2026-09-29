import json
import tempfile
import unittest
from pathlib import Path

import recorrer


class RecorridoFirmaTest(unittest.TestCase):
    def test_sin_clon_no_ejecuta_navegador(self):
        with self.assertRaises(recorrer.Corte) as error:
            recorrer.validar_entrada(recorrer.argumentos([]))
        self.assertEqual(error.exception.paso, "precondiciones")

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
        previo = {"expediente_ref": "expediente:sintetico", "propuesta": {"version_propuesta": 7},
                  "pdf": {"informe-definitivo": {"sha256": "a" * 64}},
                  "firma": {"recibo_ref": "recibo:prueba"}}
        with tempfile.TemporaryDirectory() as tmp:
            ruta = Path(tmp) / "anterior.json"
            ruta.write_text(json.dumps(previo), encoding="utf-8")
            self.assertTrue(recorrer.comparar_recuperacion(previo, ruta))
            with self.assertRaises(recorrer.Corte):
                recorrer.comparar_recuperacion({**previo, "pdf": {}}, ruta)
            with self.assertRaises(recorrer.Corte):
                recorrer.comparar_recuperacion({**previo, "firma": None}, ruta)

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
