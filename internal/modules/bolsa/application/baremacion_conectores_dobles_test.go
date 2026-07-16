package application

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type selladorTiempoBaremacionPrueba struct{}

func (selladorTiempoBaremacionPrueba) SellarTiempoFirma(
	context.Context,
	puertosbolsa.SolicitudSellarTiempoFirma,
) (puertosbolsa.SelloTiempoFirma, error) {
	return puertosbolsa.SelloTiempoFirma{}, puertosbolsa.ErrSelloTiempoNoDisponible
}

type aumentadorBaremacionPrueba struct{}

func (aumentadorBaremacionPrueba) AumentarFirma(
	context.Context,
	puertosbolsa.SolicitudAumentarFirma,
) (puertosbolsa.ResultadoAumentoFirma, error) {
	return puertosbolsa.ResultadoAumentoFirma{}, puertosbolsa.ErrAumentoFirmaNoDisponible
}

type selladorTiempoActivoBaremacionPrueba struct {
	ahora    time.Time
	llamadas int
}

func (s *selladorTiempoActivoBaremacionPrueba) SellarTiempoFirma(
	_ context.Context,
	solicitud puertosbolsa.SolicitudSellarTiempoFirma,
) (puertosbolsa.SelloTiempoFirma, error) {
	if solicitud.Validar() != nil {
		return puertosbolsa.SelloTiempoFirma{}, puertosbolsa.ErrSelloTiempoNoDisponible
	}
	s.llamadas++
	return puertosbolsa.SelloTiempoFirma{
		SelloTiempoRef: "sello-tiempo:firma:1", HuellaSelloTiempoSHA256: huellaBaremacionPrueba("5"),
		ObjetoRef: solicitud.ObjetoRef, HuellaObjetoSHA256: solicitud.HuellaObjetoSHA256,
		PoliticaSelloTiempoRef:          solicitud.Politica.PoliticaSelloTiempoRef,
		PoliticaSelloTiempoVersion:      solicitud.Politica.PoliticaSelloTiempoVersion,
		HuellaPoliticaSelloTiempoSHA256: solicitud.Politica.HuellaPoliticaSelloTiempoSHA256,
		ValidacionSelloRef:              "validacion:sello-tiempo:1", HuellaValidacionSHA256: huellaBaremacionPrueba("6"),
		SelladoEn: s.ahora,
	}, nil
}

type aumentadorActivoBaremacionPrueba struct {
	ahora    time.Time
	llamadas int
}

func (a *aumentadorActivoBaremacionPrueba) AumentarFirma(
	_ context.Context,
	solicitud puertosbolsa.SolicitudAumentarFirma,
) (puertosbolsa.ResultadoAumentoFirma, error) {
	if solicitud.Validar() != nil {
		return puertosbolsa.ResultadoAumentoFirma{}, puertosbolsa.ErrAumentoFirmaNoDisponible
	}
	a.llamadas++
	return puertosbolsa.ResultadoAumentoFirma{
		Artefacto: solicitud.Artefacto, NivelAlcanzadoClave: solicitud.Politica.NivelAumentoClave,
		PoliticaLongevidadRef:          solicitud.Politica.PoliticaLongevidadRef,
		PoliticaLongevidadVersion:      solicitud.Politica.PoliticaLongevidadVersion,
		HuellaPoliticaLongevidadSHA256: solicitud.Politica.HuellaPoliticaLongevidadSHA256,
		EvidenciaAumentoRef:            "evidencia:aumento:firma:1", HuellaEvidenciaSHA256: huellaBaremacionPrueba("7"),
		AumentadaEn: a.ahora,
	}, nil
}

type selladorSolicitudBaremacionPrueba struct{}

func (selladorSolicitudBaremacionPrueba) SellarSolicitudBaremacion(
	_ context.Context,
	carga puertosbolsa.CargaProtegida,
) (string, error) {
	if carga.Validar() != nil {
		return "", puertosbolsa.ErrCargaProtegidaInvalida
	}
	huella := sha256.Sum256(carga.Revelar())
	return "hmac-sha256:baremacion_1:" + hex.EncodeToString(huella[:]), nil
}

type seudonimizadorBaremacionPrueba struct{}

