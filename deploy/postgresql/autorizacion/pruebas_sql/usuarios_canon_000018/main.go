package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	usuarios "vec-diputacion-granada/internal/modules/usuarios/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

const persona = "per_aut18_sintetica_abcdefghijklmnopqrstuv"
const perfil = "prf_aut18_sintetico_abcdefghijklmnopqrstuv"
const cuenta = "cta_aut18_sintetica_abcdefghijklmnopqrstuv"

// La prueba utiliza los tipos y los bytes del dominio que publica la CLI.
func documentos(ahora time.Time) (core.VersionRol, core.ControlVigenciaVersionRol, core.AsignacionPerfil) {
	var concesiones []core.ConcesionRol
	poner := func(accion, tipo, finalidad string, campos []string) {
		concesiones = append(concesiones, core.ConcesionRol{Accion: accion, ModuloID: "usuarios", TipoRecurso: tipo,
			Finalidades: []string{finalidad}, CamposPermitidos: campos, GarantiaMinima: core.AuthAssuranceHigh})
	}
	poner(usuarios.AccionConsultarPreferencias, usuarios.TipoRecursoPreferencias, usuarios.FinalidadPreferenciasPropias, []string{"catalogo", "valores", "version"})
	poner(usuarios.AccionActualizarPreferencias, usuarios.TipoRecursoPreferencias, usuarios.FinalidadPreferenciasPropias, []string{"valores", "version"})
	for _, accion := range []string{usuarios.AccionConsultarImagen, usuarios.AccionActualizarImagen} {
		poner(accion, usuarios.TipoRecursoImagen, usuarios.FinalidadImagenPropia, usuarios.CamposPermitidosImagen(accion))
	}
	for _, accion := range []string{usuarios.AccionConsultarCorreos, usuarios.AccionAnadirCorreo, usuarios.AccionReenviarCorreo,
		usuarios.AccionVerificarCorreo, usuarios.AccionActivarCorreo, usuarios.AccionRetirarCorreo} {
		poner(accion, usuarios.TipoRecursoCorreos, usuarios.FinalidadCorreosPropios, usuarios.CamposPermitidosCorreos(accion))
	}
	rol := core.VersionRol{RolID: "candidato_usuarios_propios_desarrollo", Version: 1, Nombre: "areaPersonal.usuarios.rolPropio",
		Estado: core.EstadoVersionRolPublicada, Concesiones: concesiones, PublicadaPor: "seguridad:desarrollo:no-autoritativa", PublicadaEn: ahora}
	control := core.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1, Estado: core.EstadoControlVigenciaVersionRolHabilitada,
		ActualizadoPor: rol.PublicadaPor, ActualizadoEn: ahora}
	asignacion := core.AsignacionPerfil{AsignacionID: "asg_aut18_sintetica", Version: 1, PerfilActivoRef: perfil, PrincipalID: persona,
		VersionRolRef: rol.Referencia(), Estado: core.EstadoAsignacionPerfilActiva, Ambitos: []core.AmbitoPerfil{{Clave: "persona_ref", Valores: []string{persona}}},
		VigenteDesde: ahora, VigenteHasta: ahora.Add(time.Hour), EmitidaPor: "identidad:desarrollo:no-autoritativa", EmitidaEn: ahora}
	return rol, control, asignacion
}

func canonico(v interface {
	Validar() error
	HuellaSHA256() (string, error)
}) ([]byte, string, error) {
	if err := v.Validar(); err != nil {
		return nil, "", err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, "", err
	}
	h, err := v.HuellaSHA256()
	if err != nil {
		return nil, "", err
	}
	s := sha256.Sum256(b)
	if hex.EncodeToString(s[:]) != h {
		return nil, "", fmt.Errorf("huella de dominio divergente")
	}
	return b, h, nil
}

