package ports

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

const (
	claveMotivoConsultaAutoridadPrueba      = "motivo_0123456789abcdef0123456789abcdef"
	claveMotivoConsultaAutoridadAjenaPrueba = "motivo_fedcba9876543210fedcba9876543210"
	correlacionConsultaAutoridadPrueba      = "correlacion_0123456789abcdef0123456789abcdef"
	correlacionConsultaAutoridadAjenaPrueba = "correlacion_fedcba9876543210fedcba9876543210"
)

func solicitudConsultaGobernadaAutoridadPrueba(
	t *testing.T,
	selector SelectorVersionFuenteAutoridad,
) (SolicitudConsultaInternaGobernadaFuenteAutoridad, domain.DecisionAutorizacion, time.Time) {
	t.Helper()
	decision, solicitadaEn := decisionAutorizacionReforzadaPrueba(t)
	motivoCatalogo := motivoCatalogoConsultaAutoridadPrueba(claveMotivoConsultaAutoridadPrueba)
	recurso, err := RecursoAutorizableConsultaInternaFuenteAutoridad(selector, motivoCatalogo)
	if err != nil {
		t.Fatal(err)
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	decision.Accion = AccionConsultarFuenteAutoridadInterna
	decision.RecursoRef = recurso.Referencia
	decision.ModuloID = recurso.ModuloID
	decision.TipoRecurso = recurso.Tipo
	decision.ContextoRecursoHuellaSHA256 = huellaContexto
	decision.Finalidad = "gobierno_fuentes_autoridad"
	decision.CorrelacionRef = correlacionConsultaAutoridadPrueba
	decision.CamposPermitidos = []string{CampoConsultaInternaFuenteAutoridad}
	decision.Obligaciones = nil
	decision.GarantiaMinima = domain.AuthAssuranceHigh
	ligarDecisionSolicitudConsultaAutoridadPrueba(t, &decision, recurso, motivoCatalogo)
	if err := decision.ValidarEvidenciaInstantanea(); err != nil {
		t.Fatalf("decision de consulta: %v", err)
	}
	evidencia, err := NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(decision, solicitadaEn)
	if err != nil {
		t.Fatalf("evidencia de consulta: %v", err)
	}
	datosVinculo, err := decision.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatal(err)
	}
	auditoria := domain.AuditEntry{
		ActorID: decision.PrincipalID, ActorProfile: decision.PerfilActivoRef,
		AuthMethod: datosVinculo.MetodoObservado, AuthAssurance: datosVinculo.GarantiaObservada,
		AuthorizationRef: decision.DecisionRef,
		Purpose:          decision.Finalidad, Action: decision.Accion, ModuleID: recurso.ModuloID,
		SubjectRef: recurso.Referencia, Reason: motivoCatalogo.EntradaClave, CorrelationRef: decision.CorrelacionRef,
		Metadata: map[string]string{
			"fuente_id": selector.FuenteID, "fuente_version": fmt.Sprint(selector.Version),
			AtributoMotivoCatalogoIDConsultaAutoridad:      motivoCatalogo.CatalogoID,
			AtributoMotivoCatalogoVersionConsultaAutoridad: fmt.Sprint(motivoCatalogo.CatalogoVersion),
			AtributoMotivoCatalogoHuellaConsultaAutoridad:  motivoCatalogo.CatalogoHuellaSHA256,
			AtributoMotivoEntradaClaveConsultaAutoridad:    motivoCatalogo.EntradaClave,
		},
		OccurredAt: solicitadaEn,
	}
	solicitud, err := NuevaSolicitudConsultaInternaGobernadaFuenteAutoridad(
		selector, evidencia, auditoria, motivoCatalogo, decision.CorrelacionRef, solicitadaEn,
	)
	if err != nil {
		t.Fatalf("solicitud gobernada: %v", err)
	}
	return solicitud, decision, solicitadaEn
}

