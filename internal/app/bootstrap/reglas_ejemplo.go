package bootstrap

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/config"
	calendariosdomain "vec-diputacion-granada/internal/modules/calendarios/domain"
	calendariosports "vec-diputacion-granada/internal/modules/calendarios/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	"vec-diputacion-granada/internal/vec/reglas"
)

var errReglasEjemploNoValidas = errors.New("bootstrap: catalogo de reglas de ejemplo no valido")

// reglasEjemploDesarrollo es el enganche de composición para Bolsa y
// Contratación temporal. Un resolutor nulo significa «sin catálogo»: el
// consumidor mantiene su conducta actual.
type reglasEjemploDesarrollo struct {
	bolsa                *reglas.Resolutor
	contratacionTemporal *reglas.Resolutor
}

// rechazarReglasEjemploFueraDesarrollo se aplica en todas las raíces: un
// catálogo de ejemplo declarado fuera de la doble llave impide arrancar.
func rechazarReglasEjemploFueraDesarrollo(cfg config.Config) error {
	_, _, err := cfg.ReglasEjemploDesarrollo()
	return err
}

// nuevasReglasEjemploDesarrollo carga y valida al arrancar los paquetes DEMO
// declarados. Un paquete ilegible, que no sea de demostración o cuyas reglas
// vigentes no cumplan el contrato impide arrancar en lugar de ignorarse.
func nuevasReglasEjemploDesarrollo(
	cfg config.Config,
	calendarios calendariosports.ConsultaCalendarios,
	reloj reglas.Reloj,
) (reglasEjemploDesarrollo, error) {
	rutas, activas, err := cfg.ReglasEjemploDesarrollo()
	if err != nil || !activas {
		return reglasEjemploDesarrollo{}, err
	}
	calculadora := calculadoraPlazosCalendarios{consulta: calendarios}
	var compuestas reglasEjemploDesarrollo
	if compuestas.bolsa, err = nuevoResolutorReglasEjemplo(
		rutas.BolsaSourcePath, reglas.CatalogoBolsa, reglas.ModuloBolsa, calculadora, reloj,
	); err != nil {
		return reglasEjemploDesarrollo{}, err
	}
	if compuestas.contratacionTemporal, err = nuevoResolutorReglasEjemplo(
		rutas.CTSourcePath, reglas.CatalogoContratacionTemporal, reglas.ModuloContratacionTemporal, calculadora, reloj,
	); err != nil {
		return reglasEjemploDesarrollo{}, err
	}
	return compuestas, nil
}

func nuevoResolutorReglasEjemplo(
	ruta, catalogoID, moduloID string,
	calculadora reglas.CalculadoraPlazos,
	reloj reglas.Reloj,
) (*reglas.Resolutor, error) {
	if ruta == "" {
		return nil, nil
	}
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		return nil, errors.Join(errReglasEjemploNoValidas, err)
	}
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: catalogoID, ModuloID: moduloID,
		Reloj: reloj, Calculadora: calculadora, MunicipioSede: reglas.MunicipioSedeDiputacion,
	})
	if err != nil {
		return nil, errors.Join(errReglasEjemploNoValidas, err)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	vigentes, err := resolutor.Reglas(ctx)
	if err != nil || len(vigentes) == 0 || !vigentes[0].PaqueteEjemplo {
		return nil, errors.Join(errReglasEjemploNoValidas, err)
	}
	return resolutor, nil
}

// calculadoraPlazosCalendarios traduce el puerto de reglas al de Calendarios.
// El cómputo administrativo usa el calendario oficial (art. 30 de la Ley
// 39/2015); el civil cuenta de fecha a fecha en hora peninsular (art. 5.1 del
// Código Civil) y no necesita calendario.
type calculadoraPlazosCalendarios struct {
	consulta calendariosports.ConsultaCalendarios
}

func (c calculadoraPlazosCalendarios) CalcularVencimiento(
	ctx context.Context,
	solicitud reglas.SolicitudVencimiento,
) (reglas.Vencimiento, error) {
	switch solicitud.Computo {
	case reglas.ComputoAdministrativo:
		return c.administrativo(ctx, solicitud)
	case reglas.ComputoCivil:
		return civilFechaAFecha(solicitud)
	default:
		return reglas.Vencimiento{}, reglas.ErrReglaSinPlazo
	}
}

func (c calculadoraPlazosCalendarios) administrativo(
	ctx context.Context,
	solicitud reglas.SolicitudVencimiento,
) (reglas.Vencimiento, error) {
	if dependenciaMotivosRectificacionAnalisisNula(c.consulta) {
		return reglas.Vencimiento{}, reglas.ErrCalculoNoDisponible
	}
	resultado, err := c.consulta.CalcularPlazo(ctx, calendariosports.SolicitudCalculoPlazo{
		NotificadoEn: solicitud.Inicio.UTC(), Unidad: calendariosdomain.UnidadPlazo(solicitud.Unidad),
		Cantidad: solicitud.Cantidad, MunicipioSede: solicitud.MunicipioSede,
	})
	if err != nil {
		return reglas.Vencimiento{}, err
	}
	versiones := make([]string, 0, len(resultado.VersionesUtilizadas))
	for _, version := range resultado.VersionesUtilizadas {
		versiones = append(versiones, version.ID)
	}
	return reglas.Vencimiento{
		UltimoDia: resultado.Vencimiento.String(), VenceAntesDe: resultado.VenceAntesDe,
		Prorrogado: resultado.Prorrogado, Calendarios: versiones,
	}, nil
}

func civilFechaAFecha(solicitud reglas.SolicitudVencimiento) (reglas.Vencimiento, error) {
	inicio, err := calendariosdomain.FechaCivilDe(solicitud.Inicio)
	if err != nil || solicitud.Cantidad < 1 {
		return reglas.Vencimiento{}, reglas.ErrReglaSinPlazo
	}
	var fin calendariosdomain.FechaCivil
	switch solicitud.Unidad {
	case reglas.UnidadDiasNaturales:
		fin, err = inicio.SumarDias(solicitud.Cantidad)
	case reglas.UnidadMeses:
		fin, err = inicio.SumarMesesMismoDia(solicitud.Cantidad)
	case reglas.UnidadAnios:
		fin, err = inicio.SumarMesesMismoDia(12 * solicitud.Cantidad)
	default:
		// Los días hábiles no existen en el cómputo civil.
		return reglas.Vencimiento{}, reglas.ErrReglaSinPlazo
	}
	if err != nil {
		return reglas.Vencimiento{}, reglas.ErrReglaSinPlazo
	}
	venceAntesDe, err := fin.FinEnMadrid()
	if err != nil {
		return reglas.Vencimiento{}, reglas.ErrReglaSinPlazo
	}
	return reglas.Vencimiento{UltimoDia: fin.String(), VenceAntesDe: venceAntesDe}, nil
}
