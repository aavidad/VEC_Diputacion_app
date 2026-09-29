\set ON_ERROR_STOP on
-- Bytes literales de canonico/correos_test.go: Go y SQL calculan la misma
-- huella de contexto V3 sobre el mismo material.
DO $vector$ DECLARE material text:=$m${"superficie":"interna_corporativa","persona_ref":"per_rrrrrrrrrrrrrrrrrrrrrrrr","perfil_ref":"prf_pppppppppppppppppppppppp","accion":"vec.correos.anadir","finalidad_ref":"finalidad:usuarios:correos-propios:v1","version_esperada":0,"clave_operacion":"operacion-1234567890","huellas_peticion":{"activa":{"clave_ref":"h3","valor":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},"retenidas":[{"clave_ref":"h1","valor":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},{"clave_ref":"h2","valor":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}]},"correo_ref":""}$m$;
BEGIN
 IF encode(sha256(convert_to(material,'UTF8')),'hex') IS DISTINCT FROM
   'ce37f62deab6db0c5ec9e5cc408b0ddab63a1a6b0d620c371e02b70d472eb53b'
 THEN RAISE EXCEPTION 'material Go/SQL no coincide en bytes'; END IF;
 IF vec_usuarios.huella_contexto_correos(material) IS DISTINCT FROM
   '2a74f978a1b52f3985c3c53fda94b0f5cb052628fa6f93ceec4254ed61a529b5'
 THEN RAISE EXCEPTION 'contexto V3 Go/SQL no coincide'; END IF;
END $vector$;
