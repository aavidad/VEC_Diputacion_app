package confianzaatestacion

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var _ atestadorAutorizacionLigadaV3 = (*application.ServicioAtestacionesAutorizacionV3)(nil)

type registroEtapasEmisionMaterialV3Prueba struct {
	etapas []string
}

func (r *registroEtapasEmisionMaterialV3Prueba) anotar(etapa string) {
	if r != nil {
		r.etapas = append(r.etapas, etapa)
	}
}

type autorizadorEmisionMaterialV3Prueba struct {
	decision     domain.DecisionAutorizacionLigadaV3
	confirmacion ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	err          error
	cancelar     context.CancelFunc
	registro     *registroEtapasEmisionMaterialV3Prueba
	invocaciones int
	solicitud    domain.SolicitudAutorizacionLigadaV3
	resultado    domain.ResultadoContextoActorRegistradoV2
}

func (a *autorizadorEmisionMaterialV3Prueba) ExigirSolicitudLigadaV3(
	_ context.Context,
	solicitud domain.SolicitudAutorizacionLigadaV3,
	resultado domain.ResultadoContextoActorRegistradoV2,
) (
	domain.DecisionAutorizacionLigadaV3,
	ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	error,
) {
	a.invocaciones++
	a.registro.anotar("pdp_durable")
	a.solicitud = solicitud
	a.resultado = resultado
	if a.cancelar != nil {
		a.cancelar()
	}
	return a.decision, a.confirmacion, a.err
}

type atestadorEmisionMaterialV3Prueba struct {
	atestacion   ports.AtestacionAutorizacionV3
	err          error
	cancelar     context.CancelFunc
	registro     *registroEtapasEmisionMaterialV3Prueba
	invocaciones int
	decision     domain.DecisionAutorizacionLigadaV3
	motivo       domain.ReferenciaEntradaCatalogo
	resultado    domain.ResultadoContextoActorRegistradoV2
}

func (a *atestadorEmisionMaterialV3Prueba) Atestar(
	_ context.Context,
	decision domain.DecisionAutorizacionLigadaV3,
	motivo domain.ReferenciaEntradaCatalogo,
	resultado domain.ResultadoContextoActorRegistradoV2,
) (ports.AtestacionAutorizacionV3, error) {
	a.invocaciones++
	a.registro.anotar("atestacion")
	a.decision = decision
	a.motivo = motivo
	a.resultado = resultado
	if a.cancelar != nil {
		a.cancelar()
	}
	return a.atestacion, a.err
}

type relojEtapaEmisionMaterialV3Prueba struct {
	ahora    time.Time
	etapa    string
	registro *registroEtapasEmisionMaterialV3Prueba
	cancelar context.CancelFunc
}

func (r *relojEtapaEmisionMaterialV3Prueba) Ahora() time.Time {
	r.registro.anotar(r.etapa)
	if r.cancelar != nil {
		r.cancelar()
	}
	return r.ahora
}

type escenarioEmisionMaterialV3Prueba struct {
	base              escenarioConfianzaAtestacionV3Prueba
	confirmacion      ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	autorizador       *autorizadorEmisionMaterialV3Prueba
	atestador         *atestadorEmisionMaterialV3Prueba
	relojConfianza    *relojEtapaEmisionMaterialV3Prueba
	relojCapacidad    *relojEtapaEmisionMaterialV3Prueba
	confianza         *ServicioConfianzaAtestacionAutorizacionV3
	emisorCapacidades *EmisorCapacidadesAtestacionAutorizacionV3
	emisorMaterial    *EmisorMaterialAutorizacionAtestadaV3
	registro          *registroEtapasEmisionMaterialV3Prueba
}

