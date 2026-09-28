package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	materiales       []ports.MaterialPreferencias
	denegar          bool
	denegarEn        int
	reutilizar       bool
	audienciaForzada string
	primera          vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (p *proveedorPrueba) ProveerMaterialPreferencias(_ context.Context, vinculo vecdomain.VinculoAutenticacionActorV2, m ports.MaterialPreferencias) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	datos, err := vinculo.Datos()
	if err != nil || datos.Superficie != m.Superficie {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrProhibido
	}
	p.materiales = append(p.materiales, m)
	if p.denegar || p.denegarEn == len(p.materiales) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrProhibido
	}
	if p.reutilizar && len(p.materiales) > 1 {
		return p.primera, nil
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	audiencia, _ := ports.AudienciaPreferencias(m.Accion, m.Superficie)
	if p.audienciaForzada != "" {
		audiencia = p.audienciaForzada
	}
	canon, _ := json.Marshal(m)
	huella := sha256.Sum256(canon)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(fmt.Sprintf("dec_prueba_%d", len(p.materiales)), strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), m.Accion, m.PersonaRef, hex.EncodeToString(huella[:]), audiencia, ahora, ahora.Add(3*time.Second))
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

func (r *registroPrueba) CatalogoVigente(context.Context, ports.OrdenPreferencias) (domain.CatalogoPreferencias, error) {
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

type revalidadorPrefPrueba struct {
	a vecdomain.AutenticacionRevalidadaV1
}

func (r revalidadorPrefPrueba) RevalidarAutenticacionActorV1(context.Context, vecdomain.SolicitudRevalidacionAutenticacionActorV1) (vecdomain.AutenticacionRevalidadaV1, error) {
	return r.a, nil
}

type resolutorPrefPrueba struct {
	r vecdomain.ResultadoContextoActorRegistradoV2
}

func (r resolutorPrefPrueba) ResolverContextoActorRegistradoV2(context.Context, vecdomain.SolicitudContextoActor) (vecdomain.ResultadoContextoActorRegistradoV2, error) {
	return r.r, nil
}

type relojPrefPrueba struct{ ahora time.Time }

func (r relojPrefPrueba) Ahora() time.Time { return r.ahora }

func identidadPrueba(t *testing.T, superficie vecdomain.SuperficieAutenticacionActorV1, personaLetra, perfilLetra string) (vecdomain.ContextoActor, vecdomain.VinculoAutenticacionActorV2) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	z := strings.Repeat("a", 24)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	snap := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + strings.Repeat(personaLetra, 24), PersonaVersion: 1, PerfilActivoRef: "prf_" + strings.Repeat(perfilLetra, 24), PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := vecdomain.NuevoContextoActor(cuenta, snap, ahora)
	if err != nil {
		t.Fatal(err)
	}
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	huella, _ := actor.HuellaSHA256VinculadaV2()
	ac := vecdomain.AcreditacionProcedenciaComponenteContextoActorV1{ProcedenciaRef: "prc_" + z, ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("a", 64), ProcedenciaAutoridad: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1}
	man := vecdomain.ManifiestoProcedenciaContextoActorV1{Esquema: vecdomain.EsquemaManifiestoProcedenciaContextoActorV1, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, Cuenta: vecdomain.ProcedenciaCuentaContextoActorV1{CuentaRef: cuenta.CuentaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Persona: vecdomain.ProcedenciaPersonaContextoActorV1{PersonaRef: actor.PersonaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Perfil: vecdomain.ProcedenciaPerfilContextoActorV1{PerfilRef: actor.PerfilActivoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Contexto: vecdomain.ProcedenciaVinculoContextoActorV1{VinculoRef: snap.VinculoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: ac}, Vinculos: []vecdomain.ProcedenciaVinculoReferenciaContextoActorV1{}}
	bm, _ := man.RepresentacionCanonicaV1()
	hm, _ := vecdomain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(bm)
	res := vecdomain.ResultadoContextoActorRegistradoV2{RegistroContextoRef: "rca_" + z, Contexto: actor, RepresentacionCanonica: canon, HuellaSHA256: huella, ManifiestoProcedenciaCanonico: bm, ManifiestoProcedenciaHuellaSHA256: hm, AutoridadEfectiva: vecdomain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1, ResueltoEnAutoritativo: ahora}
	if err := res.Validar(); err != nil {
		t.Fatal(err)
	}
	auth := vecdomain.AutenticacionRevalidadaV1{AutenticacionRef: "aut_" + z, AutenticacionHuellaSHA256: strings.Repeat("a", 64), AsercionRef: "ase_" + z, SesionRef: "ses_" + z, ControlSesionRef: "cse_" + z, ControlSesionRevision: 1, ControlSesionHuellaSHA256: strings.Repeat("b", 64), CuentaRef: cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef, Superficie: superficie, MetodoObservado: vecdomain.AuthMethodCertificate, GarantiaObservada: vecdomain.AuthAssuranceHigh, PoliticaGarantiaRef: "pga_" + z, PoliticaGarantiaHuellaSHA256: strings.Repeat("c", 64), AutenticacionVerificadaEn: ahora.Add(-time.Minute), SesionEmitidaEn: ahora.Add(-time.Minute), SesionRevalidadaEn: ahora.Add(-time.Second), SesionValidaHasta: ahora.Add(time.Minute)}
	if err := auth.Validar(); err != nil {
		t.Fatalf("auth: %v", err)
	}
	v, err := vecdomain.CrearVinculoAutenticacionActorV2(context.Background(), revalidadorPrefPrueba{auth}, vecdomain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: auth.AutenticacionRef, SesionRef: auth.SesionRef}, resolutorPrefPrueba{res}, vecdomain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: actor.PerfilActivoRef}, relojPrefPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	return actor, v
}
func ordenPrueba(t *testing.T, p *proveedorPrueba) (ports.OrdenPreferencias, vecdomain.ContextoActor) {
	t.Helper()
	actor, v := identidadPrueba(t, vecdomain.SuperficieAutenticacionInternaCorporativaV1, "r", "p")
	orden, err := ports.NuevaOrdenPreferencias(actor, v, vecdomain.SuperficieAutenticacionInternaCorporativaV1, p)
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

func TestSuperficiePerfilYPersonaNoSeSustituyen(t *testing.T) {
	p := &proveedorPrueba{}
	actorInterno, vinculoInterno := identidadPrueba(t, vecdomain.SuperficieAutenticacionInternaCorporativaV1, "r", "p")
	actorExterno, vinculoExterno := identidadPrueba(t, vecdomain.SuperficieAutenticacionExternaPersonalV1, "r", "q")
	actorAjeno, _ := identidadPrueba(t, vecdomain.SuperficieAutenticacionInternaCorporativaV1, "s", "p")
	casos := []struct {
		nombre     string
		actor      vecdomain.ContextoActor
		vinculo    vecdomain.VinculoAutenticacionActorV2
		superficie vecdomain.SuperficieAutenticacionActorV1
		esperado   error
	}{
		{"ruta externa con vinculo interno", actorInterno, vinculoInterno, vecdomain.SuperficieAutenticacionExternaPersonalV1, ports.ErrProhibido},
		{"perfil sustituido", actorExterno, vinculoInterno, vecdomain.SuperficieAutenticacionInternaCorporativaV1, ports.ErrNoAutenticado},
		{"persona sustituida", actorAjeno, vinculoInterno, vecdomain.SuperficieAutenticacionInternaCorporativaV1, ports.ErrNoAutenticado},
		{"administracion ajena", actorInterno, vinculoInterno, vecdomain.SuperficieAutenticacionAdministracionPrivilegiadaV1, ports.ErrProhibido},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			if _, err := ports.NuevaOrdenPreferencias(tc.actor, tc.vinculo, tc.superficie, p); !errors.Is(err, tc.esperado) {
				t.Fatalf("sustitucion aceptada: %v", err)
			}
		})
	}
	orden, err := ports.NuevaOrdenPreferencias(actorExterno, vinculoExterno, vecdomain.SuperficieAutenticacionExternaPersonalV1, p)
	if err != nil {
		t.Fatalf("actor externo sin vinculo candidato rechazado: %v", err)
	}
	superficie, err := orden.Superficie()
	if err != nil || superficie != vecdomain.SuperficieAutenticacionExternaPersonalV1 {
		t.Fatal("superficie canonica perdida")
	}
	for _, accion := range []string{ports.AccionConsultarPreferencias, ports.AccionActualizarPreferencias} {
		audiencia, err := ports.AudienciaPreferencias(accion, superficie)
		if err != nil || !strings.Contains(audiencia, "externa_personal") {
			t.Fatalf("audiencia incorrecta: %q %v", audiencia, err)
		}
	}
}

func TestReplayEntreSuperficiesConservaHuellaYRecibo(t *testing.T) {
	proveedorInterno := &proveedorPrueba{}
	ordenInterno, actorInterno := ordenPrueba(t, proveedorInterno)
	c := domain.CatalogoBasePreferencias()
	fecha := time.Now().UTC()
	original := ports.ReciboPreferencias{ReciboRef: "recibo:original", PersonaRef: actorInterno.PersonaRef, Version: 1, CatalogoVersionRef: c.VersionRef, Valores: c.Predeterminados, FechaUTC: fecha}
	r := &registroPrueba{catalogo: c, recibo: original}
	s, _ := NuevoServicioPreferencias(r, time.Now)
	pet := ports.PeticionGuardarPreferencias{CatalogoVersionRef: c.VersionRef, ClaveOperacion: "operacion-1234567890", Valores: c.Predeterminados}
	if _, err := s.Guardar(context.Background(), ordenInterno, pet); err != nil {
		t.Fatal(err)
	}
	huellaInterna := r.material.HuellaPeticion
	proveedorExterno := &proveedorPrueba{}
	actorExterno, vinculoExterno := identidadPrueba(t, vecdomain.SuperficieAutenticacionExternaPersonalV1, "r", "q")
	ordenExterno, err := ports.NuevaOrdenPreferencias(actorExterno, vinculoExterno, vecdomain.SuperficieAutenticacionExternaPersonalV1, proveedorExterno)
	if err != nil {
		t.Fatal(err)
	}
	r.replay = true
	recibo, err := s.Guardar(context.Background(), ordenExterno, pet)
	if err != nil {
		t.Fatal(err)
	}
	if !recibo.Replay || recibo.ReciboRef != original.ReciboRef || !recibo.FechaUTC.Equal(fecha) || r.guardados != 1 || r.material.HuellaPeticion != huellaInterna || r.material.Superficie != vecdomain.SuperficieAutenticacionExternaPersonalV1 || r.material.PerfilRef != actorExterno.PerfilActivoRef || len(proveedorExterno.materiales) != 1 {
		t.Fatalf("replay cruzado no conservo identidad semantica y recibo: %#v %#v", recibo, r.material)
	}
	audiencia, _ := ports.AudienciaPreferencias(ports.AccionActualizarPreferencias, vecdomain.SuperficieAutenticacionExternaPersonalV1)
	if audiencia != proveedorExterno.primera.ResumenCapacidad().AudienciaConsumo() {
		t.Fatal("replay usó audiencia interna")
	}
}

func TestAudienciaV3DeOtraSuperficieNoLlegaALectura(t *testing.T) {
	p := &proveedorPrueba{audienciaForzada: ports.AudienciaConsultarPreferenciasExterna}
	orden, _ := ordenPrueba(t, p)
	r := &registroPrueba{catalogo: domain.CatalogoBasePreferencias()}
	s, _ := NuevoServicioPreferencias(r, time.Now)
	if _, err := s.Consultar(context.Background(), orden); !errors.Is(err, ports.ErrNoDisponible) || r.consultas != 0 {
		t.Fatalf("audiencia sustituida alcanzó SQL: %v", err)
	}
}
