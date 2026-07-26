package cobertura

import (
	"bytes"
	"context"
	"encoding"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type relojGobiernoOperacionCoberturaPrueba struct {
	instante time.Time
	err      error
	llamadas int
	cancelar context.CancelFunc
}

func (r *relojGobiernoOperacionCoberturaPrueba) AhoraGobiernoOperacionCobertura(
	context.Context,
) (time.Time, error) {
	r.llamadas++
	if r.cancelar != nil {
		r.cancelar()
	}
	return r.instante, r.err
}

type resolutorGobiernoOperacionCoberturaPrueba struct {
	publicacion PublicacionGobiernoOperacionCobertura
	err         error
	llamadas    int
	cancelar    context.CancelFunc
	solicitud   SolicitudResolucionGobiernoOperacionCobertura
}

func (r *resolutorGobiernoOperacionCoberturaPrueba) ResolverGobiernoOperacionCobertura(
	_ context.Context,
	solicitud SolicitudResolucionGobiernoOperacionCobertura,
) (PublicacionGobiernoOperacionCobertura, error) {
	r.llamadas++
	r.solicitud = solicitud
	if r.cancelar != nil {
		r.cancelar()
	}
	return r.publicacion, r.err
}

func TestGobiernoOperacionCoberturaResuelveAmbasAccionesDesdeServidor(
	t *testing.T,
) {
	casos := []struct {
		nombre      string
		rectificar  bool
		accion      domain.ClaveCatalogo
		claveMotivo string
		version     uint64
	}{
		{
			nombre:      "decidir",
			accion:      domain.AccionDecidirCoberturaGobernada,
			claveMotivo: "motivo_11111111111111111111111111111111",
			version:     2,
		},
		{
			nombre:      "rectificar",
			rectificar:  true,
			accion:      domain.AccionRectificarCoberturaGobernada,
			claveMotivo: "motivo_22222222222222222222222222222222",
			version:     3,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			solicitud := solicitudGobiernoOperacionCoberturaPrueba(
				t,
				caso.rectificar,
				caso.version,
			)
			instante := instanteOperacionDecisionCoberturaPrueba
			publicacion := publicacionGobiernoOperacionCoberturaPrueba(
				t,
				solicitud,
				instante,
			)
			reloj := &relojGobiernoOperacionCoberturaPrueba{
				instante: instante,
			}
			resolutor := &resolutorGobiernoOperacionCoberturaPrueba{
				publicacion: publicacion,
			}
			gobierno, err := ObtenerGobiernoOperacionCobertura(
				context.Background(),
				reloj,
				resolutor,
				solicitud,
			)
			if err != nil {
				t.Fatal(err)
			}
			relojUso := &relojGobiernoOperacionCoberturaPrueba{
				instante: instante.Add(time.Second),
			}
			datos, err := gobierno.DesplegarPara(
				context.Background(),
				relojUso,
				solicitud,
			)
			if err != nil {
				t.Fatal(err)
			}
			_, _, _, accionInterna, instanteInterno, err :=
				resolutor.solicitud.Coordenadas()
			if err != nil ||
				reloj.llamadas != 1 ||
				resolutor.llamadas != 1 ||
				relojUso.llamadas != 1 ||
				accionInterna != caso.accion ||
				!instanteInterno.Equal(instante) ||
				datos.Accion != caso.accion ||
				datos.MotivoAutorizacion.EntradaClave != caso.claveMotivo ||
				datos.Catalogo.Identidad() !=
					publicacion.Catalogo.Identidad() ||
				datos.Politica.Identidad() !=
					publicacion.Politica.Identidad() ||
				datos.PoliticaActuacion !=
					publicacion.PoliticaActuacion ||
				datos.FinalidadCTClave !=
					publicacion.PoliticaActuacion.
						FinalidadContratacionClave ||
				datos.FinalidadCTRef !=
					publicacion.PoliticaActuacion.
						FinalidadContratacionRef ||
				datos.FinalidadVEC !=
					publicacion.PoliticaActuacion.
						FinalidadAutorizacionVEC {
				t.Fatal("el gobierno servidor perdió una ligadura")
			}
		})
	}
}

