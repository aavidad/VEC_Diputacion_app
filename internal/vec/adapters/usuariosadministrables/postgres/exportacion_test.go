package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Construye solo un transporte V3 estructural para demostrar el rechazo Go.
// No tiene COSE ni decisión favorable y nunca se entrega a PostgreSQL.
type emisorContextoCruzadoPrueba struct {
	t             *testing.T
	otra          domain.EvidenciaSesionAdministracionPerfiles
	ahora         time.Time
	llamadas      int
	fechaInvalida string
}

func (e *emisorContextoCruzadoPrueba) EmitirLecturaUsuariosAdministrables(_ context.Context, actor domain.ContextoActor, _ domain.EvidenciaSesionAdministracionPerfiles, _ domain.InstantaneaAutorizacion, p ports.EmisionUsuariosAdministrables) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	e.t.Helper()
	e.llamadas++
	r := p.Recurso
	huella, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		e.t.Fatal(err)
	}
	motivo := []byte(`{}`)
	contexto := e.otra.ResultadoContexto.RepresentacionCanonica
	vinculo, err := e.otra.Vinculo.Datos()
	if err != nil {
		e.t.Fatal(err)
	}
	correlacion, err := p.Correlacion.ValorCanonico()
	if err != nil {
		e.t.Fatal(err)
	}
	decisionRef := "decision:usuarios:vector-cruzado"
	campos := []string{"denominacion_version", "perfiles", "persona_ref", "siguiente_cursor", "unidad_ref"}
	vinculoDecision := any(vinculo)
	if e.fechaInvalida != "" {
		b, err := json.Marshal(vinculo)
		if err != nil {
			e.t.Fatal(err)
		}
		var camposVinculo map[string]any
		if err := json.Unmarshal(b, &camposVinculo); err != nil {
			e.t.Fatal(err)
		}
		camposVinculo["autenticacion_verificada_en"] = e.fechaInvalida
		vinculoDecision = camposVinculo
	}
	decision, _ := json.Marshal(map[string]any{"decision_ref": decisionRef, "concedida": true, "principal_id": actor.PersonaRef, "perfil_activo_ref": actor.PerfilActivoRef,
		"correlacion_ref": correlacion, "accion": p.Accion, "modulo_id": "administracion", "tipo_recurso": r.Tipo,
		"recurso_ref": r.Referencia, "contexto_recurso_huella_sha256": huella, "finalidad": "gestion_usuarios", "garantia_minima": "alto",
		"version_rol_ref": "rol:administracion_perfiles:v5", "campos_permitidos": campos, "obligaciones": []string{"auditar"}, "vinculo_autenticacion_actor": vinculoDecision})
	hd := sha256.Sum256(decision)
	hm := sha256.Sum256(motivo)
	hc := sha256.Sum256(contexto)
	capacidad, _ := json.Marshal(map[string]any{"operacion": p.Accion, "audiencia_consumo": p.Audiencia, "efecto_ref": r.Referencia,
		"huella_efecto_sha256": huella, "decision_ref": decisionRef, "huella_decision_sha256": hex.EncodeToString(hd[:]),
		"contexto_ref": e.otra.ResultadoContexto.RegistroContextoRef, "huella_contexto_sha256": hex.EncodeToString(hc[:]), "relleno": strings.Repeat("x", 512)})
	resumen, err := ports.NuevoResumenCapacidadAtestacionAutorizacionV3(decisionRef, hex.EncodeToString(hd[:]), hex.EncodeToString(hm[:]),
		e.otra.ResultadoContexto.RegistroContextoRef, hex.EncodeToString(hc[:]), p.Accion, r.Referencia, huella, p.Audiencia, e.ahora.Add(-time.Millisecond), e.ahora.Add(2*time.Second))
	if err != nil {
		e.t.Fatal(err)
	}
	publica, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		e.t.Fatal(err)
	}
	raiz, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		e.t.Fatal(err)
	}
	m, err := ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(capacidad, resumen, decision, motivo, contexto,
		actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion, []byte("payload_sintetico"), []byte("sobre_sintetico"), []byte("evidencia_sintetica"), raiz)
	if err != nil {
		e.t.Fatal(err)
	}
	return m, nil
}

