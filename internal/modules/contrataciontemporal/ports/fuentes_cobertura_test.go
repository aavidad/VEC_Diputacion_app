package ports

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type fuenteCoberturaDoble func(
	context.Context,
	SolicitudConsultarCobertura,
) (ResultadoConsultaCobertura, error)

func (f fuenteCoberturaDoble) ConsultarCobertura(
	ctx context.Context,
	solicitud SolicitudConsultarCobertura,
) (ResultadoConsultaCobertura, error) {
	return f(ctx, solicitud)
}

type relojCoberturaDoble struct {
	ahora time.Time
}

func (r relojCoberturaDoble) Ahora() time.Time {
	return r.ahora
}

func TestConsultarCoberturaConFuenteDevuelveResultadoLigado(t *testing.T) {
	solicitud := solicitudCoberturaPrueba()
	fuente := fuenteCoberturaDoble(func(
		_ context.Context,
		recibida SolicitudConsultarCobertura,
	) (ResultadoConsultaCobertura, error) {
		if recibida != solicitud {
			t.Fatalf("solicitud alterada: %#v", recibida)
		}
		return resultadoCoberturaPrueba(recibida), nil
	})

	resultado, err := ConsultarCoberturaConFuente(
		context.Background(), fuente,
		relojCoberturaDoble{ahora: solicitud.SolicitadaEn.Add(2 * time.Second)},
		solicitud, time.Second,
	)
	if err != nil || resultado.Validar() != nil ||
		resultado.Resultado != domain.ComprobacionAfirmativa ||
		resultado.FuenteRef != "fuente:bolsa-publicada-v7" {
		t.Fatalf("consulta válida rechazada: %#v, %v", resultado, err)
	}
}

func TestConsultarCoberturaConFuenteFallaCerradoYNoFiltraError(t *testing.T) {
	solicitud := solicitudCoberturaPrueba()
	fallo := errors.New("dsn=secreto detalle privado del proveedor")
	fuente := fuenteCoberturaDoble(func(
		context.Context,
		SolicitudConsultarCobertura,
	) (ResultadoConsultaCobertura, error) {
		return resultadoCoberturaPrueba(solicitud), fallo
	})

	resultado, err := ConsultarCoberturaConFuente(
		context.Background(), fuente,
		relojCoberturaDoble{ahora: solicitud.SolicitadaEn.Add(2 * time.Second)},
		solicitud, time.Second,
	)
	if !errors.Is(err, ErrFuenteCoberturaNoDisponible) ||
		!errors.Is(err, fallo) ||
		err.Error() != ErrFuenteCoberturaNoDisponible.Error() ||
		resultado != (domain.ComprobacionCobertura{}) ||
		strings.Contains(err.Error(), "secreto") {
		t.Fatalf("el fallo filtró o aceptó resultado: %#v, %v", resultado, err)
	}
}