func ligarDecisionSolicitudConsultaAutoridadPrueba(
	t *testing.T,
	decision *domain.DecisionAutorizacion,
	recurso domain.RecursoAutorizable,
	motivo domain.ReferenciaEntradaCatalogo,
) {
	t.Helper()
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV2(
		domain.DatosSolicitudAutorizacionLigadaV2{
			VinculoAutenticacionActor: decision.VinculoAutenticacionActor,
			ReferenciaMotivo:          motivo,
			Accion:                    decision.Accion,
			Recurso:                   recurso,
			Finalidad:                 decision.Finalidad,
			CorrelacionRef:            decision.CorrelacionRef,
		},
	)
	if err != nil {
		t.Fatalf("solicitud nominal V2: %v", err)
	}
	huellaSolicitud, err := domain.HuellaSHA256SolicitudAutorizacionV2(solicitud)
	if err != nil {
		t.Fatalf("huella de solicitud: %v", err)
	}
	huellaMotivo, err := domain.HuellaSHA256MotivoAutorizacionV2(motivo)
	if err != nil {
		t.Fatalf("huella de motivo: %v", err)
	}
	decision.EsquemaHuellaSolicitud = domain.EsquemaHuellaSolicitudAutorizacionV2
	decision.SolicitudHuellaSHA256 = huellaSolicitud
	decision.EsquemaHuellaMotivo = domain.EsquemaHuellaMotivoAutorizacionV2
	decision.MotivoHuellaSHA256 = huellaMotivo
}

func motivoCatalogoConsultaAutoridadPrueba(clave string) domain.ReferenciaEntradaCatalogo {
	return domain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_consulta_fuentes_autoridad", CatalogoVersion: 3,
		CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: clave,
	}
}

func TestReferenciasServidorAutoridadSeparanNamespacesYSonOpacas(t *testing.T) {
	sufijo := "01J2ABCDEF0123456789XYZ"
	solicitud, err := NuevaReferenciaSolicitudFuenteAutoridad(
		PrefijoReferenciaSolicitudFuenteAutoridad + sufijo,
	)
	if err != nil {
		t.Fatal(err)
	}
	operacion, err := NuevaReferenciaOperacionFuenteAutoridad(
		PrefijoReferenciaOperacionFuenteAutoridad + sufijo,
	)
	if err != nil {
		t.Fatal(err)
	}
	refSolicitud, _ := solicitud.Referencia()
	refOperacion, _ := operacion.Referencia()
	if refSolicitud == refOperacion || !strings.HasPrefix(refSolicitud, PrefijoReferenciaSolicitudFuenteAutoridad) ||
		!strings.HasPrefix(refOperacion, PrefijoReferenciaOperacionFuenteAutoridad) {
		t.Fatalf("namespaces no separados: %s / %s", refSolicitud, refOperacion)
	}
	if _, err := NuevaReferenciaSolicitudFuenteAutoridad(refOperacion); !errors.Is(err, ErrReferenciaGeneradaFuenteAutoridadInvalida) {
		t.Fatalf("operacion aceptada como solicitud: %v", err)
	}
	if _, err := NuevaReferenciaOperacionFuenteAutoridad(refSolicitud); !errors.Is(err, ErrReferenciaGeneradaFuenteAutoridadInvalida) {
		t.Fatalf("solicitud aceptada como operacion: %v", err)
	}
	for _, invalida := range []string{
		PrefijoReferenciaSolicitudFuenteAutoridad + "corta",
		PrefijoReferenciaSolicitudFuenteAutoridad + "0123456789abcdefghijkl:",
		PrefijoReferenciaSolicitudFuenteAutoridad + "0123456789ábcdefghijkl",
	} {
		if _, err := NuevaReferenciaSolicitudFuenteAutoridad(invalida); err == nil {
			t.Fatalf("referencia invalida aceptada: %q", invalida)
		}
	}
	if _, err := json.Marshal(solicitud); !errors.Is(err, ErrSerializacionReferenciaAutoridad) {
		t.Fatalf("JSON no bloqueado: %v", err)
	}
	if _, err := solicitud.GobEncode(); !errors.Is(err, ErrSerializacionReferenciaAutoridad) {
		t.Fatalf("gob no bloqueado: %v", err)
	}
	if _, err := xml.Marshal(operacion); !errors.Is(err, ErrSerializacionReferenciaAutoridad) {
		t.Fatalf("XML no bloqueado: %v", err)
	}
	texto := fmt.Sprintf("%v %+v %#v", solicitud, solicitud, solicitud)
	if strings.Contains(texto, sufijo) {
		t.Fatalf("formato expuso referencia: %s", texto)
	}
}

