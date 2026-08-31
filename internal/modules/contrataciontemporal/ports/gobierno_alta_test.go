package ports

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var instanteGobiernoAltaPrueba = time.Date(
	2026, 8, 31, 12, 0, 0, 123456000, time.UTC,
)

type autoridadGobiernoAltaPrueba struct {
	llamadas atomic.Int64
	resolver func(context.Context, SolicitudGobiernoAlta) (PublicacionGobiernoAlta, error)
}

func (a *autoridadGobiernoAltaPrueba) ResolverGobiernoAlta(
	ctx context.Context,
	solicitud SolicitudGobiernoAlta,
) (PublicacionGobiernoAlta, error) {
	a.llamadas.Add(1)
	return a.resolver(ctx, solicitud)
}

func solicitudCentroGobiernoAltaPrueba() domain.SolicitudCentro {
	return domain.SolicitudCentro{
		CentroRef:     "centro:servicios-sociales",
		ContactoRef:   "contacto:centro:001",
		CategoriaRef:  "categoria:tecnica-superior",
		GrupoSubgrupo: "A1",
		MotivoClave:   "sustitucion_temporal",
		Detalle:       "Necesidad temporal acreditada",
		Periodo: domain.PeriodoPrevisto{
			Inicio: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
			Fin:    time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		RC: domain.DeclaracionRC{
			Existe:       true,
			Numero:       "rc:2026:0001",
			Fecha:        time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC),
			Importe:      domain.Importe{Centimos: 125_000, Moneda: "EUR"},
			DocumentoRef: "documento:rc:0001",
		},
		DocumentosAdjuntos: []string{
			"documento:solicitud:001",
			"documento:informe:001",
		},
		Observaciones: "Tramitacion prioritaria",
	}
}

func configuracionGobiernoAltaPrueba() ConfiguracionAltaFlujo {
	return ConfiguracionAltaFlujo{
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: "flujo:contratacion-temporal:general",
			Version:       7,
			HuellaSHA256:  strings.Repeat("a", 64),
		},
		FaseInicial:      "solicitud_centro",
		UnidadInicialRef: "unidad:rrhh:entrada",
		AccionInicial:    "registrar_solicitud_centro",
	}
}

func motivoGobiernoAltaPrueba() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion_alta",
		CatalogoVersion:      3,
		CatalogoHuellaSHA256: strings.Repeat("b", 64),
		EntradaClave:         "tramitar_solicitud_centro",
	}
}

func publicacionGobiernoAltaPrueba() PublicacionGobiernoAlta {
	return PublicacionGobiernoAlta{
		OrganizacionRef:    "organizacion:diputacion-granada",
		Solicitud:          solicitudCentroGobiernoAltaPrueba(),
		Configuracion:      configuracionGobiernoAltaPrueba(),
		MotivoAutorizacion: motivoGobiernoAltaPrueba(),
		PublicadaEn:        instanteGobiernoAltaPrueba.Add(-2 * time.Hour),
		VigenteDesde:       instanteGobiernoAltaPrueba.Add(-time.Hour),
		VigenteHasta:       instanteGobiernoAltaPrueba.Add(time.Hour),
	}
}

func nuevaSolicitudGobiernoAltaPrueba(t *testing.T, instante time.Time) SolicitudGobiernoAlta {
	t.Helper()
	solicitud, err := NuevaSolicitudGobiernoAlta(
		"organizacion:diputacion-granada",
		solicitudCentroGobiernoAltaPrueba(),
		instante,
	)
	if err != nil {
		t.Fatalf("crear solicitud de gobierno: %v", err)
	}
	return solicitud
}

func resolverGobiernoAltaPrueba(
	t *testing.T,
	publicacion PublicacionGobiernoAlta,
) (SolicitudGobiernoAlta, InstantaneaGobiernoAlta, *autoridadGobiernoAltaPrueba) {
	t.Helper()
	solicitud := nuevaSolicitudGobiernoAltaPrueba(t, instanteGobiernoAltaPrueba)
	autoridad := &autoridadGobiernoAltaPrueba{resolver: func(
		context.Context,
		SolicitudGobiernoAlta,
	) (PublicacionGobiernoAlta, error) {
		return publicacion, nil
	}}
	instantanea, err := ResolverGobiernoAltaSeguro(context.Background(), autoridad, solicitud)
	if err != nil {
		t.Fatalf("resolver gobierno nominal: %v", err)
	}
	return solicitud, instantanea, autoridad
}

