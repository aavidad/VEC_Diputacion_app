package application

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	accionFachadaPrueba      = "bolsa.convocatoria.publicar"
	moduloFachadaPrueba      = "bolsa"
	tipoRecursoFachadaPrueba = "convocatoria_bolsa"
	finalidadFachadaPrueba   = "gobernar_convocatoria"
)

type autorizadorFachadaUsoPrueba struct {
	ahora          time.Time
	campos         []string
	obligaciones   []string
	garantiaMinima domain.AuthAssurance
	mutar          func(*domain.DecisionAutorizacion)
	observar       func(domain.SolicitudAutorizacion)
	despues        func()
	llamadas       int
}

func (a *autorizadorFachadaUsoPrueba) Exigir(
	_ context.Context,
	solicitud domain.SolicitudAutorizacion,
) (domain.DecisionAutorizacion, error) {
	a.llamadas++
	decision := completarDecisionAutorizacionPrueba(solicitud, domain.DecisionAutorizacion{
		DecisionRef: fmt.Sprintf("decision:fachada:%03d", a.llamadas), Concedida: true,
		Codigo: "concedida", PrincipalID: solicitud.Principal.ID,
		PerfilActivoRef: solicitud.PerfilActivoRef, Accion: solicitud.Accion,
		RecursoRef: solicitud.Recurso.Referencia, Finalidad: solicitud.Finalidad,
		CorrelacionRef:            solicitud.CorrelacionRef,
		VinculoAutenticacionActor: solicitud.VinculoAutenticacionActor,
		AsignacionRef:             "asignacion:fachada:v1", AsignacionHuellaSHA256: strings.Repeat("a", 64),
		VersionRolRef: "rol:fachada:v1", VersionRolHuellaSHA256: strings.Repeat("b", 64),
		GarantiaMinima: a.garantiaMinima, CamposPermitidos: append([]string(nil), a.campos...),
		Obligaciones: append([]string(nil), a.obligaciones...),
		EmitidaEn:    a.ahora.Add(-time.Second), ValidaHasta: a.ahora.Add(time.Minute),
	})
	if a.observar != nil {
		a.observar(solicitud)
	}
	if a.mutar != nil {
		a.mutar(&decision)
	}
	if a.despues != nil {
		a.despues()
	}
	return decision, nil
}

type relojSecuencialFachadaPrueba struct {
	instantes []time.Time
	indice    int
}

func (r *relojSecuencialFachadaPrueba) Ahora() time.Time {
	if r == nil || r.indice >= len(r.instantes) {
		return time.Time{}
	}
	instante := r.instantes[r.indice]
	r.indice++
	return instante
}

func TestFachadaUsoAutorizacionDevuelveEvidenciaExactaEInmutable(t *testing.T) {
	ahora := instanteFachadaUsoAutorizacionPrueba()
	actor, vinculo := contextoYVinculoAutenticacionAplicacionPrueba(ahora)
	campos := []string{"expediente.estado", "expediente.observaciones"}
	politica := politicaFachadaUsoPrueba(t, campos, PerfilProteccionUsoAutorizacionInternoAlto)
	campos[0] = "campo.mutado"
	autorizador := &autorizadorFachadaUsoPrueba{
		ahora: ahora, campos: []string{"expediente.observaciones", "expediente.estado"},
		garantiaMinima: domain.AuthAssuranceHigh,
	}
	fachada := nuevaFachadaUsoAutorizacionPrueba(t, autorizador, relojUsoAutorizacionPrueba{ahora: ahora})
	evidencia, err := exigirEvidenciaFachadaPrueba(fachada, context.Background(), actor, vinculo, politica)
	if err != nil || evidencia.ValidarEn(ahora) != nil {
		t.Fatalf("exigir evidencia: evidencia=%v error=%v", evidencia, err)
	}
	datos, err := evidencia.Datos()
	if err != nil || autorizador.llamadas != 1 || len(datos.Decision.CamposPermitidos) != 2 {
		t.Fatalf("evidencia inesperada: llamadas=%d datos=%v error=%v", autorizador.llamadas, datos, err)
	}
	datos.Decision.CamposPermitidos[0] = "campo.mutado"
	deNuevo, err := evidencia.Datos()
	if err != nil || deNuevo.Decision.CamposPermitidos[0] == "campo.mutado" {
		t.Fatal("la proyeccion mutable altero la capacidad")
	}
}

