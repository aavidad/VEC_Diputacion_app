"""Contrato final de membresías de las conexiones nominales."""

from pathlib import Path
import unittest

from copia_local_conexiones import CONVOCA, GRUPOS
from copia_local_verificacion import consulta


class CopiaFalsa:
    material = Path('/copia/material')

    def leer(self):
        return {'puerto_pg': 55481}

    def entorno_modulos(self):
        return {variable: 'postgresql://vcl_prueba_' + str(i).zfill(2) +
                ':clave@localhost:55481/postgres?sslmode=verify-full&' +
                'sslrootcert=/copia/material/ca/ca.crt'
                for i, variable in enumerate(GRUPOS)}


class TestVerificacion(unittest.TestCase):
    def test_importador_exige_dos_arcos_exactos_y_los_demas_uno(self):
        sql = consulta(CopiaFalsa())
        login = 'vcl_prueba_' + str(list(GRUPOS).index(CONVOCA)).zfill(2)
        self.assertIn(f"('{login}','vec_bolsa_importacion_convoca_ejecutor',true,true)", sql)
        self.assertIn(f"('{login}','vec_bolsa_importacion_convoca_recuperador',true,true)", sql)
        self.assertEqual(sql.count(f"('{login}','vec_bolsa_importacion_convoca_"), 2)
        self.assertIn("FROM esperados EXCEPT SELECT login,grupo,hereda,cambia FROM reales", sql)
        self.assertIn("FROM reales EXCEPT SELECT login,grupo,hereda,cambia FROM esperados", sql)
        self.assertIn("HAVING count(r.login)<>(SELECT count(*) FROM esperados e WHERE e.login=l.login)", sql)
        self.assertIn("NOT EXISTS(SELECT 1 FROM reales WHERE administra)", sql)
        otro = 'vcl_prueba_00'
        self.assertEqual(sql.count(f"('{otro}','"), 1)


if __name__ == '__main__':
    unittest.main()
