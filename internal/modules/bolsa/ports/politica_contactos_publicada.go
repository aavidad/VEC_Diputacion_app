package ports

import (
	"context"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// FuentePoliticaContactos procede del gobierno de reglas de Bolsa, nunca
// del importador de días de Calendarios ni del cuerpo HTTP de un contacto.
type FuentePoliticaContactos struct {
	BolsaRef                                     string
	CatalogoRef, CatalogoHuellaSHA256            string
	TipoDia, SedeRef, Zona                       string
	DesdeMinuto, HastaMinuto                     int
	ControlFranja                                string
	IntentosPorCiclo, Ciclos, SeparacionSegundos int
	ControlSeparacion                            string
	ResultadosSinContacto                        []string
	HuellaFuenteSHA256                           string
}

type ConsultaPoliticaContactosGobernada interface {
	ObtenerPublicada(context.Context, string) (FuentePoliticaContactos, error)
}

type PublicacionPoliticaContactos interface {
	VersionActual(context.Context, string) (dominiobolsa.PoliticaContactosPublicada, bool, error)
	PublicarSiVersion(context.Context, OrdenPublicarPoliticaContactos) (ReciboPoliticaContactos, error)
}

type OrdenPublicarPoliticaContactos struct {
	Politica                               dominiobolsa.PoliticaContactosPublicada
	VersionEsperada                        uint64
	HuellaFuente                           string
	ActorRef, ClaveIdempotencia, ReciboRef string
	Material                               puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ReciboPoliticaContactos struct {
	ReciboRef, BolsaRef string
	Version             uint64
	HuellaSHA256        string
	Reutilizado         bool
}