func nuevoEscenarioEmisionMaterialV3Prueba(
	t *testing.T,
) escenarioEmisionMaterialV3Prueba {
	t.Helper()
	base := nuevoEscenarioConfianzaAtestacionV3Prueba(t)
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		base.solicitud,
		base.decision,
		base.motivo,
		base.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion, err :=
		ports.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
			context.Background(),
			registroConcesionCapacidadV3Prueba{registradaEn: base.ahora},
			orden,
		)
	if err != nil {
		t.Fatal(err)
	}
	registro := &registroEtapasEmisionMaterialV3Prueba{}
	autorizador := &autorizadorEmisionMaterialV3Prueba{
		decision: base.decision, confirmacion: confirmacion, registro: registro,
	}
	atestador := &atestadorEmisionMaterialV3Prueba{
		atestacion: base.atestacion, registro: registro,
	}
	relojConfianza := &relojEtapaEmisionMaterialV3Prueba{
		ahora: base.ahora, etapa: "confianza", registro: registro,
	}
	confianza, err := NuevoServicioConfianzaAtestacionAutorizacionV3(
		base.configuracion,
		relojConfianza,
	)
	if err != nil {
		t.Fatal(err)
	}
	entropia := make([]byte, 0, 8*32)
	for indice := 0; indice < 8; indice++ {
		entropia = append(entropia, bytes.Repeat([]byte{byte(0x80 + indice)}, 32)...)
	}
	clave := claveCapacidadAtestacionV3Prueba(
		t,
		EstadoClaveHMACCapacidadAtestacionV3Emision,
		time.Time{},
		bytes.Repeat([]byte{0x63}, 32),
	)
	relojCapacidad := &relojEtapaEmisionMaterialV3Prueba{
		ahora:    base.ahora.Add(time.Microsecond),
		etapa:    "capacidad",
		registro: registro,
	}
	emisorCapacidades, err := nuevoEmisorCapacidadesAtestacionAutorizacionV3(
		clave,
		relojCapacidad,
		bytes.NewReader(entropia),
	)
	if err != nil {
		t.Fatal(err)
	}
	emisorMaterial, err := NuevoEmisorMaterialAutorizacionAtestadaV3(
		autorizador,
		atestador,
		confianza,
		emisorCapacidades,
	)
	if err != nil {
		t.Fatal(err)
	}
	return escenarioEmisionMaterialV3Prueba{
		base: base, confirmacion: confirmacion,
		autorizador: autorizador, atestador: atestador,
		relojConfianza: relojConfianza, relojCapacidad: relojCapacidad,
		confianza: confianza, emisorCapacidades: emisorCapacidades,
		emisorMaterial: emisorMaterial, registro: registro,
	}
}

