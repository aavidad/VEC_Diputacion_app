package application

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// huellaRegularizacionDocumental liga a la autorización V3 los datos
// semánticos del acto B77. Los instantes de ejecución quedan fuera: un replay
// conserva el comando aunque el servidor lo atienda más tarde.
func huellaRegularizacionDocumental(q ports.SolicitudOperacionSituacion, actor, recibo string) (string, error) {
	if q.CausaFinalizadaEn == nil || q.SituacionEsperadaDesde.IsZero() ||
		!q.SituacionEsperadaDesde.Equal(q.SituacionEsperadaDesde.Truncate(time.Microsecond)) ||
		strings.ContainsRune(q.Motivo, '\x1f') {
		return "", domain.ErrOperacionSituacionParticipacionInvalida
	}
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		return "", ErrCambioSituacionParticipacionNoDisponible
	}
	motivo := sha256.Sum256([]byte(q.Motivo))
	partes := []string{
		"regularizacion-documental-v1", q.SolicitudRef, strconv.FormatInt(q.SolicitudVersionEsperada, 10),
		q.SolicitudContenidoSHA256, q.BolsaRef, q.ParticipacionRef, q.Justificante.Referencia,
		q.Justificante.SHA256, q.CausaFinalizadaEn.In(madrid).Format("2006-01-02"),
		strconv.FormatInt(q.SituacionEsperadaDesde.UnixMicro(), 10), hex.EncodeToString(motivo[:]),
		actor, q.Validador, q.ClaveIdempotencia, recibo,
	}
	for _, parte := range partes {
		if parte == "" || strings.ContainsRune(parte, '\x1f') {
			return "", domain.ErrOperacionSituacionParticipacionInvalida
		}
	}
	suma := sha256.Sum256([]byte(strings.Join(partes, "\x1f") + "\x1f"))
	return hex.EncodeToString(suma[:]), nil
}

// contextoRecursoRegularizacionCanonico entrega exactamente la preimagen que
// V3 sella como contexto_recurso_huella_sha256. B77 recibe estos bytes y
// rechaza cualquier atributo o ámbito distinto de los esperados.
func contextoRecursoRegularizacionCanonico(recurso dominiovec.RecursoAutorizable) ([]byte, error) {
	if err := recurso.Validar(); err != nil {
		return nil, err
	}
	canon, err := json.Marshal(struct {
		Ambitos   map[string]string `json:"ambitos"`
		Atributos map[string]string `json:"atributos"`
	}{Ambitos: recurso.Ambitos, Atributos: recurso.Atributos})
	if err != nil {
		return nil, err
	}
	huellaV3, err := recurso.HuellaContextoAutorizacionSHA256()
	suma := sha256.Sum256(canon)
	if err != nil || hex.EncodeToString(suma[:]) != huellaV3 {
		return nil, dominiovec.ErrAutorizacionDenegada
	}
	return canon, nil
}
