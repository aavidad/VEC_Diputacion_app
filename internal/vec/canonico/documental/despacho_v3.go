package documental

import "time"

const (
	esquemaAtestacionInicioEfectoV3       = "vec.documentos.atestacion-inicio-efecto.v3"
	esquemaReciboInicioEfectoV3           = "vec.documentos.recibo-inicio-efecto.v3"
	esquemaAtestacionReclamacionV3        = "vec.documentos.atestacion-reclamacion-despacho.v3"
	esquemaOrdenDespachoNominalV3         = "vec.documentos.orden-despacho-nominal.v3"
	esquemaMaterialVerificacionDespachoV3 = "vec.documentos.material-crudo-verificacion-despacho.v3"
	DuracionMaximaReclamacionDespachoV3   = 10 * time.Minute
)

// DatosAtestacionInicioEfectoV3 fija todos los campos de la preimagen que
// acredita el COMMIT de inicio. El puerto solo proyecta tipos ricos a este DTO.
type DatosAtestacionInicioEfectoV3 struct {
	InicioRef                  string
	ReservaRef                 string
	HuellaVinculoEstableSHA256 string
	SecuenciaCercado           uint64
	HuellaVinculoCercadoSHA256 string
	OrdenConsumoDurableV4Ref   string
	VersionInicioCAS           uint64
	AuditoriaInicioRef         string
	OutboxInicioRef            string
	ClaveAtestacionRef         string
	RevisionClave              uint64
	EvidenciaOperacionRef      string
	IniciadoEn                 time.Time
}

func (d DatosAtestacionInicioEfectoV3) Validar() bool {
	return ReferenciaEjecucionV3Valida(d.InicioRef) &&
		ReferenciaEjecucionV3Valida(d.ReservaRef) &&
		SHA256HexadecimalValido(d.HuellaVinculoEstableSHA256) && d.SecuenciaCercado > 0 &&
		SHA256HexadecimalValido(d.HuellaVinculoCercadoSHA256) &&
		ReferenciaEjecucionV3Valida(d.OrdenConsumoDurableV4Ref) && d.VersionInicioCAS > 0 &&
		ReferenciaEjecucionV3Valida(d.AuditoriaInicioRef) &&
		ReferenciaEjecucionV3Valida(d.OutboxInicioRef) &&
		ReferenciaEjecucionV3Valida(d.ClaveAtestacionRef) && d.RevisionClave > 0 &&
		ReferenciaEjecucionV3Valida(d.EvidenciaOperacionRef) && InstanteV3Valido(d.IniciadoEn)
}

func (d DatosAtestacionInicioEfectoV3) Bytes() []byte {
	if !d.Validar() {
		return nil
	}
	return SerializarCamposV3([]string{
		esquemaAtestacionInicioEfectoV3, d.InicioRef, d.ReservaRef,
		d.HuellaVinculoEstableSHA256, Uint64Decimal(d.SecuenciaCercado),
		d.HuellaVinculoCercadoSHA256, d.OrdenConsumoDurableV4Ref,
		Uint64Decimal(d.VersionInicioCAS), d.AuditoriaInicioRef, d.OutboxInicioRef,
		AlgoritmoHMACSHA256V3, AudienciaInicioEfectoV3, ContextoInicioEfectoV3,
		d.ClaveAtestacionRef, Uint64Decimal(d.RevisionClave), d.EvidenciaOperacionRef,
		d.IniciadoEn.Format(time.RFC3339Nano),
	})
}

// DatosReciboInicioEfectoV3 concentra forma, unicidad y huella del recibo
// durable nominal sin conocer la representacion opaca del puerto.
type DatosReciboInicioEfectoV3 struct {
	InicioRef                  string
	ReservaRef                 string
	HuellaVinculoEstableSHA256 string
	SecuenciaCercado           uint64
	HuellaVinculoCercadoSHA256 string
	OrdenConsumoDurableV4Ref   string
	VersionInicioCAS           uint64
	AuditoriaInicioRef         string
	OutboxInicioRef            string
	EvidenciaOperacionRef      string
	AtestacionValida           bool
	HuellaAtestacionSHA256     string
	IniciadoEn                 time.Time
}

