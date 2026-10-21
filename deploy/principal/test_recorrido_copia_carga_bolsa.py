"""Contrato de la carga CONVOCA con transporte HTTPS simulado."""

import base64
import hashlib
import json
import os
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from recorrido_copia import carga_bolsa


class ContextoFalso:
    def __init__(self, carpeta, fuente):
        self.raiz = carpeta
        self.carpeta = carpeta / 'recorrido'
        self.carpeta.mkdir(mode=0o700)
        self.fuente = fuente
        self.copia = self
        self.comprobaciones = 0

    def comprobar_fuente(self):
        self.comprobaciones += 1

    def leer(self, ruta=None):
        if ruta is None:
            return {'puerto_app': 8443}
        return json.loads(Path(ruta).read_bytes())

    def guardar(self, ruta, valor):
        ruta = Path(ruta)
        ruta.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        raw = (json.dumps(valor, ensure_ascii=False, separators=(',', ':')) + '\n').encode()
        if ruta.exists():
            if ruta.read_bytes() != raw:
                raise ValueError('Evidencia anterior divergente')
            return False
        descriptor = os.open(ruta, os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW, 0o600)
        with os.fdopen(descriptor, 'wb') as salida:
            salida.write(raw)
        return True

    @staticmethod
    def sha(ruta):
        return hashlib.sha256(Path(ruta).read_bytes()).hexdigest()


class Respuesta:
    def __init__(self, estado, datos):
        self.status = estado
        self.datos = json.dumps(datos).encode()

    def read(self, limite):
        return self.datos[:limite]


class Conexion:
    def __init__(self, respuestas, peticiones, host, puerto, context, timeout):
        assert host == 'localhost' and puerto == 8443 and timeout == 30
        self.respuestas = respuestas
        self.peticiones = peticiones

    def request(self, metodo, ruta, body, headers):
        assert metodo == 'POST' and headers['Content-Type'] == 'application/json'
        assert headers['Accept'] == 'application/json' and 'Authorization' not in headers
        self.peticiones.append((ruta, json.loads(body)))

    def getresponse(self):
        return self.respuestas.pop(0)

    def close(self):
        pass


class CargaBolsaTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.fuente = Path(__file__).resolve().parents[2]
        cls.xls = (cls.fuente / carga_bolsa.FUENTE).read_bytes()
        cls.huella = hashlib.sha256(cls.xls).hexdigest()

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.c = ContextoFalso(Path(self.temp.name), self.fuente)
        self.peticiones = []

    def vista(self, **cambios):
        base = {'esquema': 'vec.bolsa.rrhh.carga_convoca.vista_previa.v1',
                'huella_sha256': self.huella, 'nombre_fichero': 'resumen.xls',
                'filas_leidas': 2, 'aceptadas': 2, 'rechazadas': 0, 'bloqueo': '',
                'filas': [
                    {'numero': 4, 'estado': 'aceptada', 'posicion': 1,
                     'nombre': 'María', 'primer_apellido': 'García',
                     'segundo_apellido': 'López', 'errores': [], 'avisos': []},
                    {'numero': 5, 'estado': 'aceptada', 'posicion': 2,
                     'nombre': 'Luis', 'primer_apellido': 'Pérez',
                     'segundo_apellido': '', 'errores': [], 'avisos': []}]}
        return {'data': dict(base, **cambios)}

    def recibo(self, reutilizada=False, **cambios):
        base = {'esquema': 'vec.bolsa.rrhh.carga_convoca.recibo.v1',
                'huella_sha256': self.huella, 'acta_ref': carga_bolsa._acta(self.huella),
                'bolsa_ref': 'bolsa:administrativo:prueba', 'version_bolsa': 1,
                'reutilizada': reutilizada, 'acta_reutilizada': reutilizada,
                'filas_cargadas': 2, 'filas_excluidas': 0,
                'sustituye_a': [{'bolsa_ref': 'bolsa:anterior', 'version_bolsa': 1}],
                'auditoria_ref': 'auditoria:prueba', 'confirmada_en': '2026-10-21T01:00:00Z'}
        return {'data': dict(base, **cambios)}

    def ejecutar(self, respuestas):
        respuestas = list(respuestas)
        anteriores = len(self.peticiones)
        def conectar(host, puerto, context, timeout):
            return Conexion(respuestas, self.peticiones, host, puerto, context, timeout)
        with patch.object(carga_bolsa.ssl, 'create_default_context') as tls, \
             patch.object(carga_bolsa.http.client, 'HTTPSConnection', side_effect=conectar):
            resultado = carga_bolsa.preparar(self.c)
        self.assertEqual(tls.call_count, len(self.peticiones) - anteriores)
        tls.assert_any_call(cafile=str(self.c.raiz / 'material/ca/ca.crt'))
        self.assertEqual(tls.return_value.load_cert_chain.call_count,
                         len(self.peticiones) - anteriores)
        tls.return_value.load_cert_chain.assert_any_call(
            str(self.c.raiz / 'material/mtls/cliente.crt'),
            str(self.c.raiz / 'material/mtls/cliente.key'))
        return resultado

    def test_carga_valida_y_replay_con_mismo_cuerpo(self):
        primero = self.ejecutar([Respuesta(200, self.vista()),
                                 Respuesta(201, self.recibo())])
        segundo = self.ejecutar([Respuesta(200, self.vista()),
                                 Respuesta(200, self.recibo(reutilizada=True))])
        self.assertEqual(primero, segundo)
        self.assertEqual(set(primero), {'bolsa_ref', 'version', 'acta_ref', 'huella',
                                       'aspirante'})
        self.assertEqual(primero['aspirante'], {'fila_numero': 4,
                                                'nombre': 'María García López'})
        self.assertEqual(self.c.leer(self.c.carpeta / 'carga-bolsa/vista-previa.json')
                         ['aspirante'], primero['aspirante'])
        self.assertEqual(self.c.leer(self.c.carpeta / 'carga-bolsa/recibo.json')
                         ['sustituye_a'][0]['bolsa_ref'], 'bolsa:anterior')
        self.assertEqual(self.peticiones[0][1], self.peticiones[2][1])
        self.assertEqual(self.peticiones[1][1], self.peticiones[3][1])
        self.assertEqual(self.peticiones[0][1], {
            'nombre_fichero': 'resumen.xls', 'contenido_base64': base64.b64encode(self.xls).decode(),
            'categoria': 'administrativo'})
        self.assertEqual(self.peticiones[1][1]['excluir_filas_con_errores'], False)
        self.assertTrue((self.c.carpeta / 'carga-bolsa/replay.json').exists())
        self.assertGreater(self.c.comprobaciones, 4)

    def test_denegacion_e_incapacidad_impiden_confirmar(self):
        for estado in (401, 403, 503):
            with self.subTest(estado=estado):
                self.peticiones.clear()
                with self.assertRaises(ValueError):
                    self.ejecutar([Respuesta(estado, {'codigo': 'denegado'})])
                self.assertEqual([ruta for ruta, _ in self.peticiones], [carga_bolsa.VISTA])

    def test_vista_rechazada_impide_confirmar(self):
        with self.assertRaisesRegex(ValueError, 'Vista previa'):
            self.ejecutar([Respuesta(200, self.vista(rechazadas=1, aceptadas=1))])
        self.assertEqual(len(self.peticiones), 1)

    def test_primera_posicion_ausente_o_ambigua_impide_confirmar(self):
        filas = self.vista()['data']['filas']
        casos = ([dict(filas[0], posicion=2), filas[1]],
                 [filas[0], dict(filas[1], posicion=1)],
                 [dict(filas[0], avisos=['identidad_ambigua']), filas[1]],
                 [dict(filas[0], nombre='María\n'), filas[1]])
        for caso in casos:
            with self.subTest(filas=caso):
                self.peticiones.clear()
                with self.assertRaises(ValueError):
                    self.ejecutar([Respuesta(200, self.vista(filas=caso))])
                self.assertEqual([ruta for ruta, _ in self.peticiones], [carga_bolsa.VISTA])

    def test_recibo_de_otra_categoria_o_fichero_se_rechaza(self):
        for cambio in ({'acta_ref': 'acta:importacion-convoca:' + '0' * 64},
                       {'huella_sha256': '0' * 64}):
            with self.subTest(cambio=cambio):
                with self.assertRaisesRegex(ValueError, 'discordante'):
                    self.ejecutar([Respuesta(200, self.vista()),
                                   Respuesta(201, self.recibo(**cambio))])
                self.assertFalse((self.c.carpeta / 'carga-bolsa/recibo.json').exists())
                self.peticiones.clear()

    def test_replay_con_otra_bolsa_no_sustituye_recibo(self):
        self.ejecutar([Respuesta(200, self.vista()), Respuesta(201, self.recibo())])
        with self.assertRaisesRegex(ValueError, 'Replay CONVOCA distinto'):
            self.ejecutar([Respuesta(200, self.vista()),
                           Respuesta(200, self.recibo(reutilizada=True,
                                                      bolsa_ref='bolsa:administrativo:otra'))])
        guardado = self.c.leer(self.c.carpeta / 'carga-bolsa/recibo.json')
        self.assertEqual(guardado['bolsa_ref'], 'bolsa:administrativo:prueba')

    def test_respuesta_excesiva_se_rechaza(self):
        with self.assertRaisesRegex(ValueError, 'excesiva'):
            self.ejecutar([Respuesta(200, {'data': {'relleno': 'x' * carga_bolsa.MAX_RESPUESTA}})])
        self.assertEqual(len(self.peticiones), 1)


if __name__ == '__main__':
    unittest.main()