func TestConsultarCoberturaConFuenteRechazaCrucesDeCoordenadas(t *testing.T) {
	casos := []struct {
		nombre  string
		alterar func(*ResultadoConsultaCobertura)
	}{
		{"petición", func(r *ResultadoConsultaCobertura) {
			r.PeticionRef = "peticion:cobertura-ajena"
		}},
		{"organización", func(r *ResultadoConsultaCobertura) {
			r.OrganizacionRef = "organizacion:ajena"
		}},
		{"expediente", func(r *ResultadoConsultaCobertura) {
			r.ExpedienteRef = "expediente:ajeno"
		}},
		{"versión expediente", func(r *ResultadoConsultaCobertura) {
			r.VersionExpediente++
		}},
		{"referencia catálogo", func(r *ResultadoConsultaCobertura) {
			r.Catalogo.Referencia = "catalogo:ajeno"
		}},
		{"versión catálogo", func(r *ResultadoConsultaCobertura) {
			r.Catalogo.Version++
		}},
		{"huella catálogo", func(r *ResultadoConsultaCobertura) {
			r.Catalogo.HuellaSHA256 = strings.Repeat("b", 64)
		}},
		{"vía", func(r *ResultadoConsultaCobertura) {
			r.ViaClave = "oferta_sae"
		}},
		{"procedencia", func(r *ResultadoConsultaCobertura) {
			r.ProcedenciaClave = "sae"
		}},
		{"categoría", func(r *ResultadoConsultaCobertura) {
			r.CategoriaRef = "categoria:ajena"
		}},
		{"periodo", func(r *ResultadoConsultaCobertura) {
			r.Periodo.Fin = r.Periodo.Fin.AddDate(0, 0, 1)
		}},
		{"comprobación", func(r *ResultadoConsultaCobertura) {
			r.Comprobacion.Clave = "hay_candidaturas_disponibles"
		}},
		{"detalle libre", func(r *ResultadoConsultaCobertura) {
			r.Comprobacion.Detalle = "Una persona concreta está disponible."
		}},
		{"definición fuente", func(r *ResultadoConsultaCobertura) {
			r.DefinicionFuenteRef = "fuente:definicion-ajena"
		}},
		{"cronología", func(r *ResultadoConsultaCobertura) {
			r.Comprobacion.EvaluadaEn =
				solicitudCoberturaPrueba().SolicitadaEn.Add(-time.Microsecond)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			solicitud := solicitudCoberturaPrueba()
			fuente := fuenteCoberturaDoble(func(
				context.Context,
				SolicitudConsultarCobertura,
			) (ResultadoConsultaCobertura, error) {
				resultado := resultadoCoberturaPrueba(solicitud)
				caso.alterar(&resultado)
				return resultado, nil
			})
			if _, err := ConsultarCoberturaConFuente(
				context.Background(), fuente,
				relojCoberturaDoble{ahora: solicitud.SolicitadaEn.Add(2 * time.Second)},
				solicitud, time.Second,
			); !errors.Is(err, ErrResultadoFuenteCoberturaNoConfiable) {
				t.Fatalf("aceptó coordenada cruzada: %v", err)
			}
		})
	}
}

func TestSolicitudCoberturaMantieneListaPositivaMinimizada(t *testing.T) {
	tipo := reflect.TypeOf(SolicitudConsultarCobertura{})
	esperados := map[string]struct{}{
		"PeticionRef": {}, "OrganizacionRef": {}, "ExpedienteRef": {},
		"VersionExpediente": {}, "Catalogo": {}, "ViaClave": {},
		"Comprobacion": {}, "CategoriaRef": {}, "Periodo": {},
		"SolicitadaEn": {},
	}
	if tipo.NumField() != len(esperados) {
		t.Fatalf("la petición amplió su superficie: %d campos", tipo.NumField())
	}
	for indice := range tipo.NumField() {
		campo := tipo.Field(indice)
		if _, permitido := esperados[campo.Name]; !permitido {
			t.Fatalf("campo no minimizado en la petición: %s", campo.Name)
		}
	}
}

func TestConsultarCoberturaConFuenteImponeTiempoYCancela(t *testing.T) {
	solicitud := solicitudCoberturaPrueba()
	fuente := fuenteCoberturaDoble(func(
		ctx context.Context,
		_ SolicitudConsultarCobertura,
	) (ResultadoConsultaCobertura, error) {
		<-ctx.Done()
		return ResultadoConsultaCobertura{}, ctx.Err()
	})

	if _, err := ConsultarCoberturaConFuente(
		context.Background(), fuente,
		relojCoberturaDoble{ahora: solicitud.SolicitadaEn.Add(2 * time.Second)},
		solicitud, time.Millisecond,
	); !errors.Is(err, ErrFuenteCoberturaNoDisponible) ||
		!errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("no aplicó el timeout: %v", err)
	}

	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := ConsultarCoberturaConFuente(
		ctx, fuente,
		relojCoberturaDoble{ahora: solicitud.SolicitadaEn.Add(2 * time.Second)},
		solicitud, time.Second,
	); !errors.Is(err, ErrFuenteCoberturaNoDisponible) ||
		!errors.Is(err, context.Canceled) {
		t.Fatalf("no respetó la cancelación previa: %v", err)
	}
}

func TestConsultarCoberturaConFuenteRechazaConfiguracionYNuloTipado(t *testing.T) {
	solicitud := solicitudCoberturaPrueba()
	var nula *fuenteCoberturaNula
	for _, tiempoMaximo := range []time.Duration{
		0, -time.Second, TiempoMaximoFuenteCobertura + time.Nanosecond,
	} {
		if _, err := ConsultarCoberturaConFuente(
			context.Background(), nula,
			relojCoberturaDoble{ahora: solicitud.SolicitadaEn.Add(2 * time.Second)},
			solicitud, tiempoMaximo,
		); !errors.Is(err, ErrPeticionFuenteCoberturaInvalida) {
			t.Fatalf("aceptó configuración o nulo tipado: %v", err)
		}
	}
}