func (seudonimizadorBaremacionPrueba) SeudonimizarSujetoAlmacen(
	_ context.Context,
	s puertosvec.SolicitudSeudonimizarSujetoAlmacen,
) (string, error) {
	sujeto, ambito, err := s.RevelarParaSellado()
	if err != nil {
		return "", err
	}
	huella := sha256.Sum256([]byte(ambito + "\x00" + sujeto))
	return "hmac-sha256:seudonimo_1:" + hex.EncodeToString(huella[:]), nil
}

type seudonimizadorContadorBaremacionPrueba struct{ llamadas int }

func (s *seudonimizadorContadorBaremacionPrueba) SeudonimizarSujetoAlmacen(
	ctx context.Context,
	solicitud puertosvec.SolicitudSeudonimizarSujetoAlmacen,
) (string, error) {
	s.llamadas++
	return seudonimizadorBaremacionPrueba{}.SeudonimizarSujetoAlmacen(ctx, solicitud)
}

type generadorBaremacionPrueba struct {
	mu      sync.Mutex
	efectos int
}

func (*generadorBaremacionPrueba) NuevoIDBaremacion() (string, error) {
	return "baremacion-generada-1", nil
}
func (g *generadorBaremacionPrueba) NuevoIDDecisionTecnica() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return "decision-tecnica-001", nil
}
func (*generadorBaremacionPrueba) NuevaReferenciaManifiestoProbatorio() (string, error) {
	return "manifiesto:probatorio:decision:001", nil
}
func (*generadorBaremacionPrueba) NuevaReferenciaCorrelacion() (string, error) {
	return "correlacion:generada:1", nil
}
func (g *generadorBaremacionPrueba) NuevaReferenciaEfectoAlmacen() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.efectos++
	return "efecto:almacen:baremacion:" + strconv.Itoa(g.efectos), nil
}

func tokenReservaBaremacionPrueba(t *testing.T) puertosbolsa.TokenReservaBaremacion {
	t.Helper()
	valor := base64.RawURLEncoding.EncodeToString([]byte("token-reserva-baremacion-32-bytes"))
	token, err := puertosbolsa.NuevoTokenReservaBaremacion(valor)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func huellaBaremacionPrueba(caracter string) string { return strings.Repeat(caracter, 64) }

func accionesBaremacionIguales(a, b []puertosbolsa.AccionOperacionBaremacion) bool {
	if len(a) != len(b) {
		return false
	}
	for indice := range a {
		if a[indice] != b[indice] {
			return false
		}
	}
	return true
}

var (
	_ puertosbolsa.RepositorioBaremaciones              = (*repositorioBaremacionPrueba)(nil)
	_ puertosbolsa.FuenteDatosBaremacion                = (*fuenteBaremacionPrueba)(nil)
	_ puertosbolsa.CalculadorOficialBaremacion          = (*calculadorBaremacionPrueba)(nil)
	_ puertosbolsa.CatalogoPoliticasFirmaBaremacion     = (*politicasBaremacionPrueba)(nil)
	_ puertosbolsa.CodificadorCanonicoDecision          = (*codificadorBaremacionPrueba)(nil)
	_ puertosbolsa.AlmacenDocumentosFirmables           = (*almacenBaremacionPrueba)(nil)
	_ puertosbolsa.FirmadorInteractivo                  = (*firmadorBaremacionPrueba)(nil)
	_ puertosbolsa.ValidadorFirmaServidor               = (*validadorBaremacionPrueba)(nil)
	_ puertosbolsa.SelladorTiempoFirma                  = selladorTiempoBaremacionPrueba{}
	_ puertosbolsa.AumentadorFirmaLongeva               = aumentadorBaremacionPrueba{}
	_ puertosbolsa.SelladorSolicitudBaremacion          = selladorSolicitudBaremacionPrueba{}
	_ puertosvec.SeudonimizadorSujetoAlmacen            = seudonimizadorBaremacionPrueba{}
	_ puertosbolsa.GeneradorReferenciasOpacasBaremacion = (*generadorBaremacionPrueba)(nil)
	_ puertosvec.Autorizador                            = (*autorizadorBaremacionPrueba)(nil)
	_ FuenteSesionAutenticadaBaremacion                 = (*sesionesBaremacionPrueba)(nil)
)
