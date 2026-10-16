"""Transporte mTLS real para las rutas interceptadas del ensayo HZ39."""

from __future__ import annotations

import http.client
import os
from pathlib import Path
import ssl
import subprocess
import tempfile
from urllib.parse import urlsplit


MAX_CUERPO = 16 << 20


class TransporteHTTPS:
    def __init__(self, origen: str, ca: Path, identidad: dict, salida: Path):
        destino = urlsplit(origen)
        if (
            destino.scheme != "https"
            or destino.hostname not in {"localhost", "127.0.0.1", "::1"}
            or not destino.port
            or destino.path not in {"", "/"}
            or destino.query or destino.fragment
            or destino.username or destino.password
        ):
            raise ValueError("origen_https_loopback_invalido")
        if identidad["origin"] != origen:
            raise ValueError("certificado_de_otro_origen")
        self.origen = origen
        self.host = destino.hostname
        self.puerto = destino.port
        self.ca = Path(ca).expanduser().resolve(strict=True)
        if not self.ca.is_file():
            raise ValueError("ca_invalida")
        self.identidad = identidad
        self.salida = salida
        self.temporal = None
        self.contexto = None

    def __enter__(self):
        self.temporal = tempfile.TemporaryDirectory(prefix="mtls-", dir=self.salida)
        try:
            pem = Path(self.temporal.name) / "actor.pem"
            descriptor = os.open(pem, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
            os.close(descriptor)
            exportacion = subprocess.run(
                ["openssl", "pkcs12", "-in", self.identidad["pfxPath"],
                 "-out", str(pem), "-passin", "stdin", "-clcerts", "-nodes"],
                input=(self.identidad["passphrase"] + "\n").encode(),
                stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=10,
            )
            if exportacion.returncode or pem.stat().st_size > MAX_CUERPO:
                raise ValueError("certificado_mtls_no_exportable")
            contexto = ssl.create_default_context(cafile=str(self.ca))
            contexto.load_cert_chain(certfile=str(pem))
            self.contexto = contexto
            return self
        except BaseException:
            self.__exit__(None, None, None)
            raise

    def __exit__(self, _tipo, _valor, _traza):
        self.contexto = None
        if self.temporal is not None:
            self.temporal.cleanup()
            self.temporal = None

    def responder(self, route, origen: str):
        if origen != self.origen or self.contexto is None:
            route.abort()
            return
        solicitud = route.request
        url = urlsplit(solicitud.url)
        if f"{url.scheme}://{url.netloc}" != self.origen or url.fragment:
            route.abort()
            return
        ruta = url.path or "/"
        if url.query:
            ruta += "?" + url.query
        conexion = None
        try:
            cabeceras = {
                clave: valor for clave, valor in solicitud.all_headers().items()
                if clave.lower() != "accept-encoding"
            }
            cabeceras["Accept-Encoding"] = "identity"
            cuerpo = solicitud.post_data_buffer
            if cuerpo is not None and len(cuerpo) > MAX_CUERPO:
                raise ValueError("solicitud_demasiado_grande")
            conexion = http.client.HTTPSConnection(
                self.host, self.puerto, context=self.contexto, timeout=30
            )
            conexion.request(solicitud.method, ruta, body=cuerpo, headers=cabeceras)
            respuesta = conexion.getresponse()
            if 300 <= respuesta.status < 400:
                route.abort()
                return
            if respuesta.length is not None and respuesta.length > MAX_CUERPO:
                raise ValueError("respuesta_demasiado_grande")
            contenido = respuesta.read(MAX_CUERPO + 1)
            if len(contenido) > MAX_CUERPO:
                raise ValueError("respuesta_demasiado_grande")
            cabeceras_respuesta = respuesta.getheaders()
            if len(cabeceras_respuesta) != len({clave.lower() for clave, _ in cabeceras_respuesta}):
                raise ValueError("cabeceras_repetidas_no_representables")
            route.fulfill(status=respuesta.status, headers=dict(cabeceras_respuesta), body=contenido)
        except (OSError, ValueError, http.client.HTTPException):
            route.abort()
        finally:
            if conexion is not None:
                conexion.close()