func (d DatosReciboInicioEfectoV3) Validar() bool {
	return ReferenciasEjecucionV3Distintas(
		d.InicioRef, d.ReservaRef, d.OrdenConsumoDurableV4Ref,
		d.AuditoriaInicioRef, d.OutboxInicioRef, d.EvidenciaOperacionRef,
	) && SHA256HexadecimalValido(d.HuellaVinculoEstableSHA256) && d.SecuenciaCercado > 0 &&
		SHA256HexadecimalValido(d.HuellaVinculoCercadoSHA256) && d.VersionInicioCAS > 0 &&
		d.AtestacionValida && SHA256HexadecimalValido(d.HuellaAtestacionSHA256) &&
		InstanteV3Valido(d.IniciadoEn)
}

func (d DatosReciboInicioEfectoV3) HuellaSHA256() string {
	if !d.Validar() {
		return ""
	}
	return HuellaCamposSHA256V3([]string{
		esquemaReciboInicioEfectoV3, d.InicioRef, d.ReservaRef,
		d.HuellaVinculoEstableSHA256, Uint64Decimal(d.SecuenciaCercado),
		d.HuellaVinculoCercadoSHA256, d.OrdenConsumoDurableV4Ref,
		Uint64Decimal(d.VersionInicioCAS), d.AuditoriaInicioRef, d.OutboxInicioRef,
		d.HuellaAtestacionSHA256, d.IniciadoEn.Format(time.RFC3339Nano),
	})
}

// DatosSolicitudReclamacionV3 contiene la ventana CAS aportada por el
// consumidor de outbox.
type DatosSolicitudReclamacionV3 struct {
	ReclamacionRef string
	InicioRef      string
	OutboxRef      string
	ConsumidorRef  string
	ReclamadaEn    time.Time
	ExpiraEn       time.Time
}

func (d DatosSolicitudReclamacionV3) EsValida() bool {
	return ReferenciasEjecucionV3Distintas(
		d.ReclamacionRef, d.InicioRef, d.OutboxRef, d.ConsumidorRef,
	) && InstanteV3Valido(d.ReclamadaEn) && InstanteV3Valido(d.ExpiraEn) &&
		d.ExpiraEn.After(d.ReclamadaEn) &&
		d.ExpiraEn.Sub(d.ReclamadaEn) <= DuracionMaximaReclamacionDespachoV3
}

// DatosAtestacionReclamacionV3 fija la preimagen del segundo CAS durable.
type DatosAtestacionReclamacionV3 struct {
	Solicitud                  DatosSolicitudReclamacionV3
	HuellaReciboInicioSHA256   string
	InicioReciboRef            string
	OutboxInicioReciboRef      string
	IniciadoEn                 time.Time
	VersionReclamacionCAS      uint64
	AuditoriaReclamacionRef    string
	ClaveAtestacionRef         string
	RevisionClave              uint64
	EvidenciaOperacionRef      string
	SecuenciaCercado           uint64
	HuellaVinculoEstableSHA256 string
	HuellaVinculoCercadoSHA256 string
	OrdenConsumoDurableV4Ref   string
}

func (d DatosAtestacionReclamacionV3) Validar() bool {
	return d.Solicitud.EsValida() && SHA256HexadecimalValido(d.HuellaReciboInicioSHA256) &&
		d.Solicitud.InicioRef == d.InicioReciboRef &&
		d.Solicitud.OutboxRef == d.OutboxInicioReciboRef &&
		InstanteV3Valido(d.IniciadoEn) && !d.Solicitud.ReclamadaEn.Before(d.IniciadoEn) &&
		d.VersionReclamacionCAS > 0 && ReferenciaEjecucionV3Valida(d.AuditoriaReclamacionRef) &&
		ReferenciaEjecucionV3Valida(d.ClaveAtestacionRef) && d.RevisionClave > 0 &&
		ReferenciaEjecucionV3Valida(d.EvidenciaOperacionRef) && d.SecuenciaCercado > 0 &&
		SHA256HexadecimalValido(d.HuellaVinculoEstableSHA256) &&
		SHA256HexadecimalValido(d.HuellaVinculoCercadoSHA256) &&
		ReferenciaEjecucionV3Valida(d.OrdenConsumoDurableV4Ref)
}

