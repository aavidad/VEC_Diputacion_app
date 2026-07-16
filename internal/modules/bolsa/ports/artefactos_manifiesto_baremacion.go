package ports

// ArtefactosCanonicosManifiestoProbatorioBaremacionV3 contiene las tres
// representaciones binarias que el archivo probatorio durable debe conservar
// y contrastar. CargaProtegida impide su serializacion accidental y entrega
// siempre copias defensivas mediante Revelar.
type ArtefactosCanonicosManifiestoProbatorioBaremacionV3 struct {
	ContenidoSinHuella    CargaProtegida
	RepresentacionSellada CargaProtegida
	PreimagenHMAC         CargaProtegida
}

// ContenidoCanonicoManifiestoProbatorioBaremacionV3 reconstruye exactamente
// los bytes usados como preimagen SHA-256 por PrepararSellado, es decir,
// materialCanonico(false). Solo admite un manifiesto completo con estructura,
// huella y formato coherentes; la autenticidad del HMAC se verifica en la
// frontera criptografica externa.
func ContenidoCanonicoManifiestoProbatorioBaremacionV3(
	manifiesto ManifiestoProbatorioBaremacion,
) (CargaProtegida, error) {
	if manifiesto.Validar() != nil {
		return CargaProtegida{}, ErrSolicitudBaremacionInvalida
	}
	material, err := manifiesto.materialCanonico(false)
	if err != nil {
		return CargaProtegida{}, ErrSolicitudBaremacionInvalida
	}
	defer borrarBytesManifiesto(material)
	return NuevaCargaProtegida(material)
}

// ArtefactosCanonicosManifiestoProbatorioBaremacion reconstruye, sin aceptar
// bytes aportados por el cliente, el contenido sin huella, la representacion
// canonica que incluye la huella y la preimagen exacta del HMAC. De este modo
// PostgreSQL puede cotejar byte a byte su archivo sin poseer claves.
func ArtefactosCanonicosManifiestoProbatorioBaremacion(
	manifiesto ManifiestoProbatorioBaremacion,
) (ArtefactosCanonicosManifiestoProbatorioBaremacionV3, error) {
	contenido, err := ContenidoCanonicoManifiestoProbatorioBaremacionV3(manifiesto)
	if err != nil {
		return ArtefactosCanonicosManifiestoProbatorioBaremacionV3{}, err
	}
	representacion, err := RepresentacionCanonicaManifiestoProbatorioBaremacion(manifiesto)
	if err != nil {
		return ArtefactosCanonicosManifiestoProbatorioBaremacionV3{}, ErrSolicitudBaremacionInvalida
	}
	preimagen, err := (SolicitudSellarSelloBaremacion{
		Finalidad:              FinalidadSelloManifiestoProbatorioBaremacionV3,
		RepresentacionCanonica: representacion,
	}).MaterialCanonicoHMAC()
	if err != nil {
		return ArtefactosCanonicosManifiestoProbatorioBaremacionV3{}, ErrSolicitudBaremacionInvalida
	}
	return ArtefactosCanonicosManifiestoProbatorioBaremacionV3{
		ContenidoSinHuella: contenido, RepresentacionSellada: representacion, PreimagenHMAC: preimagen,
	}, nil
}

func borrarBytesManifiesto(valor []byte) {
	for indice := range valor {
		valor[indice] = 0
	}
}
