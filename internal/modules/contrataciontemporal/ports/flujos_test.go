package ports

import (
	"errors"
	"testing"
	"time"
)

func TestDestinoAsignacionResueltoValidaCoordenadasYVigencia(t *testing.T) {
	instante := time.Date(2026, 7, 24, 8, 30, 0, 0, time.UTC)
	solicitud := solicitudDestinoAsignacionPrueba(instante)
	destino := destinoAsignacionPrueba(solicitud)

	if err := destino.ValidarPara(solicitud, instante.Add(time.Minute)); err != nil {
		t.Fatalf("destino válido rechazado: %v", err)
	}
}

func TestDestinoAsignacionResueltoRechazaReutilizacionYCaducidad(t *testing.T) {
	instante := time.Date(2026, 7, 24, 8, 30, 0, 0, time.UTC)
	solicitud := solicitudDestinoAsignacionPrueba(instante)
	base := destinoAsignacionPrueba(solicitud)

	casos := []struct {
		nombre string
		mutar  func(*DestinoAsignacionResuelto)
		uso    time.Time
	}{
		{
			nombre: "otra organización",
			mutar: func(d *DestinoAsignacionResuelto) {
				d.OrganizacionRef = "organizacion:otra:0123456789abcdef"
			},
			uso: instante,
		},
		{
			nombre: "otro expediente",
			mutar: func(d *DestinoAsignacionResuelto) {
				d.ExpedienteRef = "expediente:otro:0123456789abcdef"
			},
			uso: instante,
		},
		{
			nombre: "otra versión",
			mutar: func(d *DestinoAsignacionResuelto) {
				d.VersionExpediente++
			},
			uso: instante,
		},
		{
			nombre: "otro actor",
			mutar: func(d *DestinoAsignacionResuelto) {
				d.ActorRef = "persona:otra:0123456789abcdef"
			},
			uso: instante,
		},
		{
			nombre: "otra unidad",
			mutar: func(d *DestinoAsignacionResuelto) {
				d.UnidadRef = "unidad:otra:0123456789abcdef"
			},
			uso: instante,
		},
		{
			nombre: "otro responsable",
			mutar: func(d *DestinoAsignacionResuelto) {
				d.ResponsableRef = "persona:otra:0123456789abcdef"
			},
			uso: instante,
		},
		{
			nombre: "evidencia sin huella",
			mutar: func(d *DestinoAsignacionResuelto) {
				d.EvidenciaHuellaSHA256 = ""
			},
			uso: instante,
		},
		{
			nombre: "caducado",
			mutar:  func(*DestinoAsignacionResuelto) {},
			uso:    base.ValidoHasta.Add(time.Microsecond),
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			destino := base
			caso.mutar(&destino)
			err := destino.ValidarPara(solicitud, caso.uso)
			if !errors.Is(err, ErrDestinoAsignacionNoDisponible) {
				t.Fatalf("se esperaba rechazo cerrado, recibido %v", err)
			}
		})
	}
}

func TestSolicitudResolverDestinoAsignacionRechazaVersionNoIncrementable(t *testing.T) {
	instante := time.Date(2026, 7, 24, 8, 30, 0, 0, time.UTC)
	solicitud := solicitudDestinoAsignacionPrueba(instante)
	solicitud.VersionExpediente = MaximoEnteroSeguroOperacionAnalisis

	if err := solicitud.Validar(); !errors.Is(err, ErrDestinoAsignacionNoDisponible) {
		t.Fatalf("se esperaba versión rechazada, recibido %v", err)
	}
}

func solicitudDestinoAsignacionPrueba(
	instante time.Time,
) SolicitudResolverDestinoAsignacion {
	return SolicitudResolverDestinoAsignacion{
		OrganizacionRef:   "organizacion:dipgra:0123456789abcdef",
		ExpedienteRef:     "expediente:contratacion:0123456789abcdef",
		VersionExpediente: 4,
		ActorRef:          "persona:tecnica:0123456789abcdef",
		UnidadRef:         "unidad:seleccion:0123456789abcdef",
		ResponsableRef:    "persona:responsable:0123456789abcdef",
		Instante:          instante,
	}
}

func destinoAsignacionPrueba(
	solicitud SolicitudResolverDestinoAsignacion,
) DestinoAsignacionResuelto {
	return DestinoAsignacionResuelto{
		OrganizacionRef:        solicitud.OrganizacionRef,
		ExpedienteRef:          solicitud.ExpedienteRef,
		VersionExpediente:      solicitud.VersionExpediente,
		ActorRef:               solicitud.ActorRef,
		UnidadRef:              solicitud.UnidadRef,
		ResponsableRef:         solicitud.ResponsableRef,
		DefinicionRef:          "catalogo:organizacion:0123456789abcdef",
		DefinicionVersion:      9,
		DefinicionHuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		EvidenciaRef:           "evidencia:destino:0123456789abcdef",
		EvidenciaHuellaSHA256:  "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		EvaluadoEn:             solicitud.Instante,
		ValidoHasta:            solicitud.Instante.Add(5 * time.Minute),
	}
}
