package bootstrap

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	usuariospg "vec-diputacion-granada/internal/modules/usuarios/adapters/postgres"
	usuariosseguridad "vec-diputacion-granada/internal/modules/usuarios/adapters/seguridad"
	usuariosapp "vec-diputacion-granada/internal/modules/usuarios/application"
	"vec-diputacion-granada/internal/modules/usuarios/canonico"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Recorrido B59 sobre un clon DESECHABLE de la principal con la lista SQL de
// B59 instalada y un Mailpit con STARTTLS. No forma parte de la puerta: sólo
// se ejecuta con VEC_B59_RECORRIDO_CLON_DSN (superusuario del clon) y
// VEC_B59_RECORRIDO_CONFIRMO=clon-desechable, porque crea LOGIN de prueba,
// siembra correos y sustituye en ese clon la fachada AD3-109 por una V3
// sintética. Es real todo lo demás: funciones SQL de Usuarios, ContextoActor
// y Bolsa, cifrado de «Mis correos», transporte SMTP y Mailpit.
func TestRecorridoB59AvisosMisCorreosClonMailpit(t *testing.T) {
	admin := os.Getenv("VEC_B59_RECORRIDO_CLON_DSN")
	if admin == "" || os.Getenv("VEC_B59_RECORRIDO_CONFIRMO") != "clon-desechable" {
		t.Skip("recorrido B59: sólo contra un clon desechable (ver comentario)")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelar()
	pool, err := pgxpool.New(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%v\n%s", err, sql)
		}
	}
	// 1) LOGIN de prueba con la única membresía de su ejecutor y V3 sintética.
	exec(`DO $$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_b59_rec_usuarios') THEN
  CREATE ROLE vec_b59_rec_usuarios LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
  GRANT vec_usuarios_ejecutor_interno TO vec_b59_rec_usuarios WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_b59_rec_bolsa') THEN
  CREATE ROLE vec_b59_rec_bolsa LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
  GRANT vec_bolsa_llamamientos_ejecutor TO vec_b59_rec_bolsa WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
 END IF; END $$`)
	exec(`CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.consumir_correo_avisos_llamamiento_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
 SELECT convert_from(p_decision,'UTF8')::jsonb->>'decision_ref', convert_from(p_capacidad,'UTF8')::jsonb->>'efecto_ref',
  convert_from(p_capacidad,'UTF8')::jsonb->>'huella_efecto_sha256', encode(sha256(p_decision),'hex'), 'aud_b59_recorrido', clock_timestamp(), true
$f$`)

	// 2) Candidata con persona vigente y una participación suya; otra
	// participación de la misma bolsa sin persona vinculada.
	var candidato, persona, pCandidata, bolsa string
	if err := pool.QueryRow(ctx, `SELECT c.candidato_ref, vec_contexto_actor_v1.persona_candidato_avisos_v1(c.candidato_ref), c.participacion_ref, x.bolsa_ref
 FROM vec_bolsa_llamamientos.vinculo_candidato c JOIN vec_bolsa_llamamientos.constitucion x ON x.acta_ref=c.acta_ref
 WHERE vec_contexto_actor_v1.persona_candidato_avisos_v1(c.candidato_ref) IS NOT NULL LIMIT 1`).Scan(&candidato, &persona, &pCandidata, &bolsa); err != nil {
		t.Fatalf("el clon no tiene una candidata con persona vigente: %v", err)
	}
	// Otra candidatura vinculada sin persona vigente en identidad, y una
	// participación sin vínculo de candidato.
	var pSinPersona, pSinVinculo string
	if err := pool.QueryRow(ctx, `SELECT c.participacion_ref FROM vec_bolsa_llamamientos.vinculo_candidato c
 WHERE vec_contexto_actor_v1.persona_candidato_avisos_v1(c.candidato_ref) IS NULL AND c.participacion_ref<>$1 LIMIT 1`, pCandidata).Scan(&pSinPersona); err != nil {
		t.Fatalf("sin candidatura sin persona: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT e.participacion_ref FROM vec_bolsa_llamamientos.constitucion_entrada e
 WHERE NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.vinculo_candidato v WHERE v.participacion_ref=e.participacion_ref) LIMIT 1`).Scan(&pSinVinculo); err != nil {
		t.Fatalf("sin participación sin vínculo: %v", err)
	}

	// 3) «Mis correos» de la candidata: dirección activa cifrada con el
	// adaptador real, añadida y confirmada desde el área personal externa.
	var claveKMS [32]byte
	if _, err := rand.Read(claveKMS[:]); err != nil {
		t.Fatal(err)
	}
	fuente, err := nuevaFuenteClavesCorreosDesarrollo(&emisorKMSDesarrollo{claveEnvoltura: claveKMS})
	if err != nil {
		t.Fatal(err)
	}
	cripto, err := usuariosseguridad.NuevoAdaptadorCorreos(fuente, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	sufijo := make([]byte, 16)
	_, _ = rand.Read(sufijo)
	correoRef := "correo:" + hex.EncodeToString(sufijo)
	activa := "activa." + hex.EncodeToString(sufijo[:3]) + "@personal.example.org"
	sobre, err := cripto.CifrarDireccionCorreo(ctx, persona, correoRef, 1, []byte(activa))
	if err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO vec_usuarios.correos_conjunto VALUES($1,1,$2,clock_timestamp()) ON CONFLICT (persona_ref) DO NOTHING`, persona, sobre.ClaveIgualdadRef)
	exec(`UPDATE vec_usuarios.correos_direccion SET activo=false WHERE persona_ref=$1 AND activo`, persona)
	exec(`INSERT INTO vec_usuarios.correos_direccion(persona_ref,correo_ref,version_sobre,clave_sobre_ref,clave_igualdad_ref,nonce,cifrado,huella_igualdad,estado,activo,creado_en,verificado_en)
 SELECT $1,$2,1,$3,c.clave_igualdad_ref,$4,$5,$6,'verificado',true,clock_timestamp(),clock_timestamp() FROM vec_usuarios.correos_conjunto c WHERE c.persona_ref=$1`,
		persona, correoRef, sobre.ClaveRef, sobre.Nonce, sobre.Cifrado, sobre.HuellaIgualdad)
	desafio, envio := "desafio:"+hex.EncodeToString(sufijo), "correo_envio:"+hex.EncodeToString(sufijo)
	exec(`INSERT INTO vec_usuarios.correos_desafio(persona_ref,correo_ref,desafio_ref,huella_codigo,clave_ref,vence_en,estado,intentos,creado_en)
 VALUES($1,$2,$3,sha256(convert_to($3,'UTF8')),'clave:codigo:recorrido',clock_timestamp()+interval '1 hour','usado',0,clock_timestamp())`, persona, correoRef, desafio)
	exec(`INSERT INTO vec_usuarios.correos_envio(envio_ref,persona_ref,correo_ref,superficie,tipo,desafio_ref,recibo_ref,reserva_sha256,estado,creado_en,resuelto_en)
 VALUES($1,$2,$3,'externa_personal','verificacion',$4,'correo_recibo:'||substr($1,14),encode(sha256(convert_to($1,'UTF8')),'hex'),'aceptado',clock_timestamp(),clock_timestamp())`, envio, persona, correoRef, desafio)

	// 4) Piezas reales: pools con los LOGIN de prueba, adaptadores y SMTP.
	dsnLogin := func(login string) string {
		u, err := url.Parse(admin)
		if err != nil {
			t.Fatal(err)
		}
		u.User = url.User(login)
		return u.String()
	}
	poolBolsa, err := pgxpool.New(ctx, dsnLogin("vec_b59_rec_bolsa"))
	if err != nil {
		t.Fatal(err)
	}
	defer poolBolsa.Close()
	poolUsuarios, err := pgxpool.New(ctx, dsnLogin("vec_b59_rec_usuarios"))
	if err != nil {
		t.Fatal(err)
	}
	repo, err := postgresbolsa.NuevoRepositorioEmisionLlamamientoPostgreSQL(poolBolsa)
	if err != nil || repo.ActivarFuentesCorreo(ctx) != nil {
		t.Fatalf("B59 de Bolsa no instalado: %v", err)
	}
	registro, err := usuariospg.NuevoRegistroCorreoAvisosPostgreSQL(ctx, poolUsuarios)
	if err != nil {
		t.Fatalf("Usuarios 000008 no acreditada: %v", err)
	}
	servicio, err := usuariosapp.NuevoServicioCorreoAvisos(registro, cripto)
	if err != nil {
		t.Fatal(err)
	}
	smtp, err := nuevoEnviadorCorreoLlamamientoDesarrollo(config.Config{SMTPHost: os.Getenv("VEC_B59_RECORRIDO_SMTP_HOST"), SMTPPort: 10025,
		SMTPModoTLS: "starttls", SMTPFrom: "vec-recorrido@dipgra.example.org", SMTPCAFile: os.Getenv("VEC_B59_RECORRIDO_SMTP_CA")})
	if err != nil || smtp == nil {
		t.Fatalf("SMTP: %v", err)
	}
	altas := correosAltaRecorrido{}
	avisador, err := aplicacionbolsa.NuevoAvisadorLlamamiento(altas, &emisorCorreoBolsaB7{smtp})
	if err != nil {
		t.Fatal(err)
	}
	if err := avisador.EstablecerCorreoAvisosPersona(repo, puenteRecorridoB59{servicio: servicio}); err != nil {
		t.Fatal(err)
	}

	// 5) Tres emisiones: con correo activo, sin correo activo y con Usuarios caído.
	actor := "per_" + strings.Repeat("R", 24)
	emitir := func(nombre string) map[string]puertosbolsa.ResultadoContactoEmision {
		t.Helper()
		token := make([]byte, 32)
		_, _ = rand.Read(token)
		sello := sha256.Sum256([]byte(nombre + hex.EncodeToString(sufijo)))
		llamamiento, clave := "llamamiento:"+hex.EncodeToString(sello[:]), "clave-b59-"+hex.EncodeToString(sello[:8])
		participaciones, _ := json.Marshal([]string{pCandidata, pSinPersona, pSinVinculo})
		huellaToken := sha256.Sum256(token)
		exec(`INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido(llamamiento_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,participaciones,configuracion,huella_comando_sha256,huella_finalizacion,estado,emitido_en,decision_ref)
 VALUES($1,'recibo:'||$1,$2,$3,$4,$5::jsonb,'{"plantilla_version":"bolsa-llamamiento-v1"}',$6,$7,'emision_reservada',date_trunc('microseconds',clock_timestamp()),'dec_'||$4)`,
			llamamiento, bolsa, actor, clave, string(participaciones), strings.Repeat("a", 64), huellaToken[:])
		presupuesto, fin := avisador.Presupuesto(ctx)
		defer fin()
		var contactos []puertosbolsa.ResultadoContactoEmision
		for i, p := range []string{pCandidata, pSinPersona, pSinVinculo} {
			resultado, f := avisador.Avisar(ctx, presupuesto, aplicacionbolsa.AvisoLlamamiento{
				BolsaRef: bolsa, UnidadRef: "unidad:rrhh", AmbitoRef: "ambito:bolsa", LlamamientoRef: llamamiento, ParticipacionRef: p,
				Asunto: "Llamamiento " + nombre, Cuerpo: "Aviso de llamamiento (recorrido B59, datos sintéticos).",
				MessageID: fmt.Sprintf("<b59-%s-%s-%d@vec.dipgra.local>", hex.EncodeToString(sufijo[:4]), nombre, i+1), Instante: time.Now().UTC().Truncate(time.Second),
			})
			recibo := sha256.Sum256([]byte(bolsa + "\x1f" + clave + "\x1f" + p))
			contactos = append(contactos, puertosbolsa.ResultadoContactoEmision{ParticipacionRef: p, Resultado: resultado, ReciboRef: "recibo:contacto:" + hex.EncodeToString(recibo[:]), FuenteCorreo: f})
		}
		emitida, err := repo.RegistrarContactos(ctx, bolsa, clave, actor, token, contactos)
		if err != nil {
			t.Fatalf("%s: registrar contactos: %v", nombre, err)
		}
		recuperada, err := repo.Recuperar(ctx, bolsa, clave)
		if err != nil {
			t.Fatalf("%s: recuperar: %v", nombre, err)
		}
		porParticipacion := map[string]puertosbolsa.ResultadoContactoEmision{}
		for i, c := range recuperada.Contactos {
			if c.FuenteCorreo == nil || emitida.Contactos[i].FuenteCorreo == nil || *c.FuenteCorreo != *emitida.Contactos[i].FuenteCorreo {
				t.Fatalf("%s: la recuperación no conserva la fuente: %+v", nombre, c)
			}
			porParticipacion[c.ParticipacionRef] = c
		}
		return porParticipacion
	}
	uno := emitir("activo")
	if c := uno[pCandidata]; c.Resultado != "enviado" || c.FuenteCorreo.Fuente != puertosbolsa.FuenteCorreoMisCorreos || c.FuenteCorreo.CorreoRef != correoRef {
		t.Fatalf("con correo activo: %+v %+v", c, c.FuenteCorreo)
	}
	if c := uno[pSinPersona]; c.Resultado != "enviado" || c.FuenteCorreo.Fuente != puertosbolsa.FuenteCorreoAltaBolsa || c.FuenteCorreo.Motivo != puertosbolsa.MotivoFuenteSinCorreoActivo {
		t.Fatalf("candidatura sin persona: %+v %+v", c, c.FuenteCorreo)
	}
	if c := uno[pSinVinculo]; c.Resultado != "enviado" || c.FuenteCorreo.Motivo != puertosbolsa.MotivoFuenteSinPersonaVinculada {
		t.Fatalf("sin vínculo: %+v %+v", c, c.FuenteCorreo)
	}
	exec(`UPDATE vec_usuarios.correos_direccion SET activo=false WHERE persona_ref=$1 AND correo_ref=$2`, persona, correoRef)
	dos := emitir("sinactivo")
	if c := dos[pCandidata]; c.Resultado != "enviado" || c.FuenteCorreo.Fuente != puertosbolsa.FuenteCorreoAltaBolsa || c.FuenteCorreo.Motivo != puertosbolsa.MotivoFuenteSinCorreoActivo {
		t.Fatalf("sin correo activo: %+v %+v", c, c.FuenteCorreo)
	}
	exec(`UPDATE vec_usuarios.correos_direccion SET activo=true WHERE persona_ref=$1 AND correo_ref=$2`, persona, correoRef)
	poolUsuarios.Close()
	tres := emitir("caido")
	if c := tres[pCandidata]; c.Resultado != "enviado" || c.FuenteCorreo.Motivo != puertosbolsa.MotivoFuenteMisCorreosNoDisponible {
		t.Fatalf("Usuarios caído: %+v %+v", c, c.FuenteCorreo)
	}

	// 6) Mailpit: a quién llegó cada aviso.
	destinos := destinosMailpitRecorrido(t, os.Getenv("VEC_B59_RECORRIDO_MAILPIT_API"))
	id := func(nombre string, i int) string {
		return fmt.Sprintf("<b59-%s-%s-%d@vec.dipgra.local>", hex.EncodeToString(sufijo[:4]), nombre, i)
	}
	esperados := map[string]string{
		id("activo", 1): activa, id("activo", 2): altas.correo(pSinPersona), id("activo", 3): altas.correo(pSinVinculo),
		id("sinactivo", 1): altas.correo(pCandidata), id("caido", 1): altas.correo(pCandidata),
	}
	for id, destino := range esperados {
		if destinos[id] != destino {
			t.Fatalf("Mailpit %s: llegó a %q, se esperaba %q", id, destinos[id], destino)
		}
	}
	t.Logf("B59 recorrido: con activo→%s (%s); sin activo→%s; candidatura sin persona→%s; sin vínculo→%s; Usuarios caído→%s",
		uno[pCandidata].FuenteCorreo.Fuente, uno[pCandidata].FuenteCorreo.CorreoRef, dos[pCandidata].FuenteCorreo.Motivo,
		uno[pSinPersona].FuenteCorreo.Motivo, uno[pSinVinculo].FuenteCorreo.Motivo, tres[pCandidata].FuenteCorreo.Motivo)
}

