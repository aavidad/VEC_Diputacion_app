package ports

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

const clavePropuestaFormalizacionPrueba = "628f47a6-5d2b-4c10-aa11-1234567890ab"

func TestSolicitudPropuestaFormalizacionNormalizaAnexosSinAlias(t *testing.T) {
	a := anexoPropuestaFormalizacionPrueba("a", "1", 1024)
	b := anexoPropuestaFormalizacionPrueba("b", "2", 2048)
	solicitud := solicitudPropuestaFormalizacionPrueba()
	solicitud.Anexos = []AnexoPropuestaFormalizacion{b, a, a}

	normalizada, err := solicitud.Normalizar()
	if err != nil {
		t.Fatalf("normalizar: %v", err)
	}
	if len(normalizada.Anexos) != 2 || normalizada.Anexos[0] != a ||
		normalizada.Anexos[1] != b || normalizada.Validar() != nil {
		t.Fatalf("normalizacion inesperada: %+v", normalizada.Anexos)
	}
	if solicitud.Validar() == nil {
		t.Fatal("forma no canonica admitida directamente por el puerto")
	}

	solicitud.Anexos[0].DocumentoRef = "anexo:mutado-origen"
	if normalizada.Anexos[1] != b {
		t.Fatal("la normalizacion conserva alias con la entrada")
	}
	clon := normalizada.Clonar()
	clon.Anexos[0].DocumentoRef = "anexo:mutado-clon"
	if normalizada.Anexos[0] != a {
		t.Fatal("el clon conserva alias con la solicitud canonica")
	}
}

func TestSolicitudPropuestaFormalizacionRechazaConflictoDeAnexo(t *testing.T) {
	solicitud := solicitudPropuestaFormalizacionPrueba()
	primero := anexoPropuestaFormalizacionPrueba("a", "1", 1024)
	segundo := primero
	segundo.HuellaSHA256 = strings.Repeat("f", 64)
	solicitud.Anexos = []AnexoPropuestaFormalizacion{primero, segundo}
	if _, err := solicitud.Normalizar(); err == nil {
		t.Fatal("misma referencia/version con otro compromiso aceptada")
	}
}

func TestSolicitudPropuestaFormalizacionAplicaLimitesAntesDelPuerto(t *testing.T) {
	t.Run("cardinalidad", func(t *testing.T) {
		solicitud := solicitudPropuestaFormalizacionPrueba()
		solicitud.Anexos = make([]AnexoPropuestaFormalizacion, MaximoAnexosPropuestaFormalizacion+1)
		if _, err := solicitud.Normalizar(); err == nil {
			t.Fatal("cardinalidad excesiva aceptada")
		}
	})

	t.Run("bytes_exactos", func(t *testing.T) {
		solicitud := solicitudPropuestaFormalizacionPrueba()
		solicitud.Anexos = []AnexoPropuestaFormalizacion{
			anexoPropuestaFormalizacionPrueba("limite", "c", MaximoBytesAnexosPropuestaFormalizacion),
		}
		if _, err := solicitud.Normalizar(); err != nil {
			t.Fatalf("limite exacto rechazado: %v", err)
		}
	})

	t.Run("suma_bytes", func(t *testing.T) {
		solicitud := solicitudPropuestaFormalizacionPrueba()
		solicitud.Anexos = []AnexoPropuestaFormalizacion{
			anexoPropuestaFormalizacionPrueba("limite", "c", MaximoBytesAnexosPropuestaFormalizacion),
			anexoPropuestaFormalizacionPrueba("extra", "d", 1),
		}
		if _, err := solicitud.Normalizar(); err == nil {
			t.Fatal("suma de bytes excesiva aceptada")
		}
	})

	t.Run("tamano_cero", func(t *testing.T) {
		solicitud := solicitudPropuestaFormalizacionPrueba()
		solicitud.Anexos = []AnexoPropuestaFormalizacion{
			anexoPropuestaFormalizacionPrueba("vacio", "e", 0),
		}
		if _, err := solicitud.Normalizar(); err == nil {
			t.Fatal("anexo sin tamano aceptado")
		}
	})
}