func TestFachadaUsoAutorizacionDeniegaCamposYObligacionesNoConsumidas(t *testing.T) {
	ahora := instanteFachadaUsoAutorizacionPrueba()
	actor, vinculo := contextoYVinculoAutenticacionAplicacionPrueba(ahora)
	politica := politicaFachadaUsoPrueba(t,
		[]string{"expediente.estado", "expediente.observaciones"},
		PerfilProteccionUsoAutorizacionInternoAlto,
	)
	casos := []struct {
		nombre       string
		campos       []string
		obligaciones []string
	}{
		{"campo ausente", []string{"expediente.estado"}, nil},
		{"campo adicional", []string{"expediente.estado", "expediente.observaciones", "expediente.dni"}, nil},
		{"campo duplicado", []string{"expediente.estado", "expediente.estado"}, nil},
		{"obligacion", []string{"expediente.estado", "expediente.observaciones"}, []string{"doble_control"}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			autorizador := &autorizadorFachadaUsoPrueba{ahora: ahora, campos: caso.campos,
				obligaciones: caso.obligaciones, garantiaMinima: domain.AuthAssuranceHigh}
			fachada := nuevaFachadaUsoAutorizacionPrueba(t, autorizador, relojUsoAutorizacionPrueba{ahora: ahora})
			evidencia, err := exigirEvidenciaFachadaPrueba(fachada, context.Background(), actor, vinculo, politica)
			comprobarDenegacionFachadaPrueba(t, evidencia, err, ahora)
		})
	}
}

func TestFachadaUsoAutorizacionAplicaSuperficieYGarantiaDelPerfil(t *testing.T) {
	ahora := instanteFachadaUsoAutorizacionPrueba()
	casos := []struct {
		nombre      string
		perfil      PerfilProteccionUsoAutorizacion
		superficie  domain.SuperficieAutenticacionActorV1
		sesion, pdp domain.AuthAssurance
		permitida   bool
		consultaPDP bool
	}{
		{"interna alta", PerfilProteccionUsoAutorizacionInternoAlto, domain.SuperficieAutenticacionInternaCorporativaV1, domain.AuthAssuranceHigh, domain.AuthAssuranceHigh, true, true},
		{"privilegiada alta", PerfilProteccionUsoAutorizacionInternoAlto, domain.SuperficieAutenticacionAdministracionPrivilegiadaV1, domain.AuthAssuranceHigh, domain.AuthAssuranceHigh, true, true},
		{"externa en perfil alto", PerfilProteccionUsoAutorizacionInternoAlto, domain.SuperficieAutenticacionExternaPersonalV1, domain.AuthAssuranceHigh, domain.AuthAssuranceHigh, false, false},
		{"interna sustancial", PerfilProteccionUsoAutorizacionInternoAlto, domain.SuperficieAutenticacionInternaCorporativaV1, domain.AuthAssuranceSubstantial, domain.AuthAssuranceSubstantial, false, false},
		{"PDP rebaja garantia", PerfilProteccionUsoAutorizacionInternoAlto, domain.SuperficieAutenticacionInternaCorporativaV1, domain.AuthAssuranceHigh, domain.AuthAssuranceSubstantial, false, true},
		{"externa ordinaria", PerfilProteccionUsoAutorizacionOrdinario, domain.SuperficieAutenticacionExternaPersonalV1, domain.AuthAssuranceSubstantial, domain.AuthAssuranceSubstantial, true, true},
		{"interna en perfil ordinario", PerfilProteccionUsoAutorizacionOrdinario, domain.SuperficieAutenticacionInternaCorporativaV1, domain.AuthAssuranceHigh, domain.AuthAssuranceHigh, false, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			actor, vinculo := nuevasCredencialesFachadaUsoAutorizacionPrueba(t, ahora, caso.superficie, caso.sesion, 30*time.Minute)
			autorizador := &autorizadorFachadaUsoPrueba{ahora: ahora, garantiaMinima: caso.pdp}
			fachada := nuevaFachadaUsoAutorizacionPrueba(t, autorizador, relojUsoAutorizacionPrueba{ahora: ahora})
			politica := politicaFachadaUsoPrueba(t, nil, caso.perfil)
			evidencia, err := exigirEvidenciaFachadaPrueba(fachada, context.Background(), actor, vinculo, politica)
			if caso.permitida && (err != nil || evidencia.ValidarEn(ahora) != nil) {
				t.Fatalf("uso permitido denegado: %v", err)
			}
			if !caso.permitida {
				comprobarDenegacionFachadaPrueba(t, evidencia, err, ahora)
			}
			esperadas := 0
			if caso.consultaPDP {
				esperadas = 1
			}
			if autorizador.llamadas != esperadas {
				t.Fatalf("consultas PDP=%d, esperadas=%d", autorizador.llamadas, esperadas)
			}
		})
	}
}

