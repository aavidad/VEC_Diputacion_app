package composicion

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	personalpostgres "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	personalrpt "vec-diputacion-granada/internal/modules/personal/adapters/rpt"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type DependenciasLectorRelacionRPT struct {
	Identidad                       ResolutorIdentidadLectorRelacionRPT
	Emisor                          EmisorMaterialLectorRelacionRPTV3
	Motivo                          vecdomain.ReferenciaEntradaCatalogo
	Lectura                         *pgxpool.Pool
	Intentos                        RegistradorIntentosLectorRPT
	ConfiguracionIntentos           ConfiguracionIntentosLectorRPT
	Ahora                           func() time.Time
	LimiteIdentidad                 time.Duration
	ResultadosTecnicos              vecports.EmisorResultadosTecnicosConContexto
	ConfiguracionResultadosTecnicos ConfiguracionResultadosTecnicosLectorRPT
}

// ComponerLectorRelacionSeleccionadaRPT prepara el consumidor de la selección
// 356 con autoridades reales y el registrador común. No monta HTTP ni provisión.
// La ausencia de SQL, concesión o registrador mantiene cada lectura cerrada.
func ComponerLectorRelacionSeleccionadaRPT(d DependenciasLectorRelacionRPT) (LectorRelacionSeleccionadaRPT, error) {
	if d.Lectura == nil || dependenciaNula(d.Intentos) || d.Ahora == nil || dependenciaNula(d.ResultadosTecnicos) || d.ConfiguracionResultadosTecnicos.validar() != nil || d.LimiteIdentidad <= 0 || d.LimiteIdentidad > 30*time.Second {
		return nil, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	a, err := NuevoProveedorAutorizacionLectorRelacionRPT(d.Identidad, d.Emisor, d.Motivo)
	if err != nil {
		return nil, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	r, err := personalpostgres.NuevoRepositorioLectorRelacionRPTPostgreSQL(d.Lectura)
	if err != nil {
		return nil, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	i, err := NuevoRegistroIntentosLectorRelacionRPT(d.Intentos, d.ConfiguracionIntentos)
	if err != nil {
		return nil, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	s, err := personalapp.NuevoServicioLectorRelacionRPT(a, r, i, d.Ahora)
	if err != nil {
		return nil, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	lector, err := personalrpt.NuevoLectorRelacionSeleccionadaRPT(s, i)
	if err != nil {
		return nil, err
	}
	return NuevoLectorRelacionSeleccionadaRPTConIdentidad(lector, d.Identidad, d.LimiteIdentidad, d.ResultadosTecnicos, d.ConfiguracionResultadosTecnicos)
}
