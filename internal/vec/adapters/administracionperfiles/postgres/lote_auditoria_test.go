package postgres

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func auditorLotePrueba(registrador ports.RegistradorIntentosAuditoria) *AutoridadLoteOrdinario {
	f := configFronteraPrueba()
	return &AutoridadLoteOrdinario{registrador: registrador, auditoria: ConfiguracionAuditoriaLote{
		Proceso: f.Proceso, Canal: f.Canal, MotivoDenegado: f.MotivoDenegado,
		MotivoError: f.MotivoError, Plazo: f.Plazo}}
}

func solicitudAuditoriaLotePrueba(t *testing.T) domain.SolicitudLoteAdministracionPerfiles {
	t.Helper()
	s, _, _ := solicitudLoteOrdinarioPrueba(t)
	s.Actor, s.Evidencia = sesionFronteraPrueba(t, time.Now().UTC().Truncate(time.Microsecond))
	return s
}

func TestLoteFalloConV2OriginalRegistraTrasCancelacionYRedactaDestino(t *testing.T) {
	s := solicitudAuditoriaLotePrueba(t)
	original := append([]byte(nil), s.Evidencia.ResultadoContexto.RepresentacionCanonica...)
	s.Cambios[0].Objetivo.PersonaRef = "SECRET/ajeno?consulta=1"
	espia := &registroFronteraPrueba{ahora: time.Now().UTC()}
	a := auditorLotePrueba(espia)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := a.finalizarFalloLote(ctx, s, domain.ErrActoAdministracionPerfilesInvalido)
	if !errors.Is(err, domain.ErrActoAdministracionPerfilesInvalido) || espia.llamadas != 1 ||
		espia.cancelado || !espia.plazo || !bytes.Equal(original, s.Evidencia.ResultadoContexto.RepresentacionCanonica) {
		t.Fatal("intento comun no conserva V2 original tras fallo")
	}
	d := espia.ordenes[0].Datos
	if d.Resultado != domain.ResultadoIntentoAuditoriaDenegado || !strings.HasPrefix(d.RecursoRef, "solicitud_admin:") ||
		strings.Contains(d.RecursoRef, "SECRET") || d.Accion != accionLoteOrdinario ||
		espia.ordenes[0].ResultadoContexto.Contexto.PersonaRef != s.Actor.PersonaRef {
		t.Fatal("el intento contiene datos de solicitud o resultado incorrecto")
	}
}

func TestLoteCommitIndeterminadoNoEntregaReciboNiReintentaNegocio(t *testing.T) {
	s := solicitudAuditoriaLotePrueba(t)
	espia := &registroFronteraPrueba{ahora: time.Now().UTC()}
	a := auditorLotePrueba(espia)
	err := a.finalizarFalloLote(context.Background(), s, errCommitLoteIndeterminado)
	if !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) || espia.llamadas != 1 ||
		espia.ordenes[0].Datos.Resultado != domain.ResultadoIntentoAuditoriaError {
		t.Fatal("commit indeterminado se declaró denegado o se reintentó")
	}
}

func TestLoteAcuseComunIncompatibleCierraResultado(t *testing.T) {
	s := solicitudAuditoriaLotePrueba(t)
	espia := &registroFronteraPrueba{}
	a := auditorLotePrueba(espia)
	err := a.finalizarFalloLote(context.Background(), s, domain.ErrControlAdministracionPerfilesInvalido)
	if !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) || espia.llamadas != 1 {
		t.Fatal("acuse no confirmado permitió respuesta de dominio")
	}
}

func TestLoteSinOriginalV2NoInventaIntento(t *testing.T) {
	s := solicitudAuditoriaLotePrueba(t)
	s.Evidencia = domain.EvidenciaSesionAdministracionPerfiles{}
	espia := &registroFronteraPrueba{ahora: time.Now().UTC()}
	a := auditorLotePrueba(espia)
	err := a.finalizarFalloLote(context.Background(), s, domain.ErrActoAdministracionPerfilesInvalido)
	if !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) || espia.llamadas != 0 {
		t.Fatal("se inventó identidad para intento sin V2")
	}
}

func TestLoteConstructorExigeRegistroComunYPlazoPrivado(t *testing.T) {
	f := configFronteraPrueba()
	cfg := ConfiguracionAuditoriaLote{Proceso: f.Proceso, Canal: f.Canal,
		MotivoDenegado: f.MotivoDenegado, MotivoError: f.MotivoError, Plazo: f.Plazo}
	pool := &poolFalso{}
	emisor := &emisorLotePrueba{}
	if a, err := nuevaAutoridadLoteOrdinario(context.Background(), pool, emisor, fuenteLotePrueba{},
		nil, cfg, "org_prueba", relojFijo(time.Now().UTC())); a != nil || err == nil {
		t.Fatal("constructor aceptó lote sin AD169")
	}
	cfg.Plazo = 2*time.Second + time.Nanosecond
	if a, err := nuevaAutoridadLoteOrdinario(context.Background(), pool, emisor, fuenteLotePrueba{},
		&registroFronteraPrueba{}, cfg, "org_prueba", relojFijo(time.Now().UTC())); a != nil || err == nil {
		t.Fatal("constructor aceptó plazo no acotado")
	}
}

type emisorLoteMaterialPrueba struct {
	material ports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (e emisorLoteMaterialPrueba) EmitirLoteOrdinario(context.Context, domain.ContextoActor,
	domain.EvidenciaSesionAdministracionPerfiles, domain.InstantaneaAutorizacion,
	domain.RecursoAutorizable, Efecto) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return e.material, nil
}

type registroLoteTrasTxPrueba struct {
	t         *testing.T
	tx        *txFalsa
	rollbacks int
	reg       *registroFronteraPrueba
}