func TestResultadoPropuestaFormalizacionLigaCommitLocalCompleto(t *testing.T) {
	solicitud := solicitudPropuestaFormalizacionPrueba()
	resultado := resultadoPropuestaFormalizacionPrueba(solicitud)
	if err := resultado.ValidarPara(solicitud); err != nil {
		t.Fatalf("resultado valido rechazado: %v", err)
	}
	replay := resultado.Clonar()
	replay.Estado = ResultadoPropuestaFormalizacionReplay
	if err := replay.ValidarPara(solicitud); err != nil || !replay.EsReplayConfirmado() {
		t.Fatalf("replay valido rechazado: replay=%v err=%v", replay.EsReplayConfirmado(), err)
	}

	mutaciones := []struct {
		nombre string
		mutar  func(*ResultadoPropuestaFormalizacion)
	}{
		{"organizacion", func(r *ResultadoPropuestaFormalizacion) { r.Solicitud.OrganizacionRef = "organizacion:otra" }},
		{"expediente", func(r *ResultadoPropuestaFormalizacion) { r.Solicitud.ExpedienteRef = "expediente:otro" }},
		{"llamamiento", func(r *ResultadoPropuestaFormalizacion) { r.Solicitud.LlamamientoRef = "llamamiento:otro" }},
		{"resolucion", func(r *ResultadoPropuestaFormalizacion) {
			r.Solicitud.ResolucionLlamamientoAceptadaRef = "resolucion:otra"
		}},
		{"recibo_aceptacion", func(r *ResultadoPropuestaFormalizacion) { r.Solicitud.ReciboResolucionAceptadaRef = "recibo:otro" }},
		{"idempotencia", func(r *ResultadoPropuestaFormalizacion) {
			r.Solicitud.ClaveIdempotencia = "728f47a6-5d2b-4c10-ba11-1234567890ab"
		}},
		{"occ", func(r *ResultadoPropuestaFormalizacion) { r.Solicitud.VersionEsperada++ }},
		{"tipo", func(r *ResultadoPropuestaFormalizacion) { r.Solicitud.TipoFormalizacion.Version++ }},
		{"plantilla", func(r *ResultadoPropuestaFormalizacion) { r.Solicitud.Plantilla.HuellaSHA256 = strings.Repeat("f", 64) }},
		{"anexo", func(r *ResultadoPropuestaFormalizacion) { r.Solicitud.Anexos[0].TamanoBytes++ }},
		{"politica", func(r *ResultadoPropuestaFormalizacion) { r.Solicitud.PoliticaFirma.Version++ }},
		{"plan", func(r *ResultadoPropuestaFormalizacion) { r.Solicitud.PlanFirma.Referencia = "plan:firma-otro" }},
		{"propuesta", func(r *ResultadoPropuestaFormalizacion) { r.PropuestaRef = "" }},
		{"recibo_local", func(r *ResultadoPropuestaFormalizacion) { r.ReciboLocalRef = "" }},
		{"auditoria", func(r *ResultadoPropuestaFormalizacion) { r.AuditoriaRef = "" }},
		{"version_resultante", func(r *ResultadoPropuestaFormalizacion) { r.VersionResultante++ }},
		{"instante", func(r *ResultadoPropuestaFormalizacion) { r.ConfirmadaEn = time.Time{} }},
		{"estado", func(r *ResultadoPropuestaFormalizacion) { r.Estado = "firmada" }},
	}
	for _, caso := range mutaciones {
		t.Run(caso.nombre, func(t *testing.T) {
			alterado := resultado.Clonar()
			caso.mutar(&alterado)
			if alterado.ValidarPara(solicitud) == nil {
				t.Fatal("resultado con ligadura mutada aceptado")
			}
		})
	}
}

func TestContratoPropuestaFormalizacionMinimizaPIIYEfectosExternos(t *testing.T) {
	modelos := []reflect.Type{
		reflect.TypeOf(SnapshotGobernadoFormalizacion{}),
		reflect.TypeOf(AnexoPropuestaFormalizacion{}),
		reflect.TypeOf(SolicitudPropuestaFormalizacion{}),
		reflect.TypeOf(ResultadoPropuestaFormalizacion{}),
	}
	for _, modelo := range modelos {
		for indice := 0; indice < modelo.NumField(); indice++ {
			nombre := strings.ToLower(modelo.Field(indice).Name)
			for _, prohibido := range []string{
				"dni", "nie", "nombre", "correo", "telefono", "direccion", "persona", "firmante", "rol",
			} {
				if strings.Contains(nombre, prohibido) {
					t.Fatalf("%s expone campo personal o funcional %s", modelo.Name(), nombre)
				}
			}
		}
	}

	resultado := reflect.TypeOf(ResultadoPropuestaFormalizacion{})
	for _, campo := range []string{
		"DocumentoGeneradoRef", "FirmaRef", "CustodiaRef", "RegistroRef",
		"DescargaRef", "IntencionDocumental", "EstadoExterno",
	} {
		if _, existe := resultado.FieldByName(campo); existe {
			t.Fatalf("el resultado afirma efecto fuera del corte: %s", campo)
		}
	}
}