func decisionDenegadaEmisionMaterialV3Prueba(
	t *testing.T,
	e escenarioEmisionMaterialV3Prueba,
) domain.DecisionAutorizacionLigadaV3 {
	t.Helper()
	datos, err := e.base.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatal(err)
	}
	version := domain.VersionRol{
		RolID: "tecnico_rrhh", Version: 1, Nombre: "Tecnico RRHH",
		Estado: domain.EstadoVersionRolPublicada,
		Concesiones: []domain.ConcesionRol{{
			Accion: "otra.accion", ModuloID: datos.Recurso.ModuloID,
			TipoRecurso:    datos.Recurso.Tipo,
			Finalidades:    []string{datos.Finalidad},
			GarantiaMinima: domain.AuthAssuranceSubstantial,
		}},
		PublicadaPor: "responsable-seguridad",
		PublicadaEn:  e.base.ahora.Add(-24 * time.Hour),
	}
	huellaCatalogo, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := domain.InstantaneaAutorizacion{
		AsignacionPerfil: domain.AsignacionPerfil{
			AsignacionID: "asig-rrhh", Version: 1,
			PerfilActivoRef: vinculo.PerfilActivoRef,
			PrincipalID:     vinculo.PrincipalID,
			VersionRolRef:   version.Referencia(),
			Estado:          domain.EstadoAsignacionPerfilActiva,
			Ambitos: []domain.AmbitoPerfil{
				{Clave: "organizacion_ref", Valores: []string{"organizacion:dipgra"}},
				{Clave: "centro_ref", Valores: []string{"centro:servicios-generales"}},
				{Clave: "categoria_ref", Valores: []string{"categoria:auxiliar-administrativo"}},
			},
			VigenteDesde: e.base.ahora.Add(-time.Hour),
			VigenteHasta: e.base.ahora.Add(time.Hour),
			EmitidaPor:   "administrador-identidades",
			EmitidaEn:    e.base.ahora.Add(-2 * time.Hour),
		},
		VersionRol: version,
		ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         domain.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor,
			ActualizadoEn:  version.PublicadaEn,
		},
		RevisionCatalogoPoliticas:     1,
		CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	evidencia, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(
		e.base.solicitud, instantanea,
		"dec_denegada_0123456789abcdef0123456789",
		e.base.ahora, e.base.ahora.Add(90*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := domain.NuevaDecisionAutorizacionLigadaV3(e.base.solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	concedida, codigo, err := decision.Resultado()
	if err != nil || concedida || codigo == "concedida" {
		t.Fatalf("fixture de denegacion invalida: %t %q %v", concedida, codigo, err)
	}
	return decision
}

func TestEmisorMaterialAutorizacionAtestadaV3DenegacionExplicitaNoAtesta(t *testing.T) {
	e := nuevoEscenarioEmisionMaterialV3Prueba(t)
	e.autorizador.decision = decisionDenegadaEmisionMaterialV3Prueba(t, e)
	e.autorizador.confirmacion = ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
	e.autorizador.err = errors.Join(
		domain.ErrAutorizacionDenegada,
		ports.ErrDenegacionExplicitaAutorizacionLigadaV3,
	)
	decision, confirmacion, material, err := e.emisorMaterial.EmitirMaterialAutorizacionAtestadaV3(
		context.Background(), e.base.solicitud, e.base.resultado,
	)
	if decision.Validar() == nil || confirmacion.Validar() == nil || material != nil ||
		!errors.Is(err, ports.ErrDenegacionExplicitaAutorizacionLigadaV3) ||
		!errors.Is(err, errEmisionMaterialAutorizacionAtestadaV3NoDisponible) ||
		e.atestador.invocaciones != 0 ||
		!reflect.DeepEqual(e.registro.etapas, []string{"pdp_durable"}) {
		t.Fatalf("denegacion explicita alcanzo emision: %v", err)
	}
	for _, salida := range []string{err.Error(), fmt.Sprintf("%+v", err)} {
		if strings.Contains(salida, "per_") || strings.Contains(salida, "prf_") ||
			strings.Contains(salida, "dec_denegada") {
			t.Fatalf("error de denegacion filtro datos: %q", salida)
		}
	}
	var log bytes.Buffer
	slog.New(slog.NewTextHandler(&log, nil)).Error("denegacion", "error", err)
	if strings.Contains(log.String(), "per_") || strings.Contains(log.String(), "prf_") ||
		strings.Contains(log.String(), "dec_denegada") {
		t.Fatalf("log de denegacion filtro datos: %q", log.String())
	}
}

func TestEmisorMaterialAutorizacionAtestadaV3NoAceptaCentinelaSinPrueba(t *testing.T) {
	casos := []struct {
		nombre   string
		preparar func(*escenarioEmisionMaterialV3Prueba) context.Context
	}{
		{"decision positiva", func(e *escenarioEmisionMaterialV3Prueba) context.Context {
			e.autorizador.err = errors.Join(domain.ErrAutorizacionDenegada,
				ports.ErrDenegacionExplicitaAutorizacionLigadaV3)
			return context.Background()
		}},
		{"decision nula", func(e *escenarioEmisionMaterialV3Prueba) context.Context {
			e.autorizador.decision = domain.DecisionAutorizacionLigadaV3{}
			e.autorizador.err = errors.Join(domain.ErrAutorizacionDenegada,
				ports.ErrDenegacionExplicitaAutorizacionLigadaV3)
			return context.Background()
		}},
		{"registro fallido", func(e *escenarioEmisionMaterialV3Prueba) context.Context {
			e.autorizador.decision = decisionDenegadaEmisionMaterialV3Prueba(t, *e)
			e.autorizador.err = errors.Join(domain.ErrAutorizacionDenegada,
				ports.ErrDenegacionExplicitaAutorizacionLigadaV3,
				ports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible)
			return context.Background()
		}},
		{"cancelacion", func(e *escenarioEmisionMaterialV3Prueba) context.Context {
			e.autorizador.decision = decisionDenegadaEmisionMaterialV3Prueba(t, *e)
			e.autorizador.err = errors.Join(domain.ErrAutorizacionDenegada,
				ports.ErrDenegacionExplicitaAutorizacionLigadaV3)
			ctx, cancelar := context.WithCancel(context.Background())
			e.autorizador.cancelar = cancelar
			return ctx
		}},
		{"plazo", func(e *escenarioEmisionMaterialV3Prueba) context.Context {
			e.autorizador.decision = decisionDenegadaEmisionMaterialV3Prueba(t, *e)
			e.autorizador.err = errors.Join(domain.ErrAutorizacionDenegada,
				ports.ErrDenegacionExplicitaAutorizacionLigadaV3,
				context.DeadlineExceeded)
			return context.Background()
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEscenarioEmisionMaterialV3Prueba(t)
			ctx := caso.preparar(&e)
			decision, confirmacion, material, err := e.emisorMaterial.EmitirMaterialAutorizacionAtestadaV3(
				ctx, e.base.solicitud, e.base.resultado,
			)
			if decision.Validar() == nil || confirmacion.Validar() == nil || material != nil ||
				errors.Is(err, ports.ErrDenegacionExplicitaAutorizacionLigadaV3) ||
				e.atestador.invocaciones != 0 {
				t.Fatalf("centinela no acreditado aceptado: %v", err)
			}
		})
	}
}

func exigirDurablesEmisionMaterialV3Prueba(
	t *testing.T,
	escenario escenarioEmisionMaterialV3Prueba,
	decision domain.DecisionAutorizacionLigadaV3,
	confirmacion ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
) {
	t.Helper()
	orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		escenario.base.solicitud,
		decision,
		escenario.base.motivo,
		escenario.base.resultado,
	)
	if err != nil || decision.ValidarPara(escenario.base.solicitud) != nil ||
		confirmacion.ValidarPara(orden) != nil {
		t.Fatalf("se perdieron los durables reales: %v", err)
	}
}

func TestEmisorMaterialAutorizacionAtestadaV3EncadenaPerfilExacto(t *testing.T) {
	escenario := nuevoEscenarioEmisionMaterialV3Prueba(t)
	decision, confirmacion, exportador, err :=
		escenario.emisorMaterial.EmitirMaterialAutorizacionAtestadaV3(
			context.Background(),
			escenario.base.solicitud,
			escenario.base.resultado,
		)
	if err != nil || exportador == nil {
		t.Fatalf("emitir material exacto: %v", err)
	}
	exigirDurablesEmisionMaterialV3Prueba(
		t, escenario, decision, confirmacion,
	)
	exportacion, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || exportacion.ValidarEstructura() != nil {
		t.Fatalf("exportar material exacto: %v", err)
	}
	if !reflect.DeepEqual(
		escenario.registro.etapas,
		[]string{"pdp_durable", "atestacion", "confianza", "capacidad"},
	) {
		t.Fatalf("orden de fronteras inesperado: %v", escenario.registro.etapas)
	}
	if escenario.atestador.motivo != escenario.base.motivo ||
		escenario.autorizador.resultado.Validar() != nil ||
		escenario.atestador.resultado.Validar() != nil ||
		!bytes.Equal(
			escenario.autorizador.resultado.RepresentacionCanonica,
			escenario.base.resultado.RepresentacionCanonica,
		) ||
		!bytes.Equal(
			escenario.atestador.resultado.RepresentacionCanonica,
			escenario.base.resultado.RepresentacionCanonica,
		) {
		t.Fatal("la cadena no reutilizó motivo y resultado exactos")
	}
}

func TestEmisorMaterialAutorizacionAtestadaV3ReemiteSinMezclarMaterial(
	t *testing.T,
) {
	escenario := nuevoEscenarioEmisionMaterialV3Prueba(t)
	var huellas []string
	for intento := 0; intento < 2; intento++ {
		decision, confirmacion, exportador, err :=
			escenario.emisorMaterial.EmitirMaterialAutorizacionAtestadaV3(
				context.Background(),
				escenario.base.solicitud,
				escenario.base.resultado,
			)
		if err != nil || exportador == nil {
			t.Fatalf("reemisión %d: %v", intento, err)
		}
		exigirDurablesEmisionMaterialV3Prueba(
			t, escenario, decision, confirmacion,
		)
		exportacion, err := exportador.ExportarMaterialParaConsumidor()
		if err != nil {
			t.Fatal(err)
		}
		huella, err := exportacion.HuellaConjuntoSHA256()
		if err != nil {
			t.Fatal(err)
		}
		huellas = append(huellas, huella)
	}
	if huellas[0] == huellas[1] {
		t.Fatal("la reemisión reutilizó el mismo material breve")
	}
}

func TestEmisorMaterialAutorizacionAtestadaV3CierraFirmaYNulosTipados(
	t *testing.T,
) {
	escenario := nuevoEscenarioEmisionMaterialV3Prueba(t)
	tipo := reflect.TypeOf(NuevoEmisorMaterialAutorizacionAtestadaV3)
	if tipo.NumIn() != 4 || tipo.NumOut() != 2 {
		t.Fatalf("firma de constructor abierta: %s", tipo)
	}
	for indice := 0; indice < tipo.NumIn(); indice++ {
		if tipo.In(indice).Kind() == reflect.String {
			t.Fatalf("cadena libre en parámetro %d: %s", indice, tipo.In(indice))
		}
	}
	metodo, existe := reflect.TypeOf(
		(*EmisorMaterialAutorizacionAtestadaV3)(nil),
	).MethodByName("EmitirMaterialAutorizacionAtestadaV3")
	if !existe || metodo.Type.NumIn() != 4 || metodo.Type.NumOut() != 4 {
		t.Fatalf("firma de emisión inesperada: %v", metodo.Type)
	}

	var autorizadorNulo *autorizadorEmisionMaterialV3Prueba
	var atestadorNulo *atestadorEmisionMaterialV3Prueba
	casos := []struct {
		nombre      string
		autorizador ports.AutorizadorSolicitudLigadaV3
		atestador   atestadorAutorizacionLigadaV3
		confianza   *ServicioConfianzaAtestacionAutorizacionV3
		emisor      *EmisorCapacidadesAtestacionAutorizacionV3
	}{
		{"autorizador_nulo", nil, escenario.atestador, escenario.confianza, escenario.emisorCapacidades},
		{"autorizador_nulo_tipado", autorizadorNulo, escenario.atestador, escenario.confianza, escenario.emisorCapacidades},
		{"atestador_nulo", escenario.autorizador, nil, escenario.confianza, escenario.emisorCapacidades},
		{"atestador_nulo_tipado", escenario.autorizador, atestadorNulo, escenario.confianza, escenario.emisorCapacidades},
		{"confianza_nula", escenario.autorizador, escenario.atestador, nil, escenario.emisorCapacidades},
		{"emisor_nulo", escenario.autorizador, escenario.atestador, escenario.confianza, nil},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			emisor, err := NuevoEmisorMaterialAutorizacionAtestadaV3(
				caso.autorizador,
				caso.atestador,
				caso.confianza,
				caso.emisor,
			)
			if emisor != nil || !errors.Is(
				err,
				errEmisionMaterialAutorizacionAtestadaV3NoDisponible,
			) {
				t.Fatalf("dependencia nula aceptada: %v, %v", emisor, err)
			}
		})
	}
}