func TestPoliticaUsoAutorizacionFallaCerradoYNoSeSerializa(t *testing.T) {
	valida := []string{accionFachadaPrueba, moduloFachadaPrueba, tipoRecursoFachadaPrueba, finalidadFachadaPrueba}
	casos := []struct {
		nombre          string
		identificadores []string
		campos          []string
		perfil          PerfilProteccionUsoAutorizacion
	}{
		{"accion vacia", []string{"", moduloFachadaPrueba, tipoRecursoFachadaPrueba, finalidadFachadaPrueba}, nil, PerfilProteccionUsoAutorizacionOrdinario},
		{"modulo comodin", []string{accionFachadaPrueba, "*", tipoRecursoFachadaPrueba, finalidadFachadaPrueba}, nil, PerfilProteccionUsoAutorizacionOrdinario},
		{"tipo no canonico", []string{accionFachadaPrueba, moduloFachadaPrueba, " tipo", finalidadFachadaPrueba}, nil, PerfilProteccionUsoAutorizacionOrdinario},
		{"finalidad no ascii", []string{accionFachadaPrueba, moduloFachadaPrueba, tipoRecursoFachadaPrueba, "revisión"}, nil, PerfilProteccionUsoAutorizacionOrdinario},
		{"perfil cero", valida, nil, PerfilProteccionUsoAutorizacionNoDeclarado},
		{"campo repetido", valida, []string{"estado", "estado"}, PerfilProteccionUsoAutorizacionOrdinario},
		{"campo comodin", valida, []string{"expediente.*"}, PerfilProteccionUsoAutorizacionOrdinario},
		{"campo no canonico", valida, []string{" estado"}, PerfilProteccionUsoAutorizacionOrdinario},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			p, err := NuevaPoliticaUsoDecisionAutorizacion(caso.identificadores[0], caso.identificadores[1],
				caso.identificadores[2], caso.identificadores[3], caso.campos, caso.perfil)
			if !errors.Is(err, ErrPoliticaUsoDecisionAutorizacionInvalida) || p.validar() == nil {
				t.Fatalf("politica invalida aceptada: %v, %v", p, err)
			}
		})
	}
	p := politicaFachadaUsoPrueba(t, []string{"expediente.estado"}, PerfilProteccionUsoAutorizacionOrdinario)
	serializaciones := []func() error{
		func() error { _, err := json.Marshal(p); return err },
		func() error { _, err := xml.Marshal(p); return err },
		func() error { var b bytes.Buffer; return gob.NewEncoder(&b).Encode(p) },
		func() error { _, err := p.MarshalText(); return err },
		func() error { _, err := p.MarshalBinary(); return err },
	}
	for indice, serializar := range serializaciones {
		if err := serializar(); !errors.Is(err, ErrSerializacionPoliticaUsoAutorizacionProhibida) {
			t.Fatalf("serializacion %d aceptada: %v", indice, err)
		}
	}
	var reconstruida PoliticaUsoDecisionAutorizacion
	if err := json.Unmarshal([]byte(`{}`), &reconstruida); !errors.Is(err, ErrSerializacionPoliticaUsoAutorizacionProhibida) {
		t.Fatalf("reconstruccion JSON aceptada: %v", err)
	}
	if strings.Contains(fmt.Sprintf("%+v", p), "expediente.estado") {
		t.Fatal("el formateo filtro la politica")
	}
}