func TestSolicitudGobiernoOperacionSoloAceptaIntencionMinima(t *testing.T) {
	decidir, err := NuevaSolicitudGobiernoDecisionCobertura(
		"organizacion_diputacion_granada",
		"expediente_temporal_2026_5487",
		2,
	)
	if err != nil ||
		decidir.accion != domain.AccionDecidirCoberturaGobernada {
		t.Fatal("no se derivó la acción de decisión")
	}
	rectificar, err := NuevaSolicitudGobiernoRectificacionCobertura(
		"organizacion_diputacion_granada",
		"expediente_temporal_2026_5487",
		3,
	)
	if err != nil ||
		rectificar.accion != domain.AccionRectificarCoberturaGobernada {
		t.Fatal("no se derivó la acción de rectificación")
	}
	casos := []struct {
		nombre, organizacion, expediente string
		version                          uint64
	}{
		{"organización", "", "expediente_temporal_2026_5487", 2},
		{"expediente", "organizacion_diputacion_granada", "", 2},
		{"versión cero", "organizacion_diputacion_granada", "expediente_temporal_2026_5487", 0},
		{"versión límite", "organizacion_diputacion_granada", "expediente_temporal_2026_5487", MaximoEnteroSeguroOperacionDecisionCobertura},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := NuevaSolicitudGobiernoDecisionCobertura(
				caso.organizacion,
				caso.expediente,
				caso.version,
			); !errors.Is(
				err,
				ErrSolicitudGobiernoOperacionCoberturaInvalida,
			) {
				t.Fatalf("se aceptaron coordenadas inválidas: %v", err)
			}
		})
	}
}

func TestGobiernoOperacionCoberturaRechazaAutoridadNoLigada(t *testing.T) {
	solicitud := solicitudGobiernoOperacionCoberturaPrueba(t, false, 2)
	instante := instanteOperacionDecisionCoberturaPrueba
	base := publicacionGobiernoOperacionCoberturaPrueba(
		t,
		solicitud,
		instante,
	)
	casos := []struct {
		nombre    string
		adulterar func(*PublicacionGobiernoOperacionCobertura)
	}{
		{"organización", func(p *PublicacionGobiernoOperacionCobertura) {
			p.OrganizacionRef = "organizacion_ajena"
		}},
		{"expediente", func(p *PublicacionGobiernoOperacionCobertura) {
			p.ExpedienteRef = "expediente_ajeno"
		}},
		{"versión", func(p *PublicacionGobiernoOperacionCobertura) {
			p.VersionExpediente++
		}},
		{"catálogo", func(p *PublicacionGobiernoOperacionCobertura) {
			p.Catalogo = domain.CatalogoViasCobertura{}
		}},
		{"política", func(p *PublicacionGobiernoOperacionCobertura) {
			p.Politica = domain.PoliticaDecisionCobertura{}
		}},
		{"huella actuación", func(p *PublicacionGobiernoOperacionCobertura) {
			p.PoliticaActuacion.HuellaSHA256 =
				strings.Repeat("0", 64)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			publicacion := base
			caso.adulterar(&publicacion)
			_, err := ObtenerGobiernoOperacionCobertura(
				context.Background(),
				&relojGobiernoOperacionCoberturaPrueba{
					instante: instante,
				},
				&resolutorGobiernoOperacionCoberturaPrueba{
					publicacion: publicacion,
				},
				solicitud,
			)
			if !errors.Is(err, ErrGobiernoOperacionCoberturaNoConfiable) {
				t.Fatalf("autoridad no ligada aceptada: %v", err)
			}
		})
	}
}

func TestGobiernoOperacionCoberturaRechazaFinalidadCTDivergente(t *testing.T) {
	solicitud := solicitudGobiernoOperacionCoberturaPrueba(t, false, 2)
	instante := instanteOperacionDecisionCoberturaPrueba
	publicacion := publicacionGobiernoOperacionCoberturaPrueba(
		t,
		solicitud,
		instante,
	)
	publicacion.PoliticaActuacion.FinalidadContratacionClave =
		"finalidad_ct_ajena"
	resellarPoliticaActuacionCoberturaPrueba(
		t,
		&publicacion.PoliticaActuacion,
	)
	_, err := ObtenerGobiernoOperacionCobertura(
		context.Background(),
		&relojGobiernoOperacionCoberturaPrueba{instante: instante},
		&resolutorGobiernoOperacionCoberturaPrueba{
			publicacion: publicacion,
		},
		solicitud,
	)
	if !errors.Is(err, ErrGobiernoOperacionCoberturaNoConfiable) {
		t.Fatalf("se aceptó una finalidad CT divergente: %v", err)
	}
}

