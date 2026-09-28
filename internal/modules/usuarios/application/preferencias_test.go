package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorPrueba struct {
	materiales []ports.MaterialPreferencias
	denegar    bool
	denegarEn  int
	reutilizar bool
	primera    vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (p *proveedorPrueba) ProveerMaterialPreferencias(_ context.Context, m ports.MaterialPreferencias) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.materiales = append(p.materiales, m)
	if p.denegar || p.denegarEn == len(p.materiales) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrProhibido
	}
	if p.reutilizar && len(p.materiales) > 1 {
		return p.primera, nil
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(fmt.Sprintf("dec_prueba_%d", len(p.materiales)), strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), m.Accion, m.PersonaRef, strings.Repeat("d", 64), "usuarios_preferencias", ahora, ahora.Add(3*time.Second))
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	raiz, err := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	v3, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
	if err == nil && len(p.materiales) == 1 {
		p.primera = v3
	}
	return v3, err
}

type registroPrueba struct {
	catalogo                             domain.CatalogoPreferencias
	estado                               ports.EstadoPreferencias
	existe                               bool
	recibo                               ports.ReciboPreferencias
	replay                               bool
	errorGuardar                         error
	errorRecuperar                       error
	consultas, recuperaciones, guardados int
	material                             ports.MaterialPreferencias
	huellaRecuperacion, huellaGuardado   string
}

func (r *registroPrueba) CatalogoVigente(context.Context) (domain.CatalogoPreferencias, error) {
	return r.catalogo, nil
}
func (r *registroPrueba) ConsultarPropias(_ context.Context, _ ports.OrdenPreferencias, m ports.MaterialPreferencias, _ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.EstadoPreferencias, bool, error) {
	r.consultas++
	r.material = m
	return r.estado, r.existe, nil
}
func (r *registroPrueba) RecuperarOperacion(_ context.Context, _ ports.OrdenPreferencias, m ports.MaterialPreferencias, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboPreferencias, bool, error) {
	r.recuperaciones++
	r.huellaRecuperacion, _ = v3.HuellaConjuntoSHA256()
	r.material = m
	return r.recibo, r.replay, r.errorRecuperar
}
func (r *registroPrueba) Guardar(_ context.Context, _ ports.OrdenPreferencias, _ ports.PeticionGuardarPreferencias, m ports.MaterialPreferencias, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboPreferencias, error) {
	r.guardados++
	r.huellaGuardado, _ = v3.HuellaConjuntoSHA256()
	r.material = m
	return r.recibo, r.errorGuardar
}

func ordenPrueba(t *testing.T, p *proveedorPrueba) (ports.OrdenPreferencias, vecdomain.ContextoActor) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_0123456789abcdefghijkl", Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	snap := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 1, PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := vecdomain.NuevoContextoActor(cuenta, snap, ahora)
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenPreferencias(actor, p)
	if err != nil {
		t.Fatal(err)
	}
	return orden, actor
}

