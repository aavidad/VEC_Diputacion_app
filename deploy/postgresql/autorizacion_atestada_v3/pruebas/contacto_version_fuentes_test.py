"""Control focal de superficies de versión; PostgreSQL real se ensaya aparte."""
from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parents[2]
AD = (ROOT/'autorizacion_atestada_v3/migraciones/000052_consulta_recibo_contacto_propio.up.sql').read_text()
T13 = (ROOT/'bolsa_registro_accesos/migraciones/000007_consulta_recibo_contacto_propio.up.sql').read_text()
STORE = (ROOT/'contacto_usuario_vec/migraciones/000002_consulta_recibo_contacto_propio.up.sql').read_text()

class VersionContacto(unittest.TestCase):
    def test_acciones_y_audiencias_distintas(self):
        for action, audience, purpose in (
            ('vec.contacto_usuario.version_propia','vec.contacto_usuario.version_propia.v1','gestion_contacto_propio'),
            ('vec.contacto_usuario.version_para_llamamiento','vec.contacto_usuario.version_llamamiento.v1','envio_llamamiento'),
        ):
            for sql in (AD,T13,STORE): self.assertIn(action,sql)
            self.assertIn(audience,AD)
            self.assertIn(purpose,AD)
        self.assertIn("p_accion='vec.contacto_usuario.version_propia' AND x->>'persona_ref' IS DISTINCT FROM sujeto",AD)
        self.assertIn("d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM encode(sha256(p_recurso),'hex')",AD)
        self.assertIn("r#>>'{ambitos,persona_ref}' IS DISTINCT FROM sujeto",AD)
        self.assertIn("NOT pg_has_role(session_user,'vec_contacto_usuario_reader','MEMBER')",STORE)
        self.assertIn("NOT pg_has_role(session_user,'vec_contacto_usuario_writer','MEMBER')",STORE)
        self.assertIn("v_revalidacion.decision_ref IS DISTINCT FROM v_consumo.decision_ref",STORE)
        self.assertNotRegex(STORE,re.compile(r'JOIN\s+vec_(?!contacto_usuario_v1)',re.I))

if __name__ == '__main__': unittest.main()