func TestFachadaUsoAutorizacionLigaPoliticaRecursoYDecisionExactos(t *testing.T) {
	ahora := instanteFachadaUsoAutorizacionPrueba()
	actor, vinculo := contextoYVinculoAutenticacionAplicacionPrueba(ahora)
	politica := politicaFachadaUsoPrueba(t, nil, PerfilProteccionUsoAutorizacionInternoAlto)
	for _, mutarRecurso := range []func(*domain.RecursoAutorizable){
		func(r *domain.RecursoAutorizable) { r.ModuloID = "personal" },
		func(r *domain.RecursoAutorizable) { r.Tipo = "expediente" },
	} {
		autorizador := &autorizadorFachadaUsoPrueba{ahora: ahora, garantiaMinima: domain.AuthAssuranceHigh}
		fachada := nuevaFachadaUsoAutorizacionPrueba(t, autorizador, relojUsoAutorizacionPrueba{ahora: ahora})
		recurso := recursoFachadaUsoAutorizacionPrueba()
		mutarRecurso(&recurso)
		evidencia, err := fachada.ExigirEvidencia(context.Background(), actor, vinculo, recurso,
			"correlacion:fachada:001", "Publicacion", politica)
		comprobarDenegacionFachadaPrueba(t, evidencia, err, ahora)
		if autorizador.llamadas != 0 {
			t.Fatal("un recurso fuera de politica consulto al PDP")
		}
	}
	mutaciones := []struct {
		nombre string
		mutar  func(*domain.DecisionAutorizacion)
	}{
		{"accion", func(d *domain.DecisionAutorizacion) { d.Accion = "bolsa.convocatoria.retirar" }},
		{"referencia", func(d *domain.DecisionAutorizacion) { d.RecursoRef = "convocatoria:otra:2026:v1" }},
		{"modulo", func(d *domain.DecisionAutorizacion) { d.ModuloID = "personal" }},
		{"tipo", func(d *domain.DecisionAutorizacion) { d.TipoRecurso = "expediente" }},
		{"contexto", func(d *domain.DecisionAutorizacion) { d.ContextoRecursoHuellaSHA256 = strings.Repeat("f", 64) }},
		{"finalidad", func(d *domain.DecisionAutorizacion) { d.Finalidad = "consultar_convocatoria" }},
		{"correlacion", func(d *domain.DecisionAutorizacion) { d.CorrelacionRef = "correlacion:otra:001" }},
		{"vinculo", func(d *domain.DecisionAutorizacion) {
			d.VinculoAutenticacionActor = domain.VinculoAutenticacionActorV1{}
		}},
	}
	for _, caso := range mutaciones {
		t.Run(caso.nombre, func(t *testing.T) {
			autorizador := &autorizadorFachadaUsoPrueba{ahora: ahora, garantiaMinima: domain.AuthAssuranceHigh, mutar: caso.mutar}
			fachada := nuevaFachadaUsoAutorizacionPrueba(t, autorizador, relojUsoAutorizacionPrueba{ahora: ahora})
			evidencia, err := exigirEvidenciaFachadaPrueba(fachada, context.Background(), actor, vinculo, politica)
			comprobarDenegacionFachadaPrueba(t, evidencia, err, ahora)
		})
	}
}

