package ports

import (
	"crypto/sha256"
	"time"
)

// PreparacionRevalidacionEvidenciasFuenteAnalisisO3 contiene únicamente los
// materiales y compromisos ya verificados que application necesita para
// coordinar desafíos nuevos. No conoce adaptadores ni invoca colaboradores.
type PreparacionRevalidacionEvidenciasFuenteAnalisisO3 struct {
	bloqueoSerializacionOperacionAnalisis
	comprobadaEn        time.Time
	materialRC          []byte
	fuenteRCEsperada    VinculoAutoridadFuenteAnalisisO3
	verificadorEsperado VinculoAutoridadFuenteAnalisisO3
	publicadorEsperado  VinculoAutoridadFuenteAnalisisO3
	materialCoste       []byte
	fuenteCosteEsperada VinculoAutoridadFuenteAnalisisO3
	verificadorCoste    VinculoAutoridadFuenteAnalisisO3
}

// ResultadoRevalidacionEvidenciasFuenteAnalisisO3 es la proyección neutral de
// las presentaciones comprobadas por application. La preparación la coteja
// localmente contra las identidades originales.
type ResultadoRevalidacionEvidenciasFuenteAnalisisO3 struct {
	bloqueoSerializacionOperacionAnalisis
	FuenteRC         ConfirmacionComprobacionAutoridadFuenteAnalisis
	VerificadorRC    ConfirmacionComprobacionAutoridadFuenteAnalisis
	PublicadorRC     ConfirmacionComprobacionAutoridadFuenteAnalisis
	FuenteCoste      ConfirmacionComprobacionAutoridadFuenteAnalisis
	VerificadorCoste ConfirmacionComprobacionAutoridadFuenteAnalisis
}