func TestFechaV2MalformadaNoSaleEnErrorNiAuditoria(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	actor, evidencia := evidenciaAdminPrueba(t, ahora)
	a := ambito{"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "unidad_admin_sintetica"}
	const secreto = "SECRETfechaPERSONAL"
	emisor := &emisorContextoCruzadoPrueba{t: t, otra: evidencia, ahora: ahora, fechaInvalida: secreto}
	registrador := &capturadorIntentoUsuarios{ahora: ahora}
	f := &Fuente{ambito: a, config: Configuracion{Proceso: "vec_admin", Canal: "administracion_privilegiada", MotivoDenegado: motivoIntentoPrueba(), MotivoError: motivoIntentoPrueba()},
		pool: &acreditacionPoolPrueba{}, fuente: fuenteSnapshotPrueba{snapshotAdminPrueba(t, actor, ahora, a)}, emisor: emisor, intentos: registrador, reloj: relojAdminPrueba{ahora}}
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	x, err := f.ListarUsuarios(ctx, actor, evidencia, ports.FiltrosUsuariosAdministrables{})
	if !errors.Is(err, ports.ErrLecturaUsuariosAdministrablesNoDisponible) || strings.Contains(err.Error(), secreto) || len(x.Personas) != 0 ||
		emisor.llamadas != 1 || len(registrador.datos) != 1 || registrador.datos[0].Resultado != domain.ResultadoIntentoAuditoriaError ||
		strings.Contains(fmt.Sprint(registrador.datos), secreto) {
		t.Fatalf("fecha privada filtrada o no auditada: err=%v llamadas=%d", err, emisor.llamadas)
	}
}

func TestOtroRegistroV2MismaPersonaPerfilNoLlegaADB(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	actor, original := evidenciaAdminPrueba(t, ahora)
	otroActor, otro := evidenciaAdminPrueba(t, ahora.Add(-time.Second))
	if otroActor.PersonaRef != actor.PersonaRef || otroActor.PerfilActivoRef != actor.PerfilActivoRef || otroActor.Instantanea.PersonaVersion != actor.Instantanea.PersonaVersion || otroActor.Instantanea.PerfilVersion != actor.Instantanea.PerfilVersion || otro.ResultadoContexto.HuellaSHA256 == original.ResultadoContexto.HuellaSHA256 {
		t.Fatal("el vector no separa dos registros V2 de la misma persona y perfil")
	}
	a := ambito{"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "unidad_admin_sintetica"}
	emisor := &emisorContextoCruzadoPrueba{t: t, otra: otro, ahora: ahora}
	registrador := &capturadorIntentoUsuarios{ahora: ahora}
	f := &Fuente{ambito: a, config: Configuracion{Proceso: "vec_admin", Canal: "administracion_privilegiada", MotivoDenegado: motivoIntentoPrueba(), MotivoError: motivoIntentoPrueba()},
		pool: &acreditacionPoolPrueba{}, fuente: fuenteSnapshotPrueba{snapshotAdminPrueba(t, actor, ahora, a)}, emisor: emisor, intentos: registrador, reloj: relojAdminPrueba{ahora}}
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// La exportación es coherente con el OTRO resultado V2. El único cambio
	// al ejercer la fuente es el registro original de esta lectura.
	p, err := materialListar(a, ports.FiltrosUsuariosAdministrables{})
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	p.correlacion, err = correlacion.ValorCanonico()
	if err != nil {
		t.Fatal(err)
	}
	m, err := emisor.EmitirLecturaUsuariosAdministrables(ctx, otroActor, otro, domain.InstantaneaAutorizacion{}, ports.EmisionUsuariosAdministrables{
		Material: p.material, Recurso: p.recurso, Accion: p.accion, Audiencia: p.audiencia, Correlacion: correlacion})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = validarExportacion(m, otroActor, otro, &p, ahora); err != nil {
		t.Fatalf("el vector no es coherente con su propio V2: %v", err)
	}
	emisor.llamadas = 0
	x, err := f.ListarUsuarios(ctx, actor, original, ports.FiltrosUsuariosAdministrables{})
	if !errors.Is(err, ports.ErrLecturaUsuariosAdministrablesNoDisponible) || len(x.Personas) != 0 || emisor.llamadas != 1 || len(registrador.datos) != 1 || registrador.datos[0].Resultado != domain.ResultadoIntentoAuditoriaError || original.ValidarPara(actor) != nil {
		t.Fatalf("contexto cruzado: err=%v llamadas=%d auditoría=%+v", err, emisor.llamadas, registrador.datos)
	}
}