func TestEmisorMaterialAutorizacionAtestadaV3FallaAntesDelPDPConCeros(
	t *testing.T,
) {
	casos := []struct {
		nombre    string
		preparar  func(*escenarioEmisionMaterialV3Prueba) context.Context
		solicitud func(escenarioEmisionMaterialV3Prueba) domain.SolicitudAutorizacionLigadaV3
		resultado func(escenarioEmisionMaterialV3Prueba) domain.ResultadoContextoActorRegistradoV2
	}{
		{
			nombre: "contexto_nulo",
			preparar: func(*escenarioEmisionMaterialV3Prueba) context.Context {
				return nil
			},
		},
		{
			nombre: "contexto_cancelado",
			preparar: func(*escenarioEmisionMaterialV3Prueba) context.Context {
				ctx, cancelar := context.WithCancel(context.Background())
				cancelar()
				return ctx
			},
		},
		{
			nombre: "solicitud_cero",
			preparar: func(*escenarioEmisionMaterialV3Prueba) context.Context {
				return context.Background()
			},
			resultado: func(e escenarioEmisionMaterialV3Prueba) domain.ResultadoContextoActorRegistradoV2 {
				return e.base.resultado
			},
		},
		{
			nombre: "resultado_cero",
			preparar: func(*escenarioEmisionMaterialV3Prueba) context.Context {
				return context.Background()
			},
			solicitud: func(e escenarioEmisionMaterialV3Prueba) domain.SolicitudAutorizacionLigadaV3 {
				return e.base.solicitud
			},
		},
		{
			nombre: "resultado_ajeno",
			preparar: func(*escenarioEmisionMaterialV3Prueba) context.Context {
				return context.Background()
			},
			solicitud: func(e escenarioEmisionMaterialV3Prueba) domain.SolicitudAutorizacionLigadaV3 {
				return e.base.solicitud
			},
			resultado: func(e escenarioEmisionMaterialV3Prueba) domain.ResultadoContextoActorRegistradoV2 {
				ajeno, err := e.base.resultado.Clonar()
				if err != nil {
					panic(err)
				}
				ajeno.RegistroContextoRef =
					"rca_22222222222222222222222222222222"
				return ajeno
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioEmisionMaterialV3Prueba(t)
			var solicitud domain.SolicitudAutorizacionLigadaV3
			var resultado domain.ResultadoContextoActorRegistradoV2
			if caso.solicitud != nil {
				solicitud = caso.solicitud(escenario)
			}
			if caso.resultado != nil {
				resultado = caso.resultado(escenario)
			}
			decision, confirmacion, material, err :=
				escenario.emisorMaterial.EmitirMaterialAutorizacionAtestadaV3(
					caso.preparar(&escenario),
					solicitud,
					resultado,
				)
			if decision.Validar() == nil || confirmacion.Validar() == nil ||
				material != nil ||
				!errors.Is(
					err,
					errEmisionMaterialAutorizacionAtestadaV3NoDisponible,
				) ||
				escenario.autorizador.invocaciones != 0 {
				t.Fatalf("fallo previo alcanzó PDP: %v", err)
			}
		})
	}
}