func TestConsultarCoberturaConFuenteRechazaResultadoFuturoYRelojNulo(t *testing.T) {
	solicitud := solicitudCoberturaPrueba()
	fuente := fuenteCoberturaDoble(func(
		context.Context,
		SolicitudConsultarCobertura,
	) (ResultadoConsultaCobertura, error) {
		resultado := resultadoCoberturaPrueba(solicitud)
		resultado.Comprobacion.EvaluadaEn = solicitud.SolicitadaEn.Add(3 * time.Second)
		return resultado, nil
	})
	reloj := relojCoberturaDoble{ahora: solicitud.SolicitadaEn.Add(2 * time.Second)}
	if _, err := ConsultarCoberturaConFuente(
		context.Background(), fuente, reloj, solicitud, time.Second,
	); !errors.Is(err, ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("aceptó un resultado fechado en el futuro: %v", err)
	}

	var relojNulo *relojCoberturaNulo
	if _, err := ConsultarCoberturaConFuente(
		context.Background(), fuente, relojNulo, solicitud, time.Second,
	); !errors.Is(err, ErrPeticionFuenteCoberturaInvalida) {
		t.Fatalf("aceptó un reloj con puntero nulo: %v", err)
	}
}

type relojCoberturaNulo struct{}

func (*relojCoberturaNulo) Ahora() time.Time {
	panic("no debe invocarse")
}

type fuenteCoberturaNula struct{}

func (*fuenteCoberturaNula) ConsultarCobertura(
	context.Context,
	SolicitudConsultarCobertura,
) (ResultadoConsultaCobertura, error) {
	panic("no debe invocarse")
}

func solicitudCoberturaPrueba() SolicitudConsultarCobertura {
	return SolicitudConsultarCobertura{
		PeticionRef:       "peticion:cobertura-001",
		OrganizacionRef:   "organizacion:diputacion-granada",
		ExpedienteRef:     "expediente:temporal-001",
		VersionExpediente: 3,
		Catalogo: domain.IdentidadCatalogoViasCobertura{
			Referencia: "catalogo:cobertura-general", Version: 7,
			HuellaSHA256: strings.Repeat("a", 64),
		},
		ViaClave: "bolsa_vigente",
		Comprobacion: domain.ComprobacionExigibleCobertura{
			Clave: "existe_bolsa_vigente", Orden: 1, Obligatoria: true,
			Procedencia: domain.ProcedenciaComprobacionCobertura{
				Clave:               "bolsa",
				DefinicionFuenteRef: "fuente:definicion-bolsa-v3",
			},
		},
		CategoriaRef: "categoria:trabajo-social",
		Periodo: domain.PeriodoPrevisto{
			Inicio: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			Fin:    time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC),
		},
		SolicitadaEn: time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC),
	}
}

func resultadoCoberturaPrueba(
	solicitud SolicitudConsultarCobertura,
) ResultadoConsultaCobertura {
	return ResultadoConsultaCobertura{
		PeticionRef:         solicitud.PeticionRef,
		OrganizacionRef:     solicitud.OrganizacionRef,
		ExpedienteRef:       solicitud.ExpedienteRef,
		VersionExpediente:   solicitud.VersionExpediente,
		Catalogo:            solicitud.Catalogo,
		ViaClave:            solicitud.ViaClave,
		ProcedenciaClave:    solicitud.Comprobacion.Procedencia.Clave,
		CategoriaRef:        solicitud.CategoriaRef,
		Periodo:             solicitud.Periodo,
		DefinicionFuenteRef: solicitud.Comprobacion.Procedencia.DefinicionFuenteRef,
		Comprobacion: domain.ComprobacionCobertura{
			Clave:      solicitud.Comprobacion.Clave,
			Resultado:  domain.ComprobacionAfirmativa,
			FuenteRef:  "fuente:bolsa-publicada-v7",
			ReciboRef:  "recibo:consulta-bolsa-001",
			EvaluadaEn: solicitud.SolicitadaEn.Add(time.Second),
		},
	}
}
