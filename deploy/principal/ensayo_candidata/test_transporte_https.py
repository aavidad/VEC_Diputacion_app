"""Contrato del transporte mTLS del ensayo, con conexión y certificado simulados."""

import importlib.util
from pathlib import Path
import ssl
import tempfile
import unittest
from types import SimpleNamespace
from unittest.mock import patch


ROOT = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("hz39_transporte", ROOT / "transporte_https.py")
transporte = importlib.util.module_from_spec(spec)
spec.loader.exec_module(transporte)
spec_runner = importlib.util.spec_from_file_location("hz39_recorrer_transporte", ROOT / "recorrer.py")
runner = importlib.util.module_from_spec(spec_runner)
spec_runner.loader.exec_module(runner)


class Solicitud:
    method = "POST"
    post_data_buffer = b'{"consulta":true}'

    def __init__(self, url="https://127.0.0.1:9443/api/consulta"):
        self.url = url

    def all_headers(self):
        return {"Host": "127.0.0.1:9443", "Cookie": "sesion=x", "X-Rol": "rrhh"}


class Ruta:
    def __init__(self, solicitud=None):
        self.request = solicitud or Solicitud()
        self.abortada = False
        self.entregada = None

    def abort(self):
        self.abortada = True

    def fulfill(self, **kwargs):
        self.entregada = kwargs


class Conexion:
    instancias = []
    estado = 200
    cuerpo = b'{"estado":"real"}'
    longitud = None
    cabeceras = [("Content-Type", "application/json")]

    def __init__(self, host, puerto, *, context, timeout):
        self.datos = (host, puerto, context, timeout)
        self.peticion = None
        self.cerrada = False
        self.__class__.instancias.append(self)

    def request(self, metodo, ruta, body, headers):
        self.peticion = (metodo, ruta, body, headers)

    def getresponse(self):
        cuerpo = self.cuerpo
        return SimpleNamespace(
            status=self.estado,
            length=self.longitud,
            read=lambda n: cuerpo[:n],
            getheaders=lambda: self.cabeceras,
        )

    def close(self):
        self.cerrada = True


