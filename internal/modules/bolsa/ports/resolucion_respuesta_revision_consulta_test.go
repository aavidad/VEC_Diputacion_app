package ports

import (
	"strings"
	"testing"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
)

func TestPaginaRevisionNoAnunciaOpcionAjenaNiMotivoSinPublicacion(t *testing.T) {
	instante := time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)
	orden := OrdenConsultaRespuestasRevision{Solicitud: SolicitudConsultarRespuestasRevision{
		BolsaRef: "bolsa:rrhh:1", Limite: 20}}
	opcion := OpcionRespuestaRevision{Resultado: "admitir_justificacion", Destino: "disponible",
		MotivoRef: core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_revision_bolsa",
			CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_admitir"}}
	pagina := PaginaRespuestasRevision{BolsaRef: orden.Solicitud.BolsaRef, AuditoriaRef: "auditoria:revision:1",
		Filas: []RespuestaEnRevisionRRHH{{RespuestaRef: "respuesta-portal:" + strings.Repeat("b", 64),
			BolsaRef: orden.Solicitud.BolsaRef, LlamamientoRef: "llamamiento:1",
			ParticipacionRef: "participacion:1", OrdenActa: 12, Respuesta: "renuncia_justificada",
			Procedimiento: "directo", RespondidaEn: instante,
			SituacionEsperadaDesde: instante.Add(-time.Hour), SituacionActual: "disponible",
			Opciones: []OpcionRespuestaRevision{opcion}}}}
	if err := pagina.ValidarPara(orden); err != nil {
		t.Fatalf("opción nominal rechazada: %v", err)
	}
	pagina.Filas[0].OrdenActa = 0
	if pagina.ValidarPara(orden) == nil {
		t.Fatal("fila sin orden de acta acreditado admitida")
	}
	pagina.Filas[0].OrdenActa = 12
	pagina.Filas[0].Opciones[0].Destino = "pendiente_incorporacion"
	if pagina.ValidarPara(orden) == nil {
		t.Fatal("opción que cambia el resultado admitida")
	}
	pagina.Filas[0].Opciones[0] = opcion
	pagina.Filas[0].Opciones[0].MotivoRef.CatalogoHuellaSHA256 = ""
	if pagina.ValidarPara(orden) == nil {
		t.Fatal("opción sin referencia normativa admitida")
	}
	pagina.Filas[0].Opciones = []OpcionRespuestaRevision{}
	if err := pagina.ValidarPara(orden); err != nil {
		t.Fatalf("fila sin regla publicada rechazada: %v", err)
	}
}