func TestGobiernoAltaResuelveUnaAutoridadYSellaInstantanea(t *testing.T) {
	solicitudEntrada := solicitudCentroGobiernoAltaPrueba()
	solicitud, err := NuevaSolicitudGobiernoAlta(
		"organizacion:diputacion-granada",
		solicitudEntrada,
		instanteGobiernoAltaPrueba,
	)
	if err != nil {
		t.Fatalf("solicitud nominal: %v", err)
	}
	solicitudEntrada.DocumentosAdjuntos[0] = "documento:mutado:entrada"
	publicacion := publicacionGobiernoAltaPrueba()
	autoridad := &autoridadGobiernoAltaPrueba{resolver: func(
		_ context.Context,
		recibida SolicitudGobiernoAlta,
	) (PublicacionGobiernoAlta, error) {
		datos, err := recibida.Datos()
		if err != nil || !solicitudesCentroGobiernoAltaIguales(
			datos.Solicitud,
			publicacion.Solicitud,
		) {
			t.Fatalf("la autoridad no recibio el clon nominal: %#v, %v", datos, err)
		}
		datos.Solicitud.DocumentosAdjuntos[0] = "documento:mutado:autoridad"
		return publicacion, nil
	}}
	instantanea, err := ResolverGobiernoAltaSeguro(
		context.Background(),
		autoridad,
		solicitud,
	)
	if err != nil || autoridad.llamadas.Load() != 1 {
		t.Fatalf("resolucion inesperada: llamadas=%d err=%v", autoridad.llamadas.Load(), err)
	}
	publicacion.Solicitud.DocumentosAdjuntos[0] = "documento:mutado:salida"
	datos, err := instantanea.Datos()
	if err != nil || datos.OrganizacionRef != "organizacion:diputacion-granada" ||
		datos.Configuracion != configuracionGobiernoAltaPrueba() ||
		datos.MotivoAutorizacion != motivoGobiernoAltaPrueba() ||
		datos.Solicitud.DocumentosAdjuntos[0] != "documento:solicitud:001" ||
		!datos.InstanteGobierno.Equal(instanteGobiernoAltaPrueba) {
		t.Fatalf("instantanea no sellada o no copiada: %#v, %v", datos, err)
	}
	if err := instantanea.ValidarPara(solicitud, instanteGobiernoAltaPrueba); err != nil {
		t.Fatalf("instantanea nominal invalida: %v", err)
	}
}

func TestGobiernoAltaEntradaInvalidaNoInvocaAutoridad(t *testing.T) {
	base := solicitudCentroGobiernoAltaPrueba()
	casos := map[string]struct {
		organizacion string
		instante     time.Time
		alterar      func(*domain.SolicitudCentro)
	}{
		"organizacion": {organizacion: "", instante: instanteGobiernoAltaPrueba},
		"instante cero": {
			organizacion: "organizacion:diputacion-granada", instante: time.Time{},
		},
		"instante no UTC": {
			organizacion: "organizacion:diputacion-granada",
			instante:     instanteGobiernoAltaPrueba.In(time.FixedZone("local", 3600)),
		},
		"centro": {
			organizacion: "organizacion:diputacion-granada", instante: instanteGobiernoAltaPrueba,
			alterar: func(s *domain.SolicitudCentro) { s.CentroRef = "" },
		},
		"documentos duplicados": {
			organizacion: "organizacion:diputacion-granada", instante: instanteGobiernoAltaPrueba,
			alterar: func(s *domain.SolicitudCentro) {
				s.DocumentosAdjuntos[1] = s.DocumentosAdjuntos[0]
			},
		},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			solicitudCentro := base
			solicitudCentro.DocumentosAdjuntos = append(
				[]string(nil), base.DocumentosAdjuntos...,
			)
			if caso.alterar != nil {
				caso.alterar(&solicitudCentro)
			}
			solicitud, err := NuevaSolicitudGobiernoAlta(
				caso.organizacion, solicitudCentro, caso.instante,
			)
			if err == nil {
				t.Fatal("se construyo una solicitud invalida")
			}
			autoridad := autoridadGobiernoAltaQueNoDebeLlamarse(t)
			_, err = ResolverGobiernoAltaSeguro(context.Background(), autoridad, solicitud)
			if !errors.Is(err, ErrGobiernoAltaNoDisponible) || autoridad.llamadas.Load() != 0 {
				t.Fatalf("entrada no cerrada: llamadas=%d err=%v", autoridad.llamadas.Load(), err)
			}
		})
	}
}

