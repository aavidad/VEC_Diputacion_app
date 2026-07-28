package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const (
	VersionHuellaConsultaRRHH      uint16 = 1
	VersionHuellaFiltrosCuadroRRHH uint16 = 1
	VersionHuellaAlcanceRRHH       uint16 = 1

	DominioHuellaConsultaCuadroRRHH  = "vec.contratacion_temporal.consulta_rrhh.cuadro.v1"
	DominioHuellaConsultaDetalleRRHH = "vec.contratacion_temporal.consulta_rrhh.detalle.v1"
	DominioHuellaFiltrosCuadroRRHH   = "vec.contratacion_temporal.filtros_rrhh.cuadro.v1"
	DominioHuellaAlcanceRRHH         = "vec.contratacion_temporal.alcance_rrhh.v1"

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

type canonAlcanceRRHH struct {
	Dominio         string `json:"dominio"`
	Version         uint16 `json:"version"`
	OrganizacionRef string `json:"organizacion_ref"`
	ClaseAmbito     string `json:"clase_ambito"`
	AmbitoRef       string `json:"ambito_ref"`
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
	exportacion, err := canonSolicitudCuadroRRHH(solicitud)
	if err != nil {
		return "", err
	}
	return exportacion.huellaSHA256, nil
}

func canonSolicitudCuadroRRHH(
	solicitud SolicitudCuadroRRHH,
) (exportacionCanonicaRRHH, error) {
	if solicitud.validar() != nil {
		return exportacionCanonicaRRHH{}, ErrSolicitudConsultaRRHHInvalida
	}
	return nuevaExportacionCanonicaRRHH(
		DominioHuellaConsultaCuadroRRHH,
		VersionHuellaConsultaRRHH,
		canonConsultaCuadroRRHH{
			Dominio: DominioHuellaConsultaCuadroRRHH, Version: VersionHuellaConsultaRRHH,
			Texto: solicitud.texto, EstadoClave: string(solicitud.estadoClave),
			FaseClave: string(solicitud.faseClave), Limite: solicitud.limite,
			Cursor: solicitud.cursor,
		},
	)
}

func huellaFiltrosCuadroRRHH(solicitud SolicitudCuadroRRHH) (string, error) {
	exportacion, err := canonFiltrosSolicitudCuadroRRHH(solicitud)
	if err != nil {
		return "", err
	}
	return exportacion.huellaSHA256, nil
}

func canonFiltrosSolicitudCuadroRRHH(
	solicitud SolicitudCuadroRRHH,
) (exportacionCanonicaRRHH, error) {
	if solicitud.validar() != nil {
		return exportacionCanonicaRRHH{}, ErrSolicitudConsultaRRHHInvalida
	}
	return nuevaExportacionCanonicaRRHH(
		DominioHuellaFiltrosCuadroRRHH,
		VersionHuellaFiltrosCuadroRRHH,
		canonFiltrosCuadroRRHH{
			Dominio: DominioHuellaFiltrosCuadroRRHH,
			Version: VersionHuellaFiltrosCuadroRRHH,
			Texto:   solicitud.texto, EstadoClave: string(solicitud.estadoClave),
			FaseClave: string(solicitud.faseClave), Limite: solicitud.limite,
		},
	)
}

func huellaSolicitudDetalleRRHH(solicitud SolicitudDetalleRRHH) (string, error) {
	exportacion, err := canonSolicitudDetalleRRHH(solicitud)
	if err != nil {
		return "", err
	}
	return exportacion.huellaSHA256, nil
}

func canonSolicitudDetalleRRHH(
	solicitud SolicitudDetalleRRHH,
) (exportacionCanonicaRRHH, error) {
	if solicitud.validar() != nil {
		return exportacionCanonicaRRHH{}, ErrSolicitudConsultaRRHHInvalida
	}
	return nuevaExportacionCanonicaRRHH(
		DominioHuellaConsultaDetalleRRHH,
		VersionHuellaConsultaRRHH,
		canonConsultaDetalleRRHH{
			Dominio: DominioHuellaConsultaDetalleRRHH, Version: VersionHuellaConsultaRRHH,
			ExpedienteRef:    solicitud.expedienteRef,
			VersionObservada: solicitud.versionObservada,
		},
	)
}

func canonAlcanceCapacidadRRHH(
	capacidad CapacidadConsultaRRHH,
) (exportacionCanonicaRRHH, error) {
	if capacidad.validarEstructura() != nil {
		return exportacionCanonicaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	return nuevaExportacionCanonicaRRHH(
		DominioHuellaAlcanceRRHH,
		VersionHuellaAlcanceRRHH,
		canonAlcanceRRHH{
			Dominio:         DominioHuellaAlcanceRRHH,
			Version:         VersionHuellaAlcanceRRHH,
			OrganizacionRef: capacidad.organizacionRef,
			ClaseAmbito:     string(capacidad.claseAmbito),
			AmbitoRef:       capacidad.ambitoRef,
		},
	)
}

func nuevaExportacionCanonicaRRHH(
	dominio string,
	version uint16,
	valor any,
) (exportacionCanonicaRRHH, error) {
	canon, err := json.Marshal(valor)
	if err != nil {
		return exportacionCanonicaRRHH{}, ErrSolicitudConsultaRRHHInvalida
	}
	suma := sha256.Sum256(canon)
	return exportacionCanonicaRRHH{
		dominio: dominio, version: version,
		bytesCanonicos: canon, huellaSHA256: hex.EncodeToString(suma[:]),
	}, nil
}