class TransporteFocal(unittest.TestCase):
    def setUp(self):
        Conexion.instancias.clear()
        Conexion.estado = 200
        Conexion.cuerpo = b'{"estado":"real"}'
        Conexion.longitud = None
        Conexion.cabeceras = [("Content-Type", "application/json")]
        self.origen = "https://127.0.0.1:9443"
        self.identidad = {"origin": self.origen, "pfxPath": "/privado/actor.p12", "passphrase": "secreto"}
        self.espacio = tempfile.TemporaryDirectory()
        self.addCleanup(self.espacio.cleanup)
        ca = Path(self.espacio.name) / "ca.pem"
        ca.write_text("CA ficticia")
        self.cliente = transporte.TransporteHTTPS(self.origen, ca, self.identidad, Path(self.espacio.name))
        self.cliente.contexto = object()

    def test_peticion_real_conserva_autoridad_cookies_y_rol_sin_anadir_connection(self):
        ruta = Ruta(Solicitud(self.origen + "/api/consulta?q=1"))
        with patch.object(transporte.http.client, "HTTPSConnection", Conexion):
            self.cliente.responder(ruta, self.origen)
        conexion = Conexion.instancias[0]
        metodo, destino, cuerpo, cabeceras = conexion.peticion
        self.assertEqual((metodo, destino, cuerpo), ("POST", "/api/consulta?q=1", Solicitud.post_data_buffer))
        self.assertEqual(cabeceras["Host"], "127.0.0.1:9443")
        self.assertEqual(cabeceras["Cookie"], "sesion=x")
        self.assertEqual(cabeceras["X-Rol"], "rrhh")
        self.assertEqual(cabeceras["Accept-Encoding"], "identity")
        self.assertNotIn("connection", {k.lower() for k in cabeceras})
        self.assertEqual(ruta.entregada["body"], Conexion.cuerpo)
        self.assertEqual(ruta.entregada["status"], 200)
        self.assertTrue(conexion.cerrada)

    def test_connection_original_se_conserva_y_otro_origen_no_sale(self):
        solicitud = Solicitud()
        solicitud.all_headers = lambda: {"Host": "127.0.0.1:9443", "Connection": "close"}
        with patch.object(transporte.http.client, "HTTPSConnection", Conexion):
            self.cliente.responder(Ruta(solicitud), self.origen)
            self.assertEqual(Conexion.instancias[0].peticion[3]["Connection"], "close")
            ajena = Ruta(Solicitud("https://127.0.0.1:9444/api/consulta"))
            self.cliente.responder(ajena, self.origen)
        self.assertTrue(ajena.abortada)
        self.assertEqual(len(Conexion.instancias), 1)

    def test_redireccion_y_respuesta_grande_se_abortan_sin_reintento(self):
        with patch.object(transporte.http.client, "HTTPSConnection", Conexion):
            Conexion.estado = 302
            ruta = Ruta()
            self.cliente.responder(ruta, self.origen)
            self.assertTrue(ruta.abortada)
            Conexion.estado = 200
            Conexion.longitud = transporte.MAX_CUERPO + 1
            ruta = Ruta()
            self.cliente.responder(ruta, self.origen)
            self.assertTrue(ruta.abortada)
        self.assertEqual(len(Conexion.instancias), 2)
        self.assertTrue(all(c.cerrada for c in Conexion.instancias))

    def test_recuperacion_bloquea_antes_del_transporte(self):
        ejecucion = runner.Ejecucion(Path(self.espacio.name), {"valores": {}})
        ejecucion.solo_lectura = True
        ejecucion.actual = {"red_fallida": []}
        ruta = Ruta()
        with patch.object(self.cliente, "responder") as enviar:
            ejecucion.filtrar(ruta, self.origen, enviar)
        enviar.assert_not_called()
        self.assertTrue(ruta.abortada)
        self.assertEqual(ejecucion.actual["red_fallida"][0]["codigo"], "efecto_bloqueado_en_recuperacion")

    def test_cabeceras_repetidas_no_se_pierden_silenciosamente(self):
        Conexion.cabeceras = [("Set-Cookie", "uno=1"), ("set-cookie", "dos=2")]
        with patch.object(transporte.http.client, "HTTPSConnection", Conexion):
            ruta = Ruta()
            self.cliente.responder(ruta, self.origen)
        self.assertTrue(ruta.abortada)
        self.assertIsNone(ruta.entregada)

    def test_ca_obligatoria_contexto_verificado_y_temporal_limpiado(self):
        contexto = SimpleNamespace(check_hostname=True, verify_mode=ssl.CERT_REQUIRED, load_cert_chain=lambda **_: None)

        def exportar(*_args, **kwargs):
            Path(_args[0][_args[0].index("-out") + 1]).write_bytes(b"PEM ficticio")
            return SimpleNamespace(returncode=0)

        self.cliente.contexto = None
        with (
            patch.object(transporte.subprocess, "run", side_effect=exportar) as openssl,
            patch.object(transporte.ssl, "create_default_context", return_value=contexto) as crear,
        ):
            with self.cliente as cliente:
                temporal = Path(cliente.temporal.name)
                self.assertEqual((temporal / "actor.pem").stat().st_mode & 0o777, 0o600)
                self.assertTrue(cliente.contexto.check_hostname)
                self.assertEqual(cliente.contexto.verify_mode, ssl.CERT_REQUIRED)
            self.assertFalse(temporal.exists())
        self.assertEqual(crear.call_args.kwargs["cafile"], str(self.cliente.ca))
        self.assertEqual(openssl.call_args.kwargs["input"], b"secreto\n")
        self.assertEqual(openssl.call_args.kwargs["stdout"], transporte.subprocess.DEVNULL)
        self.assertNotIn("secreto", " ".join(openssl.call_args.args[0]))


if __name__ == "__main__":
    unittest.main()
