package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
)

const persona = "per_aut19_sintetica_abcdefghijklmnopqrstuv"
const perfil = "prf_aut19_sintetico_abcdefghijklmnopqrstuv"
const candidato = "can_aut19_sintetico_abcdefghijklmnopqrstuv"

// La prueba utiliza los tipos y los bytes del dominio que publica la CLI.
func documentos(ahora time.Time) (core.VersionRol, core.ControlVigenciaVersionRol, core.AsignacionPerfil) {
	var concesiones []core.ConcesionRol
	poner := func(accion, tipo, finalidad string, campos []string) {
		concesiones = append(concesiones, core.ConcesionRol{Accion: accion, ModuloID: "bolsa", TipoRecurso: tipo,
			Finalidades: []string{finalidad}, CamposPermitidos: campos, GarantiaMinima: core.AuthAssuranceHigh})
	}
	poner("bolsa.historial_propio.consultar", "participaciones_candidato", "consulta_historial_propio", []string{"contratos_propios", "llamamientos_propios", "renuncias_propias"})
	poner("bolsa.participaciones_propias.consultar", "participaciones_candidato", "consulta_participaciones_propias", []string{"participaciones_candidato_minimizadas"})
	rol := core.VersionRol{RolID: "candidato_bolsa_historial_propio_desarrollo", Version: 137, Nombre: "Consulta propia de bolsa en desarrollo",
		Estado: core.EstadoVersionRolPublicada, Concesiones: concesiones, PublicadaPor: "seguridad:desarrollo:no-autoritativa", PublicadaEn: ahora}
	control := core.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1, Estado: core.EstadoControlVigenciaVersionRolHabilitada,
		ActualizadoPor: rol.PublicadaPor, ActualizadoEn: ahora}
	asignacion := core.AsignacionPerfil{AsignacionID: "asg_aut19_sintetica", Version: 1, PerfilActivoRef: perfil, PrincipalID: persona,
		VersionRolRef: rol.Referencia(), Estado: core.EstadoAsignacionPerfilActiva, Ambitos: []core.AmbitoPerfil{{Clave: "candidato_ref", Valores: []string{candidato}}},
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
CREATE ROLE aut19_ensayo_publicador LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_autorizacion_publicador_candidato_externo TO aut19_ensayo_publicador WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT EXECUTE ON FUNCTION vec_autorizacion.rol_candidato_externo_acotado_v1(jsonb) TO aut19_ensayo_publicador;
SET SESSION AUTHORIZATION aut19_ensayo_publicador;
DO $ensayo$
DECLARE rb bytea:=decode('%s','hex'); cb bytea:=decode('%s','hex'); ab bytea:=decode('%s','hex');
 d jsonb; mal jsonb; bad bytea; x jsonb; retirada jsonb; campo text; publicado record;
BEGIN
 d:=convert_from(rb,'UTF8')::jsonb;
 IF d->'retirada_en' IS DISTINCT FROM '"0001-01-01T00:00:00Z"'::jsonb
  OR vec_autorizacion.rol_candidato_externo_acotado_v1(d) IS NOT TRUE
  OR vec_autorizacion.rol_candidato_externo_acotado_v1(d-'retirada_en') IS NOT TRUE
  OR vec_autorizacion.rol_candidato_externo_acotado_v1(jsonb_set(d,'{retirada_en}','null'::jsonb)) IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT19: rol canónico Go rechazado'; END IF;
 FOREACH retirada IN ARRAY ARRAY['"2026-09-30T01:00:00Z"'::jsonb,'"0001-01-01T00:00:00+00:00"'::jsonb,'0'::jsonb,'true'::jsonb] LOOP
  mal:=jsonb_set(d,'{retirada_en}',retirada);
  IF vec_autorizacion.rol_candidato_externo_acotado_v1(mal) IS NOT FALSE
  THEN RAISE EXCEPTION 'AUT19: retirada no canónica admitida'; END IF;
  bad:=convert_to(mal::text,'UTF8');
  BEGIN
   PERFORM vec_autorizacion.publicar_rol_candidato_externo_v1(bad,encode(sha256(bad),'hex'),cb,'%s',0,NULL,
    'seguridad:desarrollo:no-autoritativa','acto:aut19:rol:rechazo');
   RAISE EXCEPTION 'AUT19: publicación de retirada no canónica aceptada';
  EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 END LOOP;
 mal:=jsonb_set(d,'{rol_id}','"candidato_usuarios_propios_desarrollo"'::jsonb);
 IF vec_autorizacion.rol_candidato_externo_acotado_v1(mal) IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT19: amplió la familia nominal'; END IF;
 FOREACH campo IN ARRAY ARRAY['extra','retirada_por','retirada_ref','motivo_retirada_codigo'] LOOP
  mal:=d||jsonb_build_object(campo,'intruso');
  IF vec_autorizacion.rol_candidato_externo_acotado_v1(mal) IS NOT FALSE
  THEN RAISE EXCEPTION 'AUT19: propiedad adicional del rol admitida'; END IF;
  bad:=convert_to(mal::text,'UTF8');
  BEGIN
   PERFORM vec_autorizacion.publicar_rol_candidato_externo_v1(bad,encode(sha256(bad),'hex'),cb,'%s',0,NULL,
    'seguridad:desarrollo:no-autoritativa','acto:aut19:rol:extra');
   RAISE EXCEPTION 'AUT19: publicación de propiedad adicional aceptada';
  EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 END LOOP;
 SELECT * INTO publicado FROM vec_autorizacion.publicar_rol_candidato_externo_v1(rb,'%s',cb,'%s',0,NULL,
  'seguridad:desarrollo:no-autoritativa','acto:aut19:rol:positivo');
 IF publicado.huella_rol IS DISTINCT FROM '%s' OR publicado.huella_control IS DISTINCT FROM '%s'
 THEN RAISE EXCEPTION 'AUT19: huellas Go del rol alteradas'; END IF;
 BEGIN
  PERFORM vec_autorizacion.publicar_rol_candidato_externo_v1(rb,publicado.huella_rol,cb,publicado.huella_control,0,NULL,
   'seguridad:desarrollo:no-autoritativa','acto:aut19:rol:cas');
  RAISE EXCEPTION 'AUT19: CAS de rol antiguo aceptado';
 EXCEPTION WHEN serialization_failure THEN NULL; END;
 d:=convert_from(ab,'UTF8')::jsonb;
 IF d->'revocada_en' IS DISTINCT FROM '"0001-01-01T00:00:00Z"'::jsonb
 THEN RAISE EXCEPTION 'AUT19: fixture de asignación omitió fecha cero Go'; END IF;
 FOREACH retirada IN ARRAY ARRAY['"2026-09-30T01:00:00Z"'::jsonb,'"0001-01-01T00:00:00+00:00"'::jsonb,'0'::jsonb,'true'::jsonb] LOOP
  bad:=convert_to(jsonb_set(d,'{revocada_en}',retirada)::text,'UTF8');
  BEGIN
   PERFORM vec_autorizacion.publicar_asignacion_candidato_externo_v1(bad,encode(sha256(bad),'hex'),0,NULL,
    'identidad:desarrollo:no-autoritativa','acto:aut19:asignacion:rechazo');
   RAISE EXCEPTION 'AUT19: revocación no canónica aceptada';
  EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 END LOOP;
 -- Compatibilidad opcional: ambos subbloques se revierten. El positivo
 -- definitivo conserva después los bytes exactos originales del dominio Go.
 FOREACH mal IN ARRAY ARRAY[d-'revocada_en',jsonb_set(d,'{revocada_en}','null'::jsonb)] LOOP
  bad:=convert_to(mal::text,'UTF8');
  BEGIN
   PERFORM vec_autorizacion.publicar_asignacion_candidato_externo_v1(bad,encode(sha256(bad),'hex'),0,NULL,
    'identidad:desarrollo:no-autoritativa','acto:aut19:asignacion:opcional');
   RAISE EXCEPTION 'revertir compatibilidad temporal' USING ERRCODE='U0001';
  EXCEPTION WHEN SQLSTATE 'U0001' THEN NULL; END;
 END LOOP;
 FOREACH campo IN ARRAY ARRAY['extra','revocada_por','revocacion_ref'] LOOP
  bad:=convert_to((d||jsonb_build_object(campo,'intruso'))::text,'UTF8');
  BEGIN
   PERFORM vec_autorizacion.publicar_asignacion_candidato_externo_v1(bad,encode(sha256(bad),'hex'),0,NULL,
    'identidad:desarrollo:no-autoritativa','acto:aut19:asignacion:extra');
   RAISE EXCEPTION 'AUT19: propiedad adicional de asignación aceptada';
  EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 END LOOP;
 SELECT * INTO publicado FROM vec_autorizacion.publicar_asignacion_candidato_externo_v1(ab,'%s',0,NULL,
  'identidad:desarrollo:no-autoritativa','acto:aut19:asignacion:positivo');
 IF publicado.huella_sha256 IS DISTINCT FROM '%s' THEN RAISE EXCEPTION 'AUT19: huella Go de asignación alterada'; END IF;
 BEGIN
  PERFORM vec_autorizacion.publicar_asignacion_candidato_externo_v1(ab,publicado.huella_sha256,0,NULL,
   'identidad:desarrollo:no-autoritativa','acto:aut19:asignacion:cas');
  RAISE EXCEPTION 'AUT19: CAS de asignación antiguo aceptado';
 EXCEPTION WHEN serialization_failure THEN NULL; END;
END $ensayo$;
RESET SESSION AUTHORIZATION;
DO $historia$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:candidato_bolsa_historial_propio_desarrollo:v137'
  AND huella_sha256='%s' AND documento=convert_from(decode('%s','hex'),'UTF8')::jsonb)
 OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.asignacion_perfil_externa WHERE asignacion_ref='asignacion:asg_aut19_sintetica:v1'
  AND huella_sha256='%s' AND documento=convert_from(decode('%s','hex'),'UTF8')::jsonb)
 THEN RAISE EXCEPTION 'AUT19: documento o huella de dominio no conservado'; END IF;
 IF (SELECT count(*) FROM vec_autorizacion.publicacion_candidato_externo_evento WHERE acto_ref LIKE 'acto:aut19:%%')<>2
 THEN RAISE EXCEPTION 'AUT19: efectos de rechazo o publicación duplicada'; END IF;
END $historia$;
ROLLBACK;
\echo AUT19-PRUEBA-OK
`, hex.EncodeToString(rb), hex.EncodeToString(cb), hex.EncodeToString(ab), ch, ch, rh, ch, rh, ch, ah, ah, rh, hex.EncodeToString(rb), ah, hex.EncodeToString(ab))
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
