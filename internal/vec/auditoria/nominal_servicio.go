package auditoria

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type ServicioNominal struct {
	emisor      EmisorMaterialV3
	fuente      FuenteAuditoriaNominal
	claveCursor [32]byte
	ahora       func() time.Time
}

func NuevoServicioNominal(emisor EmisorMaterialV3, fuente FuenteAuditoriaNominal) (*ServicioNominal, error) {
	if dependenciaNula(emisor) || dependenciaNula(fuente) {
		return nil, ErrNoDisponible
	}
	s := &ServicioNominal{emisor: emisor, fuente: fuente, ahora: time.Now}
	if _, err := rand.Read(s.claveCursor[:]); err != nil {
		return nil, ErrNoDisponible
	}
	return s, nil
}

func (s *ServicioNominal) Consultar(ctx context.Context, p PeticionNominal) (PaginaNominal, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || dependenciaNula(s.emisor) || dependenciaNula(s.fuente) ||
		p.Filtro.Validar() != nil || p.Filtro.AntesSecuencia != 0 {
		return PaginaNominal{}, ErrDenegada
	}
	f := p.Filtro
	if p.Cursor != "" {
		antes, err := s.decodificarCursorNominal(p.Cursor, f)
		if err != nil {
			return PaginaNominal{}, err
		}
		f.AntesSecuencia = antes
	}
	ahora := s.ahora().UTC().Truncate(time.Microsecond)
	c := p.Contexto
	if c.Motivo.Validar() != nil || c.Motivo.Referencia() != f.MotivoRef || c.Resultado.Validar() != nil ||
		c.Vinculo.ValidarPara(c.Resultado) != nil || !c.Vinculo.VigenteEn(ahora, c.Resultado) || c.Correlacion.Validar() != nil {
		return PaginaNominal{}, ErrDenegada
	}
	r, err := RecursoFiltroNominal(f)
	if err != nil {
		return PaginaNominal{}, err
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: c.Vinculo, ReferenciaMotivo: c.Motivo, Accion: AccionConsultarNominal,
		Recurso: r, Finalidad: f.FinalidadRef, Correlacion: c.Correlacion})
	if err != nil {
		return PaginaNominal{}, ErrDenegada
	}
	decision, confirmacion, exportador, err := s.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, c.Resultado)
	if err != nil || dependenciaNula(exportador) || ctx.Err() != nil {
		return PaginaNominal{}, ErrDenegada
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return PaginaNominal{}, ErrDenegada
	}
	q := ConsultaNominalAutorizada{Filtro: f, Material: material, Solicitud: solicitud, Decision: decision,
		Confirmacion: confirmacion, ResultadoContexto: c.Resultado}
	if ValidarConsultaNominalAutorizadaEn(q, s.ahora().UTC().Truncate(time.Microsecond)) != nil {
		return PaginaNominal{}, ErrDenegada
	}
	pagina, err := s.fuente.ConsultarAuditoriaNominal(ctx, q)
	if err != nil {
		return PaginaNominal{}, ErrNoDisponible
	}
	if ctx.Err() != nil || validarPaginaNominal(pagina, q, s.ahora().UTC().Truncate(time.Microsecond)) != nil {
		return PaginaNominal{}, ErrFuenteInvalida
	}
	filas := append([]RegistroNominal{}, pagina.Registros...)
	resultado := PaginaNominal{Registros: filas, Recibo: pagina.Recibo}
	if len(filas) > int(f.Limite) {
		resultado.Registros = filas[:f.Limite]
		resultado.SiguienteCursor, err = s.codificarCursorNominal(f, resultado.Registros[len(resultado.Registros)-1].Secuencia)
		if err != nil {
			return PaginaNominal{}, ErrNoDisponible
		}
	}
	return resultado, nil
}

func validarPaginaNominal(p PaginaFuenteNominal, q ConsultaNominalAutorizada, ahora time.Time) error {
	f := q.Filtro
	h, err := HuellaFiltroNominal(f)
	r := p.Recibo
	m := q.Material.ResumenCapacidad()
	if err != nil || len(p.Registros) > int(f.Limite)+1 || !referenciaExacta(r.AuditoriaRef, 512) ||
		r.DecisionRef != m.DecisionRef() || r.EfectoRef != m.EfectoRef() || r.HuellaFiltroSHA256 != h ||
		r.Resultado != TipoRegistroConsumoConfirmado || !instanteValido(r.RegistradaEn) ||
		r.RegistradaEn.Before(m.EmitidaEn()) || !r.RegistradaEn.Before(m.ExpiraEn()) || r.RegistradaEn.After(ahora) {
		return ErrFuenteInvalida
	}
	ids := make(map[string]struct{}, len(p.Registros))
	for i, r := range p.Registros {
		if !registroNominalValido(r, f) || (i > 0 && r.Secuencia >= p.Registros[i-1].Secuencia) {
			return ErrFuenteInvalida
		}
		if _, existe := ids[r.AuditoriaRef]; existe {
			return ErrFuenteInvalida
		}
		ids[r.AuditoriaRef] = struct{}{}
	}
	return nil
}

type cursorNominal struct {
	Version        int    `json:"v"`
	FiltroSHA256   string `json:"f"`
	AntesSecuencia uint64 `json:"s"`
}

func (s *ServicioNominal) codificarCursorNominal(f FiltroNominal, antes uint64) (string, error) {
	f.AntesSecuencia = 0
	h, err := HuellaFiltroNominal(f)
	if err != nil || antes == 0 || antes > maxSecuenciaNominal {
		return "", ErrDenegada
	}
	b, err := json.Marshal(cursorNominal{Version: 1, FiltroSHA256: h, AntesSecuencia: antes})
	if err != nil {
		return "", ErrNoDisponible
	}
	mac := hmac.New(sha256.New, s.claveCursor[:])
	_, _ = mac.Write(b)
	return base64.RawURLEncoding.EncodeToString(append(b, mac.Sum(nil)...)), nil
}

func (s *ServicioNominal) decodificarCursorNominal(token string, f FiltroNominal) (uint64, error) {
	if len(token) > 2048 {
		return 0, ErrDenegada
	}
	b, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(b) <= sha256.Size {
		return 0, ErrDenegada
	}
	cuerpo, firma := b[:len(b)-sha256.Size], b[len(b)-sha256.Size:]
	mac := hmac.New(sha256.New, s.claveCursor[:])
	_, _ = mac.Write(cuerpo)
	if !hmac.Equal(firma, mac.Sum(nil)) {
		return 0, ErrDenegada
	}
	var c cursorNominal
	f.AntesSecuencia = 0
	h, err := HuellaFiltroNominal(f)
	if err != nil || json.Unmarshal(cuerpo, &c) != nil || c.Version != 1 || c.FiltroSHA256 != h ||
		c.AntesSecuencia == 0 || c.AntesSecuencia > maxSecuenciaNominal {
		return 0, ErrDenegada
	}
	return c.AntesSecuencia, nil
}
