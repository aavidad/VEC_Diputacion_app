package bootstrap

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

// materialUsuariosAisladoDesarrollo conserva solo el gobierno técnico y las
// audiencias de Usuarios. El pool y su renovación viven mientras sirvan esas
// rutas; ninguna dependencia funcional de CT se construye aquí.
type materialUsuariosAisladoDesarrollo struct {
	materiales materialUsuariosDesarrollo
	gobierno   *pgxpool.Pool
	detener    func()
	unaVez     sync.Once
}

func (m *materialUsuariosAisladoDesarrollo) cerrar() {
	if m == nil {
		return
	}
	m.unaVez.Do(func() {
		if m.detener != nil {
			m.detener()
		}
		if m.gobierno != nil {
			m.gobierno.Close()
		}
	})
}

func abrirMaterialUsuariosAisladoDesarrollo(cfg config.Config,
	derivador *derivadorIdentidadOperacionDesarrollo,
) (_ *materialUsuariosAisladoDesarrollo, errFinal error) {
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosPreferenciasDesarrollo)
	if err != nil || !activo {
		return nil, err
	}
	if !cfg.DevelopmentEnabledByDoubleKey() || derivador == nil || !derivador.valido() {
		return nil, errComposicionUsuariosPreferencias
	}
	_, dsnGobierno, err := cfg.Normalize().ContratacionTemporalPostgreSQL.DSNSeparados()
	if err != nil {
		return nil, err
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(15*time.Second))
	defer cancelar()
	gobierno, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(
		ctx, dsnGobierno, "vec-usuarios-desarrollo-gobierno", rolGobiernoPostgreSQLContratacionTemporalDesarrollo)
	if err != nil {
		return nil, err
	}
	completa := false
	defer func() {
		if !completa {
			gobierno.Close()
		}
	}()
	reloj := relojContratacionTemporalDesarrollo{}
	base, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, reloj.Ahora())
	if err != nil {
		return nil, err
	}
	defer base.borrarCopiasEfimeras()
	// La confianza renovable es técnica. No se publica la audiencia de alta
	// ni se provisiona un perfil CT para servir a Usuarios.
	base.fuenteConfianza, err = nuevaFuenteConfianzaRenovableCTDesarrollo(gobierno, base, reloj)
	if err != nil {
		return nil, err
	}
	seleccion, err := seleccionarMaterialUsuariosDesarrollo(cfg, derivador, gobierno)
	if err != nil {
		return nil, err
	}
	materiales, err := publicarMaterialUsuariosDesarrollo(ctx, gobierno, base, reloj, seleccion)
	if err != nil {
		return nil, err
	}
	m := &materialUsuariosAisladoDesarrollo{materiales: materiales, gobierno: gobierno,
		detener: iniciarRenovacionProgramadaCTDesarrollo(base.fuenteConfianza, esperarTemporizadorCTDesarrollo)}
	completa = true
	return m, nil
}
