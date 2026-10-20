"""La preparación OH nunca reejecuta una preparación interrumpida."""

from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

from copia_local_organizacion_historica_activacion import activar


class ActivacionOHTest(unittest.TestCase):
    def test_sin_modulos_no_prepara_permisos_ni_compila(self):
        copia = SimpleNamespace(leer=lambda: {'preparada': True})
        with patch('copia_local_organizacion_historica_activacion.subprocess.check_output') as compilar:
            with self.assertRaisesRegex(ValueError, 'copia interna'):
                activar(copia, Path('/fuente'))
            compilar.assert_not_called()

    def test_marca_previa_o_interrupcion_exigen_conciliar_sin_nuevos_efectos(self):
        with tempfile.TemporaryDirectory() as temporal:
            raiz = Path(temporal)
            estado = {'preparada': True, 'entorno_modulos_sha256': 'a' * 64}
            copia = SimpleNamespace(raiz=raiz, leer=lambda: estado, existe=lambda: True)
            for guardada in (False, True):
                with self.subTest(estado_guardado=guardada):
                    if guardada:
                        estado['organizacion_historica_semilla_sha256'] = 'b' * 64
                    else:
                        (raiz / 'organizacion-historica-semilla.json').write_text('{}')
                    with patch('copia_local_organizacion_historica_activacion.oh.sembrar') as sembrar:
                        with self.assertRaisesRegex(ValueError, 'concili'):
                            activar(copia, Path('/fuente'))
                        sembrar.assert_not_called()


if __name__ == '__main__':
    unittest.main()
