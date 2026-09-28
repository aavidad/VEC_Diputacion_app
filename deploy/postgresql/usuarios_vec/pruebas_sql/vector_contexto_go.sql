\set ON_ERROR_STOP on
-- Vector emitido por Go: json.Marshal(ports.MaterialPreferencias), después
-- RecursoAutorizable.HuellaContextoAutorizacionSHA256() con ámbito de persona
-- y Atributos={material_sha256: SHA256(material literal)}.
DO $vector$
DECLARE material text:='{"persona_ref":"per_0123456789abcdefghijkl","perfil_ref":"prf_0123456789abcdefghijkl","accion":"vec.preferencias.actualizar","finalidad_ref":"finalidad:usuarios:preferencias-propias:v1","catalogo_version_ref":"usuarios-preferencias-v1","version_esperada":0,"clave_operacion":"operacion-1234567890","huella_peticion":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","valores":{"idioma":"navegador","tamano_texto":"normal","alto_contraste":false,"tema":"sistema","inicio":"cuadro","filas":20,"aviso_correo_tareas":false,"aviso_correo_plazos":false}}';
BEGIN
 IF encode(sha256(convert_to(material,'UTF8')),'hex')<>'aa8e47b6de7519c4c2ca35179a88959263d553235f46e1b5a96dbd5b5cb67780'
    OR vec_usuarios.huella_contexto_preferencias(material)<>'7cec22dba98ed3341ab10ef5bd6a259f8423c4ead83c15d42b6426f6160417ed'
 THEN RAISE EXCEPTION 'Usuarios: vector Go/SQL de contexto V3 divergente'; END IF;
END $vector$;
