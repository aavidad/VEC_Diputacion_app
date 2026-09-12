package bootstrap

import (
	"testing"

	contrataciontemporal "vec-diputacion-granada/internal/modules/contrataciontemporal"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestOrigenFiscalizacionContratacionTemporalDesarrollo(t *testing.T) {
	casos := []struct {
		nombre  string
		version uint64
		fase    domain.ClaveFase
		estado  domain.EstadoOperativo
		admite  bool
	}{
		{"inicial v5 exacta", 5, domain.FaseInformeJuridico, domain.EstadoEnCurso, true},
		{"inicial no sustituye v5", 6, domain.FaseInformeJuridico, domain.EstadoEnCurso, false},
		{"refiscalizacion con version real", 7, domain.FaseSubsanacionUnidad, domain.EstadoIncidencia, true},
		{"fase de subsanacion sin incidencia", 7, domain.FaseSubsanacionUnidad, domain.EstadoEnCurso, false},
		{"fase ajena", 7, domain.FaseFiscalizacion, domain.EstadoEnCurso, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if recibido := origenFiscalizacionContratacionTemporalDesarrolloValido(caso.version, caso.fase, caso.estado); recibido != caso.admite {
				t.Fatalf("admite=%v; quiere %v", recibido, caso.admite)
			}
		})
	}
}

func TestSolicitudAutorizacionFiscalizacionExigeVinculoDeSubsanacion(t *testing.T) {
	refiscalizacion := datosAutorizacionFiscalizacionDesarrollo(
		domain.FaseSubsanacionUnidad, domain.EstadoIncidencia,
		map[string]string{
			"retorno_previo_ref":     "retorno:fiscalizacion:001",
			"subsanacion_recibo_ref": "recibo:subsanacion:001",
		},
	)
	if !solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(refiscalizacion) {
		t.Fatal("la refiscalizacion ligada debe conservar la autorizacion nominal")
	}

	sinRecibo := refiscalizacion
	sinRecibo.Recurso.Atributos = map[string]string{"retorno_previo_ref": "retorno:fiscalizacion:001"}
	if solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(sinRecibo) {
		t.Fatal("la refiscalizacion sin recibo de subsanacion no debe autorizarse")
	}

	inicialConRetorno := datosAutorizacionFiscalizacionDesarrollo(
		domain.FaseInformeJuridico, domain.EstadoEnCurso,
		map[string]string{"retorno_previo_ref": "retorno:fiscalizacion:001"},
	)
	if solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(inicialConRetorno) {
		t.Fatal("la fiscalizacion inicial no admite un retorno previo")
	}
}

func datosAutorizacionFiscalizacionDesarrollo(
	fase domain.ClaveFase,
	estado domain.EstadoOperativo,
	atributos map[string]string,
) dominiovec.DatosSolicitudAutorizacionLigadaV3 {
	return dominiovec.DatosSolicitudAutorizacionLigadaV3{
		Accion:           contrataciontemporal.PermisoRegistrarFiscalizacion,
		ReferenciaMotivo: referenciaMotivoAutorizacionFiscalizacionDesarrollo(),
		Finalidad:        finalidadFiscalizacionContratacionTemporalDesarrollo,
		Recurso: dominiovec.RecursoAutorizable{
			Referencia: expedienteContratacionTemporalDesarrolloRef,
			ModuloID:   ports.ModuloContratacion,
			Tipo:       tipoRecursoFiscalizacionContratacionTemporalDesarrollo,
			Ambitos: map[string]string{
				"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
				"expediente_ref":   expedienteContratacionTemporalDesarrolloRef,
				"fase_previa":      string(fase),
				"estado_previo":    string(estado),
			},
			Atributos: atributos,
		},
	}
}
