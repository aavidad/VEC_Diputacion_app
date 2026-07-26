package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func nuevoEscenarioConjuntosViasC1(
	t *testing.T,
	peticionCompartida string,
	reciboCompartido string,
) escenarioConjuntosViasC1 {
	t.Helper()
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	catalogo := catalogoGlobalC1(
		t,
		entorno.inicio,
		"catalogo_cobertura_global_c1",
		1,
	)
	return crearEscenarioConCatalogoC1(
		t, entorno, catalogo, peticionCompartida, reciboCompartido,
	)
}

func crearEscenarioConCatalogoC1(
	t *testing.T,
	entorno *entornoCoberturaAplicacionPrueba,
	catalogo domain.CatalogoViasCobertura,
	peticionCompartida string,
	reciboCompartido string,
) escenarioConjuntosViasC1 {
	t.Helper()
	politica := politicaGlobalC1(
		t,
		catalogo,
		entorno.inicio,
		"politica_cobertura_global_c1",
	)
	entorno.catalogo = catalogo
	entorno.publicador.publicar = func(
		context.Context,
		ports.SolicitudConsultarCobertura,
	) (ports.ConfirmacionPublicacionCobertura, error) {
		return ports.NuevaConfirmacionPublicacionCobertura(
			entorno.publicador.identidad.AutoridadRef(),
			catalogo.Publicacion(),
			entorno.reloj.Ahora(),
		)
	}
	vias := catalogo.Vias()
	solicitudes := make([]ports.SolicitudConsultarCobertura, len(vias))
	conjuntos := make([]cobertura.ConjuntoEvidenciasCobertura, len(vias))
	for indice, via := range vias {
		peticion := fmt.Sprintf("peticion_global_via_%d_012345", indice+1)
		recibo := fmt.Sprintf("recibo_global_via_%d_012345", indice+1)
		if peticionCompartida != "" {
			peticion = peticionCompartida
		}
		if reciboCompartido != "" {
			recibo = reciboCompartido
		}
		solicitud := entorno.solicitud
		solicitud.PeticionRef = peticion
		solicitud.Catalogo = catalogo.Identidad()
		solicitud.ViaClave = via.Clave
		solicitud.Comprobacion = via.Comprobaciones[0]
		evidencia := prepararEvidenciaCoberturaPrueba(
			t,
			entorno,
			solicitud,
			recibo,
			domain.ComprobacionAfirmativa,
		)
		conjuntos[indice] = nuevoConjuntoCoberturaPrueba(
			t,
			coordenadasCoberturaPrueba(solicitud, politica),
			catalogo,
			politica,
			[]cobertura.EvidenciaConsultaCobertura{evidencia},
			entorno.reloj.Ahora(),
		)
		solicitudes[indice] = solicitud
	}
	return escenarioConjuntosViasC1{
		entorno: entorno, catalogo: catalogo, politica: politica,
		solicitudes: solicitudes, conjuntos: conjuntos,
		instante: entorno.reloj.Ahora(),
	}
}

func catalogoGlobalC1(
	t *testing.T,
	inicio time.Time,
	referencia string,
	version uint64,
) domain.CatalogoViasCobertura {
	t.Helper()
	vias := []domain.DefinicionViaCobertura{
		{
			Clave: "via_primaria", Orden: 1,
			Comprobaciones: []domain.ComprobacionExigibleCobertura{{
				Clave: "comprobacion_primaria", Orden: 1, Obligatoria: true,
				Procedencia: domain.ProcedenciaComprobacionCobertura{
					Clave:               "fuente_primaria",
					DefinicionFuenteRef: "fuente_definicion_bolsa_v3",
				},
			}},
		},
		{
			Clave: "via_secundaria", Orden: 2,
			Comprobaciones: []domain.ComprobacionExigibleCobertura{{
				Clave: "comprobacion_secundaria", Orden: 1, Obligatoria: true,
				Procedencia: domain.ProcedenciaComprobacionCobertura{
					Clave:               "fuente_secundaria",
					DefinicionFuenteRef: "fuente_definicion_bolsa_v3",
				},
			}},
		},
	}
	return publicarCatalogoGlobalC1(t, inicio, referencia, version, vias)
}