func TestConsultaInternaGobernadaLigaDecisionAuditoriaYResultado(t *testing.T) {
	escenario := nuevoEscenarioPuertoAutoridad(t)
	selector := SelectorVersionFuenteAutoridad{
		FuenteID: escenario.Fuente.ID, Version: escenario.Fuente.Version,
	}
	solicitud, decision, solicitadaEn := solicitudConsultaGobernadaAutoridadPrueba(t, selector)
	selectorRecibido, err := solicitud.Selector()
	if err != nil || selectorRecibido != selector {
		t.Fatalf("selector defensivo: %+v, %v", selectorRecibido, err)
	}
	auditoria, err := solicitud.Auditoria()
	if err != nil {
		t.Fatal(err)
	}
	auditoria.Metadata["fuente_id"] = "alterada"
	auditoriaOtraVez, err := solicitud.Auditoria()
	if err != nil || auditoriaOtraVez.Metadata["fuente_id"] != selector.FuenteID {
		t.Fatal("la solicitud compartio auditoria mutable")
	}
	motivoCatalogo, err := solicitud.MotivoCatalogo()
	if err != nil || motivoCatalogo.Validar() != nil ||
		motivoCatalogo.EntradaClave != claveMotivoConsultaAutoridadPrueba {
		t.Fatalf("referencia catalogada perdida: %+v, %v", motivoCatalogo, err)
	}
	autorizacion, err := solicitud.Autorizacion()
	if err != nil || autorizacion.ValidarEn(solicitadaEn) != nil {
		t.Fatalf("evidencia descartada: %v", err)
	}
	datosAutorizacion, _ := autorizacion.Datos()
	estado, _ := EstadoExactoFuenteAutoridad(escenario.Fuente)
	auditoriaConfirmada, err := PrepararAuditoriaResultadoConsultaInternaFuenteAutoridad(
		solicitud, ResultadoConsultaFuenteEncontrada, estado,
	)
	if err != nil {
		t.Fatal(err)
	}
	auditoriaConfirmada.ID = "auditoria:consulta:autoridad:1"
	auditoriaConfirmada.Seq = 7
	auditoriaConfirmada.IntegrityAlgorithm = "sha256-chain-v1"
	auditoriaConfirmada.PrevSignature = "firma:auditoria:autoridad:anterior"
	auditoriaConfirmada.Signature = "firma:auditoria:autoridad:actual"
	datosRecibo := DatosReciboConsultaInternaFuenteAutoridad{
		TransaccionRef: "transaccion:consulta:autoridad:1", Selector: selector,
		Resultado: ResultadoConsultaFuenteEncontrada, DecisionRef: decision.DecisionRef,
		HuellaDecisionSHA256: datosAutorizacion.HuellaDecisionSHA256,
		Estado:               estado, AuditoriaRef: auditoriaConfirmada.ID,
		AuditoriaSecuencia:             auditoriaConfirmada.Seq,
		AuditoriaAlgoritmoIntegridad:   auditoriaConfirmada.IntegrityAlgorithm,
		AuditoriaEncadenadoAnteriorRef: auditoriaConfirmada.PrevSignature,
		AuditoriaFirmaRef:              auditoriaConfirmada.Signature,
		AuditoriaConfirmada:            auditoriaConfirmada, ConfirmadaEn: solicitadaEn.Add(time.Second),
	}
	recibo, err := NuevoReciboConsultaInternaFuenteAutoridad(solicitud, datosRecibo)
	if err != nil {
		t.Fatalf("crear recibo encontrado: %v", err)
	}
	resultado := ResultadoConsultaInternaFuenteAutoridad{
		Encontrada: true, Fuente: escenario.Fuente, Estado: estado, Recibo: recibo,
	}
	clon, err := resultado.ClonarPara(solicitud)
	if err != nil || clon.ValidarPara(solicitud) != nil {
		t.Fatalf("resultado encontrado rechazado: %v", err)
	}
	clon.Fuente.ID = "alterada"
	if resultado.Fuente.ID != escenario.Fuente.ID {
		t.Fatal("resultado compartio agregado mutable")
	}
	estadoDistinto := estado
	estadoDistinto.HuellaEstadoSHA256 = huellaPuertoAutoridadPrueba('9')
	datosOtroEstado := datosRecibo
	datosOtroEstado.Estado = estadoDistinto
	datosOtroEstado.AuditoriaHuellaEntradaSHA256 = ""
	datosOtroEstado.HuellaCompromisoReciboSHA256 = ""
	if _, err := NuevoReciboConsultaInternaFuenteAutoridad(solicitud, datosOtroEstado); !errors.Is(err, ErrReciboConsultaFuenteAutoridadInvalido) {
		t.Fatalf("recibo ligado a otro snapshot aceptado: %v", err)
	}
	datosSinEstado := datosRecibo
	datosSinEstado.Estado = ReferenciaEstadoFuenteAutoridad{}
	if _, err := NuevoReciboConsultaInternaFuenteAutoridad(solicitud, datosSinEstado); !errors.Is(err, ErrReciboConsultaFuenteAutoridadInvalido) {
		t.Fatalf("resultado encontrado sin estado aceptado: %v", err)
	}

	datosAusencia := datosRecibo
	datosAusencia.TransaccionRef = "transaccion:consulta:autoridad:2"
	datosAusencia.AuditoriaRef = "auditoria:consulta:autoridad:2"
	datosAusencia.Resultado = ResultadoConsultaFuenteNoEncontrada
	datosAusencia.Estado = ReferenciaEstadoFuenteAutoridad{}
	datosAusencia.AuditoriaHuellaEntradaSHA256 = ""
	datosAusencia.HuellaCompromisoReciboSHA256 = ""
	auditoriaAusencia, err := PrepararAuditoriaResultadoConsultaInternaFuenteAutoridad(
		solicitud, datosAusencia.Resultado, datosAusencia.Estado,
	)
	if err != nil {
		t.Fatal(err)
	}
	auditoriaAusencia.ID = datosAusencia.AuditoriaRef
	auditoriaAusencia.Seq = 8
	auditoriaAusencia.IntegrityAlgorithm = "sha256-chain-v1"
	auditoriaAusencia.PrevSignature = auditoriaConfirmada.Signature
	auditoriaAusencia.Signature = "firma:auditoria:autoridad:ausencia"
	datosAusencia.AuditoriaSecuencia = auditoriaAusencia.Seq
	datosAusencia.AuditoriaAlgoritmoIntegridad = auditoriaAusencia.IntegrityAlgorithm
	datosAusencia.AuditoriaEncadenadoAnteriorRef = auditoriaAusencia.PrevSignature
	datosAusencia.AuditoriaFirmaRef = auditoriaAusencia.Signature
	datosAusencia.AuditoriaConfirmada = auditoriaAusencia
	reciboAusencia, err := NuevoReciboConsultaInternaFuenteAutoridad(solicitud, datosAusencia)
	if err != nil {
		t.Fatalf("crear recibo de ausencia: %v", err)
	}
	noEncontrada := ResultadoConsultaInternaFuenteAutoridad{Recibo: reciboAusencia}
	if err := noEncontrada.ValidarPara(solicitud); err != nil {
		t.Fatalf("ausencia autorizada con recibo rechazada: %v", err)
	}
	datosAusencia.Estado = estado
	datosAusencia.AuditoriaHuellaEntradaSHA256 = ""
	datosAusencia.HuellaCompromisoReciboSHA256 = ""
	if _, err := NuevoReciboConsultaInternaFuenteAutoridad(solicitud, datosAusencia); !errors.Is(err, ErrReciboConsultaFuenteAutoridadInvalido) {
		t.Fatalf("ausencia con snapshot encubierto aceptada: %v", err)
	}
	if _, err := json.Marshal(solicitud); !errors.Is(err, ErrSerializacionGobiernoFuenteAutoridad) {
		t.Fatalf("solicitud serializable: %v", err)
	}
	if _, err := json.Marshal(resultado); !errors.Is(err, ErrSerializacionGobiernoFuenteAutoridad) {
		t.Fatalf("resultado serializable: %v", err)
	}
	if _, err := recibo.MarshalBinary(); !errors.Is(err, ErrSerializacionGobiernoFuenteAutoridad) {
		t.Fatalf("recibo binario serializable: %v", err)
	}
	if _, err := solicitud.MarshalCBOR(); !errors.Is(err, ErrSerializacionGobiernoFuenteAutoridad) {
		t.Fatalf("solicitud CBOR serializable: %v", err)
	}
	if _, err := resultado.MarshalYAML(); !errors.Is(err, ErrSerializacionGobiernoFuenteAutoridad) {
		t.Fatalf("resultado YAML serializable: %v", err)
	}
	var reconstruido SolicitudConsultaInternaGobernadaFuenteAutoridad
	if err := reconstruido.UnmarshalCBOR([]byte{0xa0}); !errors.Is(err, ErrSerializacionGobiernoFuenteAutoridad) {
		t.Fatalf("solicitud reconstruible desde CBOR: %v", err)
	}
	if err := reconstruido.UnmarshalYAML(func(any) error { return nil }); !errors.Is(err, ErrSerializacionGobiernoFuenteAutoridad) {
		t.Fatalf("solicitud reconstruible desde YAML: %v", err)
	}
	if _, err := json.Marshal(datosRecibo); !errors.Is(err, ErrSerializacionGobiernoFuenteAutoridad) {
		t.Fatalf("datos de recibo serializables: %v", err)
	}
	textoInterno := fmt.Sprintf("%v %+v %#v", datosRecibo, resultado, recibo)
	if strings.Contains(textoInterno, decision.DecisionRef) ||
		strings.Contains(textoInterno, datosRecibo.TransaccionRef) {
		t.Fatalf("formato de gobierno expuso capacidades: %s", textoInterno)
	}
}

