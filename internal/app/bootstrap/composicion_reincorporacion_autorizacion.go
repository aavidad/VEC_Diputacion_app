package bootstrap

import (
	"log"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// El perfil autoriza la organización; el recurso y su huella siguen ligados
// al expediente, la versión, la relación y la evidencia exactos de CT130/134.
func solicitudAutorizacionReincorporacionTitularValida(ruta string, d vecdomain.DatosSolicitudAutorizacionLigadaV3) bool {
	r := d.Recurso
	if !rutaReincorporacionTitularDesarrollo(ruta) ||
		d.ReferenciaMotivo != motivoSeguimientoCeseDesarrollo(httpinterno.RutaReincorporacionesTitular) ||
		r.ModuloID != ports.ModuloContratacion || !domain.ReferenciaOpacaValida(r.Referencia) ||
		len(r.Ambitos) != 1 || r.Ambitos["organizacion_ref"] != organizacionAltaContratacionTemporalDesarrollo {
		return false
	}
	if d.Accion == accionConsultarSeguimientoCeseDesarrollo {
		return d.Finalidad == "gestionar_contratacion_temporal" && r.Tipo == "seguimiento_contratacion_temporal" &&
			len(r.Atributos) == 1 && r.Atributos["lectura"] == "reincorporacion_titular"
	}
	version, err := strconv.ParseUint(r.Atributos["version_expediente"], 10, 64)
	if err != nil || !ports.VersionOperacionAnalisisConIncrementoValida(version) ||
		r.Atributos["version_expediente"] != strconv.FormatUint(version, 10) {
		log.Print("contratacion temporal: reincorporacion denegada; causa=version_expediente_invalida")
		return false
	}
	if d.Accion == string(ports.AccionConsultarAntecedenteReincorporacionTitular) {
		return ruta == httpinterno.RutaReincorporacionesTitular &&
			d.Finalidad == ports.FinalidadLecturaReincorporacionTitular && r.Tipo == ports.TipoRecursoLecturaReincorporacionTitular &&
			len(r.Atributos) == 5 && evidenciaReincorporacionTitularValida(r.Atributos)
	}
	if d.Accion != string(domain.AccionRegistrarReincorporacionTitular) || d.Finalidad != ports.FinalidadRegistrarReincorporacionTitular ||
		r.Tipo != ports.TipoRecursoReincorporacionTitular || r.Atributos["fase_previa"] != string(domain.FaseNombramiento) ||
		r.Atributos["estado_previo"] != string(domain.EstadoEnCurso) {
		return false
	}
	if ruta == httpinterno.RutaCapacidadReincorporacionTitular {
		return len(r.Atributos) == 3
	}
	if len(r.Atributos) != 14 || !evidenciaReincorporacionTitularValida(r.Atributos) ||
		!domain.ReferenciaOpacaValida(r.Atributos["cese_evento_ref"]) || !domain.ReferenciaOpacaValida(r.Atributos["cese_recibo_ref"]) ||
		!domain.ReferenciaOpacaValida(r.Atributos["politica_ref"]) || !huellaSHA256ValidaContratacionTemporalDesarrollo(r.Atributos["politica_huella_sha256"]) {
		return false
	}
	politicaVersion, err := strconv.ParseUint(r.Atributos["politica_version"], 10, 64)
	if err != nil || !ports.VersionOperacionAnalisisValida(politicaVersion) ||
		r.Atributos["politica_version"] != strconv.FormatUint(politicaVersion, 10) {
		log.Print("contratacion temporal: reincorporacion denegada; causa=version_politica_invalida")
		return false
	}
	ambitos, errAmbitos := ports.NuevaColeccionSellosHMAC(r.Atributos["ambito_idempotencia_hmac"], nil)
	huellas, errHuellas := ports.NuevaColeccionSellosHMAC(r.Atributos["huella_peticion_hmac"], nil)
	return errAmbitos == nil && errHuellas == nil && ports.ColeccionesHMACContienenPar(ambitos, ports.DominioAmbitoReincorporacionTitular,
		huellas, ports.DominioHuellaReincorporacionTitular, r.Atributos["ambito_idempotencia_hmac"], r.Atributos["huella_peticion_hmac"])
}

func evidenciaReincorporacionTitularValida(atributos map[string]string) bool {
	fecha, err := time.Parse(time.DateOnly, atributos["fecha_efectiva"])
	return err == nil && (domain.DatosReincorporacionTitular{RelacionRef: atributos["relacion_ref"], FechaEfectiva: fecha,
		DocumentoRef: atributos["documento_ref"], DocumentoSHA256: atributos["documento_sha256"]}).Validar() == nil
}
