package bootstrap

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

// gobiernoTecnicoAisladoDesarrollo comparte únicamente la publicación y la
// confianza técnica V3. Cada consumidor publica sus propias audiencias.
type gobiernoTecnicoAisladoDesarrollo struct {
	gobierno *pgxpool.Pool
	base     materialAtestacionContratacionTemporalDesarrollo
	detener  func()
	unaVez   sync.Once
}

func (m *gobiernoTecnicoAisladoDesarrollo) cerrar() {
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
		m.base.borrarCopiasEfimeras()
	})
}

func abrirGobiernoTecnicoAisladoDesarrollo(cfg config.Config,
	derivador *derivadorIdentidadOperacionDesarrollo,
) (*gobiernoTecnicoAisladoDesarrollo, error) {
	if !cfg.DevelopmentEnabledByDoubleKey() || derivador == nil || !derivador.valido() {
		return nil, ErrActivacionDesarrolloInvalida
	}
	_, dsnGobierno, err := cfg.Normalize().ContratacionTemporalPostgreSQL.DSNSeparados()
	if err != nil {
		return nil, err
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(15*time.Second))
	defer cancelar()
	gobierno, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(
		ctx, dsnGobierno, "vec-comun-desarrollo-gobierno", rolGobiernoPostgreSQLContratacionTemporalDesarrollo)
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
	defer func() {
		if !completa {
			base.borrarCopiasEfimeras()
		}
	}()
	// La confianza renovable es técnica. No se publica la audiencia de alta
	// ni se provisiona un perfil CT para servir a Usuarios o Documentos.
	base.fuenteConfianza, err = nuevaFuenteConfianzaRenovableCTDesarrollo(gobierno, base, reloj)
	if err != nil {
		return nil, err
	}
	m := &gobiernoTecnicoAisladoDesarrollo{base: base, gobierno: gobierno,
		detener: iniciarRenovacionProgramadaCTDesarrollo(base.fuenteConfianza, esperarTemporizadorCTDesarrollo)}
	completa = true
	return m, nil
}

type materialUsuariosAisladoDesarrollo struct {
	materiales materialUsuariosDesarrollo
	*gobiernoTecnicoAisladoDesarrollo
}

func abrirMaterialUsuariosAisladoDesarrollo(cfg config.Config,
	derivador *derivadorIdentidadOperacionDesarrollo,
) (*materialUsuariosAisladoDesarrollo, error) {
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosPreferenciasDesarrollo)
	if err != nil || !activo {
		return nil, err
	}
	gobierno, err := abrirGobiernoTecnicoAisladoDesarrollo(cfg, derivador)
	if err != nil {
		return nil, err
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(15*time.Second))
	defer cancelar()
	seleccion, err := seleccionarMaterialUsuariosDesarrollo(cfg, derivador, gobierno.gobierno)
	if err != nil {
		gobierno.cerrar()
		return nil, err
	}
	materiales, err := publicarMaterialUsuariosDesarrollo(ctx, gobierno.gobierno, gobierno.base, relojContratacionTemporalDesarrollo{}, seleccion)
	if err != nil {
		gobierno.cerrar()
		return nil, err
	}
	return &materialUsuariosAisladoDesarrollo{materiales: materiales, gobiernoTecnicoAisladoDesarrollo: gobierno}, nil
}