func TestFachadaUsoAutorizacionRevalidaAlInstanteFinal(t *testing.T) {
	ahora := instanteFachadaUsoAutorizacionPrueba()
	casos := []struct {
		nombre        string
		vigenciaActor time.Duration
		instanteFinal time.Time
	}{
		{"actor vencido tras PDP", 30 * time.Second, ahora.Add(31 * time.Second)},
		{"decision vencida tras PDP", 30 * time.Minute, ahora.Add(2 * time.Minute)},
		{"reloj final nulo", 30 * time.Minute, time.Time{}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			actor, vinculo := nuevasCredencialesFachadaUsoAutorizacionPrueba(t, ahora,
				domain.SuperficieAutenticacionInternaCorporativaV1, domain.AuthAssuranceHigh, caso.vigenciaActor)
			reloj := &relojSecuencialFachadaPrueba{instantes: []time.Time{ahora, ahora, caso.instanteFinal}}
			autorizador := &autorizadorFachadaUsoPrueba{ahora: ahora, garantiaMinima: domain.AuthAssuranceHigh}
			fachada := nuevaFachadaUsoAutorizacionPrueba(t, autorizador, reloj)
			evidencia, err := exigirEvidenciaFachadaPrueba(fachada, context.Background(), actor, vinculo,
				politicaFachadaUsoPrueba(t, nil, PerfilProteccionUsoAutorizacionInternoAlto))
			comprobarDenegacionFachadaPrueba(t, evidencia, err, ahora)
			if reloj.indice != 3 {
				t.Fatalf("lecturas de reloj=%d, esperadas=3", reloj.indice)
			}
		})
	}
}

func TestFachadaUsoAutorizacionCopiaElContextoDelRecurso(t *testing.T) {
	ahora := instanteFachadaUsoAutorizacionPrueba()
	actor, vinculo := contextoYVinculoAutenticacionAplicacionPrueba(ahora)
	recurso := recursoFachadaUsoAutorizacionPrueba()
	autorizador := &autorizadorFachadaUsoPrueba{ahora: ahora, garantiaMinima: domain.AuthAssuranceHigh,
		observar: func(s domain.SolicitudAutorizacion) {
			s.Recurso.Ambitos["unidad"] = "mutada"
			s.Recurso.Atributos["revision"] = "999"
		}}
	fachada := nuevaFachadaUsoAutorizacionPrueba(t, autorizador, relojUsoAutorizacionPrueba{ahora: ahora})
	evidencia, err := fachada.ExigirEvidencia(context.Background(), actor, vinculo, recurso,
		"correlacion:fachada:001", "Publicacion",
		politicaFachadaUsoPrueba(t, nil, PerfilProteccionUsoAutorizacionInternoAlto))
	if err != nil || evidencia.ValidarEn(ahora) != nil {
		t.Fatalf("evidencia valida denegada: %v", err)
	}
	if recurso.Ambitos["unidad"] != "seleccion" || recurso.Atributos["revision"] != "7" {
		t.Fatalf("el autorizador altero el recurso del llamador: %#v", recurso)
	}
}