func TestGobiernoOperacionCoberturaFallaCerradoEnRelojYResolutor(
	t *testing.T,
) {
	solicitud := solicitudGobiernoOperacionCoberturaPrueba(t, false, 2)
	instante := instanteOperacionDecisionCoberturaPrueba
	publicacion := publicacionGobiernoOperacionCoberturaPrueba(
		t,
		solicitud,
		instante,
	)
	t.Run("contexto nulo", func(t *testing.T) {
		reloj := &relojGobiernoOperacionCoberturaPrueba{}
		_, err := ObtenerGobiernoOperacionCobertura(
			nil, reloj, &resolutorGobiernoOperacionCoberturaPrueba{},
			solicitud,
		)
		if !errors.Is(err, ErrSolicitudGobiernoOperacionCoberturaInvalida) ||
			reloj.llamadas != 0 {
			t.Fatalf("el contexto nulo alcanzó el reloj: %v", err)
		}
	})
	t.Run("reloj nulo tipado", func(t *testing.T) {
		var reloj *relojGobiernoOperacionCoberturaPrueba
		_, err := ObtenerGobiernoOperacionCobertura(
			context.Background(),
			reloj,
			&resolutorGobiernoOperacionCoberturaPrueba{},
			solicitud,
		)
		if !errors.Is(
			err,
			ErrSolicitudGobiernoOperacionCoberturaInvalida,
		) {
			t.Fatalf("se aceptó reloj nulo: %v", err)
		}
	})
	t.Run("resolutor nulo tipado", func(t *testing.T) {
		var resolutor *resolutorGobiernoOperacionCoberturaPrueba
		reloj := &relojGobiernoOperacionCoberturaPrueba{
			instante: instante,
		}
		_, err := ObtenerGobiernoOperacionCobertura(
			context.Background(),
			reloj,
			resolutor,
			solicitud,
		)
		if !errors.Is(
			err,
			ErrSolicitudGobiernoOperacionCoberturaInvalida,
		) || reloj.llamadas != 0 {
			t.Fatalf("el reloj corrió con resolutor nulo: %v", err)
		}
	})
	t.Run("reloj no canónico", func(t *testing.T) {
		reloj := &relojGobiernoOperacionCoberturaPrueba{
			instante: instante.Add(time.Nanosecond),
		}
		resolutor := &resolutorGobiernoOperacionCoberturaPrueba{}
		_, err := ObtenerGobiernoOperacionCobertura(
			context.Background(),
			reloj,
			resolutor,
			solicitud,
		)
		if !errors.Is(
			err,
			ErrGobiernoOperacionCoberturaNoConfiable,
		) || resolutor.llamadas != 0 {
			t.Fatalf("instante no canónico alcanzó el resolutor: %v", err)
		}
	})
	t.Run("error privado de reloj", func(t *testing.T) {
		const privado = "dsn=secreto-reloj"
		_, err := ObtenerGobiernoOperacionCobertura(
			context.Background(),
			&relojGobiernoOperacionCoberturaPrueba{
				err: errors.New(privado),
			},
			&resolutorGobiernoOperacionCoberturaPrueba{},
			solicitud,
		)
		if !errors.Is(
			err,
			ErrGobiernoOperacionCoberturaNoDisponible,
		) || strings.Contains(err.Error(), privado) {
			t.Fatalf("se filtró error del reloj: %v", err)
		}
	})
	t.Run("error privado de resolutor", func(t *testing.T) {
		const privado = "fila=persona-real"
		_, err := ObtenerGobiernoOperacionCobertura(
			context.Background(),
			&relojGobiernoOperacionCoberturaPrueba{
				instante: instante,
			},
			&resolutorGobiernoOperacionCoberturaPrueba{
				err: errors.New(privado),
			},
			solicitud,
		)
		if !errors.Is(
			err,
			ErrGobiernoOperacionCoberturaNoDisponible,
		) || strings.Contains(err.Error(), privado) {
			t.Fatalf("se filtró error del resolutor: %v", err)
		}
	})
	t.Run("cancelación durante reloj", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		reloj := &relojGobiernoOperacionCoberturaPrueba{
			instante: instante,
			cancelar: cancelar,
		}
		resolutor := &resolutorGobiernoOperacionCoberturaPrueba{}
		_, err := ObtenerGobiernoOperacionCobertura(
			ctx,
			reloj,
			resolutor,
			solicitud,
		)
		if !errors.Is(err, context.Canceled) ||
			resolutor.llamadas != 0 {
			t.Fatalf("cancelación no cerró la frontera: %v", err)
		}
	})
	t.Run("cancelación durante resolutor", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		resolutor := &resolutorGobiernoOperacionCoberturaPrueba{
			publicacion: publicacion,
			cancelar:    cancelar,
		}
		_, err := ObtenerGobiernoOperacionCobertura(
			ctx,
			&relojGobiernoOperacionCoberturaPrueba{
				instante: instante,
			},
			resolutor,
			solicitud,
		)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelación del resolutor fue aceptada: %v", err)
		}
	})
}