func TestGobiernoAltaRechazaCadaCampoCruzado(t *testing.T) {
	casos := map[string]func(*PublicacionGobiernoAlta){
		"organizacion": func(p *PublicacionGobiernoAlta) { p.OrganizacionRef = "organizacion:otra" },
		"centro":       func(p *PublicacionGobiernoAlta) { p.Solicitud.CentroRef = "centro:otro" },
		"contacto":     func(p *PublicacionGobiernoAlta) { p.Solicitud.ContactoRef = "contacto:otro" },
		"categoria":    func(p *PublicacionGobiernoAlta) { p.Solicitud.CategoriaRef = "categoria:otra" },
		"grupo":        func(p *PublicacionGobiernoAlta) { p.Solicitud.GrupoSubgrupo = "C1" },
		"motivo":       func(p *PublicacionGobiernoAlta) { p.Solicitud.MotivoClave = "vacante" },
		"detalle":      func(p *PublicacionGobiernoAlta) { p.Solicitud.Detalle = "Otro detalle" },
		"inicio": func(p *PublicacionGobiernoAlta) {
			p.Solicitud.Periodo.Inicio = p.Solicitud.Periodo.Inicio.AddDate(0, 0, 1)
		},
		"fin": func(p *PublicacionGobiernoAlta) {
			p.Solicitud.Periodo.Fin = p.Solicitud.Periodo.Fin.AddDate(0, 0, -1)
		},
		"rc existe": func(p *PublicacionGobiernoAlta) { p.Solicitud.RC.Existe = false },
		"rc numero": func(p *PublicacionGobiernoAlta) { p.Solicitud.RC.Numero = "rc:2026:0002" },
		"rc fecha": func(p *PublicacionGobiernoAlta) {
			p.Solicitud.RC.Fecha = p.Solicitud.RC.Fecha.AddDate(0, 0, -1)
		},
		"rc importe": func(p *PublicacionGobiernoAlta) { p.Solicitud.RC.Importe.Centimos++ },
		"rc moneda":  func(p *PublicacionGobiernoAlta) { p.Solicitud.RC.Importe.Moneda = "USD" },
		"rc documento": func(p *PublicacionGobiernoAlta) {
			p.Solicitud.RC.DocumentoRef = "documento:rc:0002"
		},
		"documentos reordenados": func(p *PublicacionGobiernoAlta) {
			p.Solicitud.DocumentosAdjuntos[0], p.Solicitud.DocumentosAdjuntos[1] =
				p.Solicitud.DocumentosAdjuntos[1], p.Solicitud.DocumentosAdjuntos[0]
		},
		"documento mutado": func(p *PublicacionGobiernoAlta) {
			p.Solicitud.DocumentosAdjuntos[0] = "documento:solicitud:002"
		},
		"observaciones": func(p *PublicacionGobiernoAlta) {
			p.Solicitud.Observaciones = "Otra observacion"
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			publicacion := publicacionGobiernoAltaPrueba()
			alterar(&publicacion)
			comprobarPublicacionGobiernoAltaRechazada(t, publicacion)
		})
	}
}

func TestGobiernoAltaRechazaCadaDecisionGobernadaInvalida(t *testing.T) {
	casos := map[string]func(*PublicacionGobiernoAlta){
		"flujo referencia": func(p *PublicacionGobiernoAlta) {
			p.Configuracion.Flujo.DefinicionRef = ""
		},
		"flujo version": func(p *PublicacionGobiernoAlta) { p.Configuracion.Flujo.Version = 0 },
		"flujo huella": func(p *PublicacionGobiernoAlta) {
			p.Configuracion.Flujo.HuellaSHA256 = strings.Repeat("0", 64)
		},
		"fase":   func(p *PublicacionGobiernoAlta) { p.Configuracion.FaseInicial = "" },
		"unidad": func(p *PublicacionGobiernoAlta) { p.Configuracion.UnidadInicialRef = "" },
		"accion": func(p *PublicacionGobiernoAlta) { p.Configuracion.AccionInicial = "" },
		"motivo catalogo": func(p *PublicacionGobiernoAlta) {
			p.MotivoAutorizacion.CatalogoID = ""
		},
		"motivo version": func(p *PublicacionGobiernoAlta) {
			p.MotivoAutorizacion.CatalogoVersion = 0
		},
		"motivo version fuera de limite": func(p *PublicacionGobiernoAlta) {
			p.MotivoAutorizacion.CatalogoVersion = int(MaximoEnteroSeguroOperacionAnalisis) + 1
		},
		"motivo huella": func(p *PublicacionGobiernoAlta) {
			p.MotivoAutorizacion.CatalogoHuellaSHA256 = strings.Repeat("0", 64)
		},
		"motivo entrada": func(p *PublicacionGobiernoAlta) {
			p.MotivoAutorizacion.EntradaClave = ""
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			publicacion := publicacionGobiernoAltaPrueba()
			alterar(&publicacion)
			comprobarPublicacionGobiernoAltaRechazada(t, publicacion)
		})
	}
}