func TestFachadaUsoAutorizacionFallaCerradoEnPrecondicionesYCancelacion(t *testing.T) {
	ahora := instanteFachadaUsoAutorizacionPrueba()
	actor, vinculo := contextoYVinculoAutenticacionAplicacionPrueba(ahora)
	autorizador := &autorizadorFachadaUsoPrueba{ahora: ahora, garantiaMinima: domain.AuthAssuranceHigh}
	politica := politicaFachadaUsoPrueba(t, nil, PerfilProteccionUsoAutorizacionInternoAlto)
	var autorizadorNulo *autorizadorFachadaUsoPrueba
	var relojNulo *relojSecuencialFachadaPrueba
	for nombre, dependencias := range map[string]struct {
		autorizador ports.Autorizador
		reloj       ports.Reloj
	}{
		"autorizador nulo tipado": {autorizadorNulo, relojUsoAutorizacionPrueba{ahora: ahora}},
		"reloj nulo tipado":       {autorizador, relojNulo},
	} {
		t.Run(nombre, func(t *testing.T) {
			fachada, err := NuevaFachadaUsoDecisionAutorizacion(dependencias.autorizador, dependencias.reloj)
			if fachada != nil || !errors.Is(err, domain.ErrConfiguracionAccesoInvalida) {
				t.Fatalf("dependencia nula aceptada: fachada=%v error=%v", fachada, err)
			}
		})
	}
	for nombre, ctx := range map[string]context.Context{"nulo": nil, "cancelado": contextoCanceladoAutorizacionPrueba()} {
		t.Run(nombre, func(t *testing.T) {
			fachada := nuevaFachadaUsoAutorizacionPrueba(t, autorizador, relojUsoAutorizacionPrueba{ahora: ahora})
			evidencia, err := exigirEvidenciaFachadaPrueba(fachada, ctx, actor, vinculo, politica)
			comprobarDenegacionFachadaPrueba(t, evidencia, err, ahora)
		})
	}
	fachada := nuevaFachadaUsoAutorizacionPrueba(t, autorizador, relojUsoAutorizacionPrueba{ahora: ahora})
	evidencia, err := exigirEvidenciaFachadaPrueba(fachada, context.Background(), actor, vinculo, PoliticaUsoDecisionAutorizacion{})
	comprobarDenegacionFachadaPrueba(t, evidencia, err, ahora)
	if autorizador.llamadas != 0 {
		t.Fatalf("se consulto PDP con precondiciones invalidas: %d", autorizador.llamadas)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	autorizador.despues = cancelar
	fachada = nuevaFachadaUsoAutorizacionPrueba(t, autorizador, relojUsoAutorizacionPrueba{ahora: ahora})
	evidencia, err = exigirEvidenciaFachadaPrueba(fachada, ctx, actor, vinculo, politica)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion concurrente no conservada: %v", err)
	}
	comprobarDenegacionFachadaPrueba(t, evidencia, err, ahora)
}

func politicaFachadaUsoPrueba(t *testing.T, campos []string, perfil PerfilProteccionUsoAutorizacion) PoliticaUsoDecisionAutorizacion {
	t.Helper()
	p, err := NuevaPoliticaUsoDecisionAutorizacion(accionFachadaPrueba, moduloFachadaPrueba,
		tipoRecursoFachadaPrueba, finalidadFachadaPrueba, campos, perfil)
	if err != nil {
		t.Fatalf("crear politica: %v", err)
	}
	return p
}

func exigirEvidenciaFachadaPrueba(f *FachadaUsoDecisionAutorizacion, ctx context.Context,
	actor domain.ContextoActor, vinculo domain.VinculoAutenticacionActorV1,
	politica PoliticaUsoDecisionAutorizacion,
) (ports.EvidenciaUsoDecisionAutorizacion, error) {
	return f.ExigirEvidencia(ctx, actor, vinculo, recursoFachadaUsoAutorizacionPrueba(),
		"correlacion:fachada:001", "Publicacion", politica)
}

func comprobarDenegacionFachadaPrueba(t *testing.T, evidencia ports.EvidenciaUsoDecisionAutorizacion,
	err error, ahora time.Time,
) {
	t.Helper()
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || evidencia.ValidarEn(ahora) == nil {
		t.Fatalf("operacion prohibida aceptada: evidencia=%v error=%v", evidencia, err)
	}
}