func TestGobiernoOperacionCoberturaImpideRetrofechaReusoYCaducidad(
	t *testing.T,
) {
	solicitud := solicitudGobiernoOperacionCoberturaPrueba(t, false, 2)
	instante := instanteOperacionDecisionCoberturaPrueba
	publicacion := publicacionGobiernoOperacionCoberturaPrueba(
		t,
		solicitud,
		instante,
	)
	gobierno, err := ObtenerGobiernoOperacionCobertura(
		context.Background(),
		&relojGobiernoOperacionCoberturaPrueba{instante: instante},
		&resolutorGobiernoOperacionCoberturaPrueba{
			publicacion: publicacion,
		},
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre    string
		instante  time.Time
		solicitud SolicitudGobiernoOperacionCobertura
	}{
		{"reloj antiguo", instante.Add(-time.Microsecond), solicitud},
		{"límite exclusivo", gobierno.validaHasta, solicitud},
		{
			"otra solicitud",
			instante,
			solicitudGobiernoOperacionCoberturaPrueba(t, true, 2),
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := gobierno.DesplegarPara(
				context.Background(),
				&relojGobiernoOperacionCoberturaPrueba{
					instante: caso.instante,
				},
				caso.solicitud,
			)
			if !errors.Is(err, ErrGobiernoOperacionCoberturaNoConfiable) {
				t.Fatalf("se reutilizó capacidad inválida: %v", err)
			}
		})
	}
	t.Run("reloj nulo al desplegar", func(t *testing.T) {
		var reloj *relojGobiernoOperacionCoberturaPrueba
		_, err := gobierno.DesplegarPara(
			context.Background(),
			reloj,
			solicitud,
		)
		if !errors.Is(err, ErrGobiernoOperacionCoberturaNoConfiable) {
			t.Fatalf("se aceptó reloj nulo al desplegar: %v", err)
		}
	})
	t.Run("cancelación antes de desplegar", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		reloj := &relojGobiernoOperacionCoberturaPrueba{
			instante: instante,
		}
		_, err := gobierno.DesplegarPara(ctx, reloj, solicitud)
		if !errors.Is(err, context.Canceled) || reloj.llamadas != 0 {
			t.Fatalf("la cancelación alcanzó el reloj: %v", err)
		}
	})
	t.Run("reloj no canónico al desplegar", func(t *testing.T) {
		_, err := gobierno.DesplegarPara(
			context.Background(),
			&relojGobiernoOperacionCoberturaPrueba{
				instante: instante.Add(time.Nanosecond),
			},
			solicitud,
		)
		if !errors.Is(err, ErrGobiernoOperacionCoberturaNoConfiable) {
			t.Fatalf("se aceptó reloj no canónico: %v", err)
		}
	})
	t.Run("vigencia más corta de actuación", func(t *testing.T) {
		brevePublicacion := publicacion
		brevePublicacion.PoliticaActuacion.Vigencia.Hasta =
			instante.Add(2 * time.Second)
		resellarPoliticaActuacionCoberturaPrueba(
			t,
			&brevePublicacion.PoliticaActuacion,
		)
		breve, err := ObtenerGobiernoOperacionCobertura(
			context.Background(),
			&relojGobiernoOperacionCoberturaPrueba{
				instante: instante,
			},
			&resolutorGobiernoOperacionCoberturaPrueba{
				publicacion: brevePublicacion,
			},
			solicitud,
		)
		if err != nil ||
			!breve.validaHasta.Equal(instante.Add(2*time.Second)) {
			t.Fatalf("no se acotó a la publicación: %v", err)
		}
	})
}

