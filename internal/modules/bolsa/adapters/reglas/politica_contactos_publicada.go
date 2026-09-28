package reglas

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	vecreglas "vec-diputacion-granada/internal/vec/reglas"
)

var ErrFuentePoliticaContactosNoDisponible = errors.New("bolsa: fuente publicada de politica de contactos no disponible")

var _ puertosbolsa.ConsultaPoliticaContactosGobernada = (*FuentePoliticaContactos)(nil)

// FuentePoliticaContactos traduce el catálogo gobernado de Bolsa. Un control
// en modo aviso no se publica como política que impide contactos fuera de hora.
type FuentePoliticaContactos struct {
	intentos *IntentosContacto
	sedeRef  string
}

func NuevaFuentePoliticaContactos(intentos *IntentosContacto, sedeRef string) *FuentePoliticaContactos {
	return &FuentePoliticaContactos{intentos: intentos, sedeRef: sedeRef}
}

func (f *FuentePoliticaContactos) ObtenerPublicada(ctx context.Context, bolsaRef string) (puertosbolsa.FuentePoliticaContactos, error) {
	var vacio puertosbolsa.FuentePoliticaContactos
	if ctx == nil || f == nil || f.intentos == nil || f.sedeRef == "" || bolsaRef == "" {
		return vacio, ErrFuentePoliticaContactosNoDisponible
	}
	politica, reglas, configurada, err := f.intentos.PoliticaIntentosTelefonicos(ctx)
	if err != nil || !configurada || politica.Validar() != nil || politica.Franja.Zona == nil ||
		politica.Franja.Zona.String() != "Europe/Madrid" || !politica.Franja.SoloDiasHabiles ||
		politica.Franja.Control != dominiobolsa.ControlReglaImpedir ||
		politica.ControlSeparacion != dominiobolsa.ControlReglaImpedir {
		return vacio, ErrFuentePoliticaContactosNoDisponible
	}
	var franja puertosbolsa.ReglaIntentosContacto
	var huellaCatalogo string
	for _, regla := range reglas {
		if regla.Clave == vecreglas.BolsaFranjaLlamadas {
			franja = regla
		}
		if regla.HuellaCatalogo == "" || (huellaCatalogo != "" && regla.HuellaCatalogo != huellaCatalogo) {
			return vacio, ErrFuentePoliticaContactosNoDisponible
		}
		huellaCatalogo = regla.HuellaCatalogo
	}
	if franja.Referencia == "" || franja.HuellaCatalogo == "" {
		return vacio, ErrFuentePoliticaContactosNoDisponible
	}
	vacio = puertosbolsa.FuentePoliticaContactos{
		BolsaRef: bolsaRef, CatalogoRef: franja.Referencia,
		CatalogoHuellaSHA256: franja.HuellaCatalogo,
		TipoDia:              dominiobolsa.TipoCalendarioHabilSede, SedeRef: f.sedeRef,
		Zona: politica.Franja.Zona.String(), DesdeMinuto: politica.Franja.DesdeMinuto,
		HastaMinuto: politica.Franja.HastaMinuto, ControlFranja: politica.Franja.Control,
		IntentosPorCiclo: politica.IntentosPorProceso, Ciclos: politica.Procesos,
		SeparacionSegundos:    int(politica.SeparacionMinima / time.Second),
		ControlSeparacion:     politica.ControlSeparacion,
		ResultadosSinContacto: append([]string(nil), politica.ResultadosSinContacto...),
	}
	version := dominiobolsa.PoliticaContactosPublicada{
		Esquema: dominiobolsa.EsquemaPoliticaContactos, BolsaRef: vacio.BolsaRef, Version: 1,
		CatalogoRef: vacio.CatalogoRef, CatalogoHuellaSHA256: vacio.CatalogoHuellaSHA256,
		TipoDia: vacio.TipoDia, SedeRef: vacio.SedeRef, Zona: vacio.Zona,
		DesdeMinuto: vacio.DesdeMinuto, HastaMinuto: vacio.HastaMinuto,
		ControlFranja: vacio.ControlFranja, IntentosPorCiclo: vacio.IntentosPorCiclo,
		Ciclos: vacio.Ciclos, SeparacionSegundos: vacio.SeparacionSegundos,
		ControlSeparacion:     vacio.ControlSeparacion,
		ResultadosSinContacto: vacio.ResultadosSinContacto,
	}
	vacio.HuellaFuenteSHA256, err = version.HuellaCanonica()
	if err != nil {
		return puertosbolsa.FuentePoliticaContactos{}, ErrFuentePoliticaContactosNoDisponible
	}
	return vacio, nil
}