// correosAltaRecorrido sustituye el dato de alta: el del clon está cifrado con
// la KMS de la principal, que no existe en local.
type correosAltaRecorrido struct{}

func (correosAltaRecorrido) correo(p string) string {
	h := sha256.Sum256([]byte(p))
	return "alta." + hex.EncodeToString(h[:3]) + "@bolsa.example.org"
}
func (c correosAltaRecorrido) CorreoParticipacion(_ context.Context, p string) (string, error) {
	return c.correo(p), nil
}

// puenteRecorridoB59 es el puente de la composición con la V3 sintética que
// acepta el doble de la fachada AD3-109 del clon.
type puenteRecorridoB59 struct {
	servicio *usuariosapp.ServicioCorreoAvisos
}

func (p puenteRecorridoB59) ConCorreoAvisoPersona(ctx context.Context, s puertosbolsa.SolicitudCorreoAvisoPersona, usar func(string)) (bool, string, error) {
	r, err := p.servicio.ConCorreoActivoAvisos(ctx, usuariosports.OrdenCorreoAvisos{Proveedor: proveedorV3SinteticoRecorrido{}}, usuariosports.SolicitudCorreoAvisos{
		BolsaRef: s.BolsaRef, UnidadRef: s.UnidadRef, AmbitoRef: s.AmbitoRef, LlamamientoRef: s.LlamamientoRef, CandidatoRef: s.CandidatoRef,
	}, usar)
	return r.Encontrado, r.CorreoRef, err
}