func TestGobiernoAltaAplicaVentanaInclusivaExclusiva(t *testing.T) {
	base := publicacionGobiernoAltaPrueba()
	casos := map[string]struct {
		instante time.Time
		alterar  func(*PublicacionGobiernoAlta)
		valida   bool
	}{
		"inicio inclusivo": {instante: base.VigenteDesde, valida: true},
		"fin exclusivo":    {instante: base.VigenteHasta},
		"antes":            {instante: base.VigenteDesde.Add(-time.Microsecond)},
		"publicada al inicio": {
			instante: base.VigenteDesde,
			alterar:  func(p *PublicacionGobiernoAlta) { p.PublicadaEn = p.VigenteDesde },
			valida:   true,
		},
		"publicada despues": {
			instante: base.VigenteDesde,
			alterar:  func(p *PublicacionGobiernoAlta) { p.PublicadaEn = p.VigenteDesde.Add(time.Microsecond) },
		},
		"ventana vacia": {
			instante: base.VigenteDesde,
			alterar:  func(p *PublicacionGobiernoAlta) { p.VigenteHasta = p.VigenteDesde },
		},
		"publicacion no canonica": {
			instante: base.VigenteDesde,
			alterar:  func(p *PublicacionGobiernoAlta) { p.PublicadaEn = p.PublicadaEn.Add(time.Nanosecond) },
		},
		"inicio no canonico": {
			instante: base.VigenteDesde,
			alterar:  func(p *PublicacionGobiernoAlta) { p.VigenteDesde = p.VigenteDesde.Add(time.Nanosecond) },
		},
		"fin no canonico": {
			instante: base.VigenteDesde,
			alterar:  func(p *PublicacionGobiernoAlta) { p.VigenteHasta = p.VigenteHasta.Add(time.Nanosecond) },
		},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			publicacion := publicacionGobiernoAltaPrueba()
			if caso.alterar != nil {
				caso.alterar(&publicacion)
			}
			solicitud := nuevaSolicitudGobiernoAltaPrueba(t, caso.instante)
			autoridad := &autoridadGobiernoAltaPrueba{resolver: func(
				context.Context, SolicitudGobiernoAlta,
			) (PublicacionGobiernoAlta, error) {
				return publicacion, nil
			}}
			instantanea, err := ResolverGobiernoAltaSeguro(
				context.Background(), autoridad, solicitud,
			)
			if caso.valida {
				if err != nil || instantanea.ValidarPara(solicitud, caso.instante) != nil {
					t.Fatalf("limite valido rechazado: %v", err)
				}
				return
			}
			if !errors.Is(err, ErrGobiernoAltaNoDisponible) {
				t.Fatalf("limite invalido aceptado: %#v, %v", instantanea, err)
			}
		})
	}

	solicitud, instantanea, _ := resolverGobiernoAltaPrueba(t, base)
	for nombre, instante := range map[string]time.Time{
		"antes": base.VigenteDesde.Add(-time.Microsecond),
		"fin":   base.VigenteHasta,
	} {
		t.Run("revalidar "+nombre, func(t *testing.T) {
			if err := instantanea.ValidarPara(solicitud, instante); !errors.Is(
				err, ErrGobiernoAltaNoDisponible,
			) {
				t.Fatalf("revalidacion fuera de ventana: %v", err)
			}
		})
	}
	otraSolicitudCentro := solicitudCentroGobiernoAltaPrueba()
	otraSolicitudCentro.Detalle = "Necesidad temporal distinta"
	otraSolicitud, err := NuevaSolicitudGobiernoAlta(
		"organizacion:diputacion-granada",
		otraSolicitudCentro,
		instanteGobiernoAltaPrueba,
	)
	if err != nil {
		t.Fatalf("otra solicitud nominal: %v", err)
	}
	if err := instantanea.ValidarPara(
		otraSolicitud, instanteGobiernoAltaPrueba,
	); !errors.Is(err, ErrGobiernoAltaNoDisponible) {
		t.Fatalf("la instantanea acepto otra solicitud: %v", err)
	}
	otroInstante := nuevaSolicitudGobiernoAltaPrueba(
		t, instanteGobiernoAltaPrueba.Add(time.Microsecond),
	)
	if err := instantanea.ValidarPara(
		otroInstante, instanteGobiernoAltaPrueba.Add(time.Microsecond),
	); !errors.Is(err, ErrGobiernoAltaNoDisponible) {
		t.Fatalf("la instantanea acepto otro instante de gobierno: %v", err)
	}
}