func TestGobiernoOperacionCoberturaClonaYProhibeSerializacion(t *testing.T) {
	solicitud := solicitudGobiernoOperacionCoberturaPrueba(t, false, 2)
	instante := instanteOperacionDecisionCoberturaPrueba
	publicacion := publicacionGobiernoOperacionCoberturaPrueba(
		t,
		solicitud,
		instante,
	)
	resolutor := &resolutorGobiernoOperacionCoberturaPrueba{
		publicacion: publicacion,
	}
	gobierno, err := ObtenerGobiernoOperacionCobertura(
		context.Background(),
		&relojGobiernoOperacionCoberturaPrueba{instante: instante},
		resolutor,
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := gobierno.DesplegarPara(
		context.Background(),
		&relojGobiernoOperacionCoberturaPrueba{
			instante: instante.Add(time.Second),
		},
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	catalogoMutado := datos.Catalogo.Publicacion()
	catalogoMutado.Vias[0].Clave = "via_mutada"
	politicaMutada := datos.Politica.Publicacion()
	politicaMutada.Vias[0].ViaClave = "via_mutada"
	segundos, err := gobierno.DesplegarPara(
		context.Background(),
		&relojGobiernoOperacionCoberturaPrueba{
			instante: instante.Add(2 * time.Second),
		},
		solicitud,
	)
	if err != nil ||
		segundos.Catalogo.Identidad() != publicacion.Catalogo.Identidad() ||
		segundos.Politica.Identidad() != publicacion.Politica.Identidad() {
		t.Fatalf("la capacidad comparte memoria mutable: %v", err)
	}

	valores := []any{
		solicitud,
		resolutor.solicitud,
		publicacion,
		gobierno,
		datos,
	}
	for _, valor := range valores {
		assertSerializacionGobiernoProhibida(t, valor, publicacion.ExpedienteRef)
	}
}

func assertSerializacionGobiernoProhibida(
	t *testing.T,
	valor any,
	secreto string,
) {
	t.Helper()
	if _, err := json.Marshal(valor); err == nil {
		t.Fatalf("JSON aceptó %T", valor)
	}
	if _, err := xml.Marshal(valor); err == nil {
		t.Fatalf("XML aceptó %T", valor)
	}
	if _, err := valor.(encoding.TextMarshaler).MarshalText(); err == nil {
		t.Fatalf("texto aceptó %T", valor)
	}
	if _, err := valor.(encoding.BinaryMarshaler).MarshalBinary(); err == nil {
		t.Fatalf("binario aceptó %T", valor)
	}
	var destino bytes.Buffer
	if err := gob.NewEncoder(&destino).Encode(valor); err == nil {
		t.Fatalf("gob aceptó %T", valor)
	}
	representaciones := []string{
		fmt.Sprintf("%v", valor),
		fmt.Sprintf("%+v", valor),
		fmt.Sprintf("%#v", valor),
		slog.Any("gobierno", valor).Value.Resolve().String(),
	}
	for _, representacion := range representaciones {
		if strings.Contains(representacion, secreto) {
			t.Fatalf("%T filtró datos: %q", valor, representacion)
		}
	}
}

func solicitudGobiernoOperacionCoberturaPrueba(
	t *testing.T,
	rectificar bool,
	version uint64,
) SolicitudGobiernoOperacionCobertura {
	t.Helper()
	var (
		solicitud SolicitudGobiernoOperacionCobertura
		err       error
	)
	if rectificar {
		solicitud, err = NuevaSolicitudGobiernoRectificacionCobertura(
			"organizacion_diputacion_granada",
			"expediente_temporal_2026_5487",
			version,
		)
	} else {
		solicitud, err = NuevaSolicitudGobiernoDecisionCobertura(
			"organizacion_diputacion_granada",
			"expediente_temporal_2026_5487",
			version,
		)
	}
	if err != nil {
		t.Fatal(err)
	}
	return solicitud
}

func publicacionGobiernoOperacionCoberturaPrueba(
	t *testing.T,
	solicitud SolicitudGobiernoOperacionCobertura,
	instante time.Time,
) PublicacionGobiernoOperacionCobertura {
	t.Helper()
	catalogo, politica := gobiernoCatalogoYPoliticaPrueba(t, instante)
	finalidadClave, finalidadRef := politica.Finalidad()
	actuacion := PublicacionPoliticaActuacionCobertura{
		Referencia:                 "politica_actuacion_cobertura_2026",
		Version:                    5,
		Canon:                      CanonHuellaPoliticaActuacionCoberturaV1(),
		OrganizacionRef:            solicitud.organizacionRef,
		Accion:                     solicitud.accion,
		Catalogo:                   catalogo.Identidad(),
		Politica:                   politica.Identidad(),
		FinalidadContratacionClave: finalidadClave,
		FinalidadContratacionRef:   finalidadRef,
		FinalidadAutorizacionVEC:   "tramitar_cobertura_temporal",
		UnidadEjecutoraRef:         "unidad_rrhh_cobertura_01",
		FaseDestino:                "asignacion_unidad",
		EstadoDestino:              domain.EstadoEnCurso,
		MotivoAutorizacionDecidir: motivoAutorizacionGobiernoPrueba(
			"motivo_11111111111111111111111111111111",
		),
		MotivoAutorizacionRectificar: motivoAutorizacionGobiernoPrueba(
			"motivo_22222222222222222222222222222222",
		),
		PublicadaEn: instante.Add(-time.Hour),
		Vigencia: domain.VigenciaCatalogoCobertura{
			Desde: instante.Add(-time.Minute),
			Hasta: instante.Add(time.Hour),
		},
	}
	resellarPoliticaActuacionCoberturaPrueba(t, &actuacion)
	return PublicacionGobiernoOperacionCobertura{
		OrganizacionRef:   solicitud.organizacionRef,
		ExpedienteRef:     solicitud.expedienteRef,
		VersionExpediente: solicitud.versionExpediente,
		Catalogo:          catalogo,
		Politica:          politica,
		PoliticaActuacion: actuacion,
	}
}

func resellarPoliticaActuacionCoberturaPrueba(
	t *testing.T,
	politica *PublicacionPoliticaActuacionCobertura,
) {
	t.Helper()
	huella, err := CalcularHuellaSHA256PoliticaActuacionCobertura(*politica)
	if err != nil {
		t.Fatal(err)
	}
	politica.HuellaSHA256 = huella
}

func motivoAutorizacionGobiernoPrueba(
	clave string,
) dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion_cobertura",
		CatalogoVersion:      7,
		CatalogoHuellaSHA256: strings.Repeat("b", 64),
		EntradaClave:         clave,
	}
}

