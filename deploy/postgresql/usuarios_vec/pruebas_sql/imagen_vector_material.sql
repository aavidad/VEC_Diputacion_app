\set ON_ERROR_STOP on
-- Bytes literales de canonico/imagen_test.go: Go y SQL calculan la misma
-- huella de petición y la misma huella de contexto V3.
DO $vector$ DECLARE material text:=$m${"superficie":"externa_personal","persona_ref":"per_rrrrrrrrrrrrrrrrrrrrrrrr","perfil_ref":"prf_pppppppppppppppppppppppp","accion":"vec.imagen.actualizar","finalidad_ref":"finalidad:usuarios:imagen-propia:v1","catalogo_version_ref":"usuarios-imagen-v1","version_esperada":3,"clave_operacion":"operacion-1234567890","huella_peticion":"5326d57059087f951ba0cf93e12118736b3d45ba1fb85a4f866fb5a04040d035","eleccion":{"modo":"foto","paleta":"verde","icono":""},"foto_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}$m$;
BEGIN
 IF encode(sha256(convert_to(material,'UTF8')),'hex') IS DISTINCT FROM
   '65c71b4daa1afa50cc4554533ea8bcdf19f2c83a58b57026d1af76a664be4887'
 THEN RAISE EXCEPTION 'material Go/SQL no coincide en bytes'; END IF;
 IF vec_usuarios.huella_contexto_imagen(material) IS DISTINCT FROM
   '318c9ea3115b62fd039316e8fc8215090e08ffc199034b7c8f672aa5c36cdae7'
 THEN RAISE EXCEPTION 'contexto V3 Go/SQL no coincide'; END IF;
 IF vec_usuarios.huella_peticion_imagen(material::jsonb) IS DISTINCT FROM
   '5326d57059087f951ba0cf93e12118736b3d45ba1fb85a4f866fb5a04040d035'
 THEN RAISE EXCEPTION 'huella de petición Go/SQL no coincide'; END IF;
END $vector$;