func TestGobiernoAltaNulosCancelacionYErrorSensible(t *testing.T) {
	solicitud := nuevaSolicitudGobiernoAltaPrueba(t, instanteGobiernoAltaPrueba)
	autoridadNoLlamada := autoridadGobiernoAltaQueNoDebeLlamarse(t)

	_, err := ResolverGobiernoAltaSeguro(nil, autoridadNoLlamada, solicitud)
	if !errors.Is(err, ErrGobiernoAltaNoDisponible) || autoridadNoLlamada.llamadas.Load() != 0 {
		t.Fatalf("contexto nil no fallo cerrado: %v", err)
	}
	var contextoTipado *contextoGobiernoAltaNulo
	_, err = ResolverGobiernoAltaSeguro(contextoTipado, autoridadNoLlamada, solicitud)
	if !errors.Is(err, ErrGobiernoAltaNoDisponible) || autoridadNoLlamada.llamadas.Load() != 0 {
		t.Fatalf("contexto nil tipado no fallo cerrado: %v", err)
	}
	_, err = ResolverGobiernoAltaSeguro(context.Background(), nil, solicitud)
	if !errors.Is(err, ErrGobiernoAltaNoDisponible) {
		t.Fatalf("autoridad nil no fallo cerrada: %v", err)
	}
	var autoridadTipada *autoridadGobiernoAltaPrueba
	_, err = ResolverGobiernoAltaSeguro(context.Background(), autoridadTipada, solicitud)
	if !errors.Is(err, ErrGobiernoAltaNoDisponible) {
		t.Fatalf("autoridad nil tipada no fallo cerrada: %v", err)
	}

	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	_, err = ResolverGobiernoAltaSeguro(ctxCancelado, autoridadNoLlamada, solicitud)
	if !errors.Is(err, context.Canceled) || autoridadNoLlamada.llamadas.Load() != 0 {
		t.Fatalf("cancelacion previa invoco autoridad: llamadas=%d err=%v", autoridadNoLlamada.llamadas.Load(), err)
	}

	ctxDurante, cancelarDurante := context.WithCancel(context.Background())
	autoridadDurante := &autoridadGobiernoAltaPrueba{resolver: func(
		context.Context, SolicitudGobiernoAlta,
	) (PublicacionGobiernoAlta, error) {
		cancelarDurante()
		return publicacionGobiernoAltaPrueba(), errors.New("dsn=secreto-privado")
	}}
	_, err = ResolverGobiernoAltaSeguro(ctxDurante, autoridadDurante, solicitud)
	if !errors.Is(err, context.Canceled) || autoridadDurante.llamadas.Load() != 1 ||
		strings.Contains(err.Error(), "secreto-privado") {
		t.Fatalf("cancelacion durante autoridad no saneada: llamadas=%d err=%v", autoridadDurante.llamadas.Load(), err)
	}

	autoridadRemota := &autoridadGobiernoAltaPrueba{resolver: func(
		context.Context, SolicitudGobiernoAlta,
	) (PublicacionGobiernoAlta, error) {
		return PublicacionGobiernoAlta{}, errors.New("token=material-sensible")
	}}
	_, err = ResolverGobiernoAltaSeguro(context.Background(), autoridadRemota, solicitud)
	if !errors.Is(err, ErrGobiernoAltaNoDisponible) || autoridadRemota.llamadas.Load() != 1 ||
		strings.Contains(err.Error(), "material-sensible") {
		t.Fatalf("error remoto filtrado o reintentado: llamadas=%d err=%v", autoridadRemota.llamadas.Load(), err)
	}
}

