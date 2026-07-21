package gobiernoconvocatorias

const (
	esquemaVinculoVerificadorReciboBorradorV1 = "bolsa.convocatoria.borrador.vinculo-verificador-recibo.v1"
	prefijoVinculoVerificadorReciboBorradorV1 = "verificador-recibo-borrador-v1:"
)

// VinculoVerificadorReciboBorrador acredita como una sola unidad la autoridad
// que relee el recibo durable y la autoridad criptografica que verifica sus
// evidencias. Sus campos son deliberadamente opacos: los adaptadores pueden
// transportar el vinculo, pero no fabricarlo a partir de referencias sueltas.
//
// La referencia derivada no contiene secretos. Es un compromiso SHA-256 de
// todas las coordenadas de ambas identidades y puede persistirse en la
// acreditacion para demostrar posteriormente la composicion que se empleo.
type VinculoVerificadorReciboBorrador struct {
	bloqueoSerializacionDiario
	esquema                string
	identidadPersistencia  IdentidadAutoridadBorrador
	identidadCriptografica IdentidadAutoridadBorrador
	referencia             string
}

type representacionIdentidadAutoridadVinculoBorrador struct {
	ProveedorRef  string `json:"proveedor_ref"`
	InstanciaRef  string `json:"instancia_ref"`
	CredencialRef string `json:"credencial_ref"`
	RolRef        string `json:"rol_ref"`
}

func representarIdentidadAutoridadVinculoBorrador(
	identidad IdentidadAutoridadBorrador,
) representacionIdentidadAutoridadVinculoBorrador {
	return representacionIdentidadAutoridadVinculoBorrador{
		ProveedorRef: identidad.ProveedorRef, InstanciaRef: identidad.InstanciaRef,
		CredencialRef: identidad.CredencialRef, RolRef: identidad.RolRef,
	}
}

func NuevoVinculoVerificadorReciboBorrador(
	identidadPersistencia, identidadCriptografica IdentidadAutoridadBorrador,
) (VinculoVerificadorReciboBorrador, error) {
	vinculo := VinculoVerificadorReciboBorrador{
		esquema:                esquemaVinculoVerificadorReciboBorradorV1,
		identidadPersistencia:  identidadPersistencia,
		identidadCriptografica: identidadCriptografica,
	}
	vinculo.referencia = vinculo.calcularReferencia()
	if !vinculo.valida() {
		return VinculoVerificadorReciboBorrador{}, ErrServicioBorradoresInvalido
	}
	return vinculo, nil
}

func (v VinculoVerificadorReciboBorrador) calcularReferencia() string {
	if v.esquema != esquemaVinculoVerificadorReciboBorradorV1 ||
		!autoridadesOperativasBorradorSeparadas(
			v.identidadPersistencia, v.identidadCriptografica,
		) {
		return ""
	}
	representacion := struct {
		Esquema                   string                                          `json:"esquema"`
		Persistencia              representacionIdentidadAutoridadVinculoBorrador `json:"persistencia"`
		VerificacionCriptografica representacionIdentidadAutoridadVinculoBorrador `json:"verificacion_criptografica"`
	}{
		Esquema:      v.esquema,
		Persistencia: representarIdentidadAutoridadVinculoBorrador(v.identidadPersistencia),
		VerificacionCriptografica: representarIdentidadAutoridadVinculoBorrador(
			v.identidadCriptografica,
		),
	}
	huella := huellaJSONCanonicaBorrador(representacion)
	if !huellaHexValida(huella) {
		return ""
	}
	return prefijoVinculoVerificadorReciboBorradorV1 + huella
}

func (v VinculoVerificadorReciboBorrador) valida() bool {
	referenciaCalculada := v.calcularReferencia()
	return referenciaCalculada != "" && referenciaProyeccionValida(v.referencia) &&
		coincideTextoConstante(v.referencia, referenciaCalculada)
}

// ReferenciaParaAcreditacion entrega exclusivamente el compromiso opaco que
// puede quedar en un recibo; nunca expone credenciales ni la estructura del
// vinculo.
func (v VinculoVerificadorReciboBorrador) ReferenciaParaAcreditacion() (string, error) {
	if !v.valida() {
		return "", ErrServicioBorradoresInvalido
	}
	return v.referencia, nil
}

func vinculosVerificadorReciboBorradorCoinciden(
	a, b VinculoVerificadorReciboBorrador,
) bool {
	return a.valida() && b.valida() &&
		a.esquema == b.esquema &&
		a.identidadPersistencia == b.identidadPersistencia &&
		a.identidadCriptografica == b.identidadCriptografica &&
		coincideTextoConstante(a.referencia, b.referencia)
}

type DescriptorVinculoVerificadorReciboBorrador interface {
	VinculoVerificadorReciboBorrador() VinculoVerificadorReciboBorrador
}

func vinculacionVerificadorServicioBorradoresValida(
	confirmador ConfirmadorAtomicoBorrador,
	verificador VerificadorReciboBorrador,
) bool {
	vinculoConfirmador := confirmador.VinculoVerificadorReciboBorrador()
	vinculoVerificador := verificador.VinculoVerificadorReciboBorrador()
	identidadConfirmador := confirmador.IdentidadAutoridadBorrador()
	identidadVerificador := verificador.IdentidadAutoridadBorrador()
	return vinculosVerificadorReciboBorradorCoinciden(vinculoConfirmador, vinculoVerificador) &&
		identidadVerificador == vinculoVerificador.identidadPersistencia &&
		autoridadesOperativasBorradorSeparadas(
			identidadConfirmador, vinculoVerificador.identidadPersistencia,
		) && autoridadesOperativasBorradorSeparadas(
		identidadConfirmador, vinculoVerificador.identidadCriptografica,
	) && autoridadesOperativasBorradorSeparadas(
		vinculoVerificador.identidadPersistencia,
		vinculoVerificador.identidadCriptografica,
	)
}