func TestGETAusenteDevuelveDefaultsSinEscritura(t *testing.T) {
	p := &proveedorPrueba{}
	o, actor := ordenPrueba(t, p)
	r := &registroPrueba{catalogo: domain.CatalogoBasePreferencias()}
	s, _ := NuevoServicioPreferencias(r, time.Now)
	vista, err := s.Consultar(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if vista.Estado.Version != 0 || vista.Estado.PersonaRef != actor.PersonaRef || vista.Estado.Valores != r.catalogo.Predeterminados || r.guardados != 0 || r.consultas != 1 {
		t.Fatalf("GET incorrecto: %#v %#v", vista, r)
	}
	if r.material.Accion != ports.AccionConsultarPreferencias || r.material.FinalidadRef != ports.FinalidadPreferenciasPropias {
		t.Fatal("lectura sin permiso/finalidad nominal")
	}
}

func TestPUTDelegaCASYRecuperaReciboOriginal(t *testing.T) {
	p := &proveedorPrueba{}
	o, actor := ordenPrueba(t, p)
	c := domain.CatalogoBasePreferencias()
	fecha := time.Now().UTC()
	r := &registroPrueba{catalogo: c, recibo: ports.ReciboPreferencias{ReciboRef: "recibo:uno", PersonaRef: actor.PersonaRef, Version: 1, CatalogoVersionRef: c.VersionRef, Valores: c.Predeterminados, FechaUTC: fecha}}
	s, _ := NuevoServicioPreferencias(r, time.Now)
	pet := ports.PeticionGuardarPreferencias{VersionEsperada: 0, CatalogoVersionRef: c.VersionRef, ClaveOperacion: "operacion-1234567890", Valores: c.Predeterminados}
	recibo, err := s.Guardar(context.Background(), o, pet)
	if err != nil {
		t.Fatal(err)
	}
	if recibo.ReciboRef != "recibo:uno" || r.guardados != 1 || r.recuperaciones != 1 || r.material.PersonaRef != actor.PersonaRef || r.material.Accion != ports.AccionActualizarPreferencias || len(r.material.HuellaPeticion) != 64 || len(p.materiales) != 2 || r.huellaRecuperacion == r.huellaGuardado || r.huellaGuardado == "" {
		t.Fatal("CAS/material no delegados")
	}
	r.replay = true
	r.recibo.Version = 1
	r.recibo.FechaUTC = fecha
	r.guardados = 0
	pet.VersionEsperada = 0
	recibo, err = s.Guardar(context.Background(), o, pet)
	if err != nil {
		t.Fatal(err)
	}
	if !recibo.Replay || recibo.ReciboRef != "recibo:uno" || !recibo.FechaUTC.Equal(fecha) || r.guardados != 0 || len(p.materiales) != 3 {
		t.Fatal("replay reescribio estado o recibo")
	}
	r.replay = false
	r.errorGuardar = ports.ErrConflicto
	_, err = s.Guardar(context.Background(), o, pet)
	if !errors.Is(err, ports.ErrConflicto) {
		t.Fatalf("CAS no propagado: %v", err)
	}
}

func TestDenegacionYValidacionPreviasAlRepositorio(t *testing.T) {
	p := &proveedorPrueba{denegar: true}
	o, _ := ordenPrueba(t, p)
	r := &registroPrueba{catalogo: domain.CatalogoBasePreferencias()}
	s, _ := NuevoServicioPreferencias(r, time.Now)
	if _, err := s.Consultar(context.Background(), o); !errors.Is(err, ports.ErrProhibido) || r.consultas != 0 {
		t.Fatalf("lectura no denegada: %v", err)
	}
	pet := ports.PeticionGuardarPreferencias{CatalogoVersionRef: r.catalogo.VersionRef, ClaveOperacion: "operacion-1234567890", Valores: r.catalogo.Predeterminados}
	if _, err := s.Guardar(context.Background(), o, pet); !errors.Is(err, ports.ErrProhibido) || r.recuperaciones != 0 {
		t.Fatalf("escritura no denegada: %v", err)
	}
	p.denegar = false
	pet.Valores.Tema = "url:libre"
	if _, err := s.Guardar(context.Background(), o, pet); !errors.Is(err, ports.ErrPeticionInvalida) || r.recuperaciones != 0 {
		t.Fatalf("codigo libre aceptado: %v", err)
	}
}

func TestCatalogoObsoletoYClaveReutilizada(t *testing.T) {
	p := &proveedorPrueba{}
	o, _ := ordenPrueba(t, p)
	c := domain.CatalogoBasePreferencias()
	r := &registroPrueba{catalogo: c}
	s, _ := NuevoServicioPreferencias(r, time.Now)
	pet := ports.PeticionGuardarPreferencias{VersionEsperada: 0, CatalogoVersionRef: "usuarios-preferencias-v0", ClaveOperacion: "operacion-1234567890", Valores: c.Predeterminados}
	if _, err := s.Guardar(context.Background(), o, pet); !errors.Is(err, ports.ErrConflicto) || r.guardados != 0 || r.recuperaciones != 1 {
		t.Fatalf("catalogo obsoleto no rechazado: %v", err)
	}
	pet.CatalogoVersionRef = c.VersionRef
	r.errorRecuperar = ports.ErrConflicto
	if _, err := s.Guardar(context.Background(), o, pet); !errors.Is(err, ports.ErrConflicto) || r.guardados != 0 {
		t.Fatalf("reuso de clave no rechazado: %v", err)
	}
}

func TestFalloSegundaCapacidadNoGuarda(t *testing.T) {
	p := &proveedorPrueba{denegarEn: 2}
	o, _ := ordenPrueba(t, p)
	c := domain.CatalogoBasePreferencias()
	r := &registroPrueba{catalogo: c}
	s, _ := NuevoServicioPreferencias(r, time.Now)
	pet := ports.PeticionGuardarPreferencias{CatalogoVersionRef: c.VersionRef, ClaveOperacion: "operacion-1234567890", Valores: c.Predeterminados}
	_, err := s.Guardar(context.Background(), o, pet)
	if !errors.Is(err, ports.ErrProhibido) || len(p.materiales) != 2 || r.recuperaciones != 1 || r.guardados != 0 {
		t.Fatalf("segunda capacidad denegada no cerró escritura: %v", err)
	}
}

func TestNoReutilizaCapacidadConsumida(t *testing.T) {
	p := &proveedorPrueba{reutilizar: true}
	o, _ := ordenPrueba(t, p)
	c := domain.CatalogoBasePreferencias()
	r := &registroPrueba{catalogo: c}
	s, _ := NuevoServicioPreferencias(r, time.Now)
	pet := ports.PeticionGuardarPreferencias{CatalogoVersionRef: c.VersionRef, ClaveOperacion: "operacion-1234567890", Valores: c.Predeterminados}
	_, err := s.Guardar(context.Background(), o, pet)
	if !errors.Is(err, ports.ErrNoDisponible) || len(p.materiales) != 2 || r.recuperaciones != 1 || r.guardados != 0 {
		t.Fatalf("capacidad V3 consumida reutilizada: %v", err)
	}
}