func TestConsultaInternaGobernadaRechazaDecisionOAuditoriaDivergente(t *testing.T) {
	escenario := nuevoEscenarioPuertoAutoridad(t)
	selector := SelectorVersionFuenteAutoridad{FuenteID: escenario.Fuente.ID, Version: 1}
	solicitud, _, solicitadaEn := solicitudConsultaGobernadaAutoridadPrueba(t, selector)
	autorizacion, _ := solicitud.Autorizacion()
	auditoria, _ := solicitud.Auditoria()
	motivoCatalogo, _ := solicitud.MotivoCatalogo()
	correlacion, _ := solicitud.CorrelacionRef()

	mutacionesAuditoria := []struct {
		nombre string
		mutar  func(*domain.AuditEntry)
	}{
		{"actor", func(a *domain.AuditEntry) { a.ActorID = "per_otro_actor_00000000000001" }},
		{"perfil", func(a *domain.AuditEntry) { a.ActorProfile = "prf_otro_perfil_000000000001" }},
		{"finalidad", func(a *domain.AuditEntry) { a.Purpose = "otra_finalidad" }},
		{"correlacion", func(a *domain.AuditEntry) { a.CorrelationRef = "corr:otra" }},
		{"accion", func(a *domain.AuditEntry) { a.Action = "otra_accion" }},
		{"recurso", func(a *domain.AuditEntry) { a.SubjectRef = "fuente:otra:v1" }},
		{"roles", func(a *domain.AuditEntry) { a.ActorRoles = []string{"rol:rrhh"} }},
		{"motivo", func(a *domain.AuditEntry) { a.Reason = claveMotivoConsultaAutoridadAjenaPrueba }},
		{"version nativa", func(a *domain.AuditEntry) { a.ObjectVersion = 1 }},
		{"firma anticipada", func(a *domain.AuditEntry) { a.Signature = "firma:no:durable" }},
	}
	for _, caso := range mutacionesAuditoria {
		t.Run("auditoria/"+caso.nombre, func(t *testing.T) {
			alterada := clonarAuditoriaFuenteAutoridad(auditoria)
			caso.mutar(&alterada)
			if _, err := NuevaSolicitudConsultaInternaGobernadaFuenteAutoridad(
				selector, autorizacion, alterada, motivoCatalogo, correlacion, solicitadaEn,
			); !errors.Is(err, ErrConsultaInternaFuenteAutoridadInvalida) {
				t.Fatalf("divergencia aceptada: %v", err)
			}
		})
	}
	datosAutorizacion, _ := autorizacion.Datos()
	decisionBase := datosAutorizacion.Decision
	mutacionesDecision := []struct {
		nombre string
		mutar  func(*domain.DecisionAutorizacion)
	}{
		{"accion", func(d *domain.DecisionAutorizacion) { d.Accion = "otra_accion" }},
		{"recurso", func(d *domain.DecisionAutorizacion) { d.RecursoRef = "fuente:otra:v1" }},
		{"contexto", func(d *domain.DecisionAutorizacion) { d.ContextoRecursoHuellaSHA256 = huellaPuertoAutoridadPrueba('8') }},
		{"campo extra", func(d *domain.DecisionAutorizacion) { d.CamposPermitidos = append(d.CamposPermitidos, "dni") }},
		{"obligacion", func(d *domain.DecisionAutorizacion) { d.Obligaciones = []string{"doble_control"} }},
	}
	for _, caso := range mutacionesDecision {
		t.Run("decision/"+caso.nombre, func(t *testing.T) {
			decision := clonarDecisionAutorizacionCanonica(decisionBase)
			caso.mutar(&decision)
			evidencia, err := NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(decision, solicitadaEn)
			if err != nil {
				return
			}
			if _, err := NuevaSolicitudConsultaInternaGobernadaFuenteAutoridad(
				selector, evidencia, auditoria, motivoCatalogo, correlacion, solicitadaEn,
			); !errors.Is(err, ErrConsultaInternaFuenteAutoridadInvalida) {
				t.Fatalf("decision divergente aceptada: %v", err)
			}
		})
	}
}

