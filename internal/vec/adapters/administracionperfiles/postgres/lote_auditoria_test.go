package postgres

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

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
