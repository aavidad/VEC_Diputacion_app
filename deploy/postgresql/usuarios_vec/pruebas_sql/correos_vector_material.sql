\set ON_ERROR_STOP on
-- Bytes literales del test Go ports/correos_test.go. Esta prueba solo compara
-- serialización; las referencias de ejemplo no pasan la guarda de negocio.
DO $vector$ DECLARE material text:=$m${"superficie":"interna_corporativa","persona_ref":"per_prueba","perfil_ref":"prf_prueba","accion":"vec.correos.anadir","finalidad_ref":"finalidad:usuarios:correos-propios:v1","version_esperada":0,"clave_operacion":"operacion-1234567890","huellas_peticion":{"activa":{"clave_ref":"h3","valor":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},"retenidas":[{"clave_ref":"h1","valor":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},{"clave_ref":"h2","valor":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}]},"correo_ref":"","sustituto_ref":""}$m$;
BEGIN
 IF encode(sha256(convert_to(material,'UTF8')),'hex') IS DISTINCT FROM
   '1e3dbc638aa4205873f5a5bd40762af85518a72f708a023d423de4538d8c6bf3'
 THEN RAISE EXCEPTION 'material Go/SQL no coincide en bytes'; END IF;
END $vector$;
DO $vector$ DECLARE material text:=$m${"superficie":"interna_corporativa","persona_ref":"per_rrrrrrrrrrrrrrrrrrrrrrrr","perfil_ref":"prf_pppppppppppppppppppppppp","accion":"vec.correos.anadir","finalidad_ref":"finalidad:usuarios:correos-propios:v1","version_esperada":0,"clave_operacion":"operacion-1234567890","huellas_peticion":{"activa":{"clave_ref":"h3","valor":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},"retenidas":[{"clave_ref":"h1","valor":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},{"clave_ref":"h2","valor":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}]},"correo_ref":"","sustituto_ref":""}$m$;
BEGIN
 IF encode(sha256(convert_to(material,'UTF8')),'hex') IS DISTINCT FROM
   '2733127cd1d454edebdd5f3e3e975fe391febac0c9d1929b85f2f158f6c082c7'
 THEN RAISE EXCEPTION 'material Go/SQL válido no coincide en bytes'; END IF;
 IF vec_usuarios.huella_contexto_correos(material) !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'contexto de correo no calculable'; END IF;
END $vector$;
