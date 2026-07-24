package ports

import (
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
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

func TestPoliticaAsignacionValidaAltaYReasignacion(t *testing.T) {
	instante := time.Date(2026, 7, 24, 9, 15, 0, 0, time.UTC)
	alta := solicitudPoliticaAsignacionPrueba(
		OperacionRegistrarAsignacion,
		instante,
	)
	if err := politicaAsignacionPrueba(alta).ValidarPara(
		alta,
		instante.Add(time.Minute),
	); err != nil {
		t.Fatalf("política de alta válida rechazada: %v", err)
	}

	reasignacion := solicitudPoliticaAsignacionPrueba(
		OperacionRegistrarReasignacion,
		instante,
	)
	if err := politicaAsignacionPrueba(reasignacion).ValidarPara(
		reasignacion,
		instante.Add(time.Minute),
	); err != nil {
		t.Fatalf("política de reasignación válida rechazada: %v", err)
	}
}

func TestPoliticaAsignacionRechazaDecisionNoLigada(t *testing.T) {
	instante := time.Date(2026, 7, 24, 9, 15, 0, 0, time.UTC)
	solicitud := solicitudPoliticaAsignacionPrueba(
		OperacionRegistrarReasignacion,
		instante,
	)
	base := politicaAsignacionPrueba(solicitud)

	casos := []struct {
		nombre string
		mutar  func(*PoliticaAsignacion)
		uso    time.Time
	}{
		{
			nombre: "otra evidencia",
			mutar: func(p *PoliticaAsignacion) {
				p.DestinoEvidenciaRef = "evidencia:otra:0123456789abcdef"
			},
			uso: instante,
		},
		{
			nombre: "otra versión",
			mutar: func(p *PoliticaAsignacion) {
				p.VersionExpediente++
			},
			uso: instante,
		},
		{
			nombre: "acción de alta",
			mutar: func(p *PoliticaAsignacion) {
				p.Accion = domain.ClaveCatalogo(AccionRegistrarAsignacion)
			},
			uso: instante,
		},
		{
			nombre: "motivo ajeno",
			mutar: func(p *PoliticaAsignacion) {
				p.MotivoReasignacion.ReferenciaCatalogo.EntradaClave =
					"destino_incorrecto"
			},
			uso: instante,
		},
		{
			nombre: "política caducada",
			mutar:  func(*PoliticaAsignacion) {},
			uso:    base.ValidaHasta.Add(time.Microsecond),
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			politica := base
			caso.mutar(&politica)
			err := politica.ValidarPara(solicitud, caso.uso)
			if !errors.Is(err, ErrPoliticaAsignacionNoDisponible) {
				t.Fatalf("se esperaba rechazo cerrado, recibido %v", err)
			}
		})
	}
}

func TestPoliticaAsignacionAplicaSegregacionConfigurada(t *testing.T) {
	instante := time.Date(2026, 7, 24, 9, 15, 0, 0, time.UTC)
	solicitud := solicitudPoliticaAsignacionPrueba(
		OperacionRegistrarAsignacion,
		instante,
	)
	solicitud.ActorRef = solicitud.Destino.ResponsableRef
	solicitud.Destino.ActorRef = solicitud.ActorRef
	politica := politicaAsignacionPrueba(solicitud)
	politica.ExigeActorDistintoResponsable = true

	err := politica.ValidarPara(solicitud, instante)
	if !errors.Is(err, ErrPoliticaAsignacionNoDisponible) {
		t.Fatalf("se esperaba segregación denegada, recibido %v", err)
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

func solicitudPoliticaAsignacionPrueba(
	operacion TipoOperacionAsignacion,
	instante time.Time,
) SolicitudResolverPoliticaAsignacion {
	destinoSolicitud := solicitudDestinoAsignacionPrueba(instante)
	solicitud := SolicitudResolverPoliticaAsignacion{
		Operacion:         operacion,
		OrganizacionRef:   destinoSolicitud.OrganizacionRef,
		ExpedienteRef:     destinoSolicitud.ExpedienteRef,
		VersionExpediente: destinoSolicitud.VersionExpediente,
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: "flujo:contratacion:0123456789abcdef",
			Version:       4,
			HuellaSHA256:  "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		FasePrevia:   "asignacion_unidad",
		EstadoPrevio: domain.EstadoEnCurso,
		ActorRef:     destinoSolicitud.ActorRef,
		PerfilRef:    "perfil:tecnico:0123456789abcdef",
		Destino:      destinoAsignacionPrueba(destinoSolicitud),
		Instante:     instante,
	}
	if operacion == OperacionRegistrarReasignacion {
		solicitud.UnidadAnteriorRef =
			"unidad:contratacion:0123456789abcdef"
		solicitud.ResponsableAnteriorRef =
			"persona:anterior:0123456789abcdef"
		solicitud.MotivoReasignacionClave = "necesidad_servicio"
	}
	return solicitud
}

func politicaAsignacionPrueba(
	solicitud SolicitudResolverPoliticaAsignacion,
) PoliticaAsignacion {
	politica := PoliticaAsignacion{
		Operacion:                    solicitud.Operacion,
		OrganizacionRef:              solicitud.OrganizacionRef,
		ExpedienteRef:                solicitud.ExpedienteRef,
		VersionExpediente:            solicitud.VersionExpediente,
		ActorRef:                     solicitud.ActorRef,
		PerfilRef:                    solicitud.PerfilRef,
		DestinoEvidenciaRef:          solicitud.Destino.EvidenciaRef,
		DestinoEvidenciaHuellaSHA256: solicitud.Destino.EvidenciaHuellaSHA256,
		DefinicionRef:                "politica:asignacion:0123456789abcdef",
		DefinicionVersion:            3,
		DefinicionHuellaSHA256:       "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		Accion: domain.ClaveCatalogo(
			AccionRegistrarAsignacion,
		),
		Finalidad:          "gestionar_contratacion_temporal",
		UnidadEjecutoraRef: "unidad:rrhh:0123456789abcdef",
		MotivoAutorizacion: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID:           "catalogo_motivos_asignacion",
			CatalogoVersion:      7,
			CatalogoHuellaSHA256: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			EntradaClave:         "gestion_asignacion",
		},
		EvaluadaEn:  solicitud.Instante,
		ValidaHasta: solicitud.Instante.Add(5 * time.Minute),
	}
	if solicitud.Operacion == OperacionRegistrarReasignacion {
		politica.Accion = domain.ClaveCatalogo(
			AccionRegistrarReasignacion,
		)
		politica.MotivoReasignacion = MotivoReasignacionGobernado{
			ReferenciaCatalogo: dominiovec.ReferenciaEntradaCatalogo{
				CatalogoID:           "catalogo_motivos_reasignacion",
				CatalogoVersion:      2,
				CatalogoHuellaSHA256: "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
				EntradaClave:         string(solicitud.MotivoReasignacionClave),
			},
			ClaveMensajeI18N: "contratacion_temporal.asignacion.motivo." +
				solicitud.MotivoReasignacionClave,
		}
	}
	return politica
}
