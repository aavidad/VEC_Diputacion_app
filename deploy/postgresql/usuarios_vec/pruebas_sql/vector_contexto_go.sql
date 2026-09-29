\set ON_ERROR_STOP on
-- Vector emitido por Go: json.Marshal(ports.MaterialPreferencias), después
-- RecursoAutorizable.HuellaContextoAutorizacionSHA256() con ámbito de persona
-- y Atributos={material_sha256: SHA256(material literal)}.
DO $vector$
DECLARE base text:='{"persona_ref":"per_0123456789abcdefghijkl","perfil_ref":"prf_0123456789abcdefghijkl","accion":"vec.preferencias.actualizar","finalidad_ref":"finalidad:usuarios:preferencias-propias:v1","catalogo_version_ref":"usuarios-preferencias-v1","version_esperada":0,"clave_operacion":"operacion-1234567890","huella_peticion":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","valores":{"idioma":"navegador","tamano_texto":"normal","alto_contraste":false,"tema":"sistema","inicio":"cuadro","filas":20,"aviso_correo_tareas":false,"aviso_correo_plazos":false}}';
 material text;
BEGIN
 material:='{"superficie":"interna_corporativa",'||substring(base FROM 2);
 IF encode(sha256(convert_to(material,'UTF8')),'hex')<>'93e51bfaf653b06d10b6033cd348af16ae15796e4c36d99fe7ae19fa6ccda517'
    OR vec_usuarios.huella_contexto_preferencias(material)<>'c00f649660c182645235e15e2e58a3d94c985dac9dca5e7b423b8a19bb03e96d'
 THEN RAISE EXCEPTION 'Usuarios: vector Go/SQL interno divergente'; END IF;
 material:='{"superficie":"externa_personal",'||substring(base FROM 2);
 IF encode(sha256(convert_to(material,'UTF8')),'hex')<>'e22d42d7a997fdbdd9a71ff10e7b805767bc5b7d28bb61870b4097f78cd4e723'
    OR vec_usuarios.huella_contexto_preferencias(material)<>'4776384ae63a032cf2c69e245aa8323c1683cf00515a2a79bdbfa9a76b4ade41'
 THEN RAISE EXCEPTION 'Usuarios: vector Go/SQL de contexto V3 divergente'; END IF;
END $vector$;