func publicarCatalogoGlobalC1(
	t *testing.T,
	inicio time.Time,
	referencia string,
	version uint64,
	vias []domain.DefinicionViaCobertura,
) domain.CatalogoViasCobertura {
	t.Helper()
	catalogo, err := domain.PublicarCatalogoViasCobertura(
		domain.BorradorCatalogoViasCobertura{
			Referencia: referencia, Version: version,
			PublicadoEn: inicio.Add(-time.Hour),
			Vigencia: domain.VigenciaCatalogoCobertura{
				Desde: inicio.Add(-time.Hour), Hasta: inicio.Add(time.Hour),
			},
			ProcedenciaRef: "procedimiento_catalogo_global_c1",
			Vias:           vias,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return catalogo
}

func politicaGlobalC1(
	t *testing.T,
	catalogo domain.CatalogoViasCobertura,
	inicio time.Time,
	referencia string,
) domain.PoliticaDecisionCobertura {
	t.Helper()
	viasCatalogo := catalogo.Vias()
	reglas := make([]domain.ReglaViaDecisionCobertura, len(viasCatalogo))
	for indice, via := range viasCatalogo {
		reglas[indice] = domain.ReglaViaDecisionCobertura{
			ViaClave:  via.Clave,
			Prioridad: uint16(len(viasCatalogo) - indice),
			Comprobaciones: []domain.ReglaComprobacionDecisionCobertura{{
				Clave: via.Comprobaciones[0].Clave,
				ResultadosHabilitantes: []domain.ResultadoComprobacion{
					domain.ComprobacionAfirmativa,
				},
				TratamientoAusencia: domain.AusenciaCoberturaBloquea,
			}},
		}
	}
	politica, err := domain.PublicarPoliticaDecisionCobertura(
		domain.BorradorPoliticaDecisionCobertura{
			Referencia: referencia, Version: 1,
			Catalogo:        catalogo.Identidad(),
			OrganizacionRef: organizacionCoberturaPrueba,
			FinalidadClave:  finalidadCoberturaClave,
			FinalidadRef:    finalidadCoberturaRef,
			PublicadaEn:     inicio.Add(-45 * time.Minute),
			Vigencia: domain.VigenciaCatalogoCobertura{
				Desde: inicio.Add(-30 * time.Minute),
				Hasta: inicio.Add(30 * time.Minute),
			},
			ProcedenciaRef: "procedimiento_politica_global_c1",
			Vias:           reglas,
		},
		catalogo,
	)
	if err != nil {
		t.Fatal(err)
	}
	return politica
}

func datosGlobalesC1(
	escenario escenarioConjuntosViasC1,
	conjuntos []cobertura.ConjuntoEvidenciasCobertura,
) cobertura.DatosPrepararConjuntosViasCobertura {
	return cobertura.DatosPrepararConjuntosViasCobertura{
		AnalisisRef:          analisisGlobalRefC1,
		AnalisisHuellaSHA256: huellaAnalisisC1,
		Catalogo:             escenario.catalogo,
		Politica:             escenario.politica,
		Conjuntos: append(
			[]cobertura.ConjuntoEvidenciasCobertura(nil),
			conjuntos...,
		),
		PreparadaEn: escenario.instante,
	}
}

func prepararGlobalC1(
	t *testing.T,
	escenario escenarioConjuntosViasC1,
	conjuntos []cobertura.ConjuntoEvidenciasCobertura,
) cobertura.PreparacionConjuntosViasCobertura {
	t.Helper()
	preparacion, err := cobertura.PrepararConjuntosViasCobertura(
		datosGlobalesC1(escenario, conjuntos),
	)
	if err != nil {
		t.Fatal(err)
	}
	return preparacion
}

func exigirPreparacionGlobalC1Rechazada(
	t *testing.T,
	datos cobertura.DatosPrepararConjuntosViasCobertura,
) {
	t.Helper()
	_, err := cobertura.PrepararConjuntosViasCobertura(datos)
	if !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("preparacion no confiable aceptada: %v", err)
	}
}

func comprobarCeroConsumoGlobalC1(
	t *testing.T,
	entorno *entornoCoberturaAplicacionPrueba,
) {
	t.Helper()
	entorno.consumidor.mu.Lock()
	defer entorno.consumidor.mu.Unlock()
	if len(entorno.consumidor.ordenes) != 0 ||
		len(entorno.consumidor.registros) != 0 {
		t.Fatal("la preparacion global ha consumido evidencia")
	}
}