func sqlPrueba(ahora time.Time) (string, error) {
	r, c, a := documentos(ahora)
	rb, rh, e := canonico(r)
	if e != nil {
		return "", e
	}
	cb, ch, e := canonico(c)
	if e != nil {
		return "", e
	}
	ab, ah, e := canonico(a)
	if e != nil {
		return "", e
	}
	sql := fmt.Sprintf(`
CREATE ROLE aut18_ensayo_publicador LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_publicador_usuarios_externo TO aut18_ensayo_publicador WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT EXECUTE ON FUNCTION vec_autorizacion.rol_usuarios_externo_acotado_v1(jsonb) TO aut18_ensayo_publicador;
SET SESSION AUTHORIZATION aut18_ensayo_publicador;
DO $ensayo$
DECLARE rb bytea:=decode('%s','hex'); cb bytea:=decode('%s','hex'); ab bytea:=decode('%s','hex');
 d jsonb; mal jsonb; bad bytea; x jsonb; retirada jsonb; campo text; publicado record;
BEGIN
 d:=convert_from(rb,'UTF8')::jsonb;
 IF d->'retirada_en' IS DISTINCT FROM '"0001-01-01T00:00:00Z"'::jsonb
  OR vec_autorizacion.rol_usuarios_externo_acotado_v1(d) IS NOT TRUE
  OR vec_autorizacion.rol_usuarios_externo_acotado_v1(d-'retirada_en') IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT18: rol canónico Go rechazado'; END IF;
 FOREACH retirada IN ARRAY ARRAY['"2026-09-30T01:00:00Z"'::jsonb,'"0001-01-01T00:00:00+00:00"'::jsonb,'null'::jsonb,'0'::jsonb,'true'::jsonb] LOOP
  mal:=jsonb_set(d,'{retirada_en}',retirada);
  IF vec_autorizacion.rol_usuarios_externo_acotado_v1(mal) IS NOT FALSE
  THEN RAISE EXCEPTION 'AUT18: retirada no canónica admitida'; END IF;
  bad:=convert_to(mal::text,'UTF8');
  BEGIN
   PERFORM vec_autorizacion.publicar_rol_usuarios_externo_v1(bad,encode(sha256(bad),'hex'),cb,'%s',0,NULL,
    'seguridad:desarrollo:no-autoritativa','acto:aut18:rol:rechazo');
   RAISE EXCEPTION 'AUT18: publicación de retirada no canónica aceptada';
  EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 END LOOP;
 FOREACH campo IN ARRAY ARRAY['extra','retirada_por','retirada_ref','motivo_retirada_codigo'] LOOP
  mal:=d||jsonb_build_object(campo,'intruso');
  IF vec_autorizacion.rol_usuarios_externo_acotado_v1(mal) IS NOT FALSE
  THEN RAISE EXCEPTION 'AUT18: propiedad adicional del rol admitida'; END IF;
  bad:=convert_to(mal::text,'UTF8');
  BEGIN
   PERFORM vec_autorizacion.publicar_rol_usuarios_externo_v1(bad,encode(sha256(bad),'hex'),cb,'%s',0,NULL,
    'seguridad:desarrollo:no-autoritativa','acto:aut18:rol:extra');
   RAISE EXCEPTION 'AUT18: publicación de propiedad adicional aceptada';
  EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 END LOOP;
 SELECT * INTO publicado FROM vec_autorizacion.publicar_rol_usuarios_externo_v1(rb,'%s',cb,'%s',0,NULL,
  'seguridad:desarrollo:no-autoritativa','acto:aut18:rol:positivo');
 IF publicado.huella_rol IS DISTINCT FROM '%s' OR publicado.huella_control IS DISTINCT FROM '%s'
 THEN RAISE EXCEPTION 'AUT18: huellas Go del rol alteradas'; END IF;
 d:=convert_from(ab,'UTF8')::jsonb;
 IF d->'revocada_en' IS DISTINCT FROM '"0001-01-01T00:00:00Z"'::jsonb
 THEN RAISE EXCEPTION 'AUT18: fixture de asignación omitió fecha cero Go'; END IF;
 FOREACH retirada IN ARRAY ARRAY['"2026-09-30T01:00:00Z"'::jsonb,'"0001-01-01T00:00:00+00:00"'::jsonb,'null'::jsonb,'0'::jsonb,'true'::jsonb] LOOP
  bad:=convert_to(jsonb_set(d,'{revocada_en}',retirada)::text,'UTF8');
  BEGIN
   PERFORM vec_autorizacion.publicar_asignacion_usuarios_externo_v1(bad,encode(sha256(bad),'hex'),0,NULL,
    'identidad:desarrollo:no-autoritativa','acto:aut18:asignacion:rechazo','%s');
   RAISE EXCEPTION 'AUT18: revocación no canónica aceptada';
  EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 END LOOP;
 FOREACH campo IN ARRAY ARRAY['extra','revocada_por','revocacion_ref'] LOOP
  bad:=convert_to((d||jsonb_build_object(campo,'intruso'))::text,'UTF8');
  BEGIN
   PERFORM vec_autorizacion.publicar_asignacion_usuarios_externo_v1(bad,encode(sha256(bad),'hex'),0,NULL,
    'identidad:desarrollo:no-autoritativa','acto:aut18:asignacion:extra','%s');
   RAISE EXCEPTION 'AUT18: propiedad adicional de asignación aceptada';
  EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 END LOOP;
 SELECT * INTO publicado FROM vec_autorizacion.publicar_asignacion_usuarios_externo_v1(ab,'%s',0,NULL,
  'identidad:desarrollo:no-autoritativa','acto:aut18:asignacion:positivo','%s');
 IF publicado.huella_sha256 IS DISTINCT FROM '%s' THEN RAISE EXCEPTION 'AUT18: huella Go de asignación alterada'; END IF;
END $ensayo$;
RESET SESSION AUTHORIZATION;
DO $historia$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:candidato_usuarios_propios_desarrollo:v1'
  AND huella_sha256='%s' AND documento=convert_from(decode('%s','hex'),'UTF8')::jsonb)
 OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.asignacion_perfil_externa WHERE asignacion_ref='asignacion:asg_aut18_sintetica:v1'
  AND huella_sha256='%s' AND documento=convert_from(decode('%s','hex'),'UTF8')::jsonb)
 THEN RAISE EXCEPTION 'AUT18: documento o huella de dominio no conservado'; END IF;
 IF (SELECT count(*) FROM vec_autorizacion.publicacion_candidato_externo_evento WHERE acto_ref LIKE 'acto:aut18:%%')<>2
 THEN RAISE EXCEPTION 'AUT18: efectos de rechazo o publicación duplicada'; END IF;
END $historia$;
DO $fuente_login$
BEGIN
 IF to_regrole('vec_externo_usuarios_v3_fuente_autorizacion_desarrollo') IS NULL THEN
  CREATE ROLE vec_externo_usuarios_v3_fuente_autorizacion_desarrollo LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
  GRANT vec_autorizacion_fuente_usuarios_externa TO vec_externo_usuarios_v3_fuente_autorizacion_desarrollo WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
 END IF;
END $fuente_login$;
SET SESSION AUTHORIZATION vec_externo_usuarios_v3_fuente_autorizacion_desarrollo;
DO $fuente$
DECLARE i record;
BEGIN
 SELECT * INTO STRICT i FROM vec_autorizacion.obtener_instantanea_usuarios_externo_v1(
  'per_aut18_sintetica_abcdefghijklmnopqrstuv','prf_aut18_sintetico_abcdefghijklmnopqrstuv');
 IF i.documento_rol->'retirada_en' IS DISTINCT FROM '"0001-01-01T00:00:00Z"'::jsonb
  OR i.documento_asignacion->'revocada_en' IS DISTINCT FROM '"0001-01-01T00:00:00Z"'::jsonb
 THEN RAISE EXCEPTION 'AUT18: fuente rechaza o altera el canon Go'; END IF;
END $fuente$;
RESET SESSION AUTHORIZATION;
DO $v3$
DECLARE f regprocedure; cuerpo text;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_autorizacion.registrar_decision_usuarios_externo_v3(bytea,bytea,numeric,numeric)'::regprocedure,
  'vec_autorizacion.revalidar_decision_usuarios_externo_v3_viva(bytea,bytea,numeric,numeric)'::regprocedure,
  'vec_autorizacion.registrar_y_revalidar_decision_usuarios_externo_v3(bytea,bytea,numeric,numeric)'::regprocedure] LOOP
  SELECT prosrc INTO STRICT cuerpo FROM pg_catalog.pg_proc WHERE oid=f;
  IF strpos(cuerpo,'vec_autorizacion.rol_usuarios_externo_acotado_v1(r.documento) IS TRUE')=0
   OR strpos(cuerpo,'vec_autorizacion.decision_contexto_actor_v3_canonica(d) IS DISTINCT FROM p_decision')=0
   OR strpos(cuerpo,'vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1(')=0
  THEN RAISE EXCEPTION 'AUT18: V3 omitió acotación, canon o acreditación'; END IF;
 END LOOP;
END $v3$;
ROLLBACK;
`, hex.EncodeToString(rb), hex.EncodeToString(cb), hex.EncodeToString(ab), ch, ch, rh, ch, rh, ch, cuenta, cuenta, ah, cuenta, ah, rh, hex.EncodeToString(rb), ah, hex.EncodeToString(ab))
	return sql, nil
}

func main() {
	s, e := sqlPrueba(time.Now().UTC().Truncate(time.Microsecond).Add(-time.Minute))
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
	fmt.Print(s)
}
