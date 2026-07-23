package ports

import (
	"crypto/sha256"
	"encoding/hex"
)

const esquemaHuellaArtefactoAnalisisO3 = "VEC-CT-ARTEFACTO-ANALISIS-O3-V3"

// huellaArtefactoAnalisisO3 es una dirección de contenido, no una autoridad
// criptográfica adicional. La autoridad procede exclusivamente de las
// respuestas y confirmaciones verificadas por O3-03.
func huellaArtefactoAnalisisO3(
	datos DatosArtefactoAnalisis,
) (string, error) {
	if datos.ArtefactoHuellaSHA256 != "" {
		return "", ErrArtefactoAnalisisNoConfiable
	}
	canon := nuevoCanonOperacionAnalisis()
	canon.texto(esquemaHuellaArtefactoAnalisisO3)
	escribirContenidoArtefactoAnalisisO3(canon, datos, false)
	contenido, err := canon.resultado()
	if err != nil {
		return "", ErrArtefactoAnalisisNoConfiable
	}
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:]), nil
}

func escribirContenidoArtefactoAnalisisO3(
	canon *canonOperacionAnalisis,
	datos DatosArtefactoAnalisis,
	incluirHuella bool,
) {
	canon.texto(datos.ArtefactoRef)
	if incluirHuella {
		canon.texto(datos.ArtefactoHuellaSHA256)
	}
	canon.texto(datos.OrganizacionRef)
	canon.texto(datos.ExpedienteRef)
	canon.enteroSinSigno(datos.VersionExpediente)
	escribirDatosFuncionalesCanonicos(canon, datos.DatosFuncionales)

	canon.texto(string(datos.ResultadoRC))
	canon.texto(datos.FuenteRCRef)
	canon.texto(datos.ReciboRCRef)
	canon.instante(datos.ValidadaEn)
	canon.booleano(datos.FechaRC != nil)
	if datos.FechaRC != nil {
		canon.instante(*datos.FechaRC)
	}
	canon.texto(datos.NumeroRC)
	canon.booleano(datos.ImporteRC != nil)
	if datos.ImporteRC != nil {
		canon.enteroConSigno(datos.ImporteRC.Centimos)
		canon.texto(datos.ImporteRC.Moneda)
	}
	canon.texto(datos.DocumentoRCRef)
	canon.texto(datos.MotivoRC.ReferenciaCatalogo.CatalogoID)
	canon.enteroConSigno(
		int64(datos.MotivoRC.ReferenciaCatalogo.CatalogoVersion),
	)
	canon.texto(datos.MotivoRC.ReferenciaCatalogo.CatalogoHuellaSHA256)
	canon.texto(datos.MotivoRC.ReferenciaCatalogo.EntradaClave)
	canon.texto(string(datos.MotivoRC.ClaveMensajeI18N))

	canon.texto(datos.PeticionRCRef)
	canon.texto(datos.HuellaPeticionRCHMAC)
	canon.texto(datos.HuellaRespuestaRC)
	canon.texto(datos.SelloRespuestaRCHMAC)
	canon.enteroSinSigno(uint64(datos.GeneracionRespuestaRC))
	canon.instante(datos.ConfirmadaRCEn)
	canon.instante(datos.RespuestaRCValidaHasta)
	escribirVinculoAutoridadFuenteAnalisisO3(
		canon,
		datos.AutoridadFuenteRC,
	)
	escribirVinculoAutoridadFuenteAnalisisO3(
		canon,
		datos.AutoridadVerificadorRC,
	)
	escribirVinculoAutoridadFuenteAnalisisO3(
		canon,
		datos.AutoridadPublicadorRC,
	)
	canon.texto(datos.PublicacionMotivoRef)
	canon.texto(datos.ReciboVerificacionMotivoRef)

	canon.booleano(datos.CostePrevisto != nil)
	if datos.CostePrevisto != nil {
		canon.enteroConSigno(datos.CostePrevisto.Centimos)
		canon.texto(datos.CostePrevisto.Moneda)
		canon.texto(datos.FuenteCosteRef)
		canon.texto(datos.ReciboCosteRef)
		canon.instante(datos.CalculadoEn)
		canon.texto(datos.PeticionCosteRef)
		canon.texto(datos.HuellaPeticionCosteHMAC)
		canon.texto(datos.HuellaRespuestaCoste)
		canon.texto(datos.SelloRespuestaCosteHMAC)
		canon.enteroSinSigno(uint64(datos.GeneracionRespuestaCoste))
		canon.instante(datos.ConfirmadaCosteEn)
		canon.instante(datos.RespuestaCosteValidaHasta)
		escribirVinculoAutoridadFuenteAnalisisO3(
			canon,
			datos.AutoridadFuenteCoste,
		)
		escribirVinculoAutoridadFuenteAnalisisO3(
			canon,
			datos.AutoridadVerificadorCoste,
		)
	}
	canon.instante(datos.PreparadoEn)
}

func escribirVinculoAutoridadFuenteAnalisisO3(
	canon *canonOperacionAnalisis,
	vinculo VinculoAutoridadFuenteAnalisisO3,
) {
	canon.texto(vinculo.RaizClaveID)
	canon.texto(vinculo.AutoridadRef)
	canon.texto(vinculo.BackendRef)
	canon.texto(string(vinculo.Rol))
	canon.enteroSinSigno(vinculo.Serie)
	canon.enteroSinSigno(uint64(vinculo.Generacion))
	canon.texto(vinculo.HuellaClaveSHA256)
	canon.instante(vinculo.CredencialEmitidaEn)
	canon.instante(vinculo.CredencialValidaHasta)
}