func NuevaPreparacionRevalidacionEvidenciasFuenteAnalisisO3(
	rc EvidenciaValidacionRCVerificadaO3,
	coste EvidenciaCalculoCosteVerificadaO3,
	comprobadaEn time.Time,
) (PreparacionRevalidacionEvidenciasFuenteAnalisisO3, error) {
	if rc.ValidarEn(comprobadaEn) != nil ||
		coste.ValidarEn(comprobadaEn) != nil ||
		rc.datos == nil {
		return PreparacionRevalidacionEvidenciasFuenteAnalisisO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	datosRC, err := rc.datos.solicitud.Datos()
	materialRC := materialDesafioSolicitudFuenteAnalisis(
		rc.datos.solicitud.datosCanonicos(),
		datosRC.HuellaPeticionHMAC,
	)
	if err != nil || len(materialRC) == 0 {
		return PreparacionRevalidacionEvidenciasFuenteAnalisisO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	preparacion := PreparacionRevalidacionEvidenciasFuenteAnalisisO3{
		comprobadaEn: comprobadaEn,
		materialRC:   append([]byte(nil), materialRC...),
		fuenteRCEsperada: vinculoAutoridadFuenteAnalisisO3(
			rc.datos.identidadFuente,
		),
		verificadorEsperado: vinculoAutoridadFuenteAnalisisO3(
			rc.datos.identidadVerificador,
		),
		publicadorEsperado: vinculoAutoridadFuenteAnalisisO3(
			rc.datos.identidadPublicador,
		),
	}
	if coste.datos != nil {
		datosCoste, errCoste := coste.datos.solicitud.Datos()
		materialCoste := materialDesafioSolicitudFuenteAnalisis(
			coste.datos.solicitud.datosCanonicos(),
			datosCoste.HuellaPeticionHMAC,
		)
		if errCoste != nil || len(materialCoste) == 0 {
			return PreparacionRevalidacionEvidenciasFuenteAnalisisO3{},
				ErrResultadoFuenteAnalisisNoConfiable
		}
		preparacion.materialCoste =
			append([]byte(nil), materialCoste...)
		preparacion.fuenteCosteEsperada =
			vinculoAutoridadFuenteAnalisisO3(
				coste.datos.identidadFuente,
			)
		preparacion.verificadorCoste =
			vinculoAutoridadFuenteAnalisisO3(
				coste.datos.identidadVerificador,
			)
	}
	if preparacion.validar() != nil {
		return PreparacionRevalidacionEvidenciasFuenteAnalisisO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return preparacion, nil
}

func (p PreparacionRevalidacionEvidenciasFuenteAnalisisO3) MaterialRC() (
	[]byte,
	error,
) {
	if p.validar() != nil {
		return nil, ErrResultadoFuenteAnalisisNoConfiable
	}
	return append([]byte(nil), p.materialRC...), nil
}

func (p PreparacionRevalidacionEvidenciasFuenteAnalisisO3) MaterialCoste() (
	[]byte,
	bool,
	error,
) {
	if p.validar() != nil {
		return nil, false, ErrResultadoFuenteAnalisisNoConfiable
	}
	if len(p.materialCoste) == 0 {
		return nil, false, nil
	}
	return append([]byte(nil), p.materialCoste...), true, nil
}

func (p PreparacionRevalidacionEvidenciasFuenteAnalisisO3) ValidarResultado(
	resultado ResultadoRevalidacionEvidenciasFuenteAnalisisO3,
) error {
	fuenteRC, errFuenteRC := resultado.FuenteRC.validarPara(
		p.materialRC,
		RolFuentePresupuestaria,
		p.comprobadaEn,
	)
	verificadorRC, errVerificadorRC :=
		resultado.VerificadorRC.validarPara(
			p.materialRC,
			RolVerificadorRespuesta,
			p.comprobadaEn,
		)
	publicadorRC, errPublicadorRC :=
		resultado.PublicadorRC.validarPara(
			p.materialRC,
			RolPublicadorCatalogo,
			p.comprobadaEn,
		)
	if p.validar() != nil || errFuenteRC != nil ||
		errVerificadorRC != nil || errPublicadorRC != nil ||
		!vinculosAutoridadFuenteAnalisisO3Iguales(
			p.fuenteRCEsperada,
			fuenteRC,
		) ||
		!vinculosAutoridadFuenteAnalisisO3Iguales(
			p.verificadorEsperado,
			verificadorRC,
		) ||
		!vinculosAutoridadFuenteAnalisisO3Iguales(
			p.publicadorEsperado,
			publicadorRC,
		) ||
		!vinculosAutoridadFuenteAnalisisO3Separados(
			fuenteRC,
			verificadorRC,
			publicadorRC,
		) {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	if len(p.materialCoste) == 0 {
		if resultado.FuenteCoste !=
			(ConfirmacionComprobacionAutoridadFuenteAnalisis{}) ||
			resultado.VerificadorCoste !=
				(ConfirmacionComprobacionAutoridadFuenteAnalisis{}) {
			return ErrResultadoFuenteAnalisisNoConfiable
		}
		return nil
	}
	fuenteCoste, errFuenteCoste := resultado.FuenteCoste.validarPara(
		p.materialCoste,
		RolCalculadorCoste,
		p.comprobadaEn,
	)
	verificadorCoste, errVerificadorCoste :=
		resultado.VerificadorCoste.validarPara(
			p.materialCoste,
			RolVerificadorRespuesta,
			p.comprobadaEn,
		)
	if errFuenteCoste != nil || errVerificadorCoste != nil ||
		!vinculosAutoridadFuenteAnalisisO3Iguales(
			p.fuenteCosteEsperada,
			fuenteCoste,
		) ||
		!vinculosAutoridadFuenteAnalisisO3Iguales(
			p.verificadorCoste,
			verificadorCoste,
		) ||
		!vinculosAutoridadFuenteAnalisisO3Separados(
			fuenteCoste,
			verificadorCoste,
		) {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return nil
}

func (p PreparacionRevalidacionEvidenciasFuenteAnalisisO3) validar() error {
	if !instanteFuenteAnalisisCanonico(p.comprobadaEn) ||
		len(p.materialRC) == 0 || len(p.materialRC) > 64*1024 ||
		!vinculoAutoridadAnalisisValido(
			p.fuenteRCEsperada,
			RolFuentePresupuestaria,
		) ||
		!vinculoAutoridadAnalisisValido(
			p.verificadorEsperado,
			RolVerificadorRespuesta,
		) ||
		!vinculoAutoridadAnalisisValido(
			p.publicadorEsperado,
			RolPublicadorCatalogo,
		) ||
		!vinculosAutoridadFuenteAnalisisO3Separados(
			p.fuenteRCEsperada,
			p.verificadorEsperado,
			p.publicadorEsperado,
		) {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	if len(p.materialCoste) == 0 {
		if p.fuenteCosteEsperada !=
			(VinculoAutoridadFuenteAnalisisO3{}) ||
			p.verificadorCoste !=
				(VinculoAutoridadFuenteAnalisisO3{}) {
			return ErrResultadoFuenteAnalisisNoConfiable
		}
		return nil
	}
	if len(p.materialCoste) > 64*1024 ||
		!vinculoAutoridadAnalisisValido(
			p.fuenteCosteEsperada,
			RolCalculadorCoste,
		) ||
		!vinculoAutoridadAnalisisValido(
			p.verificadorCoste,
			RolVerificadorRespuesta,
		) ||
		!vinculosAutoridadFuenteAnalisisO3Separados(
			p.fuenteCosteEsperada,
			p.verificadorCoste,
		) {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return nil
}

func vinculosAutoridadFuenteAnalisisO3Iguales(
	primero VinculoAutoridadFuenteAnalisisO3,
	segundo VinculoAutoridadFuenteAnalisisO3,
) bool {
	return primero.RaizClaveID == segundo.RaizClaveID &&
		primero.AutoridadRef == segundo.AutoridadRef &&
		primero.BackendRef == segundo.BackendRef &&
		primero.Rol == segundo.Rol &&
		primero.Serie == segundo.Serie &&
		primero.Generacion == segundo.Generacion &&
		primero.HuellaClaveSHA256 == segundo.HuellaClaveSHA256 &&
		primero.CredencialEmitidaEn.Equal(
			segundo.CredencialEmitidaEn,
		) &&
		primero.CredencialValidaHasta.Equal(
			segundo.CredencialValidaHasta,
		)
}

func vinculosAutoridadFuenteAnalisisO3Separados(
	vinculos ...VinculoAutoridadFuenteAnalisisO3,
) bool {
	for indice, primero := range vinculos {
		if !vinculoAutoridadAnalisisValido(primero, primero.Rol) {
			return false
		}
		for _, segundo := range vinculos[indice+1:] {
			if primero.AutoridadRef == segundo.AutoridadRef ||
				primero.BackendRef == segundo.BackendRef ||
				primero.HuellaClaveSHA256 ==
					segundo.HuellaClaveSHA256 {
				return false
			}
		}
	}
	return true
}

// ConfirmacionComprobacionAutoridadFuenteAnalisis sólo puede nacer después de
// verificar localmente la credencial y la prueba del desafío. Liga identidad,
// material, rol e instante sin exponer claves ni permitir construcción nominal
// desde application.
type ConfirmacionComprobacionAutoridadFuenteAnalisis struct {
	vinculo        VinculoAutoridadFuenteAnalisisO3
	huellaMaterial [sha256.Size]byte
	rol            RolAutoridadFuenteAnalisis
	comprobadaEn   time.Time
}

func (c ConfirmacionComprobacionAutoridadFuenteAnalisis) validarPara(
	material []byte,
	rol RolAutoridadFuenteAnalisis,
	comprobadaEn time.Time,
) (VinculoAutoridadFuenteAnalisisO3, error) {
	huella := sha256.Sum256(material)
	if len(material) == 0 || len(material) > 64*1024 ||
		c.rol != rol || !rol.valida() ||
		!c.comprobadaEn.Equal(comprobadaEn) ||
		!instanteFuenteAnalisisCanonico(c.comprobadaEn) ||
		c.huellaMaterial != huella ||
		!vinculoAutoridadAnalisisValido(c.vinculo, rol) {
		return VinculoAutoridadFuenteAnalisisO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return c.vinculo, nil
}

// ComprobacionAutoridadFuenteAnalisis contiene un desafío y su verificación
// criptográfica local. No invoca al presentador: esa coordinación pertenece a
// application.
type ComprobacionAutoridadFuenteAnalisis struct {
	confianza      ConfianzaAutoridadesFuenteAnalisis
	desafio        DesafioAutoridadFuenteAnalisis
	huellaMaterial [sha256.Size]byte
	rol            RolAutoridadFuenteAnalisis
	comprobadaEn   time.Time
}

func NuevaComprobacionAutoridadFuenteAnalisis(
	confianza ConfianzaAutoridadesFuenteAnalisis,
	material []byte,
	rol RolAutoridadFuenteAnalisis,
	comprobadaEn time.Time,
) (ComprobacionAutoridadFuenteAnalisis, error) {
	if confianza.Validar() != nil ||
		!instanteFuenteAnalisisCanonico(comprobadaEn) {
		return ComprobacionAutoridadFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	desafio, err := nuevoDesafioAutoridadFuenteAnalisis(
		material,
		confianza.organizacionRef,
		confianza.audiencia,
		rol,
	)
	if err != nil {
		return ComprobacionAutoridadFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return ComprobacionAutoridadFuenteAnalisis{
		confianza:      confianza,
		desafio:        desafio,
		huellaMaterial: sha256.Sum256(material),
		rol:            rol,
		comprobadaEn:   comprobadaEn,
	}, nil
}

func (c ComprobacionAutoridadFuenteAnalisis) Desafio() (
	DesafioAutoridadFuenteAnalisis,
	error,
) {
	if c.confianza.Validar() != nil || !c.rol.valida() ||
		!instanteFuenteAnalisisCanonico(c.comprobadaEn) {
		return DesafioAutoridadFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	contenido, err := c.desafio.Bytes()
	if err != nil {
		return DesafioAutoridadFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return DesafioAutoridadFuenteAnalisis{
		contenido: contenido,
	}, nil
}

func (c ComprobacionAutoridadFuenteAnalisis) ValidarPresentacion(
	presentacion PresentacionAutoridadFuenteAnalisis,
) (ConfirmacionComprobacionAutoridadFuenteAnalisis, error) {
	identidad, err := c.confianza.verificarPresentacion(
		presentacion,
		c.desafio,
		c.rol,
		c.comprobadaEn,
	)
	if err != nil {
		return ConfirmacionComprobacionAutoridadFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return ConfirmacionComprobacionAutoridadFuenteAnalisis{
		vinculo:        vinculoAutoridadFuenteAnalisisO3(identidad),
		huellaMaterial: c.huellaMaterial,
		rol:            c.rol,
		comprobadaEn:   c.comprobadaEn,
	}, nil
}
