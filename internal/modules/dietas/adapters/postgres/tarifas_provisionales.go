package postgres

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/dietas/domain"
)

var ErrTarifaProvisionalNoDisponible = errors.New("dietas: tarifa provisional no disponible")

// TarifaComisionProvisional une solo importes propios de Dietas. El grupo lo
// resuelve Personal; el cliente HTTP nunca lo selecciona como autoridad.
type TarifaComisionProvisional struct {
	Dieta      domain.TarifaNacionalProvisional
	EURPorKM   string
	Vehiculo   string
	Referencia string
}

type consultaTarifaProvisional interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type RepositorioTarifasProvisionales struct{ consulta consultaTarifaProvisional }

func NuevoRepositorioTarifasProvisionales(pool *pgxpool.Pool) (*RepositorioTarifasProvisionales, error) {
	if pool == nil {
		return nil, ErrTarifaProvisionalNoDisponible
	}
	return &RepositorioTarifasProvisionales{consulta: pool}, nil
}

const consultaTarifaComisionProvisional = `
SELECT v.version_ref,v.rotulo,d.pais_iso2,d.grupo,
       v.vigente_desde::text,coalesce(v.vigente_hasta::text,''),
       (d.manutencion_eur*100)::bigint,(d.alojamiento_eur*100)::bigint,
       k.eur_por_km::text
  FROM vec_dietas.version_tarifa_provisional v
  JOIN vec_dietas.importe_dieta_provisional d ON d.version_ref=v.version_ref
  JOIN vec_dietas.importe_km_provisional k ON k.version_ref=v.version_ref
 WHERE v.version_ref=$1 AND d.pais_iso2='ES' AND d.grupo=$2
   AND k.vehiculo=$3 AND v.vigente_desde<=$4::date
   AND (v.vigente_hasta IS NULL OR $4::date<v.vigente_hasta)`

// Consultar exige una versión explícita: nunca adopta en silencio una tarifa
// nueva si otra sesión publica una versión durante la edición de la comisión.
func (r *RepositorioTarifasProvisionales) Consultar(ctx context.Context, version string, grupo int, vehiculo string, fecha time.Time) (TarifaComisionProvisional, error) {
	var vacia TarifaComisionProvisional
	if r == nil || dependenciaTarifasNula(r.consulta) || ctx == nil || ctx.Err() != nil ||
		version == "" || grupo < 1 || grupo > 3 || (vehiculo != "automovil" && vehiculo != "motocicleta") ||
		fecha.IsZero() || fecha.Location() != time.UTC {
		return vacia, ErrTarifaProvisionalNoDisponible
	}
	var tarifa TarifaComisionProvisional
	var grupoLeido int
	err := r.consulta.QueryRow(ctx, consultaTarifaComisionProvisional, version, grupo, vehiculo, fecha.Format("2006-01-02")).Scan(
		&tarifa.Dieta.VersionRef, &tarifa.Dieta.Rotulo, &tarifa.Dieta.PaisISO2, &grupoLeido,
		&tarifa.Dieta.VigenteDesde, &tarifa.Dieta.VigenteHasta,
		&tarifa.Dieta.ManutencionCentimos, &tarifa.Dieta.AlojamientoTopeCentimos, &tarifa.EURPorKM,
	)
	if err != nil || ctx.Err() != nil || grupoLeido != grupo || tarifa.Dieta.VersionRef != version ||
		tarifa.Dieta.Rotulo != domain.RotuloTarifaProvisional || tarifa.Dieta.PaisISO2 != "ES" ||
		tarifa.Dieta.ManutencionCentimos < 1 || tarifa.Dieta.AlojamientoTopeCentimos < 1 {
		return vacia, ErrTarifaProvisionalNoDisponible
	}
	tarifa.Dieta.Grupo = grupoLeido
	tarifa.Vehiculo = vehiculo
	tarifa.Referencia = "tarifa:km:" + version + ":" + vehiculo
	if (domain.PoliticaKilometraje{Referencia: tarifa.Referencia, Version: version, TarifaEURPorKM: tarifa.EURPorKM}).Validar() != nil {
		return vacia, ErrTarifaProvisionalNoDisponible
	}
	return tarifa, nil
}

func dependenciaTarifasNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	return (r.Kind() == reflect.Interface || r.Kind() == reflect.Pointer) && r.IsNil()
}