func (d DatosAtestacionReclamacionV3) Bytes() []byte {
	if !d.Validar() {
		return nil
	}
	return SerializarCamposV3([]string{
		esquemaAtestacionReclamacionV3, d.HuellaReciboInicioSHA256,
		d.InicioReciboRef, d.Solicitud.ReclamacionRef, d.Solicitud.ConsumidorRef,
		Uint64Decimal(d.VersionReclamacionCAS), d.AuditoriaReclamacionRef,
		AlgoritmoHMACSHA256V3, AudienciaReclamacionDespachoV3,
		ContextoReclamacionDespachoV3, d.ClaveAtestacionRef,
		Uint64Decimal(d.RevisionClave), d.EvidenciaOperacionRef,
		Uint64Decimal(d.SecuenciaCercado), d.HuellaVinculoEstableSHA256,
		d.HuellaVinculoCercadoSHA256, d.OrdenConsumoDurableV4Ref,
		d.Solicitud.ReclamadaEn.Format(time.RFC3339Nano),
		d.Solicitud.ExpiraEn.Format(time.RFC3339Nano),
	})
}

// DatosOrdenDespachoV3 contiene los cotejos primitivos de la orden nominal.
type DatosOrdenDespachoV3 struct {
	Solicitud                   DatosSolicitudReclamacionV3
	HuellaReciboInicioSHA256    string
	HuellaReciboCalculadaSHA256 string
	VersionReclamacionCAS       uint64
	AuditoriaReclamacionRef     string
	EvidenciaOperacionRef       string
	AtestacionValida            bool
	HuellaAtestacionSHA256      string
	MensajeAtestacion           []byte
	MensajeEsperado             []byte
	IniciadoEn                  time.Time
}

func (d DatosOrdenDespachoV3) Validar() bool {
	return d.Solicitud.EsValida() && ReferenciasEjecucionV3Distintas(
		d.Solicitud.ReclamacionRef, d.Solicitud.ConsumidorRef,
		d.AuditoriaReclamacionRef, d.EvidenciaOperacionRef,
	) && SHA256HexadecimalValido(d.HuellaReciboInicioSHA256) &&
		d.HuellaReciboInicioSHA256 == d.HuellaReciboCalculadaSHA256 &&
		d.VersionReclamacionCAS > 0 && d.AtestacionValida &&
		SHA256HexadecimalValido(d.HuellaAtestacionSHA256) &&
		len(d.MensajeEsperado) > 0 && BytesIguales(d.MensajeAtestacion, d.MensajeEsperado) &&
		InstanteV3Valido(d.IniciadoEn) && !d.Solicitud.ReclamadaEn.Before(d.IniciadoEn)
}

func (d DatosOrdenDespachoV3) HuellaSHA256() string {
	if !d.Validar() {
		return ""
	}
	return HuellaCamposSHA256V3([]string{
		esquemaOrdenDespachoNominalV3, d.HuellaReciboInicioSHA256,
		d.Solicitud.ReclamacionRef, d.Solicitud.ConsumidorRef,
		Uint64Decimal(d.VersionReclamacionCAS), d.AuditoriaReclamacionRef,
		d.HuellaAtestacionSHA256, d.Solicitud.ReclamadaEn.Format(time.RFC3339Nano),
		d.Solicitud.ExpiraEn.Format(time.RFC3339Nano),
	})
}

// VinculosMaterialDespachoV3 son los identificadores durables comprometidos
// por la comprobacion conjunta de cercado, inicio y reclamacion.
type VinculosMaterialDespachoV3 struct {
	InicioRef                  string
	AtestacionInicioRef        string
	ReclamacionRef             string
	AtestacionReclamacionRef   string
	OrdenConsumoDurableV4Ref   string
	HuellaOrdenDespachoSHA256  string
	HuellaReciboInicioSHA256   string
	HuellaVinculoEstableSHA256 string
	HuellaVinculoCercadoSHA256 string
	SecuenciaCercado           uint64
	VersionInicioCAS           uint64
	VersionReclamacionCAS      uint64
}

func (v VinculosMaterialDespachoV3) EsValido() bool {
	return SHA256HexadecimalValido(v.HuellaOrdenDespachoSHA256) &&
		SHA256HexadecimalValido(v.HuellaReciboInicioSHA256) &&
		SHA256HexadecimalValido(v.HuellaVinculoEstableSHA256) &&
		SHA256HexadecimalValido(v.HuellaVinculoCercadoSHA256) &&
		v.SecuenciaCercado > 0 && v.VersionInicioCAS > 0 && v.VersionReclamacionCAS > 0
}

type PerfilMaterialDespachoV3 struct {
	Valido                  bool
	Audiencia               string
	ClaveGestionadaRef      string
	RevisionClaveGestionada uint64
	HuellaSHA256            string
}

