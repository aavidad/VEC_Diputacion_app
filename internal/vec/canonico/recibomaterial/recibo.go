package recibomaterial

import "crypto/sha256"

// HuellaPlan contiene el compromiso publicado y el vinculo que lo contextualiza.
type HuellaPlan struct {
	Referencia    string
	Version       uint32
	Suma          [sha256.Size]byte
	HuellaVinculo [sha256.Size]byte
}

func HuellaPlanValida(h HuellaPlan) bool {
	return AliasLogicoValido(h.Referencia, 512) && h.Version > 0 &&
		h.Suma != ([sha256.Size]byte{}) && h.HuellaVinculo != ([sha256.Size]byte{})
}

// Recibo contiene todos los hechos deterministas cubiertos por la atestacion.
type Recibo struct {
	Esquema                   string
	VersionEsquema            uint16
	ReferenciaDurableOriginal string
	PerfilReferencia          string
	PerfilVersion             uint32
	HuellaPerfil              [sha256.Size]byte
	Hechos                    HechosContexto
	HuellaPlan                HuellaPlan
	Instantanea               Instantanea
}

func HechosMaterialesReciboValidos(r Recibo) bool {
	return r.Esquema == EsquemaRecibo && r.VersionEsquema == EsquemaVersion &&
		AliasLogicoValido(r.PerfilReferencia, 512) && r.PerfilVersion > 0 &&
		r.HuellaPerfil != ([sha256.Size]byte{}) && HechosContextoValidos(r.Hechos) &&
		r.Hechos.AccionTecnica == AccionEscribir && HuellaPlanValida(r.HuellaPlan) &&
		InstantaneaValida(r.Instantanea)
}

func ReciboValido(r Recibo) bool {
	return HechosMaterialesReciboValidos(r) && AliasLogicoValido(r.ReferenciaDurableOriginal, 512)
}

func anexarHechosRecibo(canonico []byte, r Recibo) []byte {
	i := r.Instantanea
	canonico = AnexarTLV(canonico, 3, []byte(i.ConectorLogicoID))
	canonico = AnexarTLV(canonico, 4, []byte(r.PerfilReferencia))
	canonico = AnexarTLV(canonico, 5, Uint32(r.PerfilVersion))
	canonico = AnexarTLV(canonico, 6, r.HuellaPerfil[:])
	canonico = AnexarTLV(canonico, 7, []byte(r.Hechos.ModuloID))
	canonico = AnexarTLV(canonico, 8, []byte(r.Hechos.AccionNegocio))
	canonico = AnexarTLV(canonico, 9, []byte(r.Hechos.AccionTecnica))
	canonico = AnexarTLV(canonico, 10, []byte(r.Hechos.RecursoRef))
	canonico = AnexarTLV(canonico, 11, []byte(r.Hechos.OperacionRef))
	canonico = AnexarTLV(canonico, 12, []byte(r.Hechos.CargaRef))
	canonico = AnexarTLV(canonico, 13, []byte(r.Hechos.EfectoRef))
	canonico = AnexarTLV(canonico, 14, []byte(r.HuellaPlan.Referencia))
	canonico = AnexarTLV(canonico, 15, Uint32(r.HuellaPlan.Version))
	canonico = AnexarTLV(canonico, 16, r.HuellaPlan.Suma[:])
	canonico = AnexarTLV(canonico, 17, r.HuellaPlan.HuellaVinculo[:])
	canonico = AnexarTLV(canonico, 18, []byte(r.Hechos.Clasificacion))
	canonico = AnexarTLV(canonico, 19, []byte(i.ObjetoRef))
	canonico = AnexarTLV(canonico, 20, []byte(i.ObjetoVersion))
	canonico = AnexarTLV(canonico, 21, []byte(i.Zona))
	canonico = AnexarTLV(canonico, 22, []byte(i.MIME))
	canonico = AnexarTLV(canonico, 23, Int64(i.Tamano))
	canonico = AnexarTLV(canonico, 24, i.HuellaContenido[:])
	canonico = AnexarTLV(canonico, 25, []byte(i.EvidenciaCreacionRef))
	canonico = AnexarTLV(canonico, 26, Int64(i.AlmacenadoEn.UnixMicro()))
	canonico = AnexarTLV(canonico, 27, Bool(i.TieneRetencion))
	if i.TieneRetencion {
		canonico = AnexarTLV(canonico, 28, Int64(i.RetenidoHasta.UnixMicro()))
	}
	canonico = AnexarTLV(canonico, 29, []byte(i.EstadoInmovilizacion))
	return AnexarTLV(canonico, 30, []byte(i.EstadoObjeto))
}

func CanonicoRecibo(r Recibo) ([]byte, error) {
	if !ReciboValido(r) {
		return nil, ErrReciboNoValido
	}
	var canonico []byte
	canonico = AnexarTLV(canonico, 0, []byte(r.Esquema))
	canonico = AnexarTLV(canonico, 1, Uint16(r.VersionEsquema))
	canonico = AnexarTLV(canonico, 2, []byte(r.ReferenciaDurableOriginal))
	return anexarHechosRecibo(canonico, r), nil
}

func CanonicoIdentidadDurable(r Recibo) ([]byte, error) {
	if !HechosMaterialesReciboValidos(r) || r.ReferenciaDurableOriginal != "" {
		return nil, ErrReciboNoValido
	}
	var canonico []byte
	canonico = AnexarTLV(canonico, 0, []byte("vec.almacen.identidad-recibo-escritura-material.v2"))
	canonico = AnexarTLV(canonico, 1, Uint16(r.VersionEsquema))
	return anexarHechosRecibo(canonico, r), nil
}

// HuellaRecibo calcula el compromiso sobre los bytes canonicos completos.
func HuellaRecibo(r Recibo) ([sha256.Size]byte, error) {
	canonico, err := CanonicoRecibo(r)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(canonico), nil
}
