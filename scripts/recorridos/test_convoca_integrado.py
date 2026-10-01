"""Comprueba límites y evita convertir un cierre esperado en éxito funcional."""

from pathlib import Path
import argparse
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
import json
import sys
import threading
import unittest

from convoca_integrado import cargar_expectativas, clasificar, main, url_local


class LimitesRecorrido(unittest.TestCase):
    def test_destinos_fuera_de_loopback_y_credenciales_no_se_admiten(self):
        for url in ("https://example.com:443", "http://127.0.0.1:8080@ejemplo.invalid",
                    "http://usuario:clave@localhost:8080", "http://localhost:8080/?token=valor",
                    "http://localhost:8080/area-personal/", "http://localhost"):
            with self.subTest(url=url), self.assertRaises(ValueError):
                url_local(url)
        self.assertEqual(url_local("http://127.0.0.1:8080/"), "http://127.0.0.1:8080")
        self.assertEqual(url_local("http://[::1]:8080"), "http://[::1]:8080")

    def test_cierre_no_completa_flujo_y_solo_codigo_exacto_identifica_h6(self):
        expectativa = {"estado": "cerrada", "http": [403, 404, 503], "codigos_h6": ["h6_sin_fuente"]}
        for http in (403, 404, 503):
            resultado = clasificar("externo.mi_bolsa", {"http": http}, expectativa)
            self.assertTrue(resultado["coincide"])
            self.assertFalse(resultado["flujo_funcional_completado"])
            self.assertFalse(resultado["dependencia_h6_identificada"])
        confirmado = clasificar("externo.mi_bolsa", {"http": 503, "codigo": "h6_sin_fuente"}, expectativa)
        self.assertTrue(confirmado["dependencia_h6_identificada"])
        inesperado = clasificar("externo.mi_bolsa", {"http": 401}, expectativa)
        self.assertFalse(inesperado["coincide"])

    def test_http_200_invalido_o_inesperado_no_es_conforme(self):
        disponible = {"estado": "disponible"}
        invalido = clasificar("publico.listado", {"http": 200, "contrato_valido": False}, disponible)
        self.assertFalse(invalido["coincide"])
        cerrado = {"estado": "cerrada", "http": [403]}
        inesperado = clasificar("externo.mi_bolsa", {"http": 200, "contrato_valido": True}, cerrado)
        self.assertFalse(inesperado["coincide"])
        self.assertFalse(inesperado["flujo_funcional_completado"])

    def test_ejemplo_solo_declara_dependencias_auxiliares_concretas(self):
        capacidades, auxiliares = cargar_expectativas(Path(__file__).with_name("convoca_expectativas.json"))
        self.assertEqual(len(capacidades), 4)
        self.assertEqual(set(auxiliares), {"/api/vec/usuarios/area-personal/mi-imagen",
                                        "/api/vec/usuarios/area-personal/mis-correos"})


def ensayar_chrome() -> int:
    """Servidor de prueba privado y efímero; nunca es una fuente de producto."""
    parser = argparse.ArgumentParser()
    parser.add_argument("--ensayar-chrome", action="store_true", required=True)
    parser.add_argument("--salida", type=Path, required=True)
    parser.add_argument("--chrome", default="/usr/bin/google-chrome")
    args = parser.parse_args()
    raiz = Path(__file__).resolve().parents[2] / "web" / "static"
    if not (raiz / "bolsa" / "preparacion" / "index.html").is_file():
        parser.error("consumidor_preparacion_no_integrado")
    fixture = json.loads(Path(__file__).with_name("convoca_fixture.json").read_text())

    class Handler(SimpleHTTPRequestHandler):
        def __init__(self, *a, **kw):
            super().__init__(*a, directory=str(raiz), **kw)

        def log_message(self, *a):
            pass

        def do_GET(self):
            ruta = self.path.split("?")[0]
            if not ruta.startswith("/api/"):
                return super().do_GET()
            if self.server.externo:
                estado, datos = 403, {"error": {"codigo": "dependencia_sintetica_cerrada"}}
            elif ruta == "/api/publico/bolsa/categorias":
                estado, datos = 200, fixture["categorias"]
            elif ruta == "/api/publico/bolsa/convocatorias/sintetica-2026":
                estado, datos = 200, fixture["detalle"]
            elif ruta == "/api/publico/bolsa/convocatorias":
                estado, datos = 200, fixture["listado"]
            else:
                estado, datos = 404, {"error": {"codigo": "ruta_sintetica_desconocida"}}
            cuerpo = json.dumps(datos).encode()
            self.send_response(estado)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(cuerpo)))
            self.end_headers()
            self.wfile.write(cuerpo)

    servidores = []
    try:
        for externo in (False, True):
            servidor = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
            servidor.externo = externo
            servidores.append(servidor)
            threading.Thread(target=servidor.serve_forever, daemon=True).start()
        codigo = main(["--url-publico", f"http://127.0.0.1:{servidores[0].server_port}",
                       "--url-externo", f"http://127.0.0.1:{servidores[1].server_port}",
                       "--datos-sinteticos", "--tipo-ejecucion", "prueba_guion", "--preparacion-local",
                       "--chrome", args.chrome, "--salida", str(args.salida), "--timeout-ms", "30000"])
        informe = json.loads((args.salida / "resultado.json").read_text())
        if codigo != 2 or informe["flujo_funcional_completado"] or informe["registro_acreditado"] or informe["persistencia_acreditada"]:
            return 1
        return 0
    finally:
        for servidor in servidores:
            servidor.shutdown()
            servidor.server_close()


if __name__ == "__main__":
    if "--ensayar-chrome" in sys.argv:
        raise SystemExit(ensayar_chrome())
    unittest.main()
