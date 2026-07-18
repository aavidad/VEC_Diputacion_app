package gobiernoconvocatorias

import (
	"encoding/json"
	"time"
)

type AtestacionKMSBorrador struct {
	bloqueoSerializacionDiario
	Esquema               string
	AtestacionRef         string
	VersionAtestacion     uint32
	Estado                string
	Perfil                PerfilCifradoBorrador
	ClaveMaestraRef       string
	VersionClave          uint32
	HuellaAAD             string
	HuellaEnvolturaSHA256 string
	HuellaSobreSHA256     string
	VerificadorRef        string
	Procedencia           ProcedenciaActoBorrador
	Firma                 FirmaEvidenciaBorrador
	EmitidaEn             time.Time
	ValidaHasta           time.Time
}

func NuevaAtestacionKMSBorrador(
	atestacionRef string,
	versionAtestacion uint32,
	perfil PerfilCifradoBorrador,
	claveMaestraRef string,
	versionClave uint32,
	huellaAAD, huellaEnvoltura, huellaSobre, verificadorRef string,
	procedencia ProcedenciaActoBorrador,
	algoritmoFirma, huellaClavePublicaSHA256 string,
	emitidaEn, validaHasta time.Time,
	firmar FuncionFirmaEvidenciaBorrador,
) (AtestacionKMSBorrador, error) {
	a := AtestacionKMSBorrador{
		Esquema: esquemaAtestacionKMSBorradorV1, AtestacionRef: atestacionRef,
		VersionAtestacion: versionAtestacion, Estado: estadoAtestacionKMSVigente,
		Perfil: perfil, ClaveMaestraRef: claveMaestraRef, VersionClave: versionClave,
		HuellaAAD: huellaAAD, HuellaEnvolturaSHA256: huellaEnvoltura,
		HuellaSobreSHA256: huellaSobre, VerificadorRef: verificadorRef,
		Procedencia: procedencia,
		Firma: FirmaEvidenciaBorrador{
			AlgoritmoFirma: algoritmoFirma, VerificadorRef: verificadorRef,
			HuellaClavePublicaSHA256: huellaClavePublicaSHA256,
		},
		EmitidaEn: emitidaEn, ValidaHasta: validaHasta,
	}
	preimagen := a.preimagenFirma()
	firma, err := FirmarEvidenciaBorrador(
		algoritmoFirma, verificadorRef, huellaClavePublicaSHA256, preimagen, firmar,
	)
	if err != nil {
		return AtestacionKMSBorrador{}, ErrCifradoBorradorInvalido
	}
	a.Firma = firma
	if !a.validaEstructural() {
		return AtestacionKMSBorrador{}, ErrCifradoBorradorInvalido
	}
	return a, nil
}

func (a AtestacionKMSBorrador) preimagenFirma() []byte {
	p := a.Perfil
	representacion, err := json.Marshal(struct {
		Esquema, AtestacionRef, Estado, PerfilRef, HuellaPerfil string
		AlgoritmoAEAD, AlgoritmoEnvoltura, ClaveRef             string
		HuellaAAD, HuellaEnvoltura, HuellaSobre, VerificadorRef string
		AlgoritmoFirma, HuellaClavePublica                      string
		ProcedenciaEsquema, PerfilEjecucion, Autoridad          string
		ProveedorProcedenciaRef                                 string
		VersionAtestacion, PerfilVersion, VersionClave          uint32
		MigrableProduccion                                      bool
		EmitidaEn, ValidaHasta                                  string
	}{
		a.Esquema, a.AtestacionRef, a.Estado, p.Referencia, p.HuellaContenidoSHA256,
		p.AlgoritmoAEAD, p.AlgoritmoEnvolturaClave, a.ClaveMaestraRef,
		a.HuellaAAD, a.HuellaEnvolturaSHA256, a.HuellaSobreSHA256, a.VerificadorRef,
		a.Firma.AlgoritmoFirma, a.Firma.HuellaClavePublicaSHA256,
		a.Procedencia.Esquema, a.Procedencia.PerfilEjecucion, a.Procedencia.Autoridad,
		a.Procedencia.ProveedorRef, a.VersionAtestacion, p.Version, a.VersionClave,
		a.Procedencia.MigrableProduccion, a.EmitidaEn.Format(time.RFC3339Nano),
		a.ValidaHasta.Format(time.RFC3339Nano),
	})
	if err != nil {
		return nil
	}
	return representacion
}

func (a AtestacionKMSBorrador) DatosParaVerificacionFirma() (
	preimagen []byte,
	algoritmoFirma, verificadorRef, huellaClavePublicaSHA256 string,
	firma []byte,
	err error,
) {
	preimagen = a.preimagenFirma()
	algoritmoFirma, verificadorRef, huellaClavePublicaSHA256, firma, err =
		a.Firma.DatosParaVerificacion(preimagen)
	return
}

func (a AtestacionKMSBorrador) validaEstructural() bool {
	return a.Esquema == esquemaAtestacionKMSBorradorV1 &&
		referenciaProyeccionValida(a.AtestacionRef) && a.VersionAtestacion > 0 &&
		a.Estado == estadoAtestacionKMSVigente && a.Perfil.valida() &&
		referenciaProyeccionValida(a.ClaveMaestraRef) && a.VersionClave > 0 &&
		huellaHexValida(a.HuellaAAD) && huellaHexValida(a.HuellaEnvolturaSHA256) &&
		huellaHexValida(a.HuellaSobreSHA256) && referenciaProyeccionValida(a.VerificadorRef) &&
		a.Procedencia.valida() && a.Firma.VerificadorRef == a.VerificadorRef &&
		a.Firma.validaPara(a.preimagenFirma()) &&
		a.AtestacionRef != a.VerificadorRef && instanteOperacionCanonico(a.EmitidaEn) &&
		instanteOperacionCanonico(a.ValidaHasta) && a.ValidaHasta.After(a.EmitidaEn) &&
		a.ValidaHasta.Sub(a.EmitidaEn) <= 10*time.Minute
}
