"""El enlace por certificado conserva permisos y separación de superficies."""
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import copia_local_usuarios as usuarios
from test_copia_local_usuarios import original


class VinculoUsuariosTest(unittest.TestCase):
    def setUp(self):
        self.temporal = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporal.cleanup)
        self.material = Path(self.temporal.name)
        for carpeta in ('identidad', 'mtls'):
            (self.material / carpeta).mkdir(mode=0o700)
        self.cuenta, self.perfil = 'cta_' + 'a' * 32, 'prf_' + 'a' * 32
        self.interna = self.material / usuarios.FICHERO.format('interna')
        self.externa = self.material / usuarios.FICHERO.format('externa')
        for ruta, superficie, marca in ((self.interna, 'interna_corporativa', 'b'),
                                       (self.externa, 'externa_personal', 'c')):
            datos = original(superficie)
            datos['autoridad'] = 'no_autoritativo'
            datos['cuentas'] = [dict(certificado_sha256=marca * 64, sujeto='desarrollo:' + marca,
                                    cuenta_ref='cta_' + marca * 32, perfil_ref='prf_' + marca * 32)]
            self.escribir(ruta, datos)
        self.identidad = self.material / 'identidad/jefatura.json'
        self.escribir(self.identidad, dict(version=1, autoridad='no_autoritativo',
                      certificate_sha256='a' * 64, subject='desarrollo:jefatura',
                      roles=['ct_cargo_jefatura_firma_dev']))
        self.certificado = self.material / 'mtls/jefatura.crt'
        self.certificado.write_text('contenido sustituido por el doble de huella')
        self.certificado.chmod(0o600)
        self.huella = self.enterContext(patch.object(usuarios, 'huella_certificado', return_value='a' * 64))

    def escribir(self, ruta, valor):
        ruta.write_text(json.dumps(valor))
        ruta.chmod(0o600)

    def enlazar(self, **cambios):
        argumentos = dict(material=self.material, identidad='identidad/jefatura.json',
                          certificado='mtls/jefatura.crt', cuenta_ref=self.cuenta, perfil_ref=self.perfil)
        return usuarios.vincular_cuenta_interna(**(argumentos | cambios))

    def test_jefatura_se_anade_sin_modificar_conexiones_ni_cuenta_original(self):
        anterior = json.loads(self.interna.read_text())
        externa = self.externa.read_bytes()
        self.assertTrue(self.enlazar())
        nuevo = json.loads(self.interna.read_text())
        self.assertEqual(nuevo['cuentas'][:-1], anterior['cuentas'])
        self.assertEqual(nuevo['cuentas'][-1], dict(certificado_sha256='a' * 64,
                         sujeto='desarrollo:jefatura', cuenta_ref=self.cuenta, perfil_ref=self.perfil))
        self.assertEqual(nuevo | {'cuentas': anterior['cuentas']}, anterior)
        self.assertEqual(self.externa.read_bytes(), externa)
        self.assertEqual(self.interna.stat().st_mode & 0o777, 0o600)
        self.huella.assert_called_once_with(self.certificado)

    def test_repeticion_exacta_no_reescribe_configuracion(self):
        self.enlazar()
        antes = self.interna.read_bytes(), self.interna.stat().st_mtime_ns
        self.assertFalse(self.enlazar())
        self.assertEqual((self.interna.read_bytes(), self.interna.stat().st_mtime_ns), antes)

    def test_no_reasigna_un_certificado_ni_una_cuenta_ni_un_perfil(self):
        self.enlazar()
        antes = self.interna.read_bytes()
        for cambios in ({'perfil_ref': 'prf_' + 'b' * 32},
                        {'cuenta_ref': 'cta_' + 'b' * 32},
                        {'perfil_ref': 'prf_' + 'd' * 32}):
            with self.subTest(cambios=cambios), self.assertRaises(ValueError):
                self.enlazar(**cambios)
            self.assertEqual(self.interna.read_bytes(), antes)

    def test_cuenta_o_perfil_externos_no_se_pueden_enlazar(self):
        antes = self.interna.read_bytes()
        for cambios in ({'cuenta_ref': 'cta_' + 'c' * 32}, {'perfil_ref': 'prf_' + 'c' * 32}):
            with self.subTest(cambios=cambios), self.assertRaises(ValueError):
                self.enlazar(**cambios)
            self.assertEqual(self.interna.read_bytes(), antes)

    def test_certificado_divergente_o_referencias_libres_no_escriben(self):
        antes = self.interna.read_bytes()
        self.huella.return_value = 'd' * 64
        with self.assertRaises(ValueError):
            self.enlazar()
        self.huella.return_value = 'a' * 64
        for cambios in ({'cuenta_ref': '*'}, {'perfil_ref': 'prf_otro'},
                        {'identidad': '../jefatura.json'}, {'certificado': str(self.certificado)}):
            with self.subTest(cambios=cambios), self.assertRaises(ValueError):
                self.enlazar(**cambios)
        self.assertEqual(self.interna.read_bytes(), antes)

    def test_configuracion_enlazada_o_con_claves_repetidas_se_conserva(self):
        antes = self.interna.read_bytes()
        destino = self.material / 'identidad/otra.json'
        destino.write_bytes(antes)
        destino.chmod(0o600)
        self.interna.unlink()
        self.interna.symlink_to(destino)
        with self.assertRaises(ValueError):
            self.enlazar()
        self.assertEqual(destino.read_bytes(), antes)
        self.interna.unlink()
        self.interna.write_text('{"cuentas":[],"cuentas":[]}')
        self.interna.chmod(0o600)
        antes = self.interna.read_bytes()
        with self.assertRaises(ValueError):
            self.enlazar()
        self.assertEqual(self.interna.read_bytes(), antes)


if __name__ == '__main__':
    unittest.main()