func nuevaFachadaUsoAutorizacionPrueba(t *testing.T, autorizador ports.Autorizador,
	reloj ports.Reloj,
) *FachadaUsoDecisionAutorizacion {
	t.Helper()
	f, err := NuevaFachadaUsoDecisionAutorizacion(autorizador, reloj)
	if err != nil {
		t.Fatalf("crear fachada: %v", err)
	}
	return f
}

func recursoFachadaUsoAutorizacionPrueba() domain.RecursoAutorizable {
	return domain.RecursoAutorizable{Referencia: "convocatoria:bolsa:2026:v1",
		ModuloID: moduloFachadaPrueba, Tipo: tipoRecursoFachadaPrueba,
		Ambitos: map[string]string{"unidad": "seleccion"}, Atributos: map[string]string{"revision": "7"}}
}

func instanteFachadaUsoAutorizacionPrueba() time.Time {
	return time.Date(2026, time.July, 16, 8, 30, 0, 0, time.UTC)
}

func nuevasCredencialesFachadaUsoAutorizacionPrueba(t *testing.T, instante time.Time,
	superficie domain.SuperficieAutenticacionActorV1, garantia domain.AuthAssurance,
	vigenciaActor time.Duration,
) (domain.ContextoActor, domain.VinculoAutenticacionActorV1) {
	t.Helper()
	cuentaRef, ordinariaRef, privilegiada := "cta_0123456789abcdefghijkl", "cta_0123456789abcdefghijkl", false
	if superficie == domain.SuperficieAutenticacionAdministracionPrivilegiadaV1 {
		cuentaRef, ordinariaRef, privilegiada = "cta_"+strings.Repeat("p", 22), "cta_"+strings.Repeat("o", 22), true
	}
	cuenta := domain.CuentaAutenticadaContextoActor{CuentaRef: cuentaRef,
		Metodo: domain.AuthMethodCertificate, Garantia: garantia}
	instantanea := domain.InstantaneaContextoActor{VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 5,
		CuentaRef: cuentaRef, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 3,
		PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 4,
		Estado: domain.EstadoVinculoContextoActorActivo, VigenteDesde: instante.Add(-time.Hour),
		VigenteHasta: instante.Add(vigenciaActor)}
	actor, err := domain.NuevoContextoActor(cuenta, instantanea, instante.Add(-2*time.Minute))
	if err != nil {
		t.Fatalf("crear actor: %v", err)
	}
	autenticacion := autenticacionFachadaUsoAutorizacionPrueba(instante, cuenta, superficie)
	autenticacion.CuentaPrivilegiada, autenticacion.CuentaOrdinariaRef = privilegiada, ordinariaRef
	vinculo, err := domain.CrearVinculoAutenticacionActorV1(context.Background(),
		revalidadorVinculoAutenticacionAplicacionPrueba{resultado: autenticacion},
		domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef: autenticacion.SesionRef}, actor, instante)
	if err != nil {
		t.Fatalf("crear vinculo: %v", err)
	}
	return actor, vinculo
}

func autenticacionFachadaUsoAutorizacionPrueba(instante time.Time,
	cuenta domain.CuentaAutenticadaContextoActor,
	superficie domain.SuperficieAutenticacionActorV1,
) domain.AutenticacionRevalidadaV1 {
	return domain.AutenticacionRevalidadaV1{
		AutenticacionRef: "aut_0123456789abcdefghijkl", AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef: "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl",
		ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 7,
		ControlSesionHuellaSHA256: strings.Repeat("2", 64), CuentaRef: cuenta.CuentaRef,
		CuentaOrdinariaRef: cuenta.CuentaRef, Superficie: superficie, MetodoObservado: cuenta.Metodo,
		GarantiaObservada: cuenta.Garantia, PoliticaGarantiaRef: "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    instante.Add(-5 * time.Minute), SesionEmitidaEn: instante.Add(-4 * time.Minute),
		SesionRevalidadaEn: instante.Add(-3 * time.Minute), SesionValidaHasta: instante.Add(10 * time.Minute),
	}
}
