package ports

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

type revalidadorOrdenAutorizacionV3Prueba struct {
	resultado domain.AutenticacionRevalidadaV1
}

func (r revalidadorOrdenAutorizacionV3Prueba) RevalidarAutenticacionActorV1(
	context.Context,
	domain.SolicitudRevalidacionAutenticacionActorV1,
) (domain.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

type resolutorOrdenAutorizacionV3Prueba struct {
	resultado domain.ResultadoContextoActorRegistradoV2
}

func (r resolutorOrdenAutorizacionV3Prueba) ResolverContextoActorRegistradoV2(
	context.Context,
	domain.SolicitudContextoActor,
) (domain.ResultadoContextoActorRegistradoV2, error) {
	return r.resultado, nil
}

type relojOrdenAutorizacionV3Prueba struct{ ahora time.Time }

func (r relojOrdenAutorizacionV3Prueba) Ahora() time.Time { return r.ahora }

type registroConcesionLigadaV3Prueba struct {
	registradaEn time.Time
	err          error
	invocaciones int
	cancelar     context.CancelFunc
}

func (r *registroConcesionLigadaV3Prueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	_ context.Context,
	_ OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	r.invocaciones++
	if r.cancelar != nil {
		r.cancelar()
	}
	return r.registradaEn, r.err
}

type generadorCorrelacionOrdenAutorizacionV3Prueba struct{ valor string }

func (g generadorCorrelacionOrdenAutorizacionV3Prueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return g.valor, nil
}

type escenarioOrdenAutorizacionV3Prueba struct {
	ahora      time.Time
	solicitud  domain.SolicitudAutorizacionLigadaV3
	decision   domain.DecisionAutorizacionLigadaV3
	motivo     domain.ReferenciaEntradaCatalogo
	resultado  domain.ResultadoContextoActorRegistradoV2
	instantnea domain.InstantaneaAutorizacion
}

func nuevoEscenarioOrdenAutorizacionV3Prueba(t *testing.T) escenarioOrdenAutorizacionV3Prueba {
	t.Helper()
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl", Metodo: domain.AuthMethodCertificate,
		Garantia: domain.AuthAssuranceHigh,
	}
	instantaneaActor := domain.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 3,
		CuentaRef: cuenta.CuentaRef, CuentaVersion: 4,
		PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 2,
		PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 5,
		Estado:       domain.EstadoVinculoContextoActorActivo,
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
	}
	actor, err := domain.NuevoContextoActor(cuenta, instantaneaActor, ahora.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	representacion, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := domain.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef: "prc_0123456789abcdefghijkl", ProcedenciaVersion: 1,
		ProcedenciaHuellaSHA256: strings.Repeat("4", 64),
		ProcedenciaAutoridad:    domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := domain.ManifiestoProcedenciaContextoActorV1{
		Esquema:           domain.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: domain.ProcedenciaCuentaContextoActorV1{
			CuentaRef: cuenta.CuentaRef, Version: instantaneaActor.CuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: domain.ProcedenciaPersonaContextoActorV1{
			PersonaRef: instantaneaActor.PersonaRef, Version: instantaneaActor.PersonaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: domain.ProcedenciaPerfilContextoActorV1{
			PerfilRef: instantaneaActor.PerfilActivoRef, Version: instantaneaActor.PerfilVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: domain.ProcedenciaVinculoContextoActorV1{
			VinculoRef: instantaneaActor.VinculoRef, Version: instantaneaActor.VinculoVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: make([]domain.ProcedenciaVinculoReferenciaContextoActorV1, 0),
	}
	canonManifiesto, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	huellaManifiesto, err := domain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(canonManifiesto)
	if err != nil {
		t.Fatal(err)
	}
	resultado := domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn", Contexto: actor,
		RepresentacionCanonica: representacion, HuellaSHA256: huella,
		ManifiestoProcedenciaCanonico:     canonManifiesto,
		ManifiestoProcedenciaHuellaSHA256: huellaManifiesto,
		AutoridadEfectiva:                 domain.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo:            actor.ResueltoEn,
	}
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef:          "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256: strings.Repeat("1", 64),
		AsercionRef:               "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl",
		ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 2,
		ControlSesionHuellaSHA256: strings.Repeat("2", 64),
		CuentaRef:                 cuenta.CuentaRef, CuentaOrdinariaRef: cuenta.CuentaRef,
		Superficie:      domain.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    ahora.Add(-10 * time.Minute),
		SesionEmitidaEn:              ahora.Add(-9 * time.Minute), SesionRevalidadaEn: ahora.Add(-3 * time.Minute),
		SesionValidaHasta: ahora.Add(20 * time.Minute),
	}
	if err := resultado.Validar(); err != nil {
		t.Fatalf("resultado de contexto: %v", err)
	}
	if err := autenticacion.Validar(); err != nil {
		t.Fatalf("autenticacion: %v", err)
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV2(
		context.Background(), revalidadorOrdenAutorizacionV3Prueba{autenticacion},
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef,
		},
		resolutorOrdenAutorizacionV3Prueba{resultado},
		domain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: instantaneaActor.PerfilActivoRef},
		relojOrdenAutorizacionV3Prueba{ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	motivo := domain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion", CatalogoVersion: 2,
		CatalogoHuellaSHA256: strings.Repeat("d", 64),
		EntradaClave:         "motivo_11111111111111111111111111111111",
	}
	correlacion, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(), generadorCorrelacionOrdenAutorizacionV3Prueba{
			valor: "correlacion_11111111111111111111111111111111",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(
		domain.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: vinculo, ReferenciaMotivo: motivo,
			Accion: "bolsa.expediente.leer",
			Recurso: domain.RecursoAutorizable{
				Referencia: "expediente:1", ModuloID: "bolsa", Tipo: "expediente",
				Ambitos: map[string]string{"unidad": "seleccion"},
			},
			Finalidad: "gestion_bolsa", Correlacion: correlacion,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	version := domain.VersionRol{
		RolID: "tecnico_bolsa", Version: 1, Nombre: "Tecnico de bolsa",
		Estado: domain.EstadoVersionRolPublicada,
		Concesiones: []domain.ConcesionRol{{
			Accion: "bolsa.expediente.leer", ModuloID: "bolsa", TipoRecurso: "expediente",
			Finalidades: []string{"gestion_bolsa"}, GarantiaMinima: domain.AuthAssuranceSubstantial,
		}},
		PublicadaPor: "responsable-seguridad", PublicadaEn: ahora.Add(-24 * time.Hour),
	}
	huellaCatalogo, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := domain.InstantaneaAutorizacion{
		AsignacionPerfil: domain.AsignacionPerfil{
			AsignacionID: "asig-bolsa", Version: 1, PerfilActivoRef: instantaneaActor.PerfilActivoRef,
			PrincipalID: instantaneaActor.PersonaRef, VersionRolRef: version.Referencia(),
			Estado:       domain.EstadoAsignacionPerfilActiva,
			Ambitos:      []domain.AmbitoPerfil{{Clave: "unidad", Valores: []string{"seleccion"}}},
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
			EmitidaPor: "administrador-identidades", EmitidaEn: ahora.Add(-2 * time.Hour),
		},
		VersionRol: version,
		ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         domain.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn,
		},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	evidencia, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea, "dec_0123456789abcdef0123456789abcdef",
		ahora, ahora.Add(90*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := domain.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	return escenarioOrdenAutorizacionV3Prueba{
		ahora: ahora, solicitud: solicitud, decision: decision, motivo: motivo,
		resultado: resultado, instantnea: instantanea,
	}
}

func decisionDenegadaOrdenAutorizacionV3Prueba(
	t *testing.T,
	e escenarioOrdenAutorizacionV3Prueba,
) domain.DecisionAutorizacionLigadaV3 {
	t.Helper()
	instantanea := e.instantnea
	instantanea.VersionRol.Concesiones = append(
		[]domain.ConcesionRol(nil), e.instantnea.VersionRol.Concesiones...,
	)
	instantanea.VersionRol.Concesiones[0].Accion = "bolsa.expediente.modificar"
	evidencia, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(
		e.solicitud, instantanea, "dec_denegada56789abcdef0123456789abcdef",
		e.ahora, e.ahora.Add(90*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := domain.NuevaDecisionAutorizacionLigadaV3(e.solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	return decision
}

func TestOrdenesRegistroAutorizacionLigadaV3SonNominalesLigadasYDefensivas(t *testing.T) {
	e := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	orden, err := NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		e.solicitud, e.decision, e.motivo, e.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := orden.Datos()
	if err != nil || datos.Decision.ValidarPara(e.solicitud) != nil ||
		datos.ResultadoContexto.Validar() != nil || datos.ReferenciaMotivo != e.motivo {
		t.Fatalf("orden incompleta: %#v, %v", datos, err)
	}
	datos.ResultadoContexto.RepresentacionCanonica[0] ^= 1
	segunda, err := orden.Datos()
	if err != nil || segunda.ResultadoContexto.Validar() != nil {
		t.Fatalf("copia defensiva alterada: %v", err)
	}
	if _, err := NuevaOrdenRegistroDenegacionAutorizacionLigadaV3(
		e.solicitud, e.decision, e.motivo, e.resultado,
	); !errors.Is(err, ErrOrdenRegistroAutorizacionLigadaV3Invalida) {
		t.Fatalf("concesion aceptada por puerto de denegacion: %v", err)
	}
	if _, err := NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		e.solicitud, e.decision, domain.ReferenciaEntradaCatalogo{}, e.resultado,
	); !errors.Is(err, ErrOrdenRegistroAutorizacionLigadaV3Invalida) {
		t.Fatalf("motivo adulterado aceptado: %v", err)
	}
}

func TestCandidataRegistroDecisionAutorizacionLigadaV3EsUnionExactaYDefensiva(
	t *testing.T,
) {
	e := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	concesion, err := NuevaCandidataRegistroDecisionAutorizacionLigadaV3(
		e.solicitud, e.decision, e.motivo, e.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	esConcesion, ordenConcesion, ordenDenegacion, err := concesion.Resultado()
	if err != nil || !esConcesion ||
		!ordenConcesionAutorizacionLigadaV3Valida(ordenConcesion) ||
		ordenDenegacionAutorizacionLigadaV3Valida(ordenDenegacion) {
		t.Fatalf("variante de concesion incoherente: concedida=%t error=%v", esConcesion, err)
	}
	datos, err := ordenConcesion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datos.ResultadoContexto.RepresentacionCanonica[0] ^= 1
	segunda, err := ordenConcesion.Datos()
	if err != nil || segunda.ResultadoContexto.Validar() != nil {
		t.Fatalf("candidata retuvo una mutacion externa: %v", err)
	}

	decisionDenegada := decisionDenegadaOrdenAutorizacionV3Prueba(t, e)
	denegacion, err := NuevaCandidataRegistroDecisionAutorizacionLigadaV3(
		e.solicitud, decisionDenegada, e.motivo, e.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	esConcesion, ordenConcesion, ordenDenegacion, err = denegacion.Resultado()
	if err != nil || esConcesion ||
		ordenConcesionAutorizacionLigadaV3Valida(ordenConcesion) ||
		!ordenDenegacionAutorizacionLigadaV3Valida(ordenDenegacion) {
		t.Fatalf("variante de denegacion incoherente: concedida=%t error=%v", esConcesion, err)
	}

	ataques := []CandidataRegistroDecisionAutorizacionLigadaV3{
		{},
		{concedida: true, concesion: concesion.concesion, denegacion: denegacion.denegacion},
		{concesion: concesion.concesion},
		{concedida: true, denegacion: denegacion.denegacion},
	}
	for indice, ataque := range ataques {
		if _, _, _, err := ataque.Resultado(); !errors.Is(
			err, ErrCandidataRegistroDecisionAutorizacionLigadaV3Invalida,
		) {
			t.Fatalf("union adulterada %d aceptada: %v", indice, err)
		}
	}
}

func TestCandidataRegistroDecisionAutorizacionLigadaV3CierraCodecsYFormateo(
	t *testing.T,
) {
	e := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	candidata, err := NuevaCandidataRegistroDecisionAutorizacionLigadaV3(
		e.solicitud, e.decision, e.motivo, e.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(candidata); !errors.Is(
		err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida,
	) {
		t.Fatalf("JSON permitido: %v", err)
	}
	if _, err := candidata.MarshalText(); !errors.Is(
		err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida,
	) {
		t.Fatalf("texto permitido: %v", err)
	}
	if _, err := candidata.MarshalBinary(); !errors.Is(
		err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida,
	) {
		t.Fatalf("binario permitido: %v", err)
	}
	if _, err := candidata.GobEncode(); !errors.Is(
		err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida,
	) {
		t.Fatalf("gob permitido: %v", err)
	}
	if _, err := candidata.MarshalCBOR(); !errors.Is(
		err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida,
	) {
		t.Fatalf("CBOR permitido: %v", err)
	}
	if _, err := candidata.MarshalYAML(); !errors.Is(
		err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida,
	) {
		t.Fatalf("YAML permitido: %v", err)
	}
	if err := candidata.MarshalXML(
		xml.NewEncoder(&bytes.Buffer{}), xml.StartElement{},
	); !errors.Is(err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida) {
		t.Fatalf("XML permitido: %v", err)
	}
	if err := json.Unmarshal([]byte(`{}`), &candidata); !errors.Is(
		err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida,
	) {
		t.Fatalf("JSON de entrada permitido: %v", err)
	}
	if err := candidata.UnmarshalText(nil); !errors.Is(
		err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida,
	) {
		t.Fatalf("texto de entrada permitido: %v", err)
	}
	if err := candidata.UnmarshalBinary(nil); !errors.Is(
		err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida,
	) {
		t.Fatalf("binario de entrada permitido: %v", err)
	}
	if err := candidata.GobDecode(nil); !errors.Is(
		err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida,
	) {
		t.Fatalf("gob de entrada permitido: %v", err)
	}
	if err := candidata.UnmarshalCBOR(nil); !errors.Is(
		err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida,
	) {
		t.Fatalf("CBOR de entrada permitido: %v", err)
	}
	if err := candidata.UnmarshalYAML(func(any) error { return nil }); !errors.Is(
		err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida,
	) {
		t.Fatalf("YAML de entrada permitido: %v", err)
	}
	if err := candidata.UnmarshalXML(
		xml.NewDecoder(bytes.NewReader(nil)), xml.StartElement{},
	); !errors.Is(err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida) {
		t.Fatalf("XML de entrada permitido: %v", err)
	}
	if _, _, _, err := candidata.Resultado(); err != nil {
		t.Fatalf("un intento de deserializacion altero la candidata: %v", err)
	}
	if texto := fmt.Sprintf("%v %#v", candidata, candidata); texto !=
		"[REGISTRO-AUTORIZACION-LIGADA-V3-OPACO] [REGISTRO-AUTORIZACION-LIGADA-V3-OPACO]" {
		t.Fatalf("formateo no opaco: %q", texto)
	}
	slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)).Info(
		"candidata", "valor", candidata,
	)
}

func TestConfirmacionRegistroConcesionAutorizacionLigadaV3LigaOrdenYVentanaSinPII(t *testing.T) {
	e := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	orden, err := NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		e.solicitud, e.decision, e.motivo, e.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion, err := nuevaConfirmacionRegistroConcesionAutorizacionLigadaV3(
		orden, e.ahora.Add(time.Microsecond),
	)
	if err != nil || confirmacion.ValidarPara(orden) != nil {
		t.Fatalf("confirmacion valida rechazada: %v", err)
	}
	datos, err := confirmacion.Datos()
	if err != nil || datos.DecisionRef == "" || len(datos.DecisionHuellaSHA256) != 64 ||
		!confirmacion.DentroDeVentanaEn(datos.RegistradaEn) ||
		confirmacion.DentroDeVentanaEn(datos.ValidaHasta) {
		t.Fatalf("confirmacion/ventana invalida: %#v, %v", datos, err)
	}
	tipo := reflect.TypeOf(datos)
	for indice := 0; indice < tipo.NumField(); indice++ {
		nombre := strings.ToLower(tipo.Field(indice).Name)
		for _, prohibido := range []string{"principal", "persona", "perfil", "cuenta", "dni", "contexto"} {
			if strings.Contains(nombre, prohibido) {
				t.Fatalf("PII expuesta: %s", tipo.Field(indice).Name)
			}
		}
	}
	if confirmacion.DentroDeVentanaEn(e.ahora.Add(time.Nanosecond)) ||
		(ConfirmacionRegistroConcesionAutorizacionLigadaV3{}).DentroDeVentanaEn(e.ahora) {
		t.Fatal("instante no canonico o valor cero aceptado")
	}
	evidenciaAjena, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(
		e.solicitud, e.instantnea, "dec_otra23456789abcdef0123456789abcdef",
		e.ahora, e.ahora.Add(90*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	decisionAjena, err := domain.NuevaDecisionAutorizacionLigadaV3(e.solicitud, evidenciaAjena)
	if err != nil {
		t.Fatal(err)
	}
	ordenAjena, err := NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		e.solicitud, decisionAjena, e.motivo, e.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	if confirmacion.ValidarPara(ordenAjena) == nil {
		t.Fatal("confirmacion ajena/replay cruzado aceptado")
	}
}

func TestFabricaConfirmacionRegistroConcesionAutorizacionLigadaV3SoloTrasRetornoDurable(
	t *testing.T,
) {
	e := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	orden, err := NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		e.solicitud, e.decision, e.motivo, e.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	registro := &registroConcesionLigadaV3Prueba{registradaEn: e.ahora, cancelar: cancelar}
	confirmacion, err := RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
		ctx, registro, orden,
	)
	if err != nil || confirmacion.ValidarPara(orden) != nil || registro.invocaciones != 1 {
		t.Fatalf("retorno durable valido rechazado por cancelacion tardia: %v", err)
	}

	var registroNulo *registroConcesionLigadaV3Prueba
	confirmacion, err = RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
		context.Background(), registroNulo, orden,
	)
	if !errors.Is(err, ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) ||
		confirmacion.Validar() == nil {
		t.Fatalf("registro nulo tipado invocado o confirmado: %v", err)
	}

	ctxCancelado, cancelarAntes := context.WithCancel(context.Background())
	cancelarAntes()
	registro = &registroConcesionLigadaV3Prueba{registradaEn: e.ahora}
	_, err = RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
		ctxCancelado, registro, orden,
	)
	if !errors.Is(err, context.Canceled) || registro.invocaciones != 0 {
		t.Fatalf("cancelacion previa alcanzo registro: invocaciones=%d err=%v", registro.invocaciones, err)
	}

	secreto := "postgres://usuario:clave@servidor/vec persona=12345678Z"
	fallo := errors.New(secreto)
	registro = &registroConcesionLigadaV3Prueba{err: fallo}
	_, err = RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
		context.Background(), registro, orden,
	)
	var bitacora bytes.Buffer
	slog.New(slog.NewTextHandler(&bitacora, nil)).Error("registro", "error", err)
	if !errors.Is(err, fallo) ||
		!errors.Is(err, ErrRegistroConcesionAutorizacionLigadaV3NoDisponible) ||
		strings.Contains(fmt.Sprintf("%v %+v %#v", err, err, err), secreto) ||
		strings.Contains(bitacora.String(), secreto) {
		t.Fatalf("error de registro no saneado o no trazable: %v", err)
	}
}

func TestRegistroAutorizacionLigadaV3CierraCodecsYFormateo(t *testing.T) {
	e := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	orden, err := NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		e.solicitud, e.decision, e.motivo, e.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion, err := nuevaConfirmacionRegistroConcesionAutorizacionLigadaV3(
		orden, e.ahora.Add(time.Microsecond),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(confirmacion); !errors.Is(err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida) {
		t.Fatalf("JSON permitido: %v", err)
	}
	if _, err := confirmacion.MarshalText(); !errors.Is(err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida) {
		t.Fatalf("texto permitido: %v", err)
	}
	if _, err := confirmacion.MarshalBinary(); !errors.Is(err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida) {
		t.Fatalf("binario permitido: %v", err)
	}
	if _, err := confirmacion.GobEncode(); !errors.Is(err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida) {
		t.Fatalf("gob permitido: %v", err)
	}
	if _, err := confirmacion.MarshalCBOR(); !errors.Is(err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida) {
		t.Fatalf("CBOR permitido: %v", err)
	}
	if _, err := confirmacion.MarshalYAML(); !errors.Is(err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida) {
		t.Fatalf("YAML permitido: %v", err)
	}
	if err := confirmacion.MarshalXML(
		xml.NewEncoder(&bytes.Buffer{}), xml.StartElement{},
	); !errors.Is(err, ErrSerializacionRegistroAutorizacionLigadaV3Prohibida) {
		t.Fatalf("XML permitido: %v", err)
	}
	if texto := fmt.Sprintf("%v %#v", confirmacion, confirmacion); texto !=
		"[REGISTRO-AUTORIZACION-LIGADA-V3-OPACO] [REGISTRO-AUTORIZACION-LIGADA-V3-OPACO]" {
		t.Fatalf("formateo no opaco: %q", texto)
	}
	registro := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	registro.Info("confirmacion", "valor", confirmacion)
	if (OrdenRegistroConcesionCandidataAutorizacionLigadaV3{}).datos != nil ||
		(OrdenRegistroDenegacionAutorizacionLigadaV3{}).datos != nil ||
		(ConfirmacionRegistroConcesionAutorizacionLigadaV3{}).Validar() == nil {
		t.Fatal("valor cero se convirtio en capacidad")
	}
}

func TestResumenCandidataRegistroDecisionAutorizacionLigadaV3CubreAmbasRamas(
	t *testing.T,
) {
	escenario := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	denegada := decisionDenegadaOrdenAutorizacionV3Prueba(t, escenario)
	casos := []struct {
		nombre   string
		decision domain.DecisionAutorizacionLigadaV3
	}{
		{nombre: "concedida", decision: escenario.decision},
		{nombre: "denegada", decision: denegada},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			candidata, err := NuevaCandidataRegistroDecisionAutorizacionLigadaV3(
				escenario.solicitud,
				caso.decision,
				escenario.motivo,
				escenario.resultado,
			)
			if err != nil {
				t.Fatal(err)
			}
			resumen, err := candidata.Resumen()
			if err != nil || resumen.ValidarPara(candidata) != nil {
				t.Fatalf("resumen nominal válido rechazado: %v", err)
			}
			datos, err := resumen.Datos()
			concedida, codigo, errResultado := caso.decision.Resultado()
			emitidaEn, validaHasta, errVentana := caso.decision.VentanaValidez()
			huella, errHuella := domain.HuellaSHA256DecisionAutorizacionV3(
				caso.decision,
			)
			if err != nil || errResultado != nil || errVentana != nil ||
				errHuella != nil ||
				datos.DecisionRef == "" ||
				datos.DecisionHuellaSHA256 != huella ||
				datos.CodigoProbatorio != codigo ||
				datos.Concedida != concedida ||
				!datos.EmitidaEn.Equal(emitidaEn) ||
				!datos.ValidaHasta.Equal(validaHasta) {
				t.Fatalf("resumen incompleto: %#v / %v", datos, err)
			}
			datos.DecisionRef = "decision_adulterada_en_copia"
			segunda, err := resumen.Datos()
			if err != nil || segunda.DecisionRef == datos.DecisionRef {
				t.Fatal("el resumen compartió su copia defensiva")
			}
		})
	}
}

func TestResumenCandidataRegistroDecisionAutorizacionLigadaV3RechazaAdulteracion(
	t *testing.T,
) {
	escenario := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	candidata, err := NuevaCandidataRegistroDecisionAutorizacionLigadaV3(
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	base, err := candidata.Resumen()
	if err != nil {
		t.Fatal(err)
	}
	casos := map[string]func(*DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3){
		"referencia": func(d *DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3) {
			d.DecisionRef = "decision_ajena_0123456789abcdef"
		},
		"huella": func(d *DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3) {
			d.DecisionHuellaSHA256 = strings.Repeat("7", 64)
		},
		"codigo": func(d *DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3) {
			d.CodigoProbatorio = "accion_no_concedida"
		},
		"rama": func(d *DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3) {
			d.Concedida = false
			d.CodigoProbatorio = "accion_no_concedida"
		},
		"emision": func(d *DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3) {
			d.EmitidaEn = d.EmitidaEn.Add(time.Microsecond)
		},
		"vigencia": func(d *DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3) {
			d.ValidaHasta = d.ValidaHasta.Add(-time.Microsecond)
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			datos := *base.datos
			mutar(&datos)
			adulterado := ResumenCandidataRegistroDecisionAutorizacionLigadaV3{
				datos: &datos,
			}
			if adulterado.ValidarPara(candidata) == nil {
				t.Fatal("resumen adulterado aceptado")
			}
		})
	}

	denegada := decisionDenegadaOrdenAutorizacionV3Prueba(t, escenario)
	otraCandidata, err := NuevaCandidataRegistroDecisionAutorizacionLigadaV3(
		escenario.solicitud,
		denegada,
		escenario.motivo,
		escenario.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	if base.ValidarPara(otraCandidata) == nil {
		t.Fatal("resumen concedido aceptado para una candidata denegada")
	}
}

func TestResumenCandidataRegistroDecisionAutorizacionLigadaV3EsOpaco(
	t *testing.T,
) {
	escenario := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	candidata, err := NuevaCandidataRegistroDecisionAutorizacionLigadaV3(
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
	)
	if err != nil {
		t.Fatal(err)
	}
	resumen, err := candidata.Resumen()
	if err != nil {
		t.Fatal(err)
	}
	datos, err := resumen.Datos()
	if err != nil {
		t.Fatal(err)
	}
	for nombre, valor := range map[string]any{
		"resumen": resumen,
		"datos":   datos,
	} {
		if _, err := json.Marshal(valor); !errors.Is(
			err,
			ErrSerializacionRegistroAutorizacionLigadaV3Prohibida,
		) {
			t.Fatalf("%s serializable: %v", nombre, err)
		}
		texto := fmt.Sprintf("%v %#v", valor, valor)
		if strings.Contains(texto, datos.DecisionRef) ||
			strings.Contains(texto, datos.DecisionHuellaSHA256) {
			t.Fatalf("%s filtró material VEC: %q", nombre, texto)
		}
		var bitacora bytes.Buffer
		slog.New(slog.NewTextHandler(&bitacora, nil)).Info(
			"resumen",
			"valor",
			valor,
		)
		if strings.Contains(bitacora.String(), datos.DecisionRef) ||
			strings.Contains(bitacora.String(), datos.DecisionHuellaSHA256) {
			t.Fatalf("%s filtró material VEC en log", nombre)
		}
	}
}
