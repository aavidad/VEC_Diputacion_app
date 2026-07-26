package importacionconvoca

import (
	"errors"
	"testing"
	"time"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

func TestContratosGestionDurableRechazanReferenciasYLimites(t *testing.T) {
	conciliacion := SolicitudConciliacion{
		ImportacionRef:         "importacion:convoca:" + repetirHex("a"),
		ConciliacionRef:        "conciliacion:convoca:prueba",
		RegistroCorporativoRef: "registro:corporativo:opaco:prueba",
		Resultado:              ResultadoConciliadoConfirmado,
		ActorRef:               "actor:rrhh:prueba", MotivoCodigo: "conciliacion_confirmada",
	}
	if err := conciliacion.Validar(); err != nil {
		t.Fatalf("conciliacion valida rechazada: %v", err)
	}
	conciliacion.RegistroCorporativoRef = "Nombre Apellidos"
	if !errors.Is(conciliacion.Validar(), ErrGestionDurableInvalida) {
		t.Fatal("referencia personal libre aceptada")
	}
	expurgo := SolicitudExpurgoStaging{
		EjecucionRef: "expurgo:convoca:prueba", ActorRef: "actor:archivo:prueba",
		PoliticaRef: "politica:retencion:convoca:prueba", PoliticaVersion: 1,
		Limite: MaximoLotesPorExpurgo + 1,
	}
	if !errors.Is(expurgo.Validar(), ErrGestionDurableInvalida) {
		t.Fatal("limite de expurgo excesivo aceptado")
	}
	expurgo.Limite = 1
	expurgo.PoliticaVersion = maximaVersionPostgreSQL + 1
	if !errors.Is(expurgo.Validar(), ErrGestionDurableInvalida) {
		t.Fatal("version no representable por PostgreSQL aceptada")
	}
}

func TestEstadoImportacionAdmiteActaTrasExpurgoSinStaging(t *testing.T) {
	acta := actaGestionPrueba()
	estado := EstadoImportacion{
		Acta: acta, EstadoConciliacion: EstadoConciliacionConfirmada,
		EstadoStaging:            EstadoStagingExpurgado,
		PoliticaRetencionRef:     "politica:retencion:convoca:prueba",
		PoliticaRetencionVersion: 1,
		ConservarStagingHasta:    acta.RegistradaEn.Add(24 * time.Hour),
		Version:                  3,
	}
	if err := estado.Validar(); err != nil {
		t.Fatalf("estado expurgado invalido: %v", err)
	}
	estado.Version = maximaVersionPostgreSQL + 1
	if !errors.Is(estado.Validar(), ErrGestionDurableInvalida) {
		t.Fatal("version de estado no representable por PostgreSQL aceptada")
	}
}

func actaGestionPrueba() dominio.ActaImportacion {
	huella := repetirHex("a")
	return dominio.ActaImportacion{
		ActaRef:             "acta:importacion-convoca:" + huella,
		ImportacionRef:      "importacion:convoca:" + huella,
		HuellaFicheroSHA256: huella, NombreFichero: "sintetico.xls",
		FicheroCustodiadoRef: "almacen:objeto:convoca:" + huella,
		ActorRef:             "actor:rrhh:prueba",
		RegistradaEn:         time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC),
		Esquema:              dominio.EsquemaResumenPersona,
		FilasLeidas:          1, FilasAceptadas: 1,
		Procedencia: dominio.NuevaProcedenciaNoAutoritativa(),
	}
}

func repetirHex(valor string) string {
	resultado := ""
	for len(resultado) < 64 {
		resultado += valor
	}
	return resultado
}