func (r registroLoteTrasTxPrueba) AppendIntentoAuditoria(ctx context.Context, orden ports.OrdenIntentoAuditoria) (ports.AcuseIntentoAuditoria, error) {
	r.t.Helper()
	if r.tx.rollbacks != r.rollbacks {
		r.t.Fatal("AD169 se invocó antes del rollback del efecto")
	}
	return r.reg.AppendIntentoAuditoria(ctx, orden)
}

type poolLoteFasesPrueba struct {
	poolFalso
	falloInicio error
}

func (p *poolLoteFasesPrueba) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	if p.falloInicio != nil {
		p.comienzos++
		return nil, p.falloInicio
	}
	return p.poolFalso.BeginTx(ctx, o)
}

// El material del doble sólo satisface la estructura del transporte. AUT44
// tendría que verificar su firma y gobierno en PostgreSQL antes de un efecto.
func materialLoteEstructuralPrueba(t *testing.T, s domain.SolicitudLoteAdministracionPerfiles,
	recurso domain.RecursoAutorizable, efecto Efecto, ahora time.Time) ports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	r, err := ports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:ejemplo", strings.Repeat("a", 64),
		strings.Repeat("b", 64), s.Evidencia.ResultadoContexto.RegistroContextoRef,
		s.Evidencia.ResultadoContexto.HuellaSHA256, efecto.Accion, efecto.Referencia, h,
		efecto.Audiencia, ahora.Add(-time.Second), ahora.Add(4*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := x509.MarshalPKIXPublicKey(ed25519.PublicKey(make([]byte, ed25519.PublicKeySize)))
	if err != nil {
		t.Fatal(err)
	}
	material, err := ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(make([]byte, 512), r,
		[]byte("decision"), []byte("motivo"), s.Evidencia.ResultadoContexto.RepresentacionCanonica,
		s.Actor.Instantanea.PersonaVersion, s.Actor.Instantanea.PerfilVersion,
		[]byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return material
}

func TestLoteFasesSQLYCommitAntesDeAD169(t *testing.T) {
	for _, caso := range []struct {
		nombre           string
		fila             pgx.Row
		falloInicio      error
		falloCommit      error
		consultas        int
		commits          int
		rollbacks        int
		resultadoIntento domain.ResultadoIntentoAuditoria
		errorPublico     error
	}{
		{"rollback_sql", filaFalsa{err: errors.New("SQL privado")}, nil, nil, 1, 0, 1,
			domain.ResultadoIntentoAuditoriaError, ports.ErrAutoridadAdministracionPerfilesNoDisponible},
		{"query_42501", filaFalsa{err: &pgconn.PgError{Code: "42501", Message: "SQL privado"}}, nil, nil, 1, 0, 1,
			domain.ResultadoIntentoAuditoriaDenegado, domain.ErrAutorizacionDenegada},
		{"begin_42501", nil, &pgconn.PgError{Code: "42501", Message: "BEGIN privado"}, nil, 0, 0, 0,
			domain.ResultadoIntentoAuditoriaError, ports.ErrAutoridadAdministracionPerfilesNoDisponible},
		{"commit_42501", filaFalsa{dato: []byte(`{}`)}, nil, &pgconn.PgError{Code: "42501", Message: "COMMIT privado"}, 1, 1, 1,
			domain.ResultadoIntentoAuditoriaError, ports.ErrAutoridadAdministracionPerfilesNoDisponible},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			s := solicitudAuditoriaLotePrueba(t)
			ahora := time.Now().UTC().Truncate(time.Microsecond)
			recurso := domain.RecursoAutorizable{Referencia: s.Cambios[0].Objetivo.PersonaRef,
				ModuloID: "administracion", Tipo: "persona",
				Ambitos:   map[string]string{"organizacion_ref": "org_prueba", "unidad_ref": "unidad:prueba"},
				Atributos: map[string]string{"solicitud_sha256": strings.Repeat("a", 64)}}
			efecto := Efecto{Accion: accionLoteOrdinario, Audiencia: audienciaLoteOrdinario,
				Referencia: recurso.Referencia, Material: []byte(`{}`), CorrelacionAccesoRef: s.CorrelacionRef}
			tx := &txFalsa{fila: caso.fila, falloCommit: caso.falloCommit}
			pool := &poolLoteFasesPrueba{poolFalso: poolFalso{tx: tx}, falloInicio: caso.falloInicio}
			reg := &registroFronteraPrueba{ahora: ahora}
			a := auditorLotePrueba(registroLoteTrasTxPrueba{t: t, tx: tx, rollbacks: caso.rollbacks, reg: reg})
			a.pool, a.emisor, a.reloj = pool,
				emisorLoteMaterialPrueba{materialLoteEstructuralPrueba(t, s, recurso, efecto, ahora)}, relojFijo(ahora)
			err := a.ejecutarLote(context.Background(), s.Actor, s.Evidencia, s.InstantaneaAutorizacion,
				recurso, efecto, nil, func([]byte) error { return nil })
			if err == nil || tx.rollbacks != caso.rollbacks || tx.commits != caso.commits || tx.consultas != caso.consultas || pool.comienzos != 1 {
				t.Fatal("el efecto no cerró una sola transacción antes de AD169")
			}
			publico := a.finalizarFalloLote(context.Background(), s, err)
			if !errors.Is(publico, caso.errorPublico) ||
				reg.llamadas != 1 || reg.ordenes[0].Datos.Resultado != caso.resultadoIntento ||
				strings.Contains(publico.Error(), "privado") {
				t.Fatal("fallo transaccional no quedó auditado con error cerrado")
			}
		})
	}
}