func TestConsultaGobernadaEsSeguraEnLecturasConcurrentes(t *testing.T) {
	escenario := nuevoEscenarioPuertoAutoridad(t)
	solicitud, _, _ := solicitudConsultaGobernadaAutoridadPrueba(t, SelectorVersionFuenteAutoridad{
		FuenteID: escenario.Fuente.ID, Version: 1,
	})
	const lectores = 16
	var grupo sync.WaitGroup
	errores := make(chan error, lectores)
	for indice := 0; indice < lectores; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			auditoria, err := solicitud.Auditoria()
			if err == nil {
				auditoria.Metadata["fuente_id"] = "local"
			}
			errores <- err
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("lectura concurrente: %v", err)
		}
	}
}

type generadorReferenciasAutoridadContrato struct{}

func (generadorReferenciasAutoridadContrato) NuevaReferenciaSolicitud(context.Context) (
	ReferenciaSolicitudFuenteAutoridad,
	error,
) {
	return ReferenciaSolicitudFuenteAutoridad{}, nil
}

func (generadorReferenciasAutoridadContrato) NuevaReferenciaOperacion(context.Context) (
	ReferenciaOperacionFuenteAutoridad,
	error,
) {
	return ReferenciaOperacionFuenteAutoridad{}, nil
}

type consultaInternaAutoridadContrato struct{}

func (consultaInternaAutoridadContrato) ConsultarVersionExacta(
	context.Context,
	SolicitudConsultaInternaGobernadaFuenteAutoridad,
) (ResultadoConsultaInternaFuenteAutoridad, error) {
	return ResultadoConsultaInternaFuenteAutoridad{}, nil
}

var _ GeneradorReferenciasFuentesAutoridad = generadorReferenciasAutoridadContrato{}
var _ ConsultaInternaGobernadaFuentesAutoridad = consultaInternaAutoridadContrato{}