type proveedorV3SinteticoRecorrido struct{}

func (proveedorV3SinteticoRecorrido) ProveerMaterialCorreoAvisos(_ context.Context, m usuariosports.MaterialCorreoAvisos, _ []byte) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	recurso, _, err := canonico.RecursoCorreoAvisos(m)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	huella, _ := recurso.HuellaContextoAutorizacionSHA256()
	n := make([]byte, 8)
	_, _ = rand.Read(n)
	capacidad, _ := json.Marshal(map[string]any{"audiencia_consumo": usuariosports.AudienciaCorreoAvisosLlamamientoInterna, "operacion": usuariosports.AccionV3CorreoAvisosLlamamiento, "efecto_ref": m.BolsaRef, "huella_efecto_sha256": huella})
	decision, _ := json.Marshal(map[string]any{"decision_ref": "dec_b59_" + hex.EncodeToString(n), "accion": usuariosports.AccionV3CorreoAvisosLlamamiento, "modulo_id": "bolsa", "tipo_recurso": "bolsa_constituida",
		"finalidad": usuariosports.FinalidadCorreoAvisosLlamamiento, "recurso_ref": m.BolsaRef, "concedida": true, "contexto_recurso_huella_sha256": huella,
		"vinculo_autenticacion_actor": map[string]string{"superficie": "interna_corporativa"}})
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_b59", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_b59", strings.Repeat("c", 64), usuariosports.AccionV3CorreoAvisosLlamamiento, m.BolsaRef, huella, usuariosports.AudienciaCorreoAvisosLlamamientoInterna, ahora, ahora.Add(3*time.Second))
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(append(capacidad, bytes.Repeat([]byte(" "), 512)...), resumen, decision, []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
}

func destinosMailpitRecorrido(t *testing.T, api string) map[string]string {
	t.Helper()
	respuesta, err := http.Get(strings.TrimRight(api, "/") + "/api/v1/messages?limit=200")
	if err != nil {
		t.Fatal(err)
	}
	defer respuesta.Body.Close()
	cuerpo, _ := io.ReadAll(io.LimitReader(respuesta.Body, 4<<20))
	var lista struct {
		Messages []struct {
			MessageID string `json:"MessageID"`
			To        []struct {
				Address string `json:"Address"`
			} `json:"To"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(cuerpo, &lista); err != nil {
		t.Fatal(err)
	}
	destinos := map[string]string{}
	for _, m := range lista.Messages {
		if len(m.To) == 1 {
			destinos["<"+strings.Trim(m.MessageID, "<>")+">"] = m.To[0].Address
		}
	}
	return destinos
}
