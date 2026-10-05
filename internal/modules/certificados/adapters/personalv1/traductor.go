// Package personalv1 traduce el contrato LectorServiciosParaCertificadosV1 de
// Personal a la fuente del borrador. Personal conserva la autoridad sobre los
// servicios; aquí solo se cambia la forma, sin reglas de cómputo.
package personalv1

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/certificados/adapters/fichero"
	"vec-diputacion-granada/internal/modules/certificados/domain"
	"vec-diputacion-granada/internal/modules/certificados/ports"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

// Traducir convierte una respuesta V1. El periodo de Personal es [Desde, Hasta):
// el último día del servicio es el anterior a Hasta; Hasta vacía es un periodo
// abierto y así se conserva. V1 no trae días ni nombre: el nombre llega aparte
// y los días quedan sin aportar. La validación final es la del dominio.
func Traducir(r personalports.ResultadoServiciosParaCertificadosV1, nombre, procedenciaRef string, sintetica bool) (domain.FuenteServicios, error) {
	f := domain.FuenteServicios{Esquema: domain.EsquemaFuentePersonalV1Ensayo, Sintetica: sintetica, ProcedenciaRef: procedenciaRef,
		Nombre: nombre, Cobertura: string(r.Cobertura), Servicios: make([]domain.Servicio, 0, len(r.Servicios)),
		Corte: domain.Corte{VigenteEn: string(r.Corte.VigenteEn), ConocidoEn: r.Corte.ConocidoEn.UTC().Format(time.RFC3339Nano)}}
	if r.Corte.ConocidoEn.IsZero() || len(r.Servicios) > domain.MaxServicios {
		return domain.FuenteServicios{}, domain.ErrEntrada
	}
	for _, s := range r.Servicios {
		fin := ""
		if s.Periodo.Hasta != "" {
			hasta, err := time.Parse("2006-01-02", string(s.Periodo.Hasta))
			if err != nil || string(s.Periodo.Hasta) <= string(s.Periodo.Desde) {
				return domain.FuenteServicios{}, domain.ErrEntrada
			}
			fin = hasta.AddDate(0, 0, -1).Format("2006-01-02")
		}
		// V1 no recorta los periodos al corte: un fin previsto posterior (un
		// temporal en activo) se presenta en curso a la fecha de referencia.
		enCurso := fin > f.Corte.VigenteEn
		if enCurso {
			fin = f.Corte.VigenteEn
		}
		f.Servicios = append(f.Servicios, domain.Servicio{Inicio: string(s.Periodo.Desde), Fin: fin, EnCursoAlCorte: enCurso, Clase: s.ClaseRef,
			ClaseVersion: s.ClaseVersion, Estado: s.Estado, Certeza: string(s.Procedencia.Certeza),
			ServicioRef: s.ServicioRef, ActoRef: s.Procedencia.ActoRef})
	}
	return f, nil
}

// Muestra es una respuesta V1 sintética guardada en fichero para el ensayo
// local. No sustituye al lector de Personal, que aún no tiene implementación.
type Muestra struct {
	Esquema        string           `json:"esquema"`
	Sintetica      bool             `json:"sintetica"`
	ProcedenciaRef string           `json:"procedencia_ref"`
	Nombre         string           `json:"nombre"`
	Resultado      resultadoMuestra `json:"resultado"`
}

const EsquemaMuestra = "vec.certificados.muestra-personal-v1.ensayo"

type resultadoMuestra struct {
	EmpleadoRef  string `json:"empleado_ref"`
	OrganismoRef string `json:"organismo_ref"`
	Version      int64  `json:"version"`
	Corte        struct {
		VigenteEn  string    `json:"vigente_en"`
		ConocidoEn time.Time `json:"conocido_en"`
	} `json:"corte"`
	Cobertura string            `json:"cobertura"`
	Servicios []servicioMuestra `json:"servicios"`
}

type servicioMuestra struct {
	ServicioRef string `json:"servicio_ref"`
	RelacionRef string `json:"relacion_ref"`
	Version     int64  `json:"version"`
	Periodo     struct {
		Desde string `json:"desde"`
		Hasta string `json:"hasta"`
	} `json:"periodo"`
	Estado       string `json:"estado"`
	ClaseRef     string `json:"clase_ref"`
	ClaseVersion int64  `json:"clase_version"`
	Procedencia  struct {
		ActoRef       string `json:"acto_ref"`
		FuenteRef     string `json:"fuente_ref"`
		FuenteVersion string `json:"fuente_version"`
		Certeza       string `json:"certeza"`
	} `json:"procedencia"`
}

// resultado reconstruye la respuesta V1 tal como la entregaría Personal.
func (m Muestra) resultado() personalports.ResultadoServiciosParaCertificadosV1 {
	r := personalports.ResultadoServiciosParaCertificadosV1{EmpleadoRef: m.Resultado.EmpleadoRef, OrganismoRef: m.Resultado.OrganismoRef,
		Version: m.Resultado.Version, Cobertura: personalports.CoberturaPersonalNominalV1(m.Resultado.Cobertura),
		Corte: personaldomain.CorteEmpleadoB2{VigenteEn: personaldomain.FechaCivil(m.Resultado.Corte.VigenteEn), ConocidoEn: m.Resultado.Corte.ConocidoEn}}
	for _, s := range m.Resultado.Servicios {
		r.Servicios = append(r.Servicios, personalports.ServicioParaCertificadosV1{ServicioRef: s.ServicioRef, RelacionRef: s.RelacionRef,
			Version: s.Version, Estado: s.Estado, ClaseRef: s.ClaseRef, ClaseVersion: s.ClaseVersion,
			Periodo: personalports.PeriodoPersonalNominalV1{Desde: personaldomain.FechaCivil(s.Periodo.Desde), Hasta: personaldomain.FechaCivil(s.Periodo.Hasta)},
			Procedencia: personalports.ProcedenciaPersonalNominalV1{ActoRef: s.Procedencia.ActoRef, FuenteRef: s.Procedencia.FuenteRef,
				FuenteVersion: s.Procedencia.FuenteVersion, Certeza: personalports.CertezaPersonalNominalV1(s.Procedencia.Certeza)}})
	}
	return r
}

// FicheroMuestra lee una Muestra y la traduce con Traducir, el mismo camino
// que seguirá la respuesta del lector real de Personal.
type FicheroMuestra struct{ Ruta string }

var _ ports.FuenteServicios = FicheroMuestra{}

func (f FicheroMuestra) Obtener(ctx context.Context) (domain.FuenteServicios, error) {
	if ctx == nil || ctx.Err() != nil {
		return domain.FuenteServicios{}, domain.ErrNoDisponible
	}
	var m Muestra
	if fichero.LeerJSON(f.Ruta, &m) != nil || m.Esquema != EsquemaMuestra || !m.Sintetica {
		return domain.FuenteServicios{}, domain.ErrEntrada
	}
	fuente, err := Traducir(m.resultado(), m.Nombre, m.ProcedenciaRef, m.Sintetica)
	if err != nil || fuente.ValidarEnsayo() != nil {
		return domain.FuenteServicios{}, domain.ErrEntrada
	}
	return fuente, nil
}