func gobiernoCatalogoYPoliticaPrueba(
	t *testing.T,
	instante time.Time,
) (
	domain.CatalogoViasCobertura,
	domain.PoliticaDecisionCobertura,
) {
	t.Helper()
	vigencia := domain.VigenciaCatalogoCobertura{
		Desde: instante.Add(-time.Minute),
		Hasta: instante.Add(time.Hour),
	}
	catalogo, err := domain.PublicarCatalogoViasCobertura(
		domain.BorradorCatalogoViasCobertura{
			Referencia:     "catalogo_cobertura_gobierno_01",
			Version:        9,
			PublicadoEn:    instante.Add(-2 * time.Hour),
			Vigencia:       vigencia,
			ProcedenciaRef: "publicacion_catalogo_gobierno_01",
			Vias: []domain.DefinicionViaCobertura{{
				Clave: "via_configurable",
				Orden: 1,
				Comprobaciones: []domain.ComprobacionExigibleCobertura{{
					Clave:       "comprobacion_configurable",
					Orden:       1,
					Obligatoria: true,
					Procedencia: domain.ProcedenciaComprobacionCobertura{
						Clave:               "fuente_configurable",
						DefinicionFuenteRef: "fuente_cobertura_gobernada_01",
					},
				}},
			}},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	politica, err := domain.PublicarPoliticaDecisionCobertura(
		domain.BorradorPoliticaDecisionCobertura{
			Referencia:      "politica_decision_gobierno_01",
			Version:         4,
			Catalogo:        catalogo.Identidad(),
			OrganizacionRef: "organizacion_diputacion_granada",
			FinalidadClave:  "gestionar_cobertura_temporal",
			FinalidadRef:    "finalidad_cobertura_temporal_01",
			PublicadaEn:     instante.Add(-90 * time.Minute),
			Vigencia:        vigencia,
			ProcedenciaRef:  "publicacion_politica_gobierno_01",
			Vias: []domain.ReglaViaDecisionCobertura{{
				ViaClave:  "via_configurable",
				Prioridad: 1,
				Comprobaciones: []domain.ReglaComprobacionDecisionCobertura{{
					Clave: "comprobacion_configurable",
					ResultadosHabilitantes: []domain.ResultadoComprobacion{
						domain.ComprobacionAfirmativa,
					},
					TratamientoAusencia: domain.AusenciaCoberturaBloquea,
				}},
			}},
		},
		catalogo,
	)
	if err != nil {
		t.Fatal(err)
	}
	return catalogo, politica
}