// DatosMaterialDespachoV3 reune perfiles, ligaduras esperadas y preimagen.
type DatosMaterialDespachoV3 struct {
	Vinculos                        VinculosMaterialDespachoV3
	Cercado                         PerfilMaterialDespachoV3
	Inicio                          PerfilMaterialDespachoV3
	Reclamacion                     PerfilMaterialDespachoV3
	ClaveEsperadaRef                string
	RevisionEsperada                uint64
	HuellaOrdenEsperadaSHA256       string
	HuellaReciboEsperadaSHA256      string
	HuellaVinculoEsperadaSHA256     string
	HuellaCercadoEsperadaSHA256     string
	SecuenciaEsperada               uint64
	VersionInicioEsperada           uint64
	VersionReclamacionEsperada      uint64
	HuellaInicioEsperadaSHA256      string
	HuellaReclamacionEsperadaSHA256 string
	Mensaje                         []byte
	HuellaMensajeSHA256             string
}

func (d DatosMaterialDespachoV3) preimagen() []byte {
	return SerializarCamposV3([]string{
		esquemaMaterialVerificacionDespachoV3, AlgoritmoHMACSHA256V3,
		AudienciaComprobacionOrdenDespachoV3, ContextoComprobacionOrdenDespachoV3,
		d.Vinculos.InicioRef, d.Vinculos.AtestacionInicioRef,
		d.Vinculos.ReclamacionRef, d.Vinculos.AtestacionReclamacionRef,
		d.Vinculos.OrdenConsumoDurableV4Ref, d.Vinculos.HuellaOrdenDespachoSHA256,
		d.Vinculos.HuellaReciboInicioSHA256, d.Vinculos.HuellaVinculoEstableSHA256,
		d.Vinculos.HuellaVinculoCercadoSHA256, Uint64Decimal(d.Vinculos.SecuenciaCercado),
		Uint64Decimal(d.Vinculos.VersionInicioCAS), Uint64Decimal(d.Vinculos.VersionReclamacionCAS),
		d.Cercado.HuellaSHA256, d.Inicio.HuellaSHA256, d.Reclamacion.HuellaSHA256,
	})
}

func (d DatosMaterialDespachoV3) Validar() bool {
	perfilesValidos := d.Cercado.Valido && d.Inicio.Valido && d.Reclamacion.Valido &&
		d.Cercado.Audiencia == AudienciaTokenCercadoV3 &&
		d.Inicio.Audiencia == AudienciaInicioEfectoV3 &&
		d.Reclamacion.Audiencia == AudienciaReclamacionDespachoV3
	for _, perfil := range []PerfilMaterialDespachoV3{d.Cercado, d.Inicio, d.Reclamacion} {
		perfilesValidos = perfilesValidos && perfil.ClaveGestionadaRef == d.ClaveEsperadaRef &&
			perfil.RevisionClaveGestionada == d.RevisionEsperada &&
			SHA256HexadecimalValido(perfil.HuellaSHA256)
	}
	return perfilesValidos && d.Vinculos.EsValido() &&
		d.Vinculos.HuellaOrdenDespachoSHA256 == d.HuellaOrdenEsperadaSHA256 &&
		d.Vinculos.HuellaReciboInicioSHA256 == d.HuellaReciboEsperadaSHA256 &&
		d.Vinculos.HuellaVinculoEstableSHA256 == d.HuellaVinculoEsperadaSHA256 &&
		d.Vinculos.HuellaVinculoCercadoSHA256 == d.HuellaCercadoEsperadaSHA256 &&
		d.Vinculos.SecuenciaCercado == d.SecuenciaEsperada &&
		d.Vinculos.VersionInicioCAS == d.VersionInicioEsperada &&
		d.Vinculos.VersionReclamacionCAS == d.VersionReclamacionEsperada &&
		d.Inicio.HuellaSHA256 == d.HuellaInicioEsperadaSHA256 &&
		d.Reclamacion.HuellaSHA256 == d.HuellaReclamacionEsperadaSHA256 &&
		len(d.Mensaje) > 0 && BytesIguales(d.Mensaje, d.preimagen()) &&
		SHA256HexadecimalValido(d.HuellaMensajeSHA256) &&
		d.HuellaMensajeSHA256 == HuellaBytesSHA256(d.Mensaje)
}

func (d DatosMaterialDespachoV3) Bytes() []byte {
	preimagen := d.preimagen()
	return append([]byte(nil), preimagen...)
}
