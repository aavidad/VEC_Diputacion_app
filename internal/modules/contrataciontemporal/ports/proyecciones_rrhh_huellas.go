package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const (
	VersionHuellaConsultaRRHH      uint16 = 1
	VersionHuellaFiltrosCuadroRRHH uint16 = 1

	DominioHuellaConsultaCuadroRRHH  = "vec.contratacion_temporal.consulta_rrhh.cuadro.v1"
	DominioHuellaConsultaDetalleRRHH = "vec.contratacion_temporal.consulta_rrhh.detalle.v1"
	DominioHuellaFiltrosCuadroRRHH   = "vec.contratacion_temporal.filtros_rrhh.cuadro.v1"

	AudienciaConsumoConsultaCuadroRRHHV3  = "vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1"
	AudienciaConsumoConsultaDetalleRRHHV3 = "vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1"

	ambitoOrganizacionRecursoRRHH = "organizacion_ref"
	ambitoClaseRecursoRRHH        = "clase_ambito"
	ambitoReferenciaRecursoRRHH   = "ambito_ref"
	atributoDominioConsultaRRHH   = "consulta_dominio"
	atributoHuellaConsultaRRHH    = "consulta_huella_sha256"
)

type canonConsultaCuadroRRHH struct {
	Dominio     string `json:"dominio"`
	Version     uint16 `json:"version"`
	Texto       string `json:"texto"`
	EstadoClave string `json:"estado_clave"`
	FaseClave   string `json:"fase_clave"`
	Limite      uint16 `json:"limite"`
	Cursor      string `json:"cursor"`
}

type canonConsultaDetalleRRHH struct {
	Dominio          string `json:"dominio"`
	Version          uint16 `json:"version"`
	ExpedienteRef    string `json:"expediente_ref"`
	VersionObservada uint64 `json:"version_observada"`
}

// canonFiltrosCuadroRRHH identifica una familia de páginas. El cursor queda
// deliberadamente fuera: cada página conserva su huella de consulta exacta,
// mientras esta representación permanece estable durante toda la navegación.
type canonFiltrosCuadroRRHH struct {
	Dominio     string `json:"dominio"`
	Version     uint16 `json:"version"`
	Texto       string `json:"texto"`
	EstadoClave string `json:"estado_clave"`
	FaseClave   string `json:"fase_clave"`
	Limite      uint16 `json:"limite"`
}

func huellaSolicitudCuadroRRHH(solicitud SolicitudCuadroRRHH) (string, error) {
	if solicitud.validar() != nil {
		return "", ErrSolicitudConsultaRRHHInvalida
	}
	return huellaCanonConsultaRRHH(canonConsultaCuadroRRHH{
		Dominio: DominioHuellaConsultaCuadroRRHH, Version: VersionHuellaConsultaRRHH,
		Texto: solicitud.texto, EstadoClave: string(solicitud.estadoClave),
		FaseClave: string(solicitud.faseClave), Limite: solicitud.limite,
		Cursor: solicitud.cursor,
	})
}

func huellaFiltrosCuadroRRHH(solicitud SolicitudCuadroRRHH) (string, error) {
	if solicitud.validar() != nil {
		return "", ErrSolicitudConsultaRRHHInvalida
	}
	return huellaCanonConsultaRRHH(canonFiltrosCuadroRRHH{
		Dominio: DominioHuellaFiltrosCuadroRRHH,
		Version: VersionHuellaFiltrosCuadroRRHH,
		Texto:   solicitud.texto, EstadoClave: string(solicitud.estadoClave),
		FaseClave: string(solicitud.faseClave), Limite: solicitud.limite,
	})
}

func huellaSolicitudDetalleRRHH(solicitud SolicitudDetalleRRHH) (string, error) {
	if solicitud.validar() != nil {
		return "", ErrSolicitudConsultaRRHHInvalida
	}
	return huellaCanonConsultaRRHH(canonConsultaDetalleRRHH{
		Dominio: DominioHuellaConsultaDetalleRRHH, Version: VersionHuellaConsultaRRHH,
		ExpedienteRef:    solicitud.expedienteRef,
		VersionObservada: solicitud.versionObservada,
	})
}

func huellaCanonConsultaRRHH(valor any) (string, error) {
	canon, err := json.Marshal(valor)
	if err != nil {
		return "", ErrSolicitudConsultaRRHHInvalida
	}
	suma := sha256.Sum256(canon)
	return hex.EncodeToString(suma[:]), nil
}
