"""Contratos focales del selector y del conductor ADMIN nativo."""

import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parent))

from admin_copia import ejecutor
import copia_local_administracion


class CopiaFalsa:
    def __init__(self, estado):
        self.estado = estado

    def leer(self):
        return self.estado


class Selector(unittest.TestCase):
    def test_versionado_usa_conductor_nativo(self):
        copia = CopiaFalsa({'administracion': {'versionado': True, 'puerto': 19481}})
        with patch.object(ejecutor, 'ejecutar') as nativo:
            copia_local_administracion.ejecutar(copia, 'preparar')
        nativo.assert_called_once_with(copia, 'preparar')

    def test_fase_invalida_no_invoca_conductor(self):
        copia = CopiaFalsa({'administracion': {'versionado': True, 'puerto': 19481}})
        with patch.object(ejecutor, 'ejecutar') as nativo:
            with self.assertRaisesRegex(ValueError, 'Fase ADMIN desconocida'):
                copia_local_administracion.ejecutar(copia, 'repetir')
        nativo.assert_not_called()


class ContextoFalso:
    def __init__(self, raiz):
        self.raiz = raiz

    @staticmethod
    def guardar(ruta, valor):
        ruta.write_text(json.dumps(valor))

    @staticmethod
    def sha(ruta):
        import hashlib
        return hashlib.sha256(ruta.read_bytes()).hexdigest()

    @staticmethod
    def ahora():
        import datetime as dt
        return dt.datetime(2026, 10, 20, tzinfo=dt.timezone.utc)


class AprobacionTecnica(unittest.TestCase):
    def test_liga_fase_y_preimagen_sin_atribuir_firma(self):
        with tempfile.TemporaryDirectory() as tmp:
            run = Path(tmp)
            (run / 'aprobaciones').mkdir()
            (run / 'fuentes').mkdir()
            destino = run / 'aprobaciones/fuentes.json'
            config = {'backup': {'ref': 'preimagen:admin:ensayo', 'sha256': 'b' * 64},
                      'dba': {'fuentes': {'aprobacion_externa': str(destino),
                                          'aprobacion_ref': 'aprobacion:fuentes:ensayo'}}}
            (run / 'config.json').write_text(json.dumps(config))
            (run / 'contrato-arranque.json').write_text(json.dumps({'sid_copia': '123456789012345'}))
            (run / 'fuentes/plan.json').write_text(json.dumps({'huella_plan_sha256': 'a' * 64}))
            with patch.object(ejecutor, 'documento', side_effect=lambda p: json.loads(p.read_text())):
                ejecutor.publicar_autorizacion(ContextoFalso(run), run, 'fuentes', 'encargo:f1-131')
            self.assertEqual(json.loads(destino.read_text()), {
                'huella_plan_sha256': 'a' * 64,
                'aprobacion_ref': 'aprobacion:fuentes:ensayo',
                'backup_ref': 'preimagen:admin:ensayo'})
            registro = json.loads((run / 'aprobaciones/fuentes-contexto.json').read_text())
            self.assertIs(registro['firma_o_recibo_juridico'], False)
            self.assertEqual(registro['plan_sha256'], 'a' * 64)
            self.assertEqual(registro['preimagen_admin_sha256'], 'b' * 64)
            self.assertEqual(registro['sid_copia'], '123456789012345')

    def test_aplicacion_incierta_no_reprepara_ni_reintenta(self):
        pasos = []

        def script_falso(_c, _kit, _nombre, *argumentos):
            fase = argumentos[-1]
            pasos.append(fase)
            if fase == 'aplicar_fuentes':
                raise ValueError('resultado incierto')

        with (patch.object(ejecutor, 'script', side_effect=script_falso),
              patch.object(ejecutor, 'publicar_autorizacion', side_effect=lambda *_: pasos.append('autorizar')),
              patch.object(ejecutor, 'sql', side_effect=lambda *_: pasos.append('sql'))):
            with self.assertRaisesRegex(ValueError, 'resultado incierto'):
                ejecutor.etapa_h10(object(), Path('/kit'), Path('/run'),
                                    'fuentes', 'encargo:f1-131')
        self.assertEqual(pasos, ['preparar_fuentes', 'autorizar',
                                 'configurar_fuentes', 'sql', 'aplicar_fuentes'])


if __name__ == '__main__':
    unittest.main()
