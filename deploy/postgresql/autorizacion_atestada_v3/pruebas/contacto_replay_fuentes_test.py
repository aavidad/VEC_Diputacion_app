"""Invariantes de fuente del replay; la concurrencia requiere PostgreSQL real."""
from pathlib import Path
import unittest

PG = Path(__file__).resolve().parents[2]
AD = (PG/'autorizacion_atestada_v3/migraciones/000051_consumidor_contacto_usuario.up.sql').read_text()
STORE = (PG/'contacto_usuario_vec/migraciones/000001_almacen_contacto_usuario.up.sql').read_text()
DOWN = (PG/'contacto_usuario_vec/migraciones/000001_almacen_contacto_usuario.down.sql').read_text()

class ReplayContacto(unittest.TestCase):
    def test_hmac_versionado_antes_de_conflicto_y_sin_email_en_tabla(self):
        self.assertIn('contacto_huellas_replay_canonicas_v1', AD)
        self.assertIn('"HuellasReplay":"array"', AD)
        self.assertIn('"HuellasReplay":\'||huellas', AD)
        self.assertIn('hmac_clave_ref text NOT NULL', STORE)
        self.assertIn('hmac_valor text NOT NULL', STORE)
        self.assertNotIn('correo text', STORE)
        self.assertLess(STORE.index('registrar_y_consumir_alta_contacto_usuario'), STORE.index('IF coalesce(v_actual,0)<>v_anterior'))
        self.assertLess(STORE.index('IF coalesce(v_actual,0)<>v_anterior'), STORE.index('registrar_contacto_usuario_v1(\n        p_auditoria'))
        self.assertIn('v_previo.hmac_clave_ref', STORE)
        self.assertIn('v_previo.auditoria_central::bytea', STORE)
        self.assertIn('replay_confirmado boolean', STORE)
        self.assertIn('revalidar_actualizar_contacto_usuario_v3_atestada', STORE)
        self.assertNotIn('DELETE FROM', STORE)
        self.assertNotIn('ON CONFLICT DO NOTHING', STORE)
        self.assertIn("current_user AND rolsuper", DOWN)
        self.assertIn('historia', DOWN)

if __name__ == '__main__': unittest.main()
