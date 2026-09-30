package bootstrap

import (
	"testing"

	contrataciontemporal "vec-diputacion-granada/internal/modules/contrataciontemporal"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var faseFiscalizacionPrueba = fasesOperacionPredeterminadasCT()[operacionFaseFiscalizacionCT]

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
		{"modificacion devuelta a fiscalizacion", 8, domain.FaseFiscalizacion, domain.EstadoEnCurso, true},
		{"fiscalizacion sin version posterior", 5, domain.FaseFiscalizacion, domain.EstadoEnCurso, false},
		{"fase ajena", 7, domain.FaseNombramiento, domain.EstadoEnCurso, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if recibido := origenFiscalizacionContratacionTemporalDesarrolloValido(fasesOperacionPredeterminadasCT()[operacionFaseFiscalizacionCT], caso.version, caso.fase, caso.estado); recibido != caso.admite {
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
	if !solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(refiscalizacion, faseFiscalizacionPrueba) {
		t.Fatal("la refiscalizacion ligada debe conservar la autorizacion nominal")
	}

	sinRecibo := refiscalizacion
	sinRecibo.Recurso.Atributos = map[string]string{"retorno_previo_ref": "retorno:fiscalizacion:001"}
	if solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(sinRecibo, faseFiscalizacionPrueba) {
		t.Fatal("la refiscalizacion sin recibo de subsanacion no debe autorizarse")
	}

	inicialConRetorno := datosAutorizacionFiscalizacionDesarrollo(
		domain.FaseInformeJuridico, domain.EstadoEnCurso,
		map[string]string{"retorno_previo_ref": "retorno:fiscalizacion:001"},
	)
	if solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(inicialConRetorno, faseFiscalizacionPrueba) {
		t.Fatal("la fiscalizacion inicial no admite un retorno previo")
	}

	modificacion := datosAutorizacionFiscalizacionDesarrollo(
		domain.FaseFiscalizacion, domain.EstadoEnCurso,
		map[string]string{"modificacion_recibo_ref": "recibo:modificacion:001"},
	)
	if !solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(modificacion, faseFiscalizacionPrueba) {
		t.Fatal("la fiscalizacion de una modificacion ligada a su recibo debe autorizarse")
	}
	sinModificacion := datosAutorizacionFiscalizacionDesarrollo(domain.FaseFiscalizacion, domain.EstadoEnCurso, map[string]string{})
	if solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(sinModificacion, faseFiscalizacionPrueba) {
		t.Fatal("la fiscalizacion en fase de fiscalizacion exige el recibo de la modificacion")
	}
	mezclada := datosAutorizacionFiscalizacionDesarrollo(
		domain.FaseSubsanacionUnidad, domain.EstadoIncidencia,
		map[string]string{"retorno_previo_ref": "retorno:fiscalizacion:001", "subsanacion_recibo_ref": "recibo:subsanacion:001", "modificacion_recibo_ref": "recibo:modificacion:001"},
	)
	if solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(mezclada, faseFiscalizacionPrueba) {
		t.Fatal("una refiscalizacion no puede citar una modificacion")
	}
	// Con el expediente en los ámbitos, o con un catálogo que no admite la
	// fase, se deniega.
	conExpediente := datosAutorizacionFiscalizacionDesarrollo(domain.FaseInformeJuridico, domain.EstadoEnCurso, map[string]string{})
	conExpediente.Recurso.Ambitos["expediente_ref"] = expedienteContratacionTemporalDesarrolloRef
	if solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(conExpediente, faseFiscalizacionPrueba) {
		t.Fatal("aceptada con el expediente en los ámbitos")
	}
	soloInicial := faseOperacionDePares(faseEstadoOperacionCT{domain.FaseInformeJuridico, domain.EstadoEnCurso})
	if solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(modificacion, soloInicial) ||
		origenFiscalizacionContratacionTemporalDesarrolloValido(soloInicial, 8, domain.FaseFiscalizacion, domain.EstadoEnCurso) {
		t.Fatal("se admitió una fase que el catálogo no declara")
	}
	inicial := datosAutorizacionFiscalizacionDesarrollo(domain.FaseInformeJuridico, domain.EstadoEnCurso, map[string]string{})
	if !solicitudAutorizacionFiscalizacionContratacionTemporalDesarrolloValida(inicial, soloInicial) {
		t.Fatal("la fiscalización inicial debe admitirse con su fase en el catálogo")
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
				"fase_previa":      string(fase),
				"estado_previo":    string(estado),
			},
			Atributos: atributos,
		},
	}
}