func TestInstantaneaGobiernoAltaCopiaConcurrenteYForja(t *testing.T) {
	solicitud, instantanea, _ := resolverGobiernoAltaPrueba(
		t, publicacionGobiernoAltaPrueba(),
	)
	const concurrentes = 64
	errores := make(chan error, concurrentes)
	var grupo sync.WaitGroup
	for indice := 0; indice < concurrentes; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			datos, err := instantanea.Datos()
			if err != nil {
				errores <- err
				return
			}
			datos.Solicitud.DocumentosAdjuntos[0] = "documento:mutado:concurrente"
			if err := instantanea.ValidarPara(solicitud, instanteGobiernoAltaPrueba); err != nil {
				errores <- err
			}
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Fatalf("lectura concurrente: %v", err)
	}
	datos, err := instantanea.Datos()
	if err != nil || datos.Solicitud.DocumentosAdjuntos[0] != "documento:solicitud:001" {
		t.Fatalf("la copia concurrente altero el estado: %#v, %v", datos, err)
	}
	if _, err := (InstantaneaGobiernoAlta{}).Datos(); !errors.Is(err, ErrGobiernoAltaNoDisponible) {
		t.Fatalf("valor cero aceptado: %v", err)
	}

	for nombre, alterar := range map[string]func(*DatosInstantaneaGobiernoAlta){
		"organizacion": func(d *DatosInstantaneaGobiernoAlta) { d.OrganizacionRef = "" },
		"solicitud":    func(d *DatosInstantaneaGobiernoAlta) { d.Solicitud.CentroRef = "" },
		"flujo":        func(d *DatosInstantaneaGobiernoAlta) { d.Configuracion.Flujo.Version = 0 },
		"fase":         func(d *DatosInstantaneaGobiernoAlta) { d.Configuracion.FaseInicial = "" },
		"unidad":       func(d *DatosInstantaneaGobiernoAlta) { d.Configuracion.UnidadInicialRef = "" },
		"accion":       func(d *DatosInstantaneaGobiernoAlta) { d.Configuracion.AccionInicial = "" },
		"motivo": func(d *DatosInstantaneaGobiernoAlta) {
			d.MotivoAutorizacion.EntradaClave = ""
		},
		"instante": func(d *DatosInstantaneaGobiernoAlta) { d.InstanteGobierno = time.Time{} },
		"vigencia": func(d *DatosInstantaneaGobiernoAlta) { d.VigenteHasta = d.VigenteDesde },
	} {
		t.Run("forja "+nombre, func(t *testing.T) {
			forjados := datos
			forjados.Solicitud.DocumentosAdjuntos = append(
				[]string(nil), datos.Solicitud.DocumentosAdjuntos...,
			)
			alterar(&forjados)
			forjada := InstantaneaGobiernoAlta{datos: &forjados}
			if _, err := forjada.Datos(); !errors.Is(err, ErrGobiernoAltaNoDisponible) {
				t.Fatalf("forja aceptada: %v", err)
			}
		})
	}
}

