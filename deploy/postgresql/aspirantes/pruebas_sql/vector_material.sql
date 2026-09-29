\set ON_ERROR_STOP on
-- El material literal que serializa Go y la huella de su recurso V3
-- (canonico/material_test.go) coinciden con la que recalcula SQL.
DO $vector$ BEGIN
 IF vec_aspirantes.huella_contexto('{"superficie":"externa_personal","persona_ref":"per_llllllllllllllllllllllll","perfil_ref":"prf_pppppppppppppppppppppppp","accion":"vec.aspirantes.ficha.consultar","finalidad_ref":"finalidad:aspirantes:ficha-propia:v1","version_esperada":0,"clave_operacion":"","huellas_peticion":{},"indice_documento":{"clave_ref":"clave:indice:v1","valor":"'||repeat('ab',32)||'"}}')
    <>'d299e353358b306d5aa4df30b6da94df308d312f71500e205df8670422e0b1a4'
 THEN RAISE EXCEPTION 'vector Go/SQL divergente'; END IF;
END $vector$;
