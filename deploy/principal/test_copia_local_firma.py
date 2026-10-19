"""La orden firma: manifiesto privado, etapas en orden, entorno y verificador propio."""
import datetime as dt
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('firma', Path(__file__).with_name('copia_local_firma.py'))
firma = importlib.util.module_from_spec(spec)
spec.loader.exec_module(firma)


def escribir(ruta, contenido, modo=0o600):
    ruta.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    ruta.write_text(contenido)
    ruta.chmod(modo)
    return ruta


class Base(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.raiz = Path(self.tmp.name).resolve() / 'copia'
        self.raiz.mkdir(mode=0o700)
        self.estado = {'puerto_pg': 55481, 'puerto_app': 18481, 'puerto_smtp': 11081}
        self.copia = SimpleNamespace(nombre='vec-copia-l-firma', raiz=self.raiz, material=self.raiz / 'material',
                                     estado=self.raiz / 'estado.json', leer=lambda: dict(self.estado),
                                     entorno_modulos=lambda: {'VEC_DOCUMENTOS_ENABLED': 'true'})
        self.privada = Path(self.tmp.name).resolve() / 'privado'
        self.privada.mkdir(mode=0o700)
        self.binario = escribir(self.privada / 'verificador', '#!/bin/sh\n', 0o700)

    def tearDown(self):
        self.tmp.cleanup()

    def guardar(self, ruta, valor):
        self.estado = dict(valor)

    def manifiesto(self, **cambios):
        etapa = escribir(self.privada / 'etapa', '#!/bin/sh\nexit 0\n', 0o700)
        datos = {'version': 1, 'etapas': [{'nombre': 'admin', 'orden': [str(etapa)]}],
                 'verificador': {'binario': str(self.binario), 'puerto': 63181,
                                 'binario_sha256': hashlib.sha256(self.binario.read_bytes()).hexdigest()}}
        datos.update(cambios)
        return escribir(self.privada / 'manifiesto.json', json.dumps(datos))


class ManifiestoTest(Base):
    def test_acepta_un_manifiesto_privado_completo(self):
        manifiesto = firma.leer_manifiesto(self.manifiesto(), self.estado)
        self.assertEqual(manifiesto['verificador']['puerto'], 63181)

    def test_rechaza_claves_desconocidas(self):
        with self.assertRaises(ValueError):
            firma.leer_manifiesto(self.manifiesto(extra=1), self.estado)

    def test_rechaza_manifiesto_legible_por_otros(self):
        ruta = self.manifiesto()
        ruta.chmod(0o644)
        with self.assertRaises(ValueError):
            firma.leer_manifiesto(ruta, self.estado)

    def test_rechaza_etapa_con_ejecutable_relativo_o_repetida(self):
        for etapas in ([{'nombre': 'a', 'orden': ['etapa']}],
                       [{'nombre': 'a', 'orden': [str(self.binario)]}, {'nombre': 'a', 'orden': [str(self.binario)]}],
                       [{'nombre': 'A b', 'orden': [str(self.binario)]}]):
            with self.subTest(etapas=etapas), self.assertRaises(ValueError):
                firma.leer_manifiesto(self.manifiesto(etapas=etapas), self.estado)

    def test_rechaza_carpeta_superior_modificable_por_otros(self):
        ruta = self.manifiesto()
        self.privada.chmod(0o777)
        try:
            with self.assertRaises(ValueError):
                firma.leer_manifiesto(ruta, self.estado)
        finally:
            self.privada.chmod(0o700)

    def test_rechaza_puerto_de_la_copia_y_huella_distinta(self):
        verificador = {'binario': str(self.binario), 'puerto': 18481,
                       'binario_sha256': hashlib.sha256(self.binario.read_bytes()).hexdigest()}
        with self.assertRaises(ValueError):
            firma.leer_manifiesto(self.manifiesto(verificador=verificador), self.estado)
        verificador.update(puerto=63181, binario_sha256='0' * 64)
        with self.assertRaises(ValueError):
            firma.leer_manifiesto(self.manifiesto(verificador=verificador), self.estado)


class EtapasTest(Base):
    def etapa(self, nombre, cuerpo):
        ruta = escribir(self.privada / nombre, '#!/bin/sh\n' + cuerpo + '\n', 0o700)
        return {'nombre': nombre, 'orden': [str(ruta)]}

    def test_ejecuta_en_orden_y_no_repite_lo_hecho(self):
        traza = self.privada / 'traza'
        etapas = [self.etapa('uno', f'echo uno >> {traza}'), self.etapa('dos', f'echo dos >> {traza}')]
        self.assertEqual(firma.ejecutar_etapas(self.copia, etapas), ['uno', 'dos'])
        self.assertEqual(firma.ejecutar_etapas(self.copia, etapas), [])
        self.assertEqual(traza.read_text().split(), ['uno', 'dos'])

    def test_repite_la_etapa_si_cambia_su_ejecutable(self):
        traza = self.privada / 'traza'
        etapa = self.etapa('uno', f'echo a >> {traza}')
        firma.ejecutar_etapas(self.copia, [etapa])
        self.etapa('uno', f'echo b >> {traza}')
        self.assertEqual(firma.ejecutar_etapas(self.copia, [etapa]), ['uno'])
        self.assertEqual(traza.read_text().split(), ['a', 'b'])

    def test_el_plazo_termina_la_etapa_y_sus_hijos(self):
        etapa = self.etapa('lenta', 'sleep 37.13 & sleep 37.13')
        with patch.object(firma, 'PLAZO_ETAPA', 0.3), self.assertRaises(ValueError):
            firma.ejecutar_etapas(self.copia, [etapa])
        self.assertFalse((self.raiz / 'firma-etapas/lenta.hecha').exists())
        vivos = []
        for proc in Path('/proc').iterdir():
            try:
                if (proc / 'cmdline').read_bytes() == b'sleep\x0037.13\x00':
                    vivos.append(proc.name)
            except OSError:
                pass
        self.assertEqual(vivos, [])

    def test_una_etapa_fallida_detiene_las_siguientes(self):
        traza = self.privada / 'traza'
        etapas = [self.etapa('mal', 'exit 3'), self.etapa('despues', f'echo despues >> {traza}')]
        with self.assertRaises(ValueError):
            firma.ejecutar_etapas(self.copia, etapas)
        self.assertFalse(traza.exists())
        self.assertFalse((self.raiz / 'firma-etapas/mal.hecha').exists())

    def test_las_etapas_no_heredan_el_entorno_del_operador(self):
        salida = self.privada / 'entorno'
        with patch.dict(os.environ, {'VEC_SECRETO_PRUEBA': 'no'}):
            firma.ejecutar_etapas(self.copia, [self.etapa('entorno', f'env > {salida}')])
        entorno = salida.read_text()
        self.assertNotIn('VEC_SECRETO_PRUEBA', entorno)
        self.assertIn(f'VEC_COPIA_RAIZ={self.raiz}', entorno)


class EntornoTest(Base):
    def test_compone_las_ocho_variables_de_firma(self):
        recursos = self.raiz / 'material/identidad/firma-recursos'
        variables = firma.entorno_firma(self.copia, recursos, 63181)
        self.assertEqual(len(variables), 8)
        self.assertEqual(variables['VEC_FIRMA_VERIFICACION_URL'], 'https://127.0.0.1:63181')
        self.assertTrue(variables['VEC_CT_PLANTILLAS_SOURCE_PATH'].endswith('ct_plantillas_documentos.v4.json'))

    def test_entorno_privado_y_huella_en_el_estado(self):
        firma.componer_entorno(self.copia, {'VEC_FIRMA_VERIFICACION_ENABLED': 'true'},
                               {'puerto': 63181, 'binario_sha256': 'a' * 64}, self.guardar)
        ruta = self.raiz / 'entorno_modulos.json'
        self.assertEqual(ruta.stat().st_mode & 0o777, 0o600)
        self.assertEqual(json.loads(ruta.read_text()),
                         {'VEC_DOCUMENTOS_ENABLED': 'true', 'VEC_FIRMA_VERIFICACION_ENABLED': 'true'})
        self.assertEqual(self.estado['entorno_modulos_sha256'], hashlib.sha256(ruta.read_bytes()).hexdigest())
        self.assertEqual(self.estado['verificador'], {'puerto': 63181, 'binario_sha256': 'a' * 64})

    def test_sin_modulos_no_compone(self):
        self.copia.entorno_modulos = lambda: {}
        with self.assertRaises(ValueError):
            firma.componer_entorno(self.copia, {}, {'puerto': 1, 'binario_sha256': ''}, self.guardar)


class RecursosTest(Base):
    def preparar(self, **cambios):
        recursos = self.raiz / 'material/identidad/firma-recursos'
        for nombre in firma.RECURSOS:
            escribir(recursos / nombre, nombre)
        recursos.chmod(0o700)
        plan = recursos / 'plan-firma-publicado.json'
        config = {'plan': {'archivo': str(plan), 'referencia': 'ct.plan', 'version': 1,
                           'huella_sha256': 'd' * 64},
                  'crl_path': str(recursos / 'clientes.crl'), 'firmantes': [{}], 'admisiones': [{}],
                  'retirada_en': (dt.datetime.now(dt.timezone.utc) + dt.timedelta(days=30)).strftime('%Y-%m-%dT%H:%M:%SZ')}
        config.update(cambios)
        escribir(self.raiz / 'material/identidad/firma-vec.json', json.dumps(config))

    def test_acepta_configuracion_coherente(self):
        self.preparar()
        self.assertTrue(firma.recursos_firma(self.copia).is_dir())

    def test_rechaza_plan_ajeno_o_configuracion_retirada(self):
        for cambio in ({'crl_path': '/otra/clientes.crl'}, {'retirada_en': '2020-01-01T00:00:00Z'},
                       {'plan': {'archivo': '/otro/plan.json', 'referencia': 'r', 'version': 1, 'huella_sha256': 'd' * 64}},
                       {'admisiones': []}):
            with self.subTest(cambio=cambio):
                self.preparar(**cambio)
                with self.assertRaises(ValueError):
                    firma.recursos_firma(self.copia)


class VerificadorTest(Base):
    def test_sin_firma_no_arranca_ni_para_nada(self):
        with patch.object(firma.subprocess, 'Popen') as popen, patch.object(firma.os, 'kill') as kill:
            self.assertFalse(firma.arrancar_verificador(self.copia, self.guardar, lambda p: None))
            self.assertFalse(firma.parar_verificador(self.copia))
        popen.assert_not_called()
        kill.assert_not_called()

    def test_no_para_un_proceso_ajeno(self):
        inicio = Path(f'/proc/{os.getpid()}/stat').read_text().split()[21]
        self.estado['verificador'] = {'puerto': 63181, 'pid': os.getpid(), 'inicio_proceso': inicio}
        with patch.object(firma.os, 'kill') as kill:
            self.assertFalse(firma.parar_verificador(self.copia))
        kill.assert_not_called()

    def test_la_publicacion_v4_no_reenvia_la_salida_de_error(self):
        escribir(self.raiz / 'publicar-plantillas', '', 0o700)
        fallo = SimpleNamespace(returncode=1, stdout=b'', stderr=b'postgres://postgres:clave@localhost')
        with patch.object(firma, 'contrasena_pg', return_value='b' * 64), \
                patch.object(firma.subprocess, 'run', side_effect=[SimpleNamespace(returncode=0), fallo]):
            with self.assertRaises(ValueError) as error:
                firma.publicar_plantillas_v4(self.copia, Path('/repo'), 'go', {})
        self.assertNotIn('clave', str(error.exception))

    def test_la_credencial_del_verificador_no_va_en_los_argumentos(self):
        carpeta = self.raiz / 'verificador'
        binario = escribir(carpeta / 'grxfirma-verificador', '#!/bin/sh\n', 0o700)
        for nombre in ('ca.crt', 'tls.crt', 'tls.key', 'crl/clientes.crl'):
            escribir(carpeta / nombre, 'x')
        escribir(carpeta / 'token', 'c' * 64)
        self.estado['verificador'] = {'puerto': 63181,
                                      'binario_sha256': hashlib.sha256(binario.read_bytes()).hexdigest()}
        proceso = SimpleNamespace(pid=os.getpid(), poll=lambda: None)
        with patch.object(firma, 'comprobar_material_verificador'), \
                patch.object(firma, 'salud_verificador', side_effect=[OSError, None]), \
                patch.object(firma.subprocess, 'Popen', return_value=proceso) as popen, \
                patch.object(firma.time, 'sleep'):
            self.assertTrue(firma.arrancar_verificador(self.copia, self.guardar, lambda p: None))
        argumentos, opciones = popen.call_args
        self.assertNotIn('c' * 64, ' '.join(argumentos[0]))
        self.assertIn('-rest-solo-verificacion', argumentos[0])
        self.assertEqual(opciones['env']['GRXFIRMA_REST_TOKEN'], 'c' * 64)
        self.assertEqual(self.estado['verificador']['pid'], os.getpid())


if __name__ == '__main__':
    unittest.main()