func TestInstantaneaGobiernoAltaEsPrivadaYOpaca(t *testing.T) {
	_, instantanea, _ := resolverGobiernoAltaPrueba(t, publicacionGobiernoAltaPrueba())
	datos, err := instantanea.Datos()
	if err != nil {
		t.Fatalf("datos: %v", err)
	}
	for nombre, valor := range map[string]any{
		"instantanea":      instantanea,
		"instantanea ptr":  &instantanea,
		"datos":            datos,
		"datos ptr":        &datos,
		"instantanea cero": InstantaneaGobiernoAlta{},
		"datos cero":       DatosInstantaneaGobiernoAlta{},
	} {
		t.Run(nombre, func(t *testing.T) {
			comprobarSerializacionGobiernoAltaProhibida(t, valor)
			formato := fmt.Sprintf("%v|%+v|%#v|%s|%q|%x", valor, valor, valor, valor, valor, valor)
			if strings.Contains(formato, "diputacion") ||
				strings.Contains(formato, "servicios-sociales") ||
				!strings.Contains(formato, redaccionGobiernoAlta) {
				t.Fatalf("formato no opaco: %s", formato)
			}
		})
	}
	for nombre, destino := range map[string]any{
		"instantanea":      &instantanea,
		"datos":            &datos,
		"instantanea cero": &InstantaneaGobiernoAlta{},
		"datos cero":       &DatosInstantaneaGobiernoAlta{},
	} {
		t.Run(nombre+" decode", func(t *testing.T) {
			comprobarDeserializacionGobiernoAltaProhibida(t, destino)
		})
	}
	var instantaneaNil *InstantaneaGobiernoAlta
	var datosNil *DatosInstantaneaGobiernoAlta
	for nombre, destino := range map[string]decodificadoresGobiernoAlta{
		"instantanea nil tipada": instantaneaNil,
		"datos nil tipados":      datosNil,
	} {
		t.Run(nombre, func(t *testing.T) {
			comprobarDecodificadoresGobiernoAlta(t, destino)
		})
	}
	var registro bytes.Buffer
	slog.New(slog.NewJSONHandler(&registro, nil)).Info("gobierno", "instantanea", instantanea)
	if strings.Contains(registro.String(), "diputacion") ||
		!strings.Contains(registro.String(), redaccionGobiernoAlta) {
		t.Fatalf("log no opaco: %s", registro.String())
	}

	tipo := reflect.TypeOf(InstantaneaGobiernoAlta{})
	for indice := 0; indice < tipo.NumField(); indice++ {
		if tipo.Field(indice).IsExported() {
			t.Fatalf("estado exportado por reflexion: %s", tipo.Field(indice).Name)
		}
	}
	for _, decision := range []string{
		"Flujo", "FaseInicial", "UnidadInicialRef", "AccionInicial", "MotivoAutorizacion",
	} {
		if _, existe := reflect.TypeOf(DatosSolicitudGobiernoAlta{}).FieldByName(decision); existe {
			t.Fatalf("el cliente puede aportar la decision %s", decision)
		}
	}
	if reflect.TypeOf(PublicacionGobiernoAlta{}).NumMethod() != 0 {
		t.Fatal("la publicacion DTO adquirio metodos de autoridad")
	}
}

func TestGobiernoAltaMantieneFronteraEstructural(t *testing.T) {
	_, archivo, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no se localizo la prueba")
	}
	produccion := strings.TrimSuffix(archivo, "_test.go") + ".go"
	contenido, err := os.ReadFile(produccion)
	if err != nil {
		t.Fatalf("leer fuente: %v", err)
	}
	texto := string(contenido)
	for _, prohibido := range []string{
		"/application", "/adapters", "net/http", "database/sql", "Instante" + "Efecto",
	} {
		if strings.Contains(texto, prohibido) {
			t.Fatalf("dependencia o acoplamiento prohibido %q", prohibido)
		}
	}
	arbol, err := parser.ParseFile(token.NewFileSet(), produccion, contenido, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("analizar imports: %v", err)
	}
	for _, importacion := range arbol.Imports {
		ruta := strings.Trim(importacion.Path.Value, `"`)
		if strings.Contains(ruta, "contrataciontemporal/application") ||
			strings.Contains(ruta, "contrataciontemporal/adapters") {
			t.Fatalf("import hexagonal prohibido: %s", ruta)
		}
	}
	lineas := bytes.Count(contenido, []byte{'\n'})
	if lineas >= 500 {
		t.Fatalf("fuente fuera del objetivo preferente: %d lineas", lineas)
	}
}

func comprobarPublicacionGobiernoAltaRechazada(
	t *testing.T,
	publicacion PublicacionGobiernoAlta,
) {
	t.Helper()
	solicitud := nuevaSolicitudGobiernoAltaPrueba(t, instanteGobiernoAltaPrueba)
	autoridad := &autoridadGobiernoAltaPrueba{resolver: func(
		context.Context, SolicitudGobiernoAlta,
	) (PublicacionGobiernoAlta, error) {
		return publicacion, nil
	}}
	instantanea, err := ResolverGobiernoAltaSeguro(
		context.Background(), autoridad, solicitud,
	)
	if !errors.Is(err, ErrGobiernoAltaNoDisponible) || autoridad.llamadas.Load() != 1 ||
		!reflect.ValueOf(instantanea).IsZero() {
		t.Fatalf("publicacion invalida aceptada: llamadas=%d valor=%#v err=%v", autoridad.llamadas.Load(), instantanea, err)
	}
}