func TestEmisorMaterialAutorizacionAtestadaV3ConservaDurablesTrasPDP(
	t *testing.T,
) {
	t.Run("error_pdp_sin_concesion_acreditada", func(t *testing.T) {
		escenario := nuevoEscenarioEmisionMaterialV3Prueba(t)
		escenario.autorizador.decision = domain.DecisionAutorizacionLigadaV3{}
		escenario.autorizador.confirmacion =
			ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
		escenario.autorizador.err = errors.New("adaptador fallido")
		decision, confirmacion, material, err :=
			escenario.emisorMaterial.EmitirMaterialAutorizacionAtestadaV3(
				context.Background(),
				escenario.base.solicitud,
				escenario.base.resultado,
			)
		if decision.Validar() == nil || confirmacion.Validar() == nil ||
			material != nil || err == nil ||
			escenario.atestador.invocaciones != 0 {
			t.Fatalf("salida no acreditada conservada: %v", err)
		}
	})

	t.Run("error_pdp_opaco_con_deadline", func(t *testing.T) {
		escenario := nuevoEscenarioEmisionMaterialV3Prueba(t)
		escenario.autorizador.err = fmt.Errorf(
			"dsn=secreto persona=privada: %w",
			context.DeadlineExceeded,
		)
		decision, confirmacion, material, err :=
			escenario.emisorMaterial.EmitirMaterialAutorizacionAtestadaV3(
				context.Background(),
				escenario.base.solicitud,
				escenario.base.resultado,
			)
		exigirDurablesEmisionMaterialV3Prueba(
			t, escenario, decision, confirmacion,
		)
		if material != nil || !errors.Is(err, context.DeadlineExceeded) ||
			strings.Contains(err.Error(), "dsn=") ||
			strings.Contains(err.Error(), "persona=") ||
			escenario.atestador.invocaciones != 0 {
			t.Fatalf("error PDP no saneado: %v", err)
		}
	})

	t.Run("cancelacion_despues_pdp", func(t *testing.T) {
		escenario := nuevoEscenarioEmisionMaterialV3Prueba(t)
		ctx, cancelar := context.WithCancel(context.Background())
		escenario.autorizador.cancelar = cancelar
		decision, confirmacion, material, err :=
			escenario.emisorMaterial.EmitirMaterialAutorizacionAtestadaV3(
				ctx,
				escenario.base.solicitud,
				escenario.base.resultado,
			)
		exigirDurablesEmisionMaterialV3Prueba(
			t, escenario, decision, confirmacion,
		)
		if material != nil || !errors.Is(err, context.Canceled) ||
			escenario.atestador.invocaciones != 0 {
			t.Fatalf("cancelación tras PDP continuó: %v", err)
		}
	})

	t.Run("error_atestador", func(t *testing.T) {
		escenario := nuevoEscenarioEmisionMaterialV3Prueba(t)
		escenario.atestador.err = errors.New("ruta=/privada/dni")
		decision, confirmacion, material, err :=
			escenario.emisorMaterial.EmitirMaterialAutorizacionAtestadaV3(
				context.Background(),
				escenario.base.solicitud,
				escenario.base.resultado,
			)
		exigirDurablesEmisionMaterialV3Prueba(
			t, escenario, decision, confirmacion,
		)
		if material != nil || err == nil ||
			strings.Contains(err.Error(), "/privada/") ||
			escenario.relojConfianza.registro.etapas[len(
				escenario.relojConfianza.registro.etapas,
			)-1] != "atestacion" {
			t.Fatalf("error de atestación no cerrado: %v", err)
		}
	})

	t.Run("cancelacion_despues_atestacion", func(t *testing.T) {
		escenario := nuevoEscenarioEmisionMaterialV3Prueba(t)
		ctx, cancelar := context.WithCancel(context.Background())
		escenario.atestador.cancelar = cancelar
		decision, confirmacion, material, err :=
			escenario.emisorMaterial.EmitirMaterialAutorizacionAtestadaV3(
				ctx,
				escenario.base.solicitud,
				escenario.base.resultado,
			)
		exigirDurablesEmisionMaterialV3Prueba(
			t, escenario, decision, confirmacion,
		)
		if material != nil || !errors.Is(err, context.Canceled) ||
			len(escenario.registro.etapas) != 2 {
			t.Fatalf("cancelación tras atestación continuó: %v", err)
		}
	})

	t.Run("atestacion_cruzada", func(t *testing.T) {
		escenario := nuevoEscenarioEmisionMaterialV3Prueba(t)
		escenario.atestador.atestacion = ports.AtestacionAutorizacionV3{}
		decision, confirmacion, material, err :=
			escenario.emisorMaterial.EmitirMaterialAutorizacionAtestadaV3(
				context.Background(),
				escenario.base.solicitud,
				escenario.base.resultado,
			)
		exigirDurablesEmisionMaterialV3Prueba(
			t, escenario, decision, confirmacion,
		)
		if material != nil || err == nil ||
			escenario.relojCapacidad.registro.etapas[len(
				escenario.relojCapacidad.registro.etapas,
			)-1] != "confianza" {
			t.Fatalf("atestación cruzada alcanzó capacidad: %v", err)
		}
	})

	t.Run("cancelacion_en_confianza", func(t *testing.T) {
		escenario := nuevoEscenarioEmisionMaterialV3Prueba(t)
		ctx, cancelar := context.WithCancel(context.Background())
		escenario.relojConfianza.cancelar = cancelar
		decision, confirmacion, material, err :=
			escenario.emisorMaterial.EmitirMaterialAutorizacionAtestadaV3(
				ctx,
				escenario.base.solicitud,
				escenario.base.resultado,
			)
		exigirDurablesEmisionMaterialV3Prueba(
			t, escenario, decision, confirmacion,
		)
		if material != nil || !errors.Is(err, context.Canceled) ||
			escenario.relojCapacidad.registro.etapas[len(
				escenario.relojCapacidad.registro.etapas,
			)-1] != "confianza" {
			t.Fatalf("cancelación en confianza alcanzó capacidad: %v", err)
		}
	})

	t.Run("cancelacion_en_capacidad", func(t *testing.T) {
		escenario := nuevoEscenarioEmisionMaterialV3Prueba(t)
		ctx, cancelar := context.WithCancel(context.Background())
		escenario.relojCapacidad.cancelar = cancelar
		decision, confirmacion, material, err :=
			escenario.emisorMaterial.EmitirMaterialAutorizacionAtestadaV3(
				ctx,
				escenario.base.solicitud,
				escenario.base.resultado,
			)
		exigirDurablesEmisionMaterialV3Prueba(
			t, escenario, decision, confirmacion,
		)
		if material != nil || !errors.Is(err, context.Canceled) ||
			!reflect.DeepEqual(
				escenario.registro.etapas,
				[]string{
					"pdp_durable", "atestacion", "confianza", "capacidad",
				},
			) {
			t.Fatalf("cancelación en capacidad no cerrada: %v", err)
		}
	})

	t.Run("raiz_nominal_corrupta", func(t *testing.T) {
		escenario := nuevoEscenarioEmisionMaterialV3Prueba(t)
		entrada := escenario.confianza.raices[escenario.base.raiz.claveID]
		entrada.raizNominal.clavePublica[0] ^= 0xff
		escenario.confianza.raices[escenario.base.raiz.claveID] = entrada
		decision, confirmacion, material, err :=
			escenario.emisorMaterial.EmitirMaterialAutorizacionAtestadaV3(
				context.Background(),
				escenario.base.solicitud,
				escenario.base.resultado,
			)
		exigirDurablesEmisionMaterialV3Prueba(
			t, escenario, decision, confirmacion,
		)
		if material != nil || err == nil {
			t.Fatalf("raíz corrupta produjo material: %v", err)
		}
	})
}

func TestEmisorMaterialAutorizacionAtestadaV3RechazaMezclaSolicitudDecision(
	t *testing.T,
) {
	escenario := nuevoEscenarioEmisionMaterialV3Prueba(t)
	datos, err := escenario.base.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datos.Correlacion, err = domain.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		generadorCorrelacionConfianzaAtestacionV3Prueba{
			valor: "correlacion_22222222222222222222222222222222",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	otraSolicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		t.Fatal(err)
	}
	decision, confirmacion, material, err :=
		escenario.emisorMaterial.EmitirMaterialAutorizacionAtestadaV3(
			context.Background(),
			otraSolicitud,
			escenario.base.resultado,
		)
	if material != nil || err == nil ||
		decision.Validar() == nil ||
		confirmacion.Validar() == nil ||
		escenario.atestador.invocaciones != 0 {
		t.Fatalf("mezcla solicitud/decisión aceptada: %v", err)
	}
}
