package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestOperacionRenunciaJustificadaRRHHConservaJustificanteYRecibo(t *testing.T) {
	reglas := reglasTransicionesPrueba{destinos: map[string][]string{domain.SituacionRenuncia: {domain.SituacionDisponible, domain.SituacionExcluido}}}
	servicio, repo, ahora := servicioDesdeRenunciaPrueba(t, reglas)
	q := ports.SolicitudOperacionSituacion{
		SolicitudCambiarSituacionParticipacion: solicitudSituacionPrueba(t, ahora),
		Operacion:                              domain.OperacionReactivar,
		Justificante: domain.JustificanteOperacionSituacion{
			Tipo: domain.JustificanteSolicitudCandidato, Referencia: "justificante:renuncia", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		Validador: "persona:rrhh",
	}
	q.Destino = domain.SituacionDisponible
	q.Motivo = "Justificante de renuncia validado por RRHH"
	primero, err := servicio.Operar(context.Background(), q)
	if err != nil || primero.Situacion != domain.SituacionDisponible || primero.ReciboRef == "" || repo.escrituras != 1 {
		t.Fatalf("reactivación: resultado=%+v error=%v escrituras=%d", primero, err, repo.escrituras)
	}
	if repo.operacion.Justificante != q.Justificante || repo.operacion.Validador != q.Validador || repo.operacion.Actor != q.ResultadoContexto.Contexto.PersonaRef {
		t.Fatal("la operación no conserva su justificante, validador y actor")
	}
	// La situación vigente ya refleja el efecto: recuperar no añade otra fila.
	repo.vigente = primero.SituacionParticipacion
	segundo, err := servicio.Operar(context.Background(), q)
	if err != nil || !segundo.Reutilizada || segundo.ReciboRef != primero.ReciboRef || !segundo.Desde.Equal(primero.Desde) || repo.escrituras != 1 {
		t.Fatalf("recuperación: resultado=%+v error=%v escrituras=%d", segundo, err, repo.escrituras)
	}
	q.Justificante.Referencia = "justificante:otro"
	if _, err := servicio.Operar(context.Background(), q); !errors.Is(err, ports.ErrClaveOperacionReutilizada) || repo.escrituras != 1 {
		t.Fatalf("otro justificante con la misma clave: error=%v escrituras=%d", err, repo.escrituras)
	}
}

func TestRevisionDocumentalExigeCASYRecuperaTrasAvance(t *testing.T) {
	servicio, repo, ahora := servicioDesdeRenunciaPrueba(t, nil)
	tabla := map[string][]string{}
	for _, origen := range domain.SituacionesParticipacion() {
		tabla[origen] = domain.DestinosSituacionParticipacion(origen)
	}
	tabla[domain.SituacionRenuncia] = []string{domain.SituacionEnRevision, domain.SituacionExcluido}
	tabla[domain.SituacionEnRevision] = []string{domain.SituacionDisponible, domain.SituacionExcluido}
	politica, err := domain.NuevaPoliticaTransicionesSituacion(tabla)
	if err != nil {
		t.Fatal(err)
	}
	servicio.repositorio = &repositorioPoliticaReincorporacionPrueba{repositorioOperacionPrueba: repo, politica: politica}
	q := ports.SolicitudOperacionSituacion{
		SolicitudCambiarSituacionParticipacion: solicitudSituacionPrueba(t, ahora),
		Operacion:                              domain.OperacionRevisar,
		Justificante:                           domain.JustificanteOperacionSituacion{Tipo: domain.JustificanteSolicitudCandidato, Referencia: "documento:renuncia", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		Validador:                              "persona:rrhh",
	}
	q.Destino = domain.SituacionEnRevision
	q.Motivo = "Pendiente de acreditar el fin de la causa"
	if _, err := servicio.Operar(context.Background(), q); !errors.Is(err, domain.ErrOperacionSituacionParticipacionInvalida) || repo.escrituras != 0 {
		t.Fatalf("sin CAS: error=%v escrituras=%d", err, repo.escrituras)
	}
	q.SituacionEsperadaDesde = ahora
	res, err := servicio.Operar(context.Background(), q)
	if err != nil || res.Situacion != domain.SituacionEnRevision || repo.escrituras != 1 {
		t.Fatalf("revisión: resultado=%+v error=%v escrituras=%d", res, err, repo.escrituras)
	}
	repo.vigente = ports.SituacionParticipacion{ParticipacionRef: q.ParticipacionRef, Situacion: domain.SituacionDisponible, Desde: ahora.Add(time.Second)}
	servicio.reloj = func() time.Time { return ahora.Add(2 * time.Second) }
	repetida, err := servicio.Operar(context.Background(), q)
	if err != nil || !repetida.Reutilizada || repetida.ReciboRef != res.ReciboRef || repo.escrituras != 1 {
		t.Fatalf("revisión recuperada tras avance: resultado=%+v error=%v escrituras=%d", repetida, err, repo.escrituras)
	}
	q.Operacion = domain.OperacionRegularizar
	q.Destino = domain.SituacionDisponible
	q.ClaveIdempotencia = "regularizacion-sin-fin"
	if _, err := servicio.Operar(context.Background(), q); !errors.Is(err, domain.ErrOperacionSituacionParticipacionInvalida) {
		t.Fatalf("regularización sin fin: %v", err)
	}
}

func TestRegularizarSolicitudDocumentalEntregaVinculoAlRepositorio(t *testing.T) {
	servicio, repo, ahora := servicioDesdeRenunciaPrueba(t, nil)
	repo.vigente.Situacion = domain.SituacionEnRevision
	q := ports.SolicitudOperacionSituacion{
		SolicitudCambiarSituacionParticipacion: solicitudSituacionPrueba(t, ahora),
		Operacion:                              domain.OperacionRegularizar,
		Justificante: domain.JustificanteOperacionSituacion{Tipo: domain.JustificanteSolicitudCandidato,
			Referencia: "documento:fin-causa", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		Validador: "persona:rrhh", SituacionEsperadaDesde: ahora,
		SolicitudRef:             "solicitud-documental:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		SolicitudVersionEsperada: 1,
		SolicitudContenidoSHA256: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}
	q.Destino = domain.SituacionDisponible
	q.Motivo = "Documento de fin de causa validado"
	fin := ahora.Add(-24 * time.Hour)
	q.CausaFinalizadaEn = &fin
	q.SolicitudContenidoSHA256 = "mal"
	if _, err := servicio.Operar(context.Background(), q); !errors.Is(err, domain.ErrOperacionSituacionParticipacionInvalida) || repo.escrituras != 0 {
		t.Fatalf("contenido inválido: err=%v escrituras=%d", err, repo.escrituras)
	}
	q.SolicitudContenidoSHA256 = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	res, err := servicio.Operar(context.Background(), q)
	atributo, errCanon := huellaRegularizacionDocumental(q, q.ResultadoContexto.Contexto.PersonaRef, res.ReciboRef)
	if err != nil || res.Situacion != domain.SituacionDisponible || repo.comando.SolicitudRef != q.SolicitudRef ||
		repo.comando.SolicitudVersionEsperada != 1 || repo.comando.SolicitudContenidoSHA256 != q.SolicitudContenidoSHA256 ||
		repo.comando.Justificante != q.Justificante || errCanon != nil ||
		!strings.Contains(string(repo.comando.ContextoRecursoCanonico), atributo) {
		t.Fatalf("vínculo documental: resultado=%+v error=%v comando=%+v", res, err, repo.comando)
	}
}

func TestOperacionRenunciaRRHHNoReincorporaSinValidacionNiPoliticaPublicada(t *testing.T) {
	for _, caso := range []string{"sin_justificante", "sin_validador", "exclusion_adoptada"} {
		t.Run(caso, func(t *testing.T) {
			servicio, repo, ahora := servicioDesdeRenunciaPrueba(t, nil)
			q := ports.SolicitudOperacionSituacion{
				SolicitudCambiarSituacionParticipacion: solicitudSituacionPrueba(t, ahora),
				Operacion:                              domain.OperacionReactivar,
				Justificante: domain.JustificanteOperacionSituacion{
					Tipo: domain.JustificanteSolicitudCandidato, Referencia: "justificante:renuncia", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				},
				Validador: "persona:rrhh",
			}
			q.Destino = domain.SituacionDisponible
			esperado := domain.ErrOperacionSituacionParticipacionInvalida
			switch caso {
			case "sin_justificante":
				q.Justificante = domain.JustificanteOperacionSituacion{}
			case "sin_validador":
				q.Validador = ""
			case "exclusion_adoptada":
				repo.vigente.Situacion = domain.SituacionExcluido
				esperado = domain.ErrCambioSituacionParticipacionInvalido
			}
			if _, err := servicio.Operar(context.Background(), q); !errors.Is(err, esperado) || repo.escrituras != 0 {
				t.Fatalf("operación rechazada: error=%v escrituras=%d", err, repo.escrituras)
			}
		})
	}
}

type repositorioPoliticaReincorporacionPrueba struct {
	*repositorioOperacionPrueba
	politica domain.PoliticaTransicionesSituacion
}

func (r *repositorioPoliticaReincorporacionPrueba) PoliticaTransicionesSituacion(context.Context) (ports.PoliticaTransicionesVigente, error) {
	return ports.PoliticaTransicionesVigente{Version: 2, CatalogoRef: "vec.bolsa.reglas:2:b28.transiciones", Politica: r.politica}, nil
}

func TestOperacionRenunciaRRHHReincorporaExclusionConPoliticaPublicadaYJustificante(t *testing.T) {
	servicio, repo, ahora := servicioDesdeRenunciaPrueba(t, nil)
	repo.vigente.Situacion = domain.SituacionExcluido
	tabla := make(map[string][]string)
	for _, origen := range domain.SituacionesParticipacion() {
		tabla[origen] = domain.DestinosSituacionParticipacion(origen)
	}
	tabla[domain.SituacionExcluido] = []string{domain.SituacionDisponible}
	politica, err := domain.NuevaPoliticaTransicionesSituacion(tabla)
	if err != nil {
		t.Fatal(err)
	}
	servicio.repositorio = &repositorioPoliticaReincorporacionPrueba{repositorioOperacionPrueba: repo, politica: politica}
	q := ports.SolicitudOperacionSituacion{
		SolicitudCambiarSituacionParticipacion: solicitudSituacionPrueba(t, ahora),
		Operacion:                              domain.OperacionReactivar,
		Justificante: domain.JustificanteOperacionSituacion{
			Tipo: domain.JustificanteSolicitudCandidato, Referencia: "justificante:renuncia", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		Validador: "persona:rrhh",
	}
	q.Destino = domain.SituacionDisponible
	q.Motivo = "Reincorporación tras validar el justificante de la renuncia"
	res, err := servicio.Operar(context.Background(), q)
	if err != nil || res.Situacion != domain.SituacionDisponible || res.ReciboRef == "" || repo.escrituras != 1 {
		t.Fatalf("reincorporación: resultado=%+v error=%v escrituras=%d", res, err, repo.escrituras)
	}
	if repo.operacion.Justificante != q.Justificante || repo.operacion.Validador != q.Validador {
		t.Fatal("reincorporación sin conservar justificante y validador")
	}
}