func autoridadGobiernoAltaQueNoDebeLlamarse(t *testing.T) *autoridadGobiernoAltaPrueba {
	t.Helper()
	return &autoridadGobiernoAltaPrueba{resolver: func(
		context.Context, SolicitudGobiernoAlta,
	) (PublicacionGobiernoAlta, error) {
		t.Fatal("se invoco la autoridad")
		return PublicacionGobiernoAlta{}, nil
	}}
}

type codificadoresGobiernoAlta interface {
	MarshalText() ([]byte, error)
	MarshalBinary() ([]byte, error)
	GobEncode() ([]byte, error)
}

type decodificadoresGobiernoAlta interface {
	UnmarshalJSON([]byte) error
	UnmarshalXML(*xml.Decoder, xml.StartElement) error
	UnmarshalText([]byte) error
	UnmarshalBinary([]byte) error
	GobDecode([]byte) error
}

func comprobarSerializacionGobiernoAltaProhibida(t *testing.T, valor any) {
	t.Helper()
	_, err := json.Marshal(valor)
	if !errors.Is(err, ErrSerializacionGobiernoAltaProhibida) {
		t.Fatalf("JSON no bloqueado para %T: %v", valor, err)
	}
	_, err = xml.Marshal(valor)
	if !errors.Is(err, ErrSerializacionGobiernoAltaProhibida) {
		t.Fatalf("XML no bloqueado para %T: %v", valor, err)
	}
	codec, ok := valor.(codificadoresGobiernoAlta)
	if !ok {
		t.Fatalf("%T no expone bloqueos directos", valor)
	}
	_, err = codec.MarshalText()
	if !errors.Is(err, ErrSerializacionGobiernoAltaProhibida) {
		t.Fatalf("Text no bloqueado para %T: %v", valor, err)
	}
	_, err = codec.MarshalBinary()
	if !errors.Is(err, ErrSerializacionGobiernoAltaProhibida) {
		t.Fatalf("Binary no bloqueado para %T: %v", valor, err)
	}
	_, err = codec.GobEncode()
	if !errors.Is(err, ErrSerializacionGobiernoAltaProhibida) {
		t.Fatalf("Gob directo no bloqueado para %T: %v", valor, err)
	}
	var destino bytes.Buffer
	err = gob.NewEncoder(&destino).Encode(valor)
	if !errors.Is(err, ErrSerializacionGobiernoAltaProhibida) {
		t.Fatalf("Gob no bloqueado para %T: %v", valor, err)
	}
}

func comprobarDeserializacionGobiernoAltaProhibida(t *testing.T, destino any) {
	t.Helper()
	err := json.Unmarshal([]byte(`{}`), destino)
	if !errors.Is(err, ErrSerializacionGobiernoAltaProhibida) {
		t.Fatalf("JSON decode no bloqueado para %T: %v", destino, err)
	}
	err = xml.Unmarshal([]byte(`<gobierno/>`), destino)
	if !errors.Is(err, ErrSerializacionGobiernoAltaProhibida) {
		t.Fatalf("XML decode no bloqueado para %T: %v", destino, err)
	}
	codec, ok := destino.(decodificadoresGobiernoAlta)
	if !ok {
		t.Fatalf("%T no expone bloqueos de reconstruccion", destino)
	}
	comprobarDecodificadoresGobiernoAlta(t, codec)
}

func comprobarDecodificadoresGobiernoAlta(
	t *testing.T,
	codec decodificadoresGobiernoAlta,
) {
	t.Helper()
	for nombre, err := range map[string]error{
		"JSON":   codec.UnmarshalJSON(nil),
		"XML":    codec.UnmarshalXML(nil, xml.StartElement{}),
		"Text":   codec.UnmarshalText(nil),
		"Binary": codec.UnmarshalBinary(nil),
		"Gob":    codec.GobDecode(nil),
	} {
		if !errors.Is(err, ErrSerializacionGobiernoAltaProhibida) {
			t.Fatalf("%s decode no bloqueado: %v", nombre, err)
		}
	}
}

type contextoGobiernoAltaNulo struct{}

func (*contextoGobiernoAltaNulo) Deadline() (time.Time, bool) { return time.Time{}, false }
func (*contextoGobiernoAltaNulo) Done() <-chan struct{}       { return nil }
func (*contextoGobiernoAltaNulo) Err() error                  { return nil }
func (*contextoGobiernoAltaNulo) Value(any) any               { return nil }