func TestSolicitudPropuestaFormalizacionRechazaMutacionesDeEntrada(t *testing.T) {
	base := solicitudPropuestaFormalizacionPrueba()
	casos := []struct {
		nombre string
		mutar  func(*SolicitudPropuestaFormalizacion)
	}{
		{"uuid", func(s *SolicitudPropuestaFormalizacion) { s.ClaveIdempotencia = "no-uuid" }},
		{"resolucion", func(s *SolicitudPropuestaFormalizacion) { s.ResolucionLlamamientoAceptadaRef = "" }},
		{"recibo", func(s *SolicitudPropuestaFormalizacion) { s.ReciboResolucionAceptadaRef = "" }},
		{"version", func(s *SolicitudPropuestaFormalizacion) { s.VersionEsperada = 0 }},
		{"tipo", func(s *SolicitudPropuestaFormalizacion) { s.TipoFormalizacion.Version = 0 }},
		{"plantilla", func(s *SolicitudPropuestaFormalizacion) { s.Plantilla.HuellaSHA256 = strings.Repeat("0", 64) }},
		{"politica", func(s *SolicitudPropuestaFormalizacion) { s.PoliticaFirma.Referencia = "" }},
		{"plan", func(s *SolicitudPropuestaFormalizacion) { s.PlanFirma.Version = 0 }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			alterada := base.Clonar()
			caso.mutar(&alterada)
			if _, err := alterada.Normalizar(); err == nil {
				t.Fatal("entrada mutada aceptada")
			}
		})
	}
}

func solicitudPropuestaFormalizacionPrueba() SolicitudPropuestaFormalizacion {
	return SolicitudPropuestaFormalizacion{
		ClaveIdempotencia:                clavePropuestaFormalizacionPrueba,
		OrganizacionRef:                  "organizacion:ct-formalizacion",
		ExpedienteRef:                    "expediente:ct-formalizacion",
		LlamamientoRef:                   "llamamiento:ct-formalizacion",
		ResolucionLlamamientoAceptadaRef: "resolucion:llamamiento-aceptado",
		ReciboResolucionAceptadaRef:      "recibo:llamamiento-aceptado",
		VersionEsperada:                  13,
		TipoFormalizacion:                snapshotFormalizacionPrueba("tipo:formalizacion", "1"),
		Plantilla:                        snapshotFormalizacionPrueba("plantilla:formalizacion", "2"),
		Anexos: []AnexoPropuestaFormalizacion{
			anexoPropuestaFormalizacionPrueba("a", "3", 1024),
			anexoPropuestaFormalizacionPrueba("b", "4", 2048),
		},
		PoliticaFirma: snapshotFormalizacionPrueba("politica:firma", "5"),
		PlanFirma:     snapshotFormalizacionPrueba("plan:firma", "6"),
	}
}

func snapshotFormalizacionPrueba(referencia, digito string) SnapshotGobernadoFormalizacion {
	return SnapshotGobernadoFormalizacion{
		Referencia: referencia, Version: 7, HuellaSHA256: strings.Repeat(digito, 64),
	}
}

func anexoPropuestaFormalizacionPrueba(
	sufijo string,
	digito string,
	tamano uint64,
) AnexoPropuestaFormalizacion {
	return AnexoPropuestaFormalizacion{
		DocumentoRef: fmt.Sprintf("anexo:%s", sufijo), Version: 3,
		HuellaSHA256: strings.Repeat(digito, 64), TamanoBytes: tamano,
	}
}

func resultadoPropuestaFormalizacionPrueba(
	solicitud SolicitudPropuestaFormalizacion,
) ResultadoPropuestaFormalizacion {
	return ResultadoPropuestaFormalizacion{
		Solicitud: solicitud.Clonar(), PropuestaRef: "propuesta:formalizacion-local",
		ReciboLocalRef: "recibo:formalizacion-local", AuditoriaRef: "auditoria:formalizacion-local",
		VersionResultante: solicitud.VersionEsperada + 1,
		ConfirmadaEn:      time.Date(2026, 8, 31, 14, 0, 0, 0, time.UTC),
		Estado:            ResultadoPropuestaFormalizacionConfirmado,
	}
}
