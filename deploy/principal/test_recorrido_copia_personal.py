"""Entrada Personal B2 de la copia sin conexiones ni actores heredados."""

import hashlib
import json
from pathlib import Path
import tempfile
import unittest

from copia_local_personal import configuracion
from recorrido_copia.personal import preparar
from test_recorrido_copia_configuracion import Contexto


class PersonalTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.c = Contexto(Path(self.tmp.name))
        self.c.copia.leer = lambda: {'sid_copia': '12345678901234567', 'recorrido_completo': True}
        previo = self.c.consultar
        self.c.consultar = lambda sql: '1' if 'motivo_v2_catalogo_publicado' in sql else previo(sql)
        identidad = {'version': 1, 'autoridad': 'no_autoritativo',
                     'subject': 'desarrollo:rrhh-original', 'certificate_sha256': 'a' * 64}
        self.c.guardar(self.c.material / 'identidad/identidad.json', identidad)
        self.c.guardar(self.c.material / 'identidad/bolsa-bback.json', {
            'sujeto': identidad['subject'], 'certificado_sha256': identidad['certificate_sha256']})
        firmantes = self.c.carpeta / 'firmantes/metadata.json'
        datos = self.c.leer(firmantes)
        datos['organizacion_operativa_ref'] = self.c.parametros['organizacion_operativa_ref']
        datos['unidad_ref'] = 'unidad:rrhh'
        firmantes.write_text(json.dumps(datos))
        self.c.guardar(self.c.raiz / 'admin/h10/unidad/acuse-primero.json', {
            'estado': 'permitido', 'recibo': {'auditoria_ref': 'auditoria:unidad',
            'unidad': {'organismo_ref': 'org_' + 'b' * 32,
                       'unidad_ref': 'unidad:rrhh', 'revision': 1}}})

    def test_configuracion_propia_aceptada_por_productor(self):
        ruta = preparar(self.c)
        entrada = self.c.leer(ruta)
        p = entrada['referencias']
        self.assertEqual(p['PrincipalV3Ref'], p['ActorRef'])
        self.assertNotEqual(p['PerfilV3Ref'], self.c.leer(self.c.carpeta /
                            'firmantes/semilla-final/firmantes-datos.json')['referencias']['tecnico']['perfil']['firma'])
        self.assertEqual(entrada['personal_b2']['organismo_ref'], 'org_' + 'b' * 32)
        self.assertEqual(set(entrada['personal_b2']['dsn_files']),
                         {'personal_planes', 'personal_actos', 'personal_consultas', 'bolsa_persona', 'rpt'})
        self.assertEqual(len(entrada['personal_b2']['operaciones']), 27)
        self.assertEqual(configuracion(ruta)['esquema'], entrada['esquema'])
        self.assertEqual(preparar(self.c), ruta)

    def test_identidad_rrhh_ajena_se_rechaza(self):
        ruta = self.c.material / 'identidad/bolsa-bback.json'
        dato = self.c.leer(ruta)
        dato['sujeto'] = 'desarrollo:otro'
        ruta.write_text(json.dumps(dato))
        with self.assertRaisesRegex(ValueError, 'Identidad RRHH'):
            preparar(self.c)
        self.assertFalse((self.c.carpeta / 'personal/configuracion.json').exists())

    def test_motivo_no_publicado_se_rechaza(self):
        anterior = self.c.consultar
        self.c.consultar = lambda sql: '0' if 'motivo_v2_catalogo_publicado' in sql else anterior(sql)
        with self.assertRaisesRegex(ValueError, 'Motivo B2'):
            preparar(self.c)
        self.assertFalse((self.c.carpeta / 'personal/configuracion.json').exists())

    def test_unidad_h10_distinta_se_rechaza(self):
        ruta = self.c.raiz / 'admin/h10/unidad/acuse-primero.json'
        dato = self.c.leer(ruta)
        dato['recibo']['unidad']['unidad_ref'] = 'unidad:otra'
        ruta.write_text(json.dumps(dato))
        with self.assertRaisesRegex(ValueError, 'Unidad Personal'):
            preparar(self.c)
        self.assertFalse((self.c.carpeta / 'personal/configuracion.json').exists())


if __name__ == '__main__':
    unittest.main()
